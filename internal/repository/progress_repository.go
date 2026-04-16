package repository

import (
	"context"
	"time"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

type UserProgress struct {
	UserID      int64
	ChapterID   int64
	CurrentStep learning.FlowStep
	UpdatedAt   time.Time
}

type UserAttempt struct {
	ID        int64
	UserID    int64
	BlockID   int64
	Answer    string
	Verdict   string
	CreatedAt time.Time
}

type ProgressRepository interface {
	GetUserProgress(ctx context.Context, userID, chapterID int64) (UserProgress, error)
	SaveUserProgress(ctx context.Context, progress UserProgress) error
	CreateAttempt(ctx context.Context, attempt UserAttempt) (int64, error)
}
