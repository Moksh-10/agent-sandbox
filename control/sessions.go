package control

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

// Sessions is in-memory on purpose — easy to read while learning.
type Sessions struct {
	mu   sync.RWMutex
	byID map[string]types.Session
	loop *Loop
}

func NewSessions(loop *Loop) *Sessions {
	return &Sessions{byID: make(map[string]types.Session), loop: loop}
}

func (s *Sessions) Create(ctx context.Context, userMessage string) (types.Session, error) {
	sess := types.Session{
		ID:          fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		UserMessage: userMessage,
		Status:      types.SessionPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.mu.Lock()
	s.byID[sess.ID] = sess
	s.mu.Unlock()
	return sess, nil
}

func (s *Sessions) Get(_ context.Context, id string) (types.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.byID[id]
	if !ok {
		return types.Session{}, fmt.Errorf("session not found: %s", id)
	}
	return sess, nil
}

func (s *Sessions) save(sess types.Session) {
	s.mu.Lock()
	s.byID[sess.ID] = sess
	s.mu.Unlock()
}

func (s *Sessions) Run(ctx context.Context, sessionID string) error {
	sess, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.loop.Run(ctx, sess, s.save)
}
