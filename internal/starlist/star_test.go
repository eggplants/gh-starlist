package starlist

import (
	"strings"
	"testing"
)

func TestStarredCarriesStarredAt(t *testing.T) {
	// starredAt lives on the edge rather than on the node, so it has to be
	// grafted onto the repository.
	client, transport := newRecordingClient(t,
		`{"data":{"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":false},
			"edges":[{"starredAt":"2024-05-06T07:08:09Z","node":{"nameWithOwner":"cli/cli",
				"stargazerCount":38000,"primaryLanguage":{"name":"Go"}}}]}}}}`,
	)

	repos, err := client.Starred("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}

	want := Repo{
		NameWithOwner:  "cli/cli",
		StargazerCount: 38000,
		Language:       "Go",
		StarredAt:      "2024-05-06T07:08:09Z",
	}
	if repos[0] != want {
		t.Errorf("got %+v, want %+v", repos[0], want)
	}
	if !strings.Contains(transport.only(t).Query, "query ViewerStarred") {
		t.Error("no user should be read with the viewer query")
	}
}

func TestStarredReadsTheUserConnection(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{
			"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":false},
				"edges":[{"starredAt":"2024-01-01T00:00:00Z","node":{"nameWithOwner":"mine/repo"}}]}},
			"user":{"starredRepositories":{"pageInfo":{"hasNextPage":false},
				"edges":[{"starredAt":"2024-02-02T00:00:00Z","node":{"nameWithOwner":"theirs/repo"}}]}}}}`,
	)

	repos, err := client.Starred("octocat", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].NameWithOwner != "theirs/repo" {
		t.Fatalf("got %+v, want only the user's starred repositories", repos)
	}

	request := transport.only(t)
	if !strings.Contains(request.Query, "query UserStarred") {
		t.Errorf("a user should be read with the user query, got:\n%s", request.Query)
	}
	if request.Variables["login"] != "octocat" {
		t.Errorf("login = %v, want octocat", request.Variables["login"])
	}
}

func TestStarredFollowsPagination(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
			"edges":[{"node":{"nameWithOwner":"a/one"}}]}}}}`,
		`{"data":{"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":false},
			"edges":[{"node":{"nameWithOwner":"a/two"}}]}}}}`,
	)

	repos, err := client.Starred("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 || repos[0].NameWithOwner != "a/one" || repos[1].NameWithOwner != "a/two" {
		t.Fatalf("pages merged incorrectly: %+v", repos)
	}
	if after := transport.sent[1].Variables["after"]; after != "CUR" {
		t.Errorf("the second page should follow endCursor, got %v", after)
	}
}

func TestStarredStopsAtLimit(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
			"edges":[{"node":{"nameWithOwner":"a/one"}}]}}}}`,
	)

	repos, err := client.Starred("", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if first := transport.only(t).Variables["first"]; first != float64(1) {
		t.Errorf("first = %v, want the limit 1", first)
	}
}
