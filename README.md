# learn-go-from-python

Personal Go-learning notes for a Python developer. Conspects with
runnable examples, targeting Go 1.26.4.

**Read it as a site:** <https://oduvan.github.io/learn-go-from-python/>

The notes live under [`docs/`](docs/). The site is built with
[MkDocs Material](https://squidfunk.github.io/mkdocs-material/) and
deployed by [`.github/workflows/pages.yml`](.github/workflows/pages.yml)
on every push to `master`.

## Local preview

```bash
python3 -m venv .venv-mkdocs
.venv-mkdocs/bin/pip install 'mkdocs-material>=9,<10'
.venv-mkdocs/bin/mkdocs serve
```

Then open http://127.0.0.1:8000.
