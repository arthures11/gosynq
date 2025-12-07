package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/arthures11/gosynq/internal/models"
	"github.com/lib/pq"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateJob(ctx context.Context, job *models.Job) error {
	query := `
		INSERT INTO jobs (
			id, queue, payload, max_retries, run_at, priority, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx,
		query,
		job.ID, job.Queue, job.Payload, job.MaxRetries, job.RunAt,
		job.Priority, job.IdempotencyKey,
	).Scan(&job.CreatedAt, &job.UpdatedAt)

	return err
}

func (r *PostgresRepository) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	query := `
		SELECT id, queue, payload, max_retries, run_at, created_at, updated_at,
		       status, priority, idempotency_key, locked_by, locked_at
		FROM jobs
		WHERE id = $1
	`

	var job models.Job
	var lockedBy sql.NullString
	var lockedAt pq.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.Queue, &job.Payload, &job.MaxRetries, &job.RunAt,
		&job.CreatedAt, &job.UpdatedAt, &job.Status, &job.Priority,
		&job.IdempotencyKey, &lockedBy, &lockedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if lockedBy.Valid {
		job.LockedBy = lockedBy.String
	}

	if lockedAt.Valid {
		job.LockedAt = &lockedAt.Time
	}

	return &job, nil
}

func (r *PostgresRepository) PickJob(ctx context.Context, workerID string, timeout time.Duration) (*models.Job, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Atomic job pickup with SKIP LOCKED
	query := `
		SELECT id, queue, payload, max_retries, run_at, created_at, updated_at,
		       status, priority, idempotency_key
		FROM jobs
		WHERE status = 'pending'
		AND run_at <= NOW()
		ORDER BY priority DESC, created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`

	var job models.Job
	err = tx.QueryRowContext(ctx, query).Scan(
		&job.ID, &job.Queue, &job.Payload, &job.MaxRetries, &job.RunAt,
		&job.CreatedAt, &job.UpdatedAt, &job.Status, &job.Priority,
		&job.IdempotencyKey,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Update job status to processing and set lock
	updateQuery := `
		UPDATE jobs
		SET status = 'processing', locked_by = $1, locked_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`

	_, err = tx.ExecContext(ctx, updateQuery, workerID, job.ID)
	if err != nil {
		return nil, err
	}

	return &job, tx.Commit()
}

func (r *PostgresRepository) UpdateJobStatus(ctx context.Context, jobID string, status models.JobStatus) error {
	query := `
		UPDATE jobs
		SET status = $1, locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, status, jobID)
	return err
}

func (r *PostgresRepository) UpdateJobForRetry(ctx context.Context, jobID string, runAt time.Time) error {
	query := `
		UPDATE jobs
		SET status = 'pending', run_at = $1, locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, runAt, jobID)
	return err
}

func (r *PostgresRepository) CreateJobAttempt(ctx context.Context, attempt *models.JobAttempt) error {
	query := `
		INSERT INTO job_attempts (
			id, job_id, attempt_number, started_at, completed_at, status, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx,
		query,
		attempt.ID, attempt.JobID, attempt.AttemptNumber, attempt.StartedAt,
		attempt.CompletedAt, attempt.Status, attempt.ErrorMessage,
	)
	return err
}

func (r *PostgresRepository) ListJobs(ctx context.Context, statusFilter string, queueFilter string, limit int) ([]*models.Job, error) {
	query := `
		SELECT id, queue, payload, max_retries, run_at, created_at, updated_at,
		       status, priority, idempotency_key, locked_by, locked_at
		FROM jobs
		WHERE ($1 = '' OR status = $1)
		AND ($2 = '' OR queue = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, statusFilter, queueFilter, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		var job models.Job
		var lockedBy sql.NullString
		var lockedAt pq.NullTime

		err := rows.Scan(
			&job.ID, &job.Queue, &job.Payload, &job.MaxRetries, &job.RunAt,
			&job.CreatedAt, &job.UpdatedAt, &job.Status, &job.Priority,
			&job.IdempotencyKey, &lockedBy, &lockedAt,
		)
		if err != nil {
			return nil, err
		}

		if lockedBy.Valid {
			job.LockedBy = lockedBy.String
		}

		if lockedAt.Valid {
			job.LockedAt = &lockedAt.Time
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

func (r *PostgresRepository) GetJobAttempts(ctx context.Context, jobID string) ([]*models.JobAttempt, error) {
	query := `
		SELECT id, job_id, attempt_number, started_at, completed_at, status, error_message
		FROM job_attempts
		WHERE job_id = $1
		ORDER BY attempt_number ASC
	`

	rows, err := r.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []*models.JobAttempt
	for rows.Next() {
		var attempt models.JobAttempt
		var completedAt pq.NullTime

		err := rows.Scan(
			&attempt.ID, &attempt.JobID, &attempt.AttemptNumber, &attempt.StartedAt,
			&completedAt, &attempt.Status, &attempt.ErrorMessage,
		)
		if err != nil {
			return nil, err
		}

		if completedAt.Valid {
			attempt.CompletedAt = &completedAt.Time
		}

		attempts = append(attempts, &attempt)
	}

	return attempts, nil
}

func (r *PostgresRepository) GetJobStats(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*)
		FROM jobs
		GROUP BY status
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, nil
}
