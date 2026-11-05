package worker

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

type Server struct {
	WorkerID  string
	Workspace string
	Log       *slog.Logger
}

func (s *Server) Mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, map[string]string{"role": "sandbox-worker", "worker_id": s.WorkerID})
	})
	mux.HandleFunc("POST /v1/tools/execute", s.execute)
}

func (s *Server) execute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Call types.ToolCall `json:"call"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	start := time.Now()
	s.Log.Info("execute tool", "tool", body.Call.Name, "call_id", body.Call.ID)
	out, errMsg := RunTool(s.Workspace, body.Call)
	end := time.Now()
	jsonOK(w, map[string]any{"result": types.ToolResult{
		CallID: body.Call.ID, Name: body.Call.Name, Output: out, Error: errMsg,
		StartedAt: start, EndedAt: end, SandboxID: s.WorkerID,
	}})
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
