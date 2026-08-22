package starlist

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

type stubTransport struct {
	bodies []string
	calls  int
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if s.calls >= len(s.bodies) {
		return nil, errors.New("unexpected extra request")
	}
	body := s.bodies[s.calls]
	s.calls++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

func newTestClient(t *testing.T, bodies ...string) *Client {
	t.Helper()
	transport := &stubTransport{bodies: bodies}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test",
		Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{gql: gql}
}

func TestListsFollowsPagination(t *testing.T) {
	client := newTestClient(t,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
			"nodes":[{"id":"1","name":"A","slug":"a","isPrivate":false,"items":{"totalCount":3}}]}}}}`,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false,"endCursor":""},
			"nodes":[{"id":"2","name":"B","slug":"b","isPrivate":true,"items":{"totalCount":0}}]}}}}`,
	)

	lists, err := client.Lists("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 2 {
		t.Fatalf("got %d lists, want 2 across both pages", len(lists))
	}
	if lists[0].RepoCount != 3 || lists[1].Slug != "b" || !lists[1].IsPrivate {
		t.Errorf("pages merged incorrectly: %+v", lists)
	}
}

func TestListsStopsAtLimit(t *testing.T) {
	client := newTestClient(t,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
			"nodes":[{"id":"1","name":"A","slug":"a"}]}}}}`,
	)

	lists, err := client.Lists("", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("got %d lists, want 1", len(lists))
	}
}

func TestScopeHint(t *testing.T) {
	client := newTestClient(t,
		`{"errors":[{"type":"INSUFFICIENT_SCOPES","message":"needs user scope"}]}`,
	)

	_, err := client.CreateList("x", "", false)
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "gh auth refresh -h github.com -s user"; !bytes.Contains([]byte(err.Error()), []byte(want)) {
		t.Errorf("error should suggest %q, got %q", want, err)
	}
}
