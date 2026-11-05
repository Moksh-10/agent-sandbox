package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

func RegisterWithControl(ctx context.Context, controlURL, token string, w types.SandboxWorker) error {
	body, _ := json.Marshal(map[string]any{
		"id": w.ID, "base_url": w.BaseURL,
		"capabilities": w.Capabilities, "labels": w.Labels,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, controlURL+"/internal/workers/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("register HTTP %d", resp.StatusCode)
	}
	return nil
}

func Heartbeat(ctx context.Context, controlURL, token, workerID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, controlURL+"/internal/workers/"+workerID+"/heartbeat", nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("heartbeat HTTP %d", resp.StatusCode)
	}
	return nil
}
