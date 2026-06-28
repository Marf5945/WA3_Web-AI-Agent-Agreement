# W3A (Web AI Agent Agreement)

W3A is a portable agent-skill contract format. A signed `.w3a` file declares
data, actions, permissions, and display hints; runtimes render the experience
while `w3a-core` enforces parsing, trust, confirmation, and operation safety.

## 繁體中文介紹

W3A（Web AI Agent Agreement）是一種可攜式的 AI Agent 技能契約格式。
一份簽章後的 `.w3a` 單檔會宣告資料結構、可執行動作、權限規則與顯示提示；
不同 runtime 可以依自己的介面風格渲染體驗，但解析、驗章、確認與實際操作安全
一律由可信的 `w3a-core` 強制執行。

這個 repo 提供 W3A 規格、最小留言板範例、guided builder profile、display-only
design templates、Go conformance kit，以及多種 agent/runtime adapter 說明。目標是
讓使用者能透過結構化流程產生可驗證、可簽章、可攜帶的 agent 功能契約，而不是讓
AI 輸出直接操作後端或繞過使用者確認。

## What Is In This Bundle

- `W3A-SPEC.md` is the normative specification.
- `board.w3a` is the minimal collaborative web-board example.
- `builder/` contains the guided builder schema, templates, and examples.
- `design_templates/` contains display-only `*_ds.w3a` visual references.
- `conformance/` contains the Go conformance kit and golden vectors.
- `skills/w3a-spec/` and `adapters/` contain agent/runtime adapter notes.

## Verify

Run the bundle check from the conformance package:

```sh
cd conformance
go mod download
go run ./tools/w3a bundle-check
```

Build artifacts should stay outside the repository tree:

```sh
go build -o /tmp/w3a ./tools/w3a
```

## License

MIT. See `LICENSE`.
