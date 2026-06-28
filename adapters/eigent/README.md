# W3A for Eigent

Eigent (the open-source Cowork desktop) supports an **agent-skills format that is
Claude Code–compatible**: a Skill is a package with a root `SKILL.md` carrying
YAML `name` + `description`, and plugins use `plugin.json` (+ optional `.mcp.json`,
slash commands, agents, skills). This package already ships both, so it installs
on Eigent without a separate manifest.

## Install

Eigent installs skills from the official CLI, which can auto-detect a
Claude-Code-style agent:

```sh
npx @eigent-ai/agent-skills install -a claude-code
```

To install this package as a **custom** skill, upload it via
`Homepage > Agents > Skills`:

- **Do not upload `SKILL.md` alone.** This skill's `SKILL.md` uses relative paths
  back to the bundle (`../../W3A-SPEC.md`, `../../conformance/`, etc.), so a
  standalone upload breaks those references.
- **Upload the whole bundle as a `.zip`** with a `SKILL.md` at the archive root.
  Either repackage `skills/w3a-spec/SKILL.md` as the root with the referenced
  files alongside it, or zip the `W3A_SPEC` root (its `.claude-plugin/plugin.json`
  and `skills/w3a-spec/` are auto-discovered the same way Claude Code reads them).

## Load Order

1. `SKILL.md`
2. `AGENTS.md`
3. `W3A-SPEC.md`
4. `builder/answers.schema.json`
5. `builder/templates/`
6. `conformance/README.md`

## Operating Rules

- Authoring LLM output is an untrusted suggestion, never a gate.
- Let deterministic code enforce schema ownership, secret scan, risk/confirm
  policy, canonicalization, lint, and trust state.
- Do not let an LLM decide `risk_class`, trust state, secret-scan results, or
  canonical validity.
- Never copy user credentials, repository secrets, local paths, or chat logs into
  a generated `.w3a` or this package.

## Verify

Run from `conformance/` if a Go toolchain is available:

```sh
go run ./tools/w3a bundle-check
go run ./tools/w3a build --answers ../builder/examples/board.answers.json --out /tmp/w3a-board.draft.w3a --mock-demo /tmp/w3a-board.mock-demo.json
go run ./tools/w3a trust /tmp/w3a-board.draft.w3a
```

## References

- Eigent Agent Skills docs: https://docs.eigent.ai/core/agent-skills
- Eigent skills CLI: https://github.com/eigent-ai/agent-skills
- Eigent plugins: https://github.com/eigent-ai/agent-plugins
