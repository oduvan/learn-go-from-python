// Verifies docs/13-third-party-libraries/15-scheduled-jobs.md
package scheddemo

import (
	"context"
	"database/sql"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/oduvan/learn-go-from-python/examples/infra"
)

func JobLockKey(name string) int64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	return int64(h.Sum64())
}

func LockedRun(ctx context.Context, db *sql.DB, job string, fn func(context.Context) error) (bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	key := JobLockKey(job)
	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}
	defer func() {
		unlockCtx := context.WithoutCancel(ctx)
		_, _ = conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", key)
	}()
	return true, fn(ctx)
}

func TestLockKeyIsDeterministic(t *testing.T) {
	if JobLockKey("nightly-reindex") != JobLockKey("nightly-reindex") {
		t.Error("the same name produced two keys")
	}
	if JobLockKey("a") == JobLockKey("b") {
		t.Error("different names collided")
	}
}

// The headline claim: with N replicas racing, exactly one runs the job.
func TestOnlyOneReplicaRunsTheJob(t *testing.T) {
	db := infra.RequirePostgres(t)
	db.SetMaxOpenConns(10)
	ctx := context.Background()

	var ran, skipped atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := LockedRun(ctx, db, "test-nightly-reindex", func(context.Context) error {
				ran.Add(1)
				time.Sleep(120 * time.Millisecond)
				return nil
			})
			if err != nil {
				t.Errorf("LockedRun: %v", err)
				return
			}
			if !ok {
				skipped.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if ran.Load() != 1 || skipped.Load() != 4 {
		t.Errorf("ran=%d skipped=%d, want 1 and 4", ran.Load(), skipped.Load())
	}
}

func TestLockIsReleasedAfterwards(t *testing.T) {
	db := infra.RequirePostgres(t)
	ctx := context.Background()

	for i := range 2 {
		ok, err := LockedRun(ctx, db, "test-release", func(context.Context) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("run %d could not acquire the lock — it was not released", i)
		}
	}
}

// Even when the caller's context is cancelled, the unlock must still run.
func TestUnlockSurvivesCancellation(t *testing.T) {
	db := infra.RequirePostgres(t)
	ctx, cancel := context.WithCancel(context.Background())

	_, _ = LockedRun(ctx, db, "test-cancelled", func(context.Context) error {
		cancel() // the job's context dies mid-run
		return nil
	})

	ok, err := LockedRun(context.Background(), db, "test-cancelled", func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("the lock was still held; context.WithoutCancel on the unlock is what prevents this")
	}
}

func TestGocronRunsOnAnInterval(t *testing.T) {
	s, err := gocron.NewScheduler()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Shutdown() })

	var ticks atomic.Int32
	j, err := s.NewJob(
		gocron.DurationJob(40*time.Millisecond),
		gocron.NewTask(func(context.Context) { ticks.Add(1) }, context.Background()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		t.Fatal(err)
	}
	if j.ID().String() == "" {
		t.Error("job has no id")
	}

	s.Start()
	time.Sleep(200 * time.Millisecond)
	if ticks.Load() < 2 {
		t.Errorf("ticks = %d, want at least 2", ticks.Load())
	}
}

func TestCronExpressionIsValidatedUpFront(t *testing.T) {
	s, _ := gocron.NewScheduler()
	t.Cleanup(func() { _ = s.Shutdown() })

	if _, err := s.NewJob(gocron.CronJob("0 3 * * *", false), gocron.NewTask(func() {})); err != nil {
		t.Errorf("a valid expression was rejected: %v", err)
	}
	if _, err := s.NewJob(gocron.CronJob("not a cron", false), gocron.NewTask(func() {})); err == nil {
		t.Error("an invalid expression was accepted; the article says it fails at NewJob")
	}
}
