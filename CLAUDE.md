# Project: learn-go-from-python

A personal Go learning repository. The user maintains notes and runnable
examples as they learn the language; Claude assists with explanations,
articles (conspects), and example code.

## About the user

- Long Python background (years of experience).
- Learning Go from scratch.
- Wants Go explained **on its own terms**. Python analogies are welcome
  as one-line *supplementary* notes after the main Go explanation —
  never as the primary framing.

## Repository layout

```
learn_go/
├── CLAUDE.md
├── README.md
├── 01-ecosystem-and-installation/   ← topic 1 (covered)
│   ├── 01-what-is-go.md
│   ├── 02-go-subcommands.md
│   ├── ...
│   └── 09-demo-project/             ← runnable Go module
├── 02-language-basics/              ← topic 2 (in progress)
│   └── ...
└── NN-next-topic/
```

- Each topic gets its own folder named `NN-kebab-case-topic/` (two-digit
  prefix).
- Inside each topic folder, every file is numbered `NN-name.ext` —
  conspect `.md` files and example `.go` files share one continuous
  sequence so the intended reading/running order is always clear.
- Runnable multi-file Go projects live in subfolders inside the topic
  (e.g. `09-demo-project/`). Inside those subfolders Go's own filename
  conventions apply (`go.mod`, `main.go`, `*_test.go`, etc.) — the
  numeric-prefix rule does not.

## Workflow

1. **Discuss in chat first.** A new topic starts with conversation —
   the user asks questions; Claude answers them in chat.
2. **Then capture.** When the user is satisfied, they ask Claude to
   "make a note", "create a conspect", or "create an example file".
   Only at that point does Claude write files.
3. **One topic folder per subject.** Create the `NN-topic/` folder for
   a new subject up front (so we have a home for notes) without
   pre-populating content.

## Style of explanations and conspects

- **Go-first.** Explain the Go-side rule, syntax, and behaviour fully.
- **Python analogies are supplementary.** A single `> **From Python:**
  ...` blockquote after the main explanation is welcome where it
  clarifies; do not structure articles as Python-to-Go translations.
- **Code examples are mandatory.** Every Go concept ships with at
  least one small (3–15 line) runnable snippet in a fenced ```go
  block. Pair tiny examples with each facet rather than one big example.
  Show expected output as a `// output: ...` comment when it matters.
  Use `// compile error: ...` to label snippets that intentionally
  don't compile.
- **Source-verified.** Verify facts against go.dev / pkg.go.dev /
  Go spec before writing. End every article with a `## Sources` section
  listing the URLs consulted as markdown links.
- **GitHub-flavoured Markdown.** Use fenced code blocks with language
  tags (```go, ```bash), tables, and standard headings.

## Tooling

- Go is installed via Homebrew. The materials target **Go 1.26.4**.
- All content should be written as if 1.26.4 is the only version that
  exists. Do **not** annotate features with "Go 1.X+" or "introduced
  in Go 1.Y" — write everything as a present-tense fact. The exception
  is the version-management article itself, which is intrinsically
  about version mechanics.
- Always work in the main repo at `/Users/oduvan/www/learn_go`.
  **Never** use git worktrees for this project, even if the harness
  spawns one by default.

## Recommended editor

The user opens files in their IDE. Examples should be self-contained
(compilable as `package main` with `func main()` where possible) so
they can be pasted into the Go playground or run with `go run`.
