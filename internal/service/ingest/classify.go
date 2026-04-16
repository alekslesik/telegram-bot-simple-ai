package ingest

import (
	"strings"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

func DetectBlockType(text string) learning.BlockType {
	switch {
	case strings.Contains(text, "Условие") || strings.Contains(text, "Пример 1"):
		return learning.BlockTask
	case strings.Contains(text, "Решение") && !strings.Contains(text, "Условие"):
		return learning.BlockSolution
	default:
		return learning.BlockTheory
	}
}
