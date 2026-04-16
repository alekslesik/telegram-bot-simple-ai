package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
