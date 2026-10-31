package buffer

import "strings"

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
