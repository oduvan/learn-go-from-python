# Imports

Every `.go` file declares which packages it uses with an `import` block. Imports sit between the `package` clause and the rest of the file.

## The basic shape

```go
package main

import "fmt"

func main() {
    fmt.Println("hello")
}
```

The string `"fmt"` is an **import path**. It tells the compiler where to find the package — for standard-library packages this is just the package's name; for third-party packages it's a URL-shaped path that the module system resolves.

```go
import "fmt"                            // standard library
import "net/http"                       // standard library subpackage
import "github.com/gorilla/mux"         // third-party module
import "example.com/myapp/internal/db"  // internal to your own module
```

## Single vs grouped

Two equivalent forms. Use grouped when you have more than one — almost always.

```go
import "fmt"
import "math"
import "os"
```

vs

```go
import (
    "fmt"
    "math"
    "os"
)
```

The grouped form is what `gofmt` and editors produce automatically. Standard convention is to keep std-lib imports in one block and external imports in a separate block, separated by a blank line:

```go
import (
    "fmt"
    "net/http"
    "strings"

    "github.com/gorilla/mux"
    "go.uber.org/zap"
)
```

## Using imported identifiers

Inside the file you refer to exported names through the package's **base name** — usually the last segment of the import path:

```go
import "net/http"

func main() {
    http.ListenAndServe(":8080", nil)   // base name is "http"
}
```

Two non-obvious rules:

1. **The base name comes from the package's own `package` declaration**, not from the import path. They are *usually* the same, but not always — `package "yaml.v3"` declares itself as `package yaml`, so you reference it as `yaml.Marshal`, not `v3.Marshal`.
2. **Only exported names are visible.** Capitalized identifiers cross package boundaries; lowercase ones don't. (See [02-basic-types.md](02-basic-types.md) for the capitalization rule.)

```go
import "strings"

s := strings.ToUpper("go")     // exported — works
s := strings.toupper("go")     // compile error — lowercase, not exported
```

## Renaming with an alias

You can give an imported package a local nickname by writing it before the path:

```go
import (
    f "fmt"
    rng "math/rand"
)

func main() {
    f.Println(rng.Intn(10))
}
```

When to reach for it:

- Two imports would collide otherwise (e.g. `crypto/rand` and `math/rand`).
- The package's base name is awkward or long for the local file.

Don't alias just to shorten — `fmt.Println` is already short.

> **From Python:** ≈ `import math as m`. Same idea, syntax flipped.

## Blank import — `_` for side effects only

A blank import compiles and links the package but doesn't bind any name in your file. You can't reference anything from it. The point is to run the package's `init()` function for its side effects:

```go
import (
    "database/sql"
    _ "github.com/lib/pq"      // registers the "postgres" driver
)

func main() {
    db, err := sql.Open("postgres", connStr)
    // ...
}
```

Common cases: database drivers, image-format decoders, profiling hooks.

```go
import (
    _ "image/png"              // registers PNG decoder with image.Decode
    _ "image/jpeg"             // registers JPEG decoder
    _ "net/http/pprof"         // registers /debug/pprof/* handlers
)
```

> **From Python:** there is no direct analog. Python doesn't have an "import for side effects but never name it" form — every `import` introduces a name.

## Dot import — almost never use

`import . "path"` dumps all exported names from the package into the current file's namespace, so you can call them unqualified:

```go
import . "math"

func main() {
    fmt.Println(Pi)            // Pi without math. — works
}
```

This is **strongly discouraged** in production code — it makes it impossible to tell where an identifier came from when reading a file. The one legitimate use is inside test files that need to reach deep into an `_test` package, and even that is rare.

> **From Python:** ≈ `from math import *`. Same problem, same advice.

## Unused imports are a compile error

```go
import (
    "fmt"
    "os"            // imported but never referenced
)

func main() {
    fmt.Println("hi")
}
// compile error: "os" imported and not used
```

This is intentional — the language designers wanted import lists to stay accurate. There are two escape hatches:

1. **Blank import:** `_ "os"` if you genuinely want the side effects.
2. **Suppress while editing:** `_ = os.Getpid` somewhere in the file, *temporarily*. Use `goimports` or your editor's organize-imports command to clean these up before committing.

> **From Python:** Python warns about unused imports through linters (`pyflakes`, `ruff`) but never blocks compilation. Go elevates it to a hard error.

## Import cycles are forbidden

If package `a` imports `b`, then `b` cannot import `a` directly or indirectly.

```
package alpha → imports beta
package beta  → imports alpha    // compile error: import cycle
```

The compiler detects cycles transitively. If you hit one, the fix is usually:

- Extract the shared dependency into a third package.
- Move the conflicting code closer to its caller.
- Use an interface defined in the caller's package and have the callee implement it (Go's "accept interfaces, return concrete types" idiom).

## Standard library imports — small tour

The most-used std-lib packages:

| Import | Common use |
|---|---|
| `"fmt"` | Formatted printing, `Println`, `Printf`, `Sprintf`. |
| `"strings"` | String manipulation (`ToUpper`, `Split`, `Replace`, `Contains`). |
| `"strconv"` | String ↔ number conversion. |
| `"errors"` | `errors.New`, `errors.Is`, `errors.As`. |
| `"os"` | Process/file system, env vars, `os.Args`. |
| `"io"` | `Reader` / `Writer` interfaces, `Copy`. |
| `"bufio"` | Buffered I/O, `Scanner`. |
| `"time"` | `time.Now`, `time.Duration`, timers. |
| `"net/http"` | HTTP client and server. |
| `"encoding/json"` | JSON marshal/unmarshal. |
| `"sync"` | `Mutex`, `WaitGroup`, `Once`. |
| `"context"` | Cancellation, deadlines, request-scoped values. |
| `"log"` | Simple logging; `log/slog` for structured logs. |
| `"reflect"` | Runtime type introspection. |

Browse the full list at [pkg.go.dev/std](https://pkg.go.dev/std).

## Tooling

You almost never type imports by hand. `goimports` (and `gopls`, which most editors run on save) adds missing imports and removes unused ones automatically — see [08-additional-tools.md](../01-ecosystem-and-installation/08-additional-tools.md).

```bash
goimports -w .                 # rewrite all .go files in this dir
```

After editing, this leaves the import block clean and grouped correctly.

## Quick reference

| Form | Effect |
|---|---|
| `import "path"` | Bind by the package's own name. |
| `import alias "path"` | Bind under `alias`. |
| `import _ "path"` | Run `init()`s, no name bound — for side effects. |
| `import . "path"` | Merge package's exports into this file's namespace. Avoid. |

## Sources

- [Import declarations — go.dev/ref/spec#Import_declarations](https://go.dev/ref/spec#Import_declarations)
- [Package clause — go.dev/ref/spec#Package_clause](https://go.dev/ref/spec#Package_clause)
- [Exported identifiers — go.dev/ref/spec#Exported_identifiers](https://go.dev/ref/spec#Exported_identifiers)
- [Effective Go: package names and import paths — go.dev/doc/effective_go#package-names](https://go.dev/doc/effective_go#package-names)
- [Standard library index — pkg.go.dev/std](https://pkg.go.dev/std)
