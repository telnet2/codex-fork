# Feasibility study: `codex-agent-sdk-python` and a unified agent SDK

**Status:** study, no code yet
**Date:** 2026-10-31
**Sources examined:**

- Claude Agent SDK for Python: `anthropics/claude-agent-sdk-python` at `1ea8cb7` (v0.2.175, bundled CLI 2.1.314). MIT licensed.
- Codex: this fork at `b931e83e`, which is even with `openai/codex` `main` as of 2026-10-30.
- Existing Codex SDKs in this repo: `sdk/python` (`openai-codex`), `sdk/python-runtime` (`openai-codex-cli-bin`) and `sdk/typescript` (`@openai/codex-sdk`).

Paths prefixed `claude:` are relative to `claude-agent-sdk-python/src/claude_agent_sdk/`. All other paths are relative to this repository.

## 1. Verdict

**It is feasible.** `codex app-server` is a long-lived JSON-RPC process. It supports multi-turn threads, streams fine-grained events, and sends server-to-client requests for approvals, questions, elicitations and client-defined tools. That covers most of what the Claude Agent SDK gets from the `claude` CLI's stream-json control protocol.

Coverage by area:

| Area | Rating | Notes |
|---|---|---|
| Binary lifecycle, transport, streaming, sessions, interrupt | Full | The Codex side is often richer: server-side thread listing, fork, steer, and independently addressable subagent threads. |
| Answering approvals and questions programmatically | Full, with caveats | The SDK must answer *every* server request, because the server has no timeout. Questions (`item/tool/requestUserInput`) are experimental and only offered in Plan mode unless a feature flag is set. |
| Client-defined tools | Full, through an experimental API | Uses `thread/start.dynamicTools` and `item/tool/call`. |
| Background task management | Mostly | Through experimental `thread/backgroundTerminals/*`, plus `command/exec` and `process/*` for client-launched work. |
| Subagents | Observation and control: full. Definition: partial. | Roles are config-file based, so the SDK must write role files. |
| In-process hook callbacks | Partial, needs a bridge | Codex hooks are configured, not registered over the protocol. The SDK must host an MCP server and register `mcp_tool` hooks that point at it. |
| A permission callback that sees every tool call | Partial | Codex only asks for escalations. Seeing every call requires a `PreToolUse` hook through the same bridge. |

**Recommendation.** Do not write a Codex client from scratch. Build the unified SDK as a backend-neutral core with two adapters:

- **Claude adapter:** reuse or port the Claude SDK's transport and control layer.
- **Codex adapter:** reuse the generated pydantic models and binary packaging from `sdk/python`, with a new native-asyncio transport and request router.

Section 7 describes the unified design, and section 9 lists the spikes that must pass before committing to it.

## 2. How each binary is driven

| | Claude (`claude` CLI) | Codex (`codex app-server`) |
|---|---|---|
| Spawn | `claude --output-format stream-json --input-format stream-json --verbose …flags` (`claude:_internal/transport/subprocess_cli.py:570-795`) | `codex [-c key=value]… app-server --listen stdio://` (`sdk/python/src/openai_codex/client.py:253-256`; flags at `codex-rs/cli/src/main.rs:590-601`) |
| Framing | NDJSON on stdin and stdout | JSONL, JSON-RPC 2.0 without the `"jsonrpc"` field (`codex-rs/app-server-protocol/src/rpc.rs:1-2`; `codex-rs/app-server-transport/src/transport/stdio.rs:73,141`) |
| Other transports | None | `unix://`, `ws://IP:PORT` with token auth, `/readyz` and `/healthz` (`codex-rs/app-server-transport/src/transport/mod.rs:123-164`) |
| Handshake | `control_request{subtype:"initialize", hooks, agents, …}` (`claude:_internal/query.py:309-363`) | `initialize{clientInfo, capabilities{experimentalApi, optOutNotificationMethods, …}}`, then the `initialized` notification (`codex-rs/app-server-protocol/src/protocol/v1.rs:29-75`) |
| Configuration | CLI flags; agents and hooks go in `initialize` | `-c` overrides at spawn; per-thread fields on `thread/start`; per-turn fields on `turn/start` |
| Clients per process | One | One over stdio; several over websocket or unix socket |
| Shutdown | Close stdin, wait 5s, SIGTERM, wait 5s, SIGKILL (`claude:…/subprocess_cli.py:1024-1046`) | EOF or SIGTERM starts a ~30s drain, with a 45s hard-exit watchdog (`codex-rs/app-server-transport/src/transport/stdio.rs:41,168-180`) |
| Version | `claude -v`, warn only (`claude:…/subprocess_cli.py:1164-1224`) | No version field in the `initialize` response; parse `userAgent` or run `codex --version` |
| Bundled binary | `_bundled/claude` in the wheel, then `PATH`, then known directories | `openai-codex-cli-bin` wheel (`sdk/python-runtime`), or an explicit path |

## 3. Capability comparison

Legend:

- ✅ native
- 🟡 possible with a workaround or an experimental API
- 🔶 needs an SDK-side bridge
- ❌ not available

### 3.1 Session and turn control

| Capability | Claude Agent SDK | Codex app-server | Unified-SDK note |
|---|---|---|---|
| Long-lived multi-turn session | ✅ `ClaudeSDKClient` | ✅ thread plus repeated `turn/start` | Common `Session` object |
| One-shot query | ✅ `query()` | ✅ `thread/start` plus `turn/start`, or `ephemeral: true` | Common `run()` helper |
| Interrupt | ✅ `interrupt` control request | ✅ `turn/interrupt{threadId, turnId}` | Common |
| Steer a running turn | 🟡 stream additional user messages | ✅ `turn/steer{expectedTurnId}` | Expose `session.steer()`; on Claude, send a queued user message |
| Change model mid-session | ✅ `set_model` | ✅ per-turn `model` on `turn/start`; 🟡 `turn/settings/update` (X) | Common; applied at the next turn on Codex |
| Change permission mode mid-session | ✅ `set_permission_mode` | ✅ per-turn `approvalPolicy` and `sandboxPolicy` | Map to a common `PermissionProfile` |
| Resume | ✅ `--resume` | ✅ `thread/resume{threadId}`; 🟡 by `path` (X) | Common, keyed by session id |
| Fork | ✅ `--fork-session`, `resume_session_at` | ✅ `thread/fork` | Common |
| Rewind | ✅ `rewind_files`, which restores *files* | 🟡 `thread/revert`, which truncates *history* only and only on paginated threads (X) | Distinct capabilities: `rewind_files` and `revert_history` |
| List sessions | ✅ client-side JSONL scan (`claude:_internal/sessions.py:680`) | ✅ server-side `thread/list` with filters | Backend-provided; must pass `sourceKinds` on Codex or SDK threads may be hidden |
| Read transcript | ✅ client-side (`get_session_messages`) | ✅ `thread/read`, `thread/turns/list`, `thread/items/list` | Backend-provided |
| External session store | ✅ `SessionStore` with mirroring and a conformance harness | ❌ local rollout files only | Claude-only capability |
| Structured output | ✅ `output_format` | ✅ `turn/start.outputSchema` | Common |
| Cost reporting | ✅ `total_cost_usd` | ❌ token usage only (`thread/tokenUsage/updated`) | `usage` is common, `cost_usd` is optional |
| Context usage | ✅ `get_context_usage` | ✅ token usage notifications | Common, best effort |
| Compaction | Automatic, plus a `PreCompact` hook | ✅ `thread/compact/start`, plus hooks | Common `compact()`; no-op or error on Claude |
| Several clients on one process | ❌ | ✅ websocket or unix socket | Codex-only capability |

### 3.2 Streamed events

| Capability | Claude Agent SDK | Codex app-server |
|---|---|---|
| Text deltas | ✅ `StreamEvent` (raw Anthropic event; needs `include_partial_messages`) | ✅ `item/agentMessage/delta` |
| Reasoning deltas | ✅ raw thinking deltas inside `StreamEvent` | ✅ `item/reasoning/textDelta`, `summaryTextDelta` |
| Tool lifecycle | Tool use and tool result *content blocks* inside messages | First-class *items* with `item/started` and `item/completed`: `commandExecution`, `fileChange`, `mcpToolCall`, `dynamicToolCall`, `webSearch`, `collabAgentToolCall`, … |
| Command output streaming | ❌ the result arrives whole | ✅ `item/commandExecution/outputDelta` |
| Diff and plan updates | ❌ | ✅ `turn/diff/updated`, `turn/plan/updated` |
| End-of-turn result | ✅ `ResultMessage` | ✅ `turn/completed` |
| Hook runs visible | ✅ `HookEventMessage` (needs `include_hook_events`) | ✅ `hook/started`, `hook/completed` |
| Typed versus raw | Typed dataclasses; many system frames are raw `SystemMessage.data` | Fully typed by the generated JSON Schema (`codex-rs/app-server-protocol/schema/json`) |

The two event models differ in shape. Claude is *message-centric*: an assistant message contains content blocks. Codex is *item-centric*: each tool call, message or reasoning step is its own item with a lifecycle. Section 7.3 explains how to unify them without losing information.

### 3.3 Permissions, approvals and questions

| Capability | Claude Agent SDK | Codex app-server | Unified-SDK note |
|---|---|---|---|
| Callback on every tool call | ✅ `can_use_tool` | 🔶 only through a `PreToolUse` hook bridged to the SDK (section 5) | `on_tool_use` is native on Claude and bridged on Codex |
| Callback on escalation or approval prompts | ✅ (same callback) | ✅ `item/commandExecution/requestApproval`, `item/fileChange/requestApproval`, `item/permissions/requestApproval` | Common `on_approval` |
| Allow with modified input | ✅ `PermissionResultAllow(updated_input)` | ❌ in approvals; 🔶 through the `PreToolUse` hook, which can rewrite input | Capability flag |
| Remember for the session | ✅ `updated_permissions` (rules) | ✅ `acceptForSession`; execpolicy and network amendments | Common `scope="session"` |
| Deny and stop the turn | ✅ `deny(interrupt=True)` | ✅ `cancel` (`decline` continues the turn) | Common: `deny(interrupt=…)` |
| Answer the model's question | 🟡 `AskUserQuestion` arrives as a `can_use_tool` call, answered by `updated_input`; the SDK has no dedicated type | 🟡 `item/tool/requestUserInput` (X): structured questions and options, answered by `{answers}`; offered in Plan mode, or in Default mode with `default_mode_request_user_input` | Common `on_question(Question) -> Answers` |
| MCP elicitation | Hook output only | ✅ `mcpServer/elicitation/request` | Common `on_elicitation`; Codex only |
| Server-side auto reviewer | Permission modes `auto` and `dontAsk` | ✅ `approvalsReviewer: auto_review` (guardian), with unstable notifications | Option; not the default for API-driven runs |

### 3.4 Hooks

| | Claude Agent SDK | Codex |
|---|---|---|
| Registration | In-process Python callbacks sent by ID in `initialize`; each firing comes back as a `hook_callback` control request (`claude:_internal/query.py:319-344,640-654`) | Configuration only: `hooks.json`, a `[hooks]` table in config.toml, or plugins (`codex-rs/hooks/src/engine/discovery.rs:390-438`). Handler types are `command` and `mcp_tool`; `prompt` and `agent` parse but are not run (`codex-rs/protocol/src/hooks.rs:150-188`) |
| Events | PreToolUse, PostToolUse, PostToolUseFailure, UserPromptSubmit, Stop, SubagentStart, SubagentStop, PreCompact, Notification, PermissionRequest (`claude:types.py:284-296`) | PreModelRequest, PostModelRequest, PreToolUse, PermissionRequest, PostToolUse, PreCompact, PostCompact, SessionStart, SessionEnd, UserPromptSubmit, SubagentStart, SubagentStop, Stop, Interrupt (`codex-rs/hooks/src/lib.rs:29-44`) |
| Decision power | Block, allow, rewrite input, rewrite output, add context | PreToolUse can block or rewrite input; PermissionRequest can allow or deny |
| Trust | n/a | Hooks run only if managed or trusted, or with `bypass_hook_trust` |
| SDK feasibility | ✅ | 🔶 bridge (section 5) |

The overlapping events are PreToolUse, PostToolUse, UserPromptSubmit, Stop, SubagentStart, SubagentStop, PreCompact and PermissionRequest. These are the unified hook events. The rest are backend-specific.

### 3.5 Tools, subagents and background tasks

| Capability | Claude Agent SDK | Codex app-server | Unified-SDK note |
|---|---|---|---|
| In-process custom tools | ✅ `@tool` plus `create_sdk_mcp_server`, bridged over the control channel | 🟡 `dynamicTools` (X) on `thread/start`; calls arrive as `item/tool/call` and are answered with `{contentItems, success}` | Common `@tool`. Claude backend: SDK MCP server. Codex backend: dynamic tools. |
| External MCP servers | ✅ `mcp_servers` | ✅ config `mcp_servers` (via `-c`) | Common |
| MCP management | ✅ status, reconnect, toggle | ✅ `mcpServerStatus/list`, `config/mcpServer/reload`, OAuth login | Common subset: status and reload |
| Define subagents in code | ✅ `agents={name: AgentDefinition}` | 🟡 roles are `{description, config_file, nickname_candidates}` (`codex-rs/config/src/config_toml.rs:803-814`); the SDK must write a role config file and pass `-c agents.<name>.config_file=…` | Common `AgentDefinition`; the Codex adapter materializes files |
| Tell subagent output apart | ✅ `parent_tool_use_id` on messages | ✅ the child is a separate thread with `parentThreadId`, `agentRole` and `agent_path`; connections are auto-subscribed | Common `agent` field on every event (section 7.3) |
| Drive a subagent directly | ❌ (stop only, through `stop_task`) | ✅ `turn/start` or `turn/interrupt` on the child thread id | Codex-only capability |
| Background task events | ✅ `task_started`, `task_progress`, `task_notification`, `task_updated` | 🟡 background terminals appear as `commandExecution` items (`source: unifiedExecStartup`) plus output deltas | Common `BackgroundTask*` events |
| List and stop background tasks | 🟡 `stop_task(id)`; no listing (tasks are tracked from events) | 🟡 `thread/backgroundTerminals/list`, `terminate` and `clean` (X) | Common `tasks.list()` and `tasks.stop(id)`; on Claude, list from events |
| SDK-launched processes | ❌ | ✅ `command/exec` (sandboxed, streamed); 🟡 `process/*` (X) | Codex-only capability |

## 4. Unattended mode with sandbox and automatic approval

**Requirement:** the binary runs unattended inside a sandbox with automatic approval, and any approval request or question must be answerable by the API.

### 4.1 Codex: recommended settings

```text
sandbox                 = workspace-write   (networkAccess as required)
approvalPolicy          = on-request        (not "never"; see below)
approvalsReviewer       = user              (the SDK is the "user")
capabilities.experimentalApi = true         (for requestUserInput, dynamicTools, backgroundTerminals)
features.default_mode_request_user_input = true  (if questions are wanted outside Plan mode)
```

Do not use `approvalPolicy = never` if the API must see requests. Under `never`:

- escalations are rejected outright (`codex-rs/core/src/exec_policy.rs:220-226`);
- `request_permissions` gets an empty grant automatically;
- MCP elicitations are declined.

The SDK never sees any of these, so `never` means "deny everything outside the sandbox, silently". With `on-request` plus reviewer `user`, every escalation reaches the SDK, and the SDK's automatic responder decides.

The approval decision order is (`codex-rs/core/src/tools/approvals.rs:594-618`):

1. `PermissionRequest` hooks
2. the guardian, if the reviewer is `auto_review`
3. the client

So an SDK policy can also be enforced by a hook.

### 4.2 The SDK must always answer

The app-server does not time out server requests. A request the SDK never answers stalls the turn indefinitely, and an error reply is mapped to *deny* (`codex-rs/app-server/src/bespoke_event_handling.rs:2015-2043`). The existing `sdk/python` returns `{}` for every request type it doesn't recognise (`sdk/python/src/openai_codex/client.py:833-839`). That produces a denial that may not be intended.

The unified SDK therefore needs an **`AutoResponder`** with a guaranteed decision for every request kind:

| Request | Default decision in unattended mode | Overridable callback |
|---|---|---|
| Command approval | `accept` if the policy allows the command, otherwise `decline` | `on_approval` |
| File-change approval | `accept` within the writable roots, otherwise `decline` | `on_approval` |
| Permissions escalation | Grant the requested profile at `scope: turn` if the policy allows it, otherwise an empty grant | `on_approval` |
| Question (`requestUserInput`) | Pick the first or recommended option; free-text questions get "Use your best judgment and continue." | `on_question` |
| MCP elicitation | `decline` | `on_elicitation` |
| Dynamic tool call | Dispatch to the registered `@tool`; unknown tool returns `success: false` | `@tool` |
| Unknown method | A JSON-RPC error with a clear message | none |

Each callback runs under an SDK-side timeout, `request_timeout`, with a fallback decision when it expires. Callbacks run concurrently, never on the reader task.

### 4.3 Claude: equivalent settings

- `permission_mode="default"` with `can_use_tool` set. This forces `--permission-prompt-tool stdio`, so every tool call reaches the SDK.
- Sandbox through `ClaudeAgentOptions.sandbox`.
- `bypassPermissions` is the counterpart of Codex `never`: the callback is never invoked, and the SDK warns about this (`claude:types.py:2190-2270`).

Under both backends, the same `AutoResponder` object implements the policy.

## 5. Hook bridge for Codex (the main technical risk)

Codex cannot call back into the SDK process for hooks, but its hook engine has an `mcp_tool` handler type (`codex-rs/hooks/src/engine/dispatcher.rs:224-238`), and there are tests for it (`codex-rs/core/tests/suite/hooks_mcp.rs`). The bridge works like this:

1. At connect time, the SDK starts a local MCP server: streamable HTTP on `127.0.0.1` with a random port and a bearer token. It exposes one tool, `sdk_hook`.
2. The SDK registers the server and the hooks with `-c` overrides at spawn, or through the `thread/start.config` map:
   - `mcp_servers.__sdk.url=…`
   - `hooks.PreToolUse=[{matcher, hooks:[{type:"mcp_tool", server:"__sdk", tool:"sdk_hook", input:{callback_id:"hook_3"}}]}]`
   - `bypass_hook_trust=true`, because the hooks are SDK-generated.
3. When a hook fires, Codex calls `sdk_hook` with the event input. The SDK looks up `callback_id`, runs the user's Python callback, and returns the hook output JSON (`codex-rs/hooks/schema/generated/*.output.schema.json`).

This mirrors Claude's ID-based registration (`hookCallbackIds`), with MCP in place of the control channel. The same server can also carry a `PreToolUse` catch-all that implements `on_tool_use` for every tool call (section 3.3).

**Open questions for the spike:**

- latency per tool call;
- whether `-c` can express hook arrays, or whether the SDK must write a temporary `hooks.json` into a temporary config layer;
- whether `bypass_hook_trust` is acceptable to the user;
- whether the hook input is fully delivered to the MCP tool.

## 6. Binary lifecycle for Codex

| Step | Design |
|---|---|
| Resolve the binary | `options.cli_path`, then `openai-codex-cli-bin` (`codex_cli_bin.bundled_codex_path()`, `sdk/python-runtime/src/codex_cli_bin/__init__.py`), then `shutil.which("codex")`, then `CLINotFoundError` |
| Check the version | Parse `userAgent` from the `initialize` response, and warn below a known minimum. The existing SDK enforces ≥ 0.151.0 (`sdk/python/src/openai_codex/_runtime_requirements.py:13`). |
| Spawn | `codex -c k=v… app-server --listen stdio://` with `CODEX_HOME` (optionally an isolated temp home), `OPENAI_API_KEY` and `RUST_LOG`. The app-server ignores `CODEX_API_KEY` (`codex-rs/app-server/src/config_manager.rs:425`). |
| Handshake | `initialize` with `experimentalApi: true` and `optOutNotificationMethods` derived from what the user subscribed to; then `initialized` |
| Run | One asyncio reader task demultiplexes responses (by id), notifications (by `threadId` and `turnId`) and server requests (to the router) |
| Health | Detect process exit, then fail pending futures and raise `ProcessError` carrying the stderr tail |
| Shutdown | Unsubscribe threads, close stdin, wait (respecting the server's 30s drain), SIGTERM, wait, SIGKILL, plus an `atexit` reaper (pattern from `claude:…/subprocess_cli.py:54-68`) |
| Several sessions | Many threads share one process, which is the default. Use a process per session only for isolation (a different `CODEX_HOME` or credentials). A websocket or unix-socket mode is optional, for sharing one server between clients. |

## 7. Unified SDK design

### 7.1 Layers

```text
┌────────────────────────── user API (backend-neutral) ──────────────────────────┐
│ AgentClient · Session · run()/stream() · @tool · hooks · AutoResponder · tasks │
├──────────────────────────────── core ──────────────────────────────────────────┤
│ Event model (common kinds + backend extensions) · RequestRouter (timeouts,      │
│ fallbacks) · Capability registry · Lifecycle manager · Error types              │
├──────────── Backend adapter (one per binary; a Protocol class) ────────────────┤
│ ClaudeBackend: stream-json + control protocol                                   │
│ CodexBackend:  JSON-RPC app-server (generated pydantic models from sdk/python)  │
│ (future: other agent binaries)                                                  │
├──────────────────────────────── transport ─────────────────────────────────────┤
│ SubprocessTransport (NDJSON) · WebSocketTransport · UnixSocketTransport · Mock  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

The backend adapter interface is the unification point:

```python
class Backend(Protocol):
    name: str                      # "claude", "codex"
    capabilities: frozenset[Capability]

    async def connect(self, options: AgentOptions, router: RequestRouter) -> None: ...
    async def close(self) -> None: ...

    async def start_session(self, spec: SessionSpec) -> SessionRef: ...
    async def resume_session(self, session_id: str, spec: SessionSpec) -> SessionRef: ...
    async def fork_session(self, session_id: str, *, at: str | None = None) -> SessionRef: ...
    async def list_sessions(self, filter: SessionFilter) -> list[SessionInfo]: ...

    def send_turn(self, session: SessionRef, input: list[InputPart],
                  overrides: TurnOverrides) -> AsyncIterator[Event]: ...
    async def steer(self, session: SessionRef, input: list[InputPart]) -> None: ...
    async def interrupt(self, session: SessionRef) -> None: ...

    async def list_tasks(self, session: SessionRef) -> list[TaskInfo]: ...
    async def stop_task(self, session: SessionRef, task_id: str) -> None: ...
```

Every optional feature is a `Capability` flag, for example `STEER`, `REWIND_FILES`, `REVERT_HISTORY`, `HOOKS_INPROC`, `HOOKS_BRIDGED`, `TOOL_INPUT_REWRITE`, `QUESTIONS`, `ELICITATION`, `BACKGROUND_TASK_LIST`, `DRIVE_SUBAGENT`, `SESSION_STORE`, `COST`. Calling an unsupported feature raises `UnsupportedCapability(backend, capability)`, never a silent no-op. `client.capabilities` lets callers branch.

### 7.2 Options

The common fields cover what both binaries understand. Backend-specific fields live in a typed extension block:

```python
AgentOptions(
    cwd=..., model=..., effort=..., system_prompt=..., output_schema=...,
    permissions=PermissionProfile(sandbox="workspace-write", network=False,
                                  writable_roots=[...]),
    responder=AutoResponder(policy=..., on_approval=..., on_question=...,
                            on_elicitation=..., request_timeout=30),
    tools=[my_tool], mcp_servers={...}, agents={"reviewer": AgentDefinition(...)},
    hooks={"PreToolUse": [HookMatcher("shell", [cb])]},
    cli_path=None, env={...}, stderr=callback,
    claude=ClaudeExtras(setting_sources=..., session_store=...),
    codex=CodexExtras(config_overrides={...}, approvals_reviewer="user",
                      codex_home=None, experimental_api=True),
)
```

Each adapter maps the common fields (for example `PermissionProfile` becomes Claude `permission_mode` plus `sandbox`, or Codex `approvalPolicy` plus `sandboxPolicy`) and rejects contradictory combinations at construction time.

### 7.3 Event model: a common core that allows variation

Every event has a common envelope and a typed `kind`. The backend's original frame is always kept.

```python
@dataclass
class Event:
    kind: str                 # from the common vocabulary, or "<backend>.<name>"
    backend: str              # "claude" | "codex"
    session_id: str           # Claude session_id | Codex threadId
    turn_id: str | None       # Codex turnId | synthesized per Claude query
    item_id: str | None       # Codex itemId | Claude tool_use_id or message id
    agent: AgentRef | None    # subagent identity (see below); None = main agent
    raw: dict                 # the untouched backend frame
```

The common vocabulary is deliberately small. Each kind maps from both backends:

| Kind | Claude source | Codex source |
|---|---|---|
| `session.started` | `system/init` | `thread/started` |
| `turn.started` | first frame after a query | `turn/started` |
| `message.delta` | `StreamEvent` text delta | `item/agentMessage/delta` |
| `reasoning.delta` | `StreamEvent` thinking delta | `item/reasoning/*Delta` |
| `message.completed` | `AssistantMessage` (text blocks) | `item/completed` with `agentMessage` |
| `tool.started` | `ToolUseBlock` | `item/started` with any tool-like item |
| `tool.output.delta` | none | `item/commandExecution/outputDelta`, `item/fileChange/outputDelta` |
| `tool.completed` | `ToolResultBlock` | `item/completed` with any tool-like item |
| `hook.started` / `hook.completed` | `HookEventMessage` | `hook/started` / `hook/completed` |
| `subagent.started` / `subagent.completed` | `SubagentStart` / `SubagentStop`; `parent_tool_use_id` appears | `subAgentActivity` items; child `thread/started` |
| `task.started` / `task.progress` / `task.completed` | `task_*` system messages | `commandExecution` with source `unifiedExec*`, plus deltas |
| `usage` | `ResultMessage.usage` | `thread/tokenUsage/updated` |
| `turn.completed` | `ResultMessage` | `turn/completed` |
| `error` / `warning` | `ResultError`, `RateLimitEvent` | `error`, `warning` |

Tool events carry a normalized `tool_kind`: `shell`, `file_change`, `mcp`, `web_search`, `custom`, `subagent`, or `other`. They also carry `name`, `input`, `output`, `status`, and a typed `detail` from the backend.

Variation is allowed in two ways:

1. **Backend-namespaced kinds** for anything without a common meaning, for example `codex.turn.diff.updated`, `codex.turn.plan.updated`, `codex.guardian.review` and `claude.rate_limit`. Consumers that match only common kinds ignore them safely.
2. **`detail` and `raw`.** Every common event carries a typed backend `detail` model (such as Codex `CommandExecutionThreadItem` or Claude `ToolUseBlock`) and the untouched `raw` frame. Unknown frames become `Event(kind="<backend>.unknown", raw=…)` and are never dropped. This is the same approach that keeps the Claude SDK robust to CLI drift (`claude:_internal/message_parser.py:391-395`).

**Subagent unification.** `AgentRef(id, name, role, parent_id, path)` is filled from Claude `parent_tool_use_id` and `agent_id`/`agent_type`, or from Codex `threadId`, `parentThreadId`, `agentRole` and `agent_path`. On Codex, the adapter folds child-thread events into the parent session's stream, tagged with `agent`. A user can also open the child as its own `Session` (the `DRIVE_SUBAGENT` capability).

**Turn unification.** Claude has no turn id, so the adapter synthesizes one for each `query()` and closes it at `ResultMessage`. Codex turn ids pass through.

### 7.4 Request routing: one callback surface

Every backend-originated request becomes a typed request object handed to the `AutoResponder`:

```python
ApprovalRequest(kind: "tool_use" | "command" | "file_change" | "permissions" | "network",
                session_id, agent, tool_name, input, command, cwd, reason, paths,
                suggestions)            # Claude permission_suggestions | Codex amendments
-> Allow(updated_input=None, scope="once" | "session", amendments=None) | Deny(message, interrupt=False)

QuestionRequest(questions=[Question(id, header, text, options, multi, secret)])
-> Answers({question_id: [str]})

ElicitationRequest(server, schema | url, message) -> Accept(content) | Decline | Cancel
```

The adapter translates these as follows:

- **Codex.** `Allow` maps to `accept` or `acceptForSession`, `Deny(interrupt=False)` to `decline`, and `Deny(interrupt=True)` to `cancel`. An `Allow(updated_input=…)` raises `UnsupportedCapability(TOOL_INPUT_REWRITE)`, unless the hook bridge is active.
- **Claude.** `Allow` and `Deny` map onto `PermissionResultAllow` and `PermissionResultDeny`. A `can_use_tool` call for `AskUserQuestion` is turned into a `QuestionRequest`, and its `Answers` are written back through `updated_input`.

### 7.5 What to reuse

| From | Reuse |
|---|---|
| `sdk/python` (`openai-codex`) | The generated pydantic models (`generated/v2_all.py`) and the notification registry. Regenerate with `--experimental` so `dynamicTools`, server-request params and `backgroundTerminals` are included; today they are missing because only the stable schema is used. Also the `MessageRouter` per-turn routing idea and the `MockResponsesServer` and `AppServerHarness` test pattern. |
| `sdk/python-runtime` | Binary packaging (`openai-codex-cli-bin`) |
| `claude-agent-sdk-python` | The Transport, Query and Client layering, the control-request correlator, ID-based callback registration, defensive process handling, and the `MockTransport` testing pattern. It is MIT licensed, so porting code is allowed if the copyright notice is kept. |
| Not reusable | The `sdk/python` transport, which uses sync threads, hides the approval handler from the public API, runs the handler on the reader thread and has no request timeouts. `sdk/typescript` drives `codex exec` JSONL, which cannot carry approvals or tools. `gosdk` is an unrelated tool server. |

## 8. Gaps and risks

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| 1 | Many required Codex APIs are experimental (`dynamicTools`, `requestUserInput`, `backgroundTerminals`, paginated history and `revert`, `turn/settings/update`) | API churn breaks the adapter | Pin a minimum and a tested binary version through `openai-codex-cli-bin`; regenerate models in CI; contract tests against a real app-server |
| 2 | No in-process hooks in Codex | `hooks` and `on_tool_use` need the MCP bridge | Spike S2; mark the capability as `HOOKS_BRIDGED` |
| 3 | Server requests never time out | A slow or buggy callback stalls the run | SDK-side timeouts with fallback decisions; always reply |
| 4 | Questions only in Plan mode by default | `on_question` silently never fires | Set `default_mode_request_user_input`; document it |
| 5 | Connections are auto-subscribed to every thread, and a lagging `thread_created` is skipped (`codex-rs/app-server/src/lib.rs:1346-1350`) | A subagent's early events may be missed | Demultiplex by `threadId`; re-read child threads with `thread/read` on `subAgentActivity` |
| 6 | Guardian notifications are marked unstable | Breaks if `auto_review` is used | Default the reviewer to `user`; keep guardian events backend-namespaced |
| 7 | No cost reporting and no external session store in Codex | Feature asymmetry | Optional fields and capability flags |
| 8 | Subagent roles need config files | Extra temporary files and cleanup | Write them under a session temp directory; remove on close |
| 9 | Stdio supports one client per process | Sharing a server between SDK clients is not possible over stdio | One process per `AgentClient`; optional unix or websocket transport |
| 10 | Semantics differ even under shared names (rewind restores files on Claude but truncates history on Codex; `decline` continues the turn on Codex) | Surprising behaviour | Keep separate capability names; document per-backend semantics next to each common API |

## 9. Next steps

1. **S1, transport spike.** Write a native asyncio JSON-RPC client for `codex app-server` that does the following:
   - start a thread;
   - stream a turn;
   - answer a command approval under `on-request` with sandbox `workspace-write`;
   - answer `requestUserInput` with the flag on;
   - run a dynamic tool.

   Use the `sdk/python` mock Responses server.
2. **S2, hook bridge spike.** Write an SDK-hosted MCP server and register an `mcp_tool` `PreToolUse` hook through `-c` or a temporary config layer. Measure latency, and confirm the hook can block and rewrite input.
3. **S3, subagent spike.** Write an `AgentDefinition` to a role config file, spawn it, and fold the child-thread events into the parent stream.
4. **Core.** Write the event model, `AutoResponder`, capability registry, and the two adapters. The Claude adapter comes first, as a port, because it validates the abstraction against a known-good implementation.
5. **Conformance suite.** Run the same scenarios against both backends, with capability-aware skips, the way `testing.run_session_store_conformance` does for the Claude SDK.
