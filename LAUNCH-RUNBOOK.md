# ATB v1.15.0 and Custos launch runbook

Ordered, idempotent steps for the irreversible actions. Everything here is yours
to run. Nothing in this file has been executed against a real remote, a real
registry, a live site, or repository visibility. Commands assume the maintainer
prefix `GOCACHE=$(pwd)/.gocache/go1.26.4 GOTOOLCHAIN=go1.26.4` for Go.

Two repositories are involved:

- ATB at `~/atb`, branch `release/v1.15.0` (15 signed commits ahead of `main`).
- Custos at `~/custos-product`, branch `site/marketing-front` (2 signed commits).

## Decision 0 (read first): the history rewrite and signatures

The rewrite is proven safe on a throwaway clone (see
`docs/maintenance/history-rewrite-plan.md`): 121.45 MiB to 4.13 MiB, the gosec
blob and build caches gone, 36 branches and 25 tags preserved, the tree builds,
and gitleaks found no leaks over 886 commits.

It has one unavoidable cost. `git filter-repo` rewrites every commit, which
strips every GPG signature and changes every commit SHA. Verified on the clone:
the signed release commits come back unsigned. A clean small history and signed
historical commits cannot both be had.

gitleaks is clean, so the history holds no secrets. The 46 MB gosec binary and
the build caches are bloat, not exposure. That lowers the urgency.

Choose one path before going public:

- Path A (recommended): ship v1.15.0 with signatures intact, defer the rewrite.
  The signed commit chain is the integrity signal a sceptical reviewer checks;
  there is no secret to remediate. Schedule the rewrite as a separate, announced
  maintenance event later, accepting that it unsigns history then.
- Path B: run the rewrite before the visibility flip and before tagging, so the
  public repo starts clean and small. Accept that historical commits become
  unsigned; sign the v1.15.0 tag and every commit afterwards. Never run the
  rewrite after tagging or after going public (it breaks the tag and disrupts
  every clone).

The steps below assume Path A. If you take Path B, run section 6 first, on a
private clone, then resume at section 1 against the rewritten repo.

## 1. Final local verification (idempotent, no side effects)

```bash
cd ~/atb && git checkout release/v1.15.0
GOCACHE=$(pwd)/.gocache/go1.26.4 GOTOOLCHAIN=go1.26.4 make test-golden
GOCACHE=$(pwd)/.gocache/go1.26.4 GOTOOLCHAIN=go1.26.4 make hygiene-quick
ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh   # expect: all agree (1.15.0)
```

Clean-clone build gate (the regression that the embed-placeholder fix closed):

```bash
d=$(mktemp -d); git clone --no-local --branch release/v1.15.0 ~/atb "$d/c"
cd "$d/c" && GOCACHE=$(pwd)/.gocache/go1.26.4 GOTOOLCHAIN=go1.26.4 go build ./cmd/atb && echo OK
```

## 2. Set the security contact (one decision, one command per repo)

Pick one role mailbox, then replace the placeholder token. Abort the launch if
`security@replace-me.example` still appears anywhere after this.

```bash
cd ~/atb
grep -rl 'security@replace-me.example' . | grep -v node_modules | \
  xargs sed -i '' 's/security@replace-me.example/security@YOURDOMAIN/g'
cd ~/custos-product
grep -rl 'security@replace-me.example' . | grep -v node_modules | \
  xargs sed -i '' 's/security@replace-me.example/security@YOURDOMAIN/g'
```

Commit the result (signed) on each branch. Note: package author metadata in
`sdk/typescript/package.json` and `sdk/python/pyproject.toml` still carries the
maintainer's personal address by design; change it only if you want a
non-personal author.

## 3. Push the branches (reversible up to merge)

```bash
cd ~/atb && git push -u origin release/v1.15.0
cd ~/custos-product && git push -u origin site/marketing-front
```

Open pull requests, or fast-forward `main` if you prefer. Review with
`RELEASE-REVIEW-v1.15.0.md` before merging. Abort: delete the remote branch.

## 4. Tag once, never move (irreversible once pushed)

After `main` contains the release commit:

```bash
cd ~/atb && git checkout main && git pull --ff-only
git tag -s v1.15.0 -m "ATB v1.15.0"
git push origin v1.15.0
```

Tag-once discipline: if a problem is found after tagging, ship `v1.15.1`. Do not
force-move `v1.15.0`. There is no rollback for a pushed tag other than a new
tag.

## 5. Make the repositories public (irreversible in practice)

Flip visibility for `pcguest/atb` and `pcguest/custos-product` in the GitHub
settings. Do this only after sections 1 to 4 are clean. Once public, the current
history is public; this is why Path B, if chosen, must precede this step.

## 6. (Path B only) History rewrite, before public and before tagging

```bash
git clone --mirror ~/atb /tmp/atb-backup.git          # backup
cd ~/atb
git filter-repo --path .atb-agent --path-glob '.gocache*' --invert-paths
git reflog expire --expire=now --all && git gc --prune=now
git rev-list --objects --all | grep -E 'gosec|\.gocache' && echo "FAIL: residue" || echo "clean"
gitleaks detect --source . --log-opts="--all"
```

Then re-establish remotes, re-sign forward, and resume at section 1. Abort:
restore from `/tmp/atb-backup.git`.

## 7. Publish the SDKs (irreversible per version)

Registries are at 1.14.5; in-repo markers are 1.15.0 (verified). Publish 1.15.0:

```bash
# Python (atb-sdk, setuptools backend; needs the build front-end installed)
cd ~/atb/sdk/python && python -m build && twine upload dist/atb_sdk-1.15.0*
# TypeScript (@pcguest/atb-sdk, built with tsup; there is no prepublishOnly
# hook, so build explicitly before publish)
cd ~/atb/sdk/typescript && npm ci && npm run build && npm publish --access public
```

A published version is immutable. If a package is wrong, publish a patch
version; do not attempt to republish 1.15.0. The release workflow already
detects an already-published exact version and does not retry it. The canonical
path is the tag-triggered release workflow; the commands above are the manual
fallback.

## 8. Deploy the Custos site (reversible)

```bash
cd ~/custos-product
python3 -m http.server --directory site 8000   # local preview first
```

Confirm the page above the fold and the verification table render, then deploy
`site/` to any static host (S3 and CloudFront, GitHub Pages). The page loads no
external assets, so there is nothing to allowlist. Rollback: redeploy the prior
version or take the bucket offline.

## Order summary

Path A: 1, 2, 3, 4, 5, 7, 8. Defer 6.
Path B: 1, 6, 1 (again, on rewritten repo), 2, 3, 4, 5, 7, 8.
