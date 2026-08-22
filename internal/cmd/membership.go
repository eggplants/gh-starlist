package cmd

import (
	"fmt"
	"os"

	"github.com/eggplants/gh-starlist/internal/starlist"
	"github.com/spf13/cobra"
)

// mover applies one repository's new membership, given the set it belongs to now.
type mover func(membership starlist.Membership, repo starlist.RepoRef, list starlist.List) (listIDs []string, changed bool)

// moveRepos is shared by add and remove. The API only knows how to replace the
// complete set of lists a repository belongs to, so the current set has to be
// read first, which means walking every list the user owns.
func moveRepos(listKey string, args []string, star bool, done string, move mover) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	lists, err := client.Lists("", 0)
	if err != nil {
		return err
	}
	list, err := starlist.FindList(lists, listKey)
	if err != nil {
		return err
	}

	refs := make([]starlist.RepoRef, 0, len(args))
	for _, arg := range args {
		owner, name, err := parseRepo(arg)
		if err != nil {
			return err
		}
		ref, err := client.LookupRepo(owner, name)
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}

	membership, err := client.Membership(lists)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		listIDs, changed := move(membership, ref, list)
		if changed {
			if err := client.SetListsForRepo(ref.ID, listIDs); err != nil {
				return fmt.Errorf("%s: %w", ref.NameWithOwner, err)
			}
			fmt.Fprintf(os.Stderr, "%s: %s %q\n", ref.NameWithOwner, done, list.Name)
		} else {
			fmt.Fprintf(os.Stderr, "%s: no change\n", ref.NameWithOwner)
		}
		if star && !ref.ViewerHasStarred {
			if err := client.Star(ref.ID); err != nil {
				return fmt.Errorf("%s: %w", ref.NameWithOwner, err)
			}
			fmt.Fprintf(os.Stderr, "%s: starred\n", ref.NameWithOwner)
		}
	}
	return nil
}

func newAddCmd() *cobra.Command {
	var noStar bool

	cmd := &cobra.Command{
		Use:   "add <list> <repository>...",
		Short: "Add repositories to a star list",
		Long: `Add repositories to one of your star lists.

Repositories are accepted as OWNER/REPO or as a repository URL. A repository
that is not starred yet gets starred as well, since star lists are meant to
organize your stars; pass --no-star to only add it to the list.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return moveRepos(args[0], args[1:], !noStar, "added to",
				func(membership starlist.Membership, repo starlist.RepoRef, list starlist.List) ([]string, bool) {
					return membership.With(repo.NameWithOwner, list.ID)
				})
		},
	}

	cmd.Flags().BoolVar(&noStar, "no-star", false, "Do not star repositories that are not starred yet")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <list> <repository>...",
		Short: "Remove repositories from a star list",
		Long:  "Remove repositories from one of your star lists. They stay starred.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return moveRepos(args[0], args[1:], false, "removed from",
				func(membership starlist.Membership, repo starlist.RepoRef, list starlist.List) ([]string, bool) {
					return membership.Without(repo.NameWithOwner, list.ID)
				})
		},
	}
	return cmd
}
