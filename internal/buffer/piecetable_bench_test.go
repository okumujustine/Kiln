package buffer_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/okumujustine/kiln/internal/buffer"
)

// The file every benchmark works on: 1,000,000 lines of text.
const benchLines = 1_000_000

const benchLine = "This is the line of text"

// The middle of the file, as a line number and as a byte offset.
const (
	middleLine   = benchLines / 2
	middleOffset = middleLine * (len(benchLine) + 1) // +1 for each "\n"
)

// Baseline: the old []string approach. Insert a line in the middle,
// then delete it again so every run starts from the same file.
func BenchmarkSliceInsertMiddle(b *testing.B) {
	lines := make([]string, benchLines)
	for i := range lines {
		lines[i] = benchLine
	}
	for b.Loop() {
		lines = slices.Insert(lines, middleLine, "New insert")
		lines = slices.Delete(lines, middleLine, middleLine+1)
	}
}

func newBenchTable() *buffer.PieceTable {
	text := strings.Repeat(benchLine+"\n", benchLines)
	return buffer.New([]byte(text))
}

// The piece table on its own: insert a line in the middle, then delete it.
// No bytes of the 25 MB file move; only the piece list changes.
func BenchmarkPieceTableInsertMiddle(b *testing.B) {
	pt := newBenchTable()
	text := []byte("New insert\n")
	for b.Loop() {
		pt.Insert(middleOffset, text)
		pt.Delete(middleOffset, len(text))
	}
}

// The piece table as the editor uses it: after each edit the screen is
// redrawn, which asks for LineCount and so rebuilds the line index.
func BenchmarkPieceTableInsertMiddleAndRedraw(b *testing.B) {
	pt := newBenchTable()
	text := []byte("New insert\n")
	for b.Loop() {
		pt.Insert(middleOffset, text)
		pt.LineCount()
		pt.Delete(middleOffset, len(text))
		pt.LineCount()
	}
}

// For scale: how long one full scan for newlines over the 25 MB file takes.
func BenchmarkScanNewlines(b *testing.B) {
	data := []byte(strings.Repeat(benchLine+"\n", benchLines))
	for b.Loop() {
		bytes.Count(data, []byte{'\n'})
	}
}
