package render

import (
	"strings"
	"testing"

	"github.com/eggplants/gh-starlist/internal/starlist"
)

func TestMarkdownTOCDisambiguatesRepeatedHeadings(t *testing.T) {
	out := Markdown([]Section{
		{Name: "CLI/TUI"},
		{Name: "cli tui"},
		{Name: "CLI/TUI"},
	})

	for _, want := range []string{"(#clitui)", "(#cli-tui)", "(#clitui-1)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing anchor %s in:\n%s", want, out)
		}
	}
}

func TestMarkdownRow(t *testing.T) {
	out := Markdown([]Section{{
		Name: "Tools",
		Repos: []starlist.Repo{{
			NameWithOwner:  "cli/cli",
			Description:    "a | b\nc",
			URL:            "https://github.com/cli/cli",
			StargazerCount: 42,
		}},
	}})

	want := `| [cli/cli](https://github.com/cli/cli) | a \| b c | ⭐42 |`
	if !strings.Contains(out, want) {
		t.Errorf("want row %q in:\n%s", want, out)
	}
}

func TestMarkdownEmptySection(t *testing.T) {
	out := Markdown([]Section{{Name: "Empty"}})
	if !strings.Contains(out, "| *No repositories* | | |") {
		t.Errorf("empty section not rendered:\n%s", out)
	}
}

func TestApplyTemplate(t *testing.T) {
	body := "## TOC\n\n- [a](#a)\n"

	got := ApplyTemplate("# Title\n\n"+Placeholder+"\n\nfooter\n", body)
	want := "# Title\n\n## TOC\n\n- [a](#a)\n\nfooter\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got := ApplyTemplate("no placeholder", body); got != body {
		t.Errorf("template without placeholder should yield the body, got %q", got)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"CLI/TUI":      "clitui",
		"Python Utils": "python-utils",
		"日本語 メモ":       "日本語-メモ",
		"!!!":          "section",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMarkdownFallsBackToTheCanonicalURL(t *testing.T) {
	// A repository read from a star list may arrive without a url.
	out := Markdown([]Section{{
		Name:  "Tools",
		Repos: []starlist.Repo{{NameWithOwner: "cli/cli"}},
	}})

	if want := "[cli/cli](https://github.com/cli/cli)"; !strings.Contains(out, want) {
		t.Errorf("want %q in:\n%s", want, out)
	}
}

func TestMarkdownStructure(t *testing.T) {
	out := Markdown([]Section{
		{Name: "Tools", Repos: []starlist.Repo{{NameWithOwner: "a/b", StargazerCount: 1}}},
		{Name: "Uncategorized"},
	})

	want := strings.Join([]string{
		"## TOC",
		"",
		"- [Tools](#tools)",
		"- [Uncategorized](#uncategorized)",
		"",
		"## Tools",
		"",
		"| Repository | Description | Stars |",
		"| --- | --- | --- |",
		"| [a/b](https://github.com/a/b) |  | ⭐1 |",
		"",
		"## Uncategorized",
		"",
		"| Repository | Description | Stars |",
		"| --- | --- | --- |",
		"| *No repositories* | | |",
		"",
	}, "\n")
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestMarkdownWithNoSections(t *testing.T) {
	// An account with no lists still has to produce a valid document.
	if got, want := Markdown(nil), "## TOC\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarkdownEndsWithExactlyOneNewline(t *testing.T) {
	out := Markdown([]Section{{Name: "Tools"}})
	if !strings.HasSuffix(out, "|\n") {
		t.Errorf("output should end with a single newline, got %q", out)
	}
}

func TestApplyTemplateReplacesEveryPlaceholder(t *testing.T) {
	got := ApplyTemplate(Placeholder+"\n---\n"+Placeholder, "body\n")
	if want := "body\n---\nbody"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApplyTemplateKeepsTheTemplateNewlines(t *testing.T) {
	// The body is trimmed so the template alone decides the surrounding blank
	// lines; whatever follows the placeholder must survive.
	got := ApplyTemplate("head\n\n"+Placeholder+"\n", "b\n\n\n")
	if want := "head\n\nb\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSlugifyKeepsUnderscoresAndHyphens(t *testing.T) {
	cases := map[string]string{
		"  Padded  ":     "padded",
		"snake_case-x":   "snake_case-x",
		"C++ / Rust":     "c--rust",
		"Two  spaces":    "two--spaces",
		"tabs\tand\nnew": "tabs-and-new",
		"":               "section",
		"   ":            "section",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMarkdownEscapesPipesInDescriptions(t *testing.T) {
	out := Markdown([]Section{{
		Name: "Tools",
		Repos: []starlist.Repo{{
			NameWithOwner: "a/b",
			Description:   "  leading | and\ttabs\nand newlines  ",
		}},
	}})

	if want := `| leading \| and tabs and newlines |`; !strings.Contains(out, want) {
		t.Errorf("want %q in:\n%s", want, out)
	}
}
