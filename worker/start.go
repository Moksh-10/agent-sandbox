package worker

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Moksh-10/agent-sandbox/config"
	"github.com/Moksh-10/agent-sandbox/types"
)

func Start() {
	config.LoadDotEnvFromWD()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	concurrency, _ := strconv.Atoi(env("SANDBOX_WORKER_CONCURRENCY", "4"))
	p := &Poller{
		ControlURL: env("CONTROL_PLANE_URL", "http://127.0.0.1:8080"),
		Token:      os.Getenv("SB_AGENT_TOKEN"),
		Workspace:  env("SANDBOX_WORKSPACE", "."),
		Meta: types.SandboxWorker{
			ID:           env("SANDBOX_WORKER_ID", "sandbox-local-1"),
			Capabilities: []types.ToolName{types.ToolRead, types.ToolWrite, types.ToolShell, types.ToolGrep, types.ToolExaSearch, types.ToolContext7},
		},
		Concurrency: concurrency,
		Log:         log,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("sandbox worker polling", "control", p.ControlURL, "worker_id", p.Meta.ID)
	p.Run(ctx)
	log.Info("sandbox worker stopped")
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
