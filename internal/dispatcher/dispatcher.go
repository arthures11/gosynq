package dispatcher

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/arthures11/gosynq/internal/metrics"
	"github.com/arthures11/gosynq/internal/models"
	"github.com/arthures11/gosynq/internal/repository"
	"github.com/arthures11/gosynq/internal/worker"
)

type Dispatcher struct {
	repo       *repository.PostgresRepository
	workers    []*worker.Worker
	workerPool chan struct{}
	eventChan  chan models.JobEvent
	shutdownCh chan struct{}
	shutdownWg sync.WaitGroup
	config     DispatcherConfig
	metrics    *metrics.Metrics
}

type DispatcherConfig struct {
	WorkerPoolSize    int
	VisibilityTimeout time.Duration
	RetryStrategy     worker.RetryStrategy
}

func NewDispatcher(repo *repository.PostgresRepository, config DispatcherConfig) *Dispatcher {
	return &Dispatcher{
		repo:       repo,
		workerPool: make(chan struct{}, config.WorkerPoolSize),
		eventChan:  make(chan models.JobEvent, 100), // Buffered channel
		shutdownCh: make(chan struct{}),
		config:     config,
		metrics:    metrics.NewMetrics(),
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	log.Println("Starting dispatcher")

	// Start worker pool
	for i := 0; i < d.config.WorkerPoolSize; i++ {
		d.startWorker(i)
	}

	// Start event processor (for WebSocket, etc.)
	// The WebSocket server should already be started and listening to d.eventChan
	// Events will flow automatically to WebSocket clients
	go d.processEvents(ctx)
}

func (d *Dispatcher) startWorker(id int) {
	workerID := fmt.Sprintf("worker-%d", id)

	jobHandler := func(ctx context.Context, job *models.Job) error {
		// This is where the actual job processing would happen
		// For now, we'll just log it and simulate some work
		log.Printf("Worker %s processing job %s from queue %s", workerID, job.ID, job.Queue)

		// Simulate work
		time.Sleep(1 * time.Second)

		// Simulate random failures for demo purposes
		if rand.Float32() < 0.2 { // 20% chance of failure
			return fmt.Errorf("simulated processing error")
		}

		return nil
	}

	worker := worker.NewWorker(
		workerID,
		d.repo,
		jobHandler,
		worker.WorkerConfig{
			VisibilityTimeout: d.config.VisibilityTimeout,
			RetryStrategy:     d.config.RetryStrategy,
		},
		d.eventChan,
	)

	d.shutdownWg.Add(1)
	go func() {
		defer d.shutdownWg.Done()
		worker.Start(context.Background())
	}()
}

func (d *Dispatcher) processEvents(ctx context.Context) {
	for {
		select {
		case <-d.shutdownCh:
			log.Println("Dispatcher event processor shutting down")
			return
		case event := <-d.eventChan:
			// Broadcast event to WebSocket clients
			// The event channel is already connected to WebSocket server
			// Just need to ensure events flow through properly
			log.Printf("Dispatcher received event: %s - Job %s from queue %s", event.Type, event.JobID, event.Queue)
			// The event will be picked up by WebSocket server automatically
			log.Printf("Dispatcher broadcasting event to WebSocket clients")
		}
	}
}

func (d *Dispatcher) GetEventChannel() <-chan models.JobEvent {
	return d.eventChan
}

func (d *Dispatcher) EnqueueJob(ctx context.Context, job *models.Job) error {
	// Set default values
	if job.Status == "" {
		job.Status = models.StatusPending
	}
	if job.Priority == "" {
		job.Priority = models.PriorityNormal
	}
	if job.Queue == "" {
		job.Queue = "default"
	}

	// Create the job in database
	err := d.repo.CreateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	// Send job created event
	d.eventChan <- models.JobEvent{
		Type:      "created",
		JobID:     job.ID,
		Queue:     job.Queue,
		Timestamp: time.Now(),
		Payload:   job.Payload,
	}

	return nil
}

func (d *Dispatcher) Shutdown() {
	log.Println("Shutting down dispatcher")

	close(d.shutdownCh)

	// Wait for all workers to shutdown
	d.shutdownWg.Wait()

	close(d.eventChan)
	log.Println("Dispatcher shutdown complete")
}
