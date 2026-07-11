package engine

import "fmt"

// Init produces the initial State for a run from a seed and a content pack
// (ADR-000 D1). It is pure: same inputs, same State, forever.
func Init(seed uint64, content Content) State {
	return State{
		Seed:     seed,
		Tick:     0,
		Round:    0,
		Location: content.StartLocation,
		Holds:    nil,
		Content:  content,
	}
}

// Reduce folds one round submission into the next State, emitting an ordered
// slice of Events (ADR-000 D1/D2).
//
// The returned error is reserved for programmer misuse — a structurally
// malformed submission the shell/protocol layer should never have produced.
// Every in-game refusal (unknown verb, unknown target, illegal move, resource
// conflict) is reported as a rejected Event, never an error (ADR-000 D1).
//
// Round resolution model (sprint 0):
//   - A resource conflict (two claims on the same resource) rejects the whole
//     round before resolution: no tick, no round advance, resubmittable
//     (GDD §5.1).
//   - Otherwise every action resolves in order and the round advances the tick
//     and round counters exactly once — even if every action was rejected, the
//     round was still spent.
func Reduce(s State, sub RoundSubmission) (State, []Event, error) {
	// D1: programmer-misuse guard. Every action must name a resource and a
	// verb; an empty field is a malformed struct, not an in-game choice.
	for i := 0; i < len(sub.Actions); i++ {
		if sub.Actions[i].Resource == "" || sub.Actions[i].Verb == "" {
			return s, nil, fmt.Errorf("engine: malformed action %d: empty resource or verb", i)
		}
	}

	var events []Event

	// Detect resource conflicts by ordered scan (no maps, ADR-000 D5.2). An
	// action conflicts if an earlier action in the same submission already
	// claimed its resource.
	conflict := false
	for i := 0; i < len(sub.Actions); i++ {
		if duplicatesEarlierResource(sub.Actions, i) {
			conflict = true
			break
		}
	}
	if conflict {
		for i := 0; i < len(sub.Actions); i++ {
			if duplicatesEarlierResource(sub.Actions, i) {
				events = append(events, rejection(s, sub.Actions[i], ReasonResourceConflict))
			}
		}
		// No tick, no round advance: the round is handed back to the agent.
		return s, events, nil
	}

	ns := s
	for i := 0; i < len(sub.Actions); i++ {
		a := sub.Actions[i]

		if !isValidResource(a.Resource) {
			events = append(events, rejection(ns, a, ReasonUnknownResource))
			continue
		}

		switch a.Verb {
		case VerbWait:
			events = append(events, Event{
				Kind:     EventWaited,
				Resource: a.Resource,
				Tick:     ns.Tick,
				Round:    ns.Round,
			})

		case VerbPerform:
			// Movement is the only perform implemented this sprint, and only
			// on legs (GDD §5.2). Any other perform has no target to act on.
			if a.Resource != "legs" {
				events = append(events, rejection(ns, a, ReasonUnknownTarget))
				continue
			}
			if ns.Content.isExit(ns.Location, a.Target) {
				ns.Location = a.Target
				events = append(events, Event{
					Kind:  EventMoved,
					To:    a.Target,
					Tick:  ns.Tick,
					Round: ns.Round,
				})
			} else {
				events = append(events, rejection(ns, a, ReasonIllegalMove))
			}

		case VerbInspect, VerbTalk:
			// No inspectable objects and no NPCs exist in zone 1 this sprint
			// (the pond and the guard arrive in sprint 1). Every such target
			// is therefore unknown.
			events = append(events, rejection(ns, a, ReasonUnknownTarget))

		default:
			events = append(events, rejection(ns, a, ReasonUnknownVerb))
		}
	}

	// The round resolved: advance exactly one tick and one round.
	ns.Tick++
	ns.Round++
	return ns, events, nil
}

// duplicatesEarlierResource reports whether action i claims a resource already
// claimed by an earlier action in the slice.
func duplicatesEarlierResource(actions []Action, i int) bool {
	for j := 0; j < i; j++ {
		if actions[j].Resource == actions[i].Resource {
			return true
		}
	}
	return false
}

func rejection(s State, a Action, reason string) Event {
	return Event{
		Kind:     EventRejected,
		Reason:   reason,
		Resource: a.Resource,
		Verb:     a.Verb,
		Target:   a.Target,
		Tick:     s.Tick,
		Round:    s.Round,
	}
}
