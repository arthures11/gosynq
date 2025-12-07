package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arthures11/gosynq/internal/config"
	"github.com/arthures11/gosynq/internal/dispatcher"
	"github.com/arthures11/gosynq/internal/models"
	"github.com/arthures11/gosynq/internal/repository"
	"github.com/arthures11/gosynq/internal/websocket"
	"github.com/arthures11/gosynq/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	// Load configuration
	cfg := config.NewDefaultConfig()

	// Set up database connection
	db, err := setupDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer db.Close()

	// Create repository
	repo := repository.NewPostgresRepository(db)

	// Create dispatcher
	disp := dispatcher.NewDispatcher(repo, dispatcher.DispatcherConfig{
		WorkerPoolSize:    cfg.Worker.PoolSize,
		VisibilityTimeout: cfg.Worker.VisibilityTimeout,
		RetryStrategy: worker.RetryStrategy{
			Type:        cfg.Retries.DefaultStrategy,
			Interval:    cfg.Retries.DefaultInterval,
			MaxAttempts: cfg.Retries.MaxAttempts,
		},
	})

	// Create WebSocket server
	wsServer := websocket.NewWebSocketServer(disp.GetEventChannel())
	wsServer.Start(context.Background())

	// Start dispatcher
	ctx, cancel := context.WithCancel(context.Background())
	disp.Start(ctx)

	// Set up HTTP server
	router := setupRouter(disp, repo, wsServer)

	// Set up graceful shutdown
	go func() {
		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		// Shutdown dispatcher
		disp.Shutdown()

		// Shutdown WebSocket server
		wsServer.Shutdown()

		// TODO: Save final metrics before shutdown
		log.Println("Saving final metrics before shutdown...")

		cancel()

		// Shutdown HTTP server
		srv := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: router,
		}
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}

		os.Exit(0)
	}()

	// Start HTTP server
	log.Printf("Server starting on :%d", cfg.Server.Port)
	if err := router.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func setupDatabase(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func setupRouter(disp *dispatcher.Dispatcher, repo *repository.PostgresRepository, wsServer *websocket.WebSocketServer) *gin.Engine {
	router := gin.Default()

	// API routes
	api := router.Group("/api/v1")
	{
		jobs := api.Group("/jobs")
		{
			jobs.POST("", func(c *gin.Context) {
				// Enqueue job endpoint
				var req struct {
					Queue      string          `json:"queue"`
					Payload    json.RawMessage `json:"payload"`
					MaxRetries int             `json:"max_retries"`
					Priority   string          `json:"priority"`
				}

				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				job := &models.Job{
					ID:         uuid.New().String(),
					Queue:      req.Queue,
					Payload:    req.Payload,
					MaxRetries: req.MaxRetries,
					Priority:   models.JobPriority(req.Priority),
					RunAt:      time.Now(),
					Status:     models.StatusPending,
				}

				if err := disp.EnqueueJob(c.Request.Context(), job); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusCreated, gin.H{
					"job_id": job.ID,
					"status": "queued",
				})
			})

			jobs.GET("", func(c *gin.Context) {
				// List jobs endpoint
				status := c.Query("status")
				queue := c.Query("queue")
				limit := 100

				jobs, err := repo.ListJobs(c.Request.Context(), status, queue, limit)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, jobs)
			})

			jobs.GET("/:id", func(c *gin.Context) {
				// Get job details
				jobID := c.Param("id")

				job, err := repo.GetJobByID(c.Request.Context(), jobID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				if job == nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
					return
				}

				c.JSON(http.StatusOK, job)
			})

			// Admin endpoints
			admin := api.Group("/admin", gin.BasicAuth(gin.Accounts{
				"admin": "password", // TODO: Make configurable
			}))
			{
				admin.POST("/jobs/:id/retry", func(c *gin.Context) {
					jobID := c.Param("id")

					// Get current job
					job, err := repo.GetJobByID(c.Request.Context(), jobID)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}
					if job == nil {
						c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
						return
					}

					// Reset job to pending for retry
					err = repo.UpdateJobStatus(c.Request.Context(), jobID, models.StatusPending)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}

					c.JSON(http.StatusOK, gin.H{"status": "retry scheduled"})
				})

				admin.POST("/jobs/:id/cancel", func(c *gin.Context) {
					jobID := c.Param("id")

					// Cancel the job
					err := repo.UpdateJobStatus(c.Request.Context(), jobID, models.StatusCancelled)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}

					c.JSON(http.StatusOK, gin.H{"status": "job cancelled"})
				})

				admin.POST("/queues/:queue/pause", func(c *gin.Context) {
					// TODO: Implement queue pausing logic
					c.JSON(http.StatusOK, gin.H{"status": "queue pause not implemented"})
				})

				admin.POST("/queues/:queue/resume", func(c *gin.Context) {
					// TODO: Implement queue resuming logic
					c.JSON(http.StatusOK, gin.H{"status": "queue resume not implemented"})
				})
			}
		}

		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		// WebSocket endpoint
		api.GET("/ws", func(c *gin.Context) {
			wsServer.HandleWebSocket(c.Writer, c.Request)
		})

		// Metrics endpoint
		api.GET("/metrics", func(c *gin.Context) {
			// TODO: Implement Prometheus metrics endpoint
			c.JSON(http.StatusOK, gin.H{"status": "metrics not implemented"})
		})

		// Serve frontend
		router.Static("/frontend", "./frontend")
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/frontend/index.html")
		})
	}

	return router
}
