package cmd

import (
	"fmt"
	"os"

	"github.com/eggplants/gh-starlist/internal/render"
	"github.com/eggplants/gh-starlist/internal/starlist"
	"github.com/spf13/cobra"
)

func newViewCmd() *cobra.Command {
	var (
		user    string
		jsonOut bool
		limit   int
	)

	cmd := &cobra.Command{
		Use:   "view <list>",
		Short: "Show the repositories in a star list",
		Long:  "Show the repositories in a star list. The list is matched by name or by slug.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			client, list, err := resolve(user, args[0])
			if err != nil {
				return err
			}
			repos, err := client.ListRepos(list.ID, limit)
			if err != nil {
				return err
			}
			if jsonOut {
				if repos == nil {
					repos = []starlist.Repo{}
				}
				return printJSON(struct {
					starlist.List
					Repos []starlist.Repo `json:"repos"`
				}{List: list, Repos: repos})
			}
			if len(repos) == 0 {
				fmt.Fprintf(os.Stderr, "%s is empty\n", list.Name)
				return nil
			}

			table := render.NewTable()
			table.AddHeader([]string{"REPOSITORY", "STARS", "LANGUAGE", "DESCRIPTION"})
			for _, repo := range repos {
				table.AddField(repo.NameWithOwner)
				table.AddField(fmt.Sprintf("%d", repo.StargazerCount))
				table.AddField(repo.Language)
				table.AddField(repo.Description)
				table.EndRow()
			}
			return table.Render()
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "GitHub user owning the list (default: yourself)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	cmd.Flags().IntVarP(&limit, "limit", "L", 0, "Maximum number of repositories to fetch (0 for all)")
	return cmd
}
