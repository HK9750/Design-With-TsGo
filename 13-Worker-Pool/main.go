package main

import (
	"fmt"
	"sync"
)

func RunWorkerPool[I any, O any](workers int, jobs []I, handler func(I) O) []O {
	if workers <= 0 {
		panic("workers must be positive")
	}
	results := make([]O, len(jobs))
	type job struct {
		index int
		value I
	}
	queue := make(chan job)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range queue {
				results[item.index] = handler(item.value)
			}
		}()
	}
	for index, value := range jobs {
		queue <- job{index: index, value: value}
	}
	close(queue)
	wg.Wait()
	return results
}

func main() {
	result := RunWorkerPool(3, []int{1, 2, 3, 4}, func(n int) int { return n * n })
	fmt.Println(result)
}
