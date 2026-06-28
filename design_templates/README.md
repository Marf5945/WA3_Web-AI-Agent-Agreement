# W3A Design Templates

This folder contains agent-readable design templates, named `*_ds.w3a`.

These files are **not operable W3A contracts**. They are a separate
display-only design format inside the W3A file family and do not use the
operable W3A canonical/signature trust profile. They are design-system
reference notes for agents and renderers that need a consistent visual shell
when turning a verified W3A UI plan into an interface.

## Rules

- Treat every `*_ds.w3a` file as display guidance only.
- Never execute code, load assets, call providers, or infer permissions
  from a design template.
- Runtime safety still comes from the verified `.w3a` contract and `w3a-core`.
- Every template MUST use `ㄈㄢ：reference-only` and an empty `ㄓㄥ：`.
- `ㄝㄌㄞ` MUST include both durable source and extraction date in one field,
  using bracket labels such as
  `ㄝㄌㄞ：【source】frontend/src/style.css【extracted】2026-06-28`.
- Abstract placeholders are allowed; real action ids, concrete targets,
  provider names, permissions, policies, and secrets are not.
- Button and CTA guidance must use abstract action role slots. A role slot can
  say how a verified W3A submit/delete/search/navigation action should look,
  but it cannot define or replace that action.
- A renderer may ignore a design template when it cannot support the suggested
  layout or component style.
- Do not copy source implementation files, fonts, visual identity assets,
  screenshots, or other source assets into this folder.

## Authoring Workflow

1. Start from `_TEMPLATE_ds.w3a`.
2. Verify the source is durable enough to cite and that the license permits
   redistribution of the extracted design description. Prefer original local
   source files, design tokens, or documented selectors over generated output,
   temporary previews, or source asset bundles.
3. Extract only the framework of the experience:
   layout regions/grid/viewport, `font_stack`, density, radius, colors, effects,
   component shape/states/role, W3A block mapping, and interaction guidance.
4. Exclude content and operable semantics:
   real copy, true button labels, true action ids, backend targets, provider
   names, policy, permissions, secrets, source asset references, executable
   implementation files, fonts, and visual identity assets.
5. Describe components by role and shape only. For example, `adapter_button`
   may describe fixed icon cells, status-dot states, and truncation behavior,
   but must not say what the button does.
6. Add `action_role_slots` inside `Component Roles` when the source has buttons,
   CTAs, nav tabs, row actions, confirmations, or form submit controls. Each
   slot must match only verified W3A actions by abstract properties such as
   primary submit, secondary cancel, destructive confirm, navigation/filter, or
   row inline action.
7. Fill the standard sections in order: Purpose, Safety Boundary,
   Layout Profile, Visual Tokens, Component Roles, W3A Block Mapping,
   Interaction Guidance, and Source Notes.
8. Leave `ㄓㄥ：` empty. A design template is reference material, not a trusted
   security carrier.

## Static Design Lint

The normative rules, hardcoded constants, and error codes live in
`W3A-SPEC.md` §4A.4 (`硬編碼常量`) and §4A.5 (`Design lint 與錯誤碼`). A model or
linter MUST compare against those literals verbatim and, on any failure, emit the
matching `DS_Exxx` code plus a one-line reason and the offending line. Do not
guess or auto-repair the file.

Reject a `*_ds.w3a` fail-closed when any of these are true:

- `ㄊㄡ` is not `W3A design-template v0.1` (`DS_E001`).
- `ㄈㄢ` is not `reference-only` (`DS_E002`).
- `ㄝㄇㄜ` is not `design_template` (`DS_E003`).
- `ㄝㄌㄞ` is missing a `【source】...【extracted】YYYY-MM-DD` value (`DS_E004`).
- Any standard section is missing or out of order (`DS_E005`).
- `ㄓㄤ` is missing, or `ㄎㄟ`/`ㄓㄥ：` is non-empty (`DS_E006`).
- `ㄏㄡ` does not start with the `design://local/` scheme, or names a resolvable
  backend address (`DS_E007`).
- A line begins with an operable marker (`ㄋㄥ`/`ㄎㄜ`/`ㄕㄜ`/`ㄕ：`/`ㄓㄠ：`/`ㄉㄨ：`),
  or the file carries real action ids, providers, or secrets (`DS_E008`). Text
  that names a forbidden concept in a safety warning is allowed. The block-type
  keys `ㄇㄢ`/`ㄗㄞ`/`ㄆㄤ`/`ㄔㄠ`/`ㄙㄞ`/`ㄉㄟ` used as `W3A Block Mapping` JSON keys
  are not operable markers and MUST NOT trigger this.
- `ㄏㄠ` does not match `design.<family>.<name>` (`DS_E009`).
- The `ㄝ` metadata is reordered away from `ㄝㄇㄜ→ㄝㄕㄡ→ㄝㄌㄞ→ㄝㄉㄧ` (`DS_E010`).

## Suggested Use

1. Verify and compile the real W3A contract.
2. Ask the user whether to apply a `*_ds.w3a` template. If the user has a
   personal default template, ask about that one first.
3. If no default is chosen, use `catalog.json` to list the closest 3 candidate
   templates by tag fit, then let the user choose or skip.
4. Load the chosen `*_ds.w3a` design template as visual reference.
5. Map the UI plan's blocks and action ids onto the template's component roles.
6. Preserve provenance: every rendered action still points back to a declared
   W3A action id.
7. Confirm mutating actions through `w3a-core`, regardless of what the template
   suggests visually.
8. If a verified W3A action has no matching design role, derive a same-style
   fallback control from the chosen template's tokens and component grammar,
   notify the user that a design slot was missing, then mark that binding as
   fallback styling. Do not invent a new feature or hide the fact that the
   function comes from the verified `.w3a`.

## Catalog

`catalog.json` is a display-only recommendation index for this folder. It
exists so an agent can find a familiar interface style without guessing from
free text every time. The catalog records each template's file, handle, tags,
best-fit app types, layout tags, density, block-type affinity, and abstract
component roles.

The catalog is **not** a contract, signature carrier, permission source, or
provider registry. It MUST NOT contain real action ids, backend targets,
providers, permissions, policies, secrets, remote assets, or trust state. It may
say that a verified W3A write action can be rendered in an abstract
`primary_submit` role; it must not name or replace the real W3A action.

Selection rules:

- Always ask before applying any design template.
- Ask about the user's personal default design first when one is configured.
- When the requested/default template is unavailable, list the closest 3
  catalog matches and let the user choose.
- If the user skips design selection, the runtime may use its built-in default
  shell, but it should record that this was a user choice.
- Frequent or first-level controls are still decided from the verified `.w3a`
  function list and user discussion; the design template only affects layout,
  density, ordering, visible surfaces, and component roles.

## Current Templates

- `haler_console_ds.w3a` - extracted from the local HaLer AI Console v3.2
  interface.
- `mobile_holographic_hmi_ds.w3a` - extracted from a user-provided pasted mobile
  HMI app preview as a compact holographic native-app profile.
- `modern_industrial_glass_ds.w3a` - extracted from `kronos-preview.html` as a
  modern industrial website profile with frosted glass sections.
- `terminal_tactical_ds.w3a` - extracted from a user-provided terminal/HUD
  preview.
