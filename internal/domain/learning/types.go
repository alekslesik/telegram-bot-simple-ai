package learning

import "time"

type BlockType string
type FlowStep string
type ProgressStatus string
type SessionState string

const (
	BlockTheory   BlockType = "theory"
	BlockTask     BlockType = "task"
	BlockSolution BlockType = "solution"

	StepTheory   FlowStep = "theory"
	StepTask     FlowStep = "task"
	StepAnswer   FlowStep = "answer"
	StepReview   FlowStep = "review"
	StepSolution FlowStep = "solution"

	ProgressStatusInProgress ProgressStatus = "in_progress"
	ProgressStatusCompleted  ProgressStatus = "completed"

	SessionStateActive SessionState = "active"
	SessionStateClosed SessionState = "closed"
)

type Section struct {
	ID          int64
	Code        string
	Title       string
	Description string
	SortOrder   int
	IsActive    bool
}

type Block struct {
	ID         int64
	SectionID  int64
	ChapterID  int64
	Code       string
	BlockType  BlockType
	Title      string
	SortOrder  int
	IsActive   bool
	Difficulty string
	Tags       []string
}

type Chapter struct {
	ID        int64
	SectionID int64
	Code      string
	Title     string
	SortOrder int
}

type BlockContent struct {
	BlockID         int64
	TheoryMD        string
	TaskMD          string
	SolutionMD      string
	ImageURLs       []string
	Difficulty      string
	Tags            []string
	LanguageCode    string
	SourceType      string
	SourcePath      string
	SourcePage      *int
	SourceChunkRef  string
}

type BlockRelation struct {
	FromBlockID  int64
	ToBlockID    int64
	RelationType string
	SortOrder    int
}

type ProgressRecord struct {
	ID          int64
	UserID      int64
	BlockID     int64
	Status      ProgressStatus
	CurrentStep FlowStep
	StartedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

type AttemptRecord struct {
	ID            int64
	UserID        int64
	BlockID       int64
	AttemptNo     int
	AnswerText    string
	LLMFeedbackMD string
	Score         *float64
	CreatedAt     time.Time
}

type SessionRecord struct {
	ID              int64
	UserID          int64
	ActiveSectionID *int64
	ActiveChapterID *int64
	ActiveBlockID   *int64
	FlowStep        FlowStep
	Mode            string
	UpdatedAt       time.Time
}
