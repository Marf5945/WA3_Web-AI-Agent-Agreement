# W3A Publish Checklist

Use this checklist after a draft has passed builder gates and, for production
sharing, has a valid production signature.

## Before Publishing

- Build from confirmed `answers.json`; do not publish raw agent output.
- Run the builder gates and keep the canonical SHA-256 shown by the tool.
- Sign with a production publisher key, not the TEST ONLY key.
- Keep the private key outside this repository and outside the `.w3a`.
- Record the publisher public key fingerprint in an out-of-band channel, such as
  email, chat, release notes, or an admin-managed trust list.

## Link Rules

Use a byte-stable source:

- Good: GitHub raw, static HTTPS hosting, R2/S3 object, `.well-known` path, or a
  Google Drive file id consumed through `gdrive://FILE_ID`.
- Avoid: expiring signed URLs in the contract.
- Reject: rendered editor pages such as `docs.google.com/.../edit`, preview
  pages, or any page that returns generated HTML instead of the `.w3a` bytes.

For Google Drive, the user-facing flow should be:

1. Upload the signed `.w3a` as a file.
2. Set sharing to read-only.
3. Copy the file id, not the edit page URL.
4. Publish or share a stable handle such as `gdrive://FILE_ID`.
5. Share the public key fingerprint out-of-band.

## Static Guard

A publisher tool or adapter should fail closed when:

- The publish link contains `/edit`, `/preview`, or a non-byte render page.
- The source contains token-shaped query parameters.
- The source cannot produce stable bytes for hashing.
- The content SHA-256 changes between two immediate fetches without an explicit
  rebuild/re-sign step.
- The public key is a TEST ONLY key.

## Release Note Fields

Every release should include:

- App id and version.
- Publisher id.
- Public key fingerprint.
- Content SHA-256.
- W3A URL or provider handle.
- Optional RL/KR locations when available.

Read-only sharing reduces accidental edits, but trust still comes from
signature verification and the pinned publisher key.
