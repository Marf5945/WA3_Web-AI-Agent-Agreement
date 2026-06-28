# WA3 Import and TOFU Flow

This is the v1 fallback for a consumer who receives a WA3 share link before a
full provider adapter exists.

## Manual Import

0. Run `wa3 link-check <source>` or the host Runtime's equivalent static source
   guard. Reject rendered editor/preview pages, unstable signed URLs, and
   token-shaped query parameters before any import UI offers a download.
1. Download the `.tdy` bytes from the shared location into a quarantine/import
   staging area. If the link is a rendered page, ask the publisher for a direct
   file handle or byte download.
2. Run `wa3 trust <file>` or the host Runtime's equivalent trust inspection
   before rendering controls or exposing any operation.
3. If the result is `unsigned_draft`, treat it as a local/mock preview only.
4. If the result is `test_signed`, do not pin it and do not use it for
   production.
5. If the result is `sig_mismatch` or `revoked`, stop.
6. If the result is `signed_untrusted_key`, compare the public key fingerprint
   with the publisher's out-of-band fingerprint.
7. If it matches, pin the publisher id plus public key fingerprint in the local
   trust store.
8. Re-run trust inspection with the pinned key. The expected state is
   `signed_trusted`.

## TOFU Decision

Pin only when all are true:

- The app id and publisher id are expected.
- The public key fingerprint matches an out-of-band message.
- The source URL or provider handle is the one the publisher intended.
- The file is not TEST ONLY signed.
- There is no revocation or key-rotation warning.

Do not pin when:

- The publisher sends the fingerprint only inside the same untrusted file.
- The key changed from an already pinned app without a valid KR chain.
- The file came from a rendered editor page or an unstable export.
- The Runtime cannot distinguish draft, test, untrusted, trusted, revoked, and
  mismatch states.

## Host Import Guard

Hosts that show a generic "download" button for `.tdy` files must keep that
button separate from trust and execution. A successful download means only "the
bytes were saved"; it never means "the contract is trusted." The host should:

- Store downloads in an import staging/quarantine location until trust succeeds.
- Show the trust state from deterministic code, not from renderer text.
- Disable provider calls unless the state is `signed_trusted`.
- Use mock/local preview labels for `unsigned_draft`, `test_signed`, and
  `signed_untrusted_key`.
- Route every mutating action through core confirmation after re-reading the
  verified contract.

## Suggested UI Copy

Use plain states:

- Draft: "This file is not signed. You can preview it, but it is not trusted."
- Test: "This uses a demo key and cannot be trusted for production."
- Untrusted key: "The signature is valid, but this publisher key is new here."
- Trusted: "The signature is valid and the publisher key is pinned."
- Revoked: "This key or version has been revoked."
- Invalid: "The file signature does not match its contents."

TOFU pin state belongs to the consumer Runtime trust store. It must not be
written back into the `.tdy`.
