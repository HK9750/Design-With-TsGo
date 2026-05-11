package main

import (
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// FanOutFanIn distributes items across workers for parallel processing.
// Each worker receives an index and applies the handler to items[index].
// Results preserve input order. Uses generics: I is input type, O is output type.
// Time complexity: O(n/workers) per worker, O(n) total work.
func FanOutFanIn[I any, O any](items []I, workers int, handler func(I) O) []O {
	log.Debug("Fan-out fan-in: items=%d workers=%d", len(items), workers)
	results := make([]O, len(items))
	jobs := make(chan int)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for index := range jobs {
				log.Step("Worker %d processing item %d", workerID, index)
				results[index] = handler(items[index])
			}
			log.Debug("Worker %d finished", workerID)
		}(i)
	}

	for index := range items {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	log.Info("Fan-out fan-in complete: %d items processed by %d workers", len(items), workers)
	return results
}

func main() {
	defer log.Operation("main", "Running Fan-Out Fan-In demo")()

	logger.Section("Scenario: Parallel Image Resize Pipeline")
	log.Info("Simulating resizing 8 images across 4 workers")

	// Image dimensions (width x height product) to simulate work
	imageSizes := []int{1920, 1080, 2560, 1440, 3840, 2160, 1280, 720}
	workers := 4

	start := time.Now()
	results := FanOutFanIn(imageSizes, workers, func(size int) int {
		// Simulate resize computation
		time.Sleep(15 * time.Millisecond)
		return size / 2 // simulated resized dimension
	})
	elapsed := time.Since(start)

	logger.Section("Resize Results")
	for i, r := range results {
		logger.KeyValue("image", i)
		logger.KeyValue("originalSize", imageSizes[i])
		logger.KeyValue("resizedWidth", r)
	}

	logger.Section("Final Stats Summary")
	logger.KeyValue("totalImages", len(imageSizes))
	logger.KeyValue("numWorkers", workers)
	logger.KeyValue("totalTime", elapsed)
	log.Info("Image resize pipeline completed in %v", elapsed)
}
