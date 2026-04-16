package learning

import (
	"testing"
	"time"
)

func TestValidFlowSteps(t *testing.T) {
	steps := []FlowStep{StepTheory, StepTask, StepAnswer, StepReview, StepSolution}
	if len(steps) != 5 {
		t.Fatalf("expected 5 flow steps, got %d", len(steps))
	}

	want := []FlowStep{"theory", "task", "answer", "review", "solution"}
	for i := range steps {
		if steps[i] != want[i] {
			t.Fatalf("expected step %d to be %q, got %q", i, want[i], steps[i])
		}
	}
}

func TestProgressAndSessionStates(t *testing.T) {
	if ProgressStatusInProgress != "in_progress" {
		t.Fatalf("expected in-progress status, got %q", ProgressStatusInProgress)
	}
	if ProgressStatusCompleted != "completed" {
		t.Fatalf("expected completed status, got %q", ProgressStatusCompleted)
	}
	if SessionStateActive != "active" {
		t.Fatalf("expected active session state, got %q", SessionStateActive)
	}
	if SessionStateClosed != "closed" {
		t.Fatalf("expected closed session state, got %q", SessionStateClosed)
	}
}

func TestAttemptAndSessionRecordsMatchSchemaFields(t *testing.T) {
	now := time.Now()
	score := 4.5
	sectionID := int64(11)
	chapterID := int64(12)
	blockID := int64(13)

	attempt := AttemptRecord{
		AttemptNo:     2,
		AnswerText:    "answer",
		LLMFeedbackMD: "feedback",
		Score:         &score,
		CreatedAt:     now,
	}

	if attempt.AttemptNo != 2 {
		t.Fatalf("expected attempt number to be preserved, got %d", attempt.AttemptNo)
	}
	if attempt.LLMFeedbackMD != "feedback" {
		t.Fatalf("expected llm feedback markdown, got %q", attempt.LLMFeedbackMD)
	}

	session := SessionRecord{
		ActiveSectionID: &sectionID,
		ActiveChapterID: &chapterID,
		ActiveBlockID:   &blockID,
		FlowStep:        StepReview,
		Mode:            "learning",
		UpdatedAt:       now,
	}

	if session.ActiveSectionID == nil || *session.ActiveSectionID != sectionID {
		t.Fatal("expected active section id to be set")
	}
	if session.ActiveChapterID == nil || *session.ActiveChapterID != chapterID {
		t.Fatal("expected active chapter id to be set")
	}
	if session.ActiveBlockID == nil || *session.ActiveBlockID != blockID {
		t.Fatal("expected active block id to be set")
	}
	if session.FlowStep != StepReview {
		t.Fatalf("expected flow step to be preserved, got %q", session.FlowStep)
	}
	if session.Mode != "learning" {
		t.Fatalf("expected mode to be preserved, got %q", session.Mode)
	}
}

func TestChapterDTOShape(t *testing.T) {
	ch := Chapter{
		ID:        1,
		SectionID: 2,
		Code:      "intro",
		Title:     "Introduction",
		SortOrder: 3,
	}

	if ch.ID != 1 {
		t.Fatalf("expected id to be preserved, got %d", ch.ID)
	}
	if ch.SectionID != 2 {
		t.Fatalf("expected section id to be preserved, got %d", ch.SectionID)
	}
	if ch.Code != "intro" {
		t.Fatalf("expected code to be preserved, got %q", ch.Code)
	}
	if ch.Title != "Introduction" {
		t.Fatalf("expected title to be preserved, got %q", ch.Title)
	}
	if ch.SortOrder != 3 {
		t.Fatalf("expected sort order to be preserved, got %d", ch.SortOrder)
	}
}

func TestBlockContentAndRelationDTOShape(t *testing.T) {
	page := 7
	content := BlockContent{
		BlockID:        10,
		TheoryMD:       "theory",
		TaskMD:         "task",
		SolutionMD:     "solution",
		ImageURLs:      []string{"https://example.test/image.png"},
		Difficulty:     "medium",
		Tags:           []string{"arrays", "intro"},
		LanguageCode:   "ru",
		SourceType:     "pdf",
		SourcePath:     "docs/source.pdf",
		SourcePage:     &page,
		SourceChunkRef: "chunk-1",
		SourceMetadata: map[string]any{
			"page_width":  100,
			"page_height": 200,
		},
	}

	if content.BlockID != 10 {
		t.Fatalf("expected block id to be preserved, got %d", content.BlockID)
	}
	if content.SourcePage == nil || *content.SourcePage != 7 {
		t.Fatal("expected source page to be set")
	}
	if len(content.ImageURLs) != 1 || content.ImageURLs[0] == "" {
		t.Fatal("expected image urls to be preserved")
	}

	relation := BlockRelation{
		FromBlockID:  10,
		ToBlockID:    11,
		RelationType: RelationTaskSolution,
		SortOrder:    1,
	}

	if relation.FromBlockID != 10 || relation.ToBlockID != 11 {
		t.Fatal("expected block relation ids to be preserved")
	}
	if relation.RelationType != RelationTaskSolution {
		t.Fatalf("expected relation type to be preserved, got %q", relation.RelationType)
	}
}
