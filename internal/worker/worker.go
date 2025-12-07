package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/arthures11/gosynq/internal/models"
	"github.com/arthures11/gosynq/internal/repository"
)

type Worker struct {
	id         string
	repo       *repository.PostgresRepository
	jobHandler JobHandler
	config     WorkerConfig
	eventChan  chan<- models.JobEvent
	shutdownCh chan struct{}
}

type WorkerConfig struct {
	VisibilityTimeout time.Duration
	RetryStrategy     RetryStrategy
}

type RetryStrategy struct {
	Type        string
	Interval    int
	MaxAttempts int
}

type JobHandler func(ctx context.Context, job *models.Job) error

func NewWorker(id string, repo *repository.PostgresRepository, handler JobHandler, config WorkerConfig, eventChan chan<- models.JobEvent) *Worker {
	return &Worker{
		id:         id,
		repo:       repo,
		jobHandler: handler,
		config:     config,
		eventChan:  eventChan,
		shutdownCh: make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	log.Printf("Worker %s starting", w.id)

	for {
		select {
		case <-w.shutdownCh:
			log.Printf("Worker %s shutting down", w.id)
			return
		default:
			job, err := w.pickAndProcessJob(ctx)
			if err != nil {
				log.Printf("Worker %s error: %v", w.id, err)
				// Add some jitter to avoid thundering herd
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				continue
			}

			if job == nil {
				// No jobs available, wait a bit
				time.Sleep(1 * time.Second)
				continue
			}
		}
	}
}

func (w *Worker) pickAndProcessJob(ctx context.Context) (*models.Job, error) {
	// Atomic job pickup
	job, err := w.repo.PickJob(ctx, w.id, w.config.VisibilityTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to pick job: %w", err)
	}

	if job == nil {
		return nil, nil // No jobs available
	}

	// Send job started event
	w.eventChan <- models.JobEvent{
		Type:      "started",
		JobID:     job.ID,
		Queue:     job.Queue,
		Timestamp: time.Now(),
		Payload:   job.Payload,
	}

	// Process the job
	err = w.processJob(ctx, job)
	if err != nil {
		return job, fmt.Errorf("job processing failed: %w", err)
	}

	return job, nil
}

func (w *Worker) processJob(ctx context.Context, job *models.Job) error {
	ctx, cancel := context.WithTimeout(ctx, w.config.VisibilityTimeout)
	defer cancel()

	// Create job attempt record
	attempt := &models.JobAttempt{
		ID:            fmt.Sprintf("%s-%d", job.ID, time.Now().UnixNano()),
		JobID:         job.ID,
		AttemptNumber: 1, // TODO: Get actual attempt number
		StartedAt:     time.Now(),
		Status:        models.StatusProcessing,
	}

	err := w.repo.CreateJobAttempt(ctx, attempt)
	if err != nil {
		return fmt.Errorf("failed to create job attempt: %w", err)
	}

	// Execute the job handler
	err = w.jobHandler(ctx, job)
	if err != nil {
		attempt.Status = models.StatusFailed
		attempt.ErrorMessage = err.Error()
		attempt.CompletedAt = &time.Time{}

		// Update job attempt
		updateErr := w.repo.CreateJobAttempt(ctx, attempt)
		if updateErr != nil {
			return fmt.Errorf("failed to update job attempt: %w", updateErr)
		}

		// Handle retry logic
		return w.handleJobFailure(ctx, job, err)
	}

	// Job succeeded
	attempt.Status = models.StatusCompleted
	attempt.CompletedAt = &time.Time{}
	err = w.repo.CreateJobAttempt(ctx, attempt)
	if err != nil {
		return fmt.Errorf("failed to update job attempt: %w", err)
	}

	err = w.repo.UpdateJobStatus(ctx, job.ID, models.StatusCompleted)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Send job succeeded event
	w.eventChan <- models.JobEvent{
		Type:      "succeeded",
		JobID:     job.ID,
		Queue:     job.Queue,
		Timestamp: time.Now(),
		Payload:   job.Payload,
	}

	return nil
}

func (w *Worker) handleJobFailure(ctx context.Context, job *models.Job, err error) error {
	// Update job status to failed
	updateErr := w.repo.UpdateJobStatus(ctx, job.ID, models.StatusFailed)
	if updateErr != nil {
		return fmt.Errorf("failed to update job status: %w", updateErr)
	}

	// Send job failed event
	w.eventChan <- models.JobEvent{
		Type:      "failed",
		JobID:     job.ID,
		Queue:     job.Queue,
		Timestamp: time.Now(),
		Payload:   job.Payload,
		Error:     err.Error(),
	}

	// Check if we should retry
	if job.IsRetryable() {
		// Get current attempts
		attempts, err := w.repo.GetJobAttempts(ctx, job.ID)
		if err != nil {
			return fmt.Errorf("failed to get job attempts: %w", err)
		}

		currentAttempts := len(attempts)
		if job.ShouldRetry(currentAttempts) {
			// Calculate retry delay
			delay := w.calculateRetryDelay(currentAttempts)

			// Reset job to pending with new run_at time
			job.Status = models.StatusPending
			job.RunAt = time.Now().Add(delay)

			// TODO: Implement job update with new run_at
			log.Printf("Job %s will be retried in %v", job.ID, delay)
			return nil
		}
	}

	return nil
}

func (w *Worker) calculateRetryDelay(attempt int) time.Duration {
	switch w.config.RetryStrategy.Type {
	case "exponential":
		exponent := math.Min(float64(attempt), 10) // Cap exponent at 10
		return time.Duration(math.Pow(2, exponent)) * time.Second * time.Duration(w.config.RetryStrategy.Interval)
	case "fixed":
		return time.Duration(w.config.RetryStrategy.Interval) * time.Second
	default:
		return time.Duration(w.config.RetryStrategy.Interval) * time.Second
	}
}

func (w *Worker) Shutdown() {
	close(w.shutdownCh)
}
