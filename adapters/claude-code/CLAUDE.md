# W3A for Claude Code

Use this adapter when a user asks Claude Code to inspect, author, validate, or
implement W3A `.w3a` contracts.

## ⚠️ Claude Code cannot run Go

Claude Code's sandbox has no Go toolchain, so it **cannot execute** the
`conformance/tools/w3a` commands (`build`, `trust`, `gen-vectors`,
`bundle-check`).

- **Do static edits only** — change spec/schema/templates/adapters/manifests in
  place, and keep `W3A-SPEC.md` byte-identical to
  `skills/w3a-spec/references/W3A-SPEC.md` after any spec change.
- **Never report that `bundle-check` or `build` passed** — you did not run them.
- **Hand the Go steps to the user.** Tell them to run, on their own machine with
  Go installed:

  ```sh
  cd W3A_SPEC/conformance
  go mod download
  go run ./tools/w3a bundle-check
  ```

Without Go you can still check: JSON validity, the spec mirror byte-equality,
manifest path existence, and stale-marker absence. The "Useful Commands" section
below is **for the user to run**, not for Claude Code to execute.

## First Steps

1. Read `../../AGENTS.md`.
2. Read `../../W3A-SPEC.md` before making any normative claim.
3. Use `../../builder/answers.schema.json` and `../../builder/templates/` when
   building from guided answers.
4. Use `../../board.w3a` as the smallest collaborative-web example.
5. Use `../../conformance/README.md` before implementing parser or canonical
   behavior.

## Working Rules

- Treat `.w3a` files as signed contracts, not executable plugins.
- Never trust renderer-supplied backend targets, mutation flags, or policy.
- Preserve unknown `ㄝ` extension fields for canonical signing.
- When reviewing changes, lead with safety or compatibility regressions.
- When proposing new fields, decide whether they are security-critical. Use core
  namespace plus major bump for security-critical changes; use `ㄝ` extension or
  out-of-file runtime behavior for compatible hints.

## Useful Commands

Run from `W3A_SPEC/conformance`:

```sh
go run ./tools/w3a gen-vectors
go run ./tools/w3a canonical ../board.w3a
go run ./tools/w3a build --answers ../builder/examples/board.answers.json --out /tmp/board.draft.w3a --mock-demo /tmp/board.mock-demo.json
go run ./tools/w3a trust /tmp/board.draft.w3a
go run ./tools/w3a bundle-check
```

The conformance kit covers canonicalization, builder gates, TEST ONLY signing,
trust enum classification, golden vectors, and bundle checks.
