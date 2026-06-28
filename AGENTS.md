# W3A Agent Quickstart

This bundle teaches an agent or runtime how to read, review, and use W3A
contracts. W3A is a portable agent-skill contract: a signed `.w3a` file declares
data, actions, permissions, and display hints; the runtime renders the experience
and enforces safety.

## Read Order

1. Read `skill.json` for the bundle manifest and entry points.
2. Read `W3A-SPEC.md` for normative rules.
3. For agent-specific behavior, read the matching file under `adapters/`.
4. Read `docs/PROMOTE.md` before describing a draft/demo to production signing path.
   For the full publish/consume journey read the four stage docs:
   `docs/EXTRACT_CONTRACT.md` (webpage → builder answers; LLM output is an
   untrusted suggestion), `docs/PUBLISH_CHECKLIST.md` (sign, byte-stable link,
   out-of-band key fingerprint), `docs/IMPORT_TOFU.md` (consumer download →
   verify → TOFU pin), and `docs/RENDER_PIPELINE.md` (compile → design-template
   binding → render with provenance).
5. Use `board.w3a` as the minimal collaborative web-board example.
6. Use `builder/answers.schema.json`, `builder/templates/`, and
   `builder/templates/catalog.json` for the guided builder functional template
   profile. Use `builder/examples/board.answers.json` and
   `builder/examples/product_showcase.answers.json` or
   `builder/examples/mobile_product_app.answers.json` as minimal examples.
7. Use `design_templates/` when an agent needs a visual design reference for
   rendering a verified UI plan. These `*_ds.w3a` files are display guidance
   only, not operable contracts. Use `design_templates/catalog.json` to
   recommend the closest 3 display templates by tag, but always ask the user
   before applying one.
8. Use `conformance/README.md`, `conformance/tools/w3a`, and
   `conformance/vectors/` before implementing a parser, canonicalizer, or
   validator.

## Authoring a Design Template

Use this workflow only for `design_templates/*_ds.w3a`. A design template is a
separate display-only profile inside the W3A file family; it is **not** an
operable W3A contract and does not use the operable canonical/signature trust
profile.

1. Start from `design_templates/_TEMPLATE_ds.w3a`.
2. Confirm the source is durable and allowed to reference. Record source and
   extraction date together in `ㄝㄌㄞ` with bracket labels, for example
   `ㄝㄌㄞ：【source】frontend/src/style.css【extracted】2026-06-28`.
3. Extract only layout regions/grid/viewport, visual tokens, component
   roles/shapes/states, W3A block mapping, and interaction guidance.
4. Exclude real copy, real action ids, backend targets, providers, permissions,
   policies, secrets, remote assets, executable HTML/JS/CSS, third-party fonts,
   and logos. Abstract placeholders are allowed when they cannot operate.
5. Keep `ㄈㄢ：reference-only` and an empty `ㄓㄥ：`. A design template with
   operable namespaces such as `ㄋㄥ`, providers, permissions, concrete targets,
   or real action ids must fail closed.

## Selecting a Design Template

- Before building, continuing, or restoring any `.w3a`, ask the user whether to
  apply a `*_ds.w3a` design template.
- If the user has a personal default design template, ask about that one first.
- If the requested/default template is missing, use `design_templates/catalog.json`
  to list the closest 3 candidates by tag fit and let the user choose.
- `catalog.json` is only a display recommendation index. It must not carry true
  action ids, providers, backend targets, permissions, policies, secrets, or
  trust state.
- Mapping a verified W3A action to a design component role, such as a primary
  button shape, happens at render/UI-plan time; the design template never
  replaces the verified action definition.

## ⚠️ Go toolchain notice (read before editing)

The conformance kit under `conformance/` is Go. Some agent sandboxes — including
Claude Code's — **cannot run Go**. If `go` is unavailable in your environment:

- **Make static edits only.** Edit the spec, schema, templates, adapters, and
  manifests directly; keep the `W3A-SPEC.md` ↔ `skills/w3a-spec/references/W3A-SPEC.md`
  mirror byte-identical after any spec change.
- **Do NOT claim `bundle-check` / `build` passed.** You cannot execute them, so do
  not report Go-side results (manifest path checks, vector regeneration, signing).
- **Tell the user to run the Go steps themselves**, outside the agent, on a host
  with Go installed:

  ```sh
  cd W3A_SPEC/conformance
  go mod download            # one-time
  go run ./tools/w3a bundle-check
  go build -o /tmp/w3a ./tools/w3a   # optional: build the CLI outside the repo
  ```

- **Build artifacts outside the repo tree.** Use `go build -o /tmp/w3a`, not
  `go build ./tools/w3a`. A binary left in the repo embeds the program's own
  string table and would otherwise be flagged by the bundle-check marker scan.
  `bundle-check` already skips compiled artifacts and `.gitignore` excludes them,
  but keeping the tree clean avoids surprises.

What you *can* verify without Go: JSON validity, mirror byte-equality, path
existence for manifest entries, and absence of stale markers.

## Runtime Contract

- Trust `w3a-core`, not the renderer.
- Parse and verify a `.w3a` before using any action, target, provider, or policy.
- Classify structural tokens before rejecting them: core namespace is
  fail-closed; `ㄝ` extension namespace is skipped if unknown and marked
  untrusted.
- Preserve unknown `ㄝ` extension fields for canonical signing.
- Never let AI output directly operate. AI output lands in display-only or draft
  surfaces until the user confirms a concrete action and target.

## Standard Workflow

Publisher side (author and distribute):

0a. `extract` (optional, see `docs/EXTRACT_CONTRACT.md`): turn a webpage,
    screenshot, or description into a builder-answers proposal. The LLM output is
    an untrusted suggestion; backend handle, scope, permissions, and
    mutating/confirm flags stay unconfirmed until the user approves them.
0b. `build`: if starting from guided answers, treat LLM output as untrusted
    suggestions, validate `answers.json`, run secret/risk/canonical gates, and
    only then emit a draft or test-signed `.w3a`.
0c. `sign` + `publish` (see `docs/PUBLISH_CHECKLIST.md`): production-sign with a
    non-test key, publish to a byte-stable source, and share the public-key
    fingerprint out-of-band.

Consumer side (receive and render):

1. `import` (see `docs/IMPORT_TOFU.md`): download the bytes, run `trust`, compare
   the fingerprint out-of-band, then pin (TOFU). Optionally pass
   `trust --rl <file.w3a-rl>` to re-check the §27 revocation list.
2. `lint`: check encoding, structure, namespace classification, value safety,
   version compatibility, and limits.
3. `verify`: rebuild canonical bytes and verify Ed25519 signature and publisher
   trust.
4. `compile`: produce a UI/Agent Interface Plan without backend secrets.
5. `render` (see `docs/RENDER_PIPELINE.md`): draw a local UI from the plan,
   binding blocks/actions to an optional `*_ds.w3a` at render time; renderer
   reports only user intent.
6. `operate`: runtime re-reads the verified contract, re-checks revocation,
   confirms mutating actions, generates idempotency keys, and calls the provider
   adapter.

## Current Bundle Status

- Normative spec: `W3A-SPEC.md`.
- Example contract: `board.w3a`.
- Guided builder profile: `builder/`.
- Functional templates: `builder/templates/`.
- Agent-readable design templates: `design_templates/`.
- Minimal conformance kit: `conformance/`.
- Codex installable skill: `skills/w3a-spec/`.
- Platform adapters: `adapters/`.
