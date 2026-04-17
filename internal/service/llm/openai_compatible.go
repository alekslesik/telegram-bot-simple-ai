package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration

	// HTTPClient allows tests to inject a deterministic transport/client.
	// If nil, NewOpenAICompatible builds a client with Config.Timeout.
	HTTPClient *http.Client
}

type OpenAICompatible struct {
	cfg        Config
	httpClient *http.Client
}

var _ Provider = (*OpenAICompatible)(nil)

func NewOpenAICompatible(cfg Config) *OpenAICompatible {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	// Always honor Config.Timeout, even if the caller injected a custom client.
	httpClient.Timeout = timeout

	return &OpenAICompatible{
		cfg:        cfg,
		httpClient: httpClient,
	}
}

func (a *OpenAICompatible) ReviewAnswer(ctx context.Context, input ReviewInput) (ReviewResult, error) {
	content, err := a.chatCompletion(ctx, reviewSystemPrompt, formatReviewPrompt(input))
	if err != nil {
		return ReviewResult{}, err
	}

	return ReviewResult{Markdown: content}, nil
}

func (a *OpenAICompatible) ExplainTheory(ctx context.Context, input TheoryInput) (string, error) {
	return a.chatCompletion(ctx, theorySystemPrompt, formatTheoryPrompt(input))
}

func (a *OpenAICompatible) ExplainSolution(ctx context.Context, input SolutionInput) (string, error) {
	return a.chatCompletion(ctx, solutionSystemPrompt, formatSolutionPrompt(input))
}

func (a *OpenAICompatible) Chat(ctx context.Context, input ChatInput) (string, error) {
	return a.chatCompletion(ctx, chatSystemPrompt, formatChatPrompt(input))
}

func (a *OpenAICompatible) chatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	requestBody := struct {
		Model    string        `json:"model"`
		Messages []chatMessage `json:"messages"`
	}{
		Model: a.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat completion request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(a.cfg.BaseURL, "/")+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("build chat completion request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send chat completion request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("chat completion status %d and failed to read body: %w", resp.StatusCode, readErr)
		}

		return "", fmt.Errorf("chat completion status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode chat completion response: %w", err)
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("chat completion response contained no choices")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("chat completion response contained empty content")
	}

	return content, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

const (
	reviewSystemPrompt   = "You are a Go interview coach. Review the user's answer and reply in concise markdown."
	theorySystemPrompt   = "You are a Go interview coach. Explain the requested theory in concise markdown."
	solutionSystemPrompt = "You are a Go interview coach. Explain the reference solution in concise markdown."
	chatSystemPrompt     = "You are a Go interview prep assistant. Always respond in Russian, plain text only (no markdown, no headings, no bullet lists). Keep answers concise and conversational. Only discuss Go/programming/interview/learning topics. If asked about unrelated topics, politely refuse and redirect to Go interview prep. If the user greets you, greet back and ask a Go-related follow-up."
)

func formatReviewPrompt(input ReviewInput) string {
	return fmt.Sprintf(
		"Task:\n%s\n\nUser answer:\n%s\n\nReference solution:\n%s",
		strings.TrimSpace(input.Task),
		strings.TrimSpace(input.UserAnswer),
		strings.TrimSpace(input.ReferenceSolution),
	)
}

func formatTheoryPrompt(input TheoryInput) string {
	return fmt.Sprintf(
		"Theory:\n%s\n\nQuestion:\n%s",
		strings.TrimSpace(input.Theory),
		strings.TrimSpace(input.Question),
	)
}

func formatSolutionPrompt(input SolutionInput) string {
	return fmt.Sprintf(
		"Task:\n%s\n\nReference solution:\n%s",
		strings.TrimSpace(input.Task),
		strings.TrimSpace(input.Solution),
	)
}

func formatChatPrompt(input ChatInput) string {
	return strings.TrimSpace(input.Message)
}
