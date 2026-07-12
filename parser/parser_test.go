package parser

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// TestKnownSynonymsMapToCanonicalActions pins the happy path: known freeform
// phrasings resolve to the exact canonical action, including the resource the
// dictionary carries (GDD §5.2). This is the "known synonyms map correctly" DoD.
func TestKnownSynonymsMapToCanonicalActions(t *testing.T) {
	p := New()
	cases := []struct {
		in   string
		want engine.Action
	}{
		{"walk to the gate", engine.Action{Resource: "legs", Verb: engine.VerbPerform, Target: "gate"}},
		{"go to the still pond", engine.Action{Resource: "legs", Verb: engine.VerbPerform, Target: "still_pond"}},
		{"move north along the forest path", engine.Action{Resource: "legs", Verb: engine.VerbPerform, Target: "forest_path"}},
		{"look at my reflection", engine.Action{Resource: "attention", Verb: engine.VerbInspect, Target: "reflection"}},
		{"Examine myself.", engine.Action{Resource: "attention", Verb: engine.VerbInspect, Target: "self"}},
		{"inspect the pond", engine.Action{Resource: "attention", Verb: engine.VerbInspect, Target: "still_pond"}},
		{"wait", engine.Action{Resource: "attention", Verb: engine.VerbWait, Target: ""}},
		{"  WAIT here a moment  ", engine.Action{Resource: "attention", Verb: engine.VerbWait, Target: ""}},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			sub, ok := p.Parse(tt.in)
			if !ok {
				t.Fatalf("Parse(%q) rejected; want %+v", tt.in, tt.want)
			}
			if sub.V != engine.ProtocolVersion {
				t.Errorf("Parse(%q) V = %d, want %d", tt.in, sub.V, engine.ProtocolVersion)
			}
			if len(sub.Actions) != 1 {
				t.Fatalf("Parse(%q) produced %d actions, want 1", tt.in, len(sub.Actions))
			}
			if got := sub.Actions[0]; got.Resource != tt.want.Resource || got.Verb != tt.want.Verb || got.Target != tt.want.Target {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

// TestUnknownInputIsAFreeRejection covers the "rejections are free" side: input
// with no dictionary hit returns ok == false and never a canonical action, so it
// never reaches the engine and costs nothing (GDD P3, §13).
func TestUnknownInputIsAFreeRejection(t *testing.T) {
	p := New()
	rejects := []string{
		"",
		"   ",
		"!?.,",
		"xyzzy",
		"frobnicate the gate", // unknown verb, known target
		"waiter",              // must NOT hit the "wait" verb (whole-token match)
		"gate",                // target with no verb
		"move to the moon",    // known verb, unknown target
		"look",                // inspect with no target
		"go",                  // perform with no target
		"speak to the guard",  // talk with no known target this sprint
	}
	for _, in := range rejects {
		t.Run(in, func(t *testing.T) {
			sub, ok := p.Parse(in)
			if ok {
				t.Errorf("Parse(%q) accepted as %+v; want a free rejection", in, sub.Actions)
			}
		})
	}
}

// TestParseIsDeterministic: the same input yields the same result every call —
// no randomness, no fuzzy matching (GDD §5.2).
func TestParseIsDeterministic(t *testing.T) {
	p := New()
	inputs := []string{"walk to the gate", "look at the pond", "wait", "nonsense here", ""}
	for _, in := range inputs {
		first, ok1 := p.Parse(in)
		for i := 0; i < 5; i++ {
			got, ok := p.Parse(in)
			if ok != ok1 || !reflect.DeepEqual(got, first) {
				t.Errorf("Parse(%q) not deterministic: %+v/%v vs %+v/%v", in, got, ok, first, ok1)
			}
		}
	}
}

// hasVerbHit reports whether any token window is an exact verb-dictionary key
// whose entry matches want — an INDEPENDENT re-derivation of the hit, so the
// property test does not merely trust the parser's own matcher.
func (p *Parser) hasVerbHit(tokens []string, want VerbEntry) bool {
	for length := len(tokens); length >= 1; length-- {
		for start := 0; start+length <= len(tokens); start++ {
			phrase := strings.Join(tokens[start:start+length], " ")
			if e, ok := p.dict.Verbs[phrase]; ok && e == want {
				return true
			}
		}
	}
	return false
}

// hasTargetHit reports whether any token window is an exact target-dictionary key
// mapping to id.
func (p *Parser) hasTargetHit(tokens []string, id string) bool {
	for length := len(tokens); length >= 1; length-- {
		for start := 0; start+length <= len(tokens); start++ {
			phrase := strings.Join(tokens[start:start+length], " ")
			if got, ok := p.dict.Targets[phrase]; ok && got == id {
				return true
			}
		}
	}
	return false
}

// invariantHolds is the load-bearing check: for ANY input, if Parse accepts it,
// then every field of the produced action was an exact dictionary hit on the
// normalized input. Equivalently, no input lacking a dictionary hit can ever
// yield a state-affecting canonical action (GDD P3). Returns true when the
// invariant holds for this input.
func invariantHolds(p *Parser, in string) bool {
	sub, ok := p.Parse(in)
	if !ok {
		// A rejection is always safe: nothing reaches the engine.
		return true
	}
	if len(sub.Actions) != 1 {
		return false
	}
	a := sub.Actions[0]
	tokens := normalize(in)
	// The verb+resource must trace back to an exact verb-dictionary hit.
	if !p.hasVerbHit(tokens, VerbEntry{Verb: a.Verb, Resource: a.Resource}) {
		return false
	}
	// Non-wait actions must carry a target that traces back to an exact hit;
	// wait must carry no target.
	if a.Verb == engine.VerbWait {
		return a.Target == ""
	}
	return a.Target != "" && p.hasTargetHit(tokens, a.Target)
}

// fuzzInput is a testing/quick generator that biases toward adversarial near-
// hits: strings assembled from dictionary words, decoy words, and random tokens.
// Purely random strings almost never touch the dictionary; mixing in real
// vocabulary is what actually exercises the accept path of the invariant.
type fuzzInput string

var fuzzVocab = []string{
	// real verbs and targets
	"walk", "go", "move", "look", "inspect", "examine", "wait", "talk", "say",
	"gate", "pond", "still", "forest", "path", "reflection", "self", "myself", "water", "clearing",
	// decoys / noise that must never resolve
	"waiter", "walked", "gates", "the", "to", "a", "north", "moon", "guard", "xyzzy", "frobnicate",
}

func (fuzzInput) Generate(r *rand.Rand, size int) reflect.Value {
	n := r.Intn(6)
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if r.Intn(3) == 0 {
			// occasional raw random token, including punctuation and case
			b := make([]byte, 1+r.Intn(5))
			for j := range b {
				b[j] = byte(0x20 + r.Intn(0x5f))
			}
			parts = append(parts, string(b))
		} else {
			parts = append(parts, fuzzVocab[r.Intn(len(fuzzVocab))])
		}
	}
	return reflect.ValueOf(fuzzInput(strings.Join(parts, " ")))
}

// TestMisparseNeverProducesUndictionariedAction is the property test the issue
// names: across many generated inputs, no input ever yields a canonical action
// that was not an exact dictionary hit. A universal invariant can only be
// falsified, never spuriously fail, so a randomized generator is safe here.
func TestMisparseNeverProducesUndictionariedAction(t *testing.T) {
	p := New()
	prop := func(in fuzzInput) bool { return invariantHolds(p, string(in)) }
	if err := quick.Check(prop, &quick.Config{MaxCount: 20000}); err != nil {
		t.Fatalf("misparse-never-kills invariant violated: %v", err)
	}
	// Also sweep raw arbitrary strings (quick's default string generator) to
	// cover inputs the biased generator would rarely produce.
	rawProp := func(in string) bool { return invariantHolds(p, in) }
	if err := quick.Check(rawProp, &quick.Config{MaxCount: 20000}); err != nil {
		t.Fatalf("misparse-never-kills invariant violated on raw strings: %v", err)
	}
}

// FuzzParse pins the same invariant under Go's native fuzzing. Its seed corpus
// runs under `go test`; `go test -fuzz` explores further. The parser must never
// panic and must never break the invariant, whatever bytes arrive.
func FuzzParse(f *testing.F) {
	for _, s := range []string{"", "wait", "walk to the gate", "waiter", "look at reflection", "??!", "go go go"} {
		f.Add(s)
	}
	p := New()
	f.Fuzz(func(t *testing.T, in string) {
		if !invariantHolds(p, in) {
			t.Fatalf("invariant violated for %q", in)
		}
	})
}

// TestLoadDictionaryRejectsMalformed guards the dictionary contract: entries must
// resolve to canonical verbs and known resources, and no target may be empty.
func TestLoadDictionaryRejectsMalformed(t *testing.T) {
	bad := []string{
		`{"version":1,"verbs":{"jump":{"verb":"leap","resource":"legs"}}}`,   // non-canonical verb
		`{"version":1,"verbs":{"go":{"verb":"perform","resource":"wings"}}}`, // unknown resource
		`{"version":1,"targets":{"gate":""}}`,                                // empty target id
		`{not json`,                                                          // malformed json
	}
	for _, b := range bad {
		if _, err := NewFromBytes([]byte(b)); err == nil {
			t.Errorf("NewFromBytes(%s) succeeded; want error", b)
		}
	}
}

// TestEmbeddedDictionaryLoads confirms the checked-in dictionary is valid so
// New() never panics in production.
func TestEmbeddedDictionaryLoads(t *testing.T) {
	if _, err := LoadDictionary(embeddedDictionary); err != nil {
		t.Fatalf("embedded dictionary invalid: %v", err)
	}
}
