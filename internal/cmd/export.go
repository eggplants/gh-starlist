package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/eggplants/gh-starlist/internal/render"
	"github.com/eggplants/gh-starlist/internal/starlist"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var (
		user            string
		format          string
		sortBy          string
		templatePath    string
		outputPath      string
		uncategorized   string
		skipUncategoriz bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all star lists as Markdown or JSON",
		Long: `Export every star list, plus the starred repositories that are in no list,
as a Markdown document with a table of contents or as JSON.

With --template, the generated Markdown replaces the <!-- gh-starlist-export --> placeholder in the template file.`,
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if format != "md" && format != "json" {
				return fmt.Errorf("unknown format %q: expected md or json", format)
			}
			sorter, err := repoSorter(sortBy)
			if err != nil {
				return err
			}

			client, err := starlist.NewClient()
			if err != nil {
				return err
			}
			lists, err := client.Lists(user, 0)
			if err != nil {
				return err
			}

			listed := map[string]bool{}
			sections := make([]render.Section, 0, len(lists)+1)
			for _, list := range lists {
				repos, err := client.ListRepos(list.ID, 0)
				if err != nil {
					return fmt.Errorf("reading list %q: %w", list.Slug, err)
				}
				for _, repo := range repos {
					listed[strings.ToLower(repo.NameWithOwner)] = true
				}
				sections = append(sections, render.Section{Name: list.Name, Repos: repos})
			}

			if !skipUncategoriz {
				starred, err := client.Starred(user, 0)
				if err != nil {
					return err
				}
				rest := make([]starlist.Repo, 0, len(starred))
				for _, repo := range starred {
					if !listed[strings.ToLower(repo.NameWithOwner)] {
						rest = append(rest, repo)
					}
				}
				sections = append(sections, render.Section{Name: uncategorized, Repos: rest})
			}

			for i := range sections {
				sorter(sections[i].Repos)
			}

			if format == "json" {
				if outputPath != "" {
					return fmt.Errorf("--output is only supported with --format md")
				}
				return printJSON(sections)
			}

			body := render.Markdown(sections)
			if templatePath != "" {
				template, err := os.ReadFile(templatePath)
				if err != nil {
					return err
				}
				body = render.ApplyTemplate(string(template), body)
			}
			if outputPath == "" {
				_, err := os.Stdout.WriteString(body)
				return err
			}
			return os.WriteFile(outputPath, []byte(body), 0o644)
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "GitHub user to export (default: yourself)")
	cmd.Flags().StringVarP(&format, "format", "f", "md", "Output format: md or json")
	cmd.Flags().StringVar(&sortBy, "sort", "stars", "Sort repositories by: stars, starred-at or name")
	cmd.Flags().StringVar(&templatePath, "template", "", "Markdown template holding a <!-- gh-starlist-export --> placeholder")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Write to this file instead of stdout")
	cmd.Flags().StringVar(&uncategorized, "uncategorized-title", "Uncategorized", "Heading for starred repositories in no list")
	cmd.Flags().BoolVar(&skipUncategoriz, "no-uncategorized", false, "Omit the section of starred repositories in no list")
	return cmd
}

func repoSorter(sortBy string) (func([]starlist.Repo), error) {
	switch sortBy {
	case "stars":
		return func(repos []starlist.Repo) {
			sort.SliceStable(repos, func(i, j int) bool {
				return repos[i].StargazerCount > repos[j].StargazerCount
			})
		}, nil
	case "starred-at":
		// Star lists do not expose when a repository was starred, so entries
		// without that information keep their original order at the end.
		return func(repos []starlist.Repo) {
			sort.SliceStable(repos, func(i, j int) bool {
				return repos[i].StarredAt > repos[j].StarredAt
			})
		}, nil
	case "name":
		return func(repos []starlist.Repo) {
			sort.SliceStable(repos, func(i, j int) bool {
				return strings.ToLower(repos[i].NameWithOwner) < strings.ToLower(repos[j].NameWithOwner)
			})
		}, nil
	default:
		return nil, fmt.Errorf("unknown sort %q: expected stars, starred-at or name", sortBy)
	}
}
