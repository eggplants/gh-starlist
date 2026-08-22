package cmd

import (
	"fmt"
	"os"

	"github.com/eggplants/gh-starlist/internal/starlist"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var (
		description string
		private     bool
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a star list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			client, err := starlist.NewClient()
			if err != nil {
				return err
			}
			list, err := client.CreateList(args[0], description, private)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "created %s star list %q (%s)\n", list.Visibility(), list.Name, list.Slug)
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the list")
	cmd.Flags().BoolVar(&private, "private", false, "Make the list private")
	return cmd
}
