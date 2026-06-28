# W3A Promote Path

This document defines how a builder draft becomes a production-signed `.w3a`.
It is the documented path for report #3. It is not a test-signed-to-production
conversion flow.

## Rule

Production promotion means:

1. Start from a `.w3a` structure that already passed builder gates.
2. Rebuild or parse that same structure into the canonical model.
3. Remove any existing TEST ONLY signature container values.
4. Sign the canonical bytes with a production publisher key handle.
5. Write a new production `.w3a`.
6. Discard the test-signed artifact.

Because W3A signatures cover the parsed canonical structure, the signed content
bytes stay the same when the structure is unchanged. Only the signature value
and publisher public key change.

## File Placement

- Draft/demo outputs: `W3A_SPEC/output/` or `/tmp` during local checks.
- Golden vectors and TEST ONLY fixtures: `W3A_SPEC/conformance/vectors/`.
- Production private keys: never in this repository. Store them in OS credential
  storage, or in a passphrase-encrypted seed/key file outside the repo with
  strict permissions.
- Production `.w3a` release artifacts: write to the publisher's release/output
  location, then publish to the provider or `.well-known` host. Do not overwrite
  `board.w3a`, builder fixtures, or conformance vectors with production keys.

## v1 CLI Shape

Implemented advanced local commands:

```sh
w3a keygen --publisher <publisher_id> --key-out /secure/outside-repo/publisher-key.json
w3a sign --key-file /secure/outside-repo/publisher-key.json --in app.draft.w3a --out app.w3a
```

This v1 path is intentionally advanced and local. The key file contains the
private seed and must stay outside the repository, outside `.w3a`, outside
answers JSON, and outside logs. A host can wrap the same command shape with OS
credential storage later.

## Trust Path

After production signing, a consumer Runtime verifies the signature:

- valid signature + unpinned publisher key -> `signed_untrusted_key`
- user/admin accepts TOFU pin -> `signed_trusted`
- TEST ONLY key -> `E-TRUST-TESTKEY`, never pinnable
- revoked key -> `E-TRUST-REVOKED`

Pin state belongs in the consumer Runtime trust store, not in the `.w3a`.

For user-facing publishing and import fallback, see `docs/PUBLISH_CHECKLIST.md`
and `docs/IMPORT_TOFU.md`.

## Recovery

If the production private key is lost, the publisher cannot sign updates for
already pinned consumers. Normal recovery is KR rotation signed by the old key.
If no valid KR can be produced, consumers must follow the out-of-band trust
rebuild path in §27.4.
