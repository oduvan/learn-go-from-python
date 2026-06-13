// Package workpool runs a fixed-size pool of goroutines that each apply fn
// to inputs received over a channel. It lives under internal/ so that other
// modules cannot import it.
package workpool

import "sync"

// Run dispatches inputs across numWorkers goroutines, applies fn to each,
// and returns a map of input -> result. Each input is processed exactly once.
func Run(inputs []string, numWorkers int, fn func(string) int) map[string]int {
	type result struct {
		in  string
		out int
	}

	jobs := make(chan string, len(inputs))
	results := make(chan result, len(inputs))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for in := range jobs {
				results <- result{in: in, out: fn(in)}
			}
		}()
	}

	for _, in := range inputs {
		jobs <- in
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	out := make(map[string]int)
	for r := range results {
		out[r.in] = r.out
	}
	return out
}
