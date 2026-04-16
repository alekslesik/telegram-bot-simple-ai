BEGIN;

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
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
    chapter_id BIGINT NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    block_type TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chapter_id, code)
);

CREATE TABLE block_content (
    id BIGSERIAL PRIMARY KEY,
    block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    content_type TEXT NOT NULL,
    body TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (block_id, content_type)
);

CREATE TABLE block_relations (
    id BIGSERIAL PRIMARY KEY,
    parent_block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    child_block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (parent_block_id, child_block_id, relation_type),
    CHECK (parent_block_id <> child_block_id)
);

CREATE TABLE user_progress (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chapter_id BIGINT NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    current_block_id BIGINT REFERENCES learning_blocks(id) ON DELETE SET NULL,
    current_step TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'in_progress',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE (user_id, chapter_id)
);

CREATE TABLE user_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    block_id BIGINT NOT NULL REFERENCES learning_blocks(id) ON DELETE CASCADE,
    answer_text TEXT NOT NULL DEFAULT '',
    review_text TEXT NOT NULL DEFAULT '',
    score NUMERIC(5,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_key TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL DEFAULT 'active',
    section_id BIGINT REFERENCES sections(id) ON DELETE SET NULL,
    chapter_id BIGINT REFERENCES chapters(id) ON DELETE SET NULL,
    context JSONB NOT NULL DEFAULT '{}'::JSONB,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

CREATE TABLE ingest_jobs (
    id BIGSERIAL PRIMARY KEY,
    source_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    error_text TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE raw_chunks (
    id BIGSERIAL PRIMARY KEY,
    ingest_job_id BIGINT NOT NULL REFERENCES ingest_jobs(id) ON DELETE CASCADE,
    chapter_id BIGINT REFERENCES chapters(id) ON DELETE SET NULL,
    chunk_index INTEGER NOT NULL,
    source_ref TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (ingest_job_id, chunk_index)
);

CREATE TABLE raw_images (
    id BIGSERIAL PRIMARY KEY,
    ingest_job_id BIGINT NOT NULL REFERENCES ingest_jobs(id) ON DELETE CASCADE,
    raw_chunk_id BIGINT REFERENCES raw_chunks(id) ON DELETE SET NULL,
    image_index INTEGER NOT NULL,
    source_ref TEXT NOT NULL DEFAULT '',
    alt_text TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (ingest_job_id, image_index)
);

COMMIT;
