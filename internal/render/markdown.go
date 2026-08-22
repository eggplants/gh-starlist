package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/eggplants/gh-starlist/internal/starlist"
)

// Placeholder is the marker a template file uses to mark where to generate,
const Placeholder = "<!-- gh-starlist-export -->"

// Section is one rendered heading with the repositories under it.
type Section struct {
	Name  string          `json:"name"`
	Repos []starlist.Repo `json:"repos"`
}

// Markdown renders the sections as a table of contents followed by one table
// per section.
func Markdown(sections []Section) string {
	var builder strings.Builder
	builder.WriteString(toc(sections))
	for _, section := range sections {
		fmt.Fprintf(&builder, "## %s\n\n", section.Name)
		builder.WriteString("| Repository | Description | Stars |\n")
		builder.WriteString("| --- | --- | --- |\n")
		if len(section.Repos) == 0 {
			builder.WriteString("| *No repositories* | | |\n")
		}
		for _, repo := range section.Repos {
			fmt.Fprintf(&builder, "| [%s](%s) | %s | ⭐%d |\n",
				repo.NameWithOwner, repoURL(repo), escapeCell(repo.Description), repo.StargazerCount)
		}
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n") + "\n"
}

// ApplyTemplate substitutes body into the template, or returns body as is when
// the template holds no placeholder.
func ApplyTemplate(template, body string) string {
	if !strings.Contains(template, Placeholder) {
		return body
	}
	return strings.ReplaceAll(template, Placeholder, strings.TrimRight(body, "\n"))
}

func toc(sections []Section) string {
	var builder strings.Builder
	builder.WriteString("## TOC\n\n")
	seen := map[string]int{}
	for _, section := range sections {
		base := Slugify(section.Name)
		slug := base
		if count := seen[base]; count > 0 {
			slug = fmt.Sprintf("%s-%d", base, count)
		}
		seen[base]++
		fmt.Fprintf(&builder, "- [%s](#%s)\n", section.Name, slug)
	}
	builder.WriteString("\n")
	return builder.String()
}

var (
	nonSlugChars = regexp.MustCompile(`[^\p{L}\p{N}\s_-]`)
	whitespace   = regexp.MustCompile(`\s`)
)

// Slugify turns a heading into the anchor GitHub generates for it.
func Slugify(text string) string {
	slug := nonSlugChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(text)), "")
	slug = whitespace.ReplaceAllString(slug, "-")
	if slug == "" {
		return "section"
	}
	return slug
}

func escapeCell(text string) string {
	text = strings.ReplaceAll(text, "|", `\|`)
	return strings.Join(strings.Fields(text), " ")
}

func repoURL(repo starlist.Repo) string {
	if repo.URL != "" {
		return repo.URL
	}
	return "https://github.com/" + repo.NameWithOwner
}
