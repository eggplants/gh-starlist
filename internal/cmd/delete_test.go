package cmd

import (
	"strings"
	"testing"
)

func TestConfirmRefusesToPromptWhenNotOnATerminal(t *testing.T) {
	// A script piping the output has nobody to answer the prompt, so deleting
	// has to fail loudly rather than read from a redirected stdin.
	t.Setenv("GH_FORCE_TTY", "")

	confirmed, err := confirm("Delete star list?")
	if err == nil {
		t.Fatal("expected an error")
	}
	if confirmed {
		t.Error("a failed prompt must not count as a confirmation")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("the error should point at --yes, got %q", err)
	}
}

func TestDeleteHasAYesFlag(t *testing.T) {
	root := NewRoot("test")
	for _, sub := range root.Commands() {
		if sub.Name() != "delete" {
			continue
		}
		flag := sub.Flags().Lookup("yes")
		if flag == nil {
			t.Fatal("missing --yes")
		}
		if flag.Shorthand != "y" {
			t.Errorf("--yes shorthand = %q, want y", flag.Shorthand)
		}
		if flag.DefValue != "false" {
			t.Errorf("--yes defaults to %q, want false", flag.DefValue)
		}
		return
	}
	t.Fatal("no delete subcommand")
}
