package main

// Block 3: the operate execution layer.
//
//   wa3 operate <file.tdy> --action <id> [--input k=v ...] [--confirm]
//              [--trusted-pub <hex>] [--data-dir <dir>] [--mock-demo <seed.json>]
//
// operate is the first command that actually touches a backend. It enforces the
// post-confirmation half of the spec that §29 leaves to Runtime/Host:
//
//   1. Verify trust against the canonical signature. Only signed_trusted may
//      operate; unsigned_draft, signed_untrusted_key, test_signed, revoked, and
//      sig_mismatch are all refused (E-OPERATE-UNTRUSTED).
//   2. Re-read the action from the *verified* contract (never from caller input).
//   3. Reject undeclared inputs and re-scan input values for secrets, so a token
//      can never enter through operate (defense in depth on the build-time gate).
//   4. Core-side confirmation: a mutating action (ㄍㄞ=yes) or any action that
//      declares ㄖㄣ=yes requires an explicit --confirm. The renderer cannot
//      satisfy this; it is enforced here in core.
//   5. Resolve ㄏㄡ to a Provider and dispatch read/write/search by verb.
//   6. Emit a JSON result. Backend credentials never exist in this path.

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// verb (ㄘ) values understood by operate, mapped to provider operations.
const (
	verbRead   = "ㄎㄢ" // read / view
	verbSubmit = "ㄔㄥ" // create / submit (mutating)
	verbPress  = "ㄢ"  // act / react (mutating)
	verbSet    = "ㄗㄜ" // set / update (mutating)
	verbSearch = "ㄙㄡ" // search (read)
	verbDelete = "ㄕㄢ" // delete (mutating)
)

type operateOutput struct {
	Action   string `json:"action"`
	Verb     string `json:"verb"`
	Resource string `json:"resource"`
	Backend  string `json:"backend"`
	Mutated  bool   `json:"mutated"`
	Status   string `json:"status"`
	Result   any    `json:"result"`
}

func cmdOperate(args []string) error {
	var file, action, trustedPub, dataDir, seedPath string
	confirm := false
	inputs := record{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--action":
			i++
			if i >= len(args) {
				return errors.New("missing value for --action")
			}
			action = args[i]
		case "--input":
			i++
			if i >= len(args) {
				return errors.New("missing value for --input")
			}
			k, v, ok := strings.Cut(args[i], "=")
			if !ok || k == "" {
				return fail(eOperateInput, "--input must be key=value: "+args[i])
			}
			inputs[k] = v
		case "--confirm":
			confirm = true
		case "--trusted-pub":
			i++
			if i >= len(args) {
				return errors.New("missing value for --trusted-pub")
			}
			trustedPub = args[i]
		case "--data-dir":
			i++
			if i >= len(args) {
				return errors.New("missing value for --data-dir")
			}
			dataDir = args[i]
		case "--mock-demo":
			i++
			if i >= len(args) {
				return errors.New("missing value for --mock-demo")
			}
			seedPath = args[i]
		default:
			if strings.HasPrefix(args[i], "--") || file != "" {
				return fmt.Errorf("unexpected operate arg: %s", args[i])
			}
			file = args[i]
		}
	}
	if file == "" || action == "" {
		return errors.New("usage: wa3 operate <file.tdy> --action <id> [--input k=v ...] [--confirm] [--trusted-pub <hex>] [--data-dir <dir>] [--mock-demo <seed.json>]")
	}
	if dataDir == "" {
		dataDir = "./wa3-data"
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// (1) Trust gate — reuse the canonical verifier. Only a pinned, signed,
	// non-revoked contract may operate.
	status, err := classifyTrust(raw, trustedPub, "")
	if err != nil {
		return err
	}
	if status["state"] != "signed_trusted" {
		hint := ""
		if status["state"] == "signed_untrusted_key" {
			hint = " (pin the publisher key with --trusted-pub <hex>)"
		}
		return fail(eOperateTrust, fmt.Sprintf("refused: trust state %s [%s]%s", status["state"], status["code"], hint))
	}

	// (2) Re-read the action from the verified contract.
	text, err := normalize(raw)
	if err != nil {
		return err
	}
	doc, err := parseDoc(text)
	if err != nil {
		return err
	}
	act, ok := findAction(doc, action)
	if !ok {
		return fail(eOperateAction, "action not found in verified contract: "+action)
	}
	verb := fieldValue(act, "ㄘ")
	target := fieldValue(act, "ㄓ")
	mutates := fieldValue(act, "ㄍㄞ") == "yes"
	declaredConfirm := fieldValue(act, "ㄖㄣ") == "yes"

	// (3) Validate inputs against the declared ㄕㄡ set and re-scan for secrets.
	declared := declaredInputs(act)
	for k, v := range inputs {
		if !declared[k] {
			return fail(eOperateInput, "undeclared input for "+action+": "+k)
		}
		if err := scanSecretString(v, "input."+k, false); err != nil {
			return err
		}
	}

	// (4) Core-side confirmation for mutating / confirm-required actions.
	if (mutates || declaredConfirm) && !confirm {
		return fail(eOperateConfirm, "action "+action+" mutates the backend; re-run with --confirm to authorize")
	}

	// (5) Resolve the verified backend handle to a provider and dispatch.
	backend := strings.TrimSpace(headerValue(doc, "ㄏㄡ"))
	if backend == "" {
		return fail(eOperateProvider, "verified contract has no backend (ㄏㄡ)")
	}
	prov, err := newProvider(backend, dataDir, seedPath)
	if err != nil {
		return err
	}
	resource := resourceKey(target)

	var result any
	switch {
	case verb == verbSearch:
		result, err = prov.Search(resource, inputs)
	case mutates:
		result, err = prov.Write(resource, inputs)
	default:
		result, err = prov.Read(resource, inputs)
	}
	if err != nil {
		return err
	}

	// (6) Emit the result. Nothing in this struct can carry a credential.
	out, err := marshalJSON(operateOutput{
		Action:   action,
		Verb:     verb,
		Resource: resource,
		Backend:  backend,
		Mutated:  mutates,
		Status:   "ok",
		Result:   result,
	})
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

// findAction returns the ㄓㄠ action entry with the given id from the verified
// neng section.
func findAction(doc document, id string) (entry, bool) {
	for _, e := range doc.neng {
		if e.kind == "ㄓㄠ" && e.id == id {
			return e, true
		}
	}
	return entry{}, false
}

func fieldValue(e entry, key string) string {
	for _, p := range e.fields {
		if p.key == key {
			return strings.TrimSpace(p.value)
		}
	}
	return ""
}

// declaredInputs returns the set of input names declared on an action via its
// ㄕㄡ fields (each "name｜type").
func declaredInputs(e entry) map[string]bool {
	out := map[string]bool{}
	for _, p := range e.fields {
		if p.key != "ㄕㄡ" {
			continue
		}
		name := p.value
		if i := strings.Index(name, listSep); i >= 0 {
			name = name[:i]
		}
		out[strings.TrimSpace(name)] = true
	}
	return out
}
