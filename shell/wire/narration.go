package wire

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Narration holds the plain templates loaded from narration.json (GDD §9). The
// narrator is a pure consumer of engine events and location state (ADR-000 D2);
// it lives in the shell, never in the engine.
type Narration struct {
	Locations    map[string]string `json:"locations"`
	OnEnter      string            `json:"on_enter"`
	Wait         string            `json:"wait"`
	Idle         string            `json:"idle"`
	Won          string            `json:"won"`
	// Telegraphs maps a hazard telegraph stage (as a decimal string key) to its
	// prose; the reducer emits only the structured stage, and the shell weaves the
	// text into the round's narration where a skimming agent will miss it
	// (GDD §5.6). Grappled/Struggle/Freed narrate the grapple ladder.
	Telegraphs   map[string]string `json:"telegraphs"`
	Grappled     string            `json:"grappled"`
	Struggle     string            `json:"struggle"`
	Freed        string            `json:"freed"`
	Observations map[string]string `json:"observations"`
	Rejections   map[string]string `json:"rejections"`
}

// hazardBeats carries the hazard events a resolved round emitted, so render can
// weave them into narration in a fixed order (GDD §5.6).
type hazardBeats struct {
	telegraphStages []int
	grappled        bool
	struggled       bool
	freed           bool
}

// LoadNarration parses the plain templates from narration.json bytes.
func LoadNarration(data []byte) (Narration, error) {
	var n Narration
	if err := json.Unmarshal(data, &n); err != nil {
		return Narration{}, err
	}
	return n, nil
}

// render composes deterministic narration for a resolved round. Templates are
// looked up by key (never iterated), so output order is fixed. Observations are
// rendered in event order; the eye-color line names the color via {value}.
func (n Narration) render(location string, moved, waited bool, observations []Observation, rejections []Rejection, hz hazardBeats) string {
	var parts []string

	if moved {
		line := strings.ReplaceAll(n.OnEnter, "{location}", location)
		line = strings.ReplaceAll(line, "{description}", n.Locations[location])
		parts = append(parts, line)
	} else if waited {
		parts = append(parts, n.Wait)
	}

	// Telegraphs are woven in right after the movement/wait line — ambient, easy
	// to skim past (GDD §5.6). Rendered in event (fuse) order.
	for i := 0; i < len(hz.telegraphStages); i++ {
		if msg, ok := n.Telegraphs[strconv.Itoa(hz.telegraphStages[i])]; ok {
			parts = append(parts, msg)
		}
	}

	for i := 0; i < len(observations); i++ {
		if msg, ok := n.Observations[observations[i].Fact]; ok {
			parts = append(parts, strings.ReplaceAll(msg, "{value}", observations[i].Value))
		}
	}

	// The grapple ladder: seized, then each struggle, then broken free.
	if hz.grappled && n.Grappled != "" {
		parts = append(parts, n.Grappled)
	}
	if hz.struggled && n.Struggle != "" {
		parts = append(parts, n.Struggle)
	}
	if hz.freed && n.Freed != "" {
		parts = append(parts, n.Freed)
	}

	for i := 0; i < len(rejections); i++ {
		if msg, ok := n.Rejections[rejections[i].Reason]; ok {
			parts = append(parts, msg)
		}
	}

	if len(parts) == 0 {
		if desc, ok := n.Locations[location]; ok {
			parts = append(parts, desc)
		} else {
			parts = append(parts, n.Idle)
		}
	}

	return strings.Join(parts, " ")
}
