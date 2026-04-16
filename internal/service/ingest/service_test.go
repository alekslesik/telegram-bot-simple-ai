package ingest

import (
	"testing"
	"time"
)

func TestSortFilesByCaptureThenMtime(t *testing.T) {
	files := []InputFile{
		{Name: "FireShot Capture 307 - B.pdf", ModTime: time.Unix(20, 0)},
		{Name: "FireShot Capture 307 - A.pdf", ModTime: time.Unix(10, 0)},
		{Name: "FireShot Capture 306 - X.pdf", ModTime: time.Unix(30, 0)},
	}

	got := SortInputFiles(files)
	if got[0].Name != "FireShot Capture 306 - X.pdf" {
		t.Fatalf("unexpected first file: %s", got[0].Name)
	}
}

func TestSortInputFilesNewerFirstForSameCaptureNumber(t *testing.T) {
	files := []InputFile{
		{Name: "FireShot Capture 306 - Old.pdf", ModTime: time.Unix(10, 0)},
		{Name: "FireShot Capture 306 - New.pdf", ModTime: time.Unix(20, 0)},
	}

	got := SortInputFiles(files)
	if got[0].Name != "FireShot Capture 306 - New.pdf" {
		t.Fatalf("expected newer file first, got %q", got[0].Name)
	}
	if got[1].Name != "FireShot Capture 306 - Old.pdf" {
		t.Fatalf("expected older file second, got %q", got[1].Name)
	}
}

func TestSortInputFilesPlacesUnorderedFilesAfterCapturedOnes(t *testing.T) {
	files := []InputFile{
		{Name: "notes.pdf", ModTime: time.Unix(5, 0)},
		{Name: "FireShot Capture 306 - X.pdf", ModTime: time.Unix(30, 0)},
		{Name: "appendix.pdf", ModTime: time.Unix(1, 0)},
	}

	got := SortInputFiles(files)
	if got[0].Name != "FireShot Capture 306 - X.pdf" {
		t.Fatalf("unexpected first file: %s", got[0].Name)
	}
	if got[1].Name != "notes.pdf" {
		t.Fatalf("unexpected second file: %s", got[1].Name)
	}
	if got[2].Name != "appendix.pdf" {
		t.Fatalf("unexpected third file: %s", got[2].Name)
	}
}
