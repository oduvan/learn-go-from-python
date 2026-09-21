# Object storage and caching

Two infrastructure dependencies with one idea in common: put an
interface in front of them, so local development needs neither and
tests need neither.

> **Modules:** `github.com/aws/aws-sdk-go-v2` (with `config` and
> `service/s3`) and `github.com/valkey-io/valkey-go`.

```go
type ObjectStore interface {
    Put(ctx context.Context, key string, r io.Reader) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

## Define the interface, not the client

Handlers should not know whether a file is in S3 or on disk. The
interface above is the whole contract, and it is deliberately tiny —
the reasoning from
[the repository pattern](../09-database-sql/04-the-repository-pattern.md).

Note it is expressed in `io.Reader` and `io.ReadCloser`, so a caller
can stream a 2 GB upload without it passing through memory.

Give it a domain error:

```go
var ErrObjectNotFound = errors.New("object not found")
```

## A filesystem implementation

The local implementation is short, and it is what makes development
and tests work with no cloud credentials at all:

```go
type localStore struct{ root string }

func (s localStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
    f, err := os.Open(s.path(key))
    if errors.Is(err, os.ErrNotExist) {
        return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
    }
    return f, err
}
```

```go
st.Put(ctx, "a/b/file.txt", strings.NewReader("payload"))
rc, err := st.Get(ctx, "a/b/file.txt")   // "payload"

_, err = st.Get(ctx, "missing")
errors.Is(err, ErrObjectNotFound)        // true
st.Delete(ctx, "missing")                // nil — deleting what is absent is fine
```

Two behaviours to match across implementations: a missing object
returns your sentinel, and deleting something absent is **not** an
error. Getting those inconsistent is how an implementation swap breaks
callers.

Use `filepath.FromSlash` on the key and reject `..`, or a key from user
input escapes your root directory.

## The S3 implementation

```go
cfg, err := config.LoadDefaultConfig(ctx)
client := s3.NewFromConfig(cfg)
```

`LoadDefaultConfig` walks the standard credential chain — environment,
shared config file, instance role — so production needs no code
change. For a MinIO or Garage endpoint locally, override the base URL
in the options.

Translate the SDK's typed errors to your own:

```go
out, err := c.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &key})
if err != nil {
    var nsk *types.NoSuchKey
    var nf *types.NotFound
    if errors.As(err, &nsk) || errors.As(err, &nf) {
        return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
    }
    return nil, fmt.Errorf("getting %s: %w", key, err)
}
return out.Body, nil
```

Both types matter — which one you get depends on the operation, and
checking only `NoSuchKey` misses `HeadObject`.

Two more things worth knowing. `manager.NewUploader` handles multipart
uploads for large objects, which the plain `PutObject` does not. And a
**presigned URL** lets a client upload or download directly, so a big
file never passes through your service at all:

```go
ps := s3.NewPresignClient(client)
req, err := ps.PresignGetObject(ctx, in, s3.WithPresignExpires(15*time.Minute))
```

## Choosing the implementation at startup

```go
func NewObjectStore(cfg Config) ObjectStore {
    if cfg.S3Bucket == "" {
        return localStore{root: cfg.LocalStoragePath}
    }
    return s3Store{client: s3.NewFromConfig(awsCfg), bucket: cfg.S3Bucket}
}
```

One decision, in one place. Everything downstream holds the interface.

## Caching with valkey

Valkey is the Redis fork; `valkey-go` uses a command builder rather
than stringly-typed arguments:

```go
c, err := valkey.NewClient(valkey.ClientOption{
    InitAddress: []string{"127.0.0.1:6379"},
})
defer c.Close()

err = c.Do(ctx, c.B().Set().Key("k").Value("v").Ex(30*time.Second).Build()).Error()

got, err := c.Do(ctx, c.B().Get().Key("k").Build()).ToString()
// "v"
```

`c.B()` builds a command with the arguments checked at compile time, so
a typo in `SET` is a build error rather than a runtime one. The client
is safe for concurrent use and pipelines automatically.

### A miss is an error value

```go
_, err := c.Do(ctx, c.B().Get().Key("absent").Build()).ToString()
fmt.Println(valkey.IsValkeyNil(err))   // output: true
// valkey nil message
```

A cache miss comes back as an error, and it is not a failure. Check it
explicitly:

```go
val, err := c.Do(ctx, c.B().Get().Key(k).Build()).ToString()
switch {
case valkey.IsValkeyNil(err):
    return compute()          // miss
case err != nil:
    return compute()          // cache broken — still serve the request
default:
    return val, nil
}
```

**Always set a TTL.** `Ex(30*time.Second)` on the write; a cache
without expiry is a memory leak with extra steps.

### Pub/sub

```go
sub, cancel := c.Dedicate()
defer cancel()

go sub.Receive(ctx, sub.B().Subscribe().Channel("events").Build(),
    func(m valkey.PubSubMessage) {
        // handle m.Message
    })

c.Do(ctx, c.B().Publish().Channel("events").Message("hello").Build())
// received: hello
```

Subscribing needs a **dedicated connection** — `Dedicate()` — because
a subscribed connection cannot serve other commands.

This is how [server-sent events](../08-http-with-net-http/05-server-sent-events.md)
work across replicas: a client is connected to one instance, so an
event raised on another only reaches them if the instances share a
bus. Publish to a channel, every replica's subscriber receives it, and
each forwards to its own connected clients.

Note pub/sub is **fire and forget**. A replica that is down misses the
message entirely. For anything that must not be lost, use a real queue.

## Fail soft

Neither of these should take your service down:

```go
cache, err := valkey.NewClient(opt)
if err != nil {
    slog.Warn("cache unavailable, continuing without it", "error", err)
    cache = nil        // callers check, or use a no-op implementation
}
```

A cache being unreachable should mean slower responses, not errors. A
no-op implementation of the interface is cleaner than nil checks
everywhere — every miss, every write a success, and the rest of the
code never knows.

Object storage is usually the opposite: if uploads are the product,
failing loudly at startup is right. Decide deliberately which of the
two each dependency is.

> **From Python:** the S3 client is boto3 with explicit error types
> instead of `ClientError` plus a string code, and `valkey-go` is
> `redis-py` with a builder API. The interface-plus-local-implementation
> habit is the same one you would get from moto or fakeredis, except it
> is your own code rather than a mocking layer.

## Quick reference

| Task | Form |
|---|---|
| the abstraction | a small `ObjectStore` interface over `io.Reader` |
| local development | a filesystem implementation, no credentials |
| S3 client | `config.LoadDefaultConfig` + `s3.NewFromConfig` |
| missing object | `errors.As` for `*types.NoSuchKey` **and** `*types.NotFound` |
| large uploads | `manager.NewUploader` |
| client-direct transfer | `s3.NewPresignClient` |
| cache client | `valkey.NewClient`, commands via `c.B()` |
| a miss | `valkey.IsValkeyNil(err)` — not a failure |
| expiry | always `.Ex(d)` |
| subscribing | `c.Dedicate()` — a subscribed connection is exclusive |
| cross-replica events | publish to a channel; each replica fans out |
| when it is down | degrade for a cache, fail loudly for storage |

## Sources

- [aws-sdk-go-v2 S3 — pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3)
- [AWS SDK for Go v2 — aws.github.io/aws-sdk-go-v2/docs/](https://aws.github.io/aws-sdk-go-v2/docs/)
- [`valkey-go` — pkg.go.dev/github.com/valkey-io/valkey-go](https://pkg.go.dev/github.com/valkey-io/valkey-go)
- [Valkey commands — valkey.io/commands/](https://valkey.io/commands/)
- [Redis pub/sub — redis.io/docs/latest/develop/interact/pubsub/](https://redis.io/docs/latest/develop/interact/pubsub/)
