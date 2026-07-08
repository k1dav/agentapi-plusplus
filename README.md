# AgentAPI++ (k1dav fork)

HTTP API for programmatically controlling CLI-based AI coding agents ([Claude Code](https://github.com/anthropics/claude-code), [Codex](https://github.com/openai/codex), [Goose](https://github.com/block/goose), [Aider](https://github.com/Aider-AI/aider), [Gemini CLI](https://github.com/google-gemini/gemini-cli), [GitHub Copilot](https://github.com/github/copilot-cli), [Amp](https://ampcode.com/), [Cursor CLI](https://cursor.com/en/cli), [Auggie](https://docs.augmentcode.com/cli/overview), [Opencode](https://opencode.ai/), Amazon Q) over an in-memory PTY.

```
HTTP Request → AgentAPI++ → Terminal Emulator → claude / codex / ...
```

Fork lineage: [coder/agentapi](https://github.com/coder/agentapi) → KooshaPari/agentapi-plusplus → this repo. Binaries are built by CI when a GitHub release is published and consumed by [k1dav-c/coder-templates](https://github.com/k1dav-c/coder-templates)' `ai-agents` module.

## Fork highlights

- **Structured timeline** — thinking / tool calls / tool results captured from the agent's own transcript files, exposed via `GET /timeline` and SSE `timeline_event` (see below)
- **`DELETE /messages` resets the agent session** — clears history + timeline and sends the agent's new-session command (claude: `/clear`, codex: `/new`)
- **Runtime MCP config** — `GET /mcp` / `PUT /mcp` manage the agent's MCP servers on the fly; `?restart=true` restarts the agent process in place so changes apply immediately
- **Chat UI process view** — inline collapsible tool-call cards and a filterable timeline side panel
- **API-key auth** on mutating routes (`--api-key` / `AGENTAPI_API_KEY`)
- Extra read endpoints: `/info`, `/health`, `/version`, `/ready`, `/messages/count`

## Quick Start

```bash
OS=$(uname -s | tr "[:upper:]" "[:lower:]")
ARCH=$(uname -m | sed "s/x86_64/amd64/;s/aarch64/arm64/")
curl -fsSL "https://github.com/k1dav/agentapi-plusplus/releases/latest/download/agentapi-${OS}-${ARCH}" -o agentapi
chmod +x agentapi

# Start with Claude Code (specify --type explicitly, otherwise message formatting may break)
./agentapi server --type claude -- claude
```

The server runs on port 3284. The chat UI is at http://localhost:3284/chat, the OpenAPI schema at http://localhost:3284/openapi.json (also checked in as [openapi.json](openapi.json)).

Build from source: `go build -o agentapi main.go` (chat UI assets are embedded separately — see [Development](#development)).

## Endpoints

| Method / Path | Description |
|---|---|
| GET `/messages` | Conversation history |
| POST `/message` | Send a message (`user` or `raw` keystrokes); auth-gated |
| DELETE `/messages` | Clear history **and timeline**, and reset the agent's session (`?new_session=false` to skip); auth-gated |
| GET `/timeline` | Structured process events; `?kind=` and `?since_id=` filters |
| GET `/mcp` | Currently configured MCP servers and the config file path |
| PUT `/mcp` | Replace the MCP server set; `?restart=true` restarts the agent to apply immediately; auth-gated |
| GET `/status` | Agent status: `stable` or `running` |
| GET `/events` | SSE stream: `message_update`, `status_change`, `timeline_event`, `agent_error` |
| POST `/upload` | Upload files; auth-gated |
| GET `/info` | Version, agent type, feature flags (`features.timeline`) |
| GET `/health`, `/version`, `/ready`, `/messages/count` | Auxiliary read endpoints |

### Allowed hosts

By default only requests with a `localhost` host header are accepted. Override with `--allowed-hosts` / `AGENTAPI_ALLOWED_HOSTS` (hostnames only, no ports; `*` allows all).

## Structured timeline (thinking & tool calls)

The PTY screen only shows the agent's final answer text. For the *process* — thinking, tool calls, tool results — AgentAPI++ tails the transcript files the agent itself writes to disk (read-only sidecar; the TUI is untouched):

- Claude Code: `~/.claude/projects/<sanitized-cwd>/<session>.jsonl`
- Codex: `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`

Events are normalized to a single shape and assigned monotonic ids:

```json
{
  "id": 2,
  "kind": "tool_call",            // thinking | text | tool_call | tool_result | system
  "role": "assistant",
  "time": "2026-07-08T09:31:59Z",
  "session_id": "d1341cee-...",
  "tool_name": "Bash",
  "tool_input": {"command": "ls"},
  "tool_use_id": "toolu_01..."    // joins a tool_call with its tool_result
}
```

```bash
curl 'localhost:3284/timeline'                  # full process
curl 'localhost:3284/timeline?kind=tool_call'   # only tool calls
curl 'localhost:3284/timeline?since_id=42'      # incremental polling
curl -N localhost:3284/events                   # live: timeline_event frames
```

Behavior notes:

- Supported for `claude` and `codex` on the PTY transport; on by default, disable with `--timeline=false`, override the search directory with `--timeline-dir`
- Session switches (claude `/clear`, codex new thread) are followed automatically and marked with a `system` "session switched" event; files from previous runs are never re-ingested (mtime filter)
- The last 10,000 events are kept in memory; late SSE subscribers get the most recent 500 replayed (full history via `GET /timeline`); ids keep increasing across `DELETE /messages`, so `since_id` polling never misses events
- Thinking events appear only when the backend writes plaintext reasoning to the transcript. Some deployments encrypt/redact it at the source (Claude signature-only thinking blocks, Codex `encrypted_content`); tool calls and results are always plaintext and unaffected

## Runtime MCP configuration

Agents load MCP config at process startup. `PUT /mcp` writes the agent's config file — claude: project `.mcp.json` in the working directory, codex: the `[mcp_servers.*]` tables in `~/.codex/config.toml` (other content, including comments, is preserved verbatim) — and with `?restart=true` restarts the agent process in place so the change takes effect immediately. AgentAPI keeps serving across the restart; the agent's conversation context is reset (a new session starts).

```bash
curl 'localhost:3284/mcp'    # current servers + config path

# Full replace: servers not listed are removed. Restart to apply now.
curl -X PUT 'localhost:3284/mcp?restart=true' -H 'Content-Type: application/json' -d '{
  "servers": {
    "memory": {"command": "npx", "args": ["-y", "@modelcontextprotocol/server-memory"]},
    "remote": {"type": "http", "url": "https://mcp.example.com", "headers": {"X-Key": "..."}}
  }
}'

curl -X PUT localhost:3284/mcp -H 'Content-Type: application/json' \
  -d '{"servers":{}}'         # clear all MCP servers (applies next session)
```

Server config objects are passed through to the agent verbatim — use whatever fields the agent supports. Supported for `claude` and `codex` on the PTY transport (`features.mcp` in `GET /info`); MCP tool invocations show up in the timeline like any other tool call.

## Supported agents

`claude`, `codex`, `goose`, `aider`, `gemini`, `copilot`, `amp`, `cursor`, `auggie`, `amazonq`, `opencode`, `custom` — pass via `--type`. Message formatting and readiness detection are per-agent (`lib/msgfmt`); the structured timeline currently covers `claude` and `codex`.

## API examples

```bash
# Send a message
curl -X POST http://localhost:3284/message \
  -H "Content-Type: application/json" \
  -d '{"type":"user","content":"list the files in this directory"}'

# Watch the process live
curl -N http://localhost:3284/events

# Inspect what tools the agent used
curl -s 'http://localhost:3284/timeline?kind=tool_call' | jq '.events[] | {tool_name, tool_input}'

# Start over (clears history + timeline, agent gets /clear or /new)
curl -X DELETE http://localhost:3284/messages
```

## Chat UI

Next.js app served at `/chat`:

- Tool invocations render as collapsible cards in the conversation flow (input summary, running/done state, full input JSON and result on expand)
- A timeline side panel (top-right toggle) lists every process event with kind filters (Tools / Thinking / Text / System)
- Dark mode, mobile drawer layout

## Architecture

| Component | Description |
|-----------|-------------|
| `cmd/` | CLI commands (`server`, `attach`) |
| `lib/httpapi/` | HTTP server, routes, SSE event emitter |
| `lib/termexec/` | PTY process execution |
| `lib/screentracker/` | Screen snapshot → conversation messages |
| `lib/msgfmt/` | Agent-specific message formatting |
| `lib/transcript/` | Transcript tailing → structured timeline (discovery, tailer, per-agent parsers, watcher) |
| `chat/` | Next.js web UI |

## Configuration

```bash
export AGENTAPI_PORT=3284
export AGENTAPI_ALLOWED_HOSTS="localhost,127.0.0.1"
export AGENTAPI_ALLOWED_ORIGINS="http://localhost:3284"
export AGENTAPI_API_KEY="..."        # enables bearer auth on mutating routes
export AGENTAPI_TIMELINE=true        # structured timeline capture (default true)
```

Every `--flag` has an `AGENTAPI_<FLAG>` env equivalent (dashes → underscores). See `agentapi server --help` for the full list.

## Development

```bash
go build -o agentapi main.go   # server only; chat UI 404s with a hint
go test ./...

# Full build with embedded chat UI (what release CI does)
cd chat && bun install --frozen-lockfile && \
  NEXT_PUBLIC_BASE_PATH="/magic-base-path-placeholder" bun run build && cd .. && \
  rm -rf lib/httpapi/chat && mkdir -p lib/httpapi/chat && touch lib/httpapi/chat/marker && \
  cp -r chat/out/. lib/httpapi/chat/ && go build -o agentapi main.go
```

Releases: publish a GitHub release; `.github/workflows/release.yml` builds the chat UI, embeds it, and uploads binaries for linux/darwin × amd64/arm64 as release assets.

If you change the HTTP API, regenerate the schema snapshot: `go run main.go server --print-openapi dummy > openapi.json`.

## License

MIT — see [LICENSE](LICENSE).
