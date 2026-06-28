---
name: wa3-spec
description: Review, validate, author, build, or evolve WA3 skill-contract files (.tdy) and the WA3 specification. Use when a .tdy file is involved, when checking canonical form/signature/trust, when using the guided builder from answers JSON, or when reasoning about WA3 safety rules such as core vs renderer trust, authoring LLM boundaries, extension namespace ㄝ, RL/KR revocation, canonical signing, share-link providers, Runtime/Tile sandboxing, and operate semantics.
---

# WA3 Spec Bundle

This is the installable root package for the WA3 portable agent-skill contract.
It includes the public overview, guided builder profile, adapter notes, and Go
conformance/demo tool.

## Non-negotiable Invariants

1. `wa3-core` is trusted; renderer output and authoring LLM output are untrusted.
2. LLM output is only a suggestion. Schema ownership, provenance promotion,
   secret/risk gates, canonicalization, lint, trust classification, and signing
   decisions are enforced by deterministic code.
3. `.tdy` integrity comes from canonical Ed25519 signatures, not transport.
4. Unknown `ㄝ` extension fields are preserved for canonical signing but are not
   trusted for security decisions.
5. Mutating operate must re-check the verified contract and require confirmation
   unless a code-owned low-risk action explicitly allows a human-confirmed skip.

## Read Order

1. `skill.json` for portable bundle metadata.
2. `AGENTS.md` for the cross-agent quickstart.
3. `README.md` for the public overview and safety model.
4. `docs/PROMOTE.md` before moving a draft/demo contract toward production signing.
5. End-to-end flow docs, each owning one stage of the publish/consume journey:
   `docs/EXTRACT_CONTRACT.md` (webpage → builder answers, output untrusted),
   `docs/PUBLISH_CHECKLIST.md` (sign → byte-stable link → share),
   `docs/IMPORT_TOFU.md` (download → verify → TOFU pin), and
   `docs/RENDER_PIPELINE.md` (compile → design-template binding → render).
6. `builder/answers.schema.json`, `builder/templates/`, and
   `builder/templates/catalog.json` before building from guided answers or
   recommending a functional template.
7. `design_templates/` and `design_templates/catalog.json` when you need a
   visual reference for rendering a verified UI plan. Treat `*.dsdy` files as
   display-only design guidance, never as executable or security-bearing
   contracts.
8. `conformance/README.md` before running parser, canonicalizer, builder, trust,
   or bundle-check commands.

## Authoring a Design Template

Use this workflow only for `design_templates/*.dsdy`. A design template is a
separate display-only profile inside the WA3 file family; it is **not** an
operable WA3 contract and does not use the operable canonical/signature trust
profile.

1. Confirm the source is durable and redistributable enough to reference. Record
   source and extraction date together in `ㄝㄌㄞ` using bracket labels, for
   example `ㄝㄌㄞ：【source】frontend/src/style.css【extracted】2026-06-28`.
2. Extract only layout regions/grid/viewport, visual tokens, component
   roles/shapes/states, WA3 block mapping, and interaction guidance. For
   overlays, drawers, modals, bottom navigation, fixed headers, scanlines, glow
   layers, particles, and device frames, also record display-only reachability
   constraints: which layers are decorative, which controls must stay on top,
   what safe-area padding is needed, and whether submit/confirm controls must
   remain reachable by pointer, keyboard, and assistive technology.
3. Exclude real copy, real action ids, backend targets, providers, permissions,
   policies, secrets, remote assets, executable HTML/JS/CSS, third-party fonts,
   and logos. Abstract placeholders are allowed when they cannot operate.
4. Use `design_templates/_TEMPLATE.dsdy` and keep the standard sections:
   Purpose, Safety Boundary, Layout Profile, Visual Tokens, Component Roles,
   WA3 Block Mapping, Interaction Guidance, and Source Notes.
5. Keep `ㄈㄢ：reference-only` and an empty `ㄓㄥ：`. A template with operable
   namespaces such as `ㄋㄥ`, providers, permissions, concrete targets, or real
   action ids must fail closed.

## Selecting a Design Template

Before building, continuing, or restoring any `.tdy`, ask the user whether to
apply a `*.dsdy` design template. If they have a personal default design,
offer that first. Otherwise, or if the requested/default template is missing,
use `design_templates/catalog.json` to list the closest 3 candidates by tag fit.
The catalog is only a display recommendation index; it must not carry real
action ids, providers, backend targets, permissions, policies, secrets, or trust
state. Verified WA3 actions may be mapped to abstract component roles such as
`primary_action_button` at render time, but the design template never replaces
the verified action definition.

## Selecting a Functional Template

Use `builder/templates/catalog.json` to recommend an operable builder template
before drafting answers. For product introduction pages, landing pages, solution
showcases, brochure sites, and public product pages, start from
`product_showcase`. For app-style product experiences, mobile product apps,
bottom-tab product apps, HMI apps, or field-engineer product apps, start from
`mobile_product_app`. Then show a Feature Manifest and ask which functions to
keep, remove, or rename. Functional template selection does not replace the
separate design-template question.

## Common Commands

Run from `conformance/`:

```sh
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/board.draft.tdy --mock-demo /tmp/board.mock-demo.json
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/board.test-signed.tdy --test-sign
go run ./tools/wa3 trust /tmp/board.test-signed.tdy
go run ./tools/wa3 bundle-check
```

## Adapter Notes

- Codex can install this root folder as the skill package because this file sits
  beside `builder/`, `conformance/`, and the public overview.
- `design_templates/` contains agent-readable `*.dsdy` visual profiles. They
  can guide renderer styling, but they never override verified contract data,
  action ids, permissions, confirmation requirements, or trust state. They are a
  distinct display-only design format in the WA3 file family, not operable WA3
  contracts. Use `design_templates/catalog.json` only to recommend candidates
  after asking the user.
- OpenHands can load the root package through `.plugin/plugin.json` and
  `adapters/openhands/`.
- `skills/wa3-spec/` remains the concise Codex adapter mirror for hosts that
  only want the spec-reading skill without the full builder toolchain.
- Claude Code, OpenClaw, Hermes, Haler, LangGraph, Voiceflow, Mistral, and
  Eigent adapters live under `adapters/`.
