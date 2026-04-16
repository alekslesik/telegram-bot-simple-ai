package repository

import (
	"context"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

type Section struct {
	ID    int64
	Code  string
	Title string
}

type Block struct {
	ID        int64
	ChapterID int64
	Code      string
	Title     string
	Type      learning.BlockType
	Order     int
}

type ContentRepository interface {
	GetSectionByCode(ctx context.Context, code string) (Section, error)
	ListChapterBlocks(ctx context.Context, chapterID int64) ([]Block, error)
	GetBlock(ctx context.Context, blockID int64) (Block, error)
}
