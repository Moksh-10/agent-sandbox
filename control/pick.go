package control

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

// SandboxPicker chooses a sandbox for one session. Provider-agnostic interface.
type SandboxPicker interface {
	Allocate(ctx context.Context, sessionID string) (types.SandboxSlot, error)
	Release(ctx context.Context, slot types.SandboxSlot) error
}

// PickFromRegistry uses a worker that registered with the control plane.
type PickFromRegistry struct {
	Reg *Registry
}

func (p *PickFromRegistry) Allocate(ctx context.Context, sessionID string) (types.SandboxSlot, error) {
	w, err := p.Reg.PickAnyHealthy()
	if err != nil {
		return types.SandboxSlot{}, err
	}
	return types.SandboxSlot{
		SandboxID: w.ID, BaseURL: w.BaseURL, SessionID: sessionID, Allocated: time.Now(),
	}, nil
}

func (p *PickFromRegistry) Release(context.Context, types.SandboxSlot) error { return nil }

// PickFromFactory asks a sandbox-factory service for a fresh id+url each session.
// Real factory talks to Instabox/E2B/K8s; control plane only sees id + URL.
type PickFromFactory struct {
	Factory *FactoryClient
	Reg     *Registry
	Log     *slog.Logger
}

func (p *PickFromFactory) Allocate(ctx context.Context, sessionID string) (types.SandboxSlot, error) {
	w, err := p.Factory.Create(ctx, sessionID)
	if err != nil {
		return types.SandboxSlot{}, err
	}
	w.Labels = map[string]string{"session_id": sessionID}
	if err := p.Reg.Register(ctx, w); err != nil {
		_ = p.Factory.Destroy(ctx, w.ID)
		return types.SandboxSlot{}, err
	}
	p.Log.Info("factory gave sandbox", "id", w.ID, "url", w.BaseURL)
	return types.SandboxSlot{
		SandboxID: w.ID, BaseURL: w.BaseURL, SessionID: sessionID, Allocated: time.Now(),
	}, nil
}

func (p *PickFromFactory) Release(ctx context.Context, slot types.SandboxSlot) error {
	_ = p.Reg.Remove(ctx, slot.SandboxID)
	if err := p.Factory.Destroy(ctx, slot.SandboxID); err != nil {
		return fmt.Errorf("factory destroy: %w", err)
	}
	p.Log.Info("sandbox released", "id", slot.SandboxID)
	return nil
}
