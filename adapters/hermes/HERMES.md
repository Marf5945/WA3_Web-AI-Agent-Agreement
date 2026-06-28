# W3A for Hermes Agent

Use W3A as a contract layer for collaborative agent tools and web surfaces. The
Hermes runtime should keep a strict split between verified contract semantics,
renderer output, and user-confirmed operations.

## Entry Points

- Manifest: `../../skill.json`
- Quickstart: `../../AGENTS.md`
- Normative spec: `../../W3A-SPEC.md`
- Builder schema/templates: `../../builder/`
- Example board contract: `../../board.w3a`
- Conformance vectors: `../../conformance/vectors`

## Hermes Runtime Loop

1. Load bytes from a `.w3a` source.
2. Normalize, parse, classify namespace, and enforce §6 limits.
3. Rebuild canonical bytes and verify signature/trust before use.
4. Compile a renderer-safe interface plan.
5. Render a collaborative surface. The renderer reports only user intent.
6. Re-check the verified contract and require confirmation before mutating
   operate.

## Builder Loop

1. Collect guided answers from template questions.
2. Treat LLM suggestions as untrusted `system_suggested` content.
3. Let deterministic builder gates enforce schema ownership, secret scan,
   risk/confirm policy, canonicalization, lint, and trust output.
4. Write a draft or TEST ONLY signed `.w3a` only when every gate passes.

## Collaboration-Web Notes

- Use W3A for message boards, voting, task tiles, shared references, and
  local-first collaboration surfaces.
- Keep provider adapters behind the runtime.
- For AI summaries or drafting, send data as untrusted data blocks and land model
  output in display-only/draft surfaces.
