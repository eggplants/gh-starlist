package starlist

import "testing"

var testLists = []List{
	{ID: "1", Name: "CLI/TUI", Slug: "cli-tui"},
	{ID: "2", Name: "Python Utils", Slug: "python-utils"},
	{ID: "3", Name: "python utils", Slug: "python-utils-1"},
}

func TestFindList(t *testing.T) {
	cases := map[string]string{
		"cli-tui":      "1",
		"CLI/TUI":      "1",
		"cli/tui":      "1",
		"CLI-TUI":      "1",
		"python-utils": "2",
		"Python Utils": "2",
	}
	for key, wantID := range cases {
		list, err := FindList(testLists, key)
		if err != nil {
			t.Errorf("FindList(%q): %v", key, err)
			continue
		}
		if list.ID != wantID {
			t.Errorf("FindList(%q) = %s, want %s", key, list.ID, wantID)
		}
	}
}

func TestFindListAmbiguous(t *testing.T) {
	if _, err := FindList(testLists, "PYTHON UTILS"); err == nil {
		t.Fatal("expected an ambiguity error")
	}
}

func TestFindListMissing(t *testing.T) {
	if _, err := FindList(testLists, "nope"); err == nil {
		t.Fatal("expected a not found error")
	}
}
