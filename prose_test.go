package cassandra_test

import (
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// These tests hold D16. They check the rules of the simple-english skill that
// a machine can check, in every Markdown file and in every Go comment: no
// semicolon, no em dash, no bold text, no contraction and none of the modals
// that the skill replaces with can, will and must. A reader and a skill check
// the rest.

var (
	// contraction matches a short form of two words, which the skill forbids.
	contraction = regexp.MustCompile(`(?i)\b(?:don't|doesn't|didn't|can't|cannot's|won't|isn't|aren't|wasn't|weren't|hasn't|haven't|hadn't|shouldn't|wouldn't|couldn't|it's|that's|there's|here's|what's|let's|we're|you're|they're|i'm|i've|you've|we've|they've|we'll|you'll|it'll|i'll)\b`)
	// modal matches a modal that the skill replaces.
	modal = regexp.MustCompile(`\b(?:should|would|could|might|may)\b`)
	// inlineCode matches a code span, which holds a name and not prose.
	inlineCode = regexp.MustCompile("`[^`\n]*`")
	// quotedText matches text in double quotes, which can hold a quoted error or a phrase that a document discusses.
	quotedText = regexp.MustCompile(`"[^"\n]*"`)
	// linkTarget matches the target of a Markdown link.
	linkTarget = regexp.MustCompile(`\]\([^)]*\)`)
)

func TestProseIsSimpleEnglish(t *testing.T) {
	t.Parallel()
	for _, path := range repoFiles(t, ".md") {
		for _, p := range markdownProse(read(t, path)) {
			at := path + ":" + strconv.Itoa(p.line)
			if strings.Contains(inlineCode.ReplaceAllString(p.text, ""), "**") {
				t.Errorf("%s: has bold text (D16): %s", at, p.text)
			}
			checkProse(t, at, p.text)
		}
	}
	for _, path := range repoFiles(t, ".go") {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, group := range f.Comments {
			for _, c := range group.List {
				text := strings.TrimPrefix(strings.TrimPrefix(c.Text, "//"), "/*")
				// A comment that holds a directive or an indented example is code.
				if strings.HasPrefix(text, "go:") || strings.HasPrefix(text, "\t") || strings.HasPrefix(text, "nolint") {
					continue
				}
				checkProse(t, fset.Position(c.Pos()).String(), strings.TrimSpace(text))
			}
		}
	}
}

// prose is one line of text with its position.
type prose struct {
	line int
	text string
}

// markdownProse returns the lines of body that a person reads: not a fenced
// block, not a table row, not a heading's code, and with code spans, quoted
// text and link targets removed.
func markdownProse(body string) []prose {
	var out []prose
	inCode := false
	for i, line := range strings.Split(body, "\n") {
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "```"):
			inCode = !inCode
		case inCode, strings.HasPrefix(line, "|"), strings.HasPrefix(line, "    "):
		default:
			out = append(out, prose{line: i + 1, text: line})
		}
	}
	return out
}

// checkProse reports each rule that text breaks. at names the place of text.
func checkProse(t *testing.T, at, text string) {
	t.Helper()
	text = linkTarget.ReplaceAllString(text, "]")
	text = inlineCode.ReplaceAllString(text, "")
	text = quotedText.ReplaceAllString(text, "")
	switch {
	case strings.Contains(text, ";"):
		t.Errorf("%s: has a semicolon. Write two sentences (D16): %s", at, text)
	case strings.Contains(text, "—"):
		t.Errorf("%s: has an em dash. Name the relation or write two sentences (D16): %s", at, text)
	}
	if m := contraction.FindString(text); m != "" {
		t.Errorf("%s: has the contraction %q (D16): %s", at, m, text)
	}
	if m := modal.FindString(text); m != "" {
		t.Errorf("%s: has the modal %q. Use can, will or must (D16): %s", at, m, text)
	}
}
