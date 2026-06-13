// check_concepts is a forward-reference linter for the conspect curriculum.
//
// It reads docs/.concepts.yml, walks every English `.md` file under docs/ in
// reading order (directory then numeric-prefix), and flags any occurrence of
// a tracked concept that appears BEFORE its introducing article — unless that
// occurrence is in the concept's `ok_in` allow-list (for acknowledged
// "covered later" forward references).
//
// Exit codes:
//
//	0 — clean
//	1 — one or more violations found
//	2 — configuration error (bad regex, missing introducing file, etc.)
//
// Run from the repo root: `go run scripts/check_concepts.go`.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Concept struct {
	Pattern      string   `yaml:"pattern"`
	IntroducesIn string   `yaml:"introduces_in"`
	OkIn         []string `yaml:"ok_in"`

	re *regexp.Regexp
}

type Config struct {
	Concepts []Concept `yaml:"concepts"`
}

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fail(2, "%v", err)
	}

	cfgPath := filepath.Join(repoRoot, "docs", ".concepts.yml")
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		fail(2, "loading %s: %v", cfgPath, err)
	}

	for i := range cfg.Concepts {
		c := &cfg.Concepts[i]
		re, err := regexp.Compile(c.Pattern)
		if err != nil {
			fail(2, "concept pattern %q: %v", c.Pattern, err)
		}
		c.re = re
	}

	files, err := listEnglishDocs(filepath.Join(repoRoot, "docs"))
	if err != nil {
		fail(2, "walking docs/: %v", err)
	}

	violations := 0
	for _, c := range cfg.Concepts {
		introIdx := len(files) // "TBD" → check every file
		if !strings.EqualFold(c.IntroducesIn, "TBD") {
			target := filepath.Join("docs", filepath.FromSlash(c.IntroducesIn))
			found := false
			for i, f := range files {
				rel, _ := filepath.Rel(repoRoot, f)
				if rel == target {
					introIdx = i
					found = true
					break
				}
			}
			if !found {
				fail(2, "concept pattern %q: introduces_in %q does not match any doc",
					c.Pattern, c.IntroducesIn)
			}
		}

		okSet := make(map[string]struct{}, len(c.OkIn))
		for _, p := range c.OkIn {
			okSet[filepath.Join("docs", filepath.FromSlash(p))] = struct{}{}
		}

		for i := 0; i < introIdx; i++ {
			rel, _ := filepath.Rel(repoRoot, files[i])
			if _, ok := okSet[rel]; ok {
				continue
			}
			data, err := os.ReadFile(files[i])
			if err != nil {
				fail(2, "reading %s: %v", rel, err)
			}
			matches := c.re.FindAllStringIndex(string(data), -1)
			if len(matches) == 0 {
				continue
			}
			for _, m := range matches {
				line, col := lineCol(data, m[0])
				where := c.IntroducesIn
				if where == "" || strings.EqualFold(where, "TBD") {
					where = "(not yet introduced anywhere)"
				} else {
					where = "before " + where
				}
				fmt.Fprintf(os.Stderr, "%s:%d:%d: %q used %s\n",
					rel, line, col, c.re.FindString(string(data[m[0]:m[1]])), where)
				violations++
			}
		}
	}

	if violations > 0 {
		fmt.Fprintf(os.Stderr, "\n%d concept violation(s)\n", violations)
		os.Exit(1)
	}
	fmt.Println("docs/.concepts.yml: all tracked concepts are introduced before use ✓")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// listEnglishDocs returns all *.md files under docsDir (excluding *.uk.md),
// sorted in reading order: directory path lexicographically, then filename's
// numeric prefix before the dash.
func listEnglishDocs(docsDir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(docsDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(p)
		if !strings.HasSuffix(base, ".md") {
			return nil
		}
		if strings.HasSuffix(base, ".uk.md") {
			return nil
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Reading order: index.md first, then by directory + numeric prefix.
	sort.Slice(out, func(i, j int) bool {
		return readingOrderKey(out[i]) < readingOrderKey(out[j])
	})
	return out, nil
}

// readingOrderKey produces a sortable string that lines files up by directory
// then numeric NN- prefix, with index.md sorting first within any directory.
func readingOrderKey(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	// index.md first
	if base == "index.md" {
		return dir + "/00-index"
	}
	// Pad the leading NN- so "9" sorts before "10"
	parts := strings.SplitN(base, "-", 2)
	if len(parts) == 2 && isAllDigits(parts[0]) {
		return dir + "/" + fmt.Sprintf("%04s-%s", parts[0], parts[1])
	}
	return dir + "/" + base
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func lineCol(data []byte, off int) (int, int) {
	line, col := 1, 1
	for i := 0; i < off && i < len(data); i++ {
		if data[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git found above %s", cwd)
		}
		dir = parent
	}
}

func fail(code int, format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(code)
}
