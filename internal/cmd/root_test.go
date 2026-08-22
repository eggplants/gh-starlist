package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// execute runs the command tree with args and returns its error together with
// everything cobra wrote.
//
// Only inputs that fail before a GitHub client is built belong here: the
// commands reach the network as soon as their arguments validate.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRoot("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	return out.String(), root.Execute()
}

// captureStdout swaps os.Stdout for a pipe and returns what was written to it.
func captureStdout(t *testing.T, run func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	done := make(chan string, 1)
	go func() {
		captured, _ := io.ReadAll(reader)
		done <- string(captured)
	}()

	run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return <-done
}

func TestNewRootWiring(t *testing.T) {
	root := NewRoot("1.2.3")

	if root.Version != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", root.Version)
	}
	// The gh extension prints its own errors, so cobra must stay quiet.
	if !root.SilenceUsage || !root.SilenceErrors {
		t.Error("the root command should silence cobra's own error and usage output")
	}

	want := map[string][]string{
		"list":   {"ls"},
		"view":   nil,
		"create": nil,
		"edit":   nil,
		"delete": {"rm"},
		"add":    nil,
		"remove": nil,
		"export": nil,
	}
	got := map[string]*cobra.Command{}
	for _, sub := range root.Commands() {
		got[sub.Name()] = sub
	}
	for name, aliases := range want {
		sub, present := got[name]
		if !present {
			t.Errorf("missing subcommand %q", name)
			continue
		}
		for _, alias := range aliases {
			if !sub.HasAlias(alias) {
				t.Errorf("subcommand %q should have the alias %q", name, alias)
			}
		}
	}
}

func TestUnknownSubcommand(t *testing.T) {
	if _, err := execute(t, "bogus"); err == nil {
		t.Fatal("expected an error for an unknown subcommand")
	}
}

func TestArgumentCountIsValidatedBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"list takes no arguments", []string{"list", "extra"}},
		{"view needs a list", []string{"view"}},
		{"view takes one list", []string{"view", "a", "b"}},
		{"create needs a name", []string{"create"}},
		{"create takes one name", []string{"create", "a", "b"}},
		{"edit needs a list", []string{"edit"}},
		{"delete needs a list", []string{"delete"}},
		{"add needs a list and a repository", []string{"add", "my-list"}},
		{"remove needs a list and a repository", []string{"remove", "my-list"}},
		{"export takes no arguments", []string{"export", "extra"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := execute(t, tc.args...); err == nil {
				t.Errorf("%v should be rejected", tc.args)
			}
		})
	}
}

func TestParseRepo(t *testing.T) {
	cases := []struct {
		in          string
		owner, name string
	}{
		{"cli/cli", "cli", "cli"},
		{"BurntSushi/ripgrep", "BurntSushi", "ripgrep"},
		{"https://github.com/cli/cli", "cli", "cli"},
		{"https://github.com/cli/cli.git", "cli", "cli"},
		{"git@github.com:cli/cli.git", "cli", "cli"},
		// A leading host is accepted and dropped: star lists only ever live on
		// the account the token belongs to.
		{"github.com/cli/cli", "cli", "cli"},
	}
	for _, tc := range cases {
		owner, name, err := parseRepo(tc.in)
		if err != nil {
			t.Errorf("parseRepo(%q): %v", tc.in, err)
			continue
		}
		if owner != tc.owner || name != tc.name {
			t.Errorf("parseRepo(%q) = %s/%s, want %s/%s", tc.in, owner, name, tc.owner, tc.name)
		}
	}
}

func TestParseRepoRejectsMalformedInput(t *testing.T) {
	for _, in := range []string{"", "cli", "cli/", "/cli", "a/b/c/d", "https://github.com/cli"} {
		owner, name, err := parseRepo(in)
		if err == nil {
			t.Errorf("parseRepo(%q) = %s/%s, want an error", in, owner, name)
			continue
		}
		if !strings.Contains(err.Error(), "invalid repository") {
			t.Errorf("parseRepo(%q) error = %q, should say which argument is wrong", in, err)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := printJSON(map[string]string{"name": "a"}); err != nil {
			t.Error(err)
		}
	})

	if want := "{\n  \"name\": \"a\"\n}\n"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestPrintJSONKeepsHTMLCharactersReadable(t *testing.T) {
	// Descriptions routinely hold & and <, and escaping them to & and
	// < would make the output unpleasant to read.
	out := captureStdout(t, func() {
		if err := printJSON(map[string]string{"description": "a & b <c>"}); err != nil {
			t.Error(err)
		}
	})

	if !strings.Contains(out, "a & b <c>") {
		t.Errorf("HTML characters should not be escaped, got %q", out)
	}
}
