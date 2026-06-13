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
learn-go-from-python/
├── CLAUDE.md
├── README.md                        ← short repo intro, points at the site
├── mkdocs.yml                       ← MkDocs Material config
├── .github/workflows/pages.yml      ← builds + deploys to GitHub Pages
└── docs/                            ← MkDocs docs_dir; everything below
    │                                  is what becomes the published site
    ├── index.md                     ← home page
    ├── 01-ecosystem-and-installation/   ← topic 1 (covered)
    │   ├── 01-what-is-go.md
    │   ├── 02-go-subcommands.md
    │   ├── ...
    │   └── 09-demo-project/             ← runnable Go module
    ├── 02-language-basics/              ← topic 2 (in progress)
    │   └── ...
    └── NN-next-topic/
```

- All conspects and runnable examples live **under `docs/`**. There are
  no topic folders at the repo root anymore.
- Each topic gets its own folder named `docs/NN-kebab-case-topic/`
  (two-digit prefix).
- Inside each topic folder, every file is numbered `NN-name.ext` —
  conspect `.md` files and example `.go` files share one continuous
  sequence so the intended reading/running order is always clear.
- Runnable multi-file Go projects live in subfolders inside the topic
  (e.g. `09-demo-project/`). Inside those subfolders Go's own filename
  conventions apply (`go.mod`, `main.go`, `*_test.go`, etc.) — the
  numeric-prefix rule does not. The demo module still builds and runs
  in place — Go does not care that the module is nested under `docs/`.

## Workflow

1. **Discuss in chat first.** A new topic starts with conversation —
   the user asks questions; Claude answers them in chat.
2. **Then capture.** When the user is satisfied, they ask Claude to
   "make a note", "create a conspect", or "create an example file".
   Only at that point does Claude write files.
3. **One topic folder per subject.** Create the folder as
   `docs/NN-topic/` up front (so we have a home for notes) without
   pre-populating content. When a new topic folder is created, also:
   - add a `nav:` entry for the topic in `mkdocs.yml`,
   - add a `nav:` entry for each new conspect file inside that topic
     as files are created (titles are user-facing — drop the numeric
     prefix and use Title Case, e.g. `Variables and constants:
     02-language-basics/01-variables-and-constants.md`),
   - add the topic title and every per-file title to the
     `nav_translations:` block of the `uk` locale in `mkdocs.yml` so
     the Ukrainian navigation labels are not left in English.
4. **Every English change is followed by a Ukrainian change.** This
   project ships bilingual (English default, Ukrainian via
   mkdocs-static-i18n). Whenever you create or edit a `docs/.../NN-foo.md`
   article, you must create or update its `docs/.../NN-foo.uk.md`
   companion **in the same commit**. The same applies to `docs/index.md`
   and to anything inside `09-demo-project/README.md`. Quick rules:
   - **New article** → write `NN-foo.md` and `NN-foo.uk.md` together.
   - **Edit to an English article** → port the same edit into the `.uk.md`
     translation in the same commit. Don't let translations drift.
   - **Code blocks**: keep snippets, identifiers, and `// output: ...`
     comments **identical** between languages — only translate the prose
     around them and any natural-language code comments.
   - **Nav labels**: any new English nav label in `mkdocs.yml` needs a
     matching entry in `plugins.i18n.languages[uk].nav_translations`.
   - **`fallback_to_default: true`** keeps the site shippable while a
     translation is in flight — but treat that as the safety net, not
     the workflow. Don't merge an English-only edit without writing
     the Ukrainian one alongside.
   - The translation rule applies only to user-facing docs under
     `docs/`. `CLAUDE.md`, `README.md`, the workflow file, and other
     repo-meta files stay English-only.

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
- **Run the code, don't just claim it.** Doc-reading alone is not
  enough — every non-trivial snippet must be executed in a throwaway
  module (`go run .` in `/tmp/<name>/` or similar) to confirm the
  actual output, error messages, and compile/no-compile behaviour
  match what the prose says. Past errors in this project (e.g. the
  claim that `len("hello, 世界")` is 16, that `int(3.9)` truncates as
  a literal, or that `bufio.Scanner` has a `Lines()` method) all
  passed a doc-only review and were caught only by running the code.
  Treat runtime verification as a required step, not polish.
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

## Published site (GitHub Pages)

- Source repo: <https://github.com/oduvan/learn-go-from-python>.
- Published URL: <https://oduvan.github.io/learn-go-from-python/>.
- Static-site generator: **MkDocs Material** (`mkdocs-material>=9,<10`),
  configured in `mkdocs.yml`. Theme = `material`, light/dark toggle,
  per-topic navigation tabs, copy-button on code blocks.
- The build runs in `.github/workflows/pages.yml` on every push to
  `master`. It executes `mkdocs build --strict` (so broken intra-doc
  links or missing nav entries fail CI), then deploys via
  `actions/deploy-pages@v4`. The repo's **Settings → Pages → Source**
  must be set to **"GitHub Actions"** for the deploy step to take.
- Local preview: a venv lives at `.venv-mkdocs/` (gitignored). Run
  `.venv-mkdocs/bin/mkdocs serve` for a live-reload preview on
  http://127.0.0.1:8000. To recreate the venv:

  ```bash
  python3 -m venv .venv-mkdocs
  .venv-mkdocs/bin/pip install 'mkdocs-material>=9,<10'
  ```

- The `site/` directory (the build output) and `.venv-mkdocs/` are
  both in `.gitignore`. Don't commit them.
