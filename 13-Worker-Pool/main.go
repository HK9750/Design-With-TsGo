package main

import (
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// RunWorkerPool distributes a slice of jobs across a fixed number of workers.
// Each job is processed by the handler function. Results are returned in the
// same order as the input jobs. Panics if workers <= 0. Uses generics so I
// (input) and O (output) can be any type. Time complexity: O(n/workers) per worker.
func RunWorkerPool[I any, O any](workers int, jobs []I, handler func(I) O) []O {
	if workers <= 0 {
		log.Error("workers must be positive, got %d", workers)
		os.Exit(1)
	}

	log.Debug("Starting worker pool: workers=%d jobs=%d", workers, len(jobs))
	results := make([]O, len(jobs))

	type job struct {
		index int
		value I
	}

	queue := make(chan job)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for item := range queue {
				log.Step("Worker %d processing job %d", workerID, item.index)
				results[item.index] = handler(item.value)
			}
			log.Debug("Worker %d finished", workerID)
		}(i)
	}

	for index, value := range jobs {
		queue <- job{index: index, value: value}
	}
	close(queue)
	wg.Wait()

	log.Info("All %d jobs processed by %d workers", len(jobs), workers)
	return results
}

func main() {
	defer log.Operation("main", "Running Worker Pool demo")()

	logger.Section("Scenario: Bulk Image Thumbnail Generation Service")

	images := []int{1200, 800, 1600, 2400, 3200, 640, 1920, 1080, 2560, 1440}
	workers := 3

	log.Info("Processing %d images with %d workers...", len(images), workers)

	start := time.Now()
	result := RunWorkerPool(workers, images, func(n int) int {
		// Simulate CPU-intensive thumbnail generation
		time.Sleep(20 * time.Millisecond)
		return n * n // simulated processing
	})
	elapsed := time.Since(start)

	logger.Section("Processing Results")
	for i, r := range result {
		logger.KeyValue("image", i)
		logger.KeyValue("result", r)
	}

	logger.Section("Final Stats Summary")
	logger.KeyValue("totalJobs", len(images))
	logger.KeyValue("numWorkers", workers)
	logger.KeyValue("totalTime", elapsed)
	log.Info("Thumbnail generation completed in %v", elapsed)
}
