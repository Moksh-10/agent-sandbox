package control

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Moksh-10/agent-sandbox/config"
)

func Start() {
	config.LoadDotEnvFromWD()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	token := os.Getenv("SB_AGENT_TOKEN")
	addr := env("CONTROL_PLANE_ADDR", ":8080")

	reg := NewRegistry(45 * time.Second)
	remote := NewRemote(reg, token)

	var picker SandboxPicker = &PickFromRegistry{Reg: reg}
	if env("SANDBOX_ALLOCATOR", "pool") == "orchestrator" {
		factory := NewFactoryClient(env("ORCHESTRATOR_URL", "http://127.0.0.1:7070"), token)
		picker = &PickFromFactory{Factory: factory, Reg: reg, Log: log}
		log.Info("sandbox allocator", "mode", "orchestrator")
	} else {
		log.Info("sandbox allocator", "mode", "registry")
	}

	loop := &Loop{Picker: picker, Remote: remote, Log: log}
	sessions := NewSessions(loop)
	api := &API{Sessions: sessions, Registry: reg, Log: log}

	mux := http.NewServeMux()
	api.Mount(mux)

	srv := &http.Server{Addr: addr, Handler: BearerMiddleware(token, mux), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Info("control plane listening", "addr", addr)
		_ = srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	<-ctx.Done()
	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("control plane stopped")
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
