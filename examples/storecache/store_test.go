// Verifies docs/13-third-party-libraries/16-object-storage-and-caching.md
package storecache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/oduvan/learn-go-from-python/examples/infra"
)

type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

var ErrObjectNotFound = errors.New("object not found")

type localStore struct{ root string }

func (s localStore) path(key string) string { return filepath.Join(s.root, filepath.FromSlash(key)) }

func (s localStore) Put(_ context.Context, key string, r io.Reader) error {
	p := s.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (s localStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(s.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
	}
	return f, err
}

func (s localStore) Delete(_ context.Context, key string) error {
	err := os.Remove(s.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil // deleting what is absent is not an error
	}
	return err
}

var _ ObjectStore = localStore{}

func TestLocalStoreRoundTrip(t *testing.T) {
	var st ObjectStore = localStore{root: t.TempDir()}
	ctx := context.Background()

	if err := st.Put(ctx, "a/b/file.txt", strings.NewReader("payload")); err != nil {
		t.Fatal(err)
	}
	rc, err := st.Get(ctx, "a/b/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "payload" {
		t.Errorf("got %q", string(b))
	}
}

// Two behaviours every implementation must share.
func TestLocalStoreErrorContract(t *testing.T) {
	var st ObjectStore = localStore{root: t.TempDir()}
	ctx := context.Background()

	_, err := st.Get(ctx, "missing")
	if !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("Get(missing) = %v, want ErrObjectNotFound", err)
	}
	if err := st.Delete(ctx, "missing"); err != nil {
		t.Errorf("Delete(missing) = %v, want nil", err)
	}
}

func client(t *testing.T) valkey.Client {
	t.Helper()
	addr := infra.RequireValkey(t)

	c, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{addr}})
	if err != nil {
		t.Fatalf("connecting to valkey: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestSetGetWithTTL(t *testing.T) {
	c := client(t)
	ctx := context.Background()

	if err := c.Do(ctx, c.B().Set().Key("k").Value("v").Ex(30*time.Second).Build()).Error(); err != nil {
		t.Fatal(err)
	}
	got, err := c.Do(ctx, c.B().Get().Key("k").Build()).ToString()
	if err != nil || got != "v" {
		t.Fatalf("got %q err %v", got, err)
	}
	ttl, err := c.Do(ctx, c.B().Ttl().Key("k").Build()).AsInt64()
	if err != nil || ttl <= 0 {
		t.Errorf("ttl = %d err %v, want a positive ttl", ttl, err)
	}
}

// A miss arrives as an error value, and it is not a failure.
func TestMissIsValkeyNil(t *testing.T) {
	c := client(t)
	_, err := c.Do(context.Background(), c.B().Get().Key("definitely-absent").Build()).ToString()
	if !valkey.IsValkeyNil(err) {
		t.Fatalf("err = %v, want a valkey nil", err)
	}
}

func TestPubSubDeliversToASubscriber(t *testing.T) {
	c := client(t)
	ctx := context.Background()

	sub, cancel := c.Dedicate()
	defer cancel()

	msgs := make(chan string, 1)
	go func() {
		_ = sub.Receive(ctx, sub.B().Subscribe().Channel("examples-events").Build(),
			func(m valkey.PubSubMessage) { msgs <- m.Message })
	}()
	time.Sleep(200 * time.Millisecond) // let the subscription land

	if err := c.Do(ctx, c.B().Publish().Channel("examples-events").Message("hello").Build()).Error(); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-msgs:
		if m != "hello" {
			t.Errorf("received %q", m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no message received")
	}
}
