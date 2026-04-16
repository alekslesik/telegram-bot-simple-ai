# AI Interview Bot MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first working MVP of the Telegram interview bot with PostgreSQL persistence, chapter/block learning flow, OpenAI-compatible LLM integration, and readable-PDF ingestion for `introduction` and `algorithms`.

**Architecture:** Keep a modular monolith inside the current Go bot. Add focused service and repository packages for content, progress, sessions, LLM, and ingestion; keep Telegram handlers thin and move decision-making into a learning-flow state machine. Use parser-first ingestion with optional AI assistance only for ambiguous classification.

**Tech Stack:** Go, `go-telegram-bot-api`, PostgreSQL, SQL migrations, OpenAI-compatible HTTP client, readable PDF text extraction, Docker Compose, Go tests

---

## File Structure Map

### Existing files to modify

- Modify: `cmd/bot/main.go`
  - Wire PostgreSQL, repositories, services, and the new handler dependencies.
- Modify: `internal/bot/handlers.go`
  - Replace demo-only command routing with section/chapter/block flow and callback actions.
- Modify: `README.md`
  - Document raw content directory layout, env vars, and ingestion commands.
- Modify: `.env.example`
  - Add PostgreSQL and LLM env vars used by MVP.
- Modify: `docker-compose.yaml`
  - Add PostgreSQL service for local development.
- Modify: `docker-compose.prod.yaml`
  - Add PostgreSQL dependency or connection env passthrough for runtime deployment.
- Modify: `Makefile`
  - Add migration, ingest, and seed-style helper commands.

### New application files

- Create: `internal/config/config.go`
  - Read env vars for Telegram, PostgreSQL, and LLM.
- Create: `internal/storage/postgres/connect.go`
  - Open DB connection and expose ping/close helpers.
- Create: `internal/storage/postgres/migrations/001_initial.sql`
  - Create `users`, `sections`, `chapters`, `learning_blocks`, `block_content`, `block_relations`, `user_progress`, `user_attempts`, `user_sessions`, `ingest_jobs`, `raw_chunks`, `raw_images`.
- Create: `internal/domain/learning/types.go`
  - Shared enums and DTOs for sections, blocks, steps, and progress state.
- Create: `internal/repository/content_repository.go`
  - Repository contract for sections/chapters/blocks.
- Create: `internal/repository/progress_repository.go`
  - Repository contract for progress, attempts, and sessions.
- Create: `internal/storage/postgres/content_repository.go`
  - PostgreSQL implementation of content reads.
- Create: `internal/storage/postgres/progress_repository.go`
  - PostgreSQL implementation of session/progress writes.
- Create: `internal/service/learningflow/service.go`
  - State machine transitions and resume/random-next logic.
- Create: `internal/service/learningflow/service_test.go`
  - Unit tests for flow transitions and skip branches.
- Create: `internal/service/llm/provider.go`
  - LLM provider interface and request/response DTOs.
- Create: `internal/service/llm/openai_compatible.go`
  - OpenAI-compatible adapter (DeepSeek-friendly).
- Create: `internal/service/llm/openai_compatible_test.go`
  - HTTP-mocked tests for timeout and response handling.
- Create: `internal/service/content/service.go`
  - Read section/chapter/block data and build display payloads.
- Create: `internal/service/ingest/service.go`
  - Ingestion orchestrator for host content directories.
- Create: `internal/service/ingest/order.go`
  - `FireShot Capture N` sorting and `mtime` fallback.
- Create: `internal/service/ingest/classify.go`
  - Parser-first block type detection with optional AI-assisted fallback.
- Create: `internal/service/ingest/service_test.go`
  - Unit/integration-like tests for ordering and task/solution linking.

### New command files

- Create: `cmd/ingest/main.go`
  - CLI entrypoint to ingest `content/raw/...` into PostgreSQL.

### New tests

- Create: `internal/storage/postgres/progress_repository_test.go`
  - Repository tests against a disposable test DB or transaction-backed DB.
- Create: `internal/bot/handlers_flow_test.go`
  - End-to-end-ish handler tests for theory -> task -> review -> solution navigation.

## Task 1: Add Runtime Config and PostgreSQL Bootstrap

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/storage/postgres/connect.go`
- Modify: `.env.example`
- Modify: `cmd/bot/main.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing config test**

```go
func TestFromEnvReadsPostgresAndLLM(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://bot:bot@localhost:5432/bot?sslmode=disable")
	t.Setenv("LLM_PROVIDER", "openai_compatible")
	t.Setenv("LLM_BASE_URL", "https://api.deepseek.com")
	t.Setenv("LLM_MODEL", "deepseek-chat")

	cfg := FromEnv()

	if cfg.DatabaseURL == "" || cfg.LLMProvider == "" || cfg.LLMBaseURL == "" || cfg.LLMModel == "" {
		t.Fatal("expected config fields to be populated from env")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestFromEnvReadsPostgresAndLLM -v`  
Expected: FAIL with missing package/file or undefined `FromEnv`

- [ ] **Step 3: Write minimal config implementation**

```go
type Config struct {
	Token       string
	Username    string
	DatabaseURL string
	LLMProvider string
	LLMBaseURL  string
	LLMAPIKey   string
	LLMModel    string
}

func FromEnv() Config {
	return Config{
		Token:       strings.TrimSpace(os.Getenv("TOKEN")),
		Username:    strings.TrimSpace(os.Getenv("USERNAME")),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		LLMProvider: strings.TrimSpace(os.Getenv("LLM_PROVIDER")),
		LLMBaseURL:  strings.TrimSpace(os.Getenv("LLM_BASE_URL")),
		LLMAPIKey:   strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		LLMModel:    strings.TrimSpace(os.Getenv("LLM_MODEL")),
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestFromEnvReadsPostgresAndLLM -v`  
Expected: PASS

- [ ] **Step 5: Add DB connector and wire it in `main`**

```go
db, err := postgres.Open(ctx, cfg.DatabaseURL)
if err != nil {
	log.Fatalf("failed to connect postgres: %v", err)
}
defer db.Close()
```

- [ ] **Step 6: Update `.env.example`**

```env
DATABASE_URL=postgres://bot:bot@localhost:5432/telegram_bot_simple_ai?sslmode=disable
LLM_PROVIDER=openai_compatible
LLM_BASE_URL=https://api.deepseek.com
LLM_API_KEY=replace-me
LLM_MODEL=deepseek-chat
LLM_TIMEOUT_SEC=30
```

- [ ] **Step 7: Commit**

```bash
git add .env.example cmd/bot/main.go internal/config/config.go internal/storage/postgres/connect.go internal/config/config_test.go
git commit -m "feat(config): add postgres and llm runtime settings"
```

## Task 2: Add Database Schema and Repository Contracts

**Files:**
- Create: `internal/storage/postgres/migrations/001_initial.sql`
- Create: `internal/domain/learning/types.go`
- Create: `internal/repository/content_repository.go`
- Create: `internal/repository/progress_repository.go`
- Modify: `Makefile`
- Test: `internal/domain/learning/types_test.go`

- [ ] **Step 1: Write the failing enum/state test**

```go
func TestValidFlowSteps(t *testing.T) {
	steps := []FlowStep{StepTheory, StepTask, StepAnswer, StepReview, StepSolution}
	if len(steps) != 5 {
		t.Fatalf("expected 5 flow steps, got %d", len(steps))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/learning -run TestValidFlowSteps -v`  
Expected: FAIL with undefined `FlowStep`

- [ ] **Step 3: Add minimal domain types**

```go
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
```

- [ ] **Step 4: Create initial migration**

```sql
CREATE TABLE sections (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE chapters (
    id BIGSERIAL PRIMARY KEY,
    section_id BIGINT NOT NULL REFERENCES sections(id),
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    sort_order INT NOT NULL
);
```

- [ ] **Step 5: Add repository interfaces**

```go
type ContentRepository interface {
	GetSectionByCode(ctx context.Context, code string) (Section, error)
	ListChapterBlocks(ctx context.Context, chapterID int64) ([]Block, error)
	GetBlock(ctx context.Context, blockID int64) (Block, error)
}
```

- [ ] **Step 6: Add migration helpers to `Makefile`**

```make
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down
```

- [ ] **Step 7: Run tests**

Run: `go test ./internal/domain/learning -v`  
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add Makefile internal/domain/learning/types.go internal/domain/learning/types_test.go internal/repository/content_repository.go internal/repository/progress_repository.go internal/storage/postgres/migrations/001_initial.sql
git commit -m "feat(storage): add learning schema and repository contracts"
```

## Task 3: Implement Learning Flow State Machine

**Files:**
- Create: `internal/service/learningflow/service.go`
- Create: `internal/service/learningflow/service_test.go`
- Create: `internal/storage/postgres/progress_repository.go`
- Test: `internal/service/learningflow/service_test.go`

- [ ] **Step 1: Write the failing transition test**

```go
func TestNextFromTaskSkipAnswerMovesToSolution(t *testing.T) {
	svc := New(nil, nil)
	next := svc.NextStep(BlockTask, StepTask, ActionSkipAnswer)
	if next != StepSolution {
		t.Fatalf("expected %q, got %q", StepSolution, next)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/learningflow -run TestNextFromTaskSkipAnswerMovesToSolution -v`  
Expected: FAIL with undefined `New` or `ActionSkipAnswer`

- [ ] **Step 3: Add minimal state machine**

```go
func (s *Service) NextStep(blockType learning.BlockType, current learning.FlowStep, action Action) learning.FlowStep {
	switch {
	case blockType == learning.BlockTheory:
		return learning.StepTask
	case blockType == learning.BlockTask && action == ActionSkipAnswer:
		return learning.StepSolution
	case blockType == learning.BlockTask && action == ActionSubmitAnswer:
		return learning.StepReview
	default:
		return learning.StepSolution
	}
}
```

- [ ] **Step 4: Add repository-backed session save method**

```go
func (s *Service) SaveSessionStep(ctx context.Context, userID, blockID int64, step learning.FlowStep) error {
	return s.progressRepo.UpsertSession(ctx, repository.SessionState{
		UserID:      userID,
		ActiveBlock: blockID,
		FlowStep:    step,
	})
}
```

- [ ] **Step 5: Run flow tests**

Run: `go test ./internal/service/learningflow -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/learningflow/service.go internal/service/learningflow/service_test.go internal/storage/postgres/progress_repository.go
git commit -m "feat(flow): add learning state machine"
```

## Task 4: Implement Content Reads and Task/Solution Linking

**Files:**
- Create: `internal/service/content/service.go`
- Create: `internal/storage/postgres/content_repository.go`
- Test: `internal/service/content/service_test.go`

- [ ] **Step 1: Write the failing content payload test**

```go
func TestBuildTaskPayloadIncludesSolutionHint(t *testing.T) {
	svc := New(fakeRepo{})
	payload, err := svc.BuildBlockPayload(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if payload.BlockType != learning.BlockTask {
		t.Fatalf("expected task payload, got %s", payload.BlockType)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/content -run TestBuildTaskPayloadIncludesSolutionHint -v`  
Expected: FAIL with missing service/repo

- [ ] **Step 3: Add minimal payload builder**

```go
type BlockPayload struct {
	Title     string
	BlockType learning.BlockType
	TheoryMD  string
	TaskMD    string
	Solution  string
}
```

- [ ] **Step 4: Implement SQL relation lookup**

```go
const getLinkedSolution = `
SELECT bc.solution_md
FROM block_relations br
JOIN block_content bc ON bc.block_id = br.to_block_id
WHERE br.from_block_id = $1 AND br.relation_type = 'task_solution'
LIMIT 1
`
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/service/content -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/content/service.go internal/service/content/service_test.go internal/storage/postgres/content_repository.go
git commit -m "feat(content): add block payload loading and solution links"
```

## Task 5: Rework Telegram Menus and Callback Flow

**Files:**
- Modify: `internal/bot/handlers.go`
- Create: `internal/bot/handlers_flow_test.go`
- Modify: `cmd/bot/main.go`
- Test: `internal/bot/handlers_flow_test.go`

- [ ] **Step 1: Write the failing handler flow test**

```go
func TestTaskCallbackSkipAnswerShowsSolution(t *testing.T) {
	bot := &fakeTelegram{}
	h := Handlers{Bot: bot, Learning: fakeLearningService{}}

	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID: "1",
		Data: "flow:skip_answer:42",
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 10}},
	})

	if len(bot.sent) == 0 {
		t.Fatal("expected a reply message")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bot -run TestTaskCallbackSkipAnswerShowsSolution -v`  
Expected: FAIL with missing `Learning` field or callback branch

- [ ] **Step 3: Add focused callback parsing**

```go
parts := strings.Split(strings.TrimSpace(q.Data), ":")
if len(parts) == 3 && parts[0] == "flow" {
	action := parts[1]
	blockID, _ := strconv.ParseInt(parts[2], 10, 64)
	h.handleFlowAction(q, action, blockID)
	return
}
```

- [ ] **Step 4: Update reply keyboard**

```go
return tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📘 Введение"),
		tgbotapi.NewKeyboardButton("📚 Алгоритмы (по порядку)"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("🎲 Рандом задача"),
		tgbotapi.NewKeyboardButton("📈 Мой прогресс"),
	),
)
```

- [ ] **Step 5: Run handler tests**

Run: `go test ./internal/bot -run TestTaskCallbackSkipAnswerShowsSolution -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/bot/main.go internal/bot/handlers.go internal/bot/handlers_flow_test.go
git commit -m "feat(bot): add chapter flow callbacks and menus"
```

## Task 6: Implement OpenAI-Compatible LLM Adapter

**Files:**
- Create: `internal/service/llm/provider.go`
- Create: `internal/service/llm/openai_compatible.go`
- Create: `internal/service/llm/openai_compatible_test.go`
- Test: `internal/service/llm/openai_compatible_test.go`

- [ ] **Step 1: Write the failing adapter test**

```go
func TestReviewAnswerUsesChatCompletions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"Looks good"}}]}`))
	}))
	defer server.Close()

	client := NewOpenAICompatible(Config{BaseURL: server.URL, APIKey: "x", Model: "deepseek-chat"})
	got, err := client.ReviewAnswer(context.Background(), ReviewInput{})
	if err != nil || got.Markdown == "" {
		t.Fatal("expected review response")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/llm -run TestReviewAnswerUsesChatCompletions -v`  
Expected: FAIL with missing adapter

- [ ] **Step 3: Add minimal provider contract**

```go
type Provider interface {
	ReviewAnswer(ctx context.Context, input ReviewInput) (ReviewResult, error)
	ExplainTheory(ctx context.Context, input TheoryInput) (string, error)
	ExplainSolution(ctx context.Context, input SolutionInput) (string, error)
}
```

- [ ] **Step 4: Add HTTP adapter**

```go
reqBody := map[string]any{
	"model": a.cfg.Model,
	"messages": []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userPrompt},
	},
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/service/llm -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/llm/provider.go internal/service/llm/openai_compatible.go internal/service/llm/openai_compatible_test.go
git commit -m "feat(llm): add openai-compatible provider adapter"
```

## Task 7: Build Readable-PDF Ingestion CLI

**Files:**
- Create: `cmd/ingest/main.go`
- Create: `internal/service/ingest/service.go`
- Create: `internal/service/ingest/order.go`
- Create: `internal/service/ingest/classify.go`
- Create: `internal/service/ingest/service_test.go`
- Modify: `Makefile`
- Test: `internal/service/ingest/service_test.go`

- [ ] **Step 1: Write the failing ordering test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ingest -run TestSortFilesByCaptureThenMtime -v`  
Expected: FAIL with missing `SortInputFiles`

- [ ] **Step 3: Add ordering helper**

```go
var captureRe = regexp.MustCompile(`(?i)FireShot Capture\s+(\d+)`)

func extractOrder(name string) (int, bool) {
	m := captureRe.FindStringSubmatch(name)
	if len(m) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}
```

- [ ] **Step 4: Add block classifier skeleton**

```go
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
```

- [ ] **Step 5: Add CLI command and make target**

```make
ingest:
	go run ./cmd/ingest -root ./content/raw
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/service/ingest -v`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add Makefile cmd/ingest/main.go internal/service/ingest/service.go internal/service/ingest/order.go internal/service/ingest/classify.go internal/service/ingest/service_test.go
git commit -m "feat(ingest): add readable pdf import pipeline"
```

## Task 8: Wire Observability, Docs, and End-to-End Checks

**Files:**
- Modify: `README.md`
- Modify: `docker-compose.yaml`
- Modify: `docker-compose.prod.yaml`
- Modify: `Makefile`
- Test: `internal/bot/handlers_flow_test.go`

- [ ] **Step 1: Write the failing README contract test (manual checklist)**

```text
README must mention:
- content/raw/introduction
- content/raw/algorithms/<chapter-folder>
- FireShot Capture ordering
- DATABASE_URL and LLM_* env vars
- make ingest
```

- [ ] **Step 2: Update local compose**

```yaml
services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_DB: telegram_bot_simple_ai
      POSTGRES_USER: bot
      POSTGRES_PASSWORD: bot
    ports:
      - "5432:5432"
```

- [ ] **Step 3: Add structured flow log call**

```go
h.Logger.Info("learning flow step",
	"user_id", userID,
	"chapter", payload.ChapterCode,
	"block_id", payload.BlockID,
	"step", payload.Step,
)
```

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/bot ./internal/service/... ./internal/storage/... -v`  
Expected: PASS

- [ ] **Step 5: Run repository checks**

Run: `make preprod`  
Expected: PASS except local `docker build` may still require Docker daemon if not running

- [ ] **Step 6: Commit**

```bash
git add README.md docker-compose.yaml docker-compose.prod.yaml Makefile internal/bot/handlers.go
git commit -m "docs: add ai interview bot runtime and ingest usage"
```

## Self-Review Checklist

### Spec coverage

- Telegram flow: covered by Tasks 3, 4, 5
- PostgreSQL schema and persistence: covered by Tasks 1, 2, 3, 4
- LLM abstraction: covered by Task 6
- Readable-PDF ingestion: covered by Task 7
- Host content layout and file ordering: covered by Tasks 7 and 8
- Tests/observability/docs: covered by Task 8

### Placeholder scan

- No `TODO`, `TBD`, or “implement later” placeholders remain.
- Each code-writing task includes concrete snippets.
- Each validation step includes an exact command and expected result.

### Type consistency

- Shared enums live in `internal/domain/learning/types.go`
- `learning.BlockType` and `learning.FlowStep` are reused across services
- Repository contracts are introduced before service implementations

## Recommended Execution Order

1. Task 1
2. Task 2
3. Task 3
4. Task 4
5. Task 5
6. Task 6
7. Task 7
8. Task 8
