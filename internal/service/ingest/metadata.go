package ingest

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type DocumentMeta struct {
	SectionCode  string
	SectionTitle string
	ChapterCode  string
	ChapterTitle string
	BlockCode    string
	BlockTitle   string
	SortOrder    int
}

var codeSanitizer = regexp.MustCompile(`[^\pL\pN]+`)

func BuildDocumentMeta(root string, file InputFile, fallbackSortOrder int) (DocumentMeta, error) {
	rel, err := filepath.Rel(root, file.Path)
	if err != nil {
		return DocumentMeta{}, fmt.Errorf("build relative path for %q: %w", file.Path, err)
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 {
		return DocumentMeta{}, fmt.Errorf("file %q must be inside <section>/<file|chapter/file>", file.Path)
	}

	sectionPart := strings.TrimSpace(parts[0])
	chapterPart := "general"
	if len(parts) >= 3 {
		chapterPart = strings.TrimSpace(parts[len(parts)-2])
	}

	fileBase := strings.TrimSpace(strings.TrimSuffix(file.Name, filepath.Ext(file.Name)))
	if fileBase == "" {
		fileBase = "untitled"
	}

	sortOrder := fallbackSortOrder
	if captureNumber, ok := extractOrder(file.Name); ok {
		sortOrder = captureNumber
	}

	blockCode := fmt.Sprintf("capture-%04d-%s", sortOrder, normalizeCode(fileBase))
	if strings.TrimSpace(blockCode) == "" {
		blockCode = fmt.Sprintf("capture-%04d", sortOrder)
	}

	return DocumentMeta{
		SectionCode:  normalizeCode(sectionPart),
		SectionTitle: sectionPart,
		ChapterCode:  normalizeCode(chapterPart),
		ChapterTitle: chapterPart,
		BlockCode:    blockCode,
		BlockTitle:   fileBase,
		SortOrder:    sortOrder,
	}, nil
}

func normalizeCode(s string) string {
	normalized := strings.ToLower(strings.TrimSpace(s))
	normalized = codeSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "general"
	}
	return normalized
}
