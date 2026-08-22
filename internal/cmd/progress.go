package cmd

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/cli/go-gh/v2/pkg/text"
)

// progress draws a single self-rewriting status line on stderr while a command
// walks the API. Exporting every list of a big account is dozens of round
// trips, and without a sign of life it just looks stuck.
//
// It only draws on a terminal: piped or redirected stderr keeps the plain
// output a script would expect.
type progress struct {
	out   io.Writer
	width int

	// Star lists are read several at a time, so every field behind the line
	// is written from more than one goroutine.
	mutex  sync.Mutex
	label  string
	frame  int
	drawn  bool
	lastAt time.Time
}

// spinner frames, one step per drawn page.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// minRedraw keeps a fast connection from repainting the line hundreds of times
// a second; page boundaries can arrive back to back from the HTTP cache.
const minRedraw = 80 * time.Millisecond

// newProgress returns a reporter writing to stderr, or a disabled one when
// stderr is not a terminal or the caller asked for quiet.
func newProgress(quiet bool) *progress {
	if quiet || !term.IsTerminal(os.Stderr) {
		return &progress{}
	}
	width, _, err := term.FromEnv().Size()
	if err != nil || width <= 0 {
		width = 80
	}
	return &progress{out: os.Stderr, width: width}
}

// stage names what the command is doing now and paints the line at once, so
// the user sees a step start even before its first page lands.
func (p *progress) stage(format string, args ...interface{}) {
	if p.out == nil {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.label = fmt.Sprintf(format, args...)
	p.lastAt = time.Now()
	p.paint(0, 0)
}

// update is the starlist.Client progress hook: how many items are in hand and
// how many the connection holds.
func (p *progress) update(fetched, total int) {
	if p.out == nil {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	// The last page of a step is always drawn, however fast it arrived: a
	// step that ends on 50/56 reads as if it gave up there.
	complete := total > 0 && fetched >= total
	now := time.Now()
	if p.drawn && !complete && now.Sub(p.lastAt) < minRedraw {
		return
	}
	p.lastAt = now
	p.paint(fetched, total)
}

// paint draws the line. The caller holds the mutex.
func (p *progress) paint(fetched, total int) {
	line := fmt.Sprintf("%s %s", spinnerFrames[p.frame%len(spinnerFrames)], p.label)
	p.frame++
	switch {
	case total > 0:
		line += fmt.Sprintf(" %d/%d", min(fetched, total), total)
	case fetched > 0:
		line += fmt.Sprintf(" %d", fetched)
	}
	// Truncate by display width: a wide character wrapping the line would
	// leave débris the next erase does not reach.
	fmt.Fprintf(p.out, "\r\033[K%s", text.Truncate(p.width-1, line))
	p.drawn = true
}

// done erases the line, leaving the terminal as it was found.
func (p *progress) done() {
	if p.out == nil {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if !p.drawn {
		return
	}
	fmt.Fprint(p.out, "\r\033[K")
	p.drawn = false
}
