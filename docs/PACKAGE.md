# WA3 Skill Package

The parent directory of this `docs/` folder (`WA3_SPEC/`) is the portable WA3
skill package. Paths below are relative to that package root.

## Install Shape

Install or copy the whole `WA3_SPEC/` directory when the host supports folder
skills. The root `SKILL.md` is the entry point, and the package carries its own
builder fixtures, templates, spec, promote path, adapters, and Go
conformance/demo tool.

For a reduced Codex-only install, `skills/wa3-spec/` can be copied by itself,
but that reduced package is for spec-guided authoring/review only. It does not
carry the runnable builder/conformance toolchain.

## Smoke Test

Run from `conformance/`:

```sh
go run ./tools/wa3 bundle-check
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/board.draft.tdy --mock-demo /tmp/board.mock-demo.json
go run ./tools/wa3 trust /tmp/board.draft.tdy
```

Expected trust state for the draft is `unsigned_draft`.

## Current Adapter Status

- Codex: root folder is installable as a full skill package; nested
  `skills/wa3-spec/` is the reduced adapter.
- OpenHands: root folder includes `.plugin/plugin.json` and
  `adapters/openhands/` for native plugin-style loading.
- Haler: `adapters/haler/haler.skill.json` projects the builder and runtime
  expectations, but the app UI still needs to wire the guided screens.
- Claude Code / OpenClaw / Hermes: adapters are instruction-entry files that
  point back to `skill.json`, `AGENTS.md`, `README.md`, and `conformance/`.
- LangGraph / Voiceflow / Mistral: integration sketches are provided under
  `adapters/`; they are not the same thing as native folder-skill installs.
- Eigent: placeholder integration guidance only until an official install
  manifest is confirmed.

## Conversation-Driven Install

Use `INSTALL_AGENTS.md` when asking another agent to install or test this
package. It routes the host, chooses the native adapter when available, and
defines the smoke test and capability report.
