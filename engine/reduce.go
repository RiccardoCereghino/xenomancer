package engine

import (
	"encoding/json"
	"fmt"
)

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
	// The episode is over once an outcome is set (GDD §5.7, §7): a dead or
	// winning agent takes no further rounds. Terminality is sticky — no tick, no
	// events, no error — so a log with rounds past the terminal one folds to the
	// same state (ADR-000 D3).
	if s.Outcome != "" {
		return s, nil, nil
	}

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

	// If the agent is already grappled, an action on a held resource is either a
	// struggle (the one escape) or a resource conflict — the wolf has that limb
	// (GDD §5.6). The held-resource intercept runs before normal verb resolution.
	wasGrappled := s.grappled()
	var grappleHaz Hazard
	if wasGrappled {
		grappleHaz, _ = s.Content.hazardAt(s.Location)
	}
	struggled := false

	for i := 0; i < len(sub.Actions); i++ {
		a := sub.Actions[i]

		if !isValidResource(a.Resource) {
			events = append(events, rejection(ns, a, ReasonUnknownResource))
			continue
		}

		if wasGrappled && isHeld(ns.Holds, a.Resource) {
			// A perform on a held limb toward the grapple's struggle target is a
			// struggle; anything else claiming a held limb is a resource conflict,
			// rejected rather than silently dropped (GDD §5.6).
			if a.Verb == VerbPerform && a.Target == grappleHaz.Grapple.StruggleTarget {
				struggled = true
				events = append(events, Event{
					Kind:     EventStruggled,
					Resource: a.Resource,
					Verb:     a.Verb,
					Target:   a.Target,
					Tick:     ns.Tick,
					Round:    ns.Round,
				})
			} else {
				events = append(events, rejection(ns, a, ReasonResourceConflict))
			}
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

		case VerbInspect:
			// An inspect perceives an inspectable defined at the current
			// location (GDD §5.3). The pond reflection reveals the per-seed
			// eye color; a self-inspection in the clearing reveals species and
			// hair, never eyes (GDD §7). Anything else is an unknown target.
			insp, ok := ns.Content.inspectable(ns.Location, a.Target)
			if !ok {
				events = append(events, rejection(ns, a, ReasonUnknownTarget))
				continue
			}
			ev := Event{
				Kind:     EventObserved,
				Resource: a.Resource,
				Verb:     a.Verb,
				Target:   a.Target,
				Fact:     insp.Reveals,
				Tick:     ns.Tick,
				Round:    ns.Round,
			}
			// Only the eye-color fact carries a per-seed value; it is derived
			// on demand from the seed and never stored in State.
			if insp.Reveals == FactEyeColor {
				ev.Value = ns.Content.eyeColor(ns.Seed)
			}
			events = append(events, ev)

		case VerbTalk:
			// A talk addresses an NPC at the current location (GDD §5.4). The
			// gate guard is the first: it interrogates the agent's eye color and
			// judges the claim on the lethal path — no LLM, closed-palette
			// keyword matching only (GDD §5.4, P1). Any other target is unknown.
			npc, ok := ns.Content.npc(ns.Location, a.Target)
			if !ok {
				events = append(events, rejection(ns, a, ReasonUnknownTarget))
				continue
			}
			// The reply text rides in Args ({"say":"..."}); freeform parsing is
			// backlog 04, so the canonical claim is matched against the palette
			// directly. A missing/malformed Args decodes to an empty reply, which
			// names zero palette words → "speak plainly".
			var claimArgs struct {
				Say string `json:"say"`
			}
			if len(a.Args) > 0 {
				_ = json.Unmarshal(a.Args, &claimArgs)
			}
			claim, matches := matchClaim(ns.Content.EyeColorPalette, claimArgs.Say)
			if matches != 1 {
				// Zero or several palette words: costs one round, never kills.
				events = append(events, rejection(ns, a, ReasonUnclearClaim))
				continue
			}
			truth := ns.Content.eyeColor(ns.Seed)
			if claim == truth {
				ns.Outcome = OutcomeWon
				events = append(events, Event{
					Kind:  EventWon,
					Tick:  ns.Tick,
					Round: ns.Round + 1, // rounds elapsed after this round resolves
				})
			} else {
				ns.Outcome = OutcomeDied
				ns.Cause = CauseClaimWrong
				report := ns.Content.deathReport(ns.Seed, npc.ID, npc.Asks, claim, truth, ns.Round+1)
				events = append(events, Event{
					Kind:   EventDied,
					Tick:   ns.Tick,
					Round:  ns.Round + 1,
					Report: &report,
				})
			}

		default:
			events = append(events, rejection(ns, a, ReasonUnknownVerb))
		}

		// A win or a death ends the episode mid-round: no later action in this
		// submission resolves, though the round itself is still spent (one tick).
		if ns.Outcome != "" {
			break
		}
	}

	// Hazard phase (GDD §5.6). It runs only on a round that actually resolves —
	// the free-rejection conflict path returned above without ticking — so a free
	// rejection never advances a fuse. A terminal talk this round (win/death)
	// short-circuits it: the episode already ended.
	if ns.Outcome == "" {
		if wasGrappled {
			hazardResolveGrapple(&ns, grappleHaz, struggled, &events)
		} else {
			hazardAdvanceFuse(&ns, &events)
		}
	}

	// The round resolved: advance exactly one tick and one round.
	ns.Tick++
	ns.Round++
	return ns, events, nil
}

// hazardResolveGrapple advances a grapple by one round: it banks a struggle if
// one landed, spends a round of the escape window, and then either breaks free
// (enough struggles), kills (the window closed still held), or leaves the
// grapple running (GDD §5.6).
func hazardResolveGrapple(ns *State, haz Hazard, struggled bool, events *[]Event) {
	if struggled {
		ns.GrappleStruggles++
	}
	ns.GrappleRoundsLeft--

	if ns.GrappleStruggles >= haz.Grapple.StrugglesRequired {
		// Broke free: release the held limbs and reset the fuse from zero.
		ns.Holds = nil
		ns.Fuse = 0
		ns.GrappleRoundsLeft = 0
		ns.GrappleStruggles = 0
		*events = append(*events, Event{Kind: EventFreed, Tick: ns.Tick, Round: ns.Round})
		return
	}
	if ns.GrappleRoundsLeft == 0 {
		// The escape window closed still grappled: a fair, telegraphed death (P3).
		ns.Outcome = OutcomeDied
		ns.Cause = haz.Cause
		report := ns.Content.hazardDeathReport(ns.Seed, haz, ns.Round+1)
		*events = append(*events, Event{
			Kind:   EventDied,
			Tick:   ns.Tick,
			Round:  ns.Round + 1,
			Report: &report,
		})
	}
}

// hazardAdvanceFuse ticks the local doom clock for the round's ending location:
// out of the zone it resets, in the zone it climbs one, fires any telegraph rung
// at the new count, and springs the grapple when the fuse runs out (GDD §5.6).
func hazardAdvanceFuse(ns *State, events *[]Event) {
	haz, ok := ns.Content.hazardAt(ns.Location)
	if !ok {
		// Ended the round outside any hazard zone: the fuse pauses and resets
		// ("the wolf will not pursue you past the treeline").
		ns.Fuse = 0
		return
	}

	ns.Fuse++
	for i := 0; i < len(haz.Telegraphs); i++ {
		if haz.Telegraphs[i].At == ns.Fuse {
			*events = append(*events, Event{
				Kind:  EventTelegraph,
				Stage: haz.Telegraphs[i].Stage,
				Tick:  ns.Tick,
				Round: ns.Round,
			})
		}
	}
	if ns.Fuse >= haz.Fuse {
		// The fuse ran out: the hazard seizes the agent, sustaining a Hold on each
		// grappled resource (GDD §5.1). The escape window opens next round.
		ns.Holds = grappleHolds(haz, ns.Round+1)
		ns.GrappleRoundsLeft = haz.Grapple.Rounds
		ns.GrappleStruggles = 0
		*events = append(*events, Event{Kind: EventGrappled, Tick: ns.Tick, Round: ns.Round})
	}
}

// grappleHolds builds the sustained claims a sprung hazard places on the agent,
// one per grappled resource in content order (GDD §5.1), tagged with the hazard
// id and stamped with the round the grapple began.
func grappleHolds(haz Hazard, since uint64) []Hold {
	holds := make([]Hold, 0, len(haz.Grapple.Resources))
	for i := 0; i < len(haz.Grapple.Resources); i++ {
		holds = append(holds, Hold{Resource: haz.Grapple.Resources[i], Tag: haz.ID, Since: since})
	}
	return holds
}

// isHeld reports whether resource r is currently held, by ordered scan (no maps).
func isHeld(holds []Hold, r Resource) bool {
	for i := 0; i < len(holds); i++ {
		if holds[i].Resource == r {
			return true
		}
	}
	return false
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
