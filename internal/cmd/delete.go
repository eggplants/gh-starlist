package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <list>",
		Aliases: []string{"rm"},
		Short:   "Delete a star list",
		Long:    "Delete a star list. The repositories it holds stay starred.",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			client, list, err := resolve("", args[0])
			if err != nil {
				return err
			}
			if !yes {
				confirmed, err := confirm(fmt.Sprintf("Delete star list %q with %d repositories?", list.Name, list.RepoCount))
				if err != nil {
					return err
				}
				if !confirmed {
					return errors.New("cancelled")
				}
			}
			if err := client.DeleteList(list.ID); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "deleted star list %q\n", list.Name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

func confirm(prompt string) (bool, error) {
	terminal := term.FromEnv()
	if !terminal.IsTerminalOutput() {
		return false, errors.New("cannot prompt for confirmation, pass --yes")
	}
	fmt.Fprintf(terminal.ErrOut(), "%s (y/N) ", prompt)
	answer, err := bufio.NewReader(terminal.In()).ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
