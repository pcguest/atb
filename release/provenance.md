# Release provenance policy

ATB releases use one maintainer identity and preserve published history.

## Canonical identity

- Author and committer: `Patrick Guest <patrickcguest@proton.me>`.
- Signing key fingerprint:
  `27ED 2C1B 552E 6FBF FAD7 6703 11D0 FC6F 890E F892`.
- The fingerprint is authoritative. The key currently also carries the
  historical `patrickcguest@gmail.com` user ID; that user ID does not indicate
  a second author.

Release-line commits and tags must have a valid signature from this key. Commit
messages must not contain `Co-Authored-By`, bot `Signed-off-by`, or on-behalf-of
trailers. Automated tooling may be described in the change record, but it is
not represented as a release author.

## v1.15 history decision

The signed `v1.15.0` tag is immutable and continues to point at commit
`e0fe8a2`. It is the gated source release for that version.

An unpublished candidate commit, `10a5508`, contained a
`Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>` trailer. Before
`v1.15.1` was tagged or released, it was replaced by signed commit `4ddeb6b`
with the same version-marker change and no authorship trailers. The original
object is retained only in the read-only local backup
`atb-backup-pre-trailer-rewrite-20260629.git`.

No published tag was moved and no GitHub Release was rewritten.

## Release checks

Before a release tag is pushed:

1. Verify every release-line commit and the candidate tag signature.
2. Search commit bodies for prohibited authorship trailers.
3. Run `make gate-gold-release`.
4. Run the release workflow by manual dispatch to build and verify artifacts
   without publishing.
5. Confirm the tag version matches the CLI, Python, TypeScript, and web
   versions.

The tag-triggered workflow creates a draft GitHub Release, publishes verified
registry artifacts, attaches the signed release evidence bundle, and only then
marks the Release public.
