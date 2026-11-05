package control

import (
	"fmt"

	"github.com/Moksh-10/agent-sandbox/types"
)

// MockPlan fakes what an LLM would output: several turns, many tools, some in parallel.
// There is no real model here — just so you can watch the loop in logs.
func MockPlan(sessionID, userMessage string) ([]types.AgentTurn, int) {
	paths := []string{
		"cmd/control-plane/main.go",
		"control/loop.go",
		"control/remote.go",
		"docs/architecture.md",
		"go.mod",
	}
	patterns := []string{"Remote", "Registry", "Allocate", "parallel", "worker"}

	var turns []types.AgentTurn
	n := 0
	nextID := func() string {
		n++
		return fmt.Sprintf("%s-call-%03d", sessionID, n)
	}
	addTurn := func(calls ...types.ToolCall) {
		turns = append(turns, types.AgentTurn{Index: len(turns) + 1, Calls: calls})
	}

	t1 := make([]types.ToolCall, 0, 5)
	for i := 0; i < 5; i++ {
		t1 = append(t1, types.ToolCall{
			ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1,
			Name: types.ToolRead, Arguments: map[string]string{"path": paths[i%len(paths)]},
		})
	}
	addTurn(t1...)

	addTurn(
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolExaSearch,
			Arguments: map[string]string{"query": "control plane agent loop sandbox worker tools"}},
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolContext7,
			Arguments: map[string]string{"library": "go", "query": "http client"}},
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolContext7,
			Arguments: map[string]string{"library": "net/http", "query": "ServeMux"}},
	)

	t3 := make([]types.ToolCall, 0, 6)
	for i := 0; i < 6; i++ {
		t3 = append(t3, types.ToolCall{
			ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1,
			Name: types.ToolGrep, Arguments: map[string]string{"pattern": patterns[i%len(patterns)]},
		})
	}
	addTurn(t3...)

	addTurn(
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolShell,
			Arguments: map[string]string{"command": "go version && uname -a"}},
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolWrite,
			Arguments: map[string]string{"path": "/tmp/agent-note.txt", "content": "session:" + sessionID}},
	)

	t5 := make([]types.ToolCall, 0, 4)
	for i := 0; i < 4; i++ {
		t5 = append(t5, types.ToolCall{
			ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1,
			Name: types.ToolRead, Arguments: map[string]string{"path": paths[(i+1)%len(paths)]},
		})
	}
	addTurn(t5...)

	t6 := make([]types.ToolCall, 0, 3)
	for _, q := range []string{"instabox sandbox", "e2b dev sandbox", "firecracker microvm"} {
		t6 = append(t6, types.ToolCall{
			ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1,
			Name: types.ToolExaSearch, Arguments: map[string]string{"query": q},
		})
	}
	addTurn(t6...)

	addTurn(
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolGrep,
			Arguments: map[string]string{"pattern": userMessage}},
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolShell,
			Arguments: map[string]string{"command": "ls -la"}},
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolRead,
			Arguments: map[string]string{"path": "go.mod"}},
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolContext7,
			Arguments: map[string]string{"library": "log/slog", "query": "structured logging"}},
		types.ToolCall{ID: nextID(), SessionID: sessionID, Turn: len(turns) + 1, Name: types.ToolWrite,
			Arguments: map[string]string{"path": "/tmp/summary.txt", "content": "mock agent done"}},
	)

	total := 0
	for _, t := range turns {
		total += len(t.Calls)
	}
	return turns, total
}
