---
name: wa3-spec
description: Review, validate, author, extract design templates for, or evolve WA3 skill-contract files (.tdy) and the WA3 specification. Use when a .tdy file is involved (including display-only *.dsdy design templates), when checking a contract's structure/canonical form/signature, when reasoning about WA3 safety rules (core vs renderer trust, extension namespace ㄝ, RL/KR revocation, canonical signing, share-link providers, Runtime/Tile sandboxing), or when proposing backward-compatible changes to the spec itself. Trigger on mentions of ".tdy", "WA3", "注音碼/zhuyin keys", "canonical signature", "extension prefix ㄝ", "design template", or files under this WA3_SPEC folder.
---

# WA3 Spec

## Overview

WA3 is a portable agent-skill contract: one signed `.tdy` file declares data entities, action entry points, permissions, and display hints, while each user's agent/runtime grows its own UI shell — but **safety semantics are always enforced by `wa3-core`, never by the renderer**. This skill helps you (1) author/build a draft `.tdy` contract from user requirements, (2) review/validate an existing `.tdy` contract, and (3) evolve the specification without breaking backward compatibility.

This is the Codex adapter for the portable WA3 bundle. Use the public overview
in `../../README.md`, the quickstart in `../../AGENTS.md`, and the conformance
kit in `../../conformance/` before answering anything non-trivial. The
bundle-level manifest lives at `../../skill.json` when this skill is used from
the repo checkout.

## When to use

- A `.tdy` file is given to review, lint, validate, or explain.
- Authoring a new `.tdy` (wizard-style: identity → data source → entities → actions → blocks → policy → preference → sign/publish; see §10).
- Extracting a display-only `*.dsdy` design template from a durable, allowed
  source (see §4A).
- Questions about WA3 safety rules, canonical signing (§8), the `ㄝ` extension namespace (§3.4), RL/KR revocation & rotation (§27), share-link providers (§23.4), AI injection defense (§28), or operate semantics (§29).
- Proposing changes to the spec — must follow the backward-compatibility rules (§31).

## Non-negotiable invariants (check these first)

1. **core trusted, renderer untrusted.** Security is enforced in core; renderer only reports `action_id` / `resource_id` / `input`.
2. **Namespace before reject.** A structural token is classified first (§3.4): core tokens not in the §3.2 table → fail-closed; `ㄝ`-prefixed extension tokens → unknown ones are skipped + marked untrusted, never reject the file. Extensions must never carry security-critical meaning.
3. **Integrity from signature, not transport.** `.tdy` is Ed25519-signed over canonical content (§8). Verify before use. Unknown `ㄝ` extensions are still byte-preserved into the canonical/signature.
4. **Pin by hash/signature, not URL** for share-link providers (§23.4).
5. **Authoring LLM is untrusted.** Builder/LLM output is a suggestion only (§10.1): schema ownership, provenance promotion, secret/risk gates, canonicalization, lint, and signing decisions are enforced by deterministic code.

## Workflow

### Author / build a `.tdy` (ask before you build — §10.1 must-ask gate)
1. **Pick a template** (`board` / `task_list` / `feedback_form` / `product_showcase` / `mobile_product_app`) and load its defaults. Use `../../builder/templates/catalog.json` to recommend an operable functional template; for product introduction pages, landing pages, solution showcases, brochure sites, and public product pages, start from `product_showcase`; for app-style product experiences, mobile product apps, bottom-tab product apps, HMI apps, or field-engineer product apps, start from `mobile_product_app`.
2. **Do not invent features.** Only include entities/fields/actions the template provides or the user explicitly asked for. Anything you think of but the user did not request is a `system_suggested` proposal, never silently built in (e.g. do NOT add a "mood" field to a board nobody asked for).
3. **Present a Feature Manifest and ask first.** Before producing any `.tdy`, show the user the full list of entities/fields/actions/blocks you intend to include — each with a one-line purpose, data impact (read / write), `risk_class`, and provenance (`template_default` / `system_suggested`). Introduce it in plain language: "Here are the webpage features I found/listed; you can remove features, and you can choose presentation preferences such as larger text, density, colors, or a reference design format." Offer per-item keep / remove / rename, plus "accept all defaults" and "start building".
4. **Ask design-template selection with the manifest.** Before building, continuing, or restoring any `.tdy`, ask whether to apply a `*.dsdy` design template. If the user has a personal default design, offer that first. Otherwise, or if the requested/default template is missing, use `../../design_templates/catalog.json` to list the closest 3 candidates by tag fit. The catalog is display-only and must not carry real action ids, providers, targets, permissions, policies, secrets, or trust state.
5. **Ask presentation preferences with the manifest.** Include a small "presentation / design reference" section for font size, density, visible/hidden blocks, colors, and optional reference style. These are UI/contract hints only; they must not invent undeclared functionality or weaken core safety semantics. Mapping a verified WA3 action to a template role such as `primary_submit` happens at render/UI-plan time; the design template never replaces the verified action definition.
6. **Wait for explicit confirmation.** No "start building" / "accept all defaults" → produce no file. This confirmation is human-only; you cannot self-confirm. Loop on keep/remove/rename, design-template selection, and presentation preferences until the user says build.
7. **When missing info, ask — don't guess.** If the interface needs a decision the template and the user did not cover (anonymous? categories? who can delete?), ask; do not fill it with a default and treat it as confirmed.
8. **Then build** via deterministic code (`wa3 build`): validate answers, run secret/risk/canonical/lint gates, write to `output/` only if every gate passes. (No Go toolchain — e.g. Claude Code — means you prepare answers + manifest and ask the user to run `build`.)

### Review a `.tdy`
1. Encoding/structure checks (§6A–§6D): UTF-8/NFC/LF, no BOM/null, one app, no nesting, no duplicate keys.
2. Token classification (§3.4) and value safety (§6E): allowed schemes/paths, bounded values, ISO-8601 UTC times.
3. Trust (§6F, §27): signature verifies, publisher pinned (TOFU), not revoked, rotation chain valid.
4. Report findings with stable error codes (§30.4).

### Author a design template (`*.dsdy`, §4A)
1. Use `../../design_templates/_TEMPLATE.dsdy`.
2. Treat the file as a separate display-only WA3 design profile, not an
   operable contract and not part of the operable canonical/signature trust
   profile.
3. Record source and extraction date together in `ㄝㄌㄞ`, using bracket labels:
   `ㄝㄌㄞ：【source】...【extracted】YYYY-MM-DD`.
4. Extract only layout, visual tokens, component role/shape/state, WA3 block
   mapping, abstract action role slots, and interaction guidance. For overlays,
   drawers, modals, bottom navigation, fixed headers, scanlines, glow layers,
   particles, and device frames, also capture display-only reachability
   constraints: which layers are decorative, which controls must stay on top,
   what safe-area padding is needed, and whether submit/confirm controls must
   remain reachable by pointer, keyboard, and assistive technology.
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
   only and never new WA3 features.
8. If a verified WA3 action cannot be matched to any design role slot, report
   the missing design slot to the user and propose a display-only fallback slot
   derived from the selected template's tokens and component grammar.

### Evolve the spec
1. Decide: is the change security-critical? → core namespace + bump major (§30.1). Otherwise → `ㄝ` extension or out-of-file (RL/KR, runtime behavior).
2. Preserve backward compatibility (§31): existing `.tdy` semantics unchanged; old runtimes degrade to "less safe but not wrongly rejecting".
3. Add/adjust conformance vectors (§30.3) so cross-language implementations stay byte-identical.

## Resources

### agents/
- `agents/openai.yaml` — interface metadata for the skill.

### output/
- `../../output/` — optional local destination for generated builder artifacts.
  The folder is git-ignored and not part of the public package. Prefer `/tmp`
  for smoke tests and use `output/<name>.draft.tdy` only for local drafts.
  Agents without a Go toolchain (e.g. Claude Code) must not run `build`
  themselves — edit statically and ask the user to run it.

### bundle root
- `../../AGENTS.md` — cross-agent quickstart for Haler, Codex, Claude Code, OpenClaw, and Hermes-style agents.
- `../../skill.json` — platform-neutral portable skill manifest.
- `../../builder/` — guided builder schema, templates, sample answers, mock demo notes, and reject fixtures.
- `../../design_templates/` — optional agent-readable `*.dsdy` visual profiles for rendering verified UI plans consistently. These are display-only design references, not operable WA3 contracts; they must not override action ids, targets, permissions, confirmations, trust state, or provider policy. Use `catalog.json` to recommend the closest 3 candidates by tag, and start authoring from `_TEMPLATE.dsdy`.
- `../../conformance/` — Go-native canonicalizer, builder demo, trust classifier, bundle-check, and golden vectors; see its README for status and how to regenerate vectors.

Keep this adapter concise; use README, docs, and conformance files as the public
source of package behavior.
