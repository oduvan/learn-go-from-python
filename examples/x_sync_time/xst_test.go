// Verifies docs/13-third-party-libraries/03-golang-x-sync-and-time.md
package xsynctime

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

func TestWaitReturnsFirstError(t *testing.T) {
	g, ctx := errgroup.WithContext(context.Background())
	for i := range 4 {
		g.Go(func() error {
			if i == 2 {
				return errors.New("job 2 failed")
			}
			select {
			case <-ctx.Done():
			case <-time.After(50 * time.Millisecond):
			}
			return nil
		})
	}
	if err := g.Wait(); err == nil || err.Error() != "job 2 failed" {
		t.Fatalf("Wait() = %v, want \"job 2 failed\"", err)
	}
}

func TestSetLimitCapsConcurrency(t *testing.T) {
	var running, peak atomic.Int64
	g := new(errgroup.Group)
	g.SetLimit(3)
	for range 100 {
		g.Go(func() error {
			n := running.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			running.Add(-1)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	if got := peak.Load(); got != 3 {
		t.Errorf("peak concurrency = %d, want 3 (the article claims exactly 3)", got)
	}
}

func TestWithContextCancelsSiblings(t *testing.T) {
	g, ctx := errgroup.WithContext(context.Background())
	var sawCancel atomic.Bool

	g.Go(func() error { return errors.New("boom") })
	g.Go(func() error {
		<-ctx.Done()
		sawCancel.Store(true)
		return nil
	})

	err := g.Wait()
	if err == nil || err.Error() != "boom" {
		t.Fatalf("Wait() = %v, want \"boom\"", err)
	}
	if !sawCancel.Load() {
		t.Error("sibling never observed cancellation")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
	}
}

func TestResultsByIndexNeedNoMutex(t *testing.T) {
	g := new(errgroup.Group)
	g.SetLimit(2)
	out := make([]int, 5)
	for i := range 5 {
		g.Go(func() error { out[i] = i * i; return nil })
	}
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	want := []int{0, 1, 4, 9, 16}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("out = %v, want %v", out, want)
		}
	}
}

func TestTryGo(t *testing.T) {
	g := new(errgroup.Group)
	g.SetLimit(1)
	release := make(chan struct{})

	if !g.TryGo(func() error { <-release; return nil }) {
		t.Fatal("first TryGo = false, want true")
	}
	if g.TryGo(func() error { return nil }) {
		t.Error("second TryGo = true, want false (limit is 1)")
	}
	close(release)
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestRateLimiterPaces(t *testing.T) {
	lim := rate.NewLimiter(rate.Limit(100), 1)
	start := time.Now()
	for range 3 {
		if err := lim.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if el := time.Since(start); el < 18*time.Millisecond {
		t.Errorf("3 events at 100/s took %v, want >= 18ms", el)
	}
}

func TestAllowIsNonBlocking(t *testing.T) {
	l := rate.NewLimiter(rate.Limit(1), 1)
	if !l.Allow() {
		t.Error("first Allow() = false, want true")
	}
	if l.Allow() {
		t.Error("second Allow() = true, want false")
	}
}

func TestPerMinuteRate(t *testing.T) {
	lim := rate.NewLimiter(rate.Limit(float64(600)/60.0), 1)
	if got := float64(lim.Limit()); got != 10 {
		t.Errorf("600rpm = %v/s, want 10", got)
	}
}
