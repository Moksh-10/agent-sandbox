package types

import "time"

type SessionStatus string

const (
	SessionPending   SessionStatus = "pending"
	SessionRunning   SessionStatus = "running"
	SessionCompleted SessionStatus = "completed"
	SessionFailed    SessionStatus = "failed"
)

type Session struct {
	ID           string
	UserMessage  string
	Status       SessionStatus
	SandboxID    string
	SandboxURL   string // set when sandbox is allocated for this session
	TotalTools   int
	Completed    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ErrorMessage string
}
