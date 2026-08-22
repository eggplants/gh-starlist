package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	var (
		name        string
		description string
		private     bool
		public      bool
	)

	cmd := &cobra.Command{
		Use:   "edit <list>",
		Short: "Edit the name, description or visibility of a star list",
		Long:  "Edit a star list. Fields left out keep their current value.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			if private && public {
				return errors.New("--private and --public are mutually exclusive")
			}
			if !flags.Changed("name") && !flags.Changed("description") && !private && !public {
				return errors.New("nothing to edit: pass --name, --description, --private or --public")
			}

			client, list, err := resolve("", args[0])
			if err != nil {
				return err
			}

			var (
				newName        *string
				newDescription *string
				newPrivate     *bool
			)
			if flags.Changed("name") {
				newName = &name
			}
			if flags.Changed("description") {
				newDescription = &description
			}
			if private || public {
				newPrivate = &private
			}

			updated, err := client.UpdateList(list.ID, newName, newDescription, newPrivate)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "updated %s star list %q (%s)\n", updated.Visibility(), updated.Name, updated.Slug)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().BoolVar(&private, "private", false, "Make the list private")
	cmd.Flags().BoolVar(&public, "public", false, "Make the list public")
	return cmd
}
