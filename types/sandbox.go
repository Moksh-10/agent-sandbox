package types

import "time"

// SandboxWorker is a sandbox daemon reachable at BaseURL (worker on some host).
type SandboxWorker struct {
	ID            string
	BaseURL       string
	Capabilities  []ToolName
	LastHeartbeat time.Time
	RegisteredAt  time.Time
	Labels        map[string]string
}

// SandboxSlot is one session's lease on a sandbox (id + url for this run).
type SandboxSlot struct {
	SandboxID string
	BaseURL   string
	SessionID string
	Allocated time.Time
}
