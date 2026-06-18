# AgentAPI++ Specification

> **Wave 15 (G15)** — Platform agent terminal control plane. Branch hygiene and superset merge per [ADR-ECO-007](https://github.com/KooshaPari/phenotype-registry/blob/main/docs/adrs/ADR-ECO-007-gateway-merge-superset.md) in phenotype-registry. See [`docs/BRANCH_INVENTORY.md`](docs/BRANCH_INVENTORY.md).

## Repository Overview

AgentAPI++ provides API interfaces for agent operations.

## Architecture

```
agentapi-plusplus/
├── agent-core/         # Core agent logic
├── adapters/           # API implementations
├── ports/              # Trait definitions
├── api/                # API definitions
└── README.md
```

## Domain Model

### Bounded Contexts

| Context | Responsibility |
|---------|----------------|
| `agent` | Agent lifecycle |
| `session` | Conversation sessions |
| `tool` | Tool registration |
| `policy` | Security policies |

## xDD Practices

### TDD (Test-Driven Development)

```bash
cargo test -- --nocapture  # Fail first
cargo fix --allow-dirty  # Minimal impl
refactor
```

### BDD (Behavior-Driven Development)

Gherkin scenarios in `agent-core/features/`:

```gherkin
Feature: Tool Execution
  Scenario: Successful execution
    Given a registered tool "bash"
    When execute tool is called
    Then result contains stdout
```

### CQRS (Command Query Responsibility Segregation)

| Operation | Handler |
|-----------|----------|
| Command | `execute_tool` |
| Query | `list_tools` |

### Event Sourcing

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AgentEvent {
    ToolRegistered(ToolId),
    SessionStarted(SessionId),
    PolicyViolation { policy: PolicyId },
}
```

## SOLID Principles

| Principle | Status | Action |
|-----------|---------|---------|
| SRP | ✅ | Handlers have one responsibility |
| OCP | 🟡 | Extend via traits |
| LSP | 🟡 | Review trait bounds |
| ISP | 🟡 | Split large traits |
| DIP | ✅ | Depend on abstractions |

## Layered Architecture

```
┌──────────────────┐
│   API Layer       │  ← actix-web handlers
├──────────────────┤
│   Ports Layer    │  ← traits
├──────────────────┤
│   Application    │  ← use cases
├──────────────────┤
│     Domain       │  ← entities
├──────────────────┤
│  Infrastructure │  ← adapters
└──────────────────┘
```

## Quality Gates

```bash
cargo fmt --check
cargo clippy --all-targets
cargo test --all
cargo audit
cargo udeps
```

## Observability

- [x] Structured logging (tracing)
- [x] Metrics (metrics crate)
- [x] Health endpoints
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Correlation IDs

## Error Handling

```rust
pub type Result<T> = std::result::Result<T, AgentError>;

#[derive(Debug, thiserror::Error)]
pub enum AgentError {
    #[error("tool not found: {tool_id}")]
    ToolNotFound { tool_id: ToolId },
    #[error("session expired: {session_id}")]
    SessionExpired { session_id: SessionId },
}
```

## Testing Checklist

- [x] Unit tests with `#[test]`
- [x] Integration tests
- [ ] Property-based tests (proptest)
- [ ] Contract tests (API compatibility)
- [ ] Chaos engineering

## File Naming

| Type | Pattern |
|------|----------|
| Entities | `*_entity.rs` |
| Value Objects | `*_vo.rs` |
| Ports | `*_port.rs` |
| Commands | `*_cmd.rs` |
| Queries | `*_qry.rs` |
| Events | `*_event.rs` |
| Handlers | `*_handler.rs` |

## References

- [ ] Architecture Tests
- [ ] Mutation Testing
- [ ] Property-Based Testing
- [ ] ADR: Architecture decisions