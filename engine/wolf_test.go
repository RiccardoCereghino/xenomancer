package engine

import "testing"

// wolfContent is a compact hazard graph built in-process: clearing <-> forest_path
// (the wolf zone) <-> gate, with the shipped wolf's thresholds (fuse 12,
// telegraphs at 6/9/11, a two-struggle/two-round grapple on both hands and legs).
// It lets the reducer tests exercise the whole ladder without the filesystem.
func wolfContent(t *testing.T) Content {
	t.Helper()
	c, err := ParseContent([]byte(`{
		"start_location": "clearing",
		"locations": [
			{"id": "clearing", "exits": ["forest_path"]},
			{"id": "forest_path", "exits": ["clearing", "gate"], "hazard": {
				"id": "wolf",
				"fuse": 12,
				"cause": "hazard.wolf",
				"telegraphs": [{"at": 6, "stage": 1}, {"at": 9, "stage": 2}, {"at": 11, "stage": 3}],
				"grapple": {"resources": ["hand_left", "hand_right", "legs"], "struggle_target": "struggle", "struggles_required": 2, "rounds": 2},
				"epitaphs": ["The woods kept their promise."]
			}},
			{"id": "gate", "exits": ["forest_path"]}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	return c
}

func waitRound() RoundSubmission {
	return sub(Action{Resource: "attention", Verb: VerbWait})
}

func struggleRound() RoundSubmission {
	return sub(Action{Resource: "legs", Verb: VerbPerform, Target: "struggle"})
}

// countKind counts events of a kind; stagesFired collects telegraph stages seen.
func countKind(events []Event, kind string) int {
	n := 0
	for i := 0; i < len(events); i++ {
		if events[i].Kind == kind {
			n++
		}
	}
	return n
}

func stagesFired(events []Event) []int {
	var stages []int
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventTelegraph {
			stages = append(stages, events[i].Stage)
		}
	}
	return stages
}

// Lingering in the wolf zone climbs the fuse: stages 1/2/3 fire at 6/9/11 and the
// grapple springs at 12, seizing both hands and legs (GDD §5.6, the DoD ladder).
func TestWolfTelegraphLadderAndGrapple(t *testing.T) {
	c := wolfContent(t)

	// Enter the zone (fuse 1), then wait 11 rounds: fuse climbs 2..12.
	steps := []RoundSubmission{move("forest_path")}
	for i := 0; i < 11; i++ {
		steps = append(steps, waitRound())
	}
	final, events := drive(t, Init(1, c), steps...)

	if got := stagesFired(events); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("telegraph stages = %v, want [1 2 3]", got)
	}
	if n := countKind(events, EventGrappled); n != 1 {
		t.Errorf("grappled events = %d, want 1", n)
	}
	if final.Fuse != 12 {
		t.Errorf("fuse = %d, want 12 at grapple", final.Fuse)
	}
	if !final.grappled() || final.GrappleRoundsLeft != 2 {
		t.Errorf("GrappleRoundsLeft = %d, want 2 (grappled)", final.GrappleRoundsLeft)
	}
	if len(final.Holds) != 3 {
		t.Fatalf("holds = %v, want 3 (both hands + legs)", final.Holds)
	}
	for i := 0; i < len(final.Holds); i++ {
		if final.Holds[i].Tag != "wolf" {
			t.Errorf("hold %d tag = %q, want wolf", i, final.Holds[i].Tag)
		}
	}
	if final.Outcome != "" {
		t.Errorf("outcome = %q, want ongoing (grapple is escapable)", final.Outcome)
	}
}

// grappleThenLinger drives Init -> grappled and returns the grappled state.
func grappledState(t *testing.T, c Content) State {
	t.Helper()
	steps := []RoundSubmission{move("forest_path")}
	for i := 0; i < 11; i++ {
		steps = append(steps, waitRound())
	}
	s, _ := drive(t, Init(1, c), steps...)
	if !s.grappled() {
		t.Fatalf("setup: expected grappled state, got GrappleRoundsLeft=%d", s.GrappleRoundsLeft)
	}
	return s
}

// Two struggles inside the two-round window break free: the holds release and the
// fuse resets, and the episode continues (GDD §5.6).
func TestWolfStruggleBreaksFree(t *testing.T) {
	c := wolfContent(t)
	s := grappledState(t, c)

	final, events := drive(t, s, struggleRound(), struggleRound())

	if n := countKind(events, EventStruggled); n != 2 {
		t.Errorf("struggled events = %d, want 2", n)
	}
	if n := countKind(events, EventFreed); n != 1 {
		t.Errorf("freed events = %d, want 1", n)
	}
	if final.grappled() {
		t.Error("still grappled after two struggles")
	}
	if len(final.Holds) != 0 {
		t.Errorf("holds = %v, want released", final.Holds)
	}
	if final.Fuse != 0 {
		t.Errorf("fuse = %d, want reset to 0 on break", final.Fuse)
	}
	if final.Outcome != "" {
		t.Errorf("outcome = %q, want ongoing after breaking free", final.Outcome)
	}
}

// Failing to break free within the window is a fair, telegraphed death: cause
// hazard.wolf, and the report lists every telegraph stage that fired (P3).
func TestWolfGrappleKills(t *testing.T) {
	c := wolfContent(t)
	s := grappledState(t, c)

	// Two rounds without a struggle (waiting uses attention, not a held limb).
	final, events := drive(t, s, waitRound(), waitRound())

	if final.Outcome != OutcomeDied {
		t.Fatalf("outcome = %q, want died", final.Outcome)
	}
	if final.Cause != CauseHazardWolf {
		t.Errorf("cause = %q, want %q", final.Cause, CauseHazardWolf)
	}
	if n := countKind(events, EventDied); n != 1 {
		t.Fatalf("died events = %d, want 1", n)
	}
	var report *DeathReport
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventDied {
			report = events[i].Report
		}
	}
	if report == nil {
		t.Fatal("died event carried no report")
	}
	if report.Cause != CauseHazardWolf {
		t.Errorf("report cause = %q, want %q", report.Cause, CauseHazardWolf)
	}
	want := []string{"stage 1", "stage 2", "stage 3"}
	if len(report.TelegraphsIgnored) != len(want) {
		t.Fatalf("telegraphs_ignored = %v, want %v", report.TelegraphsIgnored, want)
	}
	for i := range want {
		if report.TelegraphsIgnored[i] != want[i] {
			t.Errorf("telegraphs_ignored[%d] = %q, want %q", i, report.TelegraphsIgnored[i], want[i])
		}
	}
	if report.Detail.Hazard != "wolf" || report.Detail.Stage != 3 {
		t.Errorf("detail = %+v, want hazard wolf stage 3", report.Detail)
	}
	if report.Epitaph == "" {
		t.Error("hazard death report has no epitaph")
	}
}

// Leaving the zone before the grapple pauses and resets the fuse; re-entering
// starts the ladder over from stage 1 (GDD §5.6).
func TestWolfLeavingResetsFuse(t *testing.T) {
	c := wolfContent(t)

	// Enter and climb to fuse 5 (below the first telegraph), then step out.
	steps := []RoundSubmission{move("forest_path"), waitRound(), waitRound(), waitRound(), waitRound()}
	s, events := drive(t, Init(1, c), steps...)
	if s.Fuse != 5 {
		t.Fatalf("fuse = %d, want 5 before leaving", s.Fuse)
	}
	if len(stagesFired(events)) != 0 {
		t.Fatalf("telegraphs fired below threshold: %v", stagesFired(events))
	}

	// Step out to the clearing: the fuse pauses and resets.
	s, _ = drive(t, s, move("clearing"))
	if s.Fuse != 0 {
		t.Errorf("fuse = %d, want 0 after leaving the zone", s.Fuse)
	}

	// Re-enter and climb to 6: stage 1 fires again, from zero.
	reenter := []RoundSubmission{move("forest_path"), waitRound(), waitRound(), waitRound(), waitRound(), waitRound()}
	s, events = drive(t, s, reenter...)
	if s.Fuse != 6 {
		t.Errorf("fuse = %d, want 6 on re-entry", s.Fuse)
	}
	if got := stagesFired(events); len(got) != 1 || got[0] != 1 {
		t.Errorf("re-entry telegraphs = %v, want [1]", got)
	}
}

// A grapple locks the held limbs: any non-struggle claim on a held resource is a
// resource-conflict rejection, never silently dropped, and the agent stays held
// (GDD §5.6). The fuse never advances a grapple during a rejected free round —
// but a held-resource conflict still resolves the round, so the window ticks.
func TestWolfGrappleRejectsHeldResource(t *testing.T) {
	c := wolfContent(t)
	s := grappledState(t, c)

	// Try to flee: a move is a perform on legs, which the wolf holds.
	final, events := drive(t, s, sub(Action{Resource: "legs", Verb: VerbPerform, Target: "clearing"}))

	rejected := false
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventRejected && events[i].Reason == ReasonResourceConflict {
			rejected = true
		}
	}
	if !rejected {
		t.Error("expected a resource_conflict rejection for a move while grappled")
	}
	if final.Location != "forest_path" {
		t.Errorf("location = %q, want forest_path (cannot flee a grapple)", final.Location)
	}
	if !final.grappled() || final.GrappleRoundsLeft != 1 {
		t.Errorf("GrappleRoundsLeft = %d, want 1 (one window round spent)", final.GrappleRoundsLeft)
	}
}

// The whole hazard path is pure: replaying a grappled state's struggle yields an
// identical next state and does not mutate the input (ADR-000 D1).
func TestWolfReduceIsPure(t *testing.T) {
	c := wolfContent(t)
	s := grappledState(t, c)
	before := s.StateHash()

	a, _, err := Reduce(s, struggleRound())
	if err != nil {
		t.Fatalf("Reduce a: %v", err)
	}
	b, _, err := Reduce(s, struggleRound())
	if err != nil {
		t.Fatalf("Reduce b: %v", err)
	}
	if a.StateHash() != b.StateHash() {
		t.Error("Reduce is not pure: two identical calls diverged")
	}
	if s.StateHash() != before {
		t.Error("Reduce mutated its input state")
	}
}
