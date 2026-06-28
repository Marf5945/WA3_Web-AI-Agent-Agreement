# Install W3A Skill Into Agents

Use this playbook when a user says something like:

> Install this W3A skill package into my agent and check whether it can run.

The package root is the parent directory of this `docs/` folder — it contains
`SKILL.md`, `skill.json`, `builder/`, and `conformance/`. All bundle-relative
paths below (e.g. `SKILL.md`, `adapters/`, `conformance/`) are relative to that
package root.

## Conversation Contract

The installing agent should:

1. Detect the host or ask which host is being used.
2. Prefer a native plugin/skill mechanism when the host has one.
3. Otherwise load the matching adapter instructions from `adapters/`.
4. Run the local smoke test if shell access is available.
5. Never copy user credentials, local paths, email, chat logs, or workspace
   private files into this package.
6. Report four capabilities:
   - can read and review `.w3a`
   - can build a draft `.w3a`
   - can classify trust state
   - can use a native install format

## Host Routing

| Host | Install mode | Adapter |
|---|---|---|
| Codex | Folder skill package | `SKILL.md`, `adapters/codex/README.md` |
| Claude Code | Claude-compatible plugin metadata | `.claude-plugin/plugin.json`, `adapters/claude-code/CLAUDE.md` |
| OpenHands | Native plugin metadata | `.plugin/plugin.json`, `adapters/openhands/README.md` |
| LangGraph | Assistant/graph integration sample | `adapters/langgraph/` |
| Voiceflow | Function/tool integration sample | `adapters/voiceflow/` |
| Mistral Agents / Le Chat | Agent/tool/MCP integration sample | `adapters/mistral/` |
| Eigent | Pending official install format | `adapters/eigent/README.md` |

## Smoke Test

Run from `conformance/`:

```sh
go run ./tools/w3a bundle-check
go run ./tools/w3a build --answers ../builder/examples/board.answers.json --out /tmp/w3a-board.draft.w3a --mock-demo /tmp/w3a-board.mock-demo.json
go run ./tools/w3a trust /tmp/w3a-board.draft.w3a
```

Expected:

- `bundle check ok`
- a draft `.w3a` is written
- trust state is `unsigned_draft`

## If Shell Access Is Not Available

The agent should still load:

1. `SKILL.md`
2. `skill.json`
3. `AGENTS.md`
4. `W3A-SPEC.md`
5. `builder/answers.schema.json`
6. `builder/templates/`

Then it should clearly say that runtime commands were not executed.

## Publication Safety

Before publishing or redistributing this package, run:

```sh
go run ./conformance/tools/w3a bundle-check
```

Also scan for personal data and credentials. The package must not contain
private workspace paths, email files, credential files, API keys, or copied
conversation logs.
