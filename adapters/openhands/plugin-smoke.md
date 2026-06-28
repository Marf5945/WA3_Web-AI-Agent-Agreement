# OpenHands Plugin Smoke Check

Use this as a checklist after loading the W3A plugin.

1. Confirm `.plugin/plugin.json` exists.
2. Confirm root `SKILL.md` exists.
3. Confirm `builder/answers.schema.json` exists.
4. Confirm `conformance/tools/w3a/main.go` exists.
5. Run:

```sh
cd conformance
go run ./tools/w3a bundle-check
go run ./tools/w3a build --answers ../builder/examples/board.answers.json --out /tmp/w3a-board.draft.w3a --mock-demo /tmp/w3a-board.mock-demo.json
go run ./tools/w3a trust /tmp/w3a-board.draft.w3a
```

Pass criteria:

- `bundle check ok`
- draft build exits successfully
- trust output has `state` equal to `unsigned_draft`
