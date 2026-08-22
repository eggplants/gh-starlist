package cmd

import (
	"strings"
	"testing"

	"github.com/eggplants/gh-starlist/internal/starlist"
)

// twoLists is the canned answer to Lists() used across the command tests.
const twoLists = `{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[
	{"id":"L_1","name":"CLI/TUI","slug":"cli-tui","description":"","isPrivate":false,"items":{"totalCount":2}},
	{"id":"L_2","name":"Python Utils","slug":"python-utils","description":"Handy python libraries","isPrivate":true,"items":{"totalCount":1}}
]}}}}`

func TestListRendersATable(t *testing.T) {
	stubGitHub(t, twoLists)

	out, err := run(t, "list")
	if err != nil {
		t.Fatal(err)
	}

	want := "CLI/TUI\tcli-tui\t2\tpublic\t\n" +
		"Python Utils\tpython-utils\t1\tprivate\tHandy python libraries\n"
	if out.stdout != want {
		t.Errorf("got %q, want %q", out.stdout, want)
	}
}

func TestListJSON(t *testing.T) {
	stubGitHub(t, twoLists)

	out, err := run(t, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var lists []starlist.List
	decodeJSON(t, out.stdout, &lists)
	if len(lists) != 2 {
		t.Fatalf("got %d lists, want 2", len(lists))
	}
	if lists[0].Slug != "cli-tui" || lists[0].RepoCount != 2 {
		t.Errorf("unexpected first list: %+v", lists[0])
	}
	if !lists[1].IsPrivate {
		t.Errorf("the second list should be private: %+v", lists[1])
	}
}

func TestListJSONOfNoListsIsAnEmptyArray(t *testing.T) {
	// `gh starlist list --json | jq '.[]'` has to work for a fresh account, so
	// the output must be [] rather than null.
	stubGitHub(t, `{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`)

	out, err := run(t, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.stdout); got != "[]" {
		t.Errorf("got %q, want []", got)
	}
}

func TestListWithNoListsSaysSoOnStderr(t *testing.T) {
	stubGitHub(t, `{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`)

	out, err := run(t, "list")
	if err != nil {
		t.Fatal(err)
	}
	if out.stdout != "" {
		t.Errorf("nothing should reach stdout, got %q", out.stdout)
	}
	if !strings.Contains(out.stderr, "no star lists found") {
		t.Errorf("stderr = %q, want a note that there are no lists", out.stderr)
	}
}

func TestListPassesTheUserAndLimitThrough(t *testing.T) {
	transport := stubGitHub(t, `{"data":{"user":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`)

	if _, err := run(t, "list", "--user", "octocat", "--limit", "5"); err != nil {
		t.Fatal(err)
	}

	if len(transport.sent) != 1 {
		t.Fatalf("got %d requests, want 1", len(transport.sent))
	}
	variables := transport.sent[0].Variables
	if variables["login"] != "octocat" {
		t.Errorf("login = %v, want octocat", variables["login"])
	}
	if variables["first"] != float64(5) {
		t.Errorf("first = %v, want the limit 5", variables["first"])
	}
}

const cliTuiRepos = `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[
	{"nameWithOwner":"dandavison/delta","description":"A syntax-highlighting pager","url":"https://github.com/dandavison/delta","stargazerCount":31830,"primaryLanguage":{"name":"Rust"}},
	{"nameWithOwner":"tomnomnom/gf","description":"A wrapper around grep","url":"https://github.com/tomnomnom/gf","stargazerCount":2139,"primaryLanguage":{"name":"Go"}}
]}}}}`

func TestViewRendersATable(t *testing.T) {
	stubGitHub(t, twoLists, cliTuiRepos)

	out, err := run(t, "view", "cli-tui")
	if err != nil {
		t.Fatal(err)
	}

	want := "dandavison/delta\t31830\tRust\tA syntax-highlighting pager\n" +
		"tomnomnom/gf\t2139\tGo\tA wrapper around grep\n"
	if out.stdout != want {
		t.Errorf("got %q, want %q", out.stdout, want)
	}
}

func TestViewResolvesAListByName(t *testing.T) {
	transport := stubGitHub(t, twoLists, cliTuiRepos)

	// The README promises cli-tui, CLI/TUI and cli/tui all reach the same list.
	if _, err := run(t, "view", "cli/tui"); err != nil {
		t.Fatal(err)
	}
	if len(transport.sent) != 2 {
		t.Fatalf("got %d requests, want 2", len(transport.sent))
	}
	if id := transport.sent[1].Variables["listId"]; id != "L_1" {
		t.Errorf("listId = %v, want L_1", id)
	}
}

func TestViewUnknownListFails(t *testing.T) {
	stubGitHub(t, twoLists)

	if _, err := run(t, "view", "nope"); err == nil {
		t.Fatal("expected an error for an unknown list")
	}
}

func TestViewJSONCarriesTheListAndItsRepos(t *testing.T) {
	stubGitHub(t, twoLists, cliTuiRepos)

	out, err := run(t, "view", "cli-tui", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var payload struct {
		Slug  string          `json:"slug"`
		Name  string          `json:"name"`
		Repos []starlist.Repo `json:"repos"`
	}
	decodeJSON(t, out.stdout, &payload)

	if payload.Slug != "cli-tui" || payload.Name != "CLI/TUI" {
		t.Errorf("the list fields should be inlined, got %+v", payload)
	}
	if len(payload.Repos) != 2 || payload.Repos[0].NameWithOwner != "dandavison/delta" {
		t.Errorf("unexpected repos: %+v", payload.Repos)
	}
}

func TestViewEmptyList(t *testing.T) {
	stubGitHub(t, twoLists, `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`)

	out, err := run(t, "view", "cli-tui")
	if err != nil {
		t.Fatal(err)
	}
	if out.stdout != "" {
		t.Errorf("nothing should reach stdout, got %q", out.stdout)
	}
	if !strings.Contains(out.stderr, "CLI/TUI is empty") {
		t.Errorf("stderr = %q, want an empty list note", out.stderr)
	}
}

func TestViewJSONOfAnEmptyListIsAnEmptyArray(t *testing.T) {
	stubGitHub(t, twoLists, `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`)

	out, err := run(t, "view", "cli-tui", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.stdout, `"repos": []`) {
		t.Errorf("repos should be [] rather than null, got %s", out.stdout)
	}
}

func TestCreate(t *testing.T) {
	transport := stubGitHub(t,
		`{"data":{"createUserList":{"list":{"id":"L_9","name":"Rust Tools","slug":"rust-tools","description":"d","isPrivate":true}}}}`,
	)

	out, err := run(t, "create", "Rust Tools", "-d", "d", "--private")
	if err != nil {
		t.Fatal(err)
	}

	variables := transport.sent[0].Variables
	if variables["name"] != "Rust Tools" || variables["description"] != "d" || variables["isPrivate"] != true {
		t.Errorf("unexpected variables: %+v", variables)
	}
	if want := `created private star list "Rust Tools" (rust-tools)`; !strings.Contains(out.stderr, want) {
		t.Errorf("stderr = %q, want %q", out.stderr, want)
	}
	if out.stdout != "" {
		t.Errorf("progress belongs on stderr, got %q on stdout", out.stdout)
	}
}

func TestEditRenamesAList(t *testing.T) {
	transport := stubGitHub(t, twoLists,
		`{"data":{"updateUserList":{"list":{"id":"L_1","name":"Rust CLI","slug":"rust-cli","isPrivate":false}}}}`,
	)

	out, err := run(t, "edit", "cli-tui", "--name", "Rust CLI")
	if err != nil {
		t.Fatal(err)
	}

	variables := transport.sent[1].Variables
	if variables["listId"] != "L_1" {
		t.Errorf("listId = %v, want L_1", variables["listId"])
	}
	if variables["name"] != "Rust CLI" {
		t.Errorf("name = %v, want the new name", variables["name"])
	}
	if variables["isPrivate"] != nil {
		t.Errorf("an untouched visibility should travel as null, got %v", variables["isPrivate"])
	}
	if want := `updated public star list "Rust CLI" (rust-cli)`; !strings.Contains(out.stderr, want) {
		t.Errorf("stderr = %q, want %q", out.stderr, want)
	}
}

func TestEditMakesAListPublic(t *testing.T) {
	transport := stubGitHub(t, twoLists,
		`{"data":{"updateUserList":{"list":{"id":"L_2","name":"Python Utils","slug":"python-utils","isPrivate":false}}}}`,
	)

	if _, err := run(t, "edit", "python-utils", "--public"); err != nil {
		t.Fatal(err)
	}

	variables := transport.sent[1].Variables
	// --public is the absence of --private, so false has to be sent explicitly.
	if got := variables["isPrivate"]; got != false {
		t.Errorf("isPrivate = %v, want false", got)
	}
	// Changing only the visibility must not blank the other fields: an unset
	// flag travels as null, never as the empty string it defaults to.
	for _, key := range []string{"name", "description"} {
		if got := variables[key]; got != nil {
			t.Errorf("%s = %#v, want null for a flag that was not passed", key, got)
		}
	}
}

func TestDeleteWithYesSkipsThePrompt(t *testing.T) {
	transport := stubGitHub(t, twoLists, `{"data":{"deleteUserList":{"user":{"login":"octocat"}}}}`)

	out, err := run(t, "delete", "cli-tui", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if id := transport.sent[1].Variables["listId"]; id != "L_1" {
		t.Errorf("listId = %v, want L_1", id)
	}
	if want := `deleted star list "CLI/TUI"`; !strings.Contains(out.stderr, want) {
		t.Errorf("stderr = %q, want %q", out.stderr, want)
	}
}

func TestDeleteWithoutYesRefusesWhenItCannotPrompt(t *testing.T) {
	transport := stubGitHub(t, twoLists)

	out, err := run(t, "delete", "cli-tui")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("got %q, want the error to point at --yes", err)
	}
	// The list was resolved, but nothing was deleted.
	if len(transport.sent) != 1 {
		t.Errorf("got %d requests, want only the list lookup", len(transport.sent))
	}
	_ = out
}

const repoRefDelta = `{"data":{"repository":{"id":"R_delta","nameWithOwner":"dandavison/delta","viewerHasStarred":true}}}`

// listMembership answers the two ListRepos calls Membership makes over the two
// canned lists: delta sits in CLI/TUI, nothing in Python Utils.
var listMembership = []interface{}{
	listItems{"L_1", `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[{"nameWithOwner":"dandavison/delta"}]}}}}`},
	listItems{"L_2", `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`},
}

func TestAddPutsARepoInAListKeepingItsOtherLists(t *testing.T) {
	bodies := append([]interface{}{twoLists, repoRefDelta}, listMembership...)
	bodies = append(bodies, `{"data":{"updateUserListsForItem":{"lists":[{"id":"L_1"},{"id":"L_2"}]}}}`)
	transport := stubGitHub(t, bodies...)

	out, err := run(t, "add", "python-utils", "dandavison/delta")
	if err != nil {
		t.Fatal(err)
	}

	// The API replaces the whole membership, so the list the repo was already
	// in has to be sent along with the new one.
	last := transport.sent[len(transport.sent)-1].Variables
	if last["itemId"] != "R_delta" {
		t.Errorf("itemId = %v, want R_delta", last["itemId"])
	}
	listIDs, ok := last["listIds"].([]interface{})
	if !ok || len(listIDs) != 2 || listIDs[0] != "L_1" || listIDs[1] != "L_2" {
		t.Errorf("listIds = %v, want both L_1 and L_2", last["listIds"])
	}
	if want := `dandavison/delta: added to "Python Utils"`; !strings.Contains(out.stderr, want) {
		t.Errorf("stderr = %q, want %q", out.stderr, want)
	}
}

func TestAddIsANoOpWhenTheRepoIsAlreadyInTheList(t *testing.T) {
	bodies := append([]interface{}{twoLists, repoRefDelta}, listMembership...)
	transport := stubGitHub(t, bodies...)

	out, err := run(t, "add", "cli-tui", "dandavison/delta")
	if err != nil {
		t.Fatal(err)
	}
	// No mutation: the transport would have refused a fifth request anyway.
	if len(transport.sent) != 4 {
		t.Errorf("got %d requests, want no mutation", len(transport.sent))
	}
	if want := "dandavison/delta: no change"; !strings.Contains(out.stderr, want) {
		t.Errorf("stderr = %q, want %q", out.stderr, want)
	}
}

func TestAddStarsARepoThatIsNotStarredYet(t *testing.T) {
	unstarred := `{"data":{"repository":{"id":"R_new","nameWithOwner":"a/new","viewerHasStarred":false}}}`
	bodies := append([]interface{}{twoLists, unstarred}, listMembership...)
	bodies = append(bodies,
		`{"data":{"updateUserListsForItem":{"lists":[{"id":"L_1"}]}}}`,
		`{"data":{"addStar":{"starrable":{"id":"R_new"}}}}`,
	)
	transport := stubGitHub(t, bodies...)

	out, err := run(t, "add", "cli-tui", "a/new")
	if err != nil {
		t.Fatal(err)
	}

	last := transport.sent[len(transport.sent)-1]
	if !strings.Contains(last.Query, "mutation Star") {
		t.Errorf("the last request should star the repository, got:\n%s", last.Query)
	}
	if last.Variables["starrableId"] != "R_new" {
		t.Errorf("starrableId = %v, want R_new", last.Variables["starrableId"])
	}
	if want := "a/new: starred"; !strings.Contains(out.stderr, want) {
		t.Errorf("stderr = %q, want %q", out.stderr, want)
	}
}

func TestAddWithNoStarLeavesTheRepoUnstarred(t *testing.T) {
	unstarred := `{"data":{"repository":{"id":"R_new","nameWithOwner":"a/new","viewerHasStarred":false}}}`
	bodies := append([]interface{}{twoLists, unstarred}, listMembership...)
	bodies = append(bodies, `{"data":{"updateUserListsForItem":{"lists":[{"id":"L_1"}]}}}`)
	transport := stubGitHub(t, bodies...)

	out, err := run(t, "add", "cli-tui", "a/new", "--no-star")
	if err != nil {
		t.Fatal(err)
	}

	for _, request := range transport.sent {
		if strings.Contains(request.Query, "mutation Star") {
			t.Error("--no-star should not star anything")
		}
	}
	if strings.Contains(out.stderr, "starred") {
		t.Errorf("stderr = %q, should not report a star", out.stderr)
	}
}

func TestAddRejectsAMalformedRepositoryBeforeMutating(t *testing.T) {
	transport := stubGitHub(t, twoLists)

	if _, err := run(t, "add", "cli-tui", "not-a-repo"); err == nil {
		t.Fatal("expected an error")
	}
	if len(transport.sent) != 1 {
		t.Errorf("got %d requests, want only the list lookup", len(transport.sent))
	}
}

func TestRemoveTakesARepoOutOfOneListOnly(t *testing.T) {
	// delta sits in both lists; removing it from CLI/TUI must leave L_2.
	inBoth := `{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[{"nameWithOwner":"dandavison/delta"}]}}}}`
	membership := []interface{}{listItems{"L_1", inBoth}, listItems{"L_2", inBoth}}
	bodies := append([]interface{}{twoLists, repoRefDelta}, membership...)
	bodies = append(bodies, `{"data":{"updateUserListsForItem":{"lists":[{"id":"L_2"}]}}}`)
	transport := stubGitHub(t, bodies...)

	out, err := run(t, "remove", "cli-tui", "dandavison/delta")
	if err != nil {
		t.Fatal(err)
	}

	last := transport.sent[len(transport.sent)-1].Variables
	listIDs, ok := last["listIds"].([]interface{})
	if !ok || len(listIDs) != 1 || listIDs[0] != "L_2" {
		t.Errorf("listIds = %v, want only L_2", last["listIds"])
	}
	if want := `dandavison/delta: removed from "CLI/TUI"`; !strings.Contains(out.stderr, want) {
		t.Errorf("stderr = %q, want %q", out.stderr, want)
	}
}

func TestRemoveNeverStars(t *testing.T) {
	unstarred := `{"data":{"repository":{"id":"R_delta","nameWithOwner":"dandavison/delta","viewerHasStarred":false}}}`
	bodies := append([]interface{}{twoLists, unstarred}, listMembership...)
	bodies = append(bodies, `{"data":{"updateUserListsForItem":{"lists":[]}}}`)
	transport := stubGitHub(t, bodies...)

	if _, err := run(t, "remove", "cli-tui", "dandavison/delta"); err != nil {
		t.Fatal(err)
	}
	for _, request := range transport.sent {
		if strings.Contains(request.Query, "mutation Star") {
			t.Error("remove should never star a repository")
		}
	}
}
