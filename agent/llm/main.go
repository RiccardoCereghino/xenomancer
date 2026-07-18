// Command llm-agent is the naive LLM player (backlog 06). It runs as an --agent
// subprocess under cmd/run (ADR-000 D8): it reads observation packets from stdin,
// asks a model provider, takes the model's FREEFORM reply, and routes that reply
// through the quarantined parser (backlog 04) to a canonical round envelope,
// which it writes to stdout for the runner. The parser is the gate — a misparse
// becomes a free wait, never a death (GDD P3): this is the quarantine proving
// itself on genuinely sloppy input. The model is strictly the player, never on
// the engine/rules path (GDD P1, ADR-000 D5 LLM-quarantine).
//
// Providers (--provider):
//   - github-models: free, OpenAI-compatible GitHub Models, keyed by GITHUB_TOKEN
//     (github.go). This is what the gated showcase uses — no external secret.
//   - anthropic (default): the Anthropic Messages API, keyed by ANTHROPIC_API_KEY
//     (client.go).
//
// Run it under the runner (a live episode):
//
//	GITHUB_TOKEN=... go run ./cmd/run --seed 1 --deadline 90s --out replay.json \
//	  --agent "go run ./agent/llm --provider github-models --report death.json"
//
// Model selection: --model (default per provider). --report writes the terminal
// packet (the death report or win) for the showcase to upload.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/RiccardoCereghino/xenomancer/engine"
	"github.com/RiccardoCereghino/xenomancer/parser"
)

// defaultModel is a current Claude model used when none is configured. It is a
// normal external-player config value, overridable per run; the showcase sweeps
// several models over it (GDD §11).
const defaultModel = "claude-sonnet-5"

const systemPrompt = `You are playing XENOMANCER, a text adventure. Each turn you receive a short
observation describing where you are and what just happened. Reply with EXACTLY ONE
action in plain English on a single line — no explanation, no lists.

Movement, looking, waiting, and speaking are all valid. Examples of good replies:
  go to the forest path
  walk to the gate
  look at my reflection
  wait
  say brown to the guard

Your goal: reach the gate and pass the guard. The guard asks the color of your
eyes; you can learn it by looking at your reflection in the still pond along the
way. When the guard asks, answer by naming the guard and a single color word, e.g.
"say brown to the guard". Keep every reply to one short line.`

// incoming captures the fields the agent reads from either an observation packet
// or a terminal (won/died) packet. A non-empty Outcome marks the terminal packet.
type incoming struct {
	Round     int    `json:"round"`
	Narration string `json:"narration"`
	Outcome   string `json:"outcome"`
	Result    struct {
		OK         bool `json:"ok"`
		Rejections []struct {
			Reason string `json:"reason"`
		} `json:"rejections"`
	} `json:"result"`
}

func main() {
	provider := flag.String("provider", envOr("XENO_PROVIDER", "anthropic"), "model provider: anthropic | github-models")
	model := flag.String("model", "", "model id; default depends on --provider")
	reportPath := flag.String("report", "", "write the terminal packet (death report / win) to this file")
	flag.Parse()

	api, err := newCompleter(*provider, *model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "llm-agent: %v\n", err)
		os.Exit(1)
	}
	p := parser.New()

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)

	var history []message
	unparsedNote := "" // feedback about the previous reply that failed to parse

	for in.Scan() {
		raw := in.Bytes()
		if len(raw) == 0 {
			continue
		}
		var pkt incoming
		if err := json.Unmarshal(raw, &pkt); err != nil {
			fmt.Fprintf(os.Stderr, "llm-agent: decode packet: %v\n", err)
			os.Exit(1)
		}
		if pkt.Outcome != "" {
			// Terminal packet (won/died): capture the post-mortem, then stop.
			if *reportPath != "" {
				_ = os.WriteFile(*reportPath, append(trimSpace(raw), '\n'), 0o644)
			}
			return
		}

		history = append(history, message{Role: "user", Content: renderObservation(pkt, unparsedNote)})
		unparsedNote = ""

		reply, err := api.complete(systemPrompt, history)
		if err != nil {
			// The job is allowed to flake (GDD §10): on any model hiccup, wait
			// this round rather than crash. The runner's caps bound the loop.
			fmt.Fprintf(os.Stderr, "llm-agent: %v (waiting this round)\n", err)
			reply = "wait"
		}
		history = append(history, message{Role: "assistant", Content: reply})

		// Route the model's freeform reply through the parser. A hit becomes a
		// canonical envelope; a miss becomes a free wait, and the model is told
		// next turn so it can rephrase (the dictionary's backlog, GDD §13).
		line := firstLine(reply)
		sub, ok := p.Parse(line)
		if !ok {
			unparsedNote = fmt.Sprintf("Your last reply %q was not a valid action. Use a simple command like \"walk to the gate\" or \"say brown to the guard\".", line)
			sub = waitSubmission()
		}
		sub.Round = pkt.Round
		if err := enc.Encode(sub); err != nil {
			fmt.Fprintf(os.Stderr, "llm-agent: encode envelope: %v\n", err)
			os.Exit(1)
		}
		out.Flush()
	}
	if err := in.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "llm-agent: read stdin: %v\n", err)
		os.Exit(1)
	}
}

// waitSubmission is the canonical no-op the agent emits when the model's reply
// could not be parsed — a free round, never a death (GDD P3).
func waitSubmission() engine.RoundSubmission {
	return engine.RoundSubmission{
		V:       engine.ProtocolVersion,
		Actions: []engine.Action{{Resource: "attention", Verb: engine.VerbWait}},
	}
}

// renderObservation turns a packet into the user turn the model reads: the
// human-facing narration, an optional note when the last reply failed to parse,
// and the prompt for one short action.
func renderObservation(pkt incoming, unparsedNote string) string {
	var b strings.Builder
	b.WriteString(pkt.Narration)
	if unparsedNote != "" {
		b.WriteString("\n(")
		b.WriteString(unparsedNote)
		b.WriteString(")")
	}
	b.WriteString("\n\nWhat do you do? Reply with one short action.")
	return b.String()
}

// firstLine returns the first non-empty line of the model's reply, trimmed. The
// system prompt asks for a single line, but models sometimes add prose; the
// parser would reject the rest anyway, so we take the first line.
func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return strings.TrimSpace(s)
}

func trimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
