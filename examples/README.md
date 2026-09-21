# Runnable examples

Every third-party example in
[`docs/13-third-party-libraries/`](../docs/13-third-party-libraries/) has a
test here that asserts what the article claims. If an article says a call
returns `true`, a test here checks that it returns `true`.

The point is that the book cannot quietly go stale. When a library changes
behaviour, a test fails and names the article that needs updating.

## Running it

```bash
cd examples
make verify
```

That starts PostgreSQL and Valkey, runs everything, and tears them down.
To keep the services up between runs:

```bash
make up
make test
```

## What needs what

| Package | Article | Needs |
|---|---|---|
| `x_sync_time` | `03-golang-x-sync-and-time` | — |
| `yaml_toml` | `04-yaml-and-toml` | — |
| `viperconf` | `05-viper` | — |
| `pgxdemo` | `06-pgx-and-postgres` | PostgreSQL |
| `gorm_basics` | `07-gorm-basics` | PostgreSQL |
| `gorm_queries` | `08-gorm-queries-and-transactions` | PostgreSQL |
| `goosedemo` | `09-goose-migrations` | PostgreSQL |
| `fiberdemo` | `10-fiber` | — |
| `templdemo` | `11-templ` | `templ` CLI to regenerate |
| `testifydemo` | `12-testify` | — |
| `tcdemo` | `13-testcontainers` | Docker |
| `obsdemo` | `14-prometheus-and-opentelemetry` | — |
| `scheddemo` | `15-scheduled-jobs` | PostgreSQL |
| `storecache` | `16-object-storage-and-caching` | Valkey |
| `authdemo` | `17-oidc-and-oauth` | — |
| `mcpdemo` | `18-mcp-servers-with-mcp-go` | — |
| `lintdemo` | `02-golangci-lint-configuration` | `golangci-lint` |

`01-choosing-and-managing-dependencies` has no runnable code — it is about
`go` subcommands.

## Skipping vs failing

A test whose service is not running **skips**, so `go test ./...` is useful
without Docker. Set `EXAMPLES_REQUIRE_INFRA=1` and a missing service becomes
a failure instead — which is what CI should do, so a misconfigured pipeline
is loud rather than quietly green. That is `make ci`.

Point the tests elsewhere with `EXAMPLES_POSTGRES_DSN` and
`EXAMPLES_VALKEY_ADDR`.

## Notes

- **One module, many packages.** Simpler than a module per example, and one
  `go get` covers everything.
- **`lintdemo` is a separate module** because it contains a deliberate
  layering violation, which is the thing being tested.
- **`templdemo/page_templ.go` is committed**, so the tests build without the
  `templ` CLI. `make generate` refreshes it when you have the CLI.
- **Tests assert the article, not just the library.** Several say so in a
  comment: if a library gains a behaviour the article says it lacks, the
  test fails and tells you to update the article. `viperconf` and
  `yaml_toml` both do this.
