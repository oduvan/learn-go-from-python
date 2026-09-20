# Files and paths

Two packages cover almost everything: `os` for the filesystem, and
`path/filepath` for manipulating the names. The whole-file calls are the
ones you reach for most.

```go
if err := os.WriteFile("notes.txt", []byte("hello\n"), 0o644); err != nil {
    return err
}

b, err := os.ReadFile("notes.txt")
if err != nil {
    return err
}
fmt.Printf("%q\n", string(b))   // output: "hello\n"
```

`ReadFile` returns `[]byte`, not a string, and it opens, reads and closes
for you. Use it whenever the file fits comfortably in memory; stream it
otherwise, which the [next article](02-readers-and-writers.md) covers.

## Permissions are octal

That `0o644` is a Unix permission bitmask — owner read/write, everyone
else read. The `0o` prefix is Go's octal literal syntax. It applies only
when the call *creates* the file; an existing file keeps its own mode.

`0o644` for data, `0o755` for directories and executables, `0o600` when
the contents are sensitive.

## Checking what something is

```go
fi, err := os.Stat("notes.txt")
if err != nil {
    return err
}
fmt.Println(fi.Name(), fi.Size(), fi.IsDir(), fi.Mode().Perm())
// output: notes.txt 6 false -rw-r--r--
```

`Perm()` prints the symbolic form of the same bits: three groups of
`rwx` for owner, group and others, with `-` where a permission is
absent. `-rw-r--r--` is `0o644` written the other way round.

## Missing files: test the error, not the path

Every filesystem call returns a `*fs.PathError` wrapping a specific
cause, so `errors.Is` answers the "does it exist" question:

```go
_, err := os.ReadFile("nope.txt")

fmt.Println(errors.Is(err, fs.ErrNotExist))   // output: true

var pe *fs.PathError
fmt.Println(errors.As(err, &pe), pe.Op)       // output: true open
```

`os.ErrNotExist` and `fs.ErrNotExist` are the same value, so either
works. There is an `os.IsNotExist(err)` helper in older code; prefer
`errors.Is`, which sees through wrapping.

Resist the urge to call `os.Stat` first and then open. Between the two
calls the answer can change, and you have to handle the failure anyway —
just open it and check the error.

## Opening for control

`os.Create` truncates or creates. `os.OpenFile` is the general form, and
appending is the common reason to reach for it:

```go
f, _ := os.Create("log.txt")
fmt.Fprintln(f, "line1")
f.Close()

af, _ := os.OpenFile("log.txt", os.O_APPEND|os.O_WRONLY, 0o644)
fmt.Fprintln(af, "line2")
af.Close()
```

```go
b, _ := os.ReadFile("log.txt")
fmt.Printf("%q\n", string(b))   // output: "line1\nline2\n"
```

An `*os.File` is an `io.Writer`, which is why `fmt.Fprintln` works on it.
In real code close with `defer`:

```go
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
```

For a file you *wrote*, a deferred `Close` whose error you discard is a
real risk. The write calls can succeed while the filesystem only reports
a failure — a full disk, a network mount dropping — when the descriptor
is closed. Close it explicitly and check the error before reporting
success.

## Directories

```go
os.MkdirAll("a/b/c", 0o755)   // creates parents, no error if it exists
os.Remove(path)               // one file or one empty directory
os.RemoveAll(dir)             // recursive; no error if absent
```

`MkdirAll` is the one to reach for — plain `Mkdir` fails if the parent is
missing, and errors when the directory already exists. Removing a file
that is not there *is* an error:

```go
fmt.Println(os.Remove("notes.txt"))        // output: <nil>
fmt.Println(os.Remove("notes.txt") != nil) // output: true
```

For scratch space, let the library pick the location and clean up after
yourself:

```go
dir, err := os.MkdirTemp("", "demo")
if err != nil {
    return err
}
defer os.RemoveAll(dir)
```

## Building paths with `filepath`

Never join paths with `+` or `/`. `filepath.Join` uses the right
separator for the platform, and it cleans the result:

```go
fmt.Println(filepath.Join("a", "b", "..", "c"))   // output: a/c
fmt.Println(filepath.Join("a", "", "b"))          // output: a/b
```

Empty segments vanish, which makes it safe to join a variable that might
be blank.

```go
fmt.Println(filepath.Base("/x/y/z.txt"))   // output: z.txt
fmt.Println(filepath.Dir("/x/y/z.txt"))    // output: /x/y
fmt.Println(filepath.Ext("/x/y/z.txt"))    // output: .txt
```

`Ext` includes the dot. Stripping it is a `strings` job:

```go
name := "z.txt"
fmt.Println(strings.TrimSuffix(name, filepath.Ext(name)))   // output: z
```

`filepath.Rel` gives the path from one place to another, which is how you
turn absolute walk results back into readable names:

```go
rel, _ := filepath.Rel("/x/y", "/x/y/z/w.txt")
fmt.Println(rel)   // output: z/w.txt
```

There is also a `path` package. It is for slash-separated things that are
*not* filesystem paths — URLs, embedded FS names. For files, use
`filepath`.

## Listing and walking

`os.ReadDir` lists one directory. It returns `fs.DirEntry` values, which
know the name and whether it is a directory without an extra syscall:

```go
ents, err := os.ReadDir(dir)
if err != nil {
    return err
}
for _, e := range ents {
    fmt.Println(e.Name(), e.IsDir())
}
```

`filepath.WalkDir` recurses. The callback receives an error argument,
and the first thing to do is check it — otherwise an unreadable
subdirectory silently truncates your walk:

```go
var found []string
walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    if !d.IsDir() && filepath.Ext(path) == ".go" {
        found = append(found, path)
    }
    return nil
})
if walkErr != nil {
    return walkErr
}
```

Returning a non-nil error stops the walk and `WalkDir` returns it.
Returning `fs.SkipDir` from a directory skips its whole subtree instead:

```go
filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    if d.IsDir() && d.Name() == "vendor" {
        return fs.SkipDir
    }
    // ...
    return nil
})
```

That is the idiom for ignoring `vendor`, `node_modules` or `.git`.

## Renaming and copying

`os.Rename` moves a file, atomically when both paths are on the same
filesystem — which is how you write a file safely: write to a temporary
name, then rename over the target, so a reader never sees a half-written
file.

There is no `os.Copy`. Open both and use `io.Copy`, in the next article.

> **From Python:** `os.ReadFile`/`WriteFile` are `pathlib.read_bytes`/
> `write_bytes`, `filepath.Join` is `os.path.join`, `WalkDir` is
> `os.walk` with a callback instead of a generator. The habit to drop is
> `if os.path.exists(...)`: here you attempt the operation and inspect
> the error with `errors.Is`.

## Quick reference

| Task | Call |
|---|---|
| read a whole file | `os.ReadFile(p)` → `[]byte` |
| write a whole file | `os.WriteFile(p, b, 0o644)` |
| open for reading / writing | `os.Open` / `os.Create` / `os.OpenFile` |
| append | `os.OpenFile(p, os.O_APPEND\|os.O_WRONLY, 0o644)` |
| metadata | `os.Stat(p)` → `fs.FileInfo` |
| does it exist | `errors.Is(err, fs.ErrNotExist)` |
| make directories | `os.MkdirAll(p, 0o755)` |
| delete | `os.Remove` / `os.RemoveAll` |
| temp directory | `os.MkdirTemp("", "prefix")` + `defer os.RemoveAll` |
| build a path | `filepath.Join(...)` |
| split a path | `Base`, `Dir`, `Ext`, `Rel` |
| list one directory | `os.ReadDir(p)` |
| walk a tree | `filepath.WalkDir(root, fn)`, `fs.SkipDir` to prune |
| move / replace atomically | `os.Rename(tmp, target)` |

## Sources

- [`os` package reference — pkg.go.dev/os](https://pkg.go.dev/os)
- [`path/filepath` package reference — pkg.go.dev/path/filepath](https://pkg.go.dev/path/filepath)
- [`io/fs` package reference — pkg.go.dev/io/fs](https://pkg.go.dev/io/fs)
- [`filepath.WalkDir` — pkg.go.dev/path/filepath#WalkDir](https://pkg.go.dev/path/filepath#WalkDir)
- [`fs.PathError` — pkg.go.dev/io/fs#PathError](https://pkg.go.dev/io/fs#PathError)
