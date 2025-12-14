package output

import (
	"fmt"
	"io"
	"os"
)

// Splog provides structured logging and output
type Splog struct {
	writer io.Writer
}

// NewSplog creates a new splog instance
func NewSplog() *Splog {
	return &Splog{
		writer: os.Stdout,
	}
}

// Info writes an info message
func (s *Splog) Info(format string, args ...interface{}) {
	fmt.Fprintf(s.writer, format+"\n", args...)
}

// Page writes output that should be paged (for now, just print)
func (s *Splog) Page(content string) {
	fmt.Fprint(s.writer, content)
}

// Newline writes a newline
func (s *Splog) Newline() {
	fmt.Fprintln(s.writer)
}
