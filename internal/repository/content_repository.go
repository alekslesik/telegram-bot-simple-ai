package repository

import (
	"context"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

type ContentRepository interface {
	GetSectionByCode(ctx context.Context, code string) (learning.Section, error)
	ListChapterBlocks(ctx context.Context, chapterID int64) ([]learning.Block, error)
	GetBlock(ctx context.Context, blockID int64) (learning.Block, error)
}
