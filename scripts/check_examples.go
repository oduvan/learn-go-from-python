// check_examples keeps examples/ honest against the third-party topic.
//
// Every article in docs/13-third-party-libraries/ that contains Go code
// must be claimed by a test package under examples/, and every test
// package must name a real article. The link is a header comment in the
// test file:
//
//	// Verifies docs/13-third-party-libraries/05-viper.md
//
// This catches the two ways the pair drifts: a new article with no test,
// and a test pointing at an article that was renamed or removed. It
// cannot tell you a test has fallen behind an article's content — only a
// human, or a failing assertion, does that.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const thirdPartyDir = "13-third-party-libraries"

// examplesExemptions lists articles with no runnable Go, and why.
var examplesExemptions = map[string]string{
	"01-choosing-and-managing-dependencies.md": "about go subcommands; its snippets are shell and go.mod fragments",
	"02-golangci-lint-configuration.md":        "verified by examples/lint_test.sh against the examples/lintdemo fixture",
}

var (
	goFence    = regexp.MustCompile("(?m)^```go$")
	verifiesRe = regexp.MustCompile(`//\s*Verifies\s+(docs/` + thirdPartyDir + `/[\w.-]+\.md)`)
)

func checkExamples(repoRoot string) int {
	articlesDir := filepath.Join(repoRoot, "docs", thirdPartyDir)
	examplesDir := filepath.Join(repoRoot, "examples")

	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "examples/ is missing entirely\n")
		return 1
	}

	entries, err := os.ReadDir(articlesDir)
	if err != nil {
		fail(2, "reading %s: %v", articlesDir, err)
	}

	withGo := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".uk.md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(articlesDir, name))
		if err != nil {
			fail(2, "reading %s: %v", name, err)
		}
		if goFence.Match(data) {
			withGo[name] = true
		}
	}

	claimed := map[string][]string{}
	var badRefs []string

	err = filepath.WalkDir(examplesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(repoRoot, path)
		m := verifiesRe.FindSubmatch(data)
		if m == nil {
			return nil // helper files need no reference
		}
		ref := string(m[1])
		if _, statErr := os.Stat(filepath.Join(repoRoot, ref)); statErr != nil {
			badRefs = append(badRefs, fmt.Sprintf("%s references %s, which does not exist", rel, ref))
			return nil
		}
		claimed[filepath.Base(ref)] = append(claimed[filepath.Base(ref)], rel)
		return nil
	})
	if err != nil {
		fail(2, "walking examples/: %v", err)
	}

	problems := 0
	for _, msg := range badRefs {
		fmt.Fprintf(os.Stderr, "examples: %s\n", msg)
		problems++
	}

	var missing []string
	for name := range withGo {
		if _, ok := examplesExemptions[name]; ok {
			continue
		}
		if len(claimed[name]) == 0 {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	for _, name := range missing {
		fmt.Fprintf(os.Stderr,
			"examples: docs/%s/%s has Go code but no test claims it.\n"+
				"          Add a package under examples/ whose test file starts with\n"+
				"          // Verifies docs/%s/%s\n"+
				"          or add it to examplesExemptions in scripts/check_examples.go with a reason.\n",
			thirdPartyDir, name, thirdPartyDir, name)
		problems++
	}

	for name := range examplesExemptions {
		if !withGo[name] {
			if _, err := os.Stat(filepath.Join(articlesDir, name)); err != nil {
				fmt.Fprintf(os.Stderr,
					"examples: stale exemption for %q in scripts/check_examples.go — the article is gone\n", name)
				problems++
			}
		}
	}

	if problems > 0 {
		fmt.Fprintf(os.Stderr, "\n%d examples/ coverage problem(s)\n", problems)
		return problems
	}
	fmt.Printf("examples/: every third-party article with Go code has a test ✓\n")
	return 0
}
