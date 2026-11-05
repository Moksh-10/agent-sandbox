# Architecture

## Split

The **control plane** runs the agent loop (here: a fixed mock plan, no real LLM). It never runs `shell` or reads your repo directly.

The **sandbox worker** runs on another process or VM. It only implements tools over HTTP.

```
  your laptop / server                         sandbox (local :9090 or Instabox)
 ┌─────────────────────────┐                ┌─────────────────────────┐
 │ control/loop.go         │   HTTP POST    │ worker/tools.go         │
 │  mock "model" turns     │ ─────────────► │  read, write, shell…    │
 │ control/remote.go       │                │ worker/server.go        │
 └─────────────────────────┘                └─────────────────────────┘
```

Instabox, E2B, or EC2 only change **how the worker gets a URL**. The control plane still calls `base_url + /v1/tools/execute`.

## Sandbox URL

Not stored in control-plane `.env` for each user. Either:

1. **Registry mode** (`SANDBOX_ALLOCATOR=pool`): worker registers on boot with its current URL.
2. **Factory mode** (`SANDBOX_ALLOCATOR=orchestrator`): `cmd/sandbox-orchestrator` returns `{sandbox_id, base_url}` per session.

## Files

| File | Role |
|------|------|
| `control/mock_plan.go` | Fake tool batches |
| `control/loop.go` | Turns + parallel dispatch |
| `control/remote.go` | HTTP to worker |
| `control/registry.go` | id → URL |
| `control/pick.go` | Allocate/release sandbox for session |
| `worker/tools.go` | Actual tool implementation |
| `worker/register.go` | Worker → control plane registration |
