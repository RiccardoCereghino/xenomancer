package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// TestGHClientRequestAndResponse drives the GitHub Models client against a stub:
// it must POST to /chat/completions with a Bearer token, fold the system prompt
// into a leading system-role message, and read choices[0].message.content.
func TestGHClientRequestAndResponse(t *testing.T) {
	var body ghRequest
	var hdr http.Header
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr, path = r.Header, r.URL.Path
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &body)
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"walk to the gate"}}]}`))
	}))
	defer srv.Close()

	c := newGHClient("tok-123", "openai/gpt-4o-mini")
	c.endpoint = srv.URL + "/chat/completions"
	reply, err := c.complete("SYS", []message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if reply != "walk to the gate" {
		t.Errorf("reply = %q", reply)
	}
	if path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", path)
	}
	if hdr.Get("authorization") != "Bearer tok-123" {
		t.Errorf("authorization = %q", hdr.Get("authorization"))
	}
	if body.Model != "openai/gpt-4o-mini" {
		t.Errorf("model = %q", body.Model)
	}
	if len(body.Messages) != 2 || body.Messages[0].Role != "system" || body.Messages[0].Content != "SYS" {
		t.Errorf("messages = %+v, want a leading system message", body.Messages)
	}
}

// TestGHClientRetriesOn429 proves the free-tier rate-limit handling: a 429 with a
// short Retry-After is retried, and the subsequent success is returned.
func TestGHClientRetriesOn429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("retry-after", "0.05") // 50ms — keeps the test fast
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited","code":"RateLimitReached"}}`))
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"wait"}}]}`))
	}))
	defer srv.Close()

	c := newGHClient("t", "m")
	c.endpoint = srv.URL + "/chat/completions"
	reply, err := c.complete("s", []message{{Role: "user", Content: "x"}})
	if err != nil {
		t.Fatalf("complete after retry: %v", err)
	}
	if reply != "wait" || atomic.LoadInt32(&calls) != 2 {
		t.Errorf("reply=%q calls=%d, want wait/2", reply, calls)
	}
}

func TestNewCompleterSelectsProvider(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh")
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := newCompleter("github-models", ""); err != nil {
		t.Errorf("github-models with GITHUB_TOKEN should succeed: %v", err)
	}
	if _, err := newCompleter("anthropic", ""); err == nil {
		t.Error("anthropic without ANTHROPIC_API_KEY should fail")
	}
	if _, err := newCompleter("bogus", ""); err == nil {
		t.Error("unknown provider should fail")
	}
}

// TestGHAgentEndToEnd runs the compiled agent with --provider github-models
// against a stub Models server: read a packet → stub model → parser → canonical
// envelope, authenticated with a GITHUB_TOKEN. No real network, no Anthropic key.
func TestGHAgentEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess integration test in -short mode")
	}
	if runtime.GOOS == "windows" {
		t.Skip("uses go run; fine on unix CI")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"walk to the gate"}}]}`))
	}))
	defer srv.Close()

	cmd := exec.Command("go", "run", "./agent/llm", "--provider", "github-models")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(cmd.Environ(), "GITHUB_TOKEN=test", "GITHUB_MODELS_BASE_URL="+srv.URL)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start agent: %v", err)
	}
	go func() {
		defer stdin.Close()
		_, _ = io.WriteString(stdin, `{"v":1,"round":1,"narration":"A clearing.","result":{"ok":true,"rejections":[]}}`+"\n")
		sc := bufio.NewScanner(stdout)
		if sc.Scan() {
			var sub engine.RoundSubmission
			if err := json.Unmarshal(sc.Bytes(), &sub); err != nil {
				t.Errorf("agent emitted non-JSON: %q", sc.Text())
			} else if len(sub.Actions) == 0 || sub.Actions[0].Target != "gate" {
				t.Errorf("agent envelope = %+v, want a move to gate", sub)
			}
		} else {
			t.Error("agent did not emit an envelope")
		}
		_, _ = io.WriteString(stdin, `{"v":1,"outcome":"won"}`+"\n")
	}()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("agent exited with error: %v", err)
	}
}
