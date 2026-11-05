package types

import "time"

type ToolName string

const (
	ToolRead      ToolName = "read"
	ToolWrite     ToolName = "write"
	ToolShell     ToolName = "shell"
	ToolGrep      ToolName = "grep"
	ToolExaSearch ToolName = "exa_search"
	ToolContext7  ToolName = "context7_query"
)

// ToolCall is what the agent loop decides; sent over HTTP to the sandbox worker.
type ToolCall struct {
	ID        string
	SessionID string
	Turn      int
	Name      ToolName
	Arguments map[string]string
}

type ToolResult struct {
	CallID    string
	Name      ToolName
	Output    string
	Error     string
	StartedAt time.Time
	EndedAt   time.Time
	SandboxID string
}

// AgentTurn = one model step. Real agents often emit several ToolCalls at once.
type AgentTurn struct {
	Index int
	Calls []ToolCall
}



// func (s *Substitutor) replaceLiterals(input []byte) []byte {
//     out := make([]byte, 0, len(input))

//     for i := 0; i < len(input); {
//         replaced := false

//         for _, token := range s.tokens {
//             if bytes.HasPrefix(input[i:], token.Dummy) {
//                 out = append(out, token.Value...)
//                 i += len(token.Dummy)
//                 replaced = true
//                 break
//             }
//         }

//         if replaced {
//             continue
//         }

//         out = append(out, input[i])
//         i++
//     }

//     return out
// }
