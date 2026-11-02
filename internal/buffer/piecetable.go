package buffer

import (
	"bytes"
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

	// Where each line starts in the document, worked out on demand.
	// Any edit marks it out of date, and the next read rebuilds it.
	lineStarts     []int
	lineIndexValid bool
}

func New(original []byte) *PieceTable {
	return &PieceTable{
		original: original,
		pieces: []piece{
			piece{source: fromOriginal, start: 0, len: len(original)},
		},
	}
}

// String gives back the whole document as one piece of text.
func (pt *PieceTable) String() string {
	var sb strings.Builder
	sb.Grow(pt.Len()) // one allocation, since we already know the final size
	for _, p := range pt.pieces {
		sb.Write(pt.bytesOf(p))
	}
	return sb.String()
}

// bytesOf hands back the actual bytes a piece points at.
func (pt *PieceTable) bytesOf(p piece) []byte {
	if p.source == fromOriginal {
		return pt.original[p.start : p.start+p.len]
	}
	return pt.add[p.start : p.start+p.len]
}

// Len is the size of the whole document in bytes.
func (pt *PieceTable) Len() int {
	total := 0
	for _, p := range pt.pieces {
		total += p.len
	}
	return total
}

// Insert puts new text into the document at the given position.
func (pt *PieceTable) Insert(offset int, text []byte) {
	// The text is about to change, so the line index is out of date.
	pt.lineIndexValid = false

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

// Delete removes `length` bytes from the document, starting at offset.
// The bytes stay in original/add; we just stop pointing at them.
func (pt *PieceTable) Delete(offset, length int) {
	// Deleting nothing is easy: do nothing.
	if length <= 0 {
		return
	}

	// Find the piece where the delete begins. If the offset is already
	// past the end of the document, there's nothing to delete.
	startIndex, startK := pt.findPiece(offset)
	if startIndex == len(pt.pieces) {
		return
	}

	// The text is about to change, so the line index is out of date.
	pt.lineIndexValid = false

	// Find the piece where the delete stops.
	endIndex, endK := pt.findPiece(offset + length)

	// Whatever survives the delete goes in here: at most a bit from
	// the start piece and a bit from the end piece.
	replacement := []piece{}

	// Keep the part of the start piece that comes before the delete,
	// unless the delete begins right at the start of that piece.
	first := pt.pieces[startIndex]
	if startK > 0 {
		replacement = append(replacement, piece{source: first.source, start: first.start, len: startK})
	}

	// If the delete stops inside a piece, keep the part after it.
	// If it runs to the very end of the document, there's no part after it.
	removeUntil := endIndex
	if endIndex < len(pt.pieces) {
		last := pt.pieces[endIndex]
		if last.len-endK > 0 {
			replacement = append(replacement, piece{source: last.source, start: last.start + endK, len: last.len - endK})
		}
		removeUntil = endIndex + 1
	}

	// Swap out every piece the delete touched, from the start piece
	// to the end piece, for whatever survived. Pieces fully inside
	// the delete simply vanish.
	pt.pieces = slices.Replace(pt.pieces, startIndex, removeUntil, replacement...)
}

// readRange copies out the document text between two offsets
// (from is included, to is not), touching only the pieces in that range.
func (pt *PieceTable) readRange(from, to int) string {
	var sb strings.Builder
	pos := 0
	for _, p := range pt.pieces {
		pieceEnd := pos + p.len

		// This piece ends before the range starts, so skip it.
		if pieceEnd <= from {
			pos = pieceEnd
			continue
		}
		// We've walked past the end of the range, so we're done.
		if pos >= to {
			break
		}

		// Take just the overlapping part of this piece.
		lo := max(from, pos) - pos
		hi := min(to, pieceEnd) - pos
		sb.Write(pt.bytesOf(p)[lo:hi])
		pos = pieceEnd
	}
	return sb.String()
}

// buildLineIndex scans the document once and records where every line starts.
func (pt *PieceTable) buildLineIndex() {
	pt.lineStarts = pt.lineStarts[:0] // reuse the old slice's memory
	pt.lineIndexValid = true

	total := pt.Len()
	if total == 0 {
		return // an empty document has no lines
	}

	pt.lineStarts = append(pt.lineStarts, 0) // the first line always starts at 0
	pos := 0
	for _, p := range pt.pieces {
		buf := pt.bytesOf(p)
		searched := 0
		for {
			// bytes.IndexByte is heavily optimised (it checks many bytes at once),
			// so this is much faster than looping one byte at a time.
			i := bytes.IndexByte(buf[searched:], '\n')
			if i < 0 {
				break
			}
			nextLineStart := pos + searched + i + 1
			// A newline at the very end of the file doesn't start a new line.
			if nextLineStart < total {
				pt.lineStarts = append(pt.lineStarts, nextLineStart)
			}
			searched += i + 1
		}
		pos += p.len
	}
}

// ensureLineIndex rebuilds the index only if an edit made it stale.
func (pt *PieceTable) ensureLineIndex() {
	if !pt.lineIndexValid {
		pt.buildLineIndex()
	}
}

// LineCount is how many lines the document has.
func (pt *PieceTable) LineCount() int {
	pt.ensureLineIndex()
	return len(pt.lineStarts)
}

// Line returns the text of line i, without its newline.
func (pt *PieceTable) Line(i int) string {
	pt.ensureLineIndex()
	if i < 0 || i >= len(pt.lineStarts) {
		return ""
	}

	// A line runs from its own start to where the next line starts.
	// The last line runs to the end of the document.
	start := pt.lineStarts[i]
	end := pt.Len()
	if i+1 < len(pt.lineStarts) {
		end = pt.lineStarts[i+1]
	}

	line := pt.readRange(start, end)
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r") // files saved on Windows end lines with \r\n
	return line
}
