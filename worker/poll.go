package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

type Poller struct {
	ControlURL  string
	Token       string
	Meta        types.SandboxWorker
	Workspace   string
	Concurrency int
	Log         *slog.Logger
	HTTP        *http.Client
}

func (p *Poller) Run(ctx context.Context) {
	if p.HTTP == nil {
		p.HTTP = &http.Client{Timeout: 40 * time.Second}
	}
	if p.Concurrency <= 0 {
		p.Concurrency = 4
	}
	p.register(ctx)
	go p.heartbeatLoop(ctx)

	sem := make(chan struct{}, p.Concurrency)
	for ctx.Err() == nil {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		call, status, err := p.poll(ctx)
		switch {
		case err != nil:
			<-sem
			p.Log.Warn("poll failed", "error", err)
			sleep(ctx, 2*time.Second)
		case status == http.StatusNotFound:
			<-sem
			p.register(ctx)
		case call == nil:
			<-sem
		default:
			go func() {
				defer func() { <-sem }()
				p.execute(*call)
			}()
		}
	}
}

func (p *Poller) register(ctx context.Context) {
	for ctx.Err() == nil {
		if err := RegisterWithControl(ctx, p.ControlURL, p.Token, p.Meta); err != nil {
			p.Log.Error("register failed", "error", err)
			sleep(ctx, 2*time.Second)
			continue
		}
		p.Log.Info("registered with control plane", "url", p.ControlURL, "worker_id", p.Meta.ID)
		return
	}
}

func (p *Poller) heartbeatLoop(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = Heartbeat(ctx, p.ControlURL, p.Token, p.Meta.ID)
		}
	}
}

func (p *Poller) poll(ctx context.Context) (*types.ToolCall, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.ControlURL+"/internal/workers/"+p.Meta.ID+"/poll", nil)
	if err != nil {
		return nil, 0, err
	}
	p.auth(req)
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var out struct {
			Call types.ToolCall `json:"call"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, 0, err
		}
		return &out.Call, resp.StatusCode, nil
	case http.StatusNoContent, http.StatusNotFound:
		return nil, resp.StatusCode, nil
	default:
		return nil, resp.StatusCode, fmt.Errorf("poll HTTP %d", resp.StatusCode)
	}
}

func (p *Poller) execute(call types.ToolCall) {
	p.Log.Info("execute tool", "tool", call.Name, "call_id", call.ID)
	start := time.Now()
	out, errMsg := RunTool(p.Workspace, call)
	res := types.ToolResult{
		CallID: call.ID, Name: call.Name, Output: out, Error: errMsg,
		StartedAt: start, EndedAt: time.Now(), SandboxID: p.Meta.ID,
	}
	body, _ := json.Marshal(map[string]any{"result": res})

	for attempt := 1; attempt <= 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.ControlURL+"/internal/workers/"+p.Meta.ID+"/results", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		p.auth(req)
		resp, err := p.HTTP.Do(req)
		cancel()
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 300 || resp.StatusCode == http.StatusGone {
				return
			}
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	p.Log.Error("could not deliver result", "call_id", call.ID)
}

func (p *Poller) auth(req *http.Request) {
	if p.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.Token)
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
