# AgentAPI++ Examples

Runnable quick-starts for every supported agent. Each one is the smallest
end-to-end command that brings the server up, sends a message, and reads
the reply. Assumes you have already installed the `agentapi` binary
(see the [install instructions](../README.md#install-binary) at the top of
the project README) and the agent CLI on your `$PATH`.

## TL;DR — pick your agent

```bash
PORT=3284
agentapi server --port=$PORT --type <agent> -- <agent-binary> [args...]
curl -s http://localhost:$PORT/status | jq
curl -s -X POST http://localhost:$PORT/message \
  -H 'Content-Type: application/json' \
  -d '{"type":"user","content":"hello"}' | jq
```

The `--type` flag is **required** for predictable message formatting; the
server uses it to detect prompts, tool calls, and "ready" markers. The
remaining columns are copy-pasteable.

## 11-agent matrix

| Agent (alias)        | `--type`     | Run command                                                                                              |
|----------------------|--------------|----------------------------------------------------------------------------------------------------------|
| Claude Code          | `claude`     | `agentapi server --port=3284 --type claude -- claude`                                                    |
| Goose                | `goose`      | `agentapi server --port=3284 --type goose -- goose session --with-builtin=developer`                     |
| Aider                | `aider`      | `agentapi server --port=3284 --type aider -- aider --model sonnet --no-auto-commits`                     |
| Codex                | `codex`      | `agentapi server --port=3284 --type codex -- codex --quiet`                                              |
| Gemini CLI           | `gemini`     | `agentapi server --port=3284 --type gemini -- gemini`                                                    |
| GitHub Copilot CLI   | `copilot`    | `agentapi server --port=3284 --type copilot -- copilot`                                                  |
| Sourcegraph Amp      | `amp`        | `agentapi server --port=3284 --type amp -- amp`                                                          |
| Augment Auggie       | `auggie`     | `agentapi server --port=3284 --type auggie -- auggie`                                                    |
| Cursor (cursor-agent)| `cursor`     | `agentapi server --port=3284 --type cursor -- cursor-agent`                                              |
| Amazon Q             | `q`/`amazonq`| `agentapi server --port=3284 --type q -- q chat`                                                         |
| Opencode             | `opencode`   | `agentapi server --port=3284 --type opencode -- opencode`                                                |
| _Anything else_      | `custom`     | `agentapi server --port=3284 --type custom -- <your-cli>` (best-effort, no per-agent message detection) |

## Sending a message from any language

The HTTP API is plain JSON, so anything that can speak HTTP can drive an
agent. Three canonical exchanges:

### 1. Post a user message

```bash
curl -s -X POST http://localhost:3284/message \
  -H 'Content-Type: application/json' \
  -d '{"type":"user","content":"summarise this repo in one sentence"}'
```

Response:

```json
{ "ok": true }
```

### 2. Read the conversation so far

```bash
curl -s http://localhost:3284/messages | jq '.messages[-3:]'
```

### 3. Subscribe to live events (SSE)

```bash
curl -N http://localhost:3284/events
```

Each event is a Server-Sent Event with a `message_update`,
`status_change`, or `agent_error` payload — the same shapes
`GET /messages` returns, so client code can reuse types.

## State persistence

Run a long-lived agent with a state file so restarts resume the
conversation instead of starting cold:

```bash
agentapi server --port=3284 --type claude \
  --state-file=./.agentapi-state.json \
  -- claude
```

On the next start, agentapi detects the file and replays the
conversation transcript into the new PTY session before unblocking
the HTTP API.

## Experimental ACP transport

Most agents are wrapped via PTY today. Some support ACP (Agent Client
Protocol) directly, which is generally lower-overhead and avoids
screen-parsing. ACP is opt-in:

```bash
agentapi server --port=3284 --experimental-acp \
  -- <agent-binary> [args...]
```

ACP mode does **not** support state persistence and is currently
labelled experimental.

## OpenAPI schema

```bash
agentapi server --print-openapi > openapi.json
```

The same JSON is served at `GET /openapi.json` on a running instance
and at `http://localhost:3284/docs` via huma's built-in Swagger UI.

## Embedding the chat UI

A static chat embed is served at `/chat` and `/chat/embed` (the root
path redirects there). If you build a custom UI, point your reverse
proxy at the same port and pass `--chat-base-path=/` to mount it at
the root.

## Production tips

- Set `--allowed-hosts` and `--allowed-origins` to lock down the
  Host/Origin headers the server will accept.
- Each request gets a fresh `X-Request-ID` (or the inbound one is
  honoured if the client sets it). The same value is attached to
  every log line for that request, so a single curl trace is fully
  correlatable to the server's stdout.
- `--pid-file` writes the agentapi PID at startup and removes it on
  clean shutdown, so supervisors can detect crashes.
