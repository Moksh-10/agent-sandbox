package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

type FactoryClient struct {
	BaseURL string
	HTTP    *http.Client
	Token   string
}

func NewFactoryClient(baseURL, token string) *FactoryClient {
	return &FactoryClient{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 5 * time.Minute},
		Token:   token,
	}
}

func (c *FactoryClient) Create(ctx context.Context, sessionID string) (types.SandboxWorker, error) {
	body, _ := json.Marshal(map[string]string{"session_id": sessionID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/sandboxes", bytes.NewReader(body))
	if err != nil {
		return types.SandboxWorker{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return types.SandboxWorker{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return types.SandboxWorker{}, fmt.Errorf("factory HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		SandboxID string `json:"sandbox_id"`
		BaseURL   string `json:"base_url"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return types.SandboxWorker{}, err
	}
	return types.SandboxWorker{
		ID: parsed.SandboxID, BaseURL: parsed.BaseURL,
		Capabilities: []types.ToolName{types.ToolRead, types.ToolWrite, types.ToolShell, types.ToolGrep, types.ToolExaSearch, types.ToolContext7},
	}, nil
}

func (c *FactoryClient) Destroy(ctx context.Context, sandboxID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+"/v1/sandboxes/"+sandboxID, nil)
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("factory destroy HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}
