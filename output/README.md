# output/

Default destination for files produced by the W3A builder.

Everything an agent or the `w3a` CLI generates — draft contracts, test-signed
contracts, canonical hashes, and mock-provider demos — is written here so
generated artifacts never get mixed into the spec, schema, templates, or vectors.

## Convention

| Artifact | Path |
| --- | --- |
| Unsigned draft | `output/<name>.draft.w3a` |
| Test-signed (opt-in) | `output/<name>.test-signed.w3a` |
| Mock-provider demo | `output/<name>.mock-demo.json` |

Example build (run from `conformance/`, requires a Go toolchain):

```sh
go run ./tools/w3a build \
  --answers ../builder/examples/board.answers.json \
  --out ../output/board.draft.w3a \
  --mock-demo ../output/board.mock-demo.json
```

## Notes

- Contents are git-ignored (see `.gitignore` in this folder); only this README and
  the ignore rule are tracked, so the folder always exists but stays empty in
  version control.
- The deterministic build gate still applies: a file lands here only after schema,
  secret-scan, risk/confirm, canonical, and lint checks pass. A failed gate writes
  nothing here and returns a stable error code.
- Agents without a Go toolchain (e.g. Claude Code) cannot run `build`; they should
  edit statically and ask the user to run the command above on a host with Go.
