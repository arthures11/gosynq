-- Create jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY,
    queue VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    max_retries INTEGER NOT NULL DEFAULT 0,
    run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority VARCHAR(50) NOT NULL DEFAULT 'normal',
    idempotency_key VARCHAR(255),
    locked_by VARCHAR(255),
    locked_at TIMESTAMPTZ,
    CONSTRAINT fk_queue CHECK (queue IN ('default', 'high', 'low', 'critical'))
);

-- Create job_attempts table
CREATE TABLE IF NOT EXISTS job_attempts (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    UNIQUE(job_id, attempt_number)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_queue ON jobs(queue);
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority);
CREATE INDEX IF NOT EXISTS idx_jobs_run_at ON jobs(run_at);
CREATE INDEX IF NOT EXISTS idx_jobs_idempotency ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_job_attempts_job_id ON job_attempts(job_id);