# Hashing and random values

A grab-bag of small standard-library packages that turn up constantly:
hashing, signing, generating tokens, encoding bytes as text, and gzip.
None is hard, and each has one way to get it wrong.

```go
sum := sha256.Sum256([]byte("hello"))
fmt.Println(hex.EncodeToString(sum[:]))
// output: 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
```

## Hashing

`sha256.Sum256` is the one-shot form. It returns a **fixed-size array**,
`[32]byte`, so passing it anywhere that wants a slice needs `sum[:]`:

```go
sum := sha256.Sum256([]byte("hello"))
fmt.Println(len(sum))       // output: 32
fmt.Printf("%x\n", sum)     // output: 2cf24dba...938b9824
```

`%x` on the array works directly, which is the shortest way to print a
digest.

For data you do not want to hold in memory, a hash is an `io.Writer`:

```go
h := sha256.New()
io.Copy(h, r)
fmt.Println(hex.EncodeToString(h.Sum(nil)))
```

`h.Sum(nil)` appends the digest to the slice you pass — `nil` means
"give me a fresh one". It does not reset the hash.

This is how you checksum a file without reading it whole, and how
`io.Copy` from the [readers and writers](02-readers-and-writers.md)
article pays off again.

**SHA-256 is not for passwords.** It is fast, which is exactly wrong for
a password. Use a deliberately slow function such as bcrypt or Argon2 —
both live outside the standard library.

## HMAC: proving a message came from you

A plain hash proves integrity. An HMAC proves *authenticity*, because
producing it requires the key:

```go
key := []byte("secret")

m := hmac.New(sha256.New, key)
m.Write([]byte("message"))
sig := m.Sum(nil)
```

Verify by recomputing and comparing with `hmac.Equal`:

```go
m2 := hmac.New(sha256.New, key)
m2.Write([]byte("message"))
fmt.Println(hmac.Equal(sig, m2.Sum(nil)))   // output: true
```

**Use `hmac.Equal`, never `==` or `bytes.Equal`.** A normal comparison
returns as soon as two bytes differ, so how long it takes leaks how much
of the signature was correct — enough, over many attempts, to recover a
valid one. `hmac.Equal` always takes the same time.

This is the mechanism behind signed cookies and inbound webhook
verification: the sender signs with a shared secret, you recompute and
compare.

For non-HMAC secrets — an API token, a shared key — the general form is:

```go
fmt.Println(subtle.ConstantTimeCompare([]byte("abc"), []byte("abc")))   // output: 1
fmt.Println(subtle.ConstantTimeCompare([]byte("abc"), []byte("abd")))   // output: 0
```

It returns an `int`, not a `bool` — `1` for equal. Note it returns `0`
for inputs of different lengths without comparing at all, so length is
not protected.

## Random values: `crypto/rand`, not `math/rand`

Two packages, and picking the wrong one is a security bug rather than a
style choice.

| Package | For |
|---|---|
| `math/rand/v2` | simulations, jitter, shuffling, test data |
| `crypto/rand` | tokens, session IDs, keys, nonces, salts |

`math/rand` is predictable from a handful of outputs. Anything an
attacker must not guess comes from `crypto/rand`:

```go
buf := make([]byte, 32)
n, err := rand.Read(buf)
fmt.Println(n, err)   // output: 32 <nil>
```

Thirty-two bytes is the usual size for a session token — 256 bits of
entropy.

## Encoding bytes as text

Random bytes are not printable, so they get encoded. The variant
matters:

```go
raw := []byte{0xfb, 0xff, 0x01}

fmt.Println(base64.StdEncoding.EncodeToString(raw))       // output: +/8B
fmt.Println(base64.URLEncoding.EncodeToString(raw))       // output: -_8B
fmt.Println(base64.RawURLEncoding.EncodeToString(raw))    // output: -_8B
```

`StdEncoding` produces `+`, `/` and `=`, all of which need escaping in a
URL or a cookie. `URLEncoding` swaps in `-` and `_`; the `Raw` variants
drop the `=` padding. For anything that travels in a URL, a header or a
cookie, use `RawURLEncoding`:

```go
tok := base64.RawURLEncoding.EncodeToString(buf)
fmt.Println(len(tok))   // output: 43
```

32 bytes becomes 43 characters with no padding and nothing to escape.

`encoding/hex` is the other option — twice as long, but unambiguous and
the convention for digests.

## `hash/fnv` for non-cryptographic hashing

When you need a number derived from a string — a shard index, a lock
key, a cache bucket — a cryptographic hash is overkill. FNV is fast and
deterministic across runs and machines:

```go
f := fnv.New64a()
f.Write([]byte("job-name"))
fmt.Println(f.Sum64())   // output: 17796606368501603464
```

That determinism is the point: Go's built-in map hashing is randomised
per process, so it cannot be used to agree on anything between
processes. FNV can.

Never use it where an attacker chooses the input — it is trivial to
produce collisions on purpose.

## gzip

`compress/gzip` wraps a writer and a reader, so it composes with
everything else:

```go
var buf bytes.Buffer
zw := gzip.NewWriter(&buf)
zw.Write(data)
zw.Close()          // required: writes the footer
```

`Close` is not optional — it flushes the final block and writes the
gzip footer. A stream without it is truncated and will not decompress.
Reading back:

```go
zr, err := gzip.NewReader(&buf)
if err != nil {
    return err
}
out, err := io.ReadAll(zr)
if err != nil {
    return err
}
```

> **From Python:** `hashlib.sha256().hexdigest()` becomes
> `hex.EncodeToString(sum[:])`, `hmac.compare_digest` becomes
> `hmac.Equal`, and `secrets.token_urlsafe(32)` becomes
> `crypto/rand` plus `base64.RawURLEncoding`. The same rule applies in
> both languages: `random` for simulations, `secrets` for security.

## Quick reference

| Task | Call |
|---|---|
| hash a slice | `sha256.Sum256(b)` → `[32]byte`, use `sum[:]` |
| hash a stream | `h := sha256.New()`, `io.Copy(h, r)`, `h.Sum(nil)` |
| digest as text | `hex.EncodeToString(...)` or `%x` |
| sign | `hmac.New(sha256.New, key)` |
| verify a signature | `hmac.Equal(a, b)` — never `==` |
| compare a secret | `subtle.ConstantTimeCompare(a, b) == 1` |
| unguessable bytes | `crypto/rand.Read(buf)` |
| token text | `base64.RawURLEncoding.EncodeToString(buf)` |
| simulations, jitter | `math/rand/v2` |
| deterministic bucket key | `fnv.New64a()` |
| compress | `gzip.NewWriter(w)` — **must `Close()`** |
| passwords | not here; bcrypt or Argon2 |

## Sources

- [`crypto/sha256` — pkg.go.dev/crypto/sha256](https://pkg.go.dev/crypto/sha256)
- [`crypto/hmac` — pkg.go.dev/crypto/hmac](https://pkg.go.dev/crypto/hmac)
- [`crypto/subtle` — pkg.go.dev/crypto/subtle](https://pkg.go.dev/crypto/subtle)
- [`crypto/rand` — pkg.go.dev/crypto/rand](https://pkg.go.dev/crypto/rand)
- [`encoding/base64` — pkg.go.dev/encoding/base64](https://pkg.go.dev/encoding/base64)
- [`hash/fnv` — pkg.go.dev/hash/fnv](https://pkg.go.dev/hash/fnv)
- [`compress/gzip` — pkg.go.dev/compress/gzip](https://pkg.go.dev/compress/gzip)
