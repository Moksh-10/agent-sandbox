package control

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

// Registry maps sandbox_id → URL for remote tool calls.
type Registry struct {
	mu      sync.RWMutex
	workers map[string]types.SandboxWorker
	ttl     time.Duration
}

func NewRegistry(ttl time.Duration) *Registry {
	if ttl <= 0 {
		ttl = 45 * time.Second
	}
	return &Registry{workers: make(map[string]types.SandboxWorker), ttl: ttl}
}

func (r *Registry) Register(_ context.Context, w types.SandboxWorker) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	w.RegisteredAt = now
	w.LastHeartbeat = now
	r.workers[w.ID] = w
	return nil
}

func (r *Registry) Heartbeat(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.workers[id]
	if !ok {
		return fmt.Errorf("worker not found: %s", id)
	}
	w.LastHeartbeat = time.Now()
	r.workers[id] = w
	return nil
}

func (r *Registry) Remove(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workers, id)
	return nil
}

func (r *Registry) List(_ context.Context) ([]types.SandboxWorker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]types.SandboxWorker, 0, len(r.workers))
	for _, w := range r.workers {
		out = append(out, w)
	}
	return out, nil
}

func (r *Registry) URLFor(id string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.workers[id]
	if !ok {
		return "", fmt.Errorf("unknown sandbox id: %s", id)
	}
	return w.BaseURL, nil
}

func (r *Registry) PickAnyHealthy() (types.SandboxWorker, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	var best *types.SandboxWorker
	for id, w := range r.workers {
		if now.Sub(w.LastHeartbeat) > r.ttl {
			delete(r.workers, id)
			continue
		}
		if best == nil || w.LastHeartbeat.After(best.LastHeartbeat) {
			copy := w
			best = &copy
		}
	}
	if best == nil {
		return types.SandboxWorker{}, fmt.Errorf("no sandbox registered — start cmd/sandbox-worker first")
	}
	return *best, nil
}
