# Compile, Design Template Binding, and Render Flow

This document names the safe path for custom UI rendering. A design template is
display guidance only; binding happens at render planning time, not through an
identity field inside the `*.dsdy`.

## Flow

1. Verify the operable `.tdy`.
2. Compile an Agent Interface Plan. The plan contains blocks and action ids, but
   no secret, token, backend target, HTTP method, or provider credential.
3. Ask the user whether to apply a `*.dsdy` design template. The three choices
   are: no template (neutral fallback renderer), the default template, or one of
   three recommended templates.
4. Resolve the design choice:
   - No template: use the neutral fallback renderer (see "Neutral Default and
     No-Template" below). Do not load any `*.dsdy`.
   - Default template: load the `selection_policy.default_handle` entry in
     `design_templates/catalog.json`.
   - Pick from recommendations: use the catalog to recommend the closest
     `fallback_candidate_count` (3) candidates by tags, block affinity, density,
     and component roles, and let the user choose one.
5. Bind block types to component roles, for example list to data list, board to
   collaboration surface, search to compact filter input.
6. Bind verified action ids to abstract action role slots, for example
   `submit_message` to `primary_submit`.
7. Render the UI. The renderer reports only user intent:
   `action_id` plus plain input values.
8. On every operation, core re-reads the verified contract, re-checks current
   `§27` revocation/key-rotation state (for example via
   `wa3 trust --rl <file.tdy-rl>` or the host's equivalent), and enforces target,
   permission, mutating, confirmation, idempotency, and provider policy. A
   revoked key or version stops the operation before any provider call.

> **Revocation freshness is a structural pre-check, not full trust.** Honoring a
> revocation list still requires verifying the list's own publisher signature per
> `§27`; the v1 `--rl` flag only tests list membership. Full RL/KR signature
> verification and the "a pinned app shipped a new signed version, re-review?"
> flow are deferred to v1.x (see `docs/STATUS.md`).

## Render-Time Assertions

Renderers and host agents should check:

- Every interactive control points back to exactly one declared action id.
- No rendered control invents an action absent from the verified plan.
- Mutating actions still route through core confirmation.
- Hidden or downgraded actions are recorded with a reason.
- The design template never supplies backend targets, providers, permissions,
  policies, trust state, or real action ids.
- If no design role matches an action, the renderer may choose fallback styling
  but must keep the verified action id and core safety semantics.
- Decorative layers, pseudo-elements, scanlines, particles, scrims, device
  frames, and visual progress rings must never intercept pointer, keyboard, or
  assistive-technology activation for a verified control.
- Confirmation controls must remain reachable by pointer and keyboard in the
  actual rendered viewport. Fixed navigation, safe-area insets, device frames,
  and modal effects must not cover or push the confirm/submit control outside
  the usable area.
- Preview persistence, such as in-memory state, local storage, fixture files, or
  mock providers, is a renderer/runtime choice. It must be labeled as preview or
  mock behavior and must not be inferred from a `*.dsdy` design template.

## Neutral Default and No-Template

The "no template" choice must still produce a usable, safe UI without loading any
`*.dsdy`. The neutral fallback renderer maps verified blocks to plain,
unstyled component roles:

- list (`ㄗㄞ`) → a plain vertical list of rows.
- board (`ㄆㄤ`) → a message/item stream with a composer when a submit action
  exists.
- detail (`ㄉㄟ`) → a labeled key/value detail panel.
- input/form (`ㄔㄠ`) → a plain form bound to the declared action inputs.
- search (`ㄙㄞ`) → a single filter input above the related list.
- text (`ㄇㄢ`) → safe plain text. This is the mandatory fallback every renderer
  must support.

The neutral renderer obeys the same render-time assertions below: every control
maps to one declared action id and mutating actions still confirm through core.
It differs from a design template only in that it provides no visual tokens,
layout profile, or component styling.

## Two Catalogs, Two Jobs

Do not confuse the two catalog files:

- `builder/templates/catalog.json` is the **functional** template index for the
  guided builder. It may name operable `template_id`s and drives what gets built.
- `design_templates/catalog.json` is the **display-only** recommendation index
  for `*.dsdy` visual shells. It never names action ids, providers, backends,
  permissions, or trust state.

Authoring picks a functional template; rendering picks a design template. They
are selected at different stages and must never be merged.

## Why There Is No App Binding Field in `.dsdy`

The design template is intentionally reusable. It does not say "I belong to this
app" because that would make display guidance look like a trust statement. The
real binding is:

- operable `.tdy` verified by core,
- Agent Interface Plan action ids and blocks,
- catalog/template display roles,
- runtime binding map created for this render session.

This is the safety guarantee behind "custom UI, unchanged backend semantics".
