package starlist

import (
	"strings"
	"testing"
)

func TestListsReadsTheUserConnection(t *testing.T) {
	// The response carries both connections; asking for a user must read the
	// user one and ignore the viewer one.
	client, transport := newRecordingClient(t,
		`{"data":{
			"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[{"id":"V","name":"Mine","slug":"mine"}]}},
			"user":{"lists":{"pageInfo":{"hasNextPage":false},
				"nodes":[{"id":"U","name":"Theirs","slug":"theirs","description":"d","isPrivate":false,
					"updatedAt":"2024-01-02T03:04:05Z","items":{"totalCount":7}}]}}}}`,
	)

	lists, err := client.Lists("octocat", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("got %d lists, want 1", len(lists))
	}

	want := List{
		ID: "U", Name: "Theirs", Slug: "theirs", Description: "d",
		IsPrivate: false, RepoCount: 7, UpdatedAt: "2024-01-02T03:04:05Z",
	}
	if lists[0] != want {
		t.Errorf("got %+v, want %+v", lists[0], want)
	}

	request := transport.only(t)
	if !strings.Contains(request.Query, "query UserLists") {
		t.Errorf("a user should be read with the user query, got:\n%s", request.Query)
	}
	if request.Variables["login"] != "octocat" {
		t.Errorf("login = %v, want octocat", request.Variables["login"])
	}
}

func TestListsSendsTheCursorOfThePreviousPage(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
			"nodes":[{"id":"1","name":"A","slug":"a"}]}}}}`,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"id":"2","name":"B","slug":"b"}]}}}}`,
	)

	if _, err := client.Lists("", 0); err != nil {
		t.Fatal(err)
	}
	if len(transport.sent) != 2 {
		t.Fatalf("got %d requests, want 2", len(transport.sent))
	}
	if after := transport.sent[0].Variables["after"]; after != nil {
		t.Errorf("the first page should start with no cursor, got %v", after)
	}
	if after := transport.sent[1].Variables["after"]; after != "CUR" {
		t.Errorf("the second page should follow endCursor, got %v", after)
	}
	if !strings.Contains(transport.sent[0].Query, "query ViewerLists") {
		t.Error("no user should be read with the viewer query")
	}
}

func TestListsEmptyResult(t *testing.T) {
	client := newTestClient(t, `{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`)

	lists, err := client.Lists("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 0 {
		t.Errorf("got %d lists, want none", len(lists))
	}
}

func TestCreateList(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"createUserList":{"list":{"id":"L_1","name":"Rust Tools","slug":"rust-tools",
			"description":"Things written in Rust","isPrivate":true}}}}`,
	)

	list, err := client.CreateList("Rust Tools", "Things written in Rust", true)
	if err != nil {
		t.Fatal(err)
	}

	want := List{ID: "L_1", Name: "Rust Tools", Slug: "rust-tools", Description: "Things written in Rust", IsPrivate: true}
	if list != want {
		t.Errorf("got %+v, want %+v", list, want)
	}

	variables := transport.only(t).Variables
	if variables["name"] != "Rust Tools" || variables["description"] != "Things written in Rust" || variables["isPrivate"] != true {
		t.Errorf("unexpected variables: %+v", variables)
	}
}

func TestUpdateListOmitsUnsetFields(t *testing.T) {
	// The API reads a null field as "keep the current value", so an unset flag
	// has to travel as null rather than as the zero value.
	name := "Rust CLI"
	client, transport := newRecordingClient(t,
		`{"data":{"updateUserList":{"list":{"id":"L_1","name":"Rust CLI","slug":"rust-cli"}}}}`,
	)

	list, err := client.UpdateList("L_1", &name, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Rust CLI" || list.Slug != "rust-cli" {
		t.Errorf("got %+v", list)
	}

	variables := transport.only(t).Variables
	if variables["listId"] != "L_1" {
		t.Errorf("listId = %v", variables["listId"])
	}
	if variables["name"] != "Rust CLI" {
		t.Errorf("name = %v, want the new name", variables["name"])
	}
	for _, key := range []string{"description", "isPrivate"} {
		value, present := variables[key]
		if !present {
			t.Errorf("%s should be sent explicitly as null", key)
		}
		if value != nil {
			t.Errorf("%s = %v, want null for an unset field", key, value)
		}
	}
}

func TestUpdateListSendsEmptyValuesWhenAsked(t *testing.T) {
	// Clearing a description means sending "", which must not be confused with
	// leaving the field untouched.
	empty := ""
	public := false
	client, transport := newRecordingClient(t,
		`{"data":{"updateUserList":{"list":{"id":"L_1","name":"A","slug":"a"}}}}`,
	)

	if _, err := client.UpdateList("L_1", nil, &empty, &public); err != nil {
		t.Fatal(err)
	}

	variables := transport.only(t).Variables
	if variables["description"] != "" {
		t.Errorf("description = %v, want an empty string", variables["description"])
	}
	if variables["isPrivate"] != false {
		t.Errorf("isPrivate = %v, want false", variables["isPrivate"])
	}
	if variables["name"] != nil {
		t.Errorf("name = %v, want null", variables["name"])
	}
}

func TestDeleteList(t *testing.T) {
	client, transport := newRecordingClient(t,
		`{"data":{"deleteUserList":{"user":{"login":"octocat"}}}}`,
	)

	if err := client.DeleteList("L_1"); err != nil {
		t.Fatal(err)
	}
	if id := transport.only(t).Variables["listId"]; id != "L_1" {
		t.Errorf("listId = %v, want L_1", id)
	}
}

func TestResolveList(t *testing.T) {
	client := newTestClient(t,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"id":"1","name":"CLI/TUI","slug":"cli-tui"},
				{"id":"2","name":"Python Utils","slug":"python-utils"}]}}}}`,
	)

	list, err := client.ResolveList("", "CLI/tui")
	if err != nil {
		t.Fatal(err)
	}
	if list.ID != "1" {
		t.Errorf("got list %s, want 1", list.ID)
	}
}

func TestResolveListUnknownKey(t *testing.T) {
	client := newTestClient(t,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},
			"nodes":[{"id":"1","name":"CLI/TUI","slug":"cli-tui"}]}}}}`,
	)

	if _, err := client.ResolveList("", "nope"); err == nil {
		t.Fatal("expected a not found error")
	}
}

func TestFindListPrefersAnExactMatchOverACaseInsensitiveOne(t *testing.T) {
	lists := []List{
		{ID: "1", Name: "Tools", Slug: "tools"},
		{ID: "2", Name: "TOOLS", Slug: "tools-1"},
	}

	list, err := FindList(lists, "TOOLS")
	if err != nil {
		t.Fatal(err)
	}
	if list.ID != "2" {
		t.Errorf("got list %s, want the exactly named 2", list.ID)
	}
}

func TestFindListAmbiguityNamesTheCandidates(t *testing.T) {
	_, err := FindList(testLists, "PYTHON UTILS")
	if err == nil {
		t.Fatal("expected an ambiguity error")
	}
	for _, want := range []string{"python-utils", "python-utils-1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should name %q, got %q", want, err)
		}
	}
}

func TestFindListEmpty(t *testing.T) {
	if _, err := FindList(nil, "anything"); err == nil {
		t.Fatal("expected a not found error for an empty set of lists")
	}
}
