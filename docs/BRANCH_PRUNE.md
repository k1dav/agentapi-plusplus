# Branch prune ledger — agentapi-plusplus

Post **Wave H2** (#531): `main` is superset of `sync/upstream-v0.12.2` + `complete-sync`.

## Taxonomy

| Bucket | Branches | Action |
|--------|----------|--------|
| merged-superset | `complete-sync`, `sync/upstream-v0.12.2` | DELETE remote (0 ahead of main) |
| backup | `backup/20260426-reconcile-0c5f958` | DELETE if 0 ahead; else tag + delete |
| hygiene | `chore/*`, `ci/*`, `docs/*` | DELETE if 0 ahead (absorbed in main) |
| dependabot | `dependabot/npm_and_yarn/*` | DELETE (post-merge cleanup) |
| wip | `wip/2026-06-17-cleanup-agentapi-plusplus-dirty` | Review; delete if empty |

## Policy

- Never delete `main`
- Run `gh api repos/KooshaPari/agentapi-plusplus/compare/main...<branch>` before delete
- Delete only when `ahead_by == 0`

## Gate after prune

```bash
go build ./...
go test ./...
```

## Audit log (2026-06-18)

- Script: `scripts/branch-prune-audit.ps1`
- Result: **0 delete candidates** — remote has `main` only (post H2 / #535)
