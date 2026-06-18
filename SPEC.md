# AgentAPI++ — Root Specification

**Status:** Canonical spec for this repository (2026-06-18)  
**Governance:** [ADR-ECO-007](https://github.com/KooshaPari/phenotype-registry/blob/main/docs/adrs/ADR-ECO-007-gateway-merge-superset.md) (phenotype-registry)  
**Upstream:** [coder/agentapi](https://github.com/coder/agentapi)  
**Fleet role:** Agent terminal control plane — HTTP API over in-memory PTY for CLI coding agents

---

## Purpose

AgentAPI++ is the Phenotype-org fork of [coder/agentapi](https://github.com/coder/agentapi). It exposes a stable HTTP + SSE interface so orchestrators, CI, MCP servers, and other automation can programmatically drive CLI-based AI coding agents (Claude Code, Cursor, Aider, Codex, Goose, Gemini CLI, GitHub Copilot, Sourcegraph Amp, Amazon Q, Auggie, Opencode, and others) without manual terminal interaction.

```txt
HTTP client → AgentAPI++ (Go server, :3284) → PTY → agent CLI → structured messages/events
```

This repository owns:

- The **agent terminal server** (`cmd/`, `lib/httpapi/`, `lib/termexec/`, `lib/screentracker/`, `lib/msgfmt/`)
- **Agent-specific message formatting** and session/status semantics
- **Fork extensions** documented in [`PRD.md`](PRD.md) (harness subprocess control, Phenotype workspace hooks, routing helpers where they serve agent control — not a separate gateway product)
- **Operational docs** for sync, branch hygiene, and integration with the Phenotype fleet

Detailed product requirements and epics live in [`PRD.md`](PRD.md). API tutorials and deep dives live under [`docs/`](docs/).

---

## Boundaries

| Layer | Owner | This repo |
|-------|-------|-----------|
| Agent terminal HTTP API | **agentapi-plusplus** (this repo) | ✅ Canonical |
| Fleet orchestration, submodule pins, cross-repo spec | **phenotype-gateway** | Consumed as `third_party/agentapi-plusplus` |
| TypeScript LLM router / multi-provider routing | **OmniRoute** (`route` peer) | ❌ Out of scope |
| CLI subscription / OAuth proxy for LLM providers | **cliproxyapi-plusplus** | ❌ Out of scope (optional integration peer) |
| Inference engine / compose runtime | **bifrost**, **PhenoCompose** | ❌ Out of scope |
| Upstream baseline | **coder/agentapi** | Tracked via sync lanes |

**In scope:** controlling agent CLIs, parsing their terminal output, REST/SSE contract on port 3284, OpenAPI schema, chat debug UI, fork-specific agent coverage beyond upstream.

**Out of scope for this repo:** owning fleet-wide routing policy, provider OAuth, model catalog, or absorbing OmniRoute or cliproxy++ functionality. Those remain separate canonical repositories per ADR-ECO-007.

phenotype-gateway may wrap or re-export this code under `packages/agentapi`; the **source of truth for agent-terminal behavior remains this fork**, not the gateway monorepo.

---

## Relationship to phenotype-gateway

[phenotype-gateway](https://github.com/KooshaPari/phenotype-gateway) pins this repository as a **git submodule**, not as a merged subdirectory:

| Field | Value |
|-------|-------|
| Submodule path | `third_party/agentapi-plusplus` |
| Pin policy | Recorded in [phenotype-gateway `docs/UPSTREAM.md`](https://github.com/KooshaPari/phenotype-gateway/blob/main/docs/UPSTREAM.md) |
| Example pin (Wave H6, 2026-06-18) | `78987040ad2112a9142b9407cfd468c984ae253a` (post H2 branch superset, PR #531) |

**Pin workflow**

1. Merge and gate changes on **this repo's `main`** first.
2. After CI passes, bump the submodule SHA in phenotype-gateway via a dedicated gateway PR.
3. Do **not** use `git submodule update --remote` on gateway until interim canonical `main` gates pass (gateway UPSTREAM policy).

Gateway owns orchestration spec and `third_party/` pins; it does **not** subsume agentapi++, OmniRoute, or cliproxy++.

When this SPEC or sync policy changes, update [`docs/UPSTREAM.md`](docs/UPSTREAM.md) in this repo and coordinate a gateway submodule bump if the pin contract changes.

---

## Upstream sync policy

Fork of [coder/agentapi](https://github.com/coder/agentapi). See also [`docs/UPSTREAM.md`](docs/UPSTREAM.md) and [`docs/BRANCH_INVENTORY.md`](docs/BRANCH_INVENTORY.md).

### Sync lanes (order matters)

| Lane | Branch pattern | Conflict resolution | When |
|------|----------------|---------------------|------|
| **Upstream import** | `sync/upstream-*` | Upstream wins on API/surface conflicts | New upstream release tags |
| **Local superset** | `complete-sync`, feature waves | Local fork wins on superset additions | After upstream lane merged |
| **Hygiene** | `feat/wave*-g*-branch-prune` | N/A — inventory + delete stale branches | Post-merge cleanup (≤5 remote branches) |

### Rules

1. **Track upstream releases** — import via `sync/upstream-*`; preserve fork-only agent formatters and endpoints.
2. **Superset merge to `main`** — local enhancements (multi-agent support, harness, Phenotype hooks) merge only after upstream lane is integrated; see Wave H2 / #531 precedent.
3. **Backup before prune** — tag or retain `backup/*` snapshots before deleting merged sync branches.
4. **API compatibility** — breaking HTTP contract changes require PRD + CHANGELOG entry and coordinated gateway pin bump.
5. **No silent divergence** — long-lived divergent branches without inventory class/action are forbidden post–Wave 15 (G15).

### Verification

```bash
git fetch origin --prune
go test ./...
# After upstream merge: smoke server on :3284, OpenAPI at /openapi.json
```

---

## Non-goals

AgentAPI++ explicitly does **not**:

| Non-goal | Canonical owner instead |
|----------|-------------------------|
| **OmniRoute** — TypeScript LLM routing, provider fan-out, desktop route plane | [OmniRoute](https://github.com/KooshaPari/OmniRoute) (`route` peer; never archive, never absorb into agentapi++) |
| **cliproxy++** — CLI OAuth proxy, subscription provider bridge | [cliproxyapi-plusplus](https://github.com/KooshaPari/cliproxyapi-plusplus) (Option B peer; gateway `third_party/` pin) |
| **Fleet orchestration monorepo** — cross-repo pins, gateway spec SSOT | phenotype-gateway |
| **Model provider implementation** | Delegated to agent CLIs and/or cliproxy++ when configured |
| **Persistent conversation database** | In-memory session state only (see PRD non-goals) |
| **End-user product UI** | Debug `/chat` only; not a standalone IDE or router product |

Optional **integration** with cliproxy++ (e.g. `--llm-provider http://localhost:8317`) is supported; **ownership** of proxy behavior stays in cliproxy++.

AgentAPI++ must not grow into a replacement for OmniRoute or cliproxy++. Routing policy that belongs at the fleet `route` or `cli_proxy` layer stays in those repos.

---

## Related documents

| Document | Role |
|----------|------|
| [`README.md`](README.md) | Quick start, endpoints, supported agents |
| [`PRD.md`](PRD.md) | Product requirements, epics, fork extensions |
| [`docs/UPSTREAM.md`](docs/UPSTREAM.md) | Short upstream fork policy |
| [`docs/BRANCH_INVENTORY.md`](docs/BRANCH_INVENTORY.md) | Wave 15 branch hygiene ledger |
| [`COMPARISON.md`](COMPARISON.md) | Positioning vs coder/agentapi and peers |
| [phenotype-gateway `docs/UPSTREAM.md`](https://github.com/KooshaPari/phenotype-gateway/blob/main/docs/UPSTREAM.md) | Fleet submodule pins including this repo |

---

*Last updated: 2026-06-18 — root spec lane `feat/agentapi-root-spec`*
