package llm

import "context"

type Provider interface {
	ReviewAnswer(ctx context.Context, input ReviewInput) (ReviewResult, error)
	ExplainTheory(ctx context.Context, input TheoryInput) (string, error)
	ExplainSolution(ctx context.Context, input SolutionInput) (string, error)
	Chat(ctx context.Context, input ChatInput) (string, error)
}

type ReviewInput struct {
	Task              string
	UserAnswer        string
	ReferenceSolution string
}

type ReviewResult struct {
	Markdown string
}

type TheoryInput struct {
	Theory   string
	Question string
}

type SolutionInput struct {
	Task     string
	Solution string
}

type ChatInput struct {
	Message string
	History []ChatMessage
}

type ChatMessage struct {
	Role    string
	Content string
}
