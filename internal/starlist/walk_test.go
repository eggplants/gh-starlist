package starlist

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

func threeLists() []List {
	return []List{
		{ID: "L_1", Slug: "one", RepoCount: 2},
		{ID: "L_2", Slug: "two", RepoCount: 1},
		{ID: "L_3", Slug: "three", RepoCount: 1},
	}
}

func itemsBody(names ...string) string {
	nodes := make([]string, 0, len(names))
	for _, name := range names {
		nodes = append(nodes, `{"nameWithOwner":"`+name+`"}`)
	}
	return fmt.Sprintf(`{"data":{"node":{"items":{"totalCount":%d,"pageInfo":{"hasNextPage":false},"nodes":[%s]}}}}`,
		len(names), strings.Join(nodes, ","))
}

func TestListReposAllKeepsTheOrderOfTheLists(t *testing.T) {
	// The answers come back in whatever order the workers finish, so the
	// result has to be put back where its list is.
	client := newTestClient(t,
		listItems{"L_1", itemsBody("cli/cli", "BurntSushi/ripgrep")},
		listItems{"L_2", itemsBody("psf/requests")},
		listItems{"L_3", itemsBody("sharkdp/fd")},
	)

	repos, err := client.ListReposAll(threeLists(), 3, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 3 {
		t.Fatalf("got %d results, want one per list", len(repos))
	}
	for index, want := range []string{"cli/cli", "psf/requests", "sharkdp/fd"} {
		if len(repos[index]) == 0 || repos[index][0].NameWithOwner != want {
			t.Errorf("list %d = %v, want it to start with %s", index, repos[index], want)
		}
	}
}

func TestListReposAllReportsTheRunningTotal(t *testing.T) {
	client := newTestClient(t,
		listItems{"L_1", itemsBody("cli/cli", "BurntSushi/ripgrep")},
		listItems{"L_2", itemsBody("psf/requests")},
		listItems{"L_3", itemsBody("sharkdp/fd")},
	)

	var reports [][2]int
	if _, err := client.ListReposAll(threeLists(), 3, func(fetched, total int) {
		reports = append(reports, [2]int{fetched, total})
	}); err != nil {
		t.Fatal(err)
	}

	if len(reports) != 3 {
		t.Fatalf("got %d reports, want one per list page", len(reports))
	}
	// The hook is serialized, so the sum only ever grows, and the last one
	// accounts for every repository the lists said they held.
	previous := 0
	for _, report := range reports {
		if report[0] < previous {
			t.Errorf("the count went backwards: %v", reports)
		}
		previous = report[0]
		if report[1] != 4 {
			t.Errorf("total = %d, want the 4 repositories the lists hold", report[1])
		}
	}
	if previous != 4 {
		t.Errorf("the last report was %d, want 4", previous)
	}
}

func TestListReposAllFailsOnTheFirstListInOrder(t *testing.T) {
	// Both lists fail; which one lost the race must not change the message.
	client := newTestClient(t,
		listItems{"L_1", `{"errors":[{"type":"NOT_FOUND","message":"gone"}]}`},
		listItems{"L_2", `{"errors":[{"type":"NOT_FOUND","message":"also gone"}]}`},
		listItems{"L_3", itemsBody("sharkdp/fd")},
	)

	_, err := client.ListReposAll(threeLists(), 3, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := `reading list "one"`; !strings.Contains(err.Error(), want) {
		t.Errorf("the error should say %q, got %q", want, err)
	}
}

func TestListReposAllOfNoLists(t *testing.T) {
	repos, err := newTestClient(t).ListReposAll(nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Errorf("got %v, want nothing", repos)
	}
}

// gateTransport holds every request until they have all arrived, so a walk
// that read the lists one after another would never finish.
type gateTransport struct {
	arrived chan struct{}
	release chan struct{}
}

func (g *gateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	g.arrived <- struct{}{}
	<-g.release
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(itemsBody("cli/cli"))),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

func TestListReposAllReadsTheListsAtTheSameTime(t *testing.T) {
	gate := &gateTransport{arrived: make(chan struct{}, 3), release: make(chan struct{})}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test",
		Transport: gate,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{gql: gql}

	done := make(chan error, 1)
	go func() {
		_, err := client.ListReposAll(threeLists(), 3, nil)
		done <- err
	}()

	for count := 0; count < 3; count++ {
		select {
		case <-gate.arrived:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of 3 lists were being read at once", count)
		}
	}
	close(gate.release)

	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestListReposAllHonorsTheWorkerLimit(t *testing.T) {
	// With one worker the second request cannot start before the first is
	// answered, so exactly one is in flight while the gate is shut.
	gate := &gateTransport{arrived: make(chan struct{}, 3), release: make(chan struct{})}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test",
		Transport: gate,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{gql: gql}

	done := make(chan error, 1)
	go func() {
		_, err := client.ListReposAll(threeLists(), 1, nil)
		done <- err
	}()

	<-gate.arrived
	select {
	case <-gate.arrived:
		t.Error("a second list was read while the worker was still busy")
	case <-time.After(100 * time.Millisecond):
	}
	close(gate.release)

	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
