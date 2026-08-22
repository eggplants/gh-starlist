package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/eggplants/gh-starlist/internal/starlist"
	"github.com/spf13/cobra"
)

// NewRoot builds the `gh starlist` command tree.
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "starlist <command>",
		Short: "Work with your GitHub star lists",
		Long: strings.TrimSpace(`
GitHub star list CLI.

Reading someone else's lists works with the default gh scopes; changing your
own needs the 'user' scope:

	gh auth refresh -h github.com -s user
`),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	root.AddCommand(
		newListCmd(),
		newViewCmd(),
		newCreateCmd(),
		newEditCmd(),
		newDeleteCmd(),
		newAddCmd(),
		newRemoveCmd(),
		newExportCmd(),
	)
	return root
}

func printJSON(value interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

// parseRepo accepts OWNER/REPO as well as a repository URL.
func parseRepo(arg string) (owner, name string, err error) {
	repo, err := repository.Parse(arg)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository %q: %w", arg, err)
	}
	if repo.Owner == "" || repo.Name == "" {
		return "", "", fmt.Errorf("invalid repository %q: expected OWNER/REPO", arg)
	}
	return repo.Owner, repo.Name, nil
}

// resolve is the shared "client plus one list" preamble of most commands.
func resolve(user, key string) (*starlist.Client, starlist.List, error) {
	client, err := starlist.NewClient()
	if err != nil {
		return nil, starlist.List{}, err
	}
	list, err := client.ResolveList(user, key)
	if err != nil {
		return nil, starlist.List{}, err
	}
	return client, list, nil
}
