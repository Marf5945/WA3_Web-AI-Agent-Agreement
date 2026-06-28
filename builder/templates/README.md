# W3A Functional Templates

This folder is the guided-builder functional template area. Templates here
declare user-visible data shapes, actions, blocks, risk classes, and default
preferences for safe `.w3a` authoring.

Functional templates are operable builder inputs. They are different from
`../../design_templates/*_ds.w3a`, which are display-only shape and visual
references and must not define actions, providers, permissions, or trust.
The optional `recommended_design_templates` field may list display-only
`*_ds.w3a` candidates by handle/file/reason, but the agent must still ask before
applying one.

## Current Templates

- `board.json` - collaborative message board.
- `task_list.json` - shared task list.
- `feedback_form.json` - feedback intake form.
- `product_showcase.json` - product introduction / landing page with product
  overview, solution sections, metrics, timeline, resources, and inquiry form.
- `mobile_product_app.json` - app-style product experience with product
  carousel, case metrics, milestone timeline, technical resource vault, and
  contact/inquiry actions.

## Agent Selection Notes

When a user asks for a product introduction page, product landing page,
solution showcase, brochure site, public product page, or launch page, suggest
`product_showcase` as the closest functional template. Then ask which
features to keep, remove, or rename, and separately ask whether to apply a
display-only `*_ds.w3a` design template.

When a user asks for an app-style product experience, mobile product app,
bottom-tab product app, HMI app, field-engineer product app, or product
brochure app, suggest `mobile_product_app` first. Keep the app's visual shell
and animation language in design-template selection; the functional template
only declares data, actions, blocks, and risk/confirmation defaults.
