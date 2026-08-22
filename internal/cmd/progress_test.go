package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressDisabledStaysSilent(t *testing.T) {
	// stderr is not a terminal under `go test`, so the reporter must be inert.
	bar := newProgress(false)
	bar.stage("Reading star lists")
	bar.update(10, 100)
	bar.done()
	if bar.out != nil {
		t.Error("progress should be disabled when stderr is not a terminal")
	}

	if quiet := newProgress(true); quiet.out != nil {
		t.Error("--quiet should disable progress")
	}
}

func TestProgressLine(t *testing.T) {
	var buffer bytes.Buffer
	bar := &progress{out: &buffer, width: 80}

	bar.stage("Reading list %q (%d/%d)", "CLI/TUI", 2, 5)
	bar.lastAt = bar.lastAt.Add(-minRedraw) // let the next paint through
	bar.update(120, 450)
	bar.done()

	lines := strings.Split(buffer.String(), "\r\033[K")
	if len(lines) != 4 {
		t.Fatalf("want a paint per call plus the final erase, got %q", buffer.String())
	}
	if want := `Reading list "CLI/TUI" (2/5)`; !strings.Contains(lines[1], want) {
		t.Errorf("stage line %q should hold %q", lines[1], want)
	}
	if want := "120/450"; !strings.Contains(lines[2], want) {
		t.Errorf("update line %q should hold %q", lines[2], want)
	}
	if lines[3] != "" {
		t.Errorf("the line should be erased at the end, got %q", lines[3])
	}
}

func TestProgressUpdateWithoutTotal(t *testing.T) {
	// A connection without totalCount still shows how much has arrived.
	var buffer bytes.Buffer
	bar := &progress{out: &buffer, width: 80}
	bar.update(37, 0)
	if !strings.Contains(buffer.String(), " 37") {
		t.Errorf("want a bare count, got %q", buffer.String())
	}
}

func TestProgressThrottlesRedraws(t *testing.T) {
	var buffer bytes.Buffer
	bar := &progress{out: &buffer, width: 80}
	bar.update(1, 10)
	painted := buffer.Len()
	bar.update(2, 10)
	if buffer.Len() != painted {
		t.Error("a redraw within minRedraw should be skipped")
	}
}
