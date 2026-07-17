package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/RiccardoCereghino/xenomancer/agent/scripted"
	"github.com/RiccardoCereghino/xenomancer/engine"
	"github.com/RiccardoCereghino/xenomancer/shell/wire"
)

const testSeed = 1

// loadZone1 reads the real zone-1 content pack and narration from disk (a test
// may do I/O; the engine may not).
func loadZone1(t *testing.T) (engine.Content, wire.Narration) {
	t.Helper()
	mapBytes, err := os.ReadFile("../../content/zone1/map.json")
	if err != nil {
		t.Fatalf("read map.json: %v", err)
	}
	c, err := engine.ParseContent(mapBytes)
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	narBytes, err := os.ReadFile("../../content/zone1/narration.json")
	if err != nil {
		t.Fatalf("read narration.json: %v", err)
	}
	nar, err := wire.LoadNarration(narBytes)
	if err != nil {
		t.Fatalf("LoadNarration: %v", err)
	}
	return c, nar
}

func goldenHash(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../agent/scripted/testdata/golden_replay.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	golden, err := engine.Decode(data)
	if err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	return golden.FinalStateHash
}

// fakeAgent reads observation packets and replies with the given envelopes, one
// per round, then closes its stdout — the in-process analogue of the scripted
// subprocess. It drives the harness over io.Pipes so the runner loop can be
// exercised without spawning a process.
func fakeAgent(pktIn io.ReadCloser, envOut io.WriteCloser, script []engine.RoundSubmission) {
	sc := bufio.NewScanner(pktIn)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	enc := json.NewEncoder(envOut)
	for i := 0; i < len(script); i++ {
		if !sc.Scan() { // wait for this round's observation packet
			break
		}
		if err := enc.Encode(script[i]); err != nil {
			break
		}
	}
	envOut.Close()             // signal episode end to the harness
	io.Copy(io.Discard, pktIn) // drain trailing packets so the harness never blocks
}

// TestRunEpisodeMatchesGolden drives the harness loop with the scripted script
// (the seed-1 win path) over in-process pipes and asserts the collected log
// folds into a replay that verifies and reproduces the committed golden's final
// state hash.
func TestRunEpisodeMatchesGolden(t *testing.T) {
	c, nar := loadZone1(t)

	pktR, pktW := io.Pipe() // harness writes packets to pktW; agent reads pktR
	envR, envW := io.Pipe() // agent writes envelopes to envW; harness reads envR
	defer pktR.Close()
	defer envR.Close()

	go fakeAgent(pktR, envW, scripted.Script())

	log, err := runEpisode(testSeed, c, nar, envR, pktW, 0, 100)
	if err != nil {
		t.Fatalf("runEpisode: %v", err)
	}
	if len(log) != len(scripted.Script()) {
		t.Fatalf("log length = %d, want %d", len(log), len(scripted.Script()))
	}

	replay, _, err := engine.BuildReplay(testSeed, c, log, "scripted")
	if err != nil {
		t.Fatalf("BuildReplay: %v", err)
	}
	ok, err := engine.Verify(replay, c)
	if err != nil || !ok {
		t.Fatalf("Verify: ok=%v err=%v", ok, err)
	}
	if replay.FinalStateHash != goldenHash(t) {
		t.Errorf("harness final hash %s != golden %s", replay.FinalStateHash, goldenHash(t))
	}
}

// TestHarnessDrivesScriptedSubprocess execs the real ported scripted-agent
// binary as an --agent subprocess and asserts the harness plays the full episode
// to the win and reproduces the golden — the end-to-end wiring the CI
// determinism fixture relies on.
func TestHarnessDrivesScriptedSubprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("skips `go run` subprocess build in -short mode")
	}
	c, nar := loadZone1(t)

	agent := exec.Command("go", "run", "github.com/RiccardoCereghino/xenomancer/agent/scripted/main")
	agent.Stderr = os.Stderr
	agentIn, err := agent.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	agentOut, err := agent.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := agent.Start(); err != nil {
		t.Fatalf("start agent: %v", err)
	}

	log, err := runEpisode(testSeed, c, nar, agentOut, agentIn, 0, 100)
	_ = agentIn.Close()
	_ = agent.Wait()
	if err != nil {
		t.Fatalf("runEpisode: %v", err)
	}

	replay, _, err := engine.BuildReplay(testSeed, c, log, "scripted")
	if err != nil {
		t.Fatalf("BuildReplay: %v", err)
	}
	ok, err := engine.Verify(replay, c)
	if err != nil || !ok {
		t.Fatalf("Verify: ok=%v err=%v", ok, err)
	}
	if replay.FinalStateHash != goldenHash(t) {
		t.Errorf("subprocess final hash %s != golden %s", replay.FinalStateHash, goldenHash(t))
	}
}

// TestWallClockInjectsCanonicalWait proves the wall-clock seam (ADR-000 D8): an
// agent that answers two rounds then hangs has canonical waits injected for the
// rounds it lets lapse, and the resulting replay still verifies. The deadline
// never touches world state — only the injected canonical action does — so the
// log remains an exact, replayable proof object.
func TestWallClockInjectsCanonicalWait(t *testing.T) {
	c, nar := loadZone1(t)

	pktR, pktW := io.Pipe()
	envR, envW := io.Pipe()
	defer pktW.Close()
	defer envR.Close()

	// A slow agent: answer rounds 1 and 2 (the two opening moves, non-terminal),
	// then hang forever (drain but never reply).
	go func() {
		sc := bufio.NewScanner(pktR)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		enc := json.NewEncoder(envW)
		script := scripted.Script()
		for i := 0; i < 2; i++ {
			if !sc.Scan() {
				break
			}
			_ = enc.Encode(script[i])
		}
		io.Copy(io.Discard, pktR) // hang: keep reading packets, never respond
	}()

	// deadline short, max-rounds 4: rounds 3 and 4 miss the deadline and inject waits.
	log, err := runEpisode(testSeed, c, nar, envR, pktW, 30*time.Millisecond, 4)
	if err != nil {
		t.Fatalf("runEpisode: %v", err)
	}
	if len(log) != 4 {
		t.Fatalf("log length = %d, want 4 (2 real + 2 injected)", len(log))
	}
	for _, r := range []int{2, 3} { // zero-based: the last two are injected waits
		if len(log[r].Actions) != 1 || log[r].Actions[0].Verb != engine.VerbWait {
			t.Errorf("round %d = %+v, want a single canonical wait", r+1, log[r].Actions)
		}
	}

	// The replay built from the log (real actions + injected waits) still verifies.
	replay, _, err := engine.BuildReplay(testSeed, c, log, "scripted")
	if err != nil {
		t.Fatalf("BuildReplay: %v", err)
	}
	ok, err := engine.Verify(replay, c)
	if err != nil || !ok {
		t.Fatalf("Verify: ok=%v err=%v", ok, err)
	}
}
