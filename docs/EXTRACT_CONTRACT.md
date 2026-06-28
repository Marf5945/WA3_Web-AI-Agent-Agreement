# Webpage to WA3 Contract Extraction

This document defines the v1 authoring path for turning a real webpage or web
app into a WA3 builder `answers.json` draft. The output is never trusted just
because an agent extracted it.

## User Flow

1. User gives a page URL, screenshot, exported HTML, or manual description.
2. Agent extracts a feature inventory: visible data types, user actions, inputs,
   filters, and likely display blocks.
3. Agent writes only `llm_suggestions` and a draft `custom_template` proposal.
   Backend handle, scope, permissions, mutating flags, confirmation behavior,
   and publish target remain unconfirmed.
4. Builder shows a Feature Manifest with each proposed entity, action, data
   impact, risk class, and provenance.
5. User chooses keep, remove, rename, or asks the agent to refine. The builder
   asks only for missing decisions that block a safe draft.
6. User explicitly chooses "start building". Only then may the builder run the
   deterministic gates and emit `.tdy`.

## What to Extract

- Entity candidates: repeated records, form fields, item details, timestamps,
  numeric counters, tags, and status values.
- Action candidates: read, submit, react/button, select/filter, search, and
  delete-like operations.
- Input/output shape: field names and WA3 primitive types only.
- Display blocks: list, detail, board, input, search, or text fallback.
- Human-readable purpose and data impact for the Feature Manifest.
- Design needs as display hints, not behavior: density, text size, preferred
  layout, or possible `*.dsdy` tag matches.
- If extracting a `*.dsdy` visual reference from a webpage or app shell, capture
  overlay and modal reachability as display guidance: decorative layers must not
  intercept controls, submit/confirm controls must stay above effects, and fixed
  headers/bottom navigation/device frames require safe-area space. Do not turn
  these observations into storage, provider, backend, permission, or trust
  semantics.

## What Not to Extract

- Cookies, access tokens, OAuth grants, API keys, session values, signed URLs,
  hidden form secrets, local filesystem paths, or any credential-shaped string.
- Backend write targets without human confirmation.
- Provider permissions or scopes inferred from page text.
- Mutating/confirmation settings inferred only from button labels.
- Real third-party scripts, inline HTML/CSS/JS, remote assets, fonts, logos, or
  tracking code.
- Rendered document edit pages such as `docs.google.com/.../edit` as stable
  byte sources. Use a stable file id or published byte endpoint instead.

Any secret-shaped value must fail with `E-VALUE-SECRET`. The repair hint is:
replace the value with a stable handle such as `gdrive://FILE_ID` or
`api://provider/resource`, then configure credentials in the Runtime credential
store.

## Mapping to `answers.json`

Use a bundled template when the page clearly matches one. For product
introduction pages, landing pages, solution showcases, brochure sites, or public
product pages, recommend `template_id: "product_showcase"` first and show the
Feature Manifest for its product overview, solution sections, metrics, timeline,
resources, and inquiry form. For app-style product experiences, mobile product
apps, bottom-tab product apps, HMI apps, or field-engineer product apps,
recommend `template_id: "mobile_product_app"` first and show the Feature
Manifest for its product carousel, case metrics, milestone timeline, technical
resource vault, and contact/inquiry actions. Use `template_id: "custom_generic"`
only when the webpage or app does not fit a bundled template.

- `app.id`, `app.backend`, `app.scope`, and `publisher.id` are `human_only`.
- `custom_template.entities`, `custom_template.actions`, and
  `custom_template.blocks` are a proposal until the user confirms them.
- `risk_class` is `code_owned`; an agent may explain risk but may not lower it.
- `human_decisions` records the user's keep/remove/rename choices.
- `llm_suggestions` records wording, grouping, and uncertain candidates as
  `system_suggested`.
- `opaque_confirmations` may be used only for documented false positives, never
  for real secrets.

## Minimal Questions

Ask these only when the page itself does not answer them safely:

1. Where should the data live: local, private, or shared?
2. Which stable backend handle should the Runtime use?
3. Which proposed actions should be writable, and should they require
   confirmation?
4. Should the app use a design template, a simple default, or a user-provided
   visual reference?

For question 4, the design choice is resolved later, at render time, not during
extraction:

- Design template or simple default: handled by `docs/RENDER_PIPELINE.md`
  (default template, three recommendations, or the neutral fallback renderer).
- User-provided visual reference: the user's own page/style becomes a new
  `*.dsdy` via the design-template extraction rules in `design_templates/README.md`,
  then binds through `docs/RENDER_PIPELINE.md`. Extraction never pulls operable
  behavior from that reference.

The user should not need to understand WA3 tokens. The UI should show plain
labels and keep the WA3 structure behind the review screen.
