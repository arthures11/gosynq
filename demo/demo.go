package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Test data
	testJobs := []map[string]interface{}{
		{
			"queue": "default",
			"payload": map[string]string{
				"task":    "send_email",
				"to":      "user@example.com",
				"subject": "Welcome!",
			},
			"max_retries": 3,
			"priority":    "normal",
		},
		{
			"queue": "high",
			"payload": map[string]string{
				"task":     "process_payment",
				"amount":   "100.00",
				"currency": "USD",
			},
			"max_retries": 5,
			"priority":    "high",
		},
		{
			"queue": "low",
			"payload": map[string]string{
				"task":        "generate_report",
				"report_type": "daily",
			},
			"max_retries": 1,
			"priority":    "low",
		},
	}

	// Enqueue jobs
	for i, jobData := range testJobs {
		fmt.Printf("Enqueuing job %d...\n", i+1)

		jsonData, err := json.Marshal(jobData)
		if err != nil {
			log.Printf("Error marshaling job %d: %v", i+1, err)
			continue
		}

		resp, err := http.Post("http://localhost:8080/api/v1/jobs", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Error enqueuing job %d: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Job %d response: %s\n", i+1, string(body))
	}

	// List jobs
	fmt.Println("\nListing all jobs:")
	resp, err := http.Get("http://localhost:8080/api/v1/jobs")
	if err != nil {
		log.Printf("Error listing jobs: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	// Wait a bit for jobs to be processed
	fmt.Println("\nWaiting for jobs to be processed...")
	time.Sleep(10 * time.Second)

	// List completed jobs
	fmt.Println("\nListing completed jobs:")
	resp, err = http.Get("http://localhost:8080/api/v1/jobs?status=completed")
	if err != nil {
		log.Printf("Error listing completed jobs: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	fmt.Println(string(body))
}
