package counter

import (
	"path/filepath"
	"testing"
)

func TestCountFile(t *testing.T) {
	n, err := CountFile(filepath.Join("testdata", "lorem.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Error("expected lorem.txt to contain words, got 0")
	}
}

func TestCountConcurrent(t *testing.T) {
	paths := []string{
		filepath.Join("testdata", "lorem.txt"),
		filepath.Join("testdata", "alice.txt"),
	}
	got := CountConcurrent(paths, 2)
	if len(got) != 2 {
		t.Errorf("expected 2 results, got %d", len(got))
	}
	for p, n := range got {
		if n == 0 {
			t.Errorf("expected words in %s, got 0", p)
		}
	}
}

func BenchmarkCountConcurrent(b *testing.B) {
	paths := []string{
		filepath.Join("testdata", "lorem.txt"),
		filepath.Join("testdata", "alice.txt"),
	}
	for i := 0; i < b.N; i++ {
		CountConcurrent(paths, 4)
	}
}
