package control

import (
	"context"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

// Remote hands tool calls to a worker through the dispatcher; the worker pulls them.
type Remote struct {
	Dispatch *Dispatcher
	Timeout  time.Duration
}

func NewRemote(d *Dispatcher, timeout time.Duration) *Remote {
	return &Remote{Dispatch: d, Timeout: timeout}
}

func (r *Remote) RunTool(ctx context.Context, sandboxID string, call types.ToolCall) (types.ToolResult, error) {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	start := time.Now()
	res, err := r.Dispatch.Submit(ctx, sandboxID, call)
	if err != nil {
		return types.ToolResult{}, err
	}
	if res.StartedAt.IsZero() {
		res.StartedAt = start
	}
	if res.EndedAt.IsZero() {
		res.EndedAt = time.Now()
	}
	res.SandboxID = sandboxID
	return res, nil
}
