package starlist

import (
	"encoding/json"
	"testing"
)

func TestVisibility(t *testing.T) {
	if got := (List{IsPrivate: true}).Visibility(); got != "private" {
		t.Errorf("got %q, want private", got)
	}
	if got := (List{}).Visibility(); got != "public" {
		t.Errorf("got %q, want public", got)
	}
}

func TestRepoOmitsAnUnknownStarredAt(t *testing.T) {
	// Star lists do not expose starredAt, so `view --json` must not claim a
	// repository was starred at the zero time.
	payload, err := json.Marshal(Repo{NameWithOwner: "cli/cli"})
	if err != nil {
		t.Fatal(err)
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if _, present := fields["starredAt"]; present {
		t.Errorf("starredAt should be omitted when unknown, got %s", payload)
	}

	payload, err = json.Marshal(Repo{NameWithOwner: "cli/cli", StarredAt: "2024-01-01T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["starredAt"] != "2024-01-01T00:00:00Z" {
		t.Errorf("starredAt should be kept when known, got %s", payload)
	}
}
