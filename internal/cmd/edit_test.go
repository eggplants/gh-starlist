package cmd

import (
	"strings"
	"testing"
)

func TestEditRejectsBothVisibilitiesBeforeAnyRequest(t *testing.T) {
	_, err := execute(t, "edit", "my-list", "--private", "--public")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("got %q, want a mutual exclusion error", err)
	}
}

func TestEditRejectsAnEmptyChangeBeforeAnyRequest(t *testing.T) {
	_, err := execute(t, "edit", "my-list")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "nothing to edit") {
		t.Errorf("got %q, want a nothing to edit error", err)
	}
	// The error has to say which flags would have worked.
	for _, want := range []string{"--name", "--description", "--private", "--public"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should mention %s, got %q", want, err)
		}
	}
}

func TestEditTreatsAnExplicitEmptyValueAsAChange(t *testing.T) {
	// `--description ""` clears the description; it must not be mistaken for an
	// omitted flag, which is why the command asks pflag whether it Changed
	// rather than comparing against the zero value.
	root := NewRoot("test")
	for _, sub := range root.Commands() {
		if sub.Name() != "edit" {
			continue
		}
		if err := sub.Flags().Set("description", ""); err != nil {
			t.Fatal(err)
		}
		if !sub.Flags().Changed("description") {
			t.Error("setting --description to an empty string should count as a change")
		}
		return
	}
	t.Fatal("no edit subcommand")
}

func TestEditFlagShorthands(t *testing.T) {
	root := NewRoot("test")
	for _, sub := range root.Commands() {
		if sub.Name() != "edit" {
			continue
		}
		for name, shorthand := range map[string]string{"name": "n", "description": "d"} {
			flag := sub.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("missing --%s", name)
				continue
			}
			if flag.Shorthand != shorthand {
				t.Errorf("--%s shorthand = %q, want %q", name, flag.Shorthand, shorthand)
			}
		}
		return
	}
	t.Fatal("no edit subcommand")
}
