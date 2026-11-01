package buffer

import (
	"slices"
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

// Insert puts new text into the document at the given position.
func (pt *PieceTable) Insert(offset int, text []byte) {
	// New text always goes at the end of the add buffer, no matter
	// where it shows up in the document. We note where it starts
	// so a piece can point at it.
	addStart := len(pt.add)
	pt.add = append(pt.add, text...)
	middle := piece{source: fromAdd, start: addStart, len: len(text)}

	// Figure out which piece the cursor is sitting in,
	// and how far into that piece it is.
	index, k := pt.findPiece(offset)

	// The cursor is past the last piece, so we're typing at the very
	// end of the file. Nothing to split, just tack the new piece on.
	if index == len(pt.pieces) {
		pt.pieces = append(pt.pieces, middle)
		return
	}

	// The cursor landed inside a piece, so we cut that piece in two:
	// everything before the cursor, and everything after it.
	old := pt.pieces[index]
	left := piece{source: old.source, start: old.start, len: k}
	right := piece{source: old.source, start: old.start + k, len: old.len - k}

	// Line them up: before, new text, after.
	// If the cursor was right at the start of the piece, there's nothing
	// "before", and if it was right at the end, there's nothing "after".
	// We leave those empty bits out instead of keeping useless pieces.
	replacement := []piece{}
	if left.len > 0 {
		replacement = append(replacement, left)
	}
	replacement = append(replacement, middle)
	if right.len > 0 {
		replacement = append(replacement, right)
	}

	// Swap the old piece out for the new ones.
	// Every other piece stays exactly where it was.
	pt.pieces = slices.Replace(pt.pieces, index, index+1, replacement...)
}

// findPiece returns the index of the piece containing offset,
// and k: how far into that piece the offset is.
func (pt *PieceTable) findPiece(offset int) (index int, k int) {
	pos := 0 // document offset where the current piece starts
	for i, p := range pt.pieces {
		if offset < (pos + p.len) {
			return i, offset - pos
		}
		pos += p.len // skip past this piece
	}
	return len(pt.pieces), 0 // offset is at the very end of the document
}
