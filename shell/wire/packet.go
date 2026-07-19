// Package wire holds the shell-facing JSONL protocol types (ADR-000 D4) and the
// pure functions that render one resolved engine round into the packet sent to
// the agent — an observation packet, or a terminal win/death packet. These live
// in the shell, never in the engine: the reducer emits domain events; the
// packet and its narration are pure consumers of that event stream (ADR-000 D2).
// Both the stdio shell and the cmd/run harness host the same reducer, so they
// share this one copy — one narrator, one wire vocabulary.
package wire

import "github.com/RiccardoCereghino/xenomancer/engine"

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

// BuildPacket renders a resolved round — its post-Reduce State and the events it
// emitted — into the observation packet sent to the agent. It is pure: it reads
// only events and state, never engine internals (ADR-000 D2). The packet's Round
// is the next round the agent should submit (state.Round + 1); calling it with
// the Init state and no events therefore yields the opening packet for round 1.
func BuildPacket(state engine.State, events []engine.Event, nar Narration) ObservationPacket {
	var rejections []Rejection
	var observations []Observation
	var hz hazardBeats
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
		case engine.EventTelegraph:
			hz.telegraphStages = append(hz.telegraphStages, e.Stage)
		case engine.EventGrappled:
			hz.grappled = true
		case engine.EventStruggled:
			hz.struggled = true
		case engine.EventFreed:
			hz.freed = true
		}
	}

	return ObservationPacket{
		V:            engine.ProtocolVersion,
		Round:        int(state.Round) + 1,
		Narration:    nar.render(state.Location, moved, waited, observations, rejections, hz),
		Holds:        HoldsOrEmpty(state.Holds),
		Observations: observationsOrEmpty(observations),
		Result: Result{
			OK:         len(rejections) == 0,
			Rejections: valueOrEmpty(rejections),
		},
	}
}

// TerminalPacket returns the terminal packet for a resolved round and true when
// the round ended the episode (a died or won event), or (nil, false) otherwise.
// It scans events by kind only (no maps), so the packet order is fixed.
func TerminalPacket(state engine.State, events []engine.Event, nar Narration) (any, bool) {
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

// HoldsOrEmpty normalizes a nil Holds slice to an empty one so the wire packet
// serializes "holds":[] rather than "holds":null.
func HoldsOrEmpty(h []engine.Hold) []engine.Hold {
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
