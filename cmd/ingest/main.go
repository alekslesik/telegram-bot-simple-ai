package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
	ingestservice "github.com/alekslesik/telegram-bot-simple/internal/service/ingest"
	"github.com/alekslesik/telegram-bot-simple/internal/storage/postgres"
)

func main() {
	root := flag.String("root", "", "root directory with readable PDF files")
	write := flag.Bool("write", false, "write parsed blocks to postgres")
	flag.Parse()

	if strings.TrimSpace(*root) == "" {
		log.Fatal("flag -root is required")
	}

	svc := ingestservice.New()
	files, err := svc.PrepareInputs(*root)
	if err != nil {
		log.Fatalf("prepare inputs: %v", err)
	}

	if !*write {
		fmt.Printf("would ingest %d file(s) from %s\n", len(files), *root)
		for _, file := range files {
			fmt.Printf("- %s\n", file.Path)
		}
		return
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required when -write is enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := postgres.Open(ctx, dbURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := ensureContentSchema(ctx, db); err != nil {
		log.Fatalf("ensure content schema: %v", err)
	}

	ingested := 0
	for i, file := range files {
		rawText, err := ingestservice.ExtractPDFText(file.Path)
		if err != nil {
			log.Printf("skip %s: extract text failed: %v", file.Path, err)
			continue
		}
		if strings.TrimSpace(rawText) == "" {
			log.Printf("skip %s: empty extracted text", file.Path)
			continue
		}

		meta, err := ingestservice.BuildDocumentMeta(*root, file, i+1)
		if err != nil {
			log.Printf("skip %s: build metadata failed: %v", file.Path, err)
			continue
		}

		sectionID, err := upsertSection(ctx, db, meta)
		if err != nil {
			log.Printf("skip %s: upsert section failed: %v", file.Path, err)
			continue
		}
		chapterID, err := upsertChapter(ctx, db, sectionID, meta)
		if err != nil {
			log.Printf("skip %s: upsert chapter failed: %v", file.Path, err)
			continue
		}

		blockType := ingestservice.DetectBlockType(rawText)
		if meta.SectionCode == "introduction" {
			// Introduction is theory-first content.
			blockType = learning.BlockTheory
		}

		contentTheory, contentTask, contentSolution := splitContentByBlockType(blockType, rawText)
		blockID, err := upsertBlock(ctx, db, sectionID, chapterID, meta, blockType)
		if err != nil {
			log.Printf("skip %s: upsert block failed: %v", file.Path, err)
			continue
		}
		if err := upsertBlockContent(ctx, db, blockID, contentTheory, contentTask, contentSolution, file.Path); err != nil {
			log.Printf("skip %s: upsert block content failed: %v", file.Path, err)
			continue
		}

		ingested++
		fmt.Printf("ingested: %s\n", file.Path)
	}

	fmt.Printf("ingested %d/%d file(s) from %s\n", ingested, len(files), *root)
}

func ensureContentSchema(ctx context.Context, db *sql.DB) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS sections (
			id BIGSERIAL PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS chapters (
			id BIGSERIAL PRIMARY KEY,
			section_id BIGINT NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
			code TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (section_id, code)
		);
		CREATE TABLE IF NOT EXISTS learning_blocks (
			id BIGSERIAL PRIMARY KEY,
			section_id BIGINT NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
			chapter_id BIGINT NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
			code TEXT NOT NULL,
			block_type TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (chapter_id, code)
		);
		CREATE TABLE IF NOT EXISTS block_content (
			id BIGSERIAL PRIMARY KEY,
			block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
			theory_md TEXT NOT NULL DEFAULT '',
			task_md TEXT NOT NULL DEFAULT '',
			solution_md TEXT NOT NULL DEFAULT '',
			image_urls TEXT[] NOT NULL DEFAULT '{}',
			difficulty TEXT NOT NULL DEFAULT '',
			tags TEXT[] NOT NULL DEFAULT '{}',
			language_code TEXT NOT NULL DEFAULT 'ru',
			source_type TEXT NOT NULL DEFAULT '',
			source_path TEXT NOT NULL DEFAULT '',
			source_page INTEGER,
			source_chunk_ref TEXT NOT NULL DEFAULT '',
			source_metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (block_id)
		);
	`
	_, err := db.ExecContext(ctx, ddl)
	return err
}

func upsertSection(ctx context.Context, db *sql.DB, meta ingestservice.DocumentMeta) (int64, error) {
	const q = `
		INSERT INTO sections (code, title, description, sort_order, is_active)
		VALUES ($1, $2, '', 0, TRUE)
		ON CONFLICT (code) DO UPDATE
		SET title = EXCLUDED.title,
			is_active = TRUE,
			updated_at = NOW()
		RETURNING id
	`
	var id int64
	if err := db.QueryRowContext(ctx, q, meta.SectionCode, meta.SectionTitle).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func upsertChapter(ctx context.Context, db *sql.DB, sectionID int64, meta ingestservice.DocumentMeta) (int64, error) {
	const q = `
		INSERT INTO chapters (section_id, code, title, description, sort_order)
		VALUES ($1, $2, $3, '', 0)
		ON CONFLICT (section_id, code) DO UPDATE
		SET title = EXCLUDED.title,
			updated_at = NOW()
		RETURNING id
	`
	var id int64
	if err := db.QueryRowContext(ctx, q, sectionID, meta.ChapterCode, meta.ChapterTitle).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func upsertBlock(
	ctx context.Context,
	db *sql.DB,
	sectionID int64,
	chapterID int64,
	meta ingestservice.DocumentMeta,
	blockType learning.BlockType,
) (int64, error) {
	const q = `
		INSERT INTO learning_blocks (section_id, chapter_id, code, block_type, title, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		ON CONFLICT (chapter_id, code) DO UPDATE
		SET block_type = EXCLUDED.block_type,
			title = EXCLUDED.title,
			sort_order = EXCLUDED.sort_order,
			is_active = TRUE,
			updated_at = NOW()
		RETURNING id
	`
	var id int64
	if err := db.QueryRowContext(ctx, q, sectionID, chapterID, meta.BlockCode, string(blockType), meta.BlockTitle, meta.SortOrder).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func upsertBlockContent(
	ctx context.Context,
	db *sql.DB,
	blockID int64,
	theoryMD string,
	taskMD string,
	solutionMD string,
	sourcePath string,
) error {
	const q = `
		INSERT INTO block_content (
			block_id, theory_md, task_md, solution_md,
			language_code, source_type, source_path, source_metadata
		)
		VALUES ($1, $2, $3, $4, 'ru', 'pdf', $5, '{}'::jsonb)
		ON CONFLICT (block_id) DO UPDATE
		SET theory_md = EXCLUDED.theory_md,
			task_md = EXCLUDED.task_md,
			solution_md = EXCLUDED.solution_md,
			source_path = EXCLUDED.source_path,
			updated_at = NOW()
	`
	_, err := db.ExecContext(ctx, q, blockID, theoryMD, taskMD, solutionMD, sourcePath)
	return err
}

func splitContentByBlockType(blockType learning.BlockType, text string) (string, string, string) {
	content := strings.TrimSpace(text)
	switch blockType {
	case learning.BlockSolution:
		return "", "", content
	case learning.BlockTask:
		return "", content, ""
	default:
		return content, "", ""
	}
}
