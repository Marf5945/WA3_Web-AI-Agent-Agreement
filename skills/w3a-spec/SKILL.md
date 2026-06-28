---
name: w3a-spec
description: Review, validate, author, extract design templates for, or evolve W3A skill-contract files (.w3a) and the W3A specification. Use when a .w3a file is involved (including display-only *_ds.w3a design templates), when checking a contract's structure/canonical form/signature, when reasoning about W3A safety rules (core vs renderer trust, extension namespace ㄝ, RL/KR revocation, canonical signing, share-link providers, Runtime/Tile sandboxing), or when proposing backward-compatible changes to the spec itself. Trigger on mentions of ".w3a", "W3A", "注音碼/zhuyin keys", "canonical signature", "extension prefix ㄝ", "design template", or files under this W3A_SPEC folder.
---

# W3A Spec

## Overview

W3A is a portable agent-skill contract: one signed `.w3a` file declares data entities, action entry points, permissions, and display hints, while each user's agent/runtime grows its own UI shell — but **safety semantics are always enforced by `w3a-core`, never by the renderer**. This skill helps you (1) author/build a draft `.w3a` contract from user requirements, (2) review/validate an existing `.w3a` contract, and (3) evolve the specification without breaking backward compatibility.

This is the Codex adapter for the portable W3A bundle. The single authoritative
spec is `references/W3A-SPEC.md` (kept in sync with the repo's
`W3A_SPEC/W3A-SPEC.md`). Read it before answering anything non-trivial. The
bundle-level quickstart and manifest live at `../../AGENTS.md` and
`../../skill.json` when this skill is used from the repo checkout.

## When to use

- A `.w3a` file is given to review, lint, validate, or explain.
- Authoring a new `.w3a` (wizard-style: identity → data source → entities → actions → blocks → policy → preference → sign/publish; see §10).
- Extracting a display-only `*_ds.w3a` design template from a durable, allowed
  source (see §4A).
- Questions about W3A safety rules, canonical signing (§8), the `ㄝ` extension namespace (§3.4), RL/KR revocation & rotation (§27), share-link providers (§23.4), AI injection defense (§28), or operate semantics (§29).
- Proposing changes to the spec — must follow the backward-compatibility rules (§31).

## Non-negotiable invariants (check these first)

1. **core trusted, renderer untrusted.** Security is enforced in core; renderer only reports `action_id` / `resource_id` / `input`.
2. **Namespace before reject.** A structural token is classified first (§3.4): core tokens not in the §3.2 table → fail-closed; `ㄝ`-prefixed extension tokens → unknown ones are skipped + marked untrusted, never reject the file. Extensions must never carry security-critical meaning.
3. **Integrity from signature, not transport.** `.w3a` is Ed25519-signed over canonical content (§8). Verify before use. Unknown `ㄝ` extensions are still byte-preserved into the canonical/signature.
4. **Pin by hash/signature, not URL** for share-link providers (§23.4).
5. **Authoring LLM is untrusted.** Builder/LLM output is a suggestion only (§10.1): schema ownership, provenance promotion, secret/risk gates, canonicalization, lint, and signing decisions are enforced by deterministic code.

## Workflow

### Author / build a `.w3a` (ask before you build — §10.1 must-ask gate)
1. **Pick a template** (`board` / `task_list` / `feedback_form` / `product_showcase` / `mobile_product_app`) and load its defaults. Use `../../builder/templates/catalog.json` to recommend an operable functional template; for product introduction pages, landing pages, solution showcases, brochure sites, and public product pages, start from `product_showcase`; for app-style product experiences, mobile product apps, bottom-tab product apps, HMI apps, or field-engineer product apps, start from `mobile_product_app`.
2. **Do not invent features.** Only include entities/fields/actions the template provides or the user explicitly asked for. Anything you think of but the user did not request is a `system_suggested` proposal, never silently built in (e.g. do NOT add a "mood" field to a board nobody asked for).
3. **Present a Feature Manifest and ask first.** Before producing any `.w3a`, show the user the full list of entities/fields/actions/blocks you intend to include — each with a one-line purpose, data impact (read / write), `risk_class`, and provenance (`template_default` / `system_suggested`). Introduce it in plain language: "Here are the webpage features I found/listed; you can remove features, and you can choose presentation preferences such as larger text, density, colors, or a reference design format." Offer per-item keep / remove / rename, plus "accept all defaults" and "start building".
4. **Ask design-template selection with the manifest.** Before building, continuing, or restoring any `.w3a`, ask whether to apply a `*_ds.w3a` design template. If the user has a personal default design, offer that first. Otherwise, or if the requested/default template is missing, use `../../design_templates/catalog.json` to list the closest 3 candidates by tag fit. The catalog is display-only and must not carry real action ids, providers, targets, permissions, policies, secrets, or trust state.
5. **Ask presentation preferences with the manifest.** Include a small "presentation / design reference" section for font size, density, visible/hidden blocks, colors, and optional reference style. These are UI/contract hints only; they must not invent undeclared functionality or weaken core safety semantics. Mapping a verified W3A action to a template role such as `primary_submit` happens at render/UI-plan time; the design template never replaces the verified action definition.
6. **Wait for explicit confirmation.** No "start building" / "accept all defaults" → produce no file. This confirmation is human-only; you cannot self-confirm. Loop on keep/remove/rename, design-template selection, and presentation preferences until the user says build.
7. **When missing info, ask — don't guess.** If the interface needs a decision the template and the user did not cover (anonymous? categories? who can delete?), ask; do not fill it with a default and treat it as confirmed.
8. **Then build** via deterministic code (`w3a build`): validate answers, run secret/risk/canonical/lint gates, write to `output/` only if every gate passes. (No Go toolchain — e.g. Claude Code — means you prepare answers + manifest and ask the user to run `build`.)

### Review a `.w3a`
1. Encoding/structure checks (§6A–§6D): UTF-8/NFC/LF, no BOM/null, one app, no nesting, no duplicate keys.
2. Token classification (§3.4) and value safety (§6E): allowed schemes/paths, bounded values, ISO-8601 UTC times.
3. Trust (§6F, §27): signature verifies, publisher pinned (TOFU), not revoked, rotation chain valid.
4. Report findings with stable error codes (§30.4).

### Author a design template (`*_ds.w3a`, §4A)
1. Use `../../design_templates/_TEMPLATE_ds.w3a`.
2. Treat the file as a separate display-only W3A design profile, not an
   operable contract and not part of the operable canonical/signature trust
   profile.
3. Record source and extraction date together in `ㄝㄌㄞ`, using bracket labels:
   `ㄝㄌㄞ：【source】...【extracted】YYYY-MM-DD`.
4. Extract only layout, visual tokens, component role/shape/state, W3A block
   mapping, abstract action role slots, and interaction guidance.
5. Exclude real copy, real action ids, backend targets, providers, permissions,
   policies, secrets, remote assets, executable HTML/JS/CSS, third-party fonts,
   and logos. Abstract placeholders are allowed when they cannot operate.
6. Keep `ㄈㄢ：reference-only` and leave `ㄓㄥ：` empty; reject any template that
   carries operable namespaces or concrete targets.
7. For every button, CTA, nav tab, row action, form submit, or confirmation
   control in the source, record only an abstract role slot such as
   `primary_submit`, `secondary_action`, `compact_icon_action`,
   `navigation_or_filter`, or `destructive_confirm`. Missing roles may be derived
   from the same visual system at render time, but they are fallback styling
   only and never new W3A features.
8. If a verified W3A action cannot be matched to any design role slot, report
   the missing design slot to the user and propose a display-only fallback slot
   derived from the selected template's tokens and component grammar.

### Evolve the spec
1. Decide: is the change security-critical? → core namespace + bump major (§30.1). Otherwise → `ㄝ` extension or out-of-file (RL/KR, runtime behavior).
2. Preserve backward compatibility (§31): existing `.w3a` semantics unchanged; old runtimes degrade to "less safe but not wrongly rejecting".
3. Add/adjust conformance vectors (§30.3) so cross-language implementations stay byte-identical.

## Resources

### references/
- `references/W3A-SPEC.md` — the full authoritative specification (§0–§31 + Appendix A). Load it for any detailed work.

### agents/
- `agents/openai.yaml` — interface metadata for the skill.

### output/
- `../../output/` — **the destination for everything the builder generates.** Always write built contracts here, never next to the spec/schema/templates. Use `output/<name>.draft.w3a` for unsigned drafts, `output/<name>.test-signed.w3a` for opt-in test-signed files, and `output/<name>.mock-demo.json` for mock-provider demos. Pass it explicitly to the CLI, e.g. `--out ../output/board.draft.w3a`. A file lands here only after the deterministic gate (schema → secret-scan → risk/confirm → canonical → lint) passes; a failed gate writes nothing. Folder contents are git-ignored; only its README is tracked. Agents without a Go toolchain (e.g. Claude Code) must not run `build` themselves — edit statically and ask the user to run it; the artifact then appears in `output/`.

### bundle root
- `../../AGENTS.md` — cross-agent quickstart for Haler, Codex, Claude Code, OpenClaw, and Hermes-style agents.
- `../../skill.json` — platform-neutral portable skill manifest.
- `../../builder/` — guided builder schema, templates, sample answers, mock demo notes, and reject fixtures.
- `../../design_templates/` — optional agent-readable `*_ds.w3a` visual profiles for rendering verified UI plans consistently. These are display-only design references, not operable W3A contracts; they must not override action ids, targets, permissions, confirmations, trust state, or provider policy. Use `catalog.json` to recommend the closest 3 candidates by tag, and start authoring from `_TEMPLATE_ds.w3a`.
- `../../conformance/` — Go-native canonicalizer, builder demo, trust classifier, bundle-check, and golden vectors; see its README for status and how to regenerate vectors.

Do not duplicate the full spec inside this SKILL.md. Keep the skill concise and
load `references/W3A-SPEC.md` only when the user asks for real W3A work.
