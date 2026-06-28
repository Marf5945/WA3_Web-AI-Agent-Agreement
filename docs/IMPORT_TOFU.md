# W3A Import and TOFU Flow

This is the v1 fallback for a consumer who receives a W3A share link before a
full provider adapter exists.

## Manual Import

1. Download the `.w3a` bytes from the shared location. If the link is a rendered
   page, ask the publisher for a direct file handle or byte download.
2. Run `w3a trust <file>` or the host Runtime's equivalent trust inspection.
3. If the result is `unsigned_draft`, treat it as a local preview only.
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

## Suggested UI Copy

Use plain states:

- Draft: "This file is not signed. You can preview it, but it is not trusted."
- Test: "This uses a demo key and cannot be trusted for production."
- Untrusted key: "The signature is valid, but this publisher key is new here."
- Trusted: "The signature is valid and the publisher key is pinned."
- Revoked: "This key or version has been revoked."
- Invalid: "The file signature does not match its contents."

TOFU pin state belongs to the consumer Runtime trust store. It must not be
written back into the `.w3a`.
