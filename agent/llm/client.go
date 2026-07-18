// Command llm-agent's Anthropic Messages API client. It is a STDLIB-ONLY
// (net/http) client — no third-party SDK, no new module dependency (backlog 06).
// The LLM is strictly a player: it never touches the engine, rules, or parser
// lethal path (GDD P1, ADR-000 D5 LLM-quarantine).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultEndpoint   = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
	defaultMaxTokens  = 512
	defaultHTTPTimout = 60 * time.Second
)

// message is one turn in the Messages API conversation.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// client is a minimal Anthropic Messages API client built on net/http only.
type client struct {
	apiKey     string
	model      string
	endpoint   string // overridable for tests; defaults to the public API
	httpClient *http.Client
}

func newClient(apiKey, model string) *client {
	// ANTHROPIC_BASE_URL overrides the endpoint (a proxy, a gateway, or a local
	// stub for tests), mirroring the official SDK env var. When set, "/v1/messages"
	// is appended to the base.
	endpoint := defaultEndpoint
	if base := os.Getenv("ANTHROPIC_BASE_URL"); base != "" {
		endpoint = strings.TrimRight(base, "/") + "/v1/messages"
	}
	return &client{
		apiKey:     apiKey,
		model:      model,
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: defaultHTTPTimout},
	}
}

type requestBody struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
}

type responseBody struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// complete sends the system prompt and conversation and returns the model's
// concatenated text reply. A refusal or API error is returned as an error so
// the caller can degrade gracefully (the showcase job is allowed to flake).
func (c *client) complete(system string, messages []message) (string, error) {
	body, err := json.Marshal(requestBody{
		Model:     c.model,
		MaxTokens: defaultMaxTokens,
		System:    system,
		Messages:  messages,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed responseBody
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return "", fmt.Errorf("api error %d: %s: %s", resp.StatusCode, parsed.Error.Type, parsed.Error.Message)
		}
		return "", fmt.Errorf("api error: status %d", resp.StatusCode)
	}
	if parsed.StopReason == "refusal" {
		return "", fmt.Errorf("model refused the request")
	}

	var text string
	for i := 0; i < len(parsed.Content); i++ {
		if parsed.Content[i].Type == "text" {
			text += parsed.Content[i].Text
		}
	}
	if text == "" {
		return "", fmt.Errorf("empty response from model")
	}
	return text, nil
}
