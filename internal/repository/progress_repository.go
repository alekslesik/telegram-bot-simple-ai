package repository

import (
	"context"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

type ProgressRepository interface {
	GetUserProgress(ctx context.Context, userID, blockID int64) (learning.ProgressRecord, error)
	SaveUserProgress(ctx context.Context, progress learning.ProgressRecord) error
	CreateAttempt(ctx context.Context, attempt learning.AttemptRecord) (int64, error)
	ListAttempts(ctx context.Context, userID, blockID int64) ([]learning.AttemptRecord, error)
	GetActiveSession(ctx context.Context, userID int64) (learning.SessionRecord, error)
	SaveSession(ctx context.Context, session learning.SessionRecord) error
	CloseSession(ctx context.Context, sessionID int64) error
}
