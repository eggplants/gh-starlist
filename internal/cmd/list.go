package cmd

import (
	"fmt"
	"os"

	"github.com/eggplants/gh-starlist/internal/render"
	"github.com/eggplants/gh-starlist/internal/starlist"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		user    string
		jsonOut bool
		limit   int
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List star lists",
		Long:    "List star lists. Another user's private lists are never visible.",
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			lists, err := client.Lists(user, limit)
			if err != nil {
				return err
			}
			if jsonOut {
				if lists == nil {
					lists = []starlist.List{}
				}
				return printJSON(lists)
			}
			if len(lists) == 0 {
				fmt.Fprintln(os.Stderr, "no star lists found")
				return nil
			}

			table := render.NewTable()
			table.AddHeader([]string{"NAME", "SLUG", "REPOS", "VISIBILITY", "DESCRIPTION"})
			for _, list := range lists {
				table.AddField(list.Name)
				table.AddField(list.Slug)
				table.AddField(fmt.Sprintf("%d", list.RepoCount))
				table.AddField(list.Visibility())
				table.AddField(list.Description)
				table.EndRow()
			}
			return table.Render()
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "GitHub user to read lists from (default: yourself)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	cmd.Flags().IntVarP(&limit, "limit", "L", 0, "Maximum number of lists to fetch (0 for all)")
	return cmd
}
