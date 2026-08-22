package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggplants/gh-starlist/internal/render"
)

// exportBodies is a whole export conversation: the lists, the repositories of
// each list, then the starred repositories.
//
// delta is in CLI/TUI, requests is in Python Utils, and the starred set adds
// fd, which is in no list. The two connections spell delta differently, which
// pins down that the uncategorized section matches names case insensitively.
var exportBodies = []string{
	twoLists,
	`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[
		{"nameWithOwner":"DanDavison/Delta","description":"A pager","url":"https://github.com/dandavison/delta","stargazerCount":31830}
	]}}}}`,
	`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[
		{"nameWithOwner":"psf/requests","description":"HTTP for humans","url":"https://github.com/psf/requests","stargazerCount":52000}
	]}}}}`,
	`{"data":{"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":false},"edges":[
		{"starredAt":"2024-03-03T00:00:00Z","node":{"nameWithOwner":"sharkdp/fd","description":"A find alternative","url":"https://github.com/sharkdp/fd","stargazerCount":34000}},
		{"starredAt":"2024-02-02T00:00:00Z","node":{"nameWithOwner":"dandavison/delta","description":"A pager","url":"https://github.com/dandavison/delta","stargazerCount":31830}},
		{"starredAt":"2024-01-01T00:00:00Z","node":{"nameWithOwner":"psf/requests","description":"HTTP for humans","url":"https://github.com/psf/requests","stargazerCount":52000}}
	]}}}}`,
}

func TestExportMarkdown(t *testing.T) {
	stubGitHub(t, exportBodies...)

	out, err := run(t, "export")
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"## TOC",
		"",
		"- [CLI/TUI](#clitui)",
		"- [Python Utils](#python-utils)",
		"- [Uncategorized](#uncategorized)",
		"",
		"## CLI/TUI",
		"",
		"| Repository | Description | Stars |",
		"| --- | --- | --- |",
		"| [DanDavison/Delta](https://github.com/dandavison/delta) | A pager | ⭐31830 |",
		"",
		"## Python Utils",
		"",
		"| Repository | Description | Stars |",
		"| --- | --- | --- |",
		"| [psf/requests](https://github.com/psf/requests) | HTTP for humans | ⭐52000 |",
		"",
		"## Uncategorized",
		"",
		"| Repository | Description | Stars |",
		"| --- | --- | --- |",
		"| [sharkdp/fd](https://github.com/sharkdp/fd) | A find alternative | ⭐34000 |",
		"",
	}, "\n")
	if out.stdout != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.stdout, want)
	}
}

func TestExportSkipsUncategorizedWithoutFetchingStars(t *testing.T) {
	// Only the lists are needed, so the starred repositories must not be read.
	transport := stubGitHub(t, exportBodies[:3]...)

	out, err := run(t, "export", "--no-uncategorized")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.stdout, "Uncategorized") {
		t.Errorf("the uncategorized section should be gone:\n%s", out.stdout)
	}
	if len(transport.sent) != 3 {
		t.Errorf("got %d requests, want no starred repositories request", len(transport.sent))
	}
}

func TestExportRenamesTheUncategorizedSection(t *testing.T) {
	stubGitHub(t, exportBodies...)

	out, err := run(t, "export", "--uncategorized-title", "Everything else")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.stdout, "## Everything else") {
		t.Errorf("want the renamed heading in:\n%s", out.stdout)
	}
	if !strings.Contains(out.stdout, "- [Everything else](#everything-else)") {
		t.Errorf("the table of contents should follow the rename:\n%s", out.stdout)
	}
}

func TestExportSortsByName(t *testing.T) {
	// Two repositories in one list, given in descending star order, so only a
	// name sort can reorder them.
	bodies := []string{
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[
			{"id":"L_1","name":"Tools","slug":"tools","items":{"totalCount":2}}]}}}}`,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[
			{"nameWithOwner":"zed/zed","stargazerCount":100},
			{"nameWithOwner":"apple/swift","stargazerCount":1}]}}}}`,
	}
	stubGitHub(t, bodies...)

	out, err := run(t, "export", "--no-uncategorized", "--sort", "name")
	if err != nil {
		t.Fatal(err)
	}

	swift := strings.Index(out.stdout, "apple/swift")
	zed := strings.Index(out.stdout, "zed/zed")
	if swift < 0 || zed < 0 || swift > zed {
		t.Errorf("apple/swift should come first:\n%s", out.stdout)
	}
}

func TestExportSortsByStarsByDefault(t *testing.T) {
	bodies := []string{
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[
			{"id":"L_1","name":"Tools","slug":"tools","items":{"totalCount":2}}]}}}}`,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[
			{"nameWithOwner":"apple/swift","stargazerCount":1},
			{"nameWithOwner":"zed/zed","stargazerCount":100}]}}}}`,
	}
	stubGitHub(t, bodies...)

	out, err := run(t, "export", "--no-uncategorized")
	if err != nil {
		t.Fatal(err)
	}

	swift := strings.Index(out.stdout, "apple/swift")
	zed := strings.Index(out.stdout, "zed/zed")
	if zed < 0 || swift < 0 || zed > swift {
		t.Errorf("the most starred repository should come first:\n%s", out.stdout)
	}
}

func TestExportJSON(t *testing.T) {
	stubGitHub(t, exportBodies...)

	out, err := run(t, "export", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}

	var sections []render.Section
	decodeJSON(t, out.stdout, &sections)

	if len(sections) != 3 {
		t.Fatalf("got %d sections, want 3", len(sections))
	}
	if sections[0].Name != "CLI/TUI" || sections[2].Name != "Uncategorized" {
		t.Errorf("unexpected section names: %+v", sections)
	}
	if len(sections[2].Repos) != 1 || sections[2].Repos[0].NameWithOwner != "sharkdp/fd" {
		t.Errorf("only fd is in no list, got %+v", sections[2].Repos)
	}
	// Only the starred connection reports starredAt.
	if sections[2].Repos[0].StarredAt != "2024-03-03T00:00:00Z" {
		t.Errorf("starredAt = %q, want it preserved", sections[2].Repos[0].StarredAt)
	}
}

func TestExportWritesToAFile(t *testing.T) {
	stubGitHub(t, exportBodies...)
	path := filepath.Join(t.TempDir(), "STARS.md")

	out, err := run(t, "export", "-o", path)
	if err != nil {
		t.Fatal(err)
	}
	if out.stdout != "" {
		t.Errorf("with --output nothing should reach stdout, got %q", out.stdout)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "## CLI/TUI") {
		t.Errorf("the file should hold the export, got:\n%s", written)
	}
}

func TestExportAppliesATemplate(t *testing.T) {
	stubGitHub(t, exportBodies...)

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.md")
	template := "# My stars\n\n" + render.Placeholder + "\n\n_generated_\n"
	if err := os.WriteFile(templatePath, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "export", "--template", templatePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(out.stdout, "# My stars\n\n## TOC\n") {
		t.Errorf("the template header should come first:\n%s", out.stdout)
	}
	if !strings.HasSuffix(out.stdout, "\n\n_generated_\n") {
		t.Errorf("the template footer should come last:\n%s", out.stdout)
	}
	if strings.Contains(out.stdout, render.Placeholder) {
		t.Error("the placeholder should have been replaced")
	}
}

func TestExportFailsOnAMissingTemplate(t *testing.T) {
	stubGitHub(t, exportBodies...)

	if _, err := run(t, "export", "--template", filepath.Join(t.TempDir(), "absent.md")); err == nil {
		t.Fatal("expected an error for a missing template")
	}
}

func TestExportRejectsOutputWithJSON(t *testing.T) {
	stubGitHub(t, exportBodies...)
	path := filepath.Join(t.TempDir(), "out.json")

	_, err := run(t, "export", "--format", "json", "-o", path)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "--output is only supported with --format md") {
		t.Errorf("got %q", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("no file should have been written")
	}
}

func TestExportOfAnAccountWithNoLists(t *testing.T) {
	stubGitHub(t,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`,
		`{"data":{"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":false},"edges":[
			{"starredAt":"2024-01-01T00:00:00Z","node":{"nameWithOwner":"a/b","stargazerCount":1}}]}}}}`,
	)

	out, err := run(t, "export")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.stdout, "## Uncategorized") {
		t.Errorf("everything starred is uncategorized:\n%s", out.stdout)
	}
	if !strings.Contains(out.stdout, "[a/b](https://github.com/a/b)") {
		t.Errorf("want the starred repository in:\n%s", out.stdout)
	}
}

func TestExportOfAnEmptyAccount(t *testing.T) {
	stubGitHub(t,
		`{"data":{"viewer":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`,
		`{"data":{"viewer":{"starredRepositories":{"pageInfo":{"hasNextPage":false},"edges":[]}}}}`,
	)

	out, err := run(t, "export")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.stdout, "| *No repositories* | | |") {
		t.Errorf("an empty account should still render a valid document:\n%s", out.stdout)
	}
}

func TestExportNamesTheListItFailedOn(t *testing.T) {
	stubGitHub(t, twoLists,
		`{"data":{"node":{"items":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`,
		`{"errors":[{"type":"NOT_FOUND","message":"gone"}]}`,
	)

	_, err := run(t, "export")
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := `reading list "python-utils"`; !strings.Contains(err.Error(), want) {
		t.Errorf("the error should say %q, got %q", want, err)
	}
}

func TestExportReadsAnotherUser(t *testing.T) {
	transport := stubGitHub(t,
		`{"data":{"user":{"lists":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`,
		`{"data":{"user":{"starredRepositories":{"pageInfo":{"hasNextPage":false},"edges":[]}}}}`,
	)

	if _, err := run(t, "export", "--user", "octocat"); err != nil {
		t.Fatal(err)
	}
	if len(transport.sent) != 2 {
		t.Fatalf("got %d requests, want 2", len(transport.sent))
	}
	for i, request := range transport.sent {
		if request.Variables["login"] != "octocat" {
			t.Errorf("request %d login = %v, want octocat", i, request.Variables["login"])
		}
	}
}
