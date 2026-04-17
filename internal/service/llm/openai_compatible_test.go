package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type cancelRoundTripper struct {
	ready chan struct{}
}

func (rt *cancelRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	close(rt.ready)
	<-req.Context().Done()
	return nil, req.Context().Err()
}

type timeoutRoundTripper struct{}

func (rt *timeoutRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestReviewAnswerUsesChatCompletions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer x" {
			t.Fatalf("unexpected authorization header %q", got)
		}

		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "deepseek-chat" {
			t.Fatalf("unexpected model %q", req.Model)
		}
		if len(req.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Fatalf("expected first role to be system, got %q", req.Messages[0].Role)
		}
		if req.Messages[1].Role != "user" {
			t.Fatalf("expected second role to be user, got %q", req.Messages[1].Role)
		}

		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Looks good"}}]}`))
	}))
	defer server.Close()

	client := NewOpenAICompatible(Config{BaseURL: server.URL, APIKey: "x", Model: "deepseek-chat"})
	got, err := client.ReviewAnswer(context.Background(), ReviewInput{})
	if err != nil || got.Markdown == "" {
		t.Fatal("expected review response")
	}
}

func TestChatIncludesHistoryMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Messages) != 4 {
			t.Fatalf("expected 4 messages (system + history pair + user), got %d", len(req.Messages))
		}
		if req.Messages[1].Role != "user" || req.Messages[1].Content != "привет" {
			t.Fatalf("unexpected first history message: %#v", req.Messages[1])
		}
		if req.Messages[2].Role != "assistant" || req.Messages[2].Content != "привет! чем помочь?" {
			t.Fatalf("unexpected second history message: %#v", req.Messages[2])
		}
		if req.Messages[3].Role != "user" || req.Messages[3].Content != "объясни map в go" {
			t.Fatalf("unexpected current user message: %#v", req.Messages[3])
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	client := NewOpenAICompatible(Config{BaseURL: server.URL, APIKey: "x", Model: "gpt-4o-mini"})
	_, err := client.Chat(context.Background(), ChatInput{
		Message: "объясни map в go",
		History: []ChatMessage{
			{Role: "user", Content: "привет"},
			{Role: "assistant", Content: "привет! чем помочь?"},
		},
	})
	if err != nil {
		t.Fatalf("expected chat without error, got %v", err)
	}
}

func TestReviewAnswerReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewOpenAICompatible(Config{BaseURL: server.URL, APIKey: "x", Model: "deepseek-chat"})

	_, err := client.ReviewAnswer(context.Background(), ReviewInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 429") {
		t.Fatalf("expected status code in error, got %v", err)
	}
}

func TestReviewAnswerReturnsErrorOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewOpenAICompatible(Config{BaseURL: server.URL, APIKey: "x", Model: "deepseek-chat"})

	_, err := client.ReviewAnswer(context.Background(), ReviewInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "decode chat completion response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestReviewAnswerReturnsErrorOnEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := NewOpenAICompatible(Config{BaseURL: server.URL, APIKey: "x", Model: "deepseek-chat"})
	_, err := client.ReviewAnswer(context.Background(), ReviewInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Fatalf("expected no choices in error, got %v", err)
	}
}

func TestReviewAnswerReturnsErrorOnEmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"   "}}]}`))
	}))
	defer server.Close()

	client := NewOpenAICompatible(Config{BaseURL: server.URL, APIKey: "x", Model: "deepseek-chat"})
	_, err := client.ReviewAnswer(context.Background(), ReviewInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "empty content") {
		t.Fatalf("expected empty content in error, got %v", err)
	}
}

func TestReviewAnswerRespectsContextCancellation(t *testing.T) {
	// RoundTripper blocks until the request context is done, then returns the context error.
	// This avoids httptest/server shutdown flakiness while still verifying ctx propagation.
	rt := &cancelRoundTripper{ready: make(chan struct{})}

	clientHTTP := &http.Client{
		Transport: rt,
		Timeout:   5 * time.Second,
	}

	// BaseURL is unused by the RoundTripper; it only needs to be a valid URL.
	client := NewOpenAICompatible(Config{
		BaseURL:    "http://example.com",
		APIKey:     "x",
		Model:      "deepseek-chat",
		Timeout:    5 * time.Second,
		HTTPClient: clientHTTP,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.ReviewAnswer(ctx, ReviewInput{})
		done <- err
	}()

	<-rt.ready
	cancel()

	err := <-done
	if err == nil {
		t.Fatal("expected error after context cancellation")
	}
}

func TestReviewAnswerUsesConfiguredTimeout(t *testing.T) {
	client := NewOpenAICompatible(Config{
		BaseURL: "http://example.com",
		APIKey:  "x",
		Model:   "deepseek-chat",
		Timeout: 100 * time.Millisecond,
		HTTPClient: &http.Client{
			Transport: &timeoutRoundTripper{},
		},
	})

	_, err := client.ReviewAnswer(context.Background(), ReviewInput{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// Error text differs across Go versions; be permissive.
	if !strings.Contains(strings.ToLower(err.Error()), "timeout") && !strings.Contains(strings.ToLower(err.Error()), "deadline") {
		t.Fatalf("expected timeout/deadline in error, got %v", err)
	}
}
