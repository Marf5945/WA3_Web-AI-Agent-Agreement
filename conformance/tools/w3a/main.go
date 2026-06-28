package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	extPrefix = "ㄝ"
	sep       = "："
	listSep   = "｜"
)

var (
	coreHeaderOrder  = []string{"ㄊㄡ", "ㄏㄠ", "ㄉㄞ", "ㄏㄡ", "ㄈㄢ"}
	actionFieldOrder = []string{"ㄘ", "ㄓ", "ㄍㄞ", "ㄖㄣ", "ㄕㄡ", "ㄉㄜ", "ㄑㄩㄢ", "ㄊㄞ"}
	blockFieldOrder  = []string{"ㄍㄜ", "ㄩㄢ", "ㄊㄠ", "ㄊㄞ"}
	prefOrder        = []string{"ㄋㄛ", "ㄘㄤ", "ㄙㄜ", "ㄗ", "ㄉㄥ", "ㄎㄞ"}
	setKeys          = map[string]bool{"ㄋㄛ": true, "ㄘㄤ": true, "ㄑㄩㄢ": true}
	sectionNames     = map[string]string{"ㄋㄥ": "neng", "ㄎㄜ": "ke", "ㄕㄜ": "she", "ㄓㄤ": "zhang"}
	rlOrder          = []string{"ㄓㄜ", "ㄎㄟ", "ㄔㄣ", "ㄑㄢ", "ㄝㄕㄜ"}
	krOrder          = []string{"ㄓㄜ", "舊鑰", "新鑰", "ㄔㄣ"}
)

const (
	eEncUTF8          = "E-ENC-UTF8"
	eEncBOM           = "E-ENC-BOM"
	eEncNull          = "E-ENC-NULL"
	eStructDupKey     = "E-STRUCT-DUPKEY"
	eStructDupID      = "E-STRUCT-DUP-ID"
	eStructParse      = "E-STRUCT-PARSE"
	eTokenUnknownCore = "E-TOKEN-UNKNOWN-CORE"
	eTokenInvalid     = "E-TOKEN-INVALID"
	eVersionFormat    = "E-VERSION-FORMAT"
	eVersionMajor     = "E-VERSION-MAJOR"
	eBuilderSchema    = "E-BUILDER-SCHEMA"
	eBuilderSecret    = "E-VALUE-SECRET"
	eBuilderCodeOwned = "E-BUILDER-CODE-OWNED"
	eBuilderHumanOnly = "E-BUILDER-HUMAN-ONLY"
	eBuilderRisk      = "E-BUILDER-RISK"
)

type w3aError struct {
	code string
	msg  string
}

func (e w3aError) Error() string {
	if e.msg == "" {
		return e.code
	}
	return e.code + ": " + e.msg
}

func fail(code, msg string) error {
	return w3aError{code: code, msg: msg}
}

type pair struct {
	key   string
	value string
	raw   string
}

type entry struct {
	kind   string
	id     string
	rawID  string
	fields []pair
}

type document struct {
	header []pair
	neng   []entry
	ke     []entry
	she    []pair
	zhang  []pair
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "canonical":
		err = cmdCanonical(os.Args[2:])
	case "build":
		err = cmdBuild(os.Args[2:])
	case "keygen":
		err = cmdKeygen(os.Args[2:])
	case "sign":
		err = cmdSign(os.Args[2:])
	case "trust":
		err = cmdTrust(os.Args[2:])
	case "gen-vectors":
		err = cmdGenVectors(os.Args[2:])
	case "bundle-check":
		err = cmdBundleCheck(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: w3a canonical <file> | build --answers <answers.json> --out <app.w3a> [--test-sign] [--mock-demo <demo.json>] | keygen --publisher <id> --key-out <private-key.json> [--force] | sign --key-file <private-key.json> --in <draft.w3a> --out <app.w3a> | trust [--trusted-pub <hex>] [--revoked-pub <hex>] [--rl <file.w3a-rl>] <file> | gen-vectors | bundle-check")
}

func cmdCanonical(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: w3a canonical <file>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var out []byte
	switch {
	case strings.HasSuffix(args[0], ".w3a-rl"):
		out, err = canonicalRL(data)
	case strings.HasSuffix(args[0], ".w3a-kr"):
		out, err = canonicalKR(data)
	default:
		out, err = canonicalizeBytes(data)
	}
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "sha256=%s\n", sha256Hex(out))
	return nil
}

type buildAnswers struct {
	AnswersSchemaVersion string           `json:"answers_schema_version"`
	TemplateID           string           `json:"template_id"`
	TemplateVersion      string           `json:"template_version"`
	App                  answerApp        `json:"app"`
	Publisher            answerPublisher  `json:"publisher"`
	CustomTemplate       *builderTemplate `json:"custom_template"`
	LLMSuggestions       []builderRecord  `json:"llm_suggestions"`
	HumanDecisions       []builderRecord  `json:"human_decisions"`
	OpaqueConfirmations  []opaqueConfirm  `json:"opaque_confirmations"`
	Raw                  map[string]any   `json:"-"`
}

type answerApp struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Backend string `json:"backend"`
	Scope   string `json:"scope"`
}

type answerPublisher struct {
	ID        string `json:"id"`
	PublicKey string `json:"public_key"`
}

type builderRecord map[string]any

type opaqueConfirm struct {
	FieldPath  string `json:"field_path"`
	Reason     string `json:"reason"`
	Provenance string `json:"provenance"`
	Writer     string `json:"writer"`
}

type builderTemplate struct {
	TemplateID      string           `json:"template_id"`
	TemplateVersion string           `json:"template_version"`
	DisplayName     string           `json:"display_name"`
	DefaultApp      answerApp        `json:"default_app"`
	Entities        []templateEntity `json:"entities"`
	Actions         []templateAction `json:"actions"`
	Blocks          []templateBlock  `json:"blocks"`
	Preferences     templatePrefs    `json:"preferences"`
}

type templateEntity struct {
	ID     string          `json:"id"`
	Fields []templateField `json:"fields"`
}

type templateField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type templateAction struct {
	ID         string          `json:"id"`
	Verb       string          `json:"verb"`
	Target     string          `json:"target"`
	Mutates    bool            `json:"mutates"`
	Confirm    bool            `json:"confirm"`
	RiskClass  string          `json:"risk_class"`
	Purpose    string          `json:"purpose"`
	DataImpact string          `json:"data_impact"`
	Inputs     []templateField `json:"inputs"`
	Outputs    []string        `json:"outputs"`
}

type templateBlock struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Source   string   `json:"source"`
	Actions  []string `json:"actions"`
	Fallback []string `json:"fallback"`
}

type templatePrefs struct {
	Readonly      []string `json:"readonly"`
	HiddenActions []string `json:"hidden_actions"`
	ThemeColors   []string `json:"theme_colors"`
	FontSizes     []string `json:"font_sizes"`
	Density       string   `json:"density"`
	VisibleBlocks []string `json:"visible_blocks"`
}

type actionDecision struct {
	Decision              string
	NewID                 string
	ConfirmDisabledByUser bool
}

type buildResult struct {
	Contract      []byte
	CanonicalHash string
	Actions       []templateAction
	Renames       map[string]string
}

func cmdBuild(args []string) error {
	var answersPath, outPath, mockDemoPath string
	testSign := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--answers":
			i++
			if i >= len(args) {
				return errors.New("missing value for --answers")
			}
			answersPath = args[i]
		case "--out":
			i++
			if i >= len(args) {
				return errors.New("missing value for --out")
			}
			outPath = args[i]
		case "--test-sign":
			testSign = true
		case "--mock-demo":
			i++
			if i >= len(args) {
				return errors.New("missing value for --mock-demo")
			}
			mockDemoPath = args[i]
		default:
			return fmt.Errorf("unknown build arg: %s", args[i])
		}
	}
	if answersPath == "" || outPath == "" {
		return errors.New("usage: w3a build --answers <answers.json> --out <app.w3a> [--test-sign] [--mock-demo <demo.json>]")
	}
	root, err := findSpecRoot()
	if err != nil {
		return err
	}
	result, tmpl, answers, err := buildFromAnswersFile(root, answersPath, testSign)
	if err != nil {
		return err
	}
	if outPath == "-" {
		if _, err := os.Stdout.Write(result.Contract); err != nil {
			return err
		}
	} else if err := os.WriteFile(outPath, result.Contract, 0o644); err != nil {
		return err
	}
	if mockDemoPath != "" {
		demo, err := mockDemoJSON(tmpl, answers, result)
		if err != nil {
			return err
		}
		if err := os.WriteFile(mockDemoPath, demo, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "canonical_sha256=%s\n", result.CanonicalHash)
	return nil
}

func buildFromAnswersFile(root, answersPath string, testSign bool) (buildResult, builderTemplate, buildAnswers, error) {
	answers, raw, err := loadAnswers(answersPath)
	if err != nil {
		return buildResult{}, builderTemplate{}, buildAnswers{}, err
	}
	answers.Raw = raw
	if err := validateAnswers(root, answers); err != nil {
		return buildResult{}, builderTemplate{}, buildAnswers{}, err
	}
	tmpl, err := resolveTemplate(root, answers)
	if err != nil {
		return buildResult{}, builderTemplate{}, buildAnswers{}, err
	}
	if tmpl.TemplateVersion != answers.TemplateVersion {
		return buildResult{}, builderTemplate{}, buildAnswers{}, fail(eBuilderSchema, "template_version mismatch")
	}
	result, err := buildContract(answers, tmpl, testSign)
	if err != nil {
		return buildResult{}, builderTemplate{}, buildAnswers{}, err
	}
	return result, tmpl, answers, nil
}

func loadAnswers(path string) (buildAnswers, map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return buildAnswers{}, nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return buildAnswers{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	var answers buildAnswers
	if err := json.Unmarshal(data, &answers); err != nil {
		return buildAnswers{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	return answers, raw, nil
}

func loadTemplate(root, id string) (builderTemplate, error) {
	path := filepath.Join(root, "builder", "templates", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return builderTemplate{}, err
	}
	var tmpl builderTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return builderTemplate{}, fmt.Errorf("%s: %w", path, err)
	}
	return tmpl, nil
}

func resolveTemplate(root string, answers buildAnswers) (builderTemplate, error) {
	if answers.TemplateID != "custom_generic" {
		return loadTemplate(root, answers.TemplateID)
	}
	if answers.CustomTemplate == nil {
		return builderTemplate{}, fail(eBuilderSchema, "custom_template is required for custom_generic")
	}
	tmpl := *answers.CustomTemplate
	if tmpl.TemplateID != "custom_generic" {
		return builderTemplate{}, fail(eBuilderSchema, "custom_template.template_id must be custom_generic")
	}
	if tmpl.TemplateVersion == "" {
		return builderTemplate{}, fail(eBuilderSchema, "custom_template.template_version is required")
	}
	if err := validateTemplate(tmpl); err != nil {
		return builderTemplate{}, err
	}
	return tmpl, nil
}

func validateAnswers(root string, answers buildAnswers) error {
	if err := checkJSON(filepath.Join(root, "builder", "answers.schema.json")); err != nil {
		return err
	}
	if answers.AnswersSchemaVersion != "1.0" {
		return fail(eBuilderSchema, "answers_schema_version must be 1.0")
	}
	if answers.TemplateID == "" || answers.TemplateVersion == "" {
		return fail(eBuilderSchema, "template_id and template_version are required")
	}
	if answers.App.ID == "" || answers.App.Version == "" || answers.App.Backend == "" || answers.App.Scope == "" {
		return fail(eBuilderSchema, "app id/version/backend/scope are required")
	}
	if answers.Publisher.ID == "" {
		return fail(eBuilderSchema, "publisher.id is required")
	}
	if err := validateBackendHandle(answers.App.Backend); err != nil {
		return err
	}
	opaque := map[string]bool{}
	for _, o := range answers.OpaqueConfirmations {
		if o.Writer != "human" || o.Provenance != "user_confirmed" || o.FieldPath == "" || o.Reason == "" {
			return fail(eBuilderHumanOnly, "opaque confirmation must be human/user_confirmed with a reason")
		}
		opaque[o.FieldPath] = true
	}
	if err := scanSecrets(answers.Raw, "", opaque); err != nil {
		return err
	}
	for _, s := range answers.LLMSuggestions {
		if stringField(s, "provenance") != "system_suggested" {
			return fail(eBuilderHumanOnly, "LLM suggestions may only be system_suggested")
		}
	}
	for _, d := range answers.HumanDecisions {
		if _, ok := d["risk_class"]; ok {
			return fail(eBuilderCodeOwned, "risk_class is code_owned")
		}
		if writer := stringField(d, "writer"); writer != "" && writer != "human" {
			return fail(eBuilderHumanOnly, "human_decisions require writer=human")
		}
		if prov := stringField(d, "provenance"); prov != "" && prov != "user_confirmed" {
			return fail(eBuilderHumanOnly, "human_decisions require user_confirmed provenance")
		}
	}
	return nil
}

func validateTemplate(tmpl builderTemplate) error {
	if tmpl.TemplateID == "" || tmpl.TemplateVersion == "" || tmpl.DisplayName == "" {
		return fail(eBuilderSchema, "template id/version/display_name are required")
	}
	if len(tmpl.Entities) == 0 || len(tmpl.Actions) == 0 || len(tmpl.Blocks) == 0 {
		return fail(eBuilderSchema, "custom template requires entities, actions, and blocks")
	}
	entityIDs := map[string]bool{}
	for _, e := range tmpl.Entities {
		if e.ID == "" || entityIDs[e.ID] {
			return fail(eBuilderSchema, "invalid or duplicate entity id: "+e.ID)
		}
		entityIDs[e.ID] = true
		fieldIDs := map[string]bool{}
		for _, f := range e.Fields {
			if err := validateField(f, fieldIDs); err != nil {
				return err
			}
		}
	}
	actionIDs := map[string]bool{}
	for _, a := range tmpl.Actions {
		if a.ID == "" || actionIDs[a.ID] {
			return fail(eBuilderSchema, "invalid or duplicate action id: "+a.ID)
		}
		actionIDs[a.ID] = true
		if !validVerb(a.Verb) {
			return fail(eBuilderSchema, "invalid action verb: "+a.ID)
		}
		if err := validateActionTarget(a.Target); err != nil {
			return err
		}
		if !validRiskClass(a.RiskClass) {
			return fail(eBuilderSchema, "invalid risk_class: "+a.ID)
		}
		if a.Mutates && !a.Confirm && a.RiskClass != "low_mutate" {
			return fail(eBuilderRisk, "mutating action without confirm requires low_mutate: "+a.ID)
		}
		inputIDs := map[string]bool{}
		for _, f := range a.Inputs {
			if err := validateField(f, inputIDs); err != nil {
				return err
			}
		}
	}
	blockIDs := map[string]bool{}
	for _, b := range tmpl.Blocks {
		if b.ID == "" || blockIDs[b.ID] {
			return fail(eBuilderSchema, "invalid or duplicate block id: "+b.ID)
		}
		blockIDs[b.ID] = true
		if !validBlockType(b.Type) {
			return fail(eBuilderSchema, "invalid block type: "+b.ID)
		}
		if b.Source != "" && !actionIDs[b.Source] && !entityIDs[b.Source] {
			return fail(eBuilderSchema, "block source must reference an action or entity: "+b.ID)
		}
		for _, id := range b.Actions {
			if !actionIDs[id] {
				return fail(eBuilderSchema, "block action reference not found: "+id)
			}
		}
	}
	return nil
}

func validateField(f templateField, seen map[string]bool) error {
	if f.Name == "" || seen[f.Name] {
		return fail(eBuilderSchema, "invalid or duplicate field name: "+f.Name)
	}
	seen[f.Name] = true
	if f.Type != "ㄐㄩ" && f.Type != "ㄗㄤ" && f.Type != "ㄋㄞ" {
		return fail(eBuilderSchema, "invalid field type: "+f.Name)
	}
	return nil
}

func validVerb(v string) bool {
	return map[string]bool{"ㄎㄢ": true, "ㄔㄥ": true, "ㄢ": true, "ㄗㄜ": true, "ㄙㄡ": true, "ㄕㄢ": true}[v]
}

func validRiskClass(v string) bool {
	return map[string]bool{"read": true, "low_mutate": true, "high_mutate": true, "irreversible": true}[v]
}

func validBlockType(v string) bool {
	return map[string]bool{"ㄇㄢ": true, "ㄗㄞ": true, "ㄆㄤ": true, "ㄔㄠ": true, "ㄙㄞ": true, "ㄉㄟ": true}[v]
}

func validateActionTarget(s string) error {
	if strings.Contains(s, "://") {
		return fail(eBuilderSchema, "action target must be a relative path")
	}
	if strings.HasPrefix(s, "/") && !strings.Contains(s, "..") {
		return nil
	}
	return fail(eBuilderSchema, "action target must start with / and must not contain ..")
}

func validateBackendHandle(s string) error {
	for _, prefix := range []string{"mock://", "gdrive://", "api://", "https://"} {
		if strings.HasPrefix(s, prefix) {
			return nil
		}
	}
	return fail(eBuilderSchema, "backend must be mock://, gdrive://, api://, or https://")
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func boolField(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func scanSecrets(v any, path string, opaque map[string]bool) error {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			p := k
			if path != "" {
				p = path + "." + k
			}
			if err := scanSecrets(vv, p, opaque); err != nil {
				return err
			}
		}
	case []any:
		for i, vv := range x {
			p := fmt.Sprintf("%s[%d]", path, i)
			if err := scanSecrets(vv, p, opaque); err != nil {
				return err
			}
		}
	case string:
		if err := scanSecretString(x, path, opaque[path]); err != nil {
			return err
		}
	}
	return nil
}

func scanSecretString(s, path string, opaque bool) error {
	hardNeedles := []string{"Bearer ", "ghp_", "sk-", "AKIA"}
	for _, n := range hardNeedles {
		if strings.Contains(s, n) {
			return fail(eBuilderSecret, path+" contains "+n)
		}
	}
	if regexp.MustCompile(`(^|[?&])(token|access_token|api_key)=`).MatchString(s) {
		return fail(eBuilderSecret, path+" contains secret query parameter")
	}
	if regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`).MatchString(s) {
		return fail(eBuilderSecret, path+" contains JWT-shaped value")
	}
	longBase64 := regexp.MustCompile(`^[A-Za-z0-9+/]{48,}={0,2}$`)
	if longBase64.MatchString(s) && !opaque {
		return fail(eBuilderSecret, path+" contains long base64-shaped value")
	}
	return nil
}

func buildContract(answers buildAnswers, tmpl builderTemplate, testSign bool) (buildResult, error) {
	decisions := map[string]actionDecision{}
	for _, d := range answers.HumanDecisions {
		id := stringField(d, "action_id")
		if id == "" {
			return buildResult{}, fail(eBuilderSchema, "human decision missing action_id")
		}
		decisions[id] = actionDecision{
			Decision:              defaultString(stringField(d, "decision"), "keep"),
			NewID:                 stringField(d, "new_id"),
			ConfirmDisabledByUser: boolField(d, "confirm_disabled_by_user"),
		}
	}
	actionIDs := map[string]bool{}
	kept := []templateAction{}
	renames := map[string]string{}
	for _, a := range tmpl.Actions {
		d := decisions[a.ID]
		if d.Decision == "" {
			d.Decision = "keep"
		}
		if d.Decision == "remove" {
			continue
		}
		if d.ConfirmDisabledByUser {
			if a.RiskClass != "low_mutate" {
				return buildResult{}, fail(eBuilderRisk, "confirm can only be disabled for low_mutate: "+a.ID)
			}
			a.Confirm = false
		}
		if d.Decision == "rename" {
			if d.NewID == "" {
				return buildResult{}, fail(eBuilderSchema, "rename requires new_id: "+a.ID)
			}
			renames[a.ID] = d.NewID
			a.ID = d.NewID
		}
		if actionIDs[a.ID] {
			return buildResult{}, fail(eBuilderSchema, "duplicate action id after decisions: "+a.ID)
		}
		actionIDs[a.ID] = true
		kept = append(kept, a)
	}
	draft := emitContract(answers, tmpl, kept, renames, "", "")
	canon, err := canonicalizeBytes([]byte(draft))
	if err != nil {
		return buildResult{}, err
	}
	pubValue := answers.Publisher.PublicKey
	sigValue := ""
	if testSign {
		pub, sig := signWithTestKey(canon)
		pubValue = "ed25519:" + base64.StdEncoding.EncodeToString(pub)
		sigValue = base64.StdEncoding.EncodeToString(sig)
	}
	out := emitContract(answers, tmpl, kept, renames, pubValue, sigValue)
	canon, err = canonicalizeBytes([]byte(out))
	if err != nil {
		return buildResult{}, err
	}
	return buildResult{Contract: []byte(out), CanonicalHash: sha256Hex(canon), Actions: kept, Renames: renames}, nil
}

func defaultString(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func emitContract(answers buildAnswers, tmpl builderTemplate, actions []templateAction, renames map[string]string, pubValue, sigValue string) string {
	lines := []string{
		"ㄊㄡ" + sep + "W3A v0.3",
		"ㄏㄠ" + sep + answers.App.ID,
		"ㄉㄞ" + sep + answers.App.Version,
		"ㄏㄡ" + sep + answers.App.Backend,
		"ㄈㄢ" + sep + answers.App.Scope,
		"",
		"ㄋㄥ",
	}
	for _, e := range tmpl.Entities {
		lines = append(lines, "ㄕ"+sep+e.ID)
		for _, f := range e.Fields {
			lines = append(lines, "  "+f.Name+sep+f.Type)
		}
	}
	for _, a := range actions {
		lines = append(lines, "ㄓㄠ"+sep+a.ID)
		lines = append(lines, "  ㄘ"+sep+a.Verb)
		lines = append(lines, "  ㄓ"+sep+a.Target)
		lines = append(lines, "  ㄍㄞ"+sep+yesNo(a.Mutates))
		if a.Mutates || a.Confirm {
			lines = append(lines, "  ㄖㄣ"+sep+yesNo(a.Confirm))
		}
		for _, in := range a.Inputs {
			lines = append(lines, "  ㄕㄡ"+sep+in.Name+listSep+in.Type)
		}
		if len(a.Outputs) > 0 {
			lines = append(lines, "  ㄉㄜ"+sep+strings.Join(a.Outputs, listSep))
		}
	}
	lines = append(lines, "", "ㄎㄜ")
	for _, b := range tmpl.Blocks {
		lines = append(lines, "ㄑㄩ"+sep+b.ID)
		lines = append(lines, "  ㄍㄜ"+sep+b.Type)
		lines = append(lines, "  ㄩㄢ"+sep+renamed(b.Source, renames))
		actionRefs := []string{}
		for _, id := range b.Actions {
			r := renamed(id, renames)
			if actionExists(r, actions) {
				actionRefs = append(actionRefs, r)
			}
		}
		if len(actionRefs) > 0 {
			lines = append(lines, "  ㄊㄠ"+sep+strings.Join(actionRefs, listSep))
		}
		if len(b.Fallback) > 0 {
			lines = append(lines, "  ㄊㄞ"+sep+strings.Join(b.Fallback, listSep))
		}
	}
	lines = append(lines, "", "ㄕㄜ")
	if len(tmpl.Preferences.Readonly) > 0 {
		lines = append(lines, "ㄋㄛ"+sep+strings.Join(tmpl.Preferences.Readonly, listSep))
	}
	hidden := filterActions(tmpl.Preferences.HiddenActions, actions, renames)
	if len(hidden) > 0 {
		lines = append(lines, "ㄘㄤ"+sep+strings.Join(hidden, listSep))
	}
	if len(tmpl.Preferences.ThemeColors) > 0 {
		lines = append(lines, "ㄙㄜ"+sep+strings.Join(tmpl.Preferences.ThemeColors, listSep))
	}
	if len(tmpl.Preferences.FontSizes) > 0 {
		lines = append(lines, "ㄗ"+sep+strings.Join(tmpl.Preferences.FontSizes, listSep))
	}
	if tmpl.Preferences.Density != "" {
		lines = append(lines, "ㄉㄥ"+sep+tmpl.Preferences.Density)
	}
	if len(tmpl.Preferences.VisibleBlocks) > 0 {
		lines = append(lines, "ㄎㄞ"+sep+strings.Join(tmpl.Preferences.VisibleBlocks, listSep))
	}
	lines = append(lines, "", "ㄓㄤ")
	lines = append(lines, "ㄓㄜ"+sep+answers.Publisher.ID)
	lines = append(lines, "ㄎㄟ"+sep+pubValue)
	lines = append(lines, "ㄔㄣ"+sep+"2026-06-27T00:00:00Z")
	lines = append(lines, "ㄓㄥ"+sep+sigValue)
	return strings.Join(lines, "\n") + "\n"
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func renamed(id string, renames map[string]string) string {
	if v, ok := renames[id]; ok {
		return v
	}
	return id
}

func actionExists(id string, actions []templateAction) bool {
	for _, a := range actions {
		if a.ID == id {
			return true
		}
	}
	return false
}

func filterActions(ids []string, actions []templateAction, renames map[string]string) []string {
	out := []string{}
	for _, id := range ids {
		r := renamed(id, renames)
		if actionExists(r, actions) {
			out = append(out, r)
		}
	}
	return out
}

func testKeypair() (ed25519.PublicKey, ed25519.PrivateKey) {
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	return priv.Public().(ed25519.PublicKey), priv
}

func signWithTestKey(msg []byte) (ed25519.PublicKey, []byte) {
	pub, priv := testKeypair()
	return pub, ed25519.Sign(priv, msg)
}

type privateKeyFile struct {
	Type        string `json:"type"`
	Publisher   string `json:"publisher"`
	PublicKey   string `json:"public_key"`
	PublicHex   string `json:"public_key_hex"`
	Fingerprint string `json:"fingerprint"`
	SeedHex     string `json:"seed_hex"`
	CreatedAt   string `json:"created_at"`
}

func cmdKeygen(args []string) error {
	var publisher, keyOut string
	force := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--publisher":
			i++
			if i >= len(args) {
				return errors.New("missing value for --publisher")
			}
			publisher = args[i]
		case "--key-out":
			i++
			if i >= len(args) {
				return errors.New("missing value for --key-out")
			}
			keyOut = args[i]
		case "--force":
			force = true
		default:
			return fmt.Errorf("unknown keygen arg: %s", args[i])
		}
	}
	if publisher == "" || keyOut == "" {
		return errors.New("usage: w3a keygen --publisher <id> --key-out <private-key.json> [--force]")
	}
	if err := ensureKeyPathOutsideRepo(keyOut); err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(keyOut); err == nil {
			return fmt.Errorf("key file exists, pass --force to overwrite: %s", keyOut)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	seed := priv.Seed()
	rec := privateKeyFile{
		Type:        "w3a-ed25519-seed-v1",
		Publisher:   publisher,
		PublicKey:   "ed25519:" + base64.StdEncoding.EncodeToString(pub),
		PublicHex:   hex.EncodeToString(pub),
		Fingerprint: keyFingerprint(pub),
		SeedHex:     hex.EncodeToString(seed),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := os.MkdirAll(filepath.Dir(keyOut), 0o700); err != nil {
		return err
	}
	body, err := marshalJSON(rec)
	if err != nil {
		return err
	}
	if err := os.WriteFile(keyOut, body, 0o600); err != nil {
		return err
	}
	public := map[string]string{
		"publisher":      publisher,
		"public_key":     rec.PublicKey,
		"public_key_hex": rec.PublicHex,
		"fingerprint":    rec.Fingerprint,
		"key_file":       keyOut,
	}
	out, err := marshalJSON(public)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

func cmdSign(args []string) error {
	var keyFile, inPath, outPath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key-file":
			i++
			if i >= len(args) {
				return errors.New("missing value for --key-file")
			}
			keyFile = args[i]
		case "--in":
			i++
			if i >= len(args) {
				return errors.New("missing value for --in")
			}
			inPath = args[i]
		case "--out":
			i++
			if i >= len(args) {
				return errors.New("missing value for --out")
			}
			outPath = args[i]
		default:
			return fmt.Errorf("unknown sign arg: %s", args[i])
		}
	}
	if keyFile == "" || inPath == "" || outPath == "" {
		return errors.New("usage: w3a sign --key-file <private-key.json> --in <draft.w3a> --out <app.w3a>")
	}
	key, priv, pub, err := loadPrivateKeyFile(keyFile)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	signed, canonHash, err := signContract(raw, key.Publisher, pub, priv)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, signed, 0o644); err != nil {
		return err
	}
	status, err := classifyTrust(signed, hex.EncodeToString(pub), "")
	if err != nil {
		return err
	}
	status["canonical_sha256"] = canonHash
	status["fingerprint"] = keyFingerprint(pub)
	out, err := marshalJSON(status)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

func ensureKeyPathOutsideRepo(path string) error {
	root, err := findSpecRoot()
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return fmt.Errorf("private key path must be outside the W3A_SPEC repository: %s", path)
	}
	return nil
}

func loadPrivateKeyFile(path string) (privateKeyFile, ed25519.PrivateKey, ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return privateKeyFile{}, nil, nil, err
	}
	var rec privateKeyFile
	if err := json.Unmarshal(data, &rec); err != nil {
		return privateKeyFile{}, nil, nil, err
	}
	if rec.Type != "w3a-ed25519-seed-v1" {
		return privateKeyFile{}, nil, nil, errors.New("unsupported key file type")
	}
	seed, err := hex.DecodeString(rec.SeedHex)
	if err != nil || len(seed) != ed25519.SeedSize {
		return privateKeyFile{}, nil, nil, errors.New("invalid ed25519 seed")
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	if rec.PublicHex != "" && !strings.EqualFold(rec.PublicHex, hex.EncodeToString(pub)) {
		return privateKeyFile{}, nil, nil, errors.New("key file public key does not match seed")
	}
	return rec, priv, pub, nil
}

func signContract(raw []byte, keyPublisher string, pub ed25519.PublicKey, priv ed25519.PrivateKey) ([]byte, string, error) {
	text, err := normalize(raw)
	if err != nil {
		return nil, "", err
	}
	doc, err := parseDoc(text)
	if err != nil {
		return nil, "", err
	}
	docPublisher := strings.TrimSpace(zhangValue(doc, "ㄓㄜ"))
	if docPublisher != "" && keyPublisher != "" && docPublisher != keyPublisher {
		return nil, "", fmt.Errorf("publisher mismatch: contract=%s key=%s", docPublisher, keyPublisher)
	}
	publisher := defaultString(docPublisher, keyPublisher)
	if publisher == "" {
		return nil, "", errors.New("publisher id is required in contract or key file")
	}
	canon, err := canonicalDoc(doc)
	if err != nil {
		return nil, "", err
	}
	sig := ed25519.Sign(priv, canon)
	lines := strings.Split(strings.TrimRight(string(canon), "\n"), "\n")
	lines = append(lines, "", "ㄓㄤ")
	lines = append(lines, "ㄓㄜ"+sep+publisher)
	lines = append(lines, "ㄎㄟ"+sep+"ed25519:"+base64.StdEncoding.EncodeToString(pub))
	lines = append(lines, "ㄔㄣ"+sep+time.Now().UTC().Format(time.RFC3339))
	lines = append(lines, "ㄓㄥ"+sep+base64.StdEncoding.EncodeToString(sig))
	return []byte(strings.Join(lines, "\n") + "\n"), sha256Hex(canon), nil
}

func keyFingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:16])
}

func mockDemoJSON(tmpl builderTemplate, answers buildAnswers, result buildResult) ([]byte, error) {
	type demoAction struct {
		ID              string `json:"id"`
		RiskClass       string `json:"risk_class"`
		ConfirmRequired bool   `json:"confirm_required"`
		Provider        string `json:"provider"`
	}
	actions := []demoAction{}
	for _, a := range result.Actions {
		actions = append(actions, demoAction{ID: a.ID, RiskClass: a.RiskClass, ConfirmRequired: a.Confirm, Provider: "mock"})
	}
	demo := map[string]any{
		"template_id": tmpl.TemplateID,
		"app_id":      answers.App.ID,
		"backend":     answers.App.Backend,
		"actions":     actions,
		"sample_data": mockSampleData(tmpl.TemplateID),
	}
	return marshalJSON(demo)
}

func mockSampleData(templateID string) any {
	switch templateID {
	case "task_list":
		return []map[string]string{{"id": "task-1", "title": "Review W3A draft", "done": "no"}}
	case "feedback_form":
		return []map[string]string{{"id": "feedback-1", "name": "Demo", "message": "Looks good"}}
	case "product_showcase":
		return map[string]any{
			"product": map[string]string{"id": "product-1", "name": "Demo Product", "category": "Product showcase", "summary": "A product overview with solution sections, resources, and inquiry intake."},
			"solutions": []map[string]string{
				{"id": "solution-1", "title": "Core solution", "summary": "Primary value proposition and proof point."},
			},
			"resources": []map[string]string{
				{"id": "resource-1", "title": "Product brief", "kind": "brief"},
			},
		}
	case "mobile_product_app":
		return map[string]any{
			"product_units": []map[string]string{
				{"id": "unit-1", "title": "Precision control unit", "category": "Mobile product app", "spec_one": "< 0.5 ms"},
			},
			"case_metrics": []map[string]string{
				{"id": "case-1", "region": "Demo region", "title": "Factory deployment", "impact": "+35% throughput"},
			},
			"resources": []map[string]string{
				{"id": "resource-1", "title": "Technical manual", "kind": "pdf", "size": "12 MB"},
			},
		}
	default:
		return []map[string]string{{"id": "msg-1", "author": "Demo", "text": "Hello W3A"}}
	}
}

func cmdTrust(args []string) error {
	var trustedHex, revokedHex, rlFile, file string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--trusted-pub":
			i++
			if i >= len(args) {
				return errors.New("missing value for --trusted-pub")
			}
			trustedHex = args[i]
		case "--revoked-pub":
			i++
			if i >= len(args) {
				return errors.New("missing value for --revoked-pub")
			}
			revokedHex = args[i]
		case "--rl":
			i++
			if i >= len(args) {
				return errors.New("missing value for --rl")
			}
			rlFile = args[i]
		default:
			if file != "" {
				return fmt.Errorf("unexpected trust arg: %s", args[i])
			}
			file = args[i]
		}
	}
	if file == "" {
		return errors.New("usage: w3a trust [--trusted-pub <hex>] [--revoked-pub <hex>] [--rl <file.w3a-rl>] <file>")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	status, err := classifyTrust(data, trustedHex, revokedHex)
	if err != nil {
		return err
	}
	if rlFile != "" {
		status, err = applyRevocationList(status, data, rlFile)
		if err != nil {
			return err
		}
	}
	out, err := marshalJSON(status)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

// applyRevocationList re-checks an already-classified contract against a §27
// revocation list (.w3a-rl). It reuses the existing RL parser and the standard
// library only. A match overrides the trust state to revoked; non-signed inputs
// are returned unchanged.
//
// Limitation (documented in docs/RENDER_PIPELINE.md and docs/STATUS.md): this is
// a structural membership check. Honoring an RL in a production runtime still
// requires verifying the RL's own publisher signature per §27 before trusting
// its entries. That verification step is intentionally deferred to v1.x.
func applyRevocationList(status map[string]string, contract []byte, rlFile string) (map[string]string, error) {
	switch status["state"] {
	case "signed_trusted", "signed_untrusted_key", "test_signed":
		// re-checkable signed states only
	default:
		return status, nil
	}
	text, err := normalize(contract)
	if err != nil {
		return nil, err
	}
	doc, err := parseDoc(text)
	if err != nil {
		return nil, err
	}
	pubValue := strings.TrimSpace(zhangValue(doc, "ㄎㄟ"))
	appID := strings.TrimSpace(headerValue(doc, "ㄏㄠ"))
	version := strings.TrimSpace(headerValue(doc, "ㄉㄞ"))
	rlRaw, err := os.ReadFile(rlFile)
	if err != nil {
		return nil, err
	}
	rl, err := parseRLKR(rlRaw)
	if err != nil {
		return nil, err
	}
	if subject, reason, ok := matchRevocation(rl.revs, pubValue, appID, version); ok {
		revoked := map[string]string{
			"state":   "revoked",
			"code":    "E-TRUST-REVOKED",
			"badge":   "revoked",
			"subject": subject,
		}
		if pk, ok := status["public_key"]; ok {
			revoked["public_key"] = pk
		}
		if reason != "" {
			revoked["reason"] = reason
		}
		return revoked, nil
	}
	return status, nil
}

// matchRevocation reports whether a contract's publisher key or app@version is
// listed in a revocation list. Key subjects use the ed25519:<base64> form; app
// subjects use <app_id>@<version>, matched against the full version or its major.
func matchRevocation(revs []string, pubValue, appID, version string) (subject, reason string, ok bool) {
	major := version
	if i := strings.Index(version, "."); i >= 0 {
		major = version[:i]
	}
	for _, rev := range revs {
		parts := strings.Split(rev, "｜")
		subj := strings.TrimSpace(parts[0])
		r := ""
		for _, f := range parts[1:] {
			f = strings.TrimSpace(f)
			if strings.HasPrefix(f, "reason=") {
				r = strings.TrimPrefix(f, "reason=")
			}
		}
		switch {
		case strings.HasPrefix(subj, "ed25519:"):
			if pubValue != "" && subj == pubValue {
				return subj, r, true
			}
		case strings.Contains(subj, "@") && appID != "":
			if subj == appID+"@"+version || subj == appID+"@"+major {
				return subj, r, true
			}
		}
	}
	return "", "", false
}

func headerValue(doc document, key string) string {
	for _, p := range doc.header {
		if p.key == key {
			return p.value
		}
	}
	return ""
}

func classifyTrust(raw []byte, trustedHex, revokedHex string) (map[string]string, error) {
	text, err := normalize(raw)
	if err != nil {
		return nil, err
	}
	doc, err := parseDoc(text)
	if err != nil {
		return nil, err
	}
	if len(doc.zhang) == 0 {
		return map[string]string{"state": "unsigned_draft", "code": "E-TRUST-NOSIG", "badge": "unsigned"}, nil
	}
	pubValue := zhangValue(doc, "ㄎㄟ")
	sigValue := zhangValue(doc, "ㄓㄥ")
	if strings.TrimSpace(sigValue) == "" {
		return map[string]string{"state": "unsigned_draft", "code": "E-TRUST-UNSIGNED", "badge": "draft"}, nil
	}
	if !strings.HasPrefix(pubValue, "ed25519:") {
		return map[string]string{"state": "sig_mismatch", "code": "E-TRUST-BADKEY", "badge": "invalid"}, nil
	}
	pub, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(pubValue, "ed25519:"))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return map[string]string{"state": "sig_mismatch", "code": "E-TRUST-BADKEY", "badge": "invalid"}, nil
	}
	sig, err := base64.StdEncoding.DecodeString(sigValue)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return map[string]string{"state": "sig_mismatch", "code": "E-TRUST-BADSIG", "badge": "invalid"}, nil
	}
	canon, err := canonicalizeBytes(raw)
	if err != nil {
		return nil, err
	}
	pubHex := hex.EncodeToString(pub)
	if !ed25519.Verify(ed25519.PublicKey(pub), canon, sig) {
		return map[string]string{"state": "sig_mismatch", "code": "E-TRUST-SIG-MISMATCH", "badge": "invalid", "public_key": pubHex}, nil
	}
	testPub, _ := testKeypair()
	if bytes.Equal(pub, testPub) {
		return map[string]string{"state": "test_signed", "code": "E-TRUST-TESTKEY", "badge": "test", "public_key": pubHex}, nil
	}
	if revokedHex != "" && strings.EqualFold(revokedHex, pubHex) {
		return map[string]string{"state": "revoked", "code": "E-TRUST-REVOKED", "badge": "revoked", "public_key": pubHex}, nil
	}
	if trustedHex != "" && strings.EqualFold(trustedHex, pubHex) {
		return map[string]string{"state": "signed_trusted", "code": "OK", "badge": "trusted", "public_key": pubHex}, nil
	}
	return map[string]string{"state": "signed_untrusted_key", "code": "E-TRUST-UNTRUSTED-KEY", "badge": "untrusted", "public_key": pubHex}, nil
}

func zhangValue(doc document, key string) string {
	for _, p := range doc.zhang {
		if p.key == key {
			return p.value
		}
	}
	return ""
}

func normalize(raw []byte) (string, error) {
	if bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
		return "", fail(eEncBOM, "byte order mark")
	}
	if bytes.Contains(raw, []byte{0}) {
		return "", fail(eEncNull, "null byte")
	}
	if !utf8.Valid(raw) {
		return "", fail(eEncUTF8, "not valid utf-8")
	}
	text := string(raw)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = norm.NFC.String(text)
	var b strings.Builder
	for _, r := range text {
		if r == '\n' {
			b.WriteRune(r)
			continue
		}
		if isC0C1Control(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String(), nil
}

func isExtension(key string) bool {
	return strings.HasPrefix(key, extPrefix)
}

func hasZWOrBidi(s string) bool {
	for _, r := range s {
		switch {
		case r >= 0x200B && r <= 0x200D:
			return true
		case r == 0xFEFF:
			return true
		case r >= 0x202A && r <= 0x202E:
			return true
		case r >= 0x2066 && r <= 0x2069:
			return true
		}
	}
	return false
}

func cleanValue(v string) string {
	var b strings.Builder
	for _, r := range v {
		if hasZWOrBidi(string(r)) {
			continue
		}
		b.WriteRune(r)
	}
	return trimRightW3ASpace(b.String())
}

func isC0C1Control(r rune) bool {
	return (r >= 0x00 && r <= 0x1F && r != '\n') || r == 0x7F || (r >= 0x80 && r <= 0x9F)
}

func trimRightW3ASpace(s string) string {
	return strings.TrimRight(s, " \t")
}

func checkToken(tok string) error {
	if hasZWOrBidi(tok) {
		return fail(eTokenInvalid, fmt.Sprintf("%q", tok))
	}
	return nil
}

func checkID(raw string) error {
	if hasZWOrBidi(raw) {
		return fail(eTokenInvalid, fmt.Sprintf("%q", raw))
	}
	return nil
}

func splitLine(line string) (indent int, p pair, err error) {
	stripped := strings.TrimLeft(line, " ")
	indent = len(line) - len(stripped)
	if !strings.Contains(stripped, sep) {
		return 0, pair{}, fail(eStructParse, "no separator: "+line)
	}
	parts := strings.SplitN(stripped, sep, 2)
	key := strings.TrimSpace(parts[0])
	if err := checkToken(key); err != nil {
		return 0, pair{}, err
	}
	rawValue := parts[1]
	return indent, pair{key: key, value: cleanValue(rawValue), raw: rawValue}, nil
}

func parseDoc(text string) (document, error) {
	doc := document{}
	section := "header"
	var cur *entry
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		bare := strings.TrimSpace(line)
		if !strings.Contains(bare, sep) {
			if dst, ok := sectionNames[bare]; ok {
				if err := checkToken(bare); err != nil {
					return doc, err
				}
				section = dst
				cur = nil
				continue
			}
		}
		indent, p, err := splitLine(line)
		if err != nil {
			return doc, err
		}
		switch section {
		case "neng":
			if indent == 0 && (p.key == "ㄕ" || p.key == "ㄓㄠ") {
				if err := checkID(p.raw); err != nil {
					return doc, err
				}
				doc.neng = append(doc.neng, entry{kind: p.key, id: p.value, rawID: p.raw})
				cur = &doc.neng[len(doc.neng)-1]
			} else if cur != nil && indent > 0 {
				cur.fields = append(cur.fields, p)
			} else {
				return doc, fail(eStructParse, "stray line in ㄋㄥ: "+line)
			}
		case "ke":
			if indent == 0 && p.key == "ㄑㄩ" {
				if err := checkID(p.raw); err != nil {
					return doc, err
				}
				doc.ke = append(doc.ke, entry{kind: p.key, id: p.value, rawID: p.raw})
				cur = &doc.ke[len(doc.ke)-1]
			} else if cur != nil && indent > 0 {
				cur.fields = append(cur.fields, p)
			} else {
				return doc, fail(eStructParse, "stray line in ㄎㄜ: "+line)
			}
		case "she":
			doc.she = append(doc.she, p)
		case "zhang":
			doc.zhang = append(doc.zhang, p)
		default:
			doc.header = append(doc.header, p)
		}
	}
	return doc, nil
}

func canonicalizeBytes(raw []byte) ([]byte, error) {
	text, err := normalize(raw)
	if err != nil {
		return nil, err
	}
	doc, err := parseDoc(text)
	if err != nil {
		return nil, err
	}
	return canonicalDoc(doc)
}

func canonicalDoc(doc document) ([]byte, error) {
	lines := []string{}
	h, err := emitScalarBlock(doc.header, coreHeaderOrder)
	if err != nil {
		return nil, err
	}
	lines = append(lines, h...)

	entities, actions := splitEntries(doc.neng)
	if len(doc.neng) > 0 {
		lines = append(lines, "ㄋㄥ")
		if err := requireUniqueEntryIDs(entities); err != nil {
			return nil, err
		}
		if err := requireUniqueEntryIDs(actions); err != nil {
			return nil, err
		}
		sortEntries(entities)
		sortEntries(actions)
		for _, e := range entities {
			lines = append(lines, "ㄕ"+sep+e.id)
			sub, err := emitEntityFields(e.fields)
			if err != nil {
				return nil, err
			}
			lines = append(lines, sub...)
		}
		for _, a := range actions {
			lines = append(lines, "ㄓㄠ"+sep+a.id)
			sub, err := emitEntryFields(a.fields, actionFieldOrder)
			if err != nil {
				return nil, err
			}
			lines = append(lines, sub...)
		}
	}

	if len(doc.ke) > 0 {
		lines = append(lines, "ㄎㄜ")
		if err := requireUniqueEntryIDs(doc.ke); err != nil {
			return nil, err
		}
		sortEntries(doc.ke)
		for _, b := range doc.ke {
			lines = append(lines, "ㄑㄩ"+sep+b.id)
			sub, err := emitEntryFields(b.fields, blockFieldOrder)
			if err != nil {
				return nil, err
			}
			lines = append(lines, sub...)
		}
	}

	if len(doc.she) > 0 {
		lines = append(lines, "ㄕㄜ")
		prefs, err := emitScalarBlock(doc.she, prefOrder)
		if err != nil {
			return nil, err
		}
		lines = append(lines, prefs...)
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func splitEntries(entries []entry) (entities []entry, actions []entry) {
	for _, e := range entries {
		if e.kind == "ㄕ" {
			entities = append(entities, e)
		} else if e.kind == "ㄓㄠ" {
			actions = append(actions, e)
		}
	}
	return entities, actions
}

func sortEntries(entries []entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].id < entries[j].id
	})
}

func requireUniqueEntryIDs(entries []entry) error {
	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.id] {
			return fail(eStructDupID, e.id)
		}
		seen[e.id] = true
	}
	return nil
}

func emitScalarBlock(pairs []pair, fixedOrder []string) ([]string, error) {
	seen := map[string]bool{}
	for _, p := range pairs {
		if seen[p.key] {
			return nil, fail(eStructDupKey, p.key)
		}
		seen[p.key] = true
	}
	order := indexMap(fixedOrder)
	core := []pair{}
	ext := []pair{}
	for _, p := range pairs {
		if isExtension(p.key) {
			ext = append(ext, p)
			continue
		}
		if _, ok := order[p.key]; !ok {
			return nil, fail(eTokenUnknownCore, p.key)
		}
		core = append(core, p)
	}
	sort.SliceStable(core, func(i, j int) bool { return order[core[i].key] < order[core[j].key] })
	sortPairsByKey(ext)
	out := []string{}
	for _, p := range append(core, ext...) {
		out = append(out, p.key+sep+emitValue(p.key, p.value))
	}
	return out, nil
}

func emitEntryFields(fields []pair, fixedOrder []string) ([]string, error) {
	seenScalar := map[string]bool{}
	seenInput := map[string]bool{}
	for _, f := range fields {
		if f.key == "ㄕㄡ" {
			id := strings.SplitN(f.raw, listSep, 2)[0]
			if err := checkID(id); err != nil {
				return nil, err
			}
			id = cleanValue(id)
			if seenInput[id] {
				return nil, fail(eStructDupID, id)
			}
			seenInput[id] = true
			continue
		}
		if seenScalar[f.key] {
			return nil, fail(eStructDupKey, f.key)
		}
		seenScalar[f.key] = true
	}
	order := indexMap(fixedOrder)
	core := []indexedPair{}
	ext := []pair{}
	for i, f := range fields {
		if isExtension(f.key) {
			ext = append(ext, f)
			continue
		}
		if _, ok := order[f.key]; !ok {
			return nil, fail(eTokenUnknownCore, f.key)
		}
		core = append(core, indexedPair{idx: i, pair: f})
	}
	sort.SliceStable(core, func(i, j int) bool {
		a, b := core[i], core[j]
		if order[a.key] != order[b.key] {
			return order[a.key] < order[b.key]
		}
		return a.idx < b.idx
	})
	sortPairsByKey(ext)
	out := []string{}
	for _, p := range core {
		out = append(out, "  "+p.key+sep+emitValue(p.key, p.value))
	}
	for _, p := range ext {
		out = append(out, "  "+p.key+sep+emitValue(p.key, p.value))
	}
	return out, nil
}

type indexedPair struct {
	idx int
	pair
}

// emitEntityFields emits a ㄕ entity's sub-fields (§8.1): core fields keep
// appearance order (UI semantic); ㄝ extension fields are moved after core and
// sorted by key, unifying with action/block records. Any duplicate key rejects.
func emitEntityFields(fields []pair) ([]string, error) {
	seen := map[string]bool{}
	core := []pair{}
	ext := []pair{}
	for _, f := range fields {
		if seen[f.key] {
			return nil, fail(eStructDupKey, f.key)
		}
		seen[f.key] = true
		if isExtension(f.key) {
			ext = append(ext, f)
			continue
		}
		core = append(core, f)
	}
	sortPairsByKey(ext)
	out := []string{}
	for _, p := range append(core, ext...) {
		out = append(out, "  "+p.key+sep+emitValue(p.key, p.value))
	}
	return out, nil
}

func emitValue(key, value string) string {
	if !setKeys[key] || !strings.Contains(value, listSep) {
		return value
	}
	parts := strings.Split(value, listSep)
	sort.Strings(parts)
	return strings.Join(parts, listSep)
}

func indexMap(keys []string) map[string]int {
	out := map[string]int{}
	for i, k := range keys {
		out[k] = i
	}
	return out
}

func sortPairsByKey(pairs []pair) {
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})
}

func parseVersion(s string, runtimeMajor int) (int, int, error) {
	plain := regexp.MustCompile(`^[0-9]+$`)
	semver := regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	var major, minor int
	if plain.MatchString(s) {
		if len(s) > 1 && s[0] == '0' {
			return 0, 0, fail(eVersionFormat, "leading zero: "+s)
		}
		fmt.Sscanf(s, "%d", &major)
	} else if semver.MatchString(s) {
		parts := strings.Split(s, ".")
		if (len(parts[0]) > 1 && parts[0][0] == '0') || (len(parts[1]) > 1 && parts[1][0] == '0') {
			return 0, 0, fail(eVersionFormat, "leading zero: "+s)
		}
		fmt.Sscanf(parts[0], "%d", &major)
		fmt.Sscanf(parts[1], "%d", &minor)
	} else {
		return 0, 0, fail(eVersionFormat, "bad version: "+s)
	}
	if major > runtimeMajor {
		return 0, 0, fail(eVersionMajor, fmt.Sprintf("major %d > runtime %d", major, runtimeMajor))
	}
	return major, minor, nil
}

type rlkrDoc struct {
	fields []pair
	revs   []string
}

func canonicalRL(raw []byte) ([]byte, error) {
	doc, err := parseRLKR(raw)
	if err != nil {
		return nil, err
	}
	lines, err := emitRLKRFields(doc.fields, rlOrder)
	if err != nil {
		return nil, err
	}
	if len(doc.revs) > 0 {
		lines = append(lines, "撤銷項：")
		revs := uniqueStrings(doc.revs)
		sort.SliceStable(revs, func(i, j int) bool {
			return revLess(revs[i], revs[j])
		})
		for _, r := range revs {
			lines = append(lines, "  "+r)
		}
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func canonicalKR(raw []byte) ([]byte, error) {
	doc, err := parseRLKR(raw)
	if err != nil {
		return nil, err
	}
	lines, err := emitRLKRFields(doc.fields, krOrder)
	if err != nil {
		return nil, err
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func parseRLKR(raw []byte) (rlkrDoc, error) {
	text, err := normalize(raw)
	if err != nil {
		return rlkrDoc{}, err
	}
	doc := rlkrDoc{}
	inRevs := false
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.TrimSpace(line) == "撤銷項：" {
			inRevs = true
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if inRevs && indent > 0 {
			doc.revs = append(doc.revs, cleanValue(strings.TrimSpace(line)))
			continue
		}
		inRevs = false
		_, p, err := splitLine(line)
		if err != nil {
			return rlkrDoc{}, err
		}
		if p.key == "ㄓㄥ" {
			continue
		}
		doc.fields = append(doc.fields, p)
	}
	return doc, nil
}

func emitRLKRFields(fields []pair, fixedOrder []string) ([]string, error) {
	seen := map[string]bool{}
	order := indexMap(fixedOrder)
	core := []pair{}
	ext := []pair{}
	for _, f := range fields {
		if seen[f.key] {
			return nil, fail(eStructDupKey, f.key)
		}
		seen[f.key] = true
		if isExtension(f.key) {
			ext = append(ext, f)
			continue
		}
		if _, ok := order[f.key]; !ok {
			return nil, fail(eTokenUnknownCore, f.key)
		}
		core = append(core, f)
	}
	sort.SliceStable(core, func(i, j int) bool { return order[core[i].key] < order[core[j].key] })
	sortPairsByKey(ext)
	out := []string{}
	for _, p := range append(core, ext...) {
		out = append(out, p.key+sep+p.value)
	}
	return out, nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func revLess(a, b string) bool {
	at, an, ar := revSortKey(a)
	bt, bn, br := revSortKey(b)
	if at != bt {
		return at < bt
	}
	if an != bn {
		return an < bn
	}
	return ar < br
}

func revSortKey(entry string) (target, notBefore, reason string) {
	parts := strings.Split(entry, listSep)
	if len(parts) > 0 {
		target = parts[0]
	}
	for _, p := range parts[1:] {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "not_before":
			notBefore = kv[1]
		case "reason":
			reason = kv[1]
		}
	}
	return target, notBefore, reason
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

const nestedFixture = "ㄊㄡ：W3A v0.3\nㄏㄠ：com.example.board\nㄉㄞ：1.0\n" +
	"ㄏㄡ：gdrive://BOARD_FILE_ID\nㄈㄢ：shared\n\n" +
	"ㄋㄥ\n" +
	"ㄓㄠ：submit_message\n  ㄕㄡ：text｜ㄐㄩ\n  ㄘ：ㄔㄥ\n  ㄓ：/messages\n" +
	"  ㄍㄞ：yes\n  ㄖㄣ：yes\n  ㄉㄜ：message\n" +
	"ㄕ：message\n  id：ㄐㄩ\n  author：ㄐㄩ\n  text：ㄐㄩ\n  created_at：ㄋㄞ\n" +
	"ㄓㄠ：read_messages\n  ㄘ：ㄎㄢ\n  ㄓ：/messages\n  ㄍㄞ：no\n  ㄉㄜ：list｜message\n\n" +
	"ㄎㄜ\n" +
	"ㄑㄩ：main_board\n  ㄍㄜ：ㄆㄤ\n  ㄩㄢ：read_messages\n" +
	"  ㄊㄠ：submit_message｜react｜search_messages\n  ㄊㄞ：ㄗㄞ｜ㄇㄢ\n\n" +
	"ㄕㄜ\n" +
	"ㄘㄤ：submit_message｜react\nㄋㄛ：react｜submit_message\nㄎㄞ：main_board\n"

const rlFixture = "ㄓㄜ：com.example.publisher\nㄝㄕㄜ：3\nㄎㄟ：ed25519:CUR\nㄑㄢ：full\n" +
	"ㄔㄣ：2026-06-27T00:00:00Z\n撤銷項：\n" +
	"  ed25519:LEAK｜reason=compromise｜not_before=2026-06-26T12:00:00Z\n" +
	"  com.example.board@2｜reason=superseded\nㄓㄥ：SIG\n"

const krFixture = "ㄓㄜ：com.example.publisher\nㄔㄣ：2026-06-27T00:00:00Z\n" +
	"新鑰：ed25519:NEW\n舊鑰：ed25519:OLD\nㄓㄥ：SIGOLD\n"

// Extension sub-field fixtures (§8.1 unified rule): ㄝ fields inside a record are
// moved after core fields and sorted by key. ext keys deliberately out of order.
const actionExtFixture = "ㄊㄡ：W3A v0.3\nㄏㄠ：com.example.x\nㄉㄞ：1.0\n\n" +
	"ㄋㄥ\nㄓㄠ：a\n  ㄝㄗㄜ：z\n  ㄓ：/x\n  ㄝㄇㄜ：m\n  ㄘ：ㄎㄢ\n"

const entityExtFixture = "ㄊㄡ：W3A v0.3\nㄏㄠ：com.example.x\nㄉㄞ：1.0\n\n" +
	"ㄋㄥ\nㄕ：message\n  ㄝㄗㄜ：z\n  id：ㄐㄩ\n  ㄝㄇㄜ：m\n  author：ㄐㄩ\n"

func cmdGenVectors(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: w3a gen-vectors")
	}
	root, err := findConformanceRoot()
	if err != nil {
		return err
	}
	vectors, err := buildVectors()
	if err != nil {
		return err
	}
	for rel, data := range vectors {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	fmt.Println("vectors regenerated under", filepath.Join(root, "vectors"))
	return nil
}

func buildVectors() (map[string][]byte, error) {
	out := map[string][]byte{}
	version, err := versionCases()
	if err != nil {
		return nil, err
	}
	out["vectors/version/cases.json"] = version
	reject, err := rejectCases()
	if err != nil {
		return nil, err
	}
	out["vectors/reject/cases.json"] = reject
	if err := addDocVector(out, "vectors/canonical/header-basic", "input.w3a",
		"ㄏㄠ：com.example.board\nㄊㄡ：W3A v0.3\nㄈㄢ：shared\nㄉㄞ：1.0\nㄏㄡ：gdrive://BOARD_FILE_ID\n", canonicalizeBytes); err != nil {
		return nil, err
	}
	if err := addDocVector(out, "vectors/extension/header-ext", "input.w3a",
		"ㄊㄡ：W3A v0.3\nㄏㄠ：com.example.board\nㄉㄞ：1.0\nㄝㄖㄜ：https://example.com/rl\nㄝㄎㄜ：ed25519:NEWKEYFP\n", canonicalizeBytes); err != nil {
		return nil, err
	}
	if err := addDocVector(out, "vectors/canonical/board-nested", "input.w3a", nestedFixture, canonicalizeBytes); err != nil {
		return nil, err
	}
	if err := addDocVector(out, "vectors/extension/action-ext", "input.w3a", actionExtFixture, canonicalizeBytes); err != nil {
		return nil, err
	}
	if err := addDocVector(out, "vectors/extension/entity-ext", "input.w3a", entityExtFixture, canonicalizeBytes); err != nil {
		return nil, err
	}
	if err := addDocVector(out, "vectors/rl/basic", "input.w3a-rl", rlFixture, canonicalRL); err != nil {
		return nil, err
	}
	if err := addDocVector(out, "vectors/kr/basic", "input.w3a-kr", krFixture, canonicalKR); err != nil {
		return nil, err
	}
	if err := addSignatureVector(out); err != nil {
		return nil, err
	}
	return out, nil
}

func addDocVector(out map[string][]byte, base, inputName, src string, fn func([]byte) ([]byte, error)) error {
	canon, err := fn([]byte(src))
	if err != nil {
		return err
	}
	out[filepath.Join(base, inputName)] = []byte(src)
	out[filepath.Join(base, "canonical.bytes")] = canon
	out[filepath.Join(base, "sha256.txt")] = []byte(sha256Hex(canon) + "\n")
	return nil
}

func versionCases() ([]byte, error) {
	type item struct {
		Input       string `json:"input"`
		Expect      string `json:"expect,omitempty"`
		ExpectError string `json:"expect_error,omitempty"`
	}
	type payload struct {
		RuntimeMajor int    `json:"runtime_major"`
		Cases        []item `json:"cases"`
	}
	p := payload{RuntimeMajor: 1}
	for _, s := range []string{"1", "1.0", "1.1", "10", "2", "01", "1.01", "1.", "1.x", ".1", ""} {
		maj, min, err := parseVersion(s, 1)
		if err != nil {
			var we w3aError
			if errors.As(err, &we) {
				p.Cases = append(p.Cases, item{Input: s, ExpectError: we.code})
				continue
			}
			return nil, err
		}
		p.Cases = append(p.Cases, item{Input: s, Expect: fmt.Sprintf("%d.%d", maj, min)})
	}
	return marshalJSON(p)
}

func rejectCases() ([]byte, error) {
	type item struct {
		Name        string `json:"name"`
		Input       string `json:"input,omitempty"`
		InputB64    string `json:"input_b64,omitempty"`
		ExpectError string `json:"expect_error"`
	}
	type payload struct {
		Cases []item `json:"cases"`
	}
	cases := []item{
		{Name: "bom", InputB64: "77u/", ExpectError: eEncBOM},
		{Name: "null", InputB64: "YQBi", ExpectError: eEncNull},
	}
	for _, tc := range []struct {
		name string
		src  string
		code string
	}{
		{"dup-key", "ㄊㄡ：a\nㄊㄡ：b\n", eStructDupKey},
		{"no-separator", "ㄊㄡ a\n", eStructParse},
		{"unknown-core", "ㄓㄠ：x\n", eTokenUnknownCore},
		{"zero-width-in-token", "ㄊㄡ\u200b：x\n", eTokenInvalid},
		{"zero-width-in-action-id", "ㄋㄥ\nㄓㄠ：sub\u200bmit\n  ㄘ：ㄎㄢ\n", eTokenInvalid},
		{"dup-action-id", "ㄋㄥ\nㄓㄠ：a\n  ㄘ：ㄎㄢ\nㄓㄠ：a\n  ㄘ：ㄎㄢ\n", eStructDupID},
		{"dup-input-id", "ㄋㄥ\nㄓㄠ：a\n  ㄕㄡ：text｜ㄐㄩ\n  ㄕㄡ：text｜ㄐㄩ\n", eStructDupID},
		{"dup-entity-field", "ㄋㄥ\nㄕ：message\n  id：ㄐㄩ\n  id：ㄗㄤ\n", eStructDupKey},
		{"rl-dup-key", "ㄓㄜ：a\nㄓㄜ：b\nㄎㄟ：k\nㄔㄣ：t\nㄑㄢ：full\nㄝㄕㄜ：1\n", eStructDupKey},
		{"rl-unknown-core", "ㄓㄜ：a\n未知：x\nㄎㄟ：k\nㄔㄣ：t\nㄑㄢ：full\nㄝㄕㄜ：1\n", eTokenUnknownCore},
	} {
		err := expectReject(tc.src, tc.code)
		if err != nil {
			return nil, err
		}
		cases = append(cases, item{Name: tc.name, Input: tc.src, ExpectError: tc.code})
	}
	return marshalJSON(payload{Cases: cases})
}

func expectReject(src, code string) error {
	_, err := canonicalizeBytes([]byte(src))
	if err == nil {
		_, err = canonicalRL([]byte(src))
	}
	if err == nil {
		return fmt.Errorf("expected %s but got success", code)
	}
	var we w3aError
	if !errors.As(err, &we) {
		return err
	}
	if we.code != code {
		return fmt.Errorf("expected %s got %s", code, we.code)
	}
	return nil
}

func addSignatureVector(out map[string][]byte) error {
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	msg, err := canonicalizeBytes([]byte(nestedFixture))
	if err != nil {
		return err
	}
	sig := ed25519.Sign(priv, msg)
	if !ed25519.Verify(pub, msg, sig) {
		return errors.New("ed25519 self-verify failed")
	}
	base := "vectors/signature/board-nested"
	out[filepath.Join(base, "seed.hex")] = []byte(hex.EncodeToString(seed) + "\n")
	out[filepath.Join(base, "public.hex")] = []byte(hex.EncodeToString(pub) + "\n")
	out[filepath.Join(base, "message.canonical")] = msg
	out[filepath.Join(base, "signature.hex")] = []byte(hex.EncodeToString(sig) + "\n")
	out[filepath.Join(base, "README.md")] = []byte(signatureReadme)
	return nil
}

const signatureReadme = "# signature/board-nested — Ed25519 golden signature\n\n" +
	"**TEST KEY — DO NOT USE IN PRODUCTION.** This keypair exists only to pin a\n" +
	"deterministic golden signature for cross-implementation conformance.\n\n" +
	"- `seed.hex` — 32-byte test seed (0x42 repeated).\n" +
	"- `public.hex` — public key derived from the seed using Go `crypto/ed25519`.\n" +
	"- `message.canonical` — the exact bytes signed: the canonical form of the\n" +
	"  `canonical/board-nested` document.\n" +
	"- `signature.hex` — Ed25519 signature (deterministic for a given key+message).\n\n" +
	"Generated by `go run ./tools/w3a gen-vectors`.\n"

func marshalJSON(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func cmdBundleCheck(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: w3a bundle-check")
	}
	specRoot, err := findSpecRoot()
	if err != nil {
		return err
	}
	if err := checkJSON(filepath.Join(specRoot, "skill.json")); err != nil {
		return err
	}
	if err := checkJSON(filepath.Join(specRoot, "adapters", "haler", "haler.skill.json")); err != nil {
		return err
	}
	if err := checkOpenAIYAML(filepath.Join(specRoot, "skills", "w3a-spec", "agents", "openai.yaml")); err != nil {
		return err
	}
	if err := checkManifestPaths(specRoot); err != nil {
		return err
	}
	if err := checkSpecHashes(specRoot); err != nil {
		return err
	}
	if err := checkVectors(specRoot); err != nil {
		return err
	}
	if err := checkBuilder(specRoot); err != nil {
		return err
	}
	if err := scanForbidden(specRoot); err != nil {
		return err
	}
	fmt.Println("bundle check ok")
	return nil
}

func checkJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func checkOpenAIYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(data)
	for _, required := range []string{"interface:", "display_name:", "short_description:", "default_prompt:"} {
		if !strings.Contains(s, required) {
			return fmt.Errorf("%s missing %s", path, required)
		}
	}
	return nil
}

func checkManifestPaths(root string) error {
	data, err := os.ReadFile(filepath.Join(root, "skill.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Entrypoints map[string]string `json:"entrypoints"`
		Adapters    map[string]string `json:"adapters"`
		Paths       map[string]string `json:"paths"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	for _, group := range []map[string]string{manifest.Entrypoints, manifest.Adapters, manifest.Paths} {
		for _, rel := range group {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				return fmt.Errorf("manifest path missing: %s", rel)
			}
		}
	}
	return nil
}

func checkSpecHashes(root string) error {
	a, err := os.ReadFile(filepath.Join(root, "W3A-SPEC.md"))
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(root, "skills", "w3a-spec", "references", "W3A-SPEC.md"))
	if err != nil {
		return err
	}
	if !bytes.Equal(a, b) {
		return errors.New("W3A-SPEC.md and skill reference differ")
	}
	return nil
}

func checkVectors(root string) error {
	expected, err := buildVectors()
	if err != nil {
		return err
	}
	conf, err := findConformanceRoot()
	if err != nil {
		conf = filepath.Join(root, "conformance")
	}
	for rel, want := range expected {
		got, err := os.ReadFile(filepath.Join(conf, rel))
		if err != nil {
			return fmt.Errorf("vector missing: %s", rel)
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("vector drift: %s", rel)
		}
	}
	return nil
}

func checkBuilder(root string) error {
	if err := checkJSON(filepath.Join(root, "builder", "answers.schema.json")); err != nil {
		return err
	}
	for _, rel := range []string{
		"builder/templates/board.json",
		"builder/templates/task_list.json",
		"builder/templates/feedback_form.json",
		"builder/templates/product_showcase.json",
		"builder/templates/mobile_product_app.json",
		"builder/templates/catalog.json",
		"builder/examples/board.answers.json",
		"builder/examples/product_showcase.answers.json",
		"builder/examples/mobile_product_app.answers.json",
		"builder/examples/custom_generic.answers.json",
	} {
		if err := checkJSON(filepath.Join(root, rel)); err != nil {
			return err
		}
	}
	tmp, err := os.MkdirTemp("", "w3a-builder-check-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	answersPath := filepath.Join(root, "builder", "examples", "board.answers.json")
	result, _, _, err := buildFromAnswersFile(root, answersPath, false)
	if err != nil {
		return fmt.Errorf("builder sample draft failed: %w", err)
	}
	if _, err := canonicalizeBytes(result.Contract); err != nil {
		return fmt.Errorf("builder sample canonical failed: %w", err)
	}
	customResult, _, _, err := buildFromAnswersFile(root, filepath.Join(root, "builder", "examples", "custom_generic.answers.json"), false)
	if err != nil {
		return fmt.Errorf("custom_generic sample draft failed: %w", err)
	}
	if _, err := canonicalizeBytes(customResult.Contract); err != nil {
		return fmt.Errorf("custom_generic sample canonical failed: %w", err)
	}
	productResult, _, _, err := buildFromAnswersFile(root, filepath.Join(root, "builder", "examples", "product_showcase.answers.json"), false)
	if err != nil {
		return fmt.Errorf("product_showcase sample draft failed: %w", err)
	}
	if _, err := canonicalizeBytes(productResult.Contract); err != nil {
		return fmt.Errorf("product_showcase sample canonical failed: %w", err)
	}
	mobileResult, _, _, err := buildFromAnswersFile(root, filepath.Join(root, "builder", "examples", "mobile_product_app.answers.json"), false)
	if err != nil {
		return fmt.Errorf("mobile_product_app sample draft failed: %w", err)
	}
	if _, err := canonicalizeBytes(mobileResult.Contract); err != nil {
		return fmt.Errorf("mobile_product_app sample canonical failed: %w", err)
	}
	testResult, _, _, err := buildFromAnswersFile(root, answersPath, true)
	if err != nil {
		return fmt.Errorf("builder sample test-sign failed: %w", err)
	}
	trust, err := classifyTrust(testResult.Contract, "", "")
	if err != nil {
		return err
	}
	if trust["state"] != "test_signed" || trust["code"] != "E-TRUST-TESTKEY" {
		return fmt.Errorf("test-sign trust mismatch: %+v", trust)
	}
	mockPath := filepath.Join(tmp, "board.mock-demo.json")
	_, tmpl, answers, err := buildFromAnswersFile(root, answersPath, false)
	if err != nil {
		return err
	}
	demo, err := mockDemoJSON(tmpl, answers, result)
	if err != nil {
		return err
	}
	if err := os.WriteFile(mockPath, demo, 0o644); err != nil {
		return err
	}
	if err := checkJSON(mockPath); err != nil {
		return err
	}
	secretReject := filepath.Join(tmp, "reject-secret.answers.json")
	boardFixture, err := os.ReadFile(filepath.Join(root, "builder", "examples", "board.answers.json"))
	if err != nil {
		return err
	}
	secretFixture := strings.ReplaceAll(string(boardFixture), "mock://board", "https://example.invalid/messages?access_"+"to"+"ken"+"=abc123")
	if err := os.WriteFile(secretReject, []byte(secretFixture), 0o644); err != nil {
		return err
	}
	rejects := map[string]string{
		secretReject: eBuilderSecret,
		filepath.Join(root, "builder", "examples", "reject-code-owned.answers.json"):     eBuilderCodeOwned,
		filepath.Join(root, "builder", "examples", "reject-llm-human-only.answers.json"): eBuilderHumanOnly,
	}
	for file, code := range rejects {
		_, _, _, err := buildFromAnswersFile(root, file, false)
		if err == nil {
			return fmt.Errorf("builder reject fixture passed unexpectedly: %s", file)
		}
		var we w3aError
		if !errors.As(err, &we) || we.code != code {
			return fmt.Errorf("builder reject fixture %s got %v, want %s", file, err, code)
		}
	}
	return nil
}

func scanForbidden(root string) error {
	forbidden := []string{
		"TO" + "DO",
		"[TO" + "DO",
		"鍵後" + "一空格",
		"needs " + "spec",
		"confirm against " + "spec",
		"ㄖ" + "ㄨ",
		"ㄒ" + "ㄩ",
		"py" + "thon3",
		"w3a_min" + ".py",
		"gen_vectors" + ".py",
		"ed25519_ref" + ".py",
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(path, string(filepath.Separator)+".DS_Store") {
			return nil
		}
		if strings.HasSuffix(path, ".pyc") {
			return nil
		}
		// Skip compiled build artifacts only. These are never authored source:
		// the Go CLI binary embeds the program's own string table (including this
		// scanner's forbidden markers), so scanning it would false-positive. They
		// are also listed in .gitignore so they never enter the published bundle.
		// Every other file type is still scanned, so marker coverage over authored
		// source is unchanged (no allowlist, no content sniffing).
		if base := d.Name(); base == "w3a" ||
			strings.HasSuffix(base, ".test") ||
			strings.HasSuffix(base, ".exe") ||
			strings.HasSuffix(base, ".o") ||
			strings.HasSuffix(base, ".out") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := string(data)
		for _, bad := range forbidden {
			if strings.Contains(s, bad) {
				return fmt.Errorf("forbidden marker %q in %s", bad, path)
			}
		}
		return nil
	})
}

func findConformanceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for _, p := range []string{cwd, filepath.Join(cwd, "W3A_SPEC", "conformance")} {
		if _, err := os.Stat(filepath.Join(p, "README.md")); err == nil {
			if _, err := os.Stat(filepath.Join(p, "vectors")); err == nil {
				return p, nil
			}
		}
	}
	root, err := findSpecRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "conformance"), nil
}

func findSpecRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	candidates := []string{cwd, filepath.Join(cwd, "W3A_SPEC"), filepath.Dir(cwd), filepath.Join(filepath.Dir(cwd), "W3A_SPEC")}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "W3A-SPEC.md")); err == nil {
			if _, err := os.Stat(filepath.Join(c, "skill.json")); err == nil {
				return c, nil
			}
		}
	}
	return "", errors.New("could not locate W3A_SPEC root")
}
