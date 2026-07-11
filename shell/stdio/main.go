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
		var sub engine.RoundSubmission
		if err := json.Unmarshal([]byte(line), &sub); err != nil {
			fatal("decode round envelope: %v", err)
		}

		next, events, err := engine.Reduce(state, sub)
		if err != nil {
			// error == programmer misuse (ADR-000 D1); a malformed envelope
			// reached the reducer. Fail loudly rather than pretend a round.
			fatal("reduce: %v", err)
		}
		state = next

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

	holds := state.Holds
	if holds == nil {
		holds = []engine.Hold{}
	}

	return ObservationPacket{
		V:            engine.ProtocolVersion,
		Round:        int(state.Round) + 1,
		Narration:    nar.render(state.Location, moved, waited, observations, rejections),
		Holds:        holds,
		Observations: observationsOrEmpty(observations),
		Result: Result{
			OK:         len(rejections) == 0,
			Rejections: valueOrEmpty(rejections),
		},
	}
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
