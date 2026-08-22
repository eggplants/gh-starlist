package starlist

import (
	"reflect"
	"testing"
)

var testMembership = Membership{
	"cli/cli": {"list-a", "list-b"},
}

func TestMembershipWith(t *testing.T) {
	got, changed := testMembership.With("CLI/cli", "list-c")
	if !changed {
		t.Fatal("adding a new list should report a change")
	}
	if want := []string{"list-a", "list-b", "list-c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	got, changed = testMembership.With("cli/cli", "list-a")
	if changed {
		t.Error("adding a list the repository already belongs to should be a no-op")
	}
	if want := []string{"list-a", "list-b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMembershipWithoutKeepsOtherLists(t *testing.T) {
	got, changed := testMembership.Without("cli/cli", "list-a")
	if !changed {
		t.Fatal("removing a list should report a change")
	}
	if want := []string{"list-b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, changed := testMembership.Without("cli/cli", "list-z"); changed {
		t.Error("removing a list the repository is not in should be a no-op")
	}
}

func TestMembershipUnknownRepo(t *testing.T) {
	got, changed := testMembership.With("owner/other", "list-a")
	if !changed {
		t.Fatal("adding an unlisted repository should report a change")
	}
	if want := []string{"list-a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMembershipDoesNotAliasStoredSlice(t *testing.T) {
	membership := Membership{"cli/cli": {"list-a", "list-b"}}
	got, _ := membership.With("cli/cli", "list-c")
	got[0] = "clobbered"
	if membership["cli/cli"][0] != "list-a" {
		t.Error("With must not share the backing array of the stored membership")
	}
}
