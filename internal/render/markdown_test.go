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
