package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

type ContentRepository struct {
	db *sql.DB
}

func NewContentRepository(db *sql.DB) *ContentRepository {
	return &ContentRepository{db: db}
}

func (r *ContentRepository) GetSectionByCode(ctx context.Context, code string) (learning.Section, error) {
	const query = `
		SELECT id, code, title, description, sort_order, is_active
		FROM sections
		WHERE code = $1
	`

	row := r.db.QueryRowContext(ctx, query, code)

	var section learning.Section
	if err := row.Scan(
		&section.ID,
		&section.Code,
		&section.Title,
		&section.Description,
		&section.SortOrder,
		&section.IsActive,
	); err != nil {
		return learning.Section{}, fmt.Errorf("get section by code %q: %w", code, err)
	}

	return section, nil
}

func (r *ContentRepository) GetChapter(ctx context.Context, chapterID int64) (learning.Chapter, error) {
	const query = `
		SELECT id, section_id, code, title, sort_order
		FROM chapters
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, chapterID)

	var chapter learning.Chapter
	if err := row.Scan(
		&chapter.ID,
		&chapter.SectionID,
		&chapter.Code,
		&chapter.Title,
		&chapter.SortOrder,
	); err != nil {
		return learning.Chapter{}, fmt.Errorf("get chapter %d: %w", chapterID, err)
	}

	return chapter, nil
}

func (r *ContentRepository) ListChaptersBySection(ctx context.Context, sectionID int64) ([]learning.Chapter, error) {
	const query = `
		SELECT id, section_id, code, title, sort_order
		FROM chapters
		WHERE section_id = $1
		ORDER BY sort_order, id
	`

	rows, err := r.db.QueryContext(ctx, query, sectionID)
	if err != nil {
		return nil, fmt.Errorf("list chapters by section %d: %w", sectionID, err)
	}
	defer rows.Close()

	var chapters []learning.Chapter
	for rows.Next() {
		var chapter learning.Chapter
		if err := rows.Scan(
			&chapter.ID,
			&chapter.SectionID,
			&chapter.Code,
			&chapter.Title,
			&chapter.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan chapter for section %d: %w", sectionID, err)
		}

		chapters = append(chapters, chapter)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chapters for section %d: %w", sectionID, err)
	}

	return chapters, nil
}

func (r *ContentRepository) ListChapterBlocks(ctx context.Context, chapterID int64) ([]learning.Block, error) {
	const query = `
		SELECT id, section_id, chapter_id, code, block_type, title, sort_order, is_active
		FROM learning_blocks
		WHERE chapter_id = $1
		ORDER BY sort_order, id
	`

	rows, err := r.db.QueryContext(ctx, query, chapterID)
	if err != nil {
		return nil, fmt.Errorf("list chapter blocks %d: %w", chapterID, err)
	}
	defer rows.Close()

	var blocks []learning.Block
	for rows.Next() {
		block, err := scanBlock(rows)
		if err != nil {
			return nil, fmt.Errorf("scan block for chapter %d: %w", chapterID, err)
		}

		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocks for chapter %d: %w", chapterID, err)
	}

	return blocks, nil
}

func (r *ContentRepository) GetBlock(ctx context.Context, blockID int64) (learning.Block, error) {
	const query = `
		SELECT id, section_id, chapter_id, code, block_type, title, sort_order, is_active
		FROM learning_blocks
		WHERE id = $1
	`

	block, err := scanBlock(r.db.QueryRowContext(ctx, query, blockID))
	if err != nil {
		return learning.Block{}, fmt.Errorf("get block %d: %w", blockID, err)
	}

	return block, nil
}

func (r *ContentRepository) GetBlockContent(ctx context.Context, blockID int64) (learning.BlockContent, error) {
	const query = `
		SELECT
			block_id,
			theory_md,
			task_md,
			solution_md,
			image_urls,
			difficulty,
			tags,
			language_code,
			source_type,
			source_path,
			source_page,
			source_chunk_ref,
			source_metadata
		FROM block_content
		WHERE block_id = $1
	`

	row := r.db.QueryRowContext(ctx, query, blockID)

	var content learning.BlockContent
	var sourcePage sql.NullInt64
	var sourceMetadata []byte
	if err := row.Scan(
		&content.BlockID,
		&content.TheoryMD,
		&content.TaskMD,
		&content.SolutionMD,
		&content.ImageURLs,
		&content.Difficulty,
		&content.Tags,
		&content.LanguageCode,
		&content.SourceType,
		&content.SourcePath,
		&sourcePage,
		&content.SourceChunkRef,
		&sourceMetadata,
	); err != nil {
		return learning.BlockContent{}, fmt.Errorf("get block content %d: %w", blockID, err)
	}

	if sourcePage.Valid {
		page := int(sourcePage.Int64)
		content.SourcePage = &page
	}

	if len(sourceMetadata) > 0 {
		if err := json.Unmarshal(sourceMetadata, &content.SourceMetadata); err != nil {
			return learning.BlockContent{}, fmt.Errorf("decode block content metadata %d: %w", blockID, err)
		}
	}

	return content, nil
}

func (r *ContentRepository) ListBlockRelations(ctx context.Context, blockID int64) ([]learning.BlockRelation, error) {
	const query = `
		SELECT from_block_id, to_block_id, relation_type, sort_order
		FROM block_relations
		WHERE from_block_id = $1
		ORDER BY sort_order, id
	`

	rows, err := r.db.QueryContext(ctx, query, blockID)
	if err != nil {
		return nil, fmt.Errorf("list block relations %d: %w", blockID, err)
	}
	defer rows.Close()

	var relations []learning.BlockRelation
	for rows.Next() {
		var relation learning.BlockRelation
		var relationType string
		if err := rows.Scan(
			&relation.FromBlockID,
			&relation.ToBlockID,
			&relationType,
			&relation.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan block relation %d: %w", blockID, err)
		}

		relation.RelationType = learning.BlockRelationType(relationType)
		relations = append(relations, relation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate block relations %d: %w", blockID, err)
	}

	return relations, nil
}

type blockScanner interface {
	Scan(dest ...any) error
}

func scanBlock(scanner blockScanner) (learning.Block, error) {
	var block learning.Block
	var blockType string
	if err := scanner.Scan(
		&block.ID,
		&block.SectionID,
		&block.ChapterID,
		&block.Code,
		&blockType,
		&block.Title,
		&block.SortOrder,
		&block.IsActive,
	); err != nil {
		return learning.Block{}, err
	}

	block.BlockType = learning.BlockType(blockType)
	return block, nil
}
