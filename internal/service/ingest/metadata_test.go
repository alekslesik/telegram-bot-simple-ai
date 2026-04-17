package ingest

import "testing"

func TestBuildDocumentMetaFromSectionAndChapterPath(t *testing.T) {
	file := InputFile{
		Path: "content/raw/introduction/Big O/FireShot Capture 312 - Intro.pdf",
		Name: "FireShot Capture 312 - Intro.pdf",
	}

	meta, err := BuildDocumentMeta("content/raw", file, 1)
	if err != nil {
		t.Fatalf("BuildDocumentMeta() error = %v", err)
	}
	if meta.SectionCode != "introduction" {
		t.Fatalf("expected section code introduction, got %q", meta.SectionCode)
	}
	if meta.ChapterCode != "big-o" {
		t.Fatalf("expected chapter code big-o, got %q", meta.ChapterCode)
	}
	if meta.SortOrder != 312 {
		t.Fatalf("expected sort order from capture number, got %d", meta.SortOrder)
	}
}

func TestBuildDocumentMetaFallsBackToProvidedSortOrder(t *testing.T) {
	file := InputFile{
		Path: "content/raw/introduction/notes.pdf",
		Name: "notes.pdf",
	}

	meta, err := BuildDocumentMeta("content/raw", file, 42)
	if err != nil {
		t.Fatalf("BuildDocumentMeta() error = %v", err)
	}
	if meta.SortOrder != 42 {
		t.Fatalf("expected fallback sort order, got %d", meta.SortOrder)
	}
	if meta.ChapterCode != "general" {
		t.Fatalf("expected general chapter for section-level file, got %q", meta.ChapterCode)
	}
}
