package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Moksh-10/agent-sandbox/config"
)

// Runs one session against a local control plane (expects worker + control already up).
func main() {
	config.LoadDotEnvFromWD()
	token := os.Getenv("SB_AGENT_TOKEN")
	if token == "" {
		token = "dev-token"
	}
	base := getenv("CONTROL_PLANE_URL", "http://127.0.0.1:8080")

	if err := ping(base+"/health", token); err != nil {
		fmt.Fprintf(os.Stderr, "control plane not reachable at %s: %v\n", base, err)
		fmt.Fprintln(os.Stderr, "start: go run ./cmd/sandbox-worker  &&  go run ./cmd/control-plane")
		os.Exit(1)
	}

	sid := postSession(base, token, "play")
	fmt.Println("session:", sid)
	postRun(base, token, sid)

	for i := 0; i < 20; i++ {
		s := getSession(base, token, sid)
		fmt.Printf("status=%s sandbox=%s tools=%d/%d\n", s.Status, s.SandboxID, s.Completed, s.TotalTools)
		if s.Status == "completed" || s.Status == "failed" {
			return
		}
		time.Sleep(time.Second)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func ping(url, token string) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func postSession(base, token, msg string) string {
	body, _ := json.Marshal(map[string]string{"message": msg})
	req, _ := http.NewRequest(http.MethodPost, base+"/v1/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var s struct{ ID string }
	_ = json.NewDecoder(resp.Body).Decode(&s)
	return s.ID
}

func postRun(base, token, id string) {
	req, _ := http.NewRequest(http.MethodPost, base+"/v1/sessions/"+id+"/run", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
}

func getSession(base, token, id string) struct {
	Status, SandboxID     string
	Completed, TotalTools int
} {
	var out struct {
		Status, SandboxID     string
		Completed, TotalTools int
	}
	req, _ := http.NewRequest(http.MethodGet, base+"/v1/sessions/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out
}
