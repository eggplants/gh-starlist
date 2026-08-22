package starlist

import (
	"fmt"
	"sort"
	"strings"
)

const listFields = `
	id
	name
	slug
	description
	isPrivate
	updatedAt
	items { totalCount }
`

const viewerListsQuery = `
query ViewerLists($first: Int!, $after: String) {
	viewer {
		lists(first: $first, after: $after) {
			totalCount
			pageInfo { hasNextPage endCursor }
			nodes { %s }
		}
	}
}`

const userListsQuery = `
query UserLists($login: String!, $first: Int!, $after: String) {
	user(login: $login) {
		lists(first: $first, after: $after) {
			totalCount
			pageInfo { hasNextPage endCursor }
			nodes { %s }
		}
	}
}`

type listsConnection struct {
	TotalCount int
	PageInfo   pageInfo
	Nodes      []struct {
		ID          string
		Name        string
		Slug        string
		Description string
		IsPrivate   bool
		UpdatedAt   string
		Items       struct{ TotalCount int }
	}
}

type pageInfo struct {
	HasNextPage bool
	EndCursor   string
}

// Lists returns the star lists of user, or of the authenticated user when user
// is empty. Another user's private lists are invisible and simply not returned.
func (c *Client) Lists(user string, limit int) ([]List, error) {
	query := fmt.Sprintf(viewerListsQuery, listFields)
	variables := map[string]interface{}{"after": (*string)(nil)}
	if user != "" {
		query = fmt.Sprintf(userListsQuery, listFields)
		variables["login"] = user
	}

	var lists []List
	for {
		first := pageLimit(limit, len(lists))
		if first <= 0 {
			break
		}
		variables["first"] = first

		var response struct {
			Viewer struct{ Lists listsConnection }
			User   struct{ Lists listsConnection }
		}
		if err := c.do(query, variables, &response); err != nil {
			return nil, err
		}

		conn := response.Viewer.Lists
		if user != "" {
			conn = response.User.Lists
		}
		for _, node := range conn.Nodes {
			lists = append(lists, List{
				ID:          node.ID,
				Name:        node.Name,
				Slug:        node.Slug,
				Description: node.Description,
				IsPrivate:   node.IsPrivate,
				RepoCount:   node.Items.TotalCount,
				UpdatedAt:   node.UpdatedAt,
			})
		}
		c.report(len(lists), conn.TotalCount)
		if !conn.PageInfo.HasNextPage {
			break
		}
		variables["after"] = conn.PageInfo.EndCursor
	}
	return lists, nil
}

// ResolveList finds the list of user named or slugged as key. The slug is
// generated server side, so both spellings have to be accepted.
func (c *Client) ResolveList(user, key string) (List, error) {
	lists, err := c.Lists(user, 0)
	if err != nil {
		return List{}, err
	}
	return FindList(lists, key)
}

// FindList matches key against the name and the slug of the given lists.
func FindList(lists []List, key string) (List, error) {
	var matches []List
	for _, list := range lists {
		if list.Slug == key || list.Name == key {
			return list, nil
		}
		if strings.EqualFold(list.Slug, key) || strings.EqualFold(list.Name, key) {
			matches = append(matches, list)
		}
	}
	switch len(matches) {
	case 0:
		return List{}, fmt.Errorf("no star list matches %q", key)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, 0, len(matches))
		for _, match := range matches {
			names = append(names, match.Slug)
		}
		sort.Strings(names)
		return List{}, fmt.Errorf("%q matches several star lists: %s", key, strings.Join(names, ", "))
	}
}

const listPayloadFields = `id name slug description isPrivate`

// CreateList creates a star list and returns it.
func (c *Client) CreateList(name, description string, private bool) (List, error) {
	query := fmt.Sprintf(`
mutation CreateList($name: String!, $description: String, $isPrivate: Boolean) {
	createUserList(input: {name: $name, description: $description, isPrivate: $isPrivate}) {
		list { %s }
	}
}`, listPayloadFields)

	var response struct {
		CreateUserList struct{ List listNode }
	}
	variables := map[string]interface{}{
		"name":        name,
		"description": description,
		"isPrivate":   private,
	}
	if err := c.do(query, variables, &response); err != nil {
		return List{}, err
	}
	return response.CreateUserList.List.toList(), nil
}

// UpdateList changes the fields of a star list. Nil fields are left untouched:
// the API treats an omitted field as "keep the current value".
func (c *Client) UpdateList(id string, name, description *string, private *bool) (List, error) {
	query := fmt.Sprintf(`
mutation UpdateList($listId: ID!, $name: String, $description: String, $isPrivate: Boolean) {
	updateUserList(input: {listId: $listId, name: $name, description: $description, isPrivate: $isPrivate}) {
		list { %s }
	}
}`, listPayloadFields)

	variables := map[string]interface{}{
		"listId":      id,
		"name":        (*string)(nil),
		"description": (*string)(nil),
		"isPrivate":   (*bool)(nil),
	}
	if name != nil {
		variables["name"] = *name
	}
	if description != nil {
		variables["description"] = *description
	}
	if private != nil {
		variables["isPrivate"] = *private
	}

	var response struct {
		UpdateUserList struct{ List listNode }
	}
	if err := c.do(query, variables, &response); err != nil {
		return List{}, err
	}
	return response.UpdateUserList.List.toList(), nil
}

// DeleteList deletes a star list. The repositories it held stay starred.
func (c *Client) DeleteList(id string) error {
	query := `
mutation DeleteList($listId: ID!) {
	deleteUserList(input: {listId: $listId}) {
		user { login }
	}
}`
	var response struct {
		DeleteUserList struct {
			User struct{ Login string }
		}
	}
	return c.do(query, map[string]interface{}{"listId": id}, &response)
}

type listNode struct {
	ID          string
	Name        string
	Slug        string
	Description string
	IsPrivate   bool
}

func (n listNode) toList() List {
	return List{
		ID:          n.ID,
		Name:        n.Name,
		Slug:        n.Slug,
		Description: n.Description,
		IsPrivate:   n.IsPrivate,
	}
}
