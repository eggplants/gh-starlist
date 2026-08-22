package starlist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

type stubTransport struct {
	mutex  sync.Mutex
	bodies []string
	byList map[string][]string
	calls  int
	// sent records every request payload, so tests can assert on the query and
	// the variables the client built.
	sent []gqlRequest
}

// gqlRequest is the payload the GraphQL client posts.
type gqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var recorded gqlRequest
	if req.Body != nil {
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &recorded); err != nil {
			return nil, err
		}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sent = append(s.sent, recorded)
	body, err := s.answer(recorded)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

func newTestClient(t *testing.T, bodies ...interface{}) *Client {
	t.Helper()
	client, _ := newRecordingClient(t, bodies...)
	return client
}

// newRecordingClient is newTestClient plus the transport, for tests that assert
// on what was actually sent.
func newRecordingClient(t *testing.T, bodies ...interface{}) (*Client, *stubTransport) {
	t.Helper()
	ordered, byList := split(bodies)
	transport := &stubTransport{bodies: ordered, byList: byList}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test",
		Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{gql: gql}, transport
}

// only returns the single request the transport recorded.
func (s *stubTransport) only(t *testing.T) gqlRequest {
	t.Helper()
	if len(s.sent) != 1 {
		t.Fatalf("got %d requests, want exactly 1", len(s.sent))
	}
	return s.sent[0]
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

func TestPageLimit(t *testing.T) {
	cases := []struct {
		name         string
		limit, count int
		want         int
	}{
		{"no limit asks for a full page", 0, 0, pageSize},
		{"a negative limit means no limit", -1, 250, pageSize},
		{"a limit past one page asks for a full page", 250, 0, pageSize},
		{"the last page asks only for the remainder", 250, 200, 50},
		{"a limit under one page asks only for it", 10, 0, 10},
		{"a reached limit asks for nothing", 100, 100, 0},
		{"an overshot limit asks for nothing", 100, 120, -20},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pageLimit(tc.limit, tc.count); got != tc.want {
				t.Errorf("pageLimit(%d, %d) = %d, want %d", tc.limit, tc.count, got, tc.want)
			}
		})
	}
}

func TestWithScopeHintLeavesOtherErrorsAlone(t *testing.T) {
	plain := errors.New("boom")
	if got := withScopeHint(plain); !errors.Is(got, plain) {
		t.Errorf("a non-GraphQL error should pass through, got %v", got)
	}
	if withScopeHint(nil) != nil {
		t.Error("a nil error should stay nil")
	}

	other := &api.GraphQLError{Errors: []api.GraphQLErrorItem{{Type: "NOT_FOUND", Message: "nope"}}}
	got := withScopeHint(other)
	if strings.Contains(got.Error(), "gh auth refresh") {
		t.Errorf("only INSUFFICIENT_SCOPES should get the scope hint, got %q", got)
	}
}

func TestScopeHintWrapsTheOriginalError(t *testing.T) {
	client := newTestClient(t,
		`{"errors":[{"type":"INSUFFICIENT_SCOPES","message":"needs user scope"}]}`,
	)

	err := client.DeleteList("L_1")
	var gqlErr *api.GraphQLError
	if !errors.As(err, &gqlErr) {
		t.Fatalf("the GraphQL error should stay unwrappable, got %T: %v", err, err)
	}
}

func TestRequestErrorsPropagate(t *testing.T) {
	// A client with no canned bodies makes the transport refuse the request.
	client := newTestClient(t)

	if _, err := client.Lists("", 0); err == nil {
		t.Error("Lists should surface a transport error")
	}
	if _, err := client.Starred("", 0); err == nil {
		t.Error("Starred should surface a transport error")
	}
	if _, err := client.ListRepos("L_1", 0); err == nil {
		t.Error("ListRepos should surface a transport error")
	}
	if _, err := client.LookupRepo("cli", "cli"); err == nil {
		t.Error("LookupRepo should surface a transport error")
	}
	if err := client.Star("R_1"); err == nil {
		t.Error("Star should surface a transport error")
	}
	if err := client.SetListsForRepo("R_1", nil); err == nil {
		t.Error("SetListsForRepo should surface a transport error")
	}
	if _, err := client.CreateList("x", "", false); err == nil {
		t.Error("CreateList should surface a transport error")
	}
	if _, err := client.UpdateList("L_1", nil, nil, nil); err == nil {
		t.Error("UpdateList should surface a transport error")
	}
	if err := client.DeleteList("L_1"); err == nil {
		t.Error("DeleteList should surface a transport error")
	}
}

func TestNewClientWithOptionsKeepsTheCallersHeaders(t *testing.T) {
	// The next-global-ID header is what keeps the API from answering with a
	// deprecation warning, so it has to survive whatever the caller passes.
	options := api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test",
		Headers:   map[string]string{"X-Custom": "yes"},
	}
	if _, err := NewClientWithOptions(options); err != nil {
		t.Fatal(err)
	}

	if _, present := options.Headers["X-Github-Next-Global-ID"]; present {
		t.Error("the caller's header map must not be modified")
	}
	if len(options.Headers) != 1 {
		t.Errorf("the caller's headers changed: %v", options.Headers)
	}
}

// listItems binds a canned response to the star list it answers for. Star
// lists are read in parallel, so a response that belongs to one of them cannot
// be matched by arrival order.
type listItems struct {
	id   string
	body string
}

// answer picks the response for a request: the queue of the star list it asks
// about when the test bound one, the next positional body otherwise. The
// caller holds the mutex.
func (s *stubTransport) answer(request gqlRequest) (string, error) {
	if id, ok := request.Variables["listId"].(string); ok && len(s.byList) > 0 {
		queue := s.byList[id]
		if len(queue) == 0 {
			return "", fmt.Errorf("no canned response left for list %s", id)
		}
		s.byList[id] = queue[1:]
		return queue[0], nil
	}
	if s.calls >= len(s.bodies) {
		return "", errors.New("unexpected extra request")
	}
	body := s.bodies[s.calls]
	s.calls++
	return body, nil
}

// split sorts canned responses into the positional queue and the per-list
// ones, so a test can mix both.
func split(bodies []interface{}) ([]string, map[string][]string) {
	ordered := make([]string, 0, len(bodies))
	byList := map[string][]string{}
	for _, body := range bodies {
		switch body := body.(type) {
		case listItems:
			byList[body.id] = append(byList[body.id], body.body)
		case string:
			ordered = append(ordered, body)
		default:
			panic(fmt.Sprintf("unsupported canned response %T", body))
		}
	}
	return ordered, byList
}
