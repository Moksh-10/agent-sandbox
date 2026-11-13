package control

import (
	"context"
	"fmt"
	"sync"

	"github.com/Moksh-10/agent-sandbox/types"
)

type Dispatcher struct {
	mu      sync.Mutex
	queues  map[string]chan types.ToolCall
	waiting map[string]chan types.ToolResult
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		queues:  make(map[string]chan types.ToolCall),
		waiting: make(map[string]chan types.ToolResult),
	}
}

func (d *Dispatcher) queue(workerID string) chan types.ToolCall {
	d.mu.Lock()
	defer d.mu.Unlock()
	q, ok := d.queues[workerID]
	if !ok {
		q = make(chan types.ToolCall, 64)
		d.queues[workerID] = q
	}
	return q
}

func (d *Dispatcher) isWaiting(callID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.waiting[callID]
	return ok
}

func (d *Dispatcher) Submit(ctx context.Context, workerID string, call types.ToolCall) (types.ToolResult, error) {
	done := make(chan types.ToolResult, 1)
	d.mu.Lock()
	d.waiting[call.ID] = done
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.waiting, call.ID)
		d.mu.Unlock()
	}()

	select {
	case d.queue(workerID) <- call:
	case <-ctx.Done():
		return types.ToolResult{}, fmt.Errorf("queue for worker %s: %w", workerID, ctx.Err())
	}

	select {
	case res := <-done:
		return res, nil
	case <-ctx.Done():
		return types.ToolResult{}, fmt.Errorf("waiting on worker %s: %w", workerID, ctx.Err())
	}
}

func (d *Dispatcher) Next(ctx context.Context, workerID string) (types.ToolCall, bool) {
	q := d.queue(workerID)
	for {
		select {
		case call := <-q:
			if d.isWaiting(call.ID) {
				return call, true
			}
		case <-ctx.Done():
			return types.ToolCall{}, false
		}
	}
}

func (d *Dispatcher) Complete(res types.ToolResult) bool {
	d.mu.Lock()
	ch, ok := d.waiting[res.CallID]
	d.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- res:
	default:
	}
	return true
}
