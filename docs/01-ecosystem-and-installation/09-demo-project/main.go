// Demo: counts words across many files using a worker pool, while recording
// a runtime trace so we can view it with `go tool trace`.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/trace"

	"example.com/demo/counter"
)

const numWorkers = 4

func main() {
	// Start a runtime trace. Everything done between trace.Start and
	// trace.Stop will appear in trace.out.
	f, err := os.Create("trace.out")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := trace.Start(f); err != nil {
		log.Fatal(err)
	}
	defer trace.Stop()

	// Gather the test files in counter/testdata as our workload.
	base := filepath.Join("counter", "testdata")
	entries, err := os.ReadDir(base)
	if err != nil {
		log.Fatal(err)
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			paths = append(paths, filepath.Join(base, e.Name()))
		}
	}

	// Amplify the workload so the trace has interesting parallelism.
	const repeat = 200
	big := make([]string, 0, len(paths)*repeat)
	for i := 0; i < repeat; i++ {
		big = append(big, paths...)
	}

	results := counter.CountConcurrent(big, numWorkers)

	var total int
	for _, n := range results {
		total += n
	}
	fmt.Printf("Processed %d jobs across %d unique files; total words: %d\n",
		len(big), len(results), total)
}
