package learning

type BlockType string
type FlowStep string

const (
	BlockTheory   BlockType = "theory"
	BlockTask     BlockType = "task"
	BlockSolution BlockType = "solution"

	StepTheory   FlowStep = "theory"
	StepTask     FlowStep = "task"
	StepAnswer   FlowStep = "answer"
	StepReview   FlowStep = "review"
	StepSolution FlowStep = "solution"
)
