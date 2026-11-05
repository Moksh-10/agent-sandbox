# sb-agent

Learning repo: **agent loop on one machine**, **tool execution on a sandbox** (local process, Instabox, etc.). Not production code.

Details: [docs/architecture.md](docs/architecture.md)

## Run (two terminals)

```bash
go run ./cmd/sandbox-worker
go run ./cmd/control-plane
```

Optional third terminal — fire one session:

```bash
go run ./cmd/play
```

Orchestrator mode (new sandbox id per session, toy factory):

```bash
go run ./cmd/sandbox-orchestrator
SANDBOX_ALLOCATOR=orchestrator go run ./cmd/control-plane
```

## Code layout

```
control/   mock agent loop + HTTP API + registry + remote HTTP to sandbox
worker/    tools run here (inside the sandbox)
types/     shared structs
cmd/       entrypoints
```

Read in order: `control/loop.go` → `control/remote.go` → `worker/tools.go`
