package main

import (
	"slices"
	"testing"
)

func BenchmarkInsertMiddle(b *testing.B) {
	lines := make([]string, 1_000_000)
	for i := range lines {
		lines[i] = "This is the line of text"
	}
	for b.Loop() {
		lines = slices.Insert(lines, 500_000, "New insert")
		lines = slices.Delete(lines, 500_000, 500_001)
	}
}
