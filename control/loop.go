package control

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

// Loop is the fake agent: allocate sandbox → turns → remote tool calls → release.
type Loop struct {
	Picker SandboxPicker
	Remote *Remote
	Log    *slog.Logger
}

func (l *Loop) Run(ctx context.Context, sess types.Session, save func(types.Session)) error {
	slot, err := l.Picker.Allocate(ctx, sess.ID)
	if err != nil {
		return err
	}
	defer l.Picker.Release(context.Background(), slot)

	sess.Status = types.SessionRunning
	sess.SandboxID = slot.SandboxID
	sess.SandboxURL = slot.BaseURL
	sess.UpdatedAt = time.Now()
	turns, total := MockPlan(sess.ID, sess.UserMessage)
	sess.TotalTools = total
	save(sess)

	l.Log.Info("session using sandbox",
		"session", sess.ID, "sandbox_id", slot.SandboxID, "url", slot.BaseURL)

	for _, turn := range turns {
		l.Log.Info("turn start", "turn", turn.Index, "parallel", len(turn.Calls))
		if err := l.runTurn(ctx, slot.SandboxID, turn); err != nil {
			sess.Status = types.SessionFailed
			sess.ErrorMessage = err.Error()
			sess.UpdatedAt = time.Now()
			save(sess)
			return err
		}
		sess.Completed += len(turn.Calls)
		sess.UpdatedAt = time.Now()
		save(sess)
		l.Log.Info("turn done", "turn", turn.Index, "completed", sess.Completed, "of", total)
	}

	sess.Status = types.SessionCompleted
	sess.UpdatedAt = time.Now()
	save(sess)
	return nil
}

func (l *Loop) runTurn(ctx context.Context, sandboxID string, turn types.AgentTurn) error {
	results := make([]types.ToolResult, len(turn.Calls))
	errs := make([]error, len(turn.Calls))
	var wg sync.WaitGroup
	wg.Add(len(turn.Calls))
	for i, call := range turn.Calls {
		i, call := i, call
		go func() {
			defer wg.Done()
			l.Log.Debug("→ remote tool", "tool", call.Name, "call_id", call.ID)
			res, err := l.Remote.RunTool(ctx, sandboxID, call)
			results[i] = res
			errs[i] = err
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	for _, res := range results {
		l.Log.Info("← tool result", "tool", res.Name, "ms", res.EndedAt.Sub(res.StartedAt).Milliseconds())
	}
	return nil
}
