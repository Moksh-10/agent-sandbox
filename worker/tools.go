package worker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Moksh-10/agent-sandbox/types"
)

func RunTool(workspace string, call types.ToolCall) (output, errMsg string) {
	switch call.Name {
	case types.ToolRead:
		p := call.Arguments["path"]
		if p == "" {
			return "", "path required"
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(workspace, p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err.Error()
		}
		return string(b), ""

	case types.ToolWrite:
		p := call.Arguments["path"]
		if !filepath.IsAbs(p) {
			p = filepath.Join(workspace, p)
		}
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(call.Arguments["content"]), 0o644); err != nil {
			return "", err.Error()
		}
		return "written " + p, ""

	case types.ToolShell:
		cmd := exec.Command("sh", "-c", call.Arguments["command"])
		cmd.Dir = workspace
		b, err := cmd.CombinedOutput()
		if err != nil {
			return string(b), err.Error()
		}
		return string(b), ""

	case types.ToolGrep:
		pattern := shellQuote(call.Arguments["pattern"])
		cmd := exec.Command("sh", "-c", "grep -R --line-number -- "+pattern+" . 2>/dev/null | head -n 20")
		cmd.Dir = workspace
		b, _ := cmd.CombinedOutput()
		if len(b) == 0 {
			return "(no matches)", ""
		}
		return string(b), ""

	case types.ToolExaSearch:
		time.Sleep(80 * time.Millisecond)
		return "[fake exa] " + call.Arguments["query"], ""

	case types.ToolContext7:
		time.Sleep(60 * time.Millisecond)
		return "[fake context7] " + call.Arguments["library"] + ": " + call.Arguments["query"], ""

	default:
		return "", "unknown tool: " + string(call.Name)
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
