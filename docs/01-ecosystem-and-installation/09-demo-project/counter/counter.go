// Package counter counts whitespace-separated words in text files.
// It is a regular (non-internal) package, so other modules could import it
// if this module were published.
package counter

import (
	"crypto/sha256"
	"os"
	"strings"

	"example.com/demo/internal/workpool"
)

// CountFile reads path and returns the number of whitespace-separated words.
// It also computes a SHA-256 hash of the contents — that work is throwaway,
// but it gives the CPU something visible to do in `go tool trace`.
func CountFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	_ = sha256.Sum256(data)
	return len(strings.Fields(string(data))), nil
}

// CountConcurrent counts words in every path using numWorkers goroutines.
// Returns a map of path -> word count. Read errors collapse to 0 to keep
// the demo focused on concurrency, not error handling.
func CountConcurrent(paths []string, numWorkers int) map[string]int {
	return workpool.Run(paths, numWorkers, func(p string) int {
		n, err := CountFile(p)
		if err != nil {
			return 0
		}
		return n
	})
}
