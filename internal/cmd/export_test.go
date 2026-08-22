package cmd

import (
	"strings"
	"testing"

	"github.com/eggplants/gh-starlist/internal/starlist"
)

func names(repos []starlist.Repo) []string {
	got := make([]string, 0, len(repos))
	for _, repo := range repos {
		got = append(got, repo.NameWithOwner)
	}
	return got
}

func TestRepoSorterByStars(t *testing.T) {
	sorter, err := repoSorter("stars")
	if err != nil {
		t.Fatal(err)
	}

	repos := []starlist.Repo{
		{NameWithOwner: "a/low", StargazerCount: 1},
		{NameWithOwner: "a/high", StargazerCount: 100},
		{NameWithOwner: "a/mid", StargazerCount: 50},
	}
	sorter(repos)

	if got, want := strings.Join(names(repos), ","), "a/high,a/mid,a/low"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestRepoSorterByStarsIsStable(t *testing.T) {
	sorter, _ := repoSorter("stars")

	repos := []starlist.Repo{
		{NameWithOwner: "a/first", StargazerCount: 5},
		{NameWithOwner: "a/second", StargazerCount: 5},
		{NameWithOwner: "a/third", StargazerCount: 5},
	}
	sorter(repos)

	if got, want := strings.Join(names(repos), ","), "a/first,a/second,a/third"; got != want {
		t.Errorf("ties should keep their original order, got %s want %s", got, want)
	}
}

func TestRepoSorterByName(t *testing.T) {
	sorter, err := repoSorter("name")
	if err != nil {
		t.Fatal(err)
	}

	repos := []starlist.Repo{
		{NameWithOwner: "zed/zed"},
		{NameWithOwner: "BurntSushi/ripgrep"},
		{NameWithOwner: "apple/swift"},
	}
	sorter(repos)

	// Sorting is case insensitive, so BurntSushi does not jump ahead of apple
	// just because it is capitalized.
	if got, want := strings.Join(names(repos), ","), "apple/swift,BurntSushi/ripgrep,zed/zed"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestRepoSorterByStarredAt(t *testing.T) {
	sorter, err := repoSorter("starred-at")
	if err != nil {
		t.Fatal(err)
	}

	// Star lists never report starredAt, so those entries sort last while
	// keeping the order the list gave them.
	repos := []starlist.Repo{
		{NameWithOwner: "a/unknown-first"},
		{NameWithOwner: "a/older", StarredAt: "2023-01-01T00:00:00Z"},
		{NameWithOwner: "a/unknown-second"},
		{NameWithOwner: "a/newer", StarredAt: "2024-01-01T00:00:00Z"},
	}
	sorter(repos)

	want := "a/newer,a/older,a/unknown-first,a/unknown-second"
	if got := strings.Join(names(repos), ","); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestRepoSorterHandlesEmptyInput(t *testing.T) {
	for _, mode := range []string{"stars", "name", "starred-at"} {
		sorter, err := repoSorter(mode)
		if err != nil {
			t.Fatal(err)
		}
		sorter(nil)
		sorter([]starlist.Repo{})
	}
}

func TestRepoSorterRejectsAnUnknownMode(t *testing.T) {
	sorter, err := repoSorter("bogus")
	if err == nil {
		t.Fatal("expected an error")
	}
	if sorter != nil {
		t.Error("no sorter should come back with an error")
	}
	for _, want := range []string{"bogus", "stars", "starred-at", "name"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should mention %q, got %q", want, err)
		}
	}
}

func TestExportRejectsAnUnknownFormatBeforeAnyRequest(t *testing.T) {
	_, err := execute(t, "export", "--format", "xml")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("got %q, want an unknown format error", err)
	}
}

func TestExportRejectsAnUnknownSortBeforeAnyRequest(t *testing.T) {
	_, err := execute(t, "export", "--sort", "bogus")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "unknown sort") {
		t.Errorf("got %q, want an unknown sort error", err)
	}
}

func TestExportDefaults(t *testing.T) {
	root := NewRoot("test")
	for _, sub := range root.Commands() {
		if sub.Name() != "export" {
			continue
		}
		flags := sub.Flags()
		defaults := map[string]string{
			"format":              "md",
			"sort":                "stars",
			"uncategorized-title": "Uncategorized",
			"no-uncategorized":    "false",
			"template":            "",
			"output":              "",
			"user":                "",
		}
		for name, want := range defaults {
			flag := flags.Lookup(name)
			if flag == nil {
				t.Errorf("missing --%s", name)
				continue
			}
			if flag.DefValue != want {
				t.Errorf("--%s defaults to %q, want %q", name, flag.DefValue, want)
			}
		}
		return
	}
	t.Fatal("no export subcommand")
}
