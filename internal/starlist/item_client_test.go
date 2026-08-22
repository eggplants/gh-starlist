package starlist

import (
	"reflect"
	"strings"
	"testing"
)

func TestListReposMapsTheNode(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"nameWithOwner":"dandavison/delta","description":"A pager","url":"https://github.com/dandavison/delta",
				"stargazerCount":31830,"isArchived":true,"primaryLanguage":{"name":"Rust"}}]}}}}`,
	)

	repos, err := client.ListRepos("L_1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}

	want := Repo{
		NameWithOwner:  "dandavison/delta",
		Description:    "A pager",
		URL:            "https://github.com/dandavison/delta",
		StargazerCount: 31830,
		Language:       "Rust",
		IsArchived:     true,
	}
	if repos[0] != want {
		t.Errorf("got %+v, want %+v", repos[0], want)
	}
	if id := transport.only(t).Variables["listId"]; id != "L_1" {
		t.Errorf("listId = %v, want L_1", id)
	}
}

func TestListReposWithoutALanguage(t *testing.T) {
	// primaryLanguage is null for a repository GitHub could not classify.
	client := newTestClient(t,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"nameWithOwner":"a/b","primaryLanguage":null}]}}}}`,
	)

	repos, err := client.ListRepos("L_1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if repos[0].Language != "" {
		t.Errorf("language = %q, want empty", repos[0].Language)
	}
}

func TestListReposFollowsPagination(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
			"nodes":[{"nameWithOwner":"a/one"}]}}}}`,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"nameWithOwner":"a/two"}]}}}}`,
	)

	repos, err := client.ListRepos("L_1", 0)
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

func TestListReposStopsAtLimit(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
			"nodes":[{"nameWithOwner":"a/one"},{"nameWithOwner":"a/two"}]}}}}`,
	)

	repos, err := client.ListRepos("L_1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}
	if first := transport.only(t).Variables["first"]; first != float64(2) {
		t.Errorf("first = %v, want the limit 2 rather than a full page", first)
	}
}

func TestLookupRepo(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"repository":{"id":"R_1","nameWithOwner":"cli/cli","viewerHasStarred":true}}}`,
	)

	ref, err := client.LookupRepo("cli", "cli")
	if err != nil {
		t.Fatal(err)
	}

	want := RepoRef{ID: "R_1", NameWithOwner: "cli/cli", ViewerHasStarred: true}
	if ref != want {
		t.Errorf("got %+v, want %+v", ref, want)
	}

	variables := transport.only(t).Variables
	if variables["owner"] != "cli" || variables["name"] != "cli" {
		t.Errorf("unexpected variables: %+v", variables)
	}
}

func TestLookupRepoNotFound(t *testing.T) {
	// A missing repository comes back as a null node rather than as an error.
	client := newTestClient(t, `{"data":{"repository":null}}`)

	_, err := client.LookupRepo("cli", "nope")
	if err == nil {
		t.Fatal("expected a not found error")
	}
	if want := "cli/nope"; !strings.Contains(err.Error(), want) {
		t.Errorf("the error should name %q, got %q", want, err)
	}
}

func TestSetListsForRepoSendsAnEmptyArrayForNoLists(t *testing.T) {
	// Removing the last list must clear the membership, so nil has to travel as
	// [] rather than as null, which the API rejects.
	client, transport := newRecordingClient(t,
		`{"data":{"updateUserListsForItem":{"lists":[]}}}`,
	)

	if err := client.SetListsForRepo("R_1", nil); err != nil {
		t.Fatal(err)
	}

	variables := transport.only(t).Variables
	if variables["itemId"] != "R_1" {
		t.Errorf("itemId = %v, want R_1", variables["itemId"])
	}
	listIDs, ok := variables["listIds"].([]interface{})
	if !ok {
		t.Fatalf("listIds = %#v, want an array", variables["listIds"])
	}
	if len(listIDs) != 0 {
		t.Errorf("listIds = %v, want empty", listIDs)
	}
}

func TestSetListsForRepoSendsTheWholeSet(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"updateUserListsForItem":{"lists":[{"id":"L_1"},{"id":"L_2"}]}}}`,
	)

	if err := client.SetListsForRepo("R_1", []string{"L_1", "L_2"}); err != nil {
		t.Fatal(err)
	}

	got := transport.only(t).Variables["listIds"]
	if want := []interface{}{"L_1", "L_2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("listIds = %v, want %v", got, want)
	}
}

func TestStar(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"addStar":{"starrable":{"id":"R_1"}}}}`,
	)

	if err := client.Star("R_1"); err != nil {
		t.Fatal(err)
	}
	if id := transport.only(t).Variables["starrableId"]; id != "R_1" {
		t.Errorf("starrableId = %v, want R_1", id)
	}
}

func TestMembershipIndexesEveryList(t *testing.T) {
	// One request per list, all in flight at once; a repository in two lists
	// gets both IDs, in list order whichever request answers first.
	client := newTestClient(t,
		listItems{"L_1", `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"nameWithOwner":"cli/cli"},{"nameWithOwner":"BurntSushi/ripgrep"}]}}}}`},
		listItems{"L_2", `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"nameWithOwner":"CLI/CLI"}]}}}}`},
	)

	membership, err := client.Membership([]List{
		{ID: "L_1", Slug: "one"},
		{ID: "L_2", Slug: "two"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// The index is keyed case-insensitively, so the two spellings of cli/cli
	// have to land on the same entry.
	want := Membership{
		"cli/cli":            {"L_1", "L_2"},
		"burntsushi/ripgrep": {"L_1"},
	}
	if !reflect.DeepEqual(membership, want) {
		t.Errorf("got %v, want %v", membership, want)
	}

	if got := membership.ListIDs("Cli/Cli"); !reflect.DeepEqual(got, []string{"L_1", "L_2"}) {
		t.Errorf("ListIDs is not case insensitive: %v", got)
	}
	if got := membership.ListIDs("unknown/repo"); got != nil {
		t.Errorf("an unlisted repository should have no lists, got %v", got)
	}
}

func TestMembershipNamesTheFailingList(t *testing.T) {
	client := newTestClient(t,
		listItems{"L_1", `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`},
		listItems{"L_2", `{"errors":[{"type":"NOT_FOUND","message":"gone"}]}`},
	)

	_, err := client.Membership([]List{{ID: "L_1", Slug: "one"}, {ID: "L_2", Slug: "two"}})
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := `reading list "two"`; !strings.Contains(err.Error(), want) {
		t.Errorf("the error should say %q, got %q", want, err)
	}
}

func TestMembershipOfNoLists(t *testing.T) {
	client := newTestClient(t)

	membership, err := client.Membership(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(membership) != 0 {
		t.Errorf("got %v, want an empty index", membership)
	}
}

func TestMembershipWithoutDoesNotAliasStoredSlice(t *testing.T) {
	membership := Membership{"cli/cli": {"list-a", "list-b"}}
	got, _ := membership.Without("cli/cli", "list-b")
	got[0] = "clobbered"
	if membership["cli/cli"][0] != "list-a" {
		t.Error("Without must not share the backing array of the stored membership")
	}
}
