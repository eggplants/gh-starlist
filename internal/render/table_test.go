package render

import (
	"io"
	"os"
	"testing"
)

// captureStdout swaps os.Stdout for a pipe and returns what was written to it.
// NewTable reads os.Stdout through term.FromEnv at call time, so the swap has
// to be in place before it runs.
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
		out, _ := io.ReadAll(reader)
		done <- string(out)
	}()

	run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return <-done
}

func TestNewTableWritesTSVWhenPiped(t *testing.T) {
	// A pipe is not a terminal, so the table has to degrade to tab separated
	// values with no header, which is what `gh starlist list | cut -f2` needs.
	t.Setenv("GH_FORCE_TTY", "")

	out := captureStdout(t, func() {
		table := NewTable()
		table.AddHeader([]string{"NAME", "SLUG"})
		table.AddField("CLI/TUI")
		table.AddField("cli-tui")
		table.EndRow()
		if err := table.Render(); err != nil {
			t.Error(err)
		}
	})

	if want := "CLI/TUI\tcli-tui\n"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestNewTableWritesColumnsOnATerminal(t *testing.T) {
	// GH_FORCE_TTY makes go-gh treat the output as a terminal and fixes the
	// width, so the padded rendering can be asserted on.
	t.Setenv("GH_FORCE_TTY", "40")

	out := captureStdout(t, func() {
		table := NewTable()
		table.AddHeader([]string{"NAME", "SLUG"})
		table.AddField("CLI/TUI")
		table.AddField("cli-tui")
		table.EndRow()
		if err := table.Render(); err != nil {
			t.Error(err)
		}
	})

	if want := "NAME     SLUG\nCLI/TUI  cli-tui\n"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}
