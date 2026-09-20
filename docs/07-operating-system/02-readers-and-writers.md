# Readers and writers

`io.Reader` and `io.Writer` are the two most important interfaces in Go.
[Interfaces](../03-object-oriented-go/02-interfaces.md) introduced them
as examples of small interfaces; this is how you actually use them.

```go
b, err := io.ReadAll(strings.NewReader("abc"))
fmt.Println(err)                 // output: <nil>
fmt.Printf("%q\n", string(b))   // output: "abc"
```

## One method each

```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
```

That is the whole contract, and it is why a file, a network connection,
an HTTP body, a gzip stream, a `strings.Reader` and a `bytes.Buffer` are
all interchangeable. A function taking an `io.Reader` works with every
one of them, including ones that did not exist when it was written.

Take the interface, not `*os.File`. It costs nothing and makes the
function testable with `strings.NewReader`.

## `io.Copy` is the workhorse

Copying moves data in fixed-size chunks, so memory stays flat no matter
how big the source is:

```go
var dst bytes.Buffer
n, err := io.Copy(&dst, strings.NewReader("stream me"))
fmt.Println(n, err, dst.String())   // output: 9 <nil> stream me
```

Copying a file is two opens and a `Copy` — there is no `os.Copy`:

```go
in, err := os.Open(src)
if err != nil {
    return err
}
defer in.Close()

out, err := os.Create(dst)
if err != nil {
    return err
}
if _, err := io.Copy(out, in); err != nil {
    out.Close()
    return err
}
return out.Close()   // the close that matters: report its error
```

`io.Discard` is a writer that throws everything away, useful for draining
a body you do not care about:

```go
n, _ := io.Copy(io.Discard, strings.NewReader("thrown away"))
fmt.Println(n)   // output: 11
```

## `io.EOF` is a value, not a failure

`Read` returns the number of bytes it got and an error. At the end of
input that error is `io.EOF`, which is a normal result:

```go
r := strings.NewReader("hello")
buf := make([]byte, 2)
for {
    n, err := r.Read(buf)
    if n > 0 {
        fmt.Printf("%q ", string(buf[:n]))
    }
    if err == io.EOF {
        break
    }
}
// output: "he" "ll" "o"
```

Two rules that catch people. **Process the bytes before checking the
error** — a `Read` may return data *and* `io.EOF` together. And always
slice to `buf[:n]`; the rest of the buffer holds whatever was there
before.

You rarely write this loop. `io.ReadAll`, `io.Copy` and `bufio.Scanner`
handle it for you.

`io.ReadFull` is the exception that treats a short read as an error:

```go
_, err := io.ReadFull(strings.NewReader("abc"), make([]byte, 10))
fmt.Println(errors.Is(err, io.ErrUnexpectedEOF), err)
// output: true unexpected EOF
```

## `bufio.Scanner` for lines

Reading line by line is common enough to have a dedicated type:

```go
sc := bufio.NewScanner(r)
for sc.Scan() {
    fmt.Println(sc.Text())
}
if err := sc.Err(); err != nil {
    return err
}
```

`Scan` returns false at end of input *and* on error, so **checking
`sc.Err()` afterwards is not optional** — skip it and a read failure
looks exactly like a clean end of file.

`Text()` returns the line without its newline. `Bytes()` avoids the
allocation but is only valid until the next `Scan`.

Change what counts as a token with `Split`:

```go
sc := bufio.NewScanner(strings.NewReader("a bb  ccc"))
sc.Split(bufio.ScanWords)
for sc.Scan() {
    fmt.Printf("%q ", sc.Text())
}
// output: "a" "bb" "ccc"
```

`bufio.ScanLines` is the default; `ScanWords`, `ScanRunes` and
`ScanBytes` are the others.

### The 64 KB limit

A `Scanner` refuses any token longer than 64 KB, and it reports this as
an error rather than truncating:

```go
long := strings.Repeat("x", 100000)

sc := bufio.NewScanner(strings.NewReader(long))
fmt.Println(sc.Scan(), sc.Err())
// output: false bufio.Scanner: token too long
```

This is the classic bug in a log processor: it works until one enormous
line arrives, then stops, and without the `sc.Err()` check it stops
*silently*. Raise the cap when lines can be long:

```go
sc := bufio.NewScanner(strings.NewReader(long))
sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
fmt.Println(sc.Scan(), sc.Err(), len(sc.Text()))
// output: true <nil> 100000
```

For genuinely unbounded lines use `bufio.Reader.ReadString('\n')`, which
grows as needed.

## `bufio.Writer` must be flushed

Wrapping a writer batches many small writes into few large ones. The
buffered data is not written until you say so:

```go
var out bytes.Buffer
w := bufio.NewWriter(&out)
w.WriteString("buffered")

fmt.Printf("%q ", out.String())   // output: ""
w.Flush()
fmt.Printf("%q\n", out.String())  // output: "buffered"
```

`defer w.Flush()` discards the error. If the data matters, flush
explicitly and check.

## Combining readers and writers

```go
var a, b bytes.Buffer
mw := io.MultiWriter(&a, &b)
fmt.Fprint(mw, "both")            // writes to both
```

```go
lb, _ := io.ReadAll(io.LimitReader(strings.NewReader("abcdefgh"), 3))
fmt.Printf("%q\n", string(lb))    // output: "abc"
```

`io.LimitReader` caps how much is read — the standard defence against a
request body that claims to be infinite. `io.TeeReader` passes data
through while copying it somewhere else, and `io.MultiReader`
concatenates readers end to end.

## Writing your own

Implement one method and everything above works with your type:

```go
type upperReader struct{ r io.Reader }

func (u upperReader) Read(p []byte) (int, error) {
    n, err := u.r.Read(p)
    for i := range p[:n] {
        if p[i] >= 'a' && p[i] <= 'z' {
            p[i] -= 32
        }
    }
    return n, err
}
```

```go
b, _ := io.ReadAll(upperReader{strings.NewReader("shout")})
fmt.Printf("%q\n", string(b))   // output: "SHOUT"
```

Note it transforms only `p[:n]` and passes the error straight through.

## Where each source comes from

| You have | Wrap it with |
|---|---|
| a `string` | `strings.NewReader(s)` |
| a `[]byte` | `bytes.NewReader(b)` |
| growable bytes | `bytes.Buffer` — reader *and* writer |
| a string being built | `strings.Builder` — writer only |
| a file | `os.Open` / `os.Create` |
| standard streams | `os.Stdin`, `os.Stdout`, `os.Stderr` |
| nothing | `io.Discard` |

`strings.NewReader` is why every example in this article is runnable
without a file, and it is how you should test anything reader-shaped.

> **From Python:** `io.Reader` is the read end of a file object, but
> narrowed to one method, so anything can be one. `io.Copy` is
> `shutil.copyfileobj`. Two differences that bite: `EOF` is a returned
> value rather than an empty string, and a `bufio.Scanner` silently caps
> lines at 64 KB where Python's iteration has no such limit.

## Quick reference

| Task | Call |
|---|---|
| read everything | `io.ReadAll(r)` |
| stream one to another | `io.Copy(w, r)` |
| copy a file | `os.Open` + `os.Create` + `io.Copy` |
| line by line | `bufio.NewScanner(r)`, then **check `sc.Err()`** |
| long lines | `sc.Buffer(...)` or `bufio.Reader.ReadString('\n')` |
| batch small writes | `bufio.NewWriter(w)` + `Flush()` |
| cap the input | `io.LimitReader(r, n)` |
| fan out | `io.MultiWriter(a, b)` |
| observe in passing | `io.TeeReader(r, w)` |
| throw away | `io.Discard` |
| a reader from a string | `strings.NewReader(s)` |

## Sources

- [`io` package reference — pkg.go.dev/io](https://pkg.go.dev/io)
- [`bufio` package reference — pkg.go.dev/bufio](https://pkg.go.dev/bufio)
- [`bufio.Scanner` — pkg.go.dev/bufio#Scanner](https://pkg.go.dev/bufio#Scanner)
- [`io.Copy` — pkg.go.dev/io#Copy](https://pkg.go.dev/io#Copy)
- [`io.Reader` — pkg.go.dev/io#Reader](https://pkg.go.dev/io#Reader)
- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
