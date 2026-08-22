package starlist

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

// pageSize is the maximum page size the GitHub GraphQL API accepts.
const pageSize = 100

// Client talks to the GitHub GraphQL API about star lists.
type Client struct {
	gql *api.GraphQLClient
}

// NewClient builds a client from the gh CLI authentication.
func NewClient() (*Client, error) {
	return NewClientWithOptions(api.ClientOptions{})
}

// NewClientWithOptions builds a client from explicit API options, adding the
// headers every star list query needs. Tests use it to supply a transport.
func NewClientWithOptions(options api.ClientOptions) (*Client, error) {
	headers := make(map[string]string, len(options.Headers)+1)
	for key, value := range options.Headers {
		headers[key] = value
	}
	// Ask for the modern node ID format; the legacy one is deprecated and
	// makes the API answer with a deprecation warning for repository IDs.
	headers["X-Github-Next-Global-ID"] = "1"
	options.Headers = headers

	gql, err := api.NewGraphQLClient(options)
	if err != nil {
		return nil, err
	}
	return &Client{gql: gql}, nil
}

func (c *Client) do(query string, variables map[string]interface{}, response interface{}) error {
	return withScopeHint(c.gql.Do(query, variables, response))
}

// withScopeHint explains how to get the scope every star list mutation needs.
// Reads work with the plain `repo` scope, writes require `user`.
func withScopeHint(err error) error {
	var gqlErr *api.GraphQLError
	if !errors.As(err, &gqlErr) {
		return err
	}
	for _, item := range gqlErr.Errors {
		if item.Type == "INSUFFICIENT_SCOPES" {
			return fmt.Errorf("%w\n\nchanging star lists needs the 'user' scope:\n\tgh auth refresh -h github.com -s user", err)
		}
	}
	return err
}

// pageLimit returns how many items to ask for, honoring an overall limit of
// which count items have already been fetched. It returns 0 when done.
func pageLimit(limit, count int) int {
	if limit <= 0 {
		return pageSize
	}
	if remaining := limit - count; remaining < pageSize {
		return remaining
	}
	return pageSize
}
