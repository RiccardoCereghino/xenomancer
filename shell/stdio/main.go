// Command stdio is the JSONL shell that hosts the pure engine reducer
// (ADR-000 D8). It reads round envelopes from stdin, folds each through the
// engine, and writes an observation packet to stdout — one JSON object per
// line, every line carrying "v": 1 (ADR-000 D4).
//
// The shell owns all I/O and narration; the engine owns none. The wire types
// and the packet builders live in shell/wire, shared with the cmd/run harness.
// Narration is rendered from plain templates in content/zone1/narration.json
// (GDD §9).
//
// The wall-clock response deadline (GDD §9, ADR-000 D8) is NOT a stdio concern:
// it lives in the cmd/run harness, which drives an agent subprocess. This shell
// is envelope-first and has no deadline.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RiccardoCereghino/xenomancer/engine"
	"github.com/RiccardoCereghino/xenomancer/parser"
	"github.com/RiccardoCereghino/xenomancer/shell/wire"
)

func main() {
	contentDir := flag.String("content", "content/zone1", "directory holding map.json and narration.json")
	seed := flag.Uint64("seed", 1, "run seed")
	flag.Parse()

	mapBytes, err := os.ReadFile(filepath.Join(*contentDir, "map.json"))
	if err != nil {
		fatal("read map.json: %v", err)
	}
	content, err := engine.ParseContent(mapBytes)
	if err != nil {
		fatal("parse content: %v", err)
	}

	narrationBytes, err := os.ReadFile(filepath.Join(*contentDir, "narration.json"))
	if err != nil {
		fatal("read narration.json: %v", err)
	}
	nar, err := wire.LoadNarration(narrationBytes)
	if err != nil {
		fatal("parse narration: %v", err)
	}

	state := engine.Init(*seed, content)

	// The freeform parser is quarantined outside the engine (ADR-000 D3/D4): it
	// maps freeform lines to canonical submissions, and only canonical actions
	// ever reach engine.Reduce and the log. A parse rejection never reaches the
	// engine, so a misparse costs no tick (GDD §5.2, P3).
	p := parser.New()

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)

	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}

		// Each line is either a canonical JSON round envelope or a freeform
		// line. A leading '{' marks the canonical envelope (ADR-000 D4);
		// anything else is freeform and goes through the parser.
		var sub engine.RoundSubmission
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &sub); err != nil {
				fatal("decode round envelope: %v", err)
			}
		} else {
			parsed, ok := p.Parse(line)
			if !ok {
				// Free rejection: no tick, no state change, nothing logged. The
				// rejected line is the dictionary's backlog (GDD §13); a simple
				// stderr note stands in until rejection telemetry (ADR-003).
				fmt.Fprintf(os.Stderr, "shell/stdio: parse reject: %q\n", line)
				if err := enc.Encode(parseRejectionPacket(state)); err != nil {
					fatal("encode packet: %v", err)
				}
				out.Flush()
				continue
			}
			sub = parsed
		}

		next, events, err := engine.Reduce(state, sub)
		if err != nil {
			// error == programmer misuse (ADR-000 D1); a malformed envelope
			// reached the reducer. Fail loudly rather than pretend a round.
			fatal("reduce: %v", err)
		}
		state = next

		// A died/won event ends the episode: emit the terminal packet (the death
		// report or the win) instead of an observation packet and stop reading —
		// no further rounds resolve (ADR-000 D4).
		if terminal, ok := wire.TerminalPacket(state, events, nar); ok {
			if err := enc.Encode(terminal); err != nil {
				fatal("encode terminal packet: %v", err)
			}
			out.Flush()
			break
		}

		packet := wire.BuildPacket(state, events, nar)
		if err := enc.Encode(packet); err != nil {
			fatal("encode packet: %v", err)
		}
		out.Flush()
	}
	if err := in.Err(); err != nil {
		fatal("read stdin: %v", err)
	}
}

// parseRejectionPacket is the "I don't understand" response for a freeform line
// with no dictionary hit. It reports a not_understood rejection and leaves the
// round unchanged: no engine.Reduce ran, so state.Round is still the pending
// round the agent may retry (GDD §5.2, P3). The reason lives in the shell, not
// the engine — the engine stays ignorant of the parser (quarantine).
func parseRejectionPacket(state engine.State) wire.ObservationPacket {
	return wire.ObservationPacket{
		V:            engine.ProtocolVersion,
		Round:        int(state.Round) + 1,
		Narration:    "I don't understand.",
		Holds:        wire.HoldsOrEmpty(state.Holds),
		Observations: []wire.Observation{},
		Result: wire.Result{
			OK:         false,
			Rejections: []wire.Rejection{{Reason: "not_understood"}},
		},
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "shell/stdio: "+format+"\n", args...)
	os.Exit(1)
}
