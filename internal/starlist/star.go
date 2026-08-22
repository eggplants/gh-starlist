package starlist

import "fmt"

const viewerStarredQuery = `
query ViewerStarred($first: Int!, $after: String) {
	viewer {
		starredRepositories(first: $first, after: $after, orderBy: {field: STARRED_AT, direction: DESC}) {
			pageInfo { hasNextPage endCursor }
			edges { starredAt node { %s } }
		}
	}
}`

const userStarredQuery = `
query UserStarred($login: String!, $first: Int!, $after: String) {
	user(login: $login) {
		starredRepositories(first: $first, after: $after, orderBy: {field: STARRED_AT, direction: DESC}) {
			pageInfo { hasNextPage endCursor }
			edges { starredAt node { %s } }
		}
	}
}`

type starredConnection struct {
	PageInfo pageInfo
	Edges    []struct {
		StarredAt string
		Node      repoNode
	}
}

// Starred returns the repositories starred by user, newest first, or those of
// the authenticated user when user is empty.
func (c *Client) Starred(user string, limit int) ([]Repo, error) {
	query := fmt.Sprintf(viewerStarredQuery, repoFields)
	variables := map[string]interface{}{"after": (*string)(nil)}
	if user != "" {
		query = fmt.Sprintf(userStarredQuery, repoFields)
		variables["login"] = user
	}

	var repos []Repo
	for {
		first := pageLimit(limit, len(repos))
		if first <= 0 {
			break
		}
		variables["first"] = first

		var response struct {
			Viewer struct{ StarredRepositories starredConnection }
			User   struct{ StarredRepositories starredConnection }
		}
		if err := c.do(query, variables, &response); err != nil {
			return nil, err
		}

		conn := response.Viewer.StarredRepositories
		if user != "" {
			conn = response.User.StarredRepositories
		}
		for _, edge := range conn.Edges {
			repo := edge.Node.toRepo()
			repo.StarredAt = edge.StarredAt
			repos = append(repos, repo)
		}
		if !conn.PageInfo.HasNextPage {
			break
		}
		variables["after"] = conn.PageInfo.EndCursor
	}
	return repos, nil
}
