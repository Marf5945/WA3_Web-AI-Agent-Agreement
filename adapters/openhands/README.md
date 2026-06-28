# WA3 for OpenHands

OpenHands supports plugin packages with `.plugin/plugin.json` metadata. This
package includes that metadata at the root, plus the full WA3 builder and
conformance kit.

## Install

Use one of the OpenHands plugin mechanisms:

```sh
openhands --plugin /path/to/WA3_SPEC
```

or configure the package path in the OpenHands plugin sources.

## Load Order

1. `SKILL.md`
2. `skill.json`
3. `AGENTS.md`
4. `README.md`
5. `builder/answers.schema.json`
6. `builder/templates/`
7. `conformance/README.md`

## OpenHands Operating Rules

- Treat authoring LLM output as untrusted suggestions.
- Let deterministic code enforce schema ownership, secret scan, risk/confirm
  policy, canonicalization, lint, and trust state.
- Never copy user credentials, repository secrets, local paths, or chat logs into
  generated `.tdy` or this package.
- If a mutating action is requested, re-check the verified contract and require
  confirmation unless the code-owned template marks it as `low_mutate` and the
  human explicitly disabled confirmation.

## Smoke Test

Run from `conformance/`:

```sh
go run ./tools/wa3 bundle-check
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/wa3-board.draft.tdy --mock-demo /tmp/wa3-board.mock-demo.json
go run ./tools/wa3 trust /tmp/wa3-board.draft.tdy
```

## Prompt To Install

```text
Install the WA3_SPEC package as an OpenHands plugin. Load SKILL.md and
AGENTS.md, run the smoke test from conformance/, then report whether build,
trust, and bundle-check pass. Do not copy secrets or local user files into the
package.
```
