BEGIN;

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL UNIQUE,
    username TEXT,
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sections (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chapters (
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

CREATE TABLE learning_blocks (
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

CREATE TABLE block_content (
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

CREATE TABLE block_relations (
    id BIGSERIAL PRIMARY KEY,
    from_block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    to_block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (from_block_id, to_block_id, relation_type),
    CHECK (from_block_id <> to_block_id)
);

CREATE TABLE user_progress (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'in_progress',
    current_step TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE (user_id, block_id)
);

CREATE TABLE user_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    attempt_no INTEGER NOT NULL,
    answer_text TEXT NOT NULL DEFAULT '',
    llm_feedback_md TEXT NOT NULL DEFAULT '',
    score NUMERIC(5,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    active_section_id BIGINT REFERENCES sections(id) ON DELETE SET NULL,
    active_chapter_id BIGINT REFERENCES chapters(id) ON DELETE SET NULL,
    active_block_id BIGINT REFERENCES learning_blocks(id) ON DELETE SET NULL,
    flow_step TEXT NOT NULL DEFAULT 'theory',
    mode TEXT NOT NULL DEFAULT 'learning',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id)
);

CREATE TABLE ingest_jobs (
    id BIGSERIAL PRIMARY KEY,
    section_id BIGINT REFERENCES sections(id) ON DELETE SET NULL,
    file_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    errors_json JSONB NOT NULL DEFAULT '[]'::JSONB,
    source_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE raw_chunks (
    id BIGSERIAL PRIMARY KEY,
    ingest_job_id BIGINT NOT NULL REFERENCES ingest_jobs(id) ON DELETE CASCADE,
    section_id BIGINT REFERENCES sections(id) ON DELETE SET NULL,
    chapter_id BIGINT REFERENCES chapters(id) ON DELETE SET NULL,
    chunk_index INTEGER NOT NULL,
    page_number INTEGER,
    text_md TEXT NOT NULL DEFAULT '',
    source_ref TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (ingest_job_id, chunk_index)
);

CREATE TABLE raw_images (
    id BIGSERIAL PRIMARY KEY,
    ingest_job_id BIGINT NOT NULL REFERENCES ingest_jobs(id) ON DELETE CASCADE,
    raw_chunk_id BIGINT REFERENCES raw_chunks(id) ON DELETE SET NULL,
    image_index INTEGER NOT NULL,
    page_number INTEGER,
    image_path TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    source_ref TEXT NOT NULL DEFAULT '',
    alt_text TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (ingest_job_id, image_index)
);

CREATE INDEX idx_chapters_section_sort_order ON chapters (section_id, sort_order);
CREATE INDEX idx_learning_blocks_chapter_sort_order ON learning_blocks (chapter_id, sort_order);
CREATE UNIQUE INDEX idx_user_progress_user_block ON user_progress (user_id, block_id);
CREATE INDEX idx_user_attempts_user_block_created_at ON user_attempts (user_id, block_id, created_at DESC);

COMMIT;
