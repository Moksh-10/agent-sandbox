package worker

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Moksh-10/agent-sandbox/config"
	"github.com/Moksh-10/agent-sandbox/types"
)

func Start() {
	config.LoadDotEnvFromWD()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	token := os.Getenv("SB_AGENT_TOKEN")
	workerID := env("SANDBOX_WORKER_ID", "sandbox-local-1")
	addr := env("SANDBOX_WORKER_ADDR", ":9090")
	publicURL := env("SANDBOX_PUBLIC_URL", "http://127.0.0.1:9090")
	controlURL := env("CONTROL_PLANE_URL", "http://127.0.0.1:8080")
	workspace := env("SANDBOX_WORKSPACE", ".")

	srv := &Server{WorkerID: workerID, Workspace: workspace, Log: log}
	mux := http.NewServeMux()
	srv.Mount(mux)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", 401)
			return
		}
		mux.ServeHTTP(w, r)
	})

	httpSrv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Info("sandbox worker listening", "addr", addr, "public_url", publicURL)
		_ = httpSrv.ListenAndServe()
	}()

	meta := types.SandboxWorker{
		ID: workerID, BaseURL: publicURL,
		Capabilities: []types.ToolName{types.ToolRead, types.ToolWrite, types.ToolShell, types.ToolGrep, types.ToolExaSearch, types.ToolContext7},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if controlURL != "" {
		if err := RegisterWithControl(ctx, controlURL, token, meta); err != nil {
			log.Error("register failed", "error", err)
		} else {
			log.Info("registered with control plane", "url", controlURL)
		}
		go func() {
			t := time.NewTicker(15 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					_ = Heartbeat(ctx, controlURL, token, workerID)
				}
			}
		}()
	}

	<-ctx.Done()
	_ = httpSrv.Shutdown(context.Background())
	log.Info("sandbox worker stopped")
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
