# Codex Adapter

For the full runnable builder package, install the `../../` folder itself. The
root `SKILL.md` sits beside `builder/`, `conformance/`, `W3A-SPEC.md`, and
`skill.json`, so build/trust/bundle-check commands can run after install.

The reduced Codex adapter lives at `../../skills/w3a-spec/`. Use it only when a
host wants a small spec-guidance skill without the builder/conformance toolchain.

For local development, keep `../../skills/w3a-spec/references/W3A-SPEC.md`
identical to `../../W3A-SPEC.md`. The reduced skill should stay concise and
point back to the full specification instead of duplicating it.
