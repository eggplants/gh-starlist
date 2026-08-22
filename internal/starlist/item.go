package starlist

import (
	"fmt"
	"strings"
)

const repoFields = `
	nameWithOwner
	description
	url
	stargazerCount
	isArchived
	primaryLanguage { name }
`

type repoNode struct {
	NameWithOwner   string
	Description     string
	URL             string
	StargazerCount  int
	IsArchived      bool
	PrimaryLanguage struct{ Name string }
}

func (n repoNode) toRepo() Repo {
	return Repo{
		NameWithOwner:  n.NameWithOwner,
		Description:    n.Description,
		URL:            n.URL,
		StargazerCount: n.StargazerCount,
		Language:       n.PrimaryLanguage.Name,
		IsArchived:     n.IsArchived,
	}
}

// ListRepos returns the repositories held by a star list.
func (c *Client) ListRepos(listID string, limit int) ([]Repo, error) {
	query := fmt.Sprintf(`
query ListItems($listId: ID!, $first: Int!, $after: String) {
	node(id: $listId) {
		... on UserList {
			items(first: $first, after: $after) {
				totalCount
				pageInfo { hasNextPage endCursor }
				nodes { ... on Repository { %s } }
			}
		}
	}
}`, repoFields)

	variables := map[string]interface{}{"listId": listID, "after": (*string)(nil)}
	var repos []Repo
	for {
		first := pageLimit(limit, len(repos))
		if first <= 0 {
			break
		}
		variables["first"] = first

		var response struct {
			Node struct {
				Items struct {
					TotalCount int
					PageInfo   pageInfo
					Nodes      []repoNode
				}
			}
		}
		if err := c.do(query, variables, &response); err != nil {
			return nil, err
		}
		for _, node := range response.Node.Items.Nodes {
			repos = append(repos, node.toRepo())
		}
		c.report(len(repos), response.Node.Items.TotalCount)
		if !response.Node.Items.PageInfo.HasNextPage {
			break
		}
		variables["after"] = response.Node.Items.PageInfo.EndCursor
	}
	return repos, nil
}

// RepoRef is the handle needed to move a repository between star lists.
type RepoRef struct {
	ID               string
	NameWithOwner    string
	ViewerHasStarred bool
}

// LookupRepo resolves owner/name to its node ID and star state.
func (c *Client) LookupRepo(owner, name string) (RepoRef, error) {
	query := `
query RepoRef($owner: String!, $name: String!) {
	repository(owner: $owner, name: $name) {
		id
		nameWithOwner
		viewerHasStarred
	}
}`
	var response struct {
		Repository struct {
			ID               string
			NameWithOwner    string
			ViewerHasStarred bool
		}
	}
	variables := map[string]interface{}{"owner": owner, "name": name}
	if err := c.do(query, variables, &response); err != nil {
		return RepoRef{}, err
	}
	if response.Repository.ID == "" {
		return RepoRef{}, fmt.Errorf("repository %s/%s not found", owner, name)
	}
	return RepoRef(response.Repository), nil
}

// SetListsForRepo replaces the whole set of star lists a repository belongs to.
// The API has no add or remove: callers must pass the complete set they want.
func (c *Client) SetListsForRepo(repoID string, listIDs []string) error {
	query := `
mutation SetLists($itemId: ID!, $listIds: [ID!]!) {
	updateUserListsForItem(input: {itemId: $itemId, listIds: $listIds}) {
		lists { id }
	}
}`
	if listIDs == nil {
		listIDs = []string{}
	}
	var response struct {
		UpdateUserListsForItem struct {
			Lists []struct{ ID string }
		}
	}
	variables := map[string]interface{}{"itemId": repoID, "listIds": listIDs}
	return c.do(query, variables, &response)
}

// Star stars a repository. Adding a repository to a list does not star it, so
// this keeps the star page and the lists consistent.
func (c *Client) Star(repoID string) error {
	query := `
mutation Star($starrableId: ID!) {
	addStar(input: {starrableId: $starrableId}) {
		starrable { id }
	}
}`
	var response struct {
		AddStar struct {
			Starrable struct{ ID string }
		}
	}
	return c.do(query, map[string]interface{}{"starrableId": repoID}, &response)
}

// Membership maps a repository name to the IDs of the lists holding it.
// The API exposes no reverse lookup on Repository, so every list is walked.
type Membership map[string][]string

// Membership builds the reverse index of the given lists.
func (c *Client) Membership(lists []List) (Membership, error) {
	repos, err := c.ListReposAll(lists, DefaultWorkers, nil)
	if err != nil {
		return nil, err
	}
	membership := Membership{}
	for index, list := range lists {
		for _, repo := range repos[index] {
			key := strings.ToLower(repo.NameWithOwner)
			membership[key] = append(membership[key], list.ID)
		}
	}
	return membership, nil
}

// ListIDs returns the lists holding nameWithOwner.
func (m Membership) ListIDs(nameWithOwner string) []string {
	return m[strings.ToLower(nameWithOwner)]
}

// With returns the list IDs of nameWithOwner plus listID, unchanged when it is
// already there.
func (m Membership) With(nameWithOwner, listID string) ([]string, bool) {
	current := m.ListIDs(nameWithOwner)
	for _, id := range current {
		if id == listID {
			return current, false
		}
	}
	return append(append([]string{}, current...), listID), true
}

// Without returns the list IDs of nameWithOwner minus listID, unchanged when it
// is not there.
func (m Membership) Without(nameWithOwner, listID string) ([]string, bool) {
	current := m.ListIDs(nameWithOwner)
	remaining := make([]string, 0, len(current))
	for _, id := range current {
		if id != listID {
			remaining = append(remaining, id)
		}
	}
	if len(remaining) == len(current) {
		return current, false
	}
	return remaining, true
}
