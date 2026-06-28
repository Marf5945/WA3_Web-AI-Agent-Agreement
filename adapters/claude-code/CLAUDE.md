# WA3 for Claude Code

Use this adapter when a user asks Claude Code to inspect, author, validate, or
implement WA3 `.tdy` contracts.

## ⚠️ Claude Code cannot run Go

Claude Code's sandbox has no Go toolchain, so it **cannot execute** the
`conformance/tools/wa3` commands (`build`, `trust`, `gen-vectors`,
`bundle-check`).

- **Do static edits only** — change public docs/schema/templates/adapters/manifests
  in place, and keep package entry points consistent after any contract or
  safety-model change.
- **Never report that `bundle-check` or `build` passed** — you did not run them.
- **Hand the Go steps to the user.** Tell them to run, on their own machine with
  Go installed:

  ```sh
  cd WA3_SPEC/conformance
  go mod download
  go run ./tools/wa3 bundle-check
  ```

Without Go you can still check: JSON validity, manifest path existence, and
stale-marker absence. The "Useful Commands" section below is **for the user to
run**, not for Claude Code to execute.

## First Steps

1. Read `../../AGENTS.md`.
2. Read `../../README.md` before making any public-package claim.
3. Use `../../builder/answers.schema.json` and `../../builder/templates/` when
   building from guided answers.
4. Use `../../board.tdy` as the smallest collaborative-web example.
5. Use `../../conformance/README.md` before implementing parser or canonical
   behavior.

## Working Rules

- Treat `.tdy` files as signed contracts, not executable plugins.
- Never trust renderer-supplied backend targets, mutation flags, or policy.
- Preserve unknown `ㄝ` extension fields for canonical signing.
- When reviewing changes, lead with safety or compatibility regressions.
- When proposing new fields, decide whether they are security-critical. Use core
  namespace plus major bump for security-critical changes; use `ㄝ` extension or
  out-of-file runtime behavior for compatible hints.

## Useful Commands

Run from `WA3_SPEC/conformance`:

```sh
go run ./tools/wa3 gen-vectors
go run ./tools/wa3 canonical ../board.tdy
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/board.draft.tdy --mock-demo /tmp/board.mock-demo.json
go run ./tools/wa3 trust /tmp/board.draft.tdy
go run ./tools/wa3 bundle-check
```

The conformance kit covers canonicalization, builder gates, TEST ONLY signing,
trust enum classification, golden vectors, and bundle checks.
