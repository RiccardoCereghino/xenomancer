// GitHub Models provider for the LLM player (a free, OpenAI-compatible inference
// API — https://models.github.ai). In GitHub Actions it authenticates with the
// built-in GITHUB_TOKEN (workflow `permissions: models: read`), so the showcase
// runs with no external secret and no per-token billing. Like the Anthropic
// client it is STDLIB-ONLY (net/http) — no third-party SDK, no new dependency.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ghDefaultBase  = "https://models.github.ai/inference"
	ghDefaultModel = "openai/gpt-4o-mini"
	ghMaxRetries   = 3
)

// completer is the provider-agnostic seam the agent loop talks to. Both the
// Anthropic client and the GitHub Models client satisfy it.
type completer interface {
	complete(system string, messages []message) (string, error)
}

// newCompleter builds the completer for the chosen provider, validating that the
// required credential is present.
func newCompleter(provider, model string) (completer, error) {
	switch provider {
	case "anthropic", "":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return nil, errors.New("ANTHROPIC_API_KEY is not set (needed for --provider anthropic)")
		}
		if model == "" {
			model = envOr("ANTHROPIC_MODEL", defaultModel)
		}
		return newClient(key, model), nil
	case "github-models", "github", "gh":
		tok := os.Getenv("GITHUB_TOKEN")
		if tok == "" {
			return nil, errors.New("GITHUB_TOKEN is not set (needed for --provider github-models)")
		}
		return newGHClient(tok, model), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (want anthropic or github-models)", provider)
	}
}

// ghClient is a minimal GitHub Models client built on net/http only.
type ghClient struct {
	token      string
	model      string
	endpoint   string // overridable via GITHUB_MODELS_BASE_URL (tests, proxies)
	httpClient *http.Client
}

func newGHClient(token, model string) *ghClient {
	if model == "" {
		model = envOr("GITHUB_MODELS_MODEL", ghDefaultModel)
	}
	base := ghDefaultBase
	if b := os.Getenv("GITHUB_MODELS_BASE_URL"); b != "" {
		base = b
	}
	return &ghClient{
		token:      token,
		model:      model,
		endpoint:   strings.TrimRight(base, "/") + "/chat/completions",
		httpClient: &http.Client{Timeout: defaultHTTPTimout},
	}
}

type ghRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type ghResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// complete sends the system prompt (as the leading system-role message, the
// OpenAI convention) plus the conversation and returns the model's reply. It
// retries a 429 a few times honoring Retry-After, since the free tier is
// rate-limited; other errors surface so the caller can degrade (flake) gracefully.
func (c *ghClient) complete(system string, messages []message) (string, error) {
	msgs := make([]message, 0, len(messages)+1)
	if system != "" {
		msgs = append(msgs, message{Role: "system", Content: system})
	}
	msgs = append(msgs, messages...)

	body, err := json.Marshal(ghRequest{Model: c.model, Messages: msgs, MaxTokens: defaultMaxTokens})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	for attempt := 0; ; attempt++ {
		text, retryAfter, err := c.attempt(body)
		if retryAfter >= 0 && attempt < ghMaxRetries {
			time.Sleep(retryAfter)
			continue
		}
		return text, err
	}
}

// attempt makes one request. It returns a non-negative retryAfter to signal a
// rate-limit that should be retried (the caller sleeps and loops); a negative
// retryAfter means "do not retry" — use text/err as final.
func (c *ghClient) attempt(body []byte) (string, time.Duration, error) {
	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", -1, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/vnd.github+json")
	req.Header.Set("authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", -1, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", -1, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", retryAfter(resp.Header.Get("retry-after")), fmt.Errorf("rate limited")
	}

	var parsed ghResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", -1, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return "", -1, fmt.Errorf("github models error %d: %s", resp.StatusCode, parsed.Error.Message)
		}
		return "", -1, fmt.Errorf("github models error: status %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return "", -1, fmt.Errorf("empty response from model")
	}
	return parsed.Choices[0].Message.Content, -1, nil
}

// retryAfter parses a Retry-After header (seconds) into a backoff, defaulting to
// two seconds when the header is absent or unparseable.
func retryAfter(header string) time.Duration {
	if header != "" {
		if secs, err := time.ParseDuration(strings.TrimSpace(header) + "s"); err == nil && secs > 0 {
			return secs
		}
	}
	return 2 * time.Second
}
