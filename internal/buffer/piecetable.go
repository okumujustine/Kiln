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

	// size is the document length in bytes, kept up to date on every edit
	// so we never have to add up all the pieces to find it.
	size int

	// lineBreaks holds the offset just after every "\n" in the document,
	// in order. Line 0 starts at 0, and line i starts at lineBreaks[i-1].
	// It's built the first time someone asks for lines, and from then on
	// every Insert and Delete patches it instead of rebuilding it.
	lineBreaks     []int
	lineIndexValid bool
}

func New(original []byte) *PieceTable {
	return &PieceTable{
		original: original,
		size:     len(original),
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
	return pt.size
}

// Insert puts new text into the document at the given position.
func (pt *PieceTable) Insert(offset int, text []byte) {
	// Patch the line index and the size before the pieces change.
	offset = min(offset, pt.size) // inserting past the end means inserting at the end
	pt.updateLinesForInsert(offset, text)
	pt.size += len(text)

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

	// A delete can't run past the end of the document.
	length = min(length, pt.size-offset)

	// Patch the line index and the size before the pieces change.
	pt.updateLinesForDelete(offset, length)
	pt.size -= length

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

// buildLineIndex scans the document once and records every line break.
// It only runs the first time lines are needed; after that, edits keep
// the index up to date themselves.
func (pt *PieceTable) buildLineIndex() {
	pt.lineBreaks = pt.lineBreaks[:0] // reuse the old slice's memory
	pt.lineIndexValid = true

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
			pt.lineBreaks = append(pt.lineBreaks, pos+searched+i+1)
			searched += i + 1
		}
		pos += p.len
	}
}

// ensureLineIndex builds the index if nobody has asked for lines yet.
func (pt *PieceTable) ensureLineIndex() {
	if !pt.lineIndexValid {
		pt.buildLineIndex()
	}
}

// firstBreakAfter finds the first line break that comes after offset.
// lineBreaks is sorted, so a binary search finds it in about 20 steps
// even for a million lines.
func (pt *PieceTable) firstBreakAfter(offset int) int {
	i, found := slices.BinarySearch(pt.lineBreaks, offset)
	if found {
		i++ // a break exactly at offset is "at", not "after"
	}
	return i
}

// updateLinesForInsert patches the line index for text inserted at offset.
func (pt *PieceTable) updateLinesForInsert(offset int, text []byte) {
	if !pt.lineIndexValid {
		return // no index yet; it will be built from scratch when needed
	}

	// Breaks at or before the insert point don't move.
	// Every break after it moves right by the length of the new text.
	i := pt.firstBreakAfter(offset)
	for j := i; j < len(pt.lineBreaks); j++ {
		pt.lineBreaks[j] += len(text)
	}

	// Any newlines inside the new text are new line breaks.
	// They all land between the breaks before and after the insert point.
	var added []int
	searched := 0
	for {
		n := bytes.IndexByte(text[searched:], '\n')
		if n < 0 {
			break
		}
		added = append(added, offset+searched+n+1)
		searched += n + 1
	}
	pt.lineBreaks = slices.Insert(pt.lineBreaks, i, added...)
}

// updateLinesForDelete patches the line index for length bytes removed at offset.
func (pt *PieceTable) updateLinesForDelete(offset, length int) {
	if !pt.lineIndexValid {
		return
	}

	// Breaks that came from a newline inside the deleted range disappear.
	// A newline at position p gives a break at p+1, so those are the
	// breaks after offset and up to offset+length.
	from := pt.firstBreakAfter(offset)
	to := pt.firstBreakAfter(offset + length)
	pt.lineBreaks = slices.Delete(pt.lineBreaks, from, to)

	// Every break after the deleted range moves left by its length.
	for j := from; j < len(pt.lineBreaks); j++ {
		pt.lineBreaks[j] -= length
	}
}

// LineCount is how many lines the document has.
func (pt *PieceTable) LineCount() int {
	pt.ensureLineIndex()
	if pt.size == 0 {
		return 0 // an empty document has no lines
	}
	count := len(pt.lineBreaks) + 1
	// A newline at the very end of the file doesn't start a new line.
	if len(pt.lineBreaks) > 0 && pt.lineBreaks[len(pt.lineBreaks)-1] == pt.size {
		count--
	}
	return count
}

// Line returns the text of line i, without its newline.
func (pt *PieceTable) Line(i int) string {
	if i < 0 || i >= pt.LineCount() {
		return ""
	}

	// Line 0 starts at the top; every other line starts just after a break.
	// A line ends where the next one starts, or at the end of the document.
	start := 0
	if i > 0 {
		start = pt.lineBreaks[i-1]
	}
	end := pt.size
	if i < len(pt.lineBreaks) {
		end = pt.lineBreaks[i]
	}

	line := pt.readRange(start, end)
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r") // files saved on Windows end lines with \r\n
	return line
}
