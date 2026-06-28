# WA3

> Pronunciation note: the formal name is **WA3**, written **WA³** and read aloud as *“WA cubed.”*

**語言 / Language / 言語：[中文](#中文) ・ [English](#english) ・ [日本語](#日本語)**

WA3 把一個 app 拆成兩個可以分開攜帶的檔案：**功能**和**介面架構**。
WA3 splits an app into two separately portable files: **function** and **interface design**.
WA3 はアプリを「機能」と「画面デザイン」の二つの持ち運び可能なファイルに分離します。

---

<a name="中文"></a>
# 中文

## 這是什麼

- **功能檔 `*.tdy`**：宣告「能做什麼」——資料結構、可執行動作、權限、確認規則。已簽章、可驗證。
- **設計架構檔 `*.dsdy`**：宣告「長什麼樣」——版面、密度、元件角色、互動風格。display-only，不帶任何權限或動作。

同一份功能檔可以套到任何一份設計架構檔。換掉底層的 `.dsdy`，整個介面就統一成你熟悉的樣子，功能完全不變。

## 為什麼要這樣拆

平常用很多 app 的痛點：每個 app 介面都不一樣，換工具就要重新記按鈕在哪；預載太多用不到的功能，想刪也刪不掉；自己最常用的功能被藏到第二層；就算 app 有「匯出快捷模板」，每個 app 都要各撈一份模板很麻煩。

WA3 的解法：你只要帶**一份自己的 `dsdy`**（你熟悉的介面架構），不管載入哪一份功能檔，都用同一套結構去渲染。介面統一、可裁剪、常用功能可以拉到第一層——而功能本身仍由各自簽章的 `.tdy` 決定，互不汙染。

## 一個具體例子

假設你拿到留言板功能檔 `board.tdy`（宣告 `read / submit / react / search` 四個動作）：

1. Runtime 先**驗章** `board.tdy`，確認動作、權限、確認規則。
2. 接著問你**要套哪一個介面架構**：若你設過個人預設架構，會先問那一個；若沒設、或功能檔標示的推薦架構你手上沒有，agent 會用 `design_templates/catalog.json` 的 **tag 比對**，列出你「已擁有」最接近的 3 個架構讓你選，或不套。
3. 選定後，`submit` 綁到你那套架構的「主要送出按鈕」角色、`react` 綁到「行內小圖示動作」⋯⋯外觀統一，但動作永遠來自驗證過的 `board.tdy`。

架構是**被推薦而不是被強迫**：功能檔即使標了建議架構，只要你沒下載或不想用，agent 不會硬套，主導權在你。設計架構也永遠不能新增功能或改權限——它只影響版面、密度、順序、哪些功能放第一層、元件長相。

## 安裝

套件根目錄就是這個 repo（含 `SKILL.md`、`skill.json`、`builder/`、`conformance/`）。安裝原則：**host 有原生 plugin / skill 機制就優先用，否則載入 `adapters/` 對應的說明**。

- **Claude Code**：用 `.claude-plugin/marketplace.json` 加入 marketplace，或直接讀 `.claude-plugin/plugin.json` + `adapters/claude-code/CLAUDE.md`。
- **Codex**：把整個 repo 當 folder skill package（`SKILL.md` 就在根目錄），參考 `adapters/codex/README.md`。
- **OpenHands**：用 `.plugin/plugin.json`（搭配 `.openhands/setup.sh`），參考 `adapters/openhands/README.md`。
- **其他 host**：載入 `adapters/` 下對應檔案（見下方支援列表）。

裝完用 smoke test 驗證（需要 Go）：

```sh
cd conformance
go mod download
go run ./tools/wa3 bundle-check
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/wa3-board.draft.tdy --mock-demo /tmp/wa3-board.mock-demo.json
go run ./tools/wa3 trust /tmp/wa3-board.draft.tdy
```

預期：`bundle check ok`、產出一份草稿 `.tdy`、trust 狀態為 `unsigned_draft`。
沒有 Go 環境時：只做靜態編輯，把 Go 指令交給有安裝 Go 的主機跑，**不要宣稱已通過**。

## 支援哪些 agent

| Host | 安裝方式 | Adapter |
|---|---|---|
| Claude Code | Claude plugin metadata | `.claude-plugin/plugin.json`、`adapters/claude-code/CLAUDE.md` |
| Codex | Folder skill package | `SKILL.md`、`adapters/codex/README.md` |
| OpenHands | 原生 plugin metadata | `.plugin/plugin.json`、`adapters/openhands/README.md` |
| LangGraph | Graph / assistant 整合範例 | `adapters/langgraph/` |
| Voiceflow | Function / tool 整合範例 | `adapters/voiceflow/` |
| Mistral Agents / Le Chat | Agent / tool / MCP 整合範例 | `adapters/mistral/` |
| Haler | Skill 設定檔 | `adapters/haler/haler.skill.json` |
| OpenClaw | Adapter 說明 | `adapters/openclaw/OPENCLAW.md` |
| Hermes | Adapter 說明 | `adapters/hermes/HERMES.md` |
| Eigent | 官方安裝格式待定 | `adapters/eigent/README.md` |

## 這個 repo 有什麼

| 路徑 | 內容 |
|------|------|
| `board.tdy` | 最小留言板功能檔範例 |
| `builder/` | 從問答 JSON 產生功能檔的 schema、功能模板、範例 |
| `builder/templates/catalog.json` | **功能模板**索引（決定「裝哪些功能」） |
| `design_templates/` | display-only 的 `*.dsdy` 視覺架構檔 |
| `design_templates/catalog.json` | **設計架構模板**索引，用 tag 推薦你熟悉的介面 |
| `conformance/` | Go conformance kit 與 golden vectors |
| `adapters/`、`skills/wa3-spec/` | 各家 agent / runtime 的接法 |
| `docs/` | 完整 publish / consume 流程文件 |

> 功能模板（`builder/templates/`）和設計架構模板（`design_templates/`）是兩條獨立的選擇：前者決定「裝哪些功能」，後者決定「介面長相」。

## 安全模型（一句話版）

可信的是 `wa3-core`，不是 renderer、也不是產生草稿的 LLM。`.tdy` 的可信度來自 canonical Ed25519 簽章；AI 產出只是建議，schema、權限、風險閘、簽章一律由確定性程式碼把關；`*.dsdy` 完全 display-only；會改資料的動作一定重新檢查契約並要求使用者確認。

---

<a name="english"></a>
# English

## What it is

- **Function file `*.tdy`** — declares *what it can do*: data shapes, executable actions, permissions, confirmation rules. Signed and verifiable.
- **Design file `*.dsdy`** — declares *what it looks like*: layout, density, component roles, interaction style. Display-only; carries no permissions or actions.

The same function file can be applied to any design file. Swap the underlying `.dsdy` and the whole interface unifies into the shell you already know — the functionality is untouched.

## Why split them

Everyday pain points: every app has a different interface, so switching tools means relearning where the buttons are; too many preloaded features you can't trim; the features you use most are buried on a second level; and even when an app exports "quick templates," you have to fish one out of every app separately.

WA3's answer: carry **one personal `dsdy`** (the interface shell you're comfortable with), and whatever function file you load is rendered with the same structure. The interface is unified, trimmable, and your frequent actions can be promoted to the first level — while the functionality is still decided by each separately signed `.tdy`, with no cross-contamination.

## A concrete example

Say you receive a message-board function file `board.tdy` (declaring `read / submit / react / search`):

1. The runtime **verifies the signature** of `board.tdy`, confirming actions, permissions, and confirmation rules.
2. It then **asks which interface design to apply**: if you've set a personal default design, it asks about that first; if you haven't, or the design the function file recommends isn't one you own, the agent uses **tag matching** in `design_templates/catalog.json` to list the 3 closest designs you *already have* for you to choose — or to skip.
3. Once chosen, `submit` binds to your design's "primary submit button" role, `react` to an "inline icon action," and so on. The look is unified, but the actions always come from the verified `board.tdy`.

A design is **recommended, never forced**: even if a function file names a suggested design, the agent won't force it when you haven't downloaded it or don't want it — you decide. A design can never add functionality or change permissions; it only affects layout, density, ordering, which features sit on the first level, and component appearance.

## Installation

The package root is this repo (it contains `SKILL.md`, `skill.json`, `builder/`, `conformance/`). Principle: **prefer the host's native plugin/skill mechanism; otherwise load the matching `adapters/` instructions.**

- **Claude Code**: add via `.claude-plugin/marketplace.json`, or read `.claude-plugin/plugin.json` + `adapters/claude-code/CLAUDE.md`.
- **Codex**: install the whole repo as a folder skill package (`SKILL.md` sits at the root); see `adapters/codex/README.md`.
- **OpenHands**: use `.plugin/plugin.json` (with `.openhands/setup.sh`); see `adapters/openhands/README.md`.
- **Other hosts**: load the matching file under `adapters/` (see the support table below).

Verify the install with the smoke test (requires Go):

```sh
cd conformance
go mod download
go run ./tools/wa3 bundle-check
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/wa3-board.draft.tdy --mock-demo /tmp/wa3-board.mock-demo.json
go run ./tools/wa3 trust /tmp/wa3-board.draft.tdy
```

Expected: `bundle check ok`, a draft `.tdy` is written, trust state is `unsigned_draft`.
Without Go: make static edits only and run the Go steps on a host that has Go — **do not claim they passed**.

## Supported agents

| Host | Install mode | Adapter |
|---|---|---|
| Claude Code | Claude plugin metadata | `.claude-plugin/plugin.json`, `adapters/claude-code/CLAUDE.md` |
| Codex | Folder skill package | `SKILL.md`, `adapters/codex/README.md` |
| OpenHands | Native plugin metadata | `.plugin/plugin.json`, `adapters/openhands/README.md` |
| LangGraph | Graph / assistant integration sample | `adapters/langgraph/` |
| Voiceflow | Function / tool integration sample | `adapters/voiceflow/` |
| Mistral Agents / Le Chat | Agent / tool / MCP integration sample | `adapters/mistral/` |
| Haler | Skill config file | `adapters/haler/haler.skill.json` |
| OpenClaw | Adapter notes | `adapters/openclaw/OPENCLAW.md` |
| Hermes | Adapter notes | `adapters/hermes/HERMES.md` |
| Eigent | Official format pending | `adapters/eigent/README.md` |

## What's in this repo

| Path | Contents |
|------|----------|
| `board.tdy` | Minimal collaborative message-board function file |
| `builder/` | Schema, function templates, and examples for building a function file from answers JSON |
| `builder/templates/catalog.json` | **Function** template index (decides *which features to include*) |
| `design_templates/` | Display-only `*.dsdy` visual design files |
| `design_templates/catalog.json` | **Design** template index; recommends a familiar interface by tag |
| `conformance/` | Go conformance kit and golden vectors |
| `adapters/`, `skills/wa3-spec/` | How each agent / runtime integrates |
| `docs/` | Full publish / consume flow docs |

> Function templates (`builder/templates/`) and design templates (`design_templates/`) are two independent choices: the former decides *which features*, the latter decides *the look*.

## Safety model (one line)

Trust `wa3-core`, not the renderer and not the drafting LLM. A `.tdy`'s trust comes from a canonical Ed25519 signature; AI output is only a suggestion, while schema, permissions, risk gates, and signing are enforced by deterministic code; `*.dsdy` files are display-only; mutating actions always re-check the contract and require user confirmation.

---

<a name="日本語"></a>
# 日本語

## これは何か

- **機能ファイル `*.tdy`**：「何ができるか」を宣言します——データ構造、実行可能なアクション、権限、確認ルール。署名済みで検証可能です。
- **デザインファイル `*.dsdy`**：「どう見えるか」を宣言します——レイアウト、密度、コンポーネントの役割、操作スタイル。表示専用で、権限やアクションは一切持ちません。

同じ機能ファイルを任意のデザインファイルに適用できます。下層の `.dsdy` を差し替えれば、機能はそのままに、画面全体が使い慣れたシェルに統一されます。

## なぜ分けるのか

日常の悩み：アプリごとに画面が違い、ツールを変えるたびにボタンの位置を覚え直す。使わない機能がプリロードされすぎて削れない。よく使う機能が二階層目に隠れている。アプリが「ショートカットテンプレート」を書き出せても、アプリごとに別々に取り出すのが面倒。

WA3 の答え：**自分の `dsdy`（使い慣れた画面シェル）を一つ持ち歩く**だけで、どの機能ファイルを読み込んでも同じ構造で描画されます。画面は統一され、不要な機能は削減でき、よく使う操作を第一階層に引き上げられます。一方で機能自体は、それぞれ個別に署名された `.tdy` が決め、互いに干渉しません。

## 具体例

メッセージボードの機能ファイル `board.tdy`（`read / submit / react / search` を宣言）を受け取ったとします：

1. ランタイムはまず `board.tdy` の**署名を検証**し、アクション・権限・確認ルールを確かめます。
2. 次に**どのデザインを適用するか**を尋ねます。個人の既定デザインを設定済みならそれを最初に確認し、未設定、または機能ファイルが推奨するデザインを持っていない場合は、`design_templates/catalog.json` の**タグ照合**で、あなたが*すでに持っている*最も近い 3 つのデザインを提示して選ばせます（適用しない選択も可）。
3. 選ぶと、`submit` はあなたのデザインの「主送信ボタン」の役割に、`react` は「インラインのアイコン操作」に割り当てられます。見た目は統一されますが、アクションは常に検証済みの `board.tdy` に由来します。

デザインは**推奨であり強制ではありません**。機能ファイルが推奨デザインを指定していても、ダウンロードしていない・使いたくない場合、エージェントは無理に適用しません。決めるのはあなたです。デザインが機能を追加したり権限を変えたりすることは決してなく、レイアウト・密度・順序・第一階層に置く機能・コンポーネントの見た目だけに影響します。

## インストール

パッケージのルートはこのリポジトリ（`SKILL.md`、`skill.json`、`builder/`、`conformance/` を含む）です。原則：**ホストにネイティブの plugin / skill 機構があれば優先し、なければ `adapters/` の対応手順を読み込む。**

- **Claude Code**：`.claude-plugin/marketplace.json` で marketplace に追加するか、`.claude-plugin/plugin.json` + `adapters/claude-code/CLAUDE.md` を読み込む。
- **Codex**：リポジトリ全体を folder skill package として導入（`SKILL.md` はルートにあります）。`adapters/codex/README.md` を参照。
- **OpenHands**：`.plugin/plugin.json`（`.openhands/setup.sh` と併用）。`adapters/openhands/README.md` を参照。
- **その他のホスト**：`adapters/` 配下の対応ファイルを読み込む（下の対応表を参照）。

スモークテストで確認（Go が必要）：

```sh
cd conformance
go mod download
go run ./tools/wa3 bundle-check
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/wa3-board.draft.tdy --mock-demo /tmp/wa3-board.mock-demo.json
go run ./tools/wa3 trust /tmp/wa3-board.draft.tdy
```

期待値：`bundle check ok`、ドラフト `.tdy` が書き出される、trust 状態が `unsigned_draft`。
Go 環境がない場合：静的な編集のみ行い、Go のステップは Go を入れたホストで実行してください。**通過したと主張しないこと。**

## 対応エージェント

| Host | インストール方式 | Adapter |
|---|---|---|
| Claude Code | Claude plugin metadata | `.claude-plugin/plugin.json`、`adapters/claude-code/CLAUDE.md` |
| Codex | Folder skill package | `SKILL.md`、`adapters/codex/README.md` |
| OpenHands | ネイティブ plugin metadata | `.plugin/plugin.json`、`adapters/openhands/README.md` |
| LangGraph | Graph / assistant 連携サンプル | `adapters/langgraph/` |
| Voiceflow | Function / tool 連携サンプル | `adapters/voiceflow/` |
| Mistral Agents / Le Chat | Agent / tool / MCP 連携サンプル | `adapters/mistral/` |
| Haler | Skill 設定ファイル | `adapters/haler/haler.skill.json` |
| OpenClaw | Adapter ノート | `adapters/openclaw/OPENCLAW.md` |
| Hermes | Adapter ノート | `adapters/hermes/HERMES.md` |
| Eigent | 公式フォーマット未定 | `adapters/eigent/README.md` |

## このリポジトリの中身

| パス | 内容 |
|------|------|
| `board.tdy` | 最小のメッセージボード機能ファイル例 |
| `builder/` | 問答 JSON から機能ファイルを生成する schema・機能テンプレート・例 |
| `builder/templates/catalog.json` | **機能**テンプレート索引（「どの機能を入れるか」を決める） |
| `design_templates/` | 表示専用の `*.dsdy` ビジュアルデザインファイル |
| `design_templates/catalog.json` | **デザイン**テンプレート索引。タグで使い慣れた画面を推薦 |
| `conformance/` | Go conformance kit と golden vectors |
| `adapters/`、`skills/wa3-spec/` | 各エージェント / ランタイムの連携方法 |
| `docs/` | 発行 / 取り込みフローの全ドキュメント |

> 機能テンプレート（`builder/templates/`）とデザインテンプレート（`design_templates/`）は独立した二つの選択です。前者は「どの機能か」、後者は「見た目」を決めます。

## セーフティモデル（一行）

信頼するのは `wa3-core` であり、renderer でも下書きを作る LLM でもありません。`.tdy` の信頼は canonical Ed25519 署名から得られ、AI の出力は提案にすぎず、schema・権限・リスクゲート・署名は確定的なコードが強制します。`*.dsdy` は完全に表示専用で、データを変更するアクションは必ず契約を再確認しユーザーの確認を求めます。

---

## License / 授權 / ライセンス

This project is licensed under the MIT License. See `LICENSE` for the full text.

本專案採用 MIT License 授權；完整授權條款請見 `LICENSE`。

本プロジェクトは MIT License の下で提供されています。全文は `LICENSE` を参照してください。
