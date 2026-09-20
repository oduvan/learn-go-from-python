// check_stdlib_only enforces the book's core/third-party split.
//
// Topics 01-12 teach the language and the standard library. Every external
// dependency — anything that needs a line in go.mod — belongs in the
// third-party topic. This check reads every fenced ```go block in the core
// topics and flags any import path that looks like a module path, i.e. one
// whose first segment contains a dot: github.com/..., gorm.io/...,
// gopkg.in/..., golang.org/x/... Standard-library paths never do.
//
// Documented exceptions live in stdlibOnlyExceptions below, each with a
// reason, so an intentional seam stays visible instead of silently widening.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// thirdPartyTopic is the one topic folder allowed to import external modules.
const thirdPartyTopic = "13-third-party-libraries"

// stdlibOnlyExceptions maps a docs-relative file to why it may name a module.
// Keep this list short, and give every entry a reason.
var stdlibOnlyExceptions = map[string]string{
	"02-language-basics/16-imports.md": "the subject is import paths themselves; every stdlib path is a single dot-free segment, so showing what a third-party path looks like, how import groups are ordered, and what a blank import registers requires naming real modules",
}

// docExampleDomains are reserved for documentation (RFC 2606). A path under
// one of them stands in for the reader's own module rather than a real
// dependency, so it does not count as a third-party import.
var docExampleDomains = []string{"example.com/", "example.org/", "example.net/"}

var (
	goFenceRe  = regexp.MustCompile("(?s)```go\\n(.*?)```")
	importRe   = regexp.MustCompile(`(?m)^\s*(?:_\s+|\.\s+|[\w]+\s+)?"([^"]+)"`)
	importKwRe = regexp.MustCompile(`(?m)^\s*import\s+(?:_\s+|\.\s+|[\w]+\s+)?"([^"]+)"`)
)

// isModulePath reports whether an import path belongs to an external module.
// Every stdlib path is dot-free in its first segment; every module path has a
// hostname there.
func isModulePath(p string) bool {
	for _, d := range docExampleDomains {
		if strings.HasPrefix(p, d) {
			return false
		}
	}
	first, _, _ := strings.Cut(p, "/")
	return strings.Contains(first, ".")
}

func checkStdlibOnly(repoRoot string) int {
	docsDir := filepath.Join(repoRoot, "docs")
	files, err := listEnglishDocs(docsDir)
	if err != nil {
		fail(2, "walking docs/: %v", err)
	}

	violations := 0
	for _, f := range files {
		rel, _ := filepath.Rel(docsDir, f)
		rel = filepath.ToSlash(rel)

		if strings.HasPrefix(rel, thirdPartyTopic+"/") {
			continue
		}
		if _, ok := stdlibOnlyExceptions[rel]; ok {
			continue
		}

		data, err := os.ReadFile(f)
		if err != nil {
			fail(2, "reading %s: %v", rel, err)
		}

		seen := map[string]bool{}
		for _, block := range goFenceRe.FindAllStringSubmatch(string(data), -1) {
			body := block[1]
			for _, re := range []*regexp.Regexp{importRe, importKwRe} {
				for _, m := range re.FindAllStringSubmatch(body, -1) {
					if isModulePath(m[1]) {
						seen[m[1]] = true
					}
				}
			}
		}

		paths := make([]string, 0, len(seen))
		for p := range seen {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			fmt.Fprintf(os.Stderr,
				"docs/%s: imports %q — core topics are standard library only; move this to %s/\n",
				rel, p, thirdPartyTopic)
			violations++
		}
	}

	if violations > 0 {
		fmt.Fprintf(os.Stderr, "\n%d core/third-party split violation(s)\n", violations)
		return violations
	}
	fmt.Println("docs/: core topics import only the standard library ✓")
	return 0
}
