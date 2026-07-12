// Command stdio is the JSONL shell that hosts the pure engine reducer
// (ADR-000 D8). It reads round envelopes from stdin, folds each through the
// engine, and writes an observation packet to stdout — one JSON object per
// line, every line carrying "v": 1 (ADR-000 D4).
//
// The shell owns all I/O and narration; the engine owns none. Narration is
// rendered from plain templates in content/zone1/narration.json (GDD §9).
//
// The wall-clock response deadline (GDD §9, ADR-000 D8) is NOT implemented this
// sprint — it is a shell concern and only a design seam here.
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
)

// ObservationPacket is the engine -> agent packet (ADR-000 D4):
//
//	{"v":1,"round":18,"narration":"...","holds":[...],"result":{"ok":true,"rejections":[]}}
type ObservationPacket struct {
	V            int           `json:"v"`
	Round        int           `json:"round"`
	Narration    string        `json:"narration"`
	Holds        []engine.Hold `json:"holds"`
	Observations []Observation `json:"observations"`
	Result       Result        `json:"result"`
}

// Observation mirrors an engine observed{fact,value} event for the agent. The
// eye-color value appears here (and in narration) ONLY on the round the pond
// reflection is inspected — never before (GDD §5.3).
type Observation struct {
	Fact  string `json:"fact"`
	Value string `json:"value,omitempty"`
}

// Result reports whether the round resolved cleanly and lists any rejections.
type Result struct {
	OK         bool        `json:"ok"`
	Rejections []Rejection `json:"rejections"`
}

// Rejection mirrors a rejected Event for the agent.
type Rejection struct {
	Reason   string `json:"reason"`
	Resource string `json:"resource,omitempty"`
	Verb     string `json:"verb,omitempty"`
	Target   string `json:"target,omitempty"`
}

// WonPacket is the terminal packet on a win (ADR-000 D4): the agent is inside
// the walls (GDD §7). Round carries the rounds elapsed.
type WonPacket struct {
	V         int    `json:"v"`
	Outcome   string `json:"outcome"`
	Round     uint64 `json:"round"`
	Narration string `json:"narration"`
}

// DeathPacket is the terminal packet on a death: the full GDD §5.7 death report
// (embedded, so its fields sit at the top level) plus the protocol "v" and the
// outcome. The epitaph is the death prose, so there is no separate narration.
type DeathPacket struct {
	V       int    `json:"v"`
	Outcome string `json:"outcome"`
	engine.DeathReport
}

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
	nar, err := loadNarration(narrationBytes)
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
		if terminal, ok := terminalPacket(state, events, nar); ok {
			if err := enc.Encode(terminal); err != nil {
				fatal("encode terminal packet: %v", err)
			}
			out.Flush()
			break
		}

		packet := buildPacket(state, events, nar)
		if err := enc.Encode(packet); err != nil {
			fatal("encode packet: %v", err)
		}
		out.Flush()
	}
	if err := in.Err(); err != nil {
		fatal("read stdin: %v", err)
	}
}

func buildPacket(state engine.State, events []engine.Event, nar narration) ObservationPacket {
	var rejections []Rejection
	var observations []Observation
	moved := false
	waited := false
	for i := 0; i < len(events); i++ {
		e := events[i]
		switch e.Kind {
		case engine.EventRejected:
			rejections = append(rejections, Rejection{
				Reason:   e.Reason,
				Resource: e.Resource,
				Verb:     e.Verb,
				Target:   e.Target,
			})
		case engine.EventObserved:
			observations = append(observations, Observation{Fact: e.Fact, Value: e.Value})
		case engine.EventMoved:
			moved = true
		case engine.EventWaited:
			waited = true
		}
	}

	return ObservationPacket{
		V:            engine.ProtocolVersion,
		Round:        int(state.Round) + 1,
		Narration:    nar.render(state.Location, moved, waited, observations, rejections),
		Holds:        holdsOrEmpty(state.Holds),
		Observations: observationsOrEmpty(observations),
		Result: Result{
			OK:         len(rejections) == 0,
			Rejections: valueOrEmpty(rejections),
		},
	}
}

// parseRejectionPacket is the "I don't understand" response for a freeform line
// with no dictionary hit. It reports a not_understood rejection and leaves the
// round unchanged: no engine.Reduce ran, so state.Round is still the pending
// round the agent may retry (GDD §5.2, P3). The reason lives in the shell, not
// the engine — the engine stays ignorant of the parser (quarantine).
func parseRejectionPacket(state engine.State) ObservationPacket {
	return ObservationPacket{
		V:            engine.ProtocolVersion,
		Round:        int(state.Round) + 1,
		Narration:    "I don't understand.",
		Holds:        holdsOrEmpty(state.Holds),
		Observations: []Observation{},
		Result: Result{
			OK:         false,
			Rejections: []Rejection{{Reason: "not_understood"}},
		},
	}
}

// terminalPacket returns the terminal packet for a resolved round and true when
// the round ended the episode (a died or won event), or (nil, false) otherwise.
// It scans events by kind only (no maps), so the packet order is fixed.
func terminalPacket(state engine.State, events []engine.Event, nar narration) (any, bool) {
	for i := 0; i < len(events); i++ {
		switch events[i].Kind {
		case engine.EventWon:
			return WonPacket{
				V:         engine.ProtocolVersion,
				Outcome:   engine.OutcomeWon,
				Round:     events[i].Round,
				Narration: nar.Won,
			}, true
		case engine.EventDied:
			// The reducer always attaches a Report to a died event; guard the
			// pointer so a malformed event can never nil-panic the shell.
			if events[i].Report == nil {
				continue
			}
			return DeathPacket{
				V:           engine.ProtocolVersion,
				Outcome:     engine.OutcomeDied,
				DeathReport: *events[i].Report,
			}, true
		}
	}
	return nil, false
}

func holdsOrEmpty(h []engine.Hold) []engine.Hold {
	if h == nil {
		return []engine.Hold{}
	}
	return h
}

func observationsOrEmpty(o []Observation) []Observation {
	if o == nil {
		return []Observation{}
	}
	return o
}

func valueOrEmpty(r []Rejection) []Rejection {
	if r == nil {
		return []Rejection{}
	}
	return r
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "shell/stdio: "+format+"\n", args...)
	os.Exit(1)
}
