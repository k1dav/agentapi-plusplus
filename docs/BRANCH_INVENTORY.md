# Branch inventory — 2026-06-18

Wave 15 (G15) hygiene. Unique commits vs `origin/main` at audit time.

| Unique commits | Branch | Class | Action |
|----------------|--------|-------|--------|
| 22 | feat/wave-h2-branch-superset | upstream/docs | Merge PR #532 |
| 18 | complete-sync | upstream | Review — may supersede #531 |
| 18 | chore/pin-actions | chore | Batch merge or close |
| 18 | chore/worklog-seed-agentapi-plusplus | chore | Batch merge or close |
| 15 | backup/20260426-reconcile-0c5f958 | backup | Cherry-pick or delete |
| 14 | chore/* (multiple) | chore | Batch merge theme groups |
| 13 | ci/add-golangci-lint | ci | Merge if green |
| 10 | chore/audit-agentapi-plusplus-2026-06-08 | chore | Merge if green |
| 9 | chore/SD1-004-sota-2026-06-11 | chore | Merge if green |
| 8 | agentapi-plusplus/chore/sast-pin-governance-clean | chore | Merge if green |
| 8 | docs/agentapi-plusplus-sladge-* | docs | Fold into docs/ |
| 1 | backup/20260426-agentapi-local-main-4a3c145 | backup | **Delete** post-diff |
| 1 | sync/upstream-v0.12.2 | upstream | **Delete** post-#531 |
| 1 | fix/pull-request-target | fix | **Delete** if merged |
| 1 | dependabot/* (5) | deps | Merge PRs #508–514 |
| 2 | wip/2026-06-17-prepush-agentapi-plusplus-stash | wip | **Delete** after review |

**Target:** ≤5 remote branches (`main` + active integration lanes).

## Post-merge delete candidates

After PR merges, delete remote refs that have no unique commits or are fully absorbed:

```bash
git push origin --delete sync/upstream-v0.12.2 fix/pull-request-target \
  backup/20260426-agentapi-local-main-4a3c145 \
  wip/2026-06-17-prepush-agentapi-plusplus-stash
```
