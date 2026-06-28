# WA3 for OpenClaw

This adapter is a platform-neutral instruction file for OpenClaw-style agents.
If OpenClaw provides a native manifest later, keep this file as the human-readable
entry point and map the manifest back to `../../skill.json`.

## Load

1. `../../skill.json`
2. `../../AGENTS.md`
3. `../../README.md`
4. `../../builder/answers.schema.json` and `../../builder/templates/` when
   running the guided builder flow
5. `../../conformance/README.md` when implementing or testing parser, builder,
   trust, or canonical behavior

## Do

- Review `.tdy` contracts against §6 safety rules, §8 canonical signing, §23.4
  share-link provider rules, §27 revocation/rotation, §28 AI injection defense,
  and §29 operate semantics.
- Compile only to an interface plan. Do not put secrets, backend URLs, provider
  tokens, or executable code into renderer-facing output.
- Ask for explicit user confirmation before any mutating operate.
- Treat authoring LLM output as untrusted suggestions and let deterministic
  builder gates decide schema ownership, secret/risk policy, and canonical form.

## Do Not

- Do not treat `.tdy` as code to execute.
- Do not let AI-generated text trigger operate directly.
- Do not reject unknown `ㄝ` extension fields just because the current runtime
  does not understand them.
- Do not invent new security-critical extension fields; use core namespace and a
  major version change.
