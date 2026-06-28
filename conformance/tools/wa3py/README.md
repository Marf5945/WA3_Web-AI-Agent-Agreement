# wa3lite — pure-Python fallback pipeline

The canonical WA3 implementation is Go (`conformance/tools/wa3`). `wa3lite.py`
is the **fallback channel**: when an agent environment cannot run Go, it drives
the whole journey in Python and can emit a runnable front+back app from a
verified `.tdy`.

Routing is handled by the `conformance/wa3` dispatcher: it prefers Go and falls
back to Python automatically; the `app` / `serve` subcommands always run in
Python (Go has no app generator).

```sh
conformance/wa3 build --answers ... --out app.tdy   # Go if present, else Python
conformance/wa3 app   --contract app.signed.tdy ...  # always Python
```

## What it covers

| Stage | Command | Notes |
| --- | --- | --- |
| Block 1 — collect backend | `init --answers a.json [--backend h]` | interactive prompt **or** `--backend`; scheme allowlist + `E-VALUE-SECRET` gate; writes `app.backend` (human_only / user_confirmed) |
| build source contract | `build --answers a.json --out app.tdy [--sign --key k.json --pin-out pin.json]` | replicates Go `emitContract`; can sign in one step |
| sign / trust | `keygen`, `sign`, `trust` | Ed25519; trust states match Go |
| Block 2 — providers | (used by operate) | `mock://` in-memory (seeded from `--mock-demo`) and `local://` JSON files under `--data-dir` (sandboxed, traversal-guarded). `gdrive://`/`api://`/`https://` fail closed |
| Block 3 — operate | `operate app.tdy --action id [--input k=v] [--confirm] --trusted-pub hex [--data-dir d] [--mock-demo seed]` | verify trust (only `signed_trusted` runs) → re-read action from verified contract → reject undeclared inputs + re-scan for secrets → **core-side confirm** for mutating (`ㄍㄞ=yes`) → dispatch to provider |
| app channel | `app --contract c.tdy --trusted-pub hex --out-dir d` | generates `index.html` from the verified contract |
| run app | `serve --contract c.tdy --trusted-pub hex --data-dir d [--port N]` | tiny HTTP backend wired to the operate layer; the confirm gate is enforced server-side (mutating calls without `confirm` → 403) |

## Parity with Go (verified)

Byte-for-byte against the Go-generated golden vectors in
`conformance/vectors/`:

* canonical form matches on all 5 canonical/extension vectors;
* the signature vector's Go signature verifies under Python, and Python
  re-signing the same seed reproduces Go's exact signature bytes (deterministic
  Ed25519).

So a contract signed by `wa3lite` is accepted by `wa3 trust`, and a Go-signed
contract operates under `wa3lite` — the fallback is wire-compatible, not a
look-alike.

## Safety boundaries (unchanged from the spec)

* The backend handle (`ㄏㄡ`) is a stable **namespace**, never a credential;
  the secret gate guarantees no token reaches it and providers never read,
  cache, accept, or emit credentials.
* operate refuses `unsigned_draft`, `signed_untrusted_key`, `test_signed`,
  `revoked`, and `sig_mismatch`.
* Mutating actions require explicit confirmation in core (the renderer cannot
  satisfy it).

## Requirements

Python 3.8+ and the `cryptography` package (Ed25519). No network access needed.
