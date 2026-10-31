package buffer

import (
	"strings"
)

type source uint8

const (
	fromOriginal source = iota
	fromAdd
)

type piece struct {
	source source
	start  int
	len    int
}

type PieceTable struct {
	original []byte
	add      []byte
	pieces   []piece
}

func New(original []byte) *PieceTable {
	return &PieceTable{
		original: original,
		pieces: []piece{
			piece{source: fromOriginal, start: 0, len: len(original)},
		},
	}
}

func (pt *PieceTable) String() string {
	var sb strings.Builder

	for _, p := range pt.pieces {
		if p.source == fromOriginal {
			sb.Write(pt.original[p.start : p.start+p.len])
		}

		if p.source == fromAdd {
			sb.Write(pt.add[p.start : p.start+p.len])
		}
	}

	return sb.String()
}

// Insert adds text at the given byte offset.
func (pt *PieceTable) Insert(offset int, text []byte) {
	originalLength := len(pt.original)

	// Store the new text at the end of the add buffer.
	addStart := len(pt.add) // where this text will begin inside add
	pt.add = append(pt.add, text...)

	// Build the three pieces.
	left := piece{source: fromOriginal, start: 0, len: offset}
	middle := piece{source: fromAdd, start: addStart, len: len(text)}
	right := piece{source: fromOriginal, start: offset, len: originalLength - offset}

	// Replace the old single piece with the three new ones.
	pt.pieces = []piece{left, middle, right}
}
