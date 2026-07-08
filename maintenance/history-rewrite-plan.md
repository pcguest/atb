# Git history rewrite plan (prepared, not executed)

This runbook removes build artefacts and a large binary from git history before
the repository is made public. It is prepared for review. Do not run it without
the maintainer's explicit go-ahead. It rewrites every commit SHA and requires a
force-push.

## Why

The working tree is clean and the paths below are gitignored, but they remain in
history and bloat every clone.

| Path | Largest object | Notes |
| --- | --- | --- |
| `.atb-agent/bin/gosec` | 46.8 MB | Security scanner binary committed in v1.1.0 prep, later deleted. |
| `.gocache*` | many 5 to 16 MB blobs | Go build and test caches committed by mistake. Covers `.gocache`, `.gocache-docs`, `.gocache-go-build`, and `.gocache-go-test`; one variant holds a built `atb` binary. |

None of these paths are tracked now, and `git check-ignore` confirms they are
ignored, so a rewrite will not re-introduce them.

## Pre-flight

1. Confirm the working tree is clean and the v1.15.0 release is tagged and pushed first, or decide to rewrite before tagging. A rewrite changes the commit SHAs that existing tags and published GitHub releases point at.
2. Take a full mirror backup: `git clone --mirror . ../atb-backup.git`.
3. Confirm `git filter-repo` is installed (`brew install git-filter-repo`).

## Rewrite

```bash
git filter-repo \
  --path .atb-agent \
  --path-glob '.gocache*' \
  --invert-paths
```

The `.gocache*` glob is required: history contains `.gocache`, `.gocache-docs`,
`.gocache-go-build`, and `.gocache-go-test`. Naming only `.gocache` and
`.gocache-docs` leaves thousands of cache objects behind. `git filter-repo`
refuses to run on a non-fresh clone by default; use `--force` only after the
backup above exists.

This was dry-run on a mirror clone of the current repo. Expected results, which
you can confirm after running it for real:

| Measure | Before | After |
| --- | --- | --- |
| Pack size | 121.45 MiB | 4.13 MiB |
| `gosec` objects in history | 1 | 0 |
| `.gocache*` objects in history | many | 0 |
| `.atb-agent` objects in history | present | 0 |
| Branches (`refs/heads`) | 36 | 36 |
| Tags | 25 | 25 |

Only stale `refs/remotes/origin/*` are pruned (filter-repo removes the origin
remote); they are rebuilt on the next fetch. The rewritten tree builds, and
`gitleaks detect --log-opts=--all` reported no leaks over 886 commits.

## Secret scan before going public

History has not been scanned for credentials. Before the repository is public,
run a dedicated scanner over the full history and triage any hit:

```bash
gitleaks detect --source . --log-opts="--all"
```

If a secret is found, add its path or pattern to the same `filter-repo` run, then
rotate the exposed credential. Removal from history does not undo exposure.

## After the rewrite

- Force-push all branches and tags. Every collaborator must re-clone; old clones
  cannot fast-forward.
- Re-create or re-point any published release tags whose commit SHAs changed.
- Verify the result: `git rev-list --objects --all | grep -iE 'gosec|gocache'`
  must return nothing, and `git count-objects -vH` should show a smaller pack.

## Risk

This is a force operation. It is irreversible without the backup. It invalidates
existing commit SHAs, signed-tag references, and any external link that pins a
commit. Run it only with the maintainer's explicit confirmation.
