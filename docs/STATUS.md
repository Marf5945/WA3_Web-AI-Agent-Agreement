# W3A Bundle — Status

Single authoritative spec: `W3A-SPEC.md` (§0–§31 + Appendix A). A byte-identical
copy is mirrored at `skills/w3a-spec/references/W3A-SPEC.md` (enforced by
`bundle-check`).

## Done

**Specification.** Core format and safety rules (§0–§12); Runtime / Canvas
sandbox / Skill Host (§13–§26); revocation & rotation with normative RL/KR
canonical (§27, incl. §27.6/§27.7); AI injection defense (§28); operate
semantics — idempotency / retry / concurrency / paging (§29); version negotiation
and error codes (§30); backward-compatibility rules and the `ㄝ` extension
namespace (§31, §3.4); nested canonical ordering and normalization (§8.1);
share-link provider allowlist (§23.4).

**Conformance kit (Go-native).** `conformance/tools/w3a` implements encoding
normalization, version parsing, namespace classification, the nested canonical
serializer, RL/KR canonical, stable error codes, golden-vector generation, and
`bundle-check`. Ed25519 uses the Go standard library (`crypto/ed25519`).

**Golden vectors** (`conformance/vectors/`): version parsing (accept + reject),
encoding/structural rejects with error codes, header canonical, `ㄝ` extension
ordering, full nested document canonical, RL and KR canonical, and a
deterministic Ed25519 signature (test key only — see that vector's README).

**Portable bundle.** `skill.json` is the single source of truth; `AGENTS.md` is
the cross-agent entry; adapters exist for Codex, Haler, Claude Code, OpenClaw,
and Hermes-style agents.

**Builder v1 demo.** `builder/answers.schema.json`, the `board` / `task_list` /
`feedback_form` / `product_showcase` / `mobile_product_app` templates, the
functional template catalog, sample answers, reject fixtures, `w3a build`,
`w3a keygen`, `w3a sign`, `w3a trust`, TEST ONLY signing, secret/risk gates,
`custom_generic` webpage-extraction answers, and mock-provider demo JSON are
present.
`bundle-check` exercises the happy path and the key reject cases.

**Agent install adapters.** Root `SKILL.md`, `docs/INSTALL_AGENTS.md`, `.plugin/`,
`.openhands/` (setup + hooks), `.claude-plugin/`, and adapters for Codex, Haler,
Claude Code, OpenHands, OpenClaw, Hermes, LangGraph, Voiceflow, Mistral, and
Eigent are present. Installable-skill targets: Claude Code / Codex
(`.claude-plugin/`), OpenHands (`.plugin/` + `.openhands/`), and Eigent (its
agent-skills format is Claude-Code-compatible — install the bundle as a `.zip`
with a root `SKILL.md`, or via `npx @eigent-ai/agent-skills`). LangGraph /
Voiceflow / Mistral remain integration sketches (graph config / function-API /
Agents-Tools-MCP), not skill packages.

## Builder v1 decisions

Builder v1 is an authoring safety gate, not merely a file generator: it should
let non-markdown users safely produce a parseable, canonicalizable, no-secret
draft `.w3a`.

- Output: default `app.draft.w3a` + canonical hash. `app.test-signed.w3a` is
  produced only with `--test-sign`.
- Draft signature: include `ㄓㄤ` with publisher/public key if known; keep
  `ㄓㄥ：` blank. Runtime should report `E-TRUST-UNSIGNED` for blank signatures.
- Test signing: TEST ONLY keys are permanently denied by formal runtimes
  (`E-TRUST-TESTKEY`) and exist only for vectors/demo.
- Production promotion: do not convert `app.test-signed.w3a` into a formal file.
  Reuse the same canonical structure and sign it again with a production
  publisher key handle; discard the test-signed artifact.
- Publisher keys: v1 formal signing is CLI/advanced only (`keygen`, `sign`, and
  `PROMOTE.md`). The CLI key-file path refuses repo-local private keys. Haler UI
  v1 produces unsigned drafts and does not manage private keys.
- Wizard answers: JSON, system-filled by Haler/Codex/Claude-style agents, with
  `answers_schema_version`.
- Authoring LLM is an untrusted suggester, not a gate. Code must re-check every
  LLM claim through schema ownership, template/catalog data, secret-scan,
  risk/confirm policy, canonicalization, and lint.
- `answers.schema.json` must mark field ownership: `llm_suggestable`,
  `code_owned`, or `human_only`.
- Resume model: one `answers.json` remains the single source of truth; review
  state lives inside it.
- Provenance: `template_default`, `system_suggested`, `user_confirmed`.
- LLM may only write `system_suggested`; only explicit human review may promote
  a value to `user_confirmed`.
- Templates v1: `board`, `task_list`, `feedback_form`, `product_showcase`,
  `mobile_product_app`, and `custom_generic` for webpage extraction.
  `product_showcase` is the default functional recommendation for product
  introduction pages, landing pages, solution showcases, brochure sites, and
  public product pages. `mobile_product_app` is the default functional
  recommendation for app-style product experiences, mobile product apps,
  bottom-tab product apps, HMI apps, and field-engineer product apps.
  `document_search` moves to v1.1 because provider auth/search semantics are a
  larger slice.
- Suggested actions must show one-line purpose + data impact, with choices:
  keep, remove, rename, or accept all defaults.
- Feature Manifest must explicitly tell the user they may remove listed webpage
  features and may choose presentation preferences such as larger text, density,
  colors, hidden/visible blocks, or a reference design format.
- Mutating actions default to `ㄍㄞ：yes` and `ㄖㄣ：yes`. Every action carries a
  code-owned `risk_class` (`read` / `low_mutate` / `high_mutate` / `irreversible`)
  that neither the LLM nor the user may set or lower. Confirm can be disabled only
  for `risk_class == low_mutate` with explicit human override, warning, and
  `confirm_disabled_by_user`; `high_mutate` and `irreversible` cannot disable
  confirmation.
- Secret gate: answers JSON and `.w3a` must hard-error on token-shaped values
  (`E-VALUE-SECRET`) and direct users to Runtime credential store.
- Haler v1 CTA: primary "從模板建立 W3A"; secondary "檢查 W3A 檔案".
- Added user-flow docs: `EXTRACT_CONTRACT.md`, `PUBLISH_CHECKLIST.md`,
  `IMPORT_TOFU.md`, and `RENDER_PIPELINE.md`.

## Build / verify

```sh
cd conformance
go mod download              # one-time: fetch golang.org/x/text
go run ./tools/w3a bundle-check
```

`bundle-check` validates: manifests parse, every manifest path exists, the spec
and its skill reference are byte-identical, the on-disk vectors match a fresh
regeneration, and no stale markers remain.

## Next

- Replace the builder demo's minimal schema checks with full JSON Schema
  validation when the runtime dependency policy is decided.
- Wire the Haler app UI to the builder projection (template picker, guided
  review, and repair actions); the portable manifest paths are already present.
- Replace integration sketches with native manifests when LangGraph, Voiceflow,
  Mistral, or Eigent host-specific packaging requirements are finalized.
- Claude Code plugin install — `.claude-plugin/plugin.json` and
  `.claude-plugin/marketplace.json` are in place; the plugin root is `W3A_SPEC`
  and the `skills/w3a-spec/` skill is auto-discovered. Install via
  `/plugin marketplace add <owner/repo>` then `/plugin install w3a-spec@w3a`.
- OpenClaw / Hermes native manifests — pending each platform's official manifest
  schema; the generic markdown adapter is the fallback until then.
- Haler installer manifest — `haler.skill.json` stays a projection of
  `skill.json` until the Haler installer schema is fixed.
- Extend the mock demo from inspectable JSON to a minimal operate loop when the
  runtime host surface is ready.

## Still pending decisions

- Final Haler UI projection for the guided questions and review screen.
- Exact migration/downgrade rules when a template version bump changes fields
  that were previously `user_confirmed`.
- Formal publisher-key UX split:
  - v1: CLI `keygen` / `sign`, local key-file path outside the repo, production
    re-sign path documented in `PROMOTE.md`.
  - v1.2: OS credential-store integration, import/export UI, encrypted seed
    backup, KR/RL management UI, and provider publish integration.
- Confirmed defaults still to wire into implementations: OS credential store
  first (macOS Keychain / Windows Credential Store / libsecret), encrypted seed
  file fallback with strict permissions, and RL/KR publication under
  `.well-known` using hash/signature pinning rather than URL trust.
- Real provider contracts beyond the current mock-provider demo.
- Official Haler installer schema; current Haler adapter remains a projection
  until that schema is fixed.

## Known gaps (deferred to v1.x)

- Revocation freshness: `w3a trust --rl <file.w3a-rl>` is a structural membership
  pre-check only. Verifying the revocation list's own publisher signature per
  §27 before honoring its entries is not yet implemented.
- Version re-review: when a publisher ships a new signed version of an app a
  consumer already pinned, there is no defined "re-review and re-pin" flow yet.
  `docs/IMPORT_TOFU.md` covers first import and key rotation, not version bumps.
- Consumer-side RL/KR fetch cadence and `.well-known` refresh are still manual.
