package main

// Block 1: the interactive "where does this app's data live?" collection step.
//
//   wa3 init --answers <answers.json> [--backend <handle>] [--out <answers.json>]
//
// This is the still-unimplemented init/wizard reduced to its one human-only
// decision: the backend handle that becomes the contract's ㄏㄡ. Per the spec,
// answers are system-written, not hand-authored, so init *merges* a validated
// backend into an existing answers JSON rather than asking the user to edit it.
//
//   * --backend supplies the handle non-interactively (tests, scripts).
//   * Without --backend, init prompts once on stdin (works in a pipe too).
//
// Validation is the substance of this step:
//   * the handle must match the scheme allowlist (validateBackendHandle:
//     mock://, gdrive://, api://, https://, local://);
//   * token/secret shapes are rejected with E-VALUE-SECRET (scanSecretString);
//   * the value is treated as a human-only, user_confirmed decision.
//
// The merged answers then flow through `build` unchanged into ㄏㄡ.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

func cmdInit(args []string) error {
	var answersPath, backend, outPath string
	backendSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--answers":
			i++
			if i >= len(args) {
				return errors.New("missing value for --answers")
			}
			answersPath = args[i]
		case "--backend":
			i++
			if i >= len(args) {
				return errors.New("missing value for --backend")
			}
			backend = args[i]
			backendSet = true
		case "--out":
			i++
			if i >= len(args) {
				return errors.New("missing value for --out")
			}
			outPath = args[i]
		default:
			return fmt.Errorf("unknown init arg: %s", args[i])
		}
	}
	if answersPath == "" {
		return errors.New("usage: wa3 init --answers <answers.json> [--backend <handle>] [--out <answers.json>]")
	}
	if outPath == "" {
		outPath = answersPath
	}

	// Load the existing answers as a raw map so unrelated fields are preserved
	// byte-for-byte through the merge.
	data, err := os.ReadFile(answersPath)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: %w", answersPath, err)
	}

	// Collect the backend handle: flag first, otherwise an interactive prompt.
	if !backendSet {
		backend, err = promptBackend(os.Stdin, os.Stderr)
		if err != nil {
			return err
		}
	}
	backend = strings.TrimSpace(backend)
	if backend == "" {
		return fail(eBuilderHumanOnly, "backend is a required human-only decision; none provided")
	}

	// Validation gates (the human-only decision must still pass code-owned checks).
	if err := scanSecretString(backend, "app.backend", false); err != nil {
		return err // E-VALUE-SECRET
	}
	if err := validateBackendHandle(backend); err != nil {
		return err // E-BUILDER-SCHEMA
	}

	// Merge into app.backend, marking provenance for the audit trail.
	app, _ := doc["app"].(map[string]any)
	if app == nil {
		app = map[string]any{}
	}
	app["backend"] = backend
	doc["app"] = app

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, append(out, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "backend recorded (owner=human_only, provenance=user_confirmed): %s -> %s\n", backend, outPath)
	return nil
}

// promptBackend asks once for a backend handle. It reads a single line so it
// works both for an interactive terminal and a piped answer in tests.
func promptBackend(in *os.File, errw *os.File) (string, error) {
	fmt.Fprintln(errw, "Where should this app's data live?")
	fmt.Fprintln(errw, "Enter a backend handle (e.g. mock://board, local://board, gdrive://FILE_ID, api://provider/resource).")
	fmt.Fprintln(errw, "Do not paste tokens, API keys, or cookies — credentials belong in the Runtime credential store.")
	fmt.Fprint(errw, "backend> ")
	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", fail(eBuilderHumanOnly, "no backend handle provided on stdin")
	}
	return strings.TrimSpace(line), nil
}
