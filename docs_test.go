package cassandra_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// These tests read the repository rather than the package. They hold the
// layout of D3: four documents at the root, every other one in docs/, and
// every one of those named in the table in AGENTS.md.

// rootDocs are the only Markdown files at the root.
var rootDocs = []string{"AGENTS.md", "CLAUDE.md", "CONTRIBUTING.md", "README.md"}

// markdownLink matches a relative link to a Markdown file.
var markdownLink = regexp.MustCompile(`\]\((?:\./)?([^)#:]+\.md)(#[^)]*)?\)`)

// bareMention matches a document named in running text, which is how a Go
// comment or a workflow comment points at one, as in docs/PLAN.md.
var bareMention = regexp.MustCompile(`(?:^|[\s` + "`" + `(])((?:docs/)?[A-Z][A-Z_]*\.md)`)

// decisionRef matches a reference to a decision, such as D17.
var decisionRef = regexp.MustCompile(`\bD([1-9][0-9]?)\b`)

// decisionTitle matches the first line of a decision file.
var decisionTitle = regexp.MustCompile(`^# D(\d+)\. (.+)$`)

// indexRow matches a row of the index in docs/decisions/README.md.
var indexRow = regexp.MustCompile(`(?m)^\| \[D(\d+)\]\(([^)]+)\) \| (.+?) \| (.+) \|$`)

// amendsRef matches the part of a status that names the decisions that it
// amends or replaces, or that amend or replace it.
var amendsRef = regexp.MustCompile(`(Amends|Replaces|Amended by|Replaced by) (D\d+(?:(?:, | and |, and )D\d+)*)`)

// countInProse matches a count that a document writes, such as 10 hard rules.
var countInProse = regexp.MustCompile(`\b(\d+) (hard rules|decisions)\b`)

// otherRepo matches the name of a sibling repository before a decision
// number, as in "dbmeta D62". Such a reference names a decision of that
// repository, not of this one.
var otherRepo = regexp.MustCompile("(?:dbimp|dbmeta|dburl|n1ql|usql)[^\\s]*`?[ (]*$")

func TestTheRootHoldsFourDocuments(t *testing.T) {
	t.Parallel()
	matches, err := filepath.Glob("*.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(matches, rootDocs) {
		t.Errorf("the root holds %v, want %v. Every other document goes in docs/ (D3)", matches, rootDocs)
	}
}

// TestClaudeImportsAgents holds D28. AGENTS.md holds the rules, because Codex
// and the other agents read it, and CLAUDE.md imports it, so that Claude Code
// reads the same rules. A symbolic link does not work, because a Windows
// checkout writes a link as a small text file.
func TestClaudeImportsAgents(t *testing.T) {
	t.Parallel()
	info, err := os.Lstat("CLAUDE.md")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("CLAUDE.md is a symbolic link. Make it a file that holds @AGENTS.md (D28)")
	}
	if got := strings.TrimSpace(read(t, "CLAUDE.md")); got != "@AGENTS.md" {
		t.Errorf("CLAUDE.md holds %q. It holds only @AGENTS.md, and the rules go in AGENTS.md (D28)", got)
	}
}

func TestEveryLinkResolves(t *testing.T) {
	t.Parallel()
	for _, path := range repoFiles(t, ".md", ".go", ".yml") {
		body := read(t, path)
		for _, m := range markdownLink.FindAllStringSubmatch(body, -1) {
			target := filepath.Join(filepath.Dir(path), m[1])
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s: the link to %s does not resolve", path, m[1])
			}
		}
		if strings.HasSuffix(path, ".md") {
			continue
		}
		for _, m := range bareMention.FindAllStringSubmatch(body, -1) {
			if _, err := os.Stat(m[1]); err != nil {
				t.Errorf("%s: names %s, which is not a file", path, m[1])
			}
		}
	}
}

func TestEveryDecisionReferenceExists(t *testing.T) {
	t.Parallel()
	written := decisions(t)
	for _, path := range repoFiles(t, ".md", ".go", ".yml") {
		body := read(t, path)
		for _, m := range decisionRef.FindAllStringSubmatchIndex(body, -1) {
			if otherRepo.MatchString(body[max(0, m[0]-24):m[0]]) {
				continue
			}
			if n := body[m[2]:m[3]]; !hasDecision(written, n) {
				t.Errorf("%s: refers to D%s, which is not in docs/decisions", path, n)
			}
		}
	}
}

func TestTheDecisionIndexIsComplete(t *testing.T) {
	t.Parallel()
	index := read(t, filepath.Join("docs", "decisions", "README.md"))
	indexed := make(map[string]bool)
	for _, m := range indexRow.FindAllStringSubmatch(index, -1) {
		n := m[1]
		indexed[n] = true
		d, ok := decisions(t)[n]
		if !ok {
			t.Errorf("the index holds D%s, which no file defines", n)
			continue
		}
		if m[2] != d.file {
			t.Errorf("D%s: the index links %s, want %s", n, m[2], d.file)
		}
		if m[3] != d.title || m[4] != d.status {
			t.Errorf("D%s: the index row is wrong. Row to use:\n| [D%s](%s) | %s | %s |", n, n, d.file, d.title, d.status)
		}
	}
	for n, d := range decisions(t) {
		if !indexed[n] {
			t.Errorf("D%s is written and is not in docs/decisions/README.md. Row to add:\n| [D%s](%s) | %s | %s |", n, n, d.file, d.title, d.status)
		}
	}
}

// TestAnAmendmentPointsBothWays checks that a decision that amends or
// replaces another is named by that other in its own status. A reader who
// lands on the older decision must see that a later one changed it.
func TestAnAmendmentPointsBothWays(t *testing.T) {
	t.Parallel()
	forward := make(map[string]bool)
	backward := make(map[string]bool)
	for n, d := range decisions(t) {
		for _, m := range amendsRef.FindAllStringSubmatch(d.status, -1) {
			for _, other := range decisionRef.FindAllStringSubmatch(m[2], -1) {
				if m[1] == "Amends" || m[1] == "Replaces" {
					forward[n+">"+other[1]] = true
				} else {
					backward[other[1]+">"+n] = true
				}
			}
		}
	}
	for pair := range forward {
		if !backward[pair] {
			from, to, _ := strings.Cut(pair, ">")
			t.Errorf("D%s amends D%s, and the status of D%s does not say \"Amended by D%s\"", from, to, to, from)
		}
	}
	for pair := range backward {
		if !forward[pair] {
			from, to, _ := strings.Cut(pair, ">")
			t.Errorf("D%s says it is amended by D%s, and the status of D%s does not say \"Amends D%s\"", from, to, to, from)
		}
	}
}

// TestTheCountsInProseAreRight checks the numbers that the documents write
// for the hard rules and for the decisions.
func TestTheCountsInProseAreRight(t *testing.T) {
	t.Parallel()
	_, rest, ok := strings.Cut(read(t, "AGENTS.md"), "\n## Hard rules\n")
	if !ok {
		t.Fatal("AGENTS.md has no Hard rules section")
	}
	section, _, _ := strings.Cut(rest, "\n## ")
	rules := len(regexp.MustCompile(`(?m)^\d+\. `).FindAllString(section, -1))
	want := map[string]int{"hard rules": rules, "decisions": len(decisions(t))}
	for _, path := range repoFiles(t, ".md") {
		for _, m := range countInProse.FindAllStringSubmatch(read(t, path), -1) {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				t.Fatal(err)
			}
			if n != want[m[2]] {
				t.Errorf("%s: says %s %s, and there are %d", path, m[1], m[2], want[m[2]])
			}
		}
	}
}

// TestEveryDocumentIsInBothTables checks that every document in docs/ is named
// in AGENTS.md, for an agent, and in README.md, for a person.
func TestEveryDocumentIsInBothTables(t *testing.T) {
	t.Parallel()
	docs, err := filepath.Glob(filepath.Join("docs", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	docs = append(docs, filepath.Join("docs", "decisions", "README.md"))
	for _, table := range []string{"AGENTS.md", "README.md"} {
		body := read(t, table)
		for _, doc := range docs {
			name := filepath.ToSlash(doc)
			if strings.HasSuffix(name, "decisions/README.md") {
				name = "docs/decisions/"
			}
			if !strings.Contains(body, name) {
				t.Errorf("%s does not name %s, so no reader finds it (D3)", table, name)
			}
		}
	}
}

// TestNoSectionHeadingIsRepeated checks that a level two heading appears once
// in the document that holds it. Two sections with one name make a link to
// either one ambiguous.
func TestNoSectionHeadingIsRepeated(t *testing.T) {
	t.Parallel()
	for _, path := range repoFiles(t, ".md") {
		seen := make(map[string]bool)
		inCode := false
		for line := range strings.Lines(read(t, path)) {
			line = strings.TrimRight(line, "\n")
			if strings.HasPrefix(line, "```") {
				inCode = !inCode
			}
			if inCode || !strings.HasPrefix(line, "## ") {
				continue
			}
			if seen[line] {
				t.Errorf("%s: the heading %q appears twice", path, line)
			}
			seen[line] = true
		}
	}
}

func TestEveryTestNameInTheDocsExists(t *testing.T) {
	t.Parallel()
	written := make(map[string]bool)
	for _, path := range repoFiles(t, ".go") {
		for _, m := range regexp.MustCompile(`(?m)^func ((?:Test|Fuzz|Benchmark)[A-Za-z0-9]+)\(`).FindAllStringSubmatch(read(t, path), -1) {
			written[m[1]] = true
		}
	}
	for _, path := range repoFiles(t, ".md") {
		for _, name := range regexp.MustCompile(`\b(?:Test|Fuzz|Benchmark)[A-Z][A-Za-z0-9]*`).FindAllString(read(t, path), -1) {
			// TestMain is the hook that go test calls, not a test.
			if !written[name] && name != "TestMain" {
				t.Errorf("%s: names %s, which no test defines", path, name)
			}
		}
	}
}

// decision is one file of docs/decisions.
type decision struct {
	file   string
	title  string
	status string
}

// decisions returns every decision in docs/decisions, by number. Each file
// opens with "# D<n>. <title>", a blank line and "Status: <status>.".
func decisions(t *testing.T) map[string]decision {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("docs", "decisions", "D*.md"))
	if err != nil {
		t.Fatal(err)
	}
	written := make(map[string]decision)
	for _, file := range files {
		lines := strings.SplitN(read(t, file), "\n", 4)
		m := decisionTitle.FindStringSubmatch(lines[0])
		if m == nil || len(lines) < 3 || !strings.HasPrefix(lines[2], "Status: ") || !strings.HasSuffix(lines[2], ".") {
			t.Errorf("%s: must open with \"# D<n>. <title>\", a blank line and \"Status: <status>.\"", file)
			continue
		}
		base := filepath.Base(file)
		num, _ := strconv.Atoi(m[1])
		if want := fmt.Sprintf("D%03d-", num); !strings.HasPrefix(base, want) {
			t.Errorf("%s: the file name must start with %s", file, want)
		}
		written[m[1]] = decision{
			file:   base,
			title:  m[2],
			status: strings.TrimSuffix(strings.TrimPrefix(lines[2], "Status: "), "."),
		}
	}
	if len(written) == 0 {
		t.Fatal("docs/decisions holds no decision")
	}
	return written
}

// hasDecision reports whether written holds decision n.
func hasDecision(written map[string]decision, n string) bool {
	_, ok := written[n]
	return ok
}

// repoFiles returns every file with one of exts, outside the folders whose
// name starts with a dot, except .github.
func repoFiles(t *testing.T, exts ...string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && path != "." && strings.HasPrefix(d.Name(), ".") && d.Name() != ".github":
			return filepath.SkipDir
		case !d.IsDir() && slices.Contains(exts, filepath.Ext(path)):
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func read(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
