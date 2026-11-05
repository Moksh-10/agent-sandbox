package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

// Remote sends tool calls from control plane to sandbox worker over HTTP.
type Remote struct {
	HTTP     *http.Client
	Registry *Registry
	Token    string
}

func NewRemote(reg *Registry, token string) *Remote {
	return &Remote{
		HTTP:     &http.Client{Timeout: 2 * time.Minute},
		Registry: reg,
		Token:    token,
	}
}

func (r *Remote) RunTool(ctx context.Context, sandboxID string, call types.ToolCall) (types.ToolResult, error) {
	baseURL, err := r.Registry.URLFor(sandboxID)
	if err != nil {
		return types.ToolResult{}, err
	}

	body, _ := json.Marshal(struct {
		Call types.ToolCall `json:"call"`
	}{call})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/tools/execute", bytes.NewReader(body))
	if err != nil {
		return types.ToolResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := r.authToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	start := time.Now()
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return types.ToolResult{}, fmt.Errorf("worker unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return types.ToolResult{}, fmt.Errorf("worker HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Result types.ToolResult `json:"result"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return types.ToolResult{}, err
	}
	if out.Result.StartedAt.IsZero() {
		out.Result.StartedAt = start
	}
	if out.Result.EndedAt.IsZero() {
		out.Result.EndedAt = time.Now()
	}
	out.Result.SandboxID = sandboxID
	return out.Result, nil
}

func (r *Remote) authToken() string {
	if t := os.Getenv("INSTABOX_PORT_TOKEN"); t != "" {
		return t
	}
	return r.Token
}
