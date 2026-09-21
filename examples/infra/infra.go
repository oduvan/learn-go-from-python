// Package infra holds helpers shared by the example tests: connection
// strings for the services docker-compose.yml starts, and skips for when
// those services are not running.
package infra

import (
	"context"
	"database/sql"
	"net"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Defaults match docker-compose.yml. Override with the environment when
// pointing at something else (a CI service container, for instance).
const (
	defaultPostgresDSN = "postgres://postgres:pw@localhost:55433/book?sslmode=disable"
	defaultValkeyAddr  = "127.0.0.1:56379"
)

func PostgresDSN() string {
	if v := os.Getenv("EXAMPLES_POSTGRES_DSN"); v != "" {
		return v
	}
	return defaultPostgresDSN
}

func ValkeyAddr() string {
	if v := os.Getenv("EXAMPLES_VALKEY_ADDR"); v != "" {
		return v
	}
	return defaultValkeyAddr
}

// RequirePostgres returns an open pool, or skips the test when the server
// is not reachable. Set EXAMPLES_REQUIRE_INFRA=1 to make absence a failure
// instead — which is what CI should do, so a missing service is loud.
func RequirePostgres(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", PostgresDSN())
	if err != nil {
		unavailable(t, "opening postgres: %v", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		unavailable(t, "postgres unreachable at %s: %v", PostgresDSN(), err)
		return nil
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// RequireValkey skips unless something is listening on the Valkey port.
func RequireValkey(t *testing.T) string {
	t.Helper()

	conn, err := net.DialTimeout("tcp", ValkeyAddr(), 2*time.Second)
	if err != nil {
		unavailable(t, "valkey unreachable at %s: %v", ValkeyAddr(), err)
		return ""
	}
	conn.Close()
	return ValkeyAddr()
}

// RequireDocker skips unless a Docker daemon is answering.
func RequireDocker(t *testing.T) {
	t.Helper()

	if os.Getenv("EXAMPLES_SKIP_DOCKER") != "" {
		t.Skip("EXAMPLES_SKIP_DOCKER set")
	}
	for _, sock := range dockerSockets() {
		if c, err := net.DialTimeout("unix", sock, 2*time.Second); err == nil {
			c.Close()
			return
		}
	}
	unavailable(t, "no docker daemon reachable")
}

func dockerSockets() []string {
	var out []string
	if h := os.Getenv("DOCKER_HOST"); len(h) > 7 && h[:7] == "unix://" {
		out = append(out, h[7:])
	}
	home, _ := os.UserHomeDir()
	return append(out,
		"/var/run/docker.sock",
		home+"/.colima/default/docker.sock",
		home+"/.docker/run/docker.sock",
	)
}

func unavailable(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("EXAMPLES_REQUIRE_INFRA") != "" {
		t.Fatalf("EXAMPLES_REQUIRE_INFRA is set: "+format, args...)
	}
	t.Skipf(format, args...)
}
