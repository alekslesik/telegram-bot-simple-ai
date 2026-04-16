package repository

import (
	"context"
	"time"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

type UserProgress struct {
	ID             int64
	UserID         int64
	ChapterID      int64
	CurrentBlockID *int64
	CurrentStep    learning.FlowStep
	Status         string
	StartedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

type UserAttempt struct {
	ID         int64
	UserID     int64
	BlockID    int64
	AnswerText string
	ReviewText string
	Score      *float64
	CreatedAt  time.Time
}

type ProgressRepository interface {
	GetUserProgress(ctx context.Context, userID, chapterID int64) (UserProgress, error)
	SaveUserProgress(ctx context.Context, progress UserProgress) error
	CreateAttempt(ctx context.Context, attempt UserAttempt) (int64, error)
}
