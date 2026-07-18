package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// TestClientBuildsAndParsesRequest exercises the stdlib client against a stub
// Messages API server — no real network, so CI stays hermetic and zero-token.
func TestClientBuildsAndParsesRequest(t *testing.T) {
	var gotBody requestBody
	var gotHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"go to the gate"}],"stop_reason":"end_turn"}`))
	}))
	defer srv.Close()

	c := newClient("test-key", "claude-test")
	c.endpoint = srv.URL
	reply, err := c.complete("system", []message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if reply != "go to the gate" {
		t.Errorf("reply = %q, want %q", reply, "go to the gate")
	}
	if gotHeaders.Get("x-api-key") != "test-key" {
		t.Errorf("x-api-key header = %q", gotHeaders.Get("x-api-key"))
	}
	if gotHeaders.Get("anthropic-version") != anthropicVersion {
		t.Errorf("anthropic-version header = %q", gotHeaders.Get("anthropic-version"))
	}
	if gotBody.Model != "claude-test" || gotBody.System != "system" || len(gotBody.Messages) != 1 {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
}

func TestClientSurfacesRefusalAndErrors(t *testing.T) {
	refuse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[],"stop_reason":"refusal"}`))
	}))
	defer refuse.Close()
	c := newClient("k", "m")
	c.endpoint = refuse.URL
	if _, err := c.complete("s", []message{{Role: "user", Content: "x"}}); err == nil {
		t.Error("expected an error on refusal")
	}

	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"bad"}}`))
	}))
	defer fail.Close()
	c.endpoint = fail.URL
	if _, err := c.complete("s", []message{{Role: "user", Content: "x"}}); err == nil {
		t.Error("expected an error on 400")
	}
}

func TestRenderObservationAndFirstLine(t *testing.T) {
	var pkt incoming
	pkt.Narration = "You stand at the gate."
	got := renderObservation(pkt, `Your last reply "banana" was not a valid action.`)
	if !strings.Contains(got, "You stand at the gate.") || !strings.Contains(got, "banana") {
		t.Errorf("renderObservation missing content: %q", got)
	}
	if firstLine("  \n say brown to the guard \n because my eyes are brown") != "say brown to the guard" {
		t.Errorf("firstLine did not return the first non-empty trimmed line")
	}
}

func TestWaitSubmissionIsCanonicalWait(t *testing.T) {
	s := waitSubmission()
	if s.V != engine.ProtocolVersion || len(s.Actions) != 1 || s.Actions[0].Verb != engine.VerbWait {
		t.Fatalf("unexpected wait submission: %+v", s)
	}
}

// TestAgentEndToEnd runs the compiled agent against a stub Messages API server:
// it reads an observation packet, "asks the model" (the stub), routes the reply
// through the parser, and emits a canonical envelope — the full agent loop, no
// real network. Then it feeds a terminal packet and checks the report is written.
func TestAgentEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess integration test in -short mode")
	}
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell-free go run; fine on unix CI")
	}

	// The stub returns a freeform move, then a guard claim on the second call.
	replies := []string{"walk to the gate", "say green to the guard"}
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := replies[min(call, len(replies)-1)]
		call++
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"` + text + `"}],"stop_reason":"end_turn"}`))
	}))
	defer srv.Close()

	report := filepath.Join(t.TempDir(), "death.json")
	cmd := exec.Command("go", "run", "./agent/llm", "--report", report)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(cmd.Environ(), "ANTHROPIC_API_KEY=test", "ANTHROPIC_BASE_URL="+srv.URL)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start agent: %v", err)
	}

	// Feed one observation packet, read the emitted envelope, then a terminal
	// packet, then close — the agent stops on the terminal and writes the report.
	go func() {
		defer stdin.Close()
		_, _ = io.WriteString(stdin, `{"v":1,"round":1,"narration":"You are in a clearing.","result":{"ok":true,"rejections":[]}}`+"\n")
		sc := bufio.NewScanner(stdout)
		if sc.Scan() {
			var sub engine.RoundSubmission
			if err := json.Unmarshal(sc.Bytes(), &sub); err != nil {
				t.Errorf("agent emitted non-JSON: %q", sc.Text())
			} else if len(sub.Actions) == 0 || sub.Actions[0].Verb != engine.VerbPerform || sub.Actions[0].Target != "gate" {
				t.Errorf("agent envelope = %+v, want perform legs gate", sub)
			}
		} else {
			t.Error("agent did not emit an envelope")
		}
		_, _ = io.WriteString(stdin, `{"v":1,"outcome":"died","cause":"social.claim_wrong","epitaph":"gone"}`+"\n")
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("agent exited with error: %v", err)
	}
	data, err := readFile(report)
	if err != nil {
		t.Fatalf("report not written: %v", err)
	}
	if !strings.Contains(data, "social.claim_wrong") {
		t.Errorf("report = %q, want the death report", data)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}
