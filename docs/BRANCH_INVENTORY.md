# Branch inventory (Wave 15 G15 hygiene)

**ADR:** [ADR-ECO-007](https://github.com/KooshaPari/phenotype-registry/blob/main/docs/adrs/ADR-ECO-007-gateway-merge-superset.md) — gateway merge superset boundaries  
**Lane:** `feat/wave15-g15-branch-prune`  
**Snapshot date:** 2026-06-18  
**Main HEAD:** `2e5bc97b44397730e2593d818e2be6d0697630cf`

## Policy

Remote branches are capped at **≤5** per Wave 15 ledger. After G15 cleanup the retained set is:

| Branch | Role |
|--------|------|
| `main` | Canonical integration branch |
| `dependabot/npm_and_yarn/agentapi-plusplus/chat/babel/plugin-transform-modules-systemjs-7.29.4` | Open deps PR #509 |
| `dependabot/npm_and_yarn/agentapi-plusplus/chat/fast-uri-3.1.2` | Open deps PR #508 |

`feat/wave15-g15-branch-prune` is the active hygiene lane (this PR) and is deleted after merge.

## Pre-merge context

- **#531** merged superset + `complete-sync` content into `main`.
- **#532** merged upstream sync policy docs.
- **#533** merged initial branch inventory (superseded by this CSV + prune).

## Deleted branches (Wave 15 G15)

| Class | Count | Rationale |
|-------|-------|-----------|
| `backup/*` | 2 | Safety snapshots; unique commits absorbed or obsolete post-#531 |
| `complete-sync` | 1 | Superset merged via #531 |
| `sync/upstream-v0.12.2`, `fix/pull-request-target` | 2 | Upstream sync lanes merged pre-audit |
| `feat/wave-h2-branch-superset`, `feat/wave-h1-funding-yml` | 2 | Wave H lanes merged (#531, #530) |
| `chore/*`, `ci/*`, `docs/*`, `wip/*` | 25 | Stale batch branches; no unique value post-superset |
| `dependabot/.../next-15.5.18` | 1 | Superseded by #514 on `main` |

Eight branches were already deleted on remote before this audit (`deleted-pre-audit` in CSV).

## Machine-readable inventory

See [`branch-inventory.csv`](branch-inventory.csv) for the full pre-prune snapshot with `ahead`, `behind`, `class`, and `action` columns.

## Verification

```bash
git fetch origin --prune
git branch -r | wc -l   # expect ≤5 including HEAD alias
```
