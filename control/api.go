package control

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

type API struct {
	Sessions   *Sessions
	Registry   *Registry
	Dispatcher *Dispatcher
	Log        *slog.Logger
}

func (a *API) Mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]string{"role": "control-plane", "status": "ok"})
	})
	mux.HandleFunc("POST /v1/sessions", a.createSession)
	mux.HandleFunc("GET /v1/sessions/{id}", a.getSession)
	mux.HandleFunc("POST /v1/sessions/{id}/run", a.runSession)
	mux.HandleFunc("POST /internal/workers/register", a.registerWorker)
	mux.HandleFunc("POST /internal/workers/{id}/heartbeat", a.heartbeat)
	mux.HandleFunc("GET /internal/workers/{id}/poll", a.pollWorker)
	mux.HandleFunc("POST /internal/workers/{id}/results", a.postResult)
}

func (a *API) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, "message required", 400)
		return
	}
	sess, err := a.Sessions.Create(r.Context(), req.Message)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	a.Log.Info("session created", "id", sess.ID)
	writeJSON(w, 201, sess)
}

func (a *API) getSession(w http.ResponseWriter, r *http.Request) {
	sess, err := a.Sessions.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	writeJSON(w, 200, sess)
}

func (a *API) runSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.Sessions.Get(r.Context(), id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_ = a.Sessions.Run(ctx, id)
	}()
	writeJSON(w, 202, map[string]string{"session_id": id, "status": "running"})
}

func (a *API) registerWorker(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID           string            `json:"id"`
		BaseURL      string            `json:"base_url"`
		Capabilities []types.ToolName  `json:"capabilities"`
		Labels       map[string]string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "id required", 400)
		return
	}
	worker := types.SandboxWorker{ID: req.ID, BaseURL: req.BaseURL, Capabilities: req.Capabilities, Labels: req.Labels}
	if err := a.Registry.Register(r.Context(), worker); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	a.Log.Info("sandbox worker registered", "id", worker.ID, "url", worker.BaseURL)
	writeJSON(w, 200, map[string]string{"status": "registered"})
}

func (a *API) heartbeat(w http.ResponseWriter, r *http.Request) {
	if err := a.Registry.Heartbeat(r.Context(), r.PathValue("id")); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) pollWorker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.Registry.Heartbeat(r.Context(), id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	call, ok := a.Dispatcher.Next(ctx, id)
	if !ok {
		w.WriteHeader(204)
		return
	}
	writeJSON(w, 200, map[string]any{"call": call})
}

func (a *API) postResult(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Result types.ToolResult `json:"result"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if !a.Dispatcher.Complete(body.Result) {
		http.Error(w, "no caller waiting for this call", 410)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func BearerMiddleware(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}
