// Command run is the bidirectional runner and agent harness (ADR-000 D8,
// GDD §9). It hosts the pure engine reducer in-process and drives an agent
// subprocess, speaking the JSONL wire protocol in BOTH directions: it writes
// observation packets to the agent's stdin and reads round envelopes back from
// the agent's stdout (ADR-000 D4). At episode end it folds the collected action
// log into a replay file (ADR-000 D6).
//
// This is the harness the scripted (and later LLM) agents run on. It is a shell:
// it hosts the reducer unchanged and owns all I/O, narration, and the wall-clock
// deadline. The wall-clock lives here, never in the reducer — a missed deadline
// is resolved by injecting a canonical wait into the log, so replays stay exact
// (ADR-000 D8).
//
// Usage:
//
//	go run ./cmd/run --agent "go run ./agent/scripted/main" --seed 1 --out replay.json
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RiccardoCereghino/xenomancer/engine"
	"github.com/RiccardoCereghino/xenomancer/shell/wire"
)

func main() {
	agentCmd := flag.String("agent", "", "agent subprocess command, e.g. \"go run ./agent/scripted/main\" (required)")
	contentDir := flag.String("content", "content/zone1", "directory holding map.json and narration.json")
	seed := flag.Uint64("seed", 1, "run seed")
	bracket := flag.String("bracket", "scripted", "replay bracket label (ADR-000 D6)")
	out := flag.String("out", "", "replay output path; empty writes to stdout")
	deadline := flag.Duration("deadline", 0, "wall-clock deadline per agent response; 0 disables it (ADR-000 D8, off in tests; 60-90s live)")
	maxRounds := flag.Int("max-rounds", 1000, "safety cap on rounds per episode")
	flag.Parse()

	if strings.TrimSpace(*agentCmd) == "" {
		fatal("--agent is required")
	}

	content, nar := loadContent(*contentDir)

	// Launch the agent subprocess and wire its stdio to the harness: the agent
	// reads observation packets from its stdin and writes round envelopes to its
	// stdout. Its stderr is inherited so agent diagnostics reach the operator.
	argv := strings.Fields(*agentCmd)
	agent := exec.Command(argv[0], argv[1:]...)
	agent.Stderr = os.Stderr
	agentIn, err := agent.StdinPipe()
	if err != nil {
		fatal("agent stdin pipe: %v", err)
	}
	agentOut, err := agent.StdoutPipe()
	if err != nil {
		fatal("agent stdout pipe: %v", err)
	}
	if err := agent.Start(); err != nil {
		fatal("start agent %q: %v", *agentCmd, err)
	}

	log, err := runEpisode(*seed, content, nar, agentOut, agentIn, *deadline, *maxRounds)
	if err != nil {
		_ = agent.Process.Kill()
		_ = agent.Wait()
		fatal("episode: %v", err)
	}

	// The agent exits on EOF once the harness stops feeding it packets. If it is
	// still alive (episode ended on a terminal packet, cap, or timeout), close
	// its stdin and reap it.
	_ = agentIn.Close()
	if werr := agent.Wait(); werr != nil {
		// A killed or non-zero agent is not fatal to the replay: the log is the
		// proof object, and it is already complete.
		fmt.Fprintf(os.Stderr, "cmd/run: agent exited: %v\n", werr)
	}

	replay, _, err := engine.BuildReplay(*seed, content, log, *bracket)
	if err != nil {
		fatal("build replay: %v", err)
	}
	encoded, err := engine.Encode(replay)
	if err != nil {
		fatal("encode replay: %v", err)
	}
	// A trailing newline: conventional for a committed text file, and it makes a
	// regenerated golden byte-identical to the one on disk.
	encoded = append(encoded, '\n')
	if err := writeReplay(*out, encoded); err != nil {
		fatal("write replay: %v", err)
	}
}

// runEpisode drives one full episode: it sends the opening observation packet,
// then loops — read a round envelope from the agent, fold it, send the next
// packet — until a died/won event ends the episode, the agent closes its stdout,
// or the round cap is hit. It returns the canonical action log (the collected
// envelopes, including any injected on a deadline miss), which the caller folds
// into the replay.
//
// The loop is factored out of main so the wall-clock path can be exercised over
// in-process pipes without spawning a subprocess.
func runEpisode(seed uint64, content engine.Content, nar wire.Narration, agentOut io.Reader, agentIn io.Writer, deadline time.Duration, maxRounds int) ([]engine.RoundSubmission, error) {
	state := engine.Init(seed, content)
	enc := json.NewEncoder(agentIn)

	reader := newEnvelopeReader(agentOut)
	defer reader.stop()

	// The opening packet: round 1, from the Init state, no events. A bidirectional
	// agent needs an observation before it can act (an LLM agent especially); the
	// scripted agent reads and discards it.
	if err := enc.Encode(wire.BuildPacket(state, nil, nar)); err != nil {
		return nil, fmt.Errorf("send opening packet: %w", err)
	}

	var log []engine.RoundSubmission
	for round := 1; round <= maxRounds; round++ {
		env, status, err := reader.next(deadline)
		if err != nil {
			return nil, fmt.Errorf("read round %d: %w", round, err)
		}
		switch status {
		case readEOF:
			// The agent finished the episode by closing its stdout.
			return log, nil
		case readTimeout:
			// Deadline miss: inject a canonical wait for this round (ADR-000 D8).
			// The wall-clock never touches world state directly — only through
			// this canonical action, so the replay stays exact.
			env = canonicalWait(round)
			fmt.Fprintf(os.Stderr, "cmd/run: round %d missed the %s deadline; injecting canonical wait\n", round, deadline)
		}

		log = append(log, env)

		next, events, rerr := engine.Reduce(state, env)
		if rerr != nil {
			// A malformed envelope reached the reducer (ADR-000 D1). Fail loudly.
			return nil, fmt.Errorf("reduce round %d: %w", round, rerr)
		}
		state = next

		// A died/won event ends the episode: emit the terminal packet and stop.
		// The log up to and including this round is the complete proof object.
		if terminal, ok := wire.TerminalPacket(state, events, nar); ok {
			_ = enc.Encode(terminal) // best effort: the agent may already be gone
			return log, nil
		}

		if err := enc.Encode(wire.BuildPacket(state, events, nar)); err != nil {
			// The agent closed its stdin — it is done reading packets and the
			// episode is over. The log up to here is complete.
			return log, nil
		}
	}
	return log, nil
}

// readStatus reports how a per-round read resolved.
type readStatus int

const (
	readOK      readStatus = iota // an envelope was read
	readEOF                       // the agent closed its stdout
	readTimeout                   // the wall-clock deadline elapsed first
)

// envelopeReader reads round envelopes from the agent's stdout on a background
// goroutine so a per-round wall-clock deadline can preempt a blocking read
// (ADR-000 D8). With deadline == 0 the read simply blocks, which is the default
// (and the only) behavior in tests and the determinism CI.
type envelopeReader struct {
	lines   chan string
	scanErr chan error
}

func newEnvelopeReader(r io.Reader) *envelopeReader {
	er := &envelopeReader{
		lines:   make(chan string),
		scanErr: make(chan error, 1),
	}
	go func() {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			er.lines <- sc.Text()
		}
		er.scanErr <- sc.Err()
		close(er.lines)
	}()
	return er
}

// next returns the agent's next round envelope. With deadline > 0 it returns
// readTimeout if the deadline elapses first; with deadline <= 0 it blocks until
// a line arrives or the agent closes its stdout.
func (er *envelopeReader) next(deadline time.Duration) (engine.RoundSubmission, readStatus, error) {
	var timeout <-chan time.Time
	if deadline > 0 {
		timer := time.NewTimer(deadline)
		defer timer.Stop()
		timeout = timer.C
	}

	select {
	case line, ok := <-er.lines:
		if !ok {
			return engine.RoundSubmission{}, readEOF, <-er.scanErr
		}
		line = strings.TrimSpace(line)
		if line == "" {
			// Skip blank lines without consuming the round.
			return er.next(deadline)
		}
		var sub engine.RoundSubmission
		if err := json.Unmarshal([]byte(line), &sub); err != nil {
			return engine.RoundSubmission{}, readOK, fmt.Errorf("decode round envelope: %w", err)
		}
		return sub, readOK, nil
	case <-timeout:
		return engine.RoundSubmission{}, readTimeout, nil
	}
}

// stop releases the background scan goroutine if it is blocked handing off a
// line. It does not close the underlying reader; the caller owns that.
func (er *envelopeReader) stop() {
	// Drain any pending line so the goroutine can observe channel close on its
	// next Scan. Non-blocking: if nothing is pending, return immediately.
	select {
	case <-er.lines:
	default:
	}
}

// canonicalWait is the wait envelope the harness injects for a round the agent
// let lapse (ADR-000 D8). It claims attention and waits — a canonical, log-safe
// action; the round's number is cosmetic (the reducer counts rounds itself).
func canonicalWait(round int) engine.RoundSubmission {
	return engine.RoundSubmission{
		V:     engine.ProtocolVersion,
		Round: round,
		Actions: []engine.Action{
			{Resource: "attention", Verb: engine.VerbWait},
		},
	}
}

func loadContent(dir string) (engine.Content, wire.Narration) {
	mapBytes, err := os.ReadFile(filepath.Join(dir, "map.json"))
	if err != nil {
		fatal("read map.json: %v", err)
	}
	content, err := engine.ParseContent(mapBytes)
	if err != nil {
		fatal("parse content: %v", err)
	}
	narBytes, err := os.ReadFile(filepath.Join(dir, "narration.json"))
	if err != nil {
		fatal("read narration.json: %v", err)
	}
	nar, err := wire.LoadNarration(narBytes)
	if err != nil {
		fatal("parse narration: %v", err)
	}
	return content, nar
}

func writeReplay(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cmd/run: "+format+"\n", args...)
	os.Exit(1)
}
