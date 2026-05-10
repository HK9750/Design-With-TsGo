package main

import (
	"fmt"
	"sync"
)

func FanOutFanIn[I any, O any](items []I, workers int, handler func(I) O) []O {
	results := make([]O, len(items))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				results[index] = handler(items[index])
			}
		}()
	}
	for index := range items {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	return results
}

func main() {
	fmt.Println(FanOutFanIn([]int{1, 2, 3}, 2, func(n int) int { return n * 10 }))
}
