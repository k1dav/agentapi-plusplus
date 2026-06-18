# Boundary — agentapi-plusplus

**Status:** Active (Wave H / G15)  
**Disposition:** DYNAMIC-KEEP → phenotype-gateway (`packages/agentapi`)  
**Registry row:** `gw-agentapi-pp` in [disposition-index.json](https://github.com/KooshaPari/phenotype-registry/blob/main/registry/disposition-index.json)  
**Charter:** [boundary-shaping.md](https://github.com/KooshaPari/phenotype-registry/blob/main/docs/rationalization/boundary-shaping.md)  
**ADR:** [ADR-ECO-007](https://github.com/KooshaPari/phenotype-registry/blob/main/docs/adrs/ADR-ECO-007-gateway-merge-superset.md)

---

## Role

Canonical **agent terminal control plane** — HTTP/SSE API over in-memory PTY for CLI coding agents. Fork of [coder/agentapi](https://github.com/coder/agentapi) with Phenotype harness extensions.

Root product spec: [`SPEC.md`](../SPEC.md) (or PR #536 `feat/agentapi-root-spec` until merged).

---

## Stack

| Tier | Language | Justification |
|------|----------|---------------|
| Core | Go | Upstream agentapi server; PTY + HTTP API |
| Edge | TypeScript (chat UI) | Debug/operator UI only — not fleet routing |

---

## Owns

| Path / concern | Notes |
|----------------|-------|
| `cmd/`, `lib/httpapi/`, `lib/termexec/`, `lib/screentracker/`, `lib/msgfmt/` | Agent terminal server |
| `docs/`, `openapi/` | Operator + API contract docs |
| Fork extensions in `PRD.md` | Harness subprocess control, Phenotype workspace hooks |

---

## Forbidden (out of scope)

| Concern | Canonical owner |
|---------|-----------------|
| Fleet orchestration, submodule pins | phenotype-gateway |
| LLM multi-provider routing | OmniRoute |
| CLI subscription / OAuth proxy | cliproxyapi-plusplus |
| Inference runtime / compose | bifrost, PhenoCompose |
| macOS menu-bar client | cliproxy++ (vibeproxy absorbed) |

---

## Upstream sync

See [`docs/UPSTREAM.md`](./UPSTREAM.md). Merge order: `sync/upstream-*` → `complete-sync` → tag before branch prune.

phenotype-gateway consumes via submodule `third_party/agentapi-plusplus`; pin policy in [phenotype-gateway UPSTREAM.md](https://github.com/KooshaPari/phenotype-gateway/blob/main/docs/UPSTREAM.md).

---

## verify

```bash
go build ./...
go test ./...
```

Branch hygiene: ≤5 remotes after Wave H2 (#531, #535). See [`BRANCH_PRUNE.md`](./BRANCH_PRUNE.md).

---

## FSM

| Field | Value |
|-------|-------|
| Wave | H |
| fsm | `done` |
| PR | agentapi-plusplus#535, #531 |
| relocated_date | 2026-06-18 |
