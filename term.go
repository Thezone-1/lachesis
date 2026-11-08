package main

import (
	"fmt"
	"io"
	"os"
)

// isTerminal reports whether f is an interactive terminal (no dependencies).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// liveView redraws a block of lines in place.
type liveView struct {
	w     io.Writer
	lines int
}

func newLiveView(w io.Writer) *liveView {
	fmt.Fprint(w, "\x1b[?25l") // hide cursor
	return &liveView{w: w}
}

func (v *liveView) draw(lines []string) {
	if v.lines > 0 {
		fmt.Fprintf(v.w, "\x1b[%dA", v.lines)
	}
	for _, l := range lines {
		fmt.Fprintf(v.w, "\r\x1b[2K%s\n", l)
	}
	v.lines = len(lines)
}

func (v *liveView) close() {
	fmt.Fprint(v.w, "\x1b[?25h") // restore cursor
}
