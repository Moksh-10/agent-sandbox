package control

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// StartFactory is a toy sandbox creator — returns a new sandbox_id each time.
// Replace this with Instabox/E2B API calls when you learn the real provider.
func StartFactory() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	token := os.Getenv("SB_AGENT_TOKEN")
	addr := getenv("ORCHESTRATOR_ADDR", ":7070")
	workerBase := getenv("ORCHESTRATOR_WORKER_BASE_URL", "http://127.0.0.1:9090")

	mu := sync.Mutex{}
	live := map[string]string{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"role": "sandbox-factory", "status": "ok"})
	})
	mux.HandleFunc("POST /v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		id := fmt.Sprintf("sb-%d", time.Now().UnixNano())
		mu.Lock()
		live[id] = req.SessionID
		mu.Unlock()
		log.Info("created sandbox slot", "id", id, "session", req.SessionID, "worker_url", workerBase)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]string{"sandbox_id": id, "base_url": workerBase})
	})
	mux.HandleFunc("DELETE /v1/sandboxes/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		delete(live, r.PathValue("id"))
		mu.Unlock()
		w.WriteHeader(204)
	})

	var h http.Handler = mux
	if token != "" {
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				http.Error(w, "unauthorized", 401)
				return
			}
			mux.ServeHTTP(w, r)
		})
	}
	log.Info("sandbox-factory listening", "addr", addr)
	_ = http.ListenAndServe(addr, h)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
