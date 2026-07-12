// Package parser is the quarantined freeform parser (GDD §5.2, ADR-000 D3/D4).
// It maps freeform agent text to canonical engine.RoundSubmissions by
// deterministic dictionary lookup ONLY — no fuzzy matching, no model, no
// randomness. The same input always yields the same canonical action, or the
// same rejection.
//
// It lives OUTSIDE /engine on purpose: the engine never imports it, and only
// canonical actions cross into the engine and the replay log. So the replay
// path never depends on parser behavior, and parser evolution can never
// invalidate a replay (ADR-000 D3).
//
// Its load-bearing guarantee is P3 — a misparse never kills (GDD §5.2): input
// with no dictionary hit is a free rejection ("I don't understand") that costs
// no tick and never reaches the engine, and no freeform string can ever produce
// a state-affecting action that was not an exact dictionary hit.
package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// RejectMessage is returned for any freeform line with no dictionary hit
// (GDD §5.2). The rejection is free: no tick, no state change.
const RejectMessage = "I don't understand."

// VerbEntry maps a freeform verb phrase to a canonical verb and the resource it
// claims. Movement is a perform on legs, so both the verb and the resource are
// data (GDD §5.1/§5.2). Phrase is the freeform key; it is normalized on load.
type VerbEntry struct {
	Phrase   string `json:"phrase"`
	Verb     string `json:"verb"`
	Resource string `json:"resource"`
}

// TargetEntry maps a freeform target phrase to a canonical target id — a
// location id (a move target) or an inspectable id (an inspect target). Phrase
// is the freeform key; it is normalized on load.
type TargetEntry struct {
	Phrase string `json:"phrase"`
	Target string `json:"target"`
}

// canonicalVerb resolves a freeform verb phrase to its canonical verb+resource.
type canonicalVerb struct {
	verb     string
	resource string
}

// Dictionary is a versioned, immutable lookup table. Runtime resolution is a
// pure keyed lookup by normalized phrase, so it is order-independent — the
// "no map iteration" law binds only /engine, and this package never iterates
// these maps in a state-affecting path.
type Dictionary struct {
	Version int
	verbs   map[string]canonicalVerb
	targets map[string]string
	// maxVerbWords is the longest verb phrase in words, so Parse can try a
	// longest-prefix match without scanning the whole map.
	maxVerbWords int
}

// dictionaryFile is the on-disk JSON shape (checked in as data, AI-authored
// offline; runtime is pure table lookup — GDD §5.2, ADR-000 D3).
type dictionaryFile struct {
	Version int           `json:"version"`
	Verbs   []VerbEntry   `json:"verbs"`
	Targets []TargetEntry `json:"targets"`
}

// LoadDictionary validates and loads a dictionary from its plaintext bytes.
// The parser performs no filesystem access itself; the caller (a shell or a
// test) reads the file and passes the bytes here, mirroring engine.ParseContent.
func LoadDictionary(data []byte) (Dictionary, error) {
	var f dictionaryFile
	if err := json.Unmarshal(data, &f); err != nil {
		return Dictionary{}, fmt.Errorf("parser: dictionary parse: %w", err)
	}

	d := Dictionary{
		Version: f.Version,
		verbs:   make(map[string]canonicalVerb, len(f.Verbs)),
		targets: make(map[string]string, len(f.Targets)),
	}

	for i := 0; i < len(f.Verbs); i++ {
		e := f.Verbs[i]
		phrase := Normalize(e.Phrase)
		if phrase == "" {
			return Dictionary{}, fmt.Errorf("parser: verb entry %d has an empty phrase", i)
		}
		if !isCanonicalVerb(e.Verb) {
			return Dictionary{}, fmt.Errorf("parser: verb phrase %q maps to non-canonical verb %q", e.Phrase, e.Verb)
		}
		if e.Resource == "" {
			return Dictionary{}, fmt.Errorf("parser: verb phrase %q maps to an empty resource", e.Phrase)
		}
		if _, dup := d.verbs[phrase]; dup {
			return Dictionary{}, fmt.Errorf("parser: duplicate verb phrase %q", phrase)
		}
		d.verbs[phrase] = canonicalVerb{verb: e.Verb, resource: e.Resource}
		if n := len(strings.Fields(phrase)); n > d.maxVerbWords {
			d.maxVerbWords = n
		}
	}

	for i := 0; i < len(f.Targets); i++ {
		e := f.Targets[i]
		phrase := Normalize(e.Phrase)
		if phrase == "" {
			return Dictionary{}, fmt.Errorf("parser: target entry %d has an empty phrase", i)
		}
		if e.Target == "" {
			return Dictionary{}, fmt.Errorf("parser: target phrase %q maps to an empty target id", e.Phrase)
		}
		if _, dup := d.targets[phrase]; dup {
			return Dictionary{}, fmt.Errorf("parser: duplicate target phrase %q", phrase)
		}
		d.targets[phrase] = e.Target
	}

	return d, nil
}

// Result is the outcome of parsing one freeform line. When OK is true,
// Submission holds a single canonical action ready for the engine. When OK is
// false, Submission is the zero value (no actions) and Message is the free
// rejection text — a misparse never produces an action (P3).
type Result struct {
	Submission engine.RoundSubmission
	OK         bool
	Message    string
}

// Parse maps one freeform line to a canonical submission by deterministic
// lookup only. It is the sole seam through which freeform text may become a
// canonical action: any input that does not resolve to a verb phrase followed
// by a valid target (or a bare wait) is a free rejection, never an action.
func (d Dictionary) Parse(line string) Result {
	tokens := strings.Fields(Normalize(line))
	if len(tokens) == 0 {
		return reject()
	}

	// Longest-prefix match the leading tokens against the verb table, so
	// multi-word verb phrases ("look at") win over shorter ones when present.
	max := d.maxVerbWords
	if max > len(tokens) {
		max = len(tokens)
	}
	for n := max; n >= 1; n-- {
		phrase := strings.Join(tokens[:n], " ")
		cv, ok := d.verbs[phrase]
		if !ok {
			continue
		}
		rest := tokens[n:]

		if cv.verb == engine.VerbWait {
			// Wait takes no target; any trailing tokens make it unrecognized.
			if len(rest) != 0 {
				return reject()
			}
			return accept(engine.Action{Resource: cv.resource, Verb: cv.verb})
		}

		if len(rest) == 0 {
			return reject()
		}
		target, ok := d.targets[strings.Join(rest, " ")]
		if !ok {
			return reject()
		}
		return accept(engine.Action{Resource: cv.resource, Verb: cv.verb, Target: target})
	}

	return reject()
}

// Normalize reduces a freeform line to its lookup key: lowercase, every
// non-alphanumeric rune becomes a space, whitespace is collapsed, and the ends
// are trimmed (GDD §5.2). It is deterministic and does no fuzzy matching.
func Normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func isCanonicalVerb(v string) bool {
	switch v {
	case engine.VerbInspect, engine.VerbPerform, engine.VerbTalk, engine.VerbWait:
		return true
	}
	return false
}

func accept(a engine.Action) Result {
	return Result{
		OK: true,
		Submission: engine.RoundSubmission{
			V:       engine.ProtocolVersion,
			Actions: []engine.Action{a},
		},
	}
}

func reject() Result {
	return Result{OK: false, Message: RejectMessage}
}
