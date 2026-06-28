# W3A Conformance Kit (v0)

Minimal Go reference + golden vectors that pin cross-language behaviour for the
parts most likely to diverge. The conformance tool is Go-native and uses the
repository's Go module dependencies.

## What's here

- `tools/w3a/main.go` — Go-native reference: encoding normalization (§6A),
  version `ㄉㄞ` parsing (§30.1 / 附錄 A.3), `ㄝ` namespace classification
  (§3.4), stable error codes (§30.4), nested canonical serializer (§8 / §8.1),
  RL/KR canonical (§27.6), Ed25519 signature vectors, and bundle checks.
- `vectors/` — frozen golden vectors:
  - `version/cases.json` — `ㄉㄞ` parsing (good + reject), runtime_major=1.
  - `reject/cases.json` — encoding & structural rejects with error codes.
  - `canonical/header-basic/` — header reordered to canonical order + sha256.
  - `extension/header-ext/` — unknown/known `ㄝ` extensions sorted after core,
    preserved, + sha256.
  - `extension/action-ext/` and `extension/entity-ext/` — record-level `ㄝ`
    extensions moved after core fields and sorted by key.

## Setup (once)

The tool needs one module dependency (`golang.org/x/text/unicode/norm`, for NFC).
From `conformance/`, fetch it once (writes `go.sum`):

```sh
go mod download   # or: go mod tidy
```

## Run

```sh
go run ./tools/w3a gen-vectors       # regenerate vectors/
go run ./tools/w3a canonical some.w3a # print canonical bytes, sha256 on stderr
go run ./tools/w3a build --answers ../builder/examples/board.answers.json --out /tmp/board.draft.w3a --mock-demo /tmp/board.mock-demo.json
go run ./tools/w3a build --answers ../builder/examples/board.answers.json --out /tmp/board.test-signed.w3a --test-sign
go run ./tools/w3a keygen --publisher com.demo.publisher --key-out /tmp/w3a-demo-key.json
go run ./tools/w3a sign --key-file /tmp/w3a-demo-key.json --in /tmp/board.draft.w3a --out /tmp/board.production.w3a
go run ./tools/w3a trust /tmp/board.test-signed.w3a
go run ./tools/w3a trust --rl /tmp/list.w3a-rl /tmp/board.test-signed.w3a  # re-check §27 revocation list
go run ./tools/w3a bundle-check       # validate bundle + vectors
```

When you compile the CLI, build outside the repo so the binary never lands in
the scanned tree:

```sh
go build -o /tmp/w3a ./tools/w3a      # NOT: go build ./tools/w3a
```

`bundle-check` skips compiled build artifacts (the `w3a` binary, `*.test`,
`*.exe`, `*.o`, `*.out`) and these are `.gitignore`d, so they never enter the
published bundle. Every authored source file is still scanned.

CI should run `go run ./tools/w3a bundle-check`. To update golden vectors after
an intentional canonical change, run `go run ./tools/w3a gen-vectors` and review
the vector diff.

`build` is the builder v1 demo path. It treats LLM output as untrusted, validates
the answers file, rejects token-shaped secrets, enforces template-owned
`risk_class` and confirmation rules, canonicalizes the generated contract, and
only writes the `.w3a` when every gate passes. `--test-sign` uses the deterministic
TEST ONLY key; formal runtimes must reject it as `E-TRUST-TESTKEY`.

`keygen` and `sign` are the advanced local production-signing path. `keygen`
refuses to write private key files inside the W3A_SPEC repository; keep the key
file outside the repo or in a host credential store wrapper. `sign` reuses the
same canonical serializer as `trust` and writes a production-signed `.w3a`.

## Scope (v0)

This v0 pins the byte-level behaviours needed before runtimes can interoperate:

1. **Nested-document canonical** — DONE (§8.1). entities (`ㄕ`), actions (`ㄓㄠ`),
   blocks (`ㄑㄩ`) with indented sub-fields, list-value ordering. Vector:
   `canonical/board-nested`.
2. **RL/KR canonical** (§27.6 / §8.1) — DONE. `撤銷項` list sorting + fixed field
   order. Vectors: `rl/basic`, `kr/basic`.
3. **Signature vectors** — DONE. Ed25519 sign/verify round-trip with a fixed
   test seed, marked "TEST KEY — DO NOT USE IN PRODUCTION". Vector:
   `signature/board-nested`.

## Pinned decisions

`tools/w3a` pins a few byte-level decisions so other implementations can match
the golden vectors exactly:

- **Canonical separator/spacing.** §8 confirms the canonical line form is
  `key：value`, using the full-width `：` (U+FF1A) with **no surrounding spaces**.
  Zhuyin applies to structural tokens; the separator is fixed punctuation, not a
  token.
- **NFC implementation.** The Go reference uses the official
  `golang.org/x/text/unicode/norm` package. Go's standard library does not
  provide Unicode normalization; replacing this with a no-op changes canonical
  bytes.
- **Control/format stripping.** The reference does not use Unicode category
  tables for canonical stripping. It follows the explicit §8.1 code point lists:
  strip C0/C1 controls except LF, strip the listed zero-width/bidi controls from
  values, and reject those zero-width/bidi controls in structural tokens.
- **Sort key.** Core extension keys and record ids are ordered by ascending
  UTF-8 string bytes. For valid UTF-8 this is equivalent to Unicode code point
  order, so Go can use ordinary string sorting. Demonstrated by
  `extension/header-ext` (`ㄝㄎㄜ` before `ㄝㄖㄜ`).
- **Record extension placement.** Every container uses the same rule: emit core
  fields first, then all `ㄝ` extension fields sorted by key. Demonstrated by
  `extension/action-ext` and `extension/entity-ext`.
- **Higher-major rejection** is evaluated against `runtime_major` (here 1), so
  `2` and `10` reject with `E-VERSION-MAJOR`; format errors (`01`, `1.x`, …)
  reject with `E-VERSION-FORMAT` and take precedence is not needed since they are
  syntactically distinct.

## Spec items this addresses

附錄 A.3 (version parsing) — covered by `version/cases.json`.
附錄 A.2 / §8 (unknown-extension preservation) — demonstrated by
`extension/header-ext`; the reference uses "normalize + keep + sort", equivalent
to byte-preservation when NFC is applied identically by signer and verifier.
附錄 A.11 (validator dependency) — resolved for this repository: the conformance
kit is Go-native and tied to the existing Go module.
