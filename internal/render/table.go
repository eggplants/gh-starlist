package render

import (
	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/cli/go-gh/v2/pkg/term"
)

// NewTable builds a table printer that renders columns on a terminal and
// tab separated values when the output is piped.
func NewTable() tableprinter.TablePrinter {
	terminal := term.FromEnv()
	width, _, err := terminal.Size()
	if err != nil {
		width = 80
	}
	return tableprinter.New(terminal.Out(), terminal.IsTerminalOutput(), width)
}
