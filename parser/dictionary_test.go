package parser

import (
	"os"
	"testing"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// testDict is a small dictionary built in-process so the parser tests need no
// filesystem (mirrors engine's testContent). It covers all four canonical verbs
// and a couple of targets, enough to exercise every parse branch.
func testDict(t *testing.T) Dictionary {
	t.Helper()
	const data = `{
	  "version": 1,
	  "verbs": [
	    {"phrase": "walk",    "verb": "perform", "resource": "legs"},
	    {"phrase": "walk to", "verb": "perform", "resource": "legs"},
	    {"phrase": "go",      "verb": "perform", "resource": "legs"},
	    {"phrase": "look",    "verb": "inspect", "resource": "attention"},
	    {"phrase": "look at", "verb": "inspect", "resource": "attention"},
	    {"phrase": "wait",    "verb": "wait",    "resource": "attention"},
	    {"phrase": "say",     "verb": "talk",    "resource": "voice"}
	  ],
	  "targets": [
	    {"phrase": "gate",          "target": "gate"},
	    {"phrase": "the gate",      "target": "gate"},
	    {"phrase": "forest path",   "target": "forest_path"},
	    {"phrase": "my reflection", "target": "reflection"},
	    {"phrase": "self",          "target": "self"}
	  ]
	}`
	d, err := LoadDictionary([]byte(data))
	if err != nil {
		t.Fatalf("LoadDictionary: %v", err)
	}
	return d
}

// A known synonym maps to exactly the canonical action the dictionary names —
// resource, verb, and target (DoD: known synonyms map to correct canonical
// actions).
func TestParseKnownSynonyms(t *testing.T) {
	d := testDict(t)
	tests := []struct {
		name string
		line string
		want engine.Action
	}{
		{"single-word move", "walk gate", engine.Action{Resource: "legs", Verb: engine.VerbPerform, Target: "gate"}},
		{"multi-word verb", "walk to the gate", engine.Action{Resource: "legs", Verb: engine.VerbPerform, Target: "gate"}},
		{"move synonym", "go forest path", engine.Action{Resource: "legs", Verb: engine.VerbPerform, Target: "forest_path"}},
		{"inspect self", "look at self", engine.Action{Resource: "attention", Verb: engine.VerbInspect, Target: "self"}},
		{"inspect reflection", "look at my reflection", engine.Action{Resource: "attention", Verb: engine.VerbInspect, Target: "reflection"}},
		{"bare inspect", "look self", engine.Action{Resource: "attention", Verb: engine.VerbInspect, Target: "self"}},
		{"wait", "wait", engine.Action{Resource: "attention", Verb: engine.VerbWait}},
		{"talk", "say the gate", engine.Action{Resource: "voice", Verb: engine.VerbTalk, Target: "gate"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := d.Parse(tt.line)
			if !res.OK {
				t.Fatalf("Parse(%q) rejected, want action %+v", tt.line, tt.want)
			}
			if res.Submission.V != engine.ProtocolVersion {
				t.Errorf("Parse(%q) V = %d, want %d", tt.line, res.Submission.V, engine.ProtocolVersion)
			}
			if len(res.Submission.Actions) != 1 {
				t.Fatalf("Parse(%q) produced %d actions, want 1", tt.line, len(res.Submission.Actions))
			}
			got := res.Submission.Actions[0]
			if got.Resource != tt.want.Resource || got.Verb != tt.want.Verb || got.Target != tt.want.Target {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

// Case, punctuation, and whitespace variants of a known phrase still parse to
// the same canonical action: normalization is lossy in exactly the spec'd ways.
func TestParseNormalizationVariants(t *testing.T) {
	d := testDict(t)
	want := engine.Action{Resource: "legs", Verb: engine.VerbPerform, Target: "gate"}
	variants := []string{
		"walk to the gate",
		"WALK TO THE GATE",
		"  walk   to  the   gate  ",
		"Walk to the gate!",
		"walk, to the gate.",
		"walk\tto\tthe\tgate",
	}
	for _, v := range variants {
		res := d.Parse(v)
		if !res.OK || len(res.Submission.Actions) != 1 || !sameAction(res.Submission.Actions[0], want) {
			t.Errorf("Parse(%q) = %+v (ok=%v), want single %+v", v, res.Submission.Actions, res.OK, want)
		}
	}
}

// sameAction compares the fields the parser sets. Args is never populated by the
// parser (and json.RawMessage is not == comparable), so it is excluded.
func sameAction(a, b engine.Action) bool {
	return a.Resource == b.Resource && a.Verb == b.Verb && a.Target == b.Target
}

func TestNormalize(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Walk To The Gate", "walk to the gate"},
		{"  look   at  self  ", "look at self"},
		{"go, to the gate!", "go to the gate"},
		{"---", ""},
		{"", ""},
		{"look\tat\nself", "look at self"},
		{"gate2", "gate2"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// The load-bearing property (GDD P3, backlog §Property): NO freeform input can
// ever produce a state-affecting action that was not an exact dictionary hit.
// Every input below is NOT a (known verb phrase + valid target) pair, so each
// must yield a free rejection carrying zero actions — never a canonical action.
// A rejection can never advance a fuse, make a claim, or kill.
func TestMisparseNeverProducesAction(t *testing.T) {
	d := testDict(t)

	// A deterministic corpus of non-hits, spanning every way a line can fail:
	// gibberish, unknown verb + known target, known verb + unknown target,
	// verb-only (needs a target), wait with a trailing target, punctuation-only,
	// empty, and near-misses of known phrases.
	knownVerbs := []string{"walk", "walk to", "go", "look", "look at", "say"}
	knownTargets := []string{"gate", "the gate", "forest path", "my reflection", "self"}
	gibberishVerbs := []string{"frobnicate", "teleport", "yeet", "sudo", "walkto", "gowalk", "loo", "sayy"}
	gibberishTargets := []string{"moon", "the void", "dragon", "nowhere", "gates", "reflectio", "forestpath"}

	var corpus []string
	corpus = append(corpus,
		"", "   ", "!!!", "...", "42", "the the the",
		"wait now", "wait gate", "wait please", // wait takes no target
		"walk", "go", "look at", "walk to", "say", // verb with no/insufficient target
		"gate", "self", "my reflection", // target with no verb
		"attack the gate", "open gate", "run to the gate", // unknown verb + known target
	)
	// unknown verb x known target
	for _, v := range gibberishVerbs {
		for _, tg := range knownTargets {
			corpus = append(corpus, v+" "+tg)
		}
	}
	// known verb x unknown target
	for _, v := range knownVerbs {
		if v == "wait" {
			continue
		}
		for _, tg := range gibberishTargets {
			corpus = append(corpus, v+" "+tg)
		}
	}
	// gibberish x gibberish
	for _, v := range gibberishVerbs {
		for _, tg := range gibberishTargets {
			corpus = append(corpus, v+" "+tg)
		}
	}

	for _, line := range corpus {
		res := d.Parse(line)
		if res.OK {
			t.Errorf("Parse(%q) produced a canonical action, want free rejection", line)
		}
		if len(res.Submission.Actions) != 0 {
			t.Errorf("Parse(%q) rejection carried %d actions, want 0", line, len(res.Submission.Actions))
		}
		if !res.OK && res.Message != RejectMessage {
			t.Errorf("Parse(%q) message = %q, want %q", line, res.Message, RejectMessage)
		}
	}
}

// Parse is a pure function of the input: the same line twice yields an identical
// result (mirrors engine's TestReduceIsPure). Determinism is the whole point of
// lookup-only parsing.
func TestParseIsDeterministic(t *testing.T) {
	d := testDict(t)
	lines := []string{"walk to the gate", "look at my reflection", "wait", "frobnicate the moon", ""}
	for _, line := range lines {
		a := d.Parse(line)
		b := d.Parse(line)
		if a.OK != b.OK || a.Message != b.Message {
			t.Errorf("Parse(%q) not deterministic: %+v vs %+v", line, a, b)
		}
		if len(a.Submission.Actions) != len(b.Submission.Actions) {
			t.Errorf("Parse(%q) action count differs across calls", line)
		}
		for i := range a.Submission.Actions {
			if !sameAction(a.Submission.Actions[i], b.Submission.Actions[i]) {
				t.Errorf("Parse(%q) action %d differs across calls", line, i)
			}
		}
	}
}

// LoadDictionary rejects malformed data: non-canonical verbs, empty fields, and
// duplicate phrases are authoring errors caught at load, not at parse.
func TestLoadDictionaryValidation(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"non-canonical verb", `{"verbs":[{"phrase":"jump","verb":"leap","resource":"legs"}]}`},
		{"empty resource", `{"verbs":[{"phrase":"walk","verb":"perform","resource":""}]}`},
		{"empty verb phrase", `{"verbs":[{"phrase":"  ","verb":"wait","resource":"attention"}]}`},
		{"duplicate verb phrase", `{"verbs":[{"phrase":"walk","verb":"perform","resource":"legs"},{"phrase":"walk","verb":"wait","resource":"attention"}]}`},
		{"empty target id", `{"targets":[{"phrase":"gate","target":""}]}`},
		{"duplicate target phrase", `{"targets":[{"phrase":"gate","target":"gate"},{"phrase":"gate","target":"clearing"}]}`},
		{"malformed json", `{"verbs":`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := LoadDictionary([]byte(tt.data)); err == nil {
				t.Errorf("LoadDictionary(%s) = nil error, want failure", tt.name)
			}
		})
	}
}

// The shipped zone-1 dictionary loads and resolves the canonical walkthrough
// phrases to the ids the engine expects (guards the data file, not just code).
func TestShippedDictionaryLoads(t *testing.T) {
	d := loadShipped(t)
	cases := []struct {
		line   string
		verb   string
		target string
	}{
		{"look at myself", engine.VerbInspect, "self"},
		{"walk to the forest", engine.VerbPerform, "forest_path"},
		{"walk to the pond", engine.VerbPerform, "still_pond"},
		{"look at my reflection", engine.VerbInspect, "reflection"},
		{"walk to the gate", engine.VerbPerform, "gate"},
		{"wait", engine.VerbWait, ""},
	}
	for _, c := range cases {
		res := d.Parse(c.line)
		if !res.OK || len(res.Submission.Actions) != 1 {
			t.Fatalf("Parse(%q) not accepted with one action: %+v", c.line, res)
		}
		got := res.Submission.Actions[0]
		if got.Verb != c.verb || got.Target != c.target {
			t.Errorf("Parse(%q) = verb %q target %q, want verb %q target %q", c.line, got.Verb, got.Target, c.verb, c.target)
		}
	}
}

// loadShipped reads the real zone-1 dictionary from disk (a test may do I/O;
// the parser may not).
func loadShipped(t *testing.T) Dictionary {
	t.Helper()
	data, err := os.ReadFile("../content/zone1/dictionary.json")
	if err != nil {
		t.Fatalf("read dictionary.json: %v", err)
	}
	d, err := LoadDictionary(data)
	if err != nil {
		t.Fatalf("LoadDictionary(shipped): %v", err)
	}
	return d
}
