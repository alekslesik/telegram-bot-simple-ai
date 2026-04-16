# AI Telegram Interview Bot Design (MVP)

## Goal

Transform the current demo Telegram bot into an interview preparation bot for Go-focused algorithm training with:

- guided learning flow per item;
- persistent user progress;
- AI-assisted explanations and answer review;
- content ingestion from provided PDF materials into PostgreSQL.

This MVP includes two sections only:

1. `introduction` (theory only, optional)
2. `algorithms` (chapters with theory/tasks/solutions)

## Scope and Constraints

### In scope (MVP)

- Telegram bot flow with reply keyboard + inline step actions.
- Sequential and random learning modes.
- Optional steps for user answer and AI review.
- PostgreSQL-backed content/progress storage.
- Ingestion pipeline: parser-first + AI-assisted structuring.
- LLM provider abstraction with default OpenAI-compatible provider (DeepSeek-friendly).

### Out of scope (MVP)

- Other planned sections (`golang core`, `system design`, `advanced code`, `sorting`, `graphs`).
- Admin web panel.
- Complex recommendation engine.
- Multi-service microservice split.

## Product Flow

Primary user journey for algorithm items:

`theory -> task -> user answer (optional) -> review (optional) -> reference solution -> next`

### Rules

- `introduction` is recommended but not mandatory.
- User can skip answer step and still continue.
- User can skip AI review and go directly to reference solution.
- Bot always keeps progress state and resumes from the latest step.

## Content Model

Content is chapter-oriented and allows mixed sequence:

- theory block
- task block
- theory block
- task block
- ...

Each block has explicit type and order index inside chapter.

### Block Types

- `theory`
- `task`
- `solution`

Rules:

- one card is usually exactly one block;
- one card is either pure theory or a task;
- in rare cases a task is split across two cards: one `task` card and one separate `solution` card;
- a solution always exists, either embedded in the task card or stored in a separate solution card.

### Section Layout

- `introduction`
  - chapters with theory blocks only
- `algorithms`
  - chapters such as `two-pointers`, `hash-map`, etc.
  - mixed theory, task, and optional separate solution blocks

## Raw Content Layout on Host

Content is uploaded as files on the host in a directory tree:

- `content/raw/introduction/*.pdf`
- `content/raw/algorithms/<chapter-folder>/*.pdf`

Each chapter has its own folder, for example:

- `content/raw/algorithms/two-pointers/`
- `content/raw/algorithms/hash-map/`

### File Naming Reality

The ingestion pipeline must support the user's actual saved filenames, for example:

- `FireShot Capture 306 - Что такое хеш-таблица_ - algocode - [algocode.io].pdf`

Rules:

- Cyrillic filenames are allowed.
- The primary order key is the number after `FireShot Capture`.
- If multiple files share the same capture number inside a chapter, order them by file modification time (`mtime`) with newer files taking priority.
- If a capture number is missing, fall back to `mtime` ordering.
- Block type (`theory|task|solution`) is determined from PDF content, not from filename.

## Telegram UX

### Reply Keyboard (bottom menu)

- `📘 Введение`
- `📚 Алгоритмы (по порядку)`
- `🎲 Рандом задача`
- `📈 Мой прогресс`
- `⚙️ Настройки`

### Inline Actions by Step

- Theory: `К задаче` / `Дальше`
- Task: `✍️ Ответить` / `⏭ Пропустить ответ`
- Review: `🔍 Проверить` / `⏭ Пропустить разбор`
- Solution: `➡️ Следующая` / `🎲 Рандом`

### User Answer Format

The bot accepts free-form user responses in MVP:

- Go code
- textual explanation
- mixed code + explanation
- any other free-form answer

The review step analyzes whatever the user sent instead of enforcing a rigid submission format.

### Random Mode

- default: prioritize not completed tasks;
- optional filters: chapter and difficulty.

## Architecture (Modular Monolith)

Single deployable service with clear internal modules:

1. `transport/telegram`
   - update handling, keyboards, callback routing
2. `service/learning_flow`
   - step transitions and progress rules
3. `service/content`
   - chapter/block/task retrieval
4. `service/llm`
   - explanation/review/hint use-cases through provider interface
5. `service/ingest`
   - PDF import, parse, normalize, validate, publish
6. `repository/postgres`
   - data access for all domains

Layering rule: `transport -> service -> repository`.

## Data Model (PostgreSQL)

## Tables

- `users`
  - `id`, `telegram_user_id` (unique), profile fields, timestamps
- `sections`
  - `id`, `code`, `title`, `is_active`
- `chapters`
  - `id`, `section_id`, `code`, `title`, `sort_order`
- `learning_blocks`
  - `id`, `section_id`, `chapter_id`, `block_type` (`theory|task|solution`), `title`, `sort_order`, `is_active`
- `block_content`
  - `id`, `block_id`, `theory_md`, `task_md`, `solution_md`, `image_urls`, `difficulty`, `tags`, `language_code`, source metadata
- `block_relations`
  - `id`, `from_block_id`, `to_block_id`, `relation_type` (`task_solution`)
- `user_progress`
  - `id`, `user_id`, `block_id`, `status`, `current_step`, timestamps
- `user_attempts`
  - `id`, `user_id`, `block_id`, `attempt_no`, `answer_text`, `llm_feedback_md`, `score`, `created_at`
- `user_sessions`
  - `id`, `user_id`, `active_section_id`, `active_chapter_id`, `active_block_id`, `flow_step`, `mode`, `updated_at`
- `ingest_jobs`
  - `id`, `section_id`, `file_name`, `status`, timing, `errors_json`
- `raw_chunks` (staging)
  - `id`, `ingest_job_id`, `source_pdf`, pages, extracted text chunk
- `raw_images` (staging)
  - `id`, `ingest_job_id`, `source_pdf`, page, image_path

## Required indexes

- `users(telegram_user_id)` unique
- `chapters(section_id, sort_order)`
- `learning_blocks(chapter_id, sort_order)`
- `user_progress(user_id, block_id)` unique
- `user_attempts(user_id, block_id, created_at desc)`

## LLM Abstraction

Provider interface:

- `ExplainTheory`
- `ReviewAnswer`
- `GiveHint`
- `ExplainSolution`

MVP provider:

- `openai_compatible` adapter
- DeepSeek can be used via base URL + model config

Language behavior:

- Go is the default language in MVP;
- design should allow additional languages later without schema rewrite.

Configuration:

- `LLM_PROVIDER`
- `LLM_BASE_URL`
- `LLM_API_KEY`
- `LLM_MODEL`
- `LLM_TIMEOUT_SEC`

Guardrails:

- strict system prompts bound to current block context;
- token/output limits;
- timeout and short retry;
- fallback to non-AI solution view if provider fails.

## Ingestion Strategy: Parser-First + AI-Assisted Structuring

1. Parse readable PDF text to staging (`raw_chunks`, `raw_images`).
2. Apply deterministic parsing rules:
   - detect title, theory, condition, examples, solution areas;
   - use chapter-local `FireShot Capture N` order as initial sequence order.
3. Use AI optionally to classify/refine ambiguous chunk boundaries and infer block type when rules are inconclusive.
4. Validate and flag risky blocks:
   - missing task or solution;
   - too-short content;
   - duplicates.
5. Publish validated blocks to production tables.

Notes:

- text-readable PDFs are primary input format;
- image-only PDFs require OCR fallback and are lower quality.
- task-to-solution linking uses a hybrid strategy:
  - first by local order proximity inside a chapter;
  - then by title/content similarity checks;
  - ambiguous links are flagged as `needs_review`.

## Error Handling

- Missing/invalid block content: skip block safely and propose next block.
- LLM failure: continue flow with reference solution, mark review as system-skipped.
- Unexpected user action for current step: return instructional prompt and available buttons.

## Testing Plan (MVP)

- unit: state machine transitions for all steps and skip branches;
- unit: repository operations for progress/session updates;
- unit: LLM adapter behavior on timeout/error/empty response;
- integration: one ingestion sample from PDF text to published blocks;
- integration: end-to-end chat scenario from theory to next block.

## Observability

Structured logs:

- `user_id`, `section`, `chapter`, `block_id`, `step`
- `provider`, `model`, `latency_ms`, `result`

Metrics:

- completed blocks, skip rates, review usage rate;
- average transition time between steps;
- ingestion success/failure counts and flagged block counts.

## Delivery Plan (High-Level)

1. Introduce domain model + PostgreSQL schema + migrations.
2. Implement learning flow state machine and progress persistence.
3. Rework Telegram menus/actions to chapter/block flow.
4. Implement LLM provider abstraction + openai-compatible adapter.
5. Implement ingestion pipeline for readable PDFs.
6. Add tests, logs, and baseline metrics.

## Decisions Captured

- Modular monolith architecture.
- PostgreSQL in MVP.
- Section split: `introduction` + `algorithms`.
- `introduction` is optional.
- Flow allows skip for answer and review steps.
- LLM is provider-agnostic with default openai-compatible setup.
- PDF handling strategy is parser-first with optional AI assistance.
- Raw file naming supports `FireShot Capture N ...` with Cyrillic filenames.
- Duplicate capture numbers are resolved by file modification time.
- User submissions are free-form; review analyzes whatever was sent.
- Go is the default content language in MVP, with future multi-language extensibility.
