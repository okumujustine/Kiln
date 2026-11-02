package buffer_test

import (
	"slices"
	"testing"

	"github.com/okumujustine/kiln/internal/buffer"
	"github.com/stretchr/testify/assert"
)

func TestPieceTable_String(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
		want  string
	}{
		{name: "Normal Text", input: []byte("ab\ncd\nef\n"), want: "ab\ncd\nef\n"},
		{name: "Empty Byte", input: []byte(""), want: ""},
		{name: "Nil Input", input: nil, want: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := buffer.New(testCase.input).String()
			assert.Equal(t, testCase.want, got)
		})

	}

}

func TestPieceTable_Insert(t *testing.T) {
	testCases := []struct {
		name  string
		index int
		input []byte
		want  string
	}{
		{name: "Adds 'X' at index 1", index: 1, input: []byte("X"), want: "aXb\ncd\nef\n"},
		{name: "Adds 'XYZ' at index 4", index: 4, input: []byte("XYZ"), want: "ab\ncXYZd\nef\n"},
	}

	for _, testCase := range testCases {
		pt := buffer.New([]byte("ab\ncd\nef\n"))
		pt.Insert(testCase.index, testCase.input)

		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, pt.String(), testCase.want)
		})
	}
}

func TestPieceTable_InsertMultiple(t *testing.T) {
	type insert struct {
		offset int
		text   string
	}

	testCases := []struct {
		name    string
		inserts []insert
		want    string
	}{
		{
			name:    "two inserts in different pieces",
			inserts: []insert{{1, "X"}, {8, "Z"}},
			want:    "aXb\ncd\neZf\n",
		},
		{
			name:    "insert at end of document",
			inserts: []insert{{9, "Z"}},
			want:    "ab\ncd\nef\nZ",
		},
		{
			name:    "insert at start of document",
			inserts: []insert{{0, "Z"}},
			want:    "Zab\ncd\nef\n",
		},
		{
			name:    "insert at a piece boundary",
			inserts: []insert{{1, "X"}, {2, "Y"}},
			want:    "aXYb\ncd\nef\n",
		},
		{
			name:    "insert before an earlier insert",
			inserts: []insert{{4, "X"}, {1, "Y"}},
			want:    "aYb\ncXd\nef\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pt := buffer.New([]byte("ab\ncd\nef\n"))
			for _, ins := range testCase.inserts {
				pt.Insert(ins.offset, []byte(ins.text))
			}
			assert.Equal(t, testCase.want, pt.String())
		})
	}
}

func TestPieceTable_Delete(t *testing.T) {
	type op struct {
		insert bool // true = Insert, false = Delete
		offset int
		length int    // for Delete
		text   string // for Insert
	}

	// Builds the 6-piece state we traced by hand: "aXb\nc\neZf\n"
	sixPieces := []op{
		{insert: true, offset: 1, text: "X"},
		{insert: true, offset: 8, text: "Z"},
		{offset: 5, length: 1},
	}
	with := func(extra op) []op { return append(slices.Clone(sixPieces), extra) }

	testCases := []struct {
		name string
		ops  []op
		want string
	}{
		{"inside one piece", []op{{offset: 3, length: 2}}, "ab\n\nef\n"},
		{"from the start", []op{{offset: 0, length: 3}}, "cd\nef\n"},
		{"to the end", []op{{offset: 6, length: 3}}, "ab\ncd\n"},
		{"everything", []op{{offset: 0, length: 9}}, ""},
		{"across pieces", []op{{insert: true, offset: 1, text: "X"}, {offset: 0, length: 4}}, "cd\nef\n"},
		{"build six pieces", sixPieces, "aXb\nc\neZf\n"},
		{"practice: Delete(2, 6)", with(op{offset: 2, length: 6}), "aXf\n"},
		{"practice: Delete(0, 2)", with(op{offset: 0, length: 2}), "b\nc\neZf\n"},
		{"practice: Delete(8, 2)", with(op{offset: 8, length: 2}), "aXb\nc\neZ"},
		{"zero length does nothing", []op{{offset: 3, length: 0}}, "ab\ncd\nef\n"},
		{"past the end does nothing", []op{{offset: 9, length: 2}}, "ab\ncd\nef\n"},
		{"delete then re-insert", []op{{offset: 2, length: 1}, {insert: true, offset: 2, text: "\n"}}, "ab\ncd\nef\n"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pt := buffer.New([]byte("ab\ncd\nef\n"))
			for _, o := range testCase.ops {
				if o.insert {
					pt.Insert(o.offset, []byte(o.text))
				} else {
					pt.Delete(o.offset, o.length)
				}
			}
			assert.Equal(t, testCase.want, pt.String())
		})
	}
}

func TestPieceTable_Lines(t *testing.T) {
	testCases := []struct {
		name  string
		build func() *buffer.PieceTable
		want  []string
	}{
		{"plain file", func() *buffer.PieceTable {
			return buffer.New([]byte("ab\ncd\nef\n"))
		}, []string{"ab", "cd", "ef"}},
		{"no newline at the end", func() *buffer.PieceTable {
			return buffer.New([]byte("ab\ncd"))
		}, []string{"ab", "cd"}},
		{"empty file", func() *buffer.PieceTable {
			return buffer.New(nil)
		}, nil},
		{"blank line in the middle", func() *buffer.PieceTable {
			return buffer.New([]byte("ab\n\ncd\n"))
		}, []string{"ab", "", "cd"}},
		{"windows line endings", func() *buffer.PieceTable {
			return buffer.New([]byte("ab\r\ncd\r\n"))
		}, []string{"ab", "cd"}},
		{"a line made of several pieces", func() *buffer.PieceTable {
			pt := buffer.New([]byte("ab\ncd\nef\n"))
			pt.Insert(1, []byte("X"))
			pt.Insert(8, []byte("Z"))
			return pt
		}, []string{"aXb", "cd", "eZf"}},
		{"inserting a newline splits a line", func() *buffer.PieceTable {
			pt := buffer.New([]byte("ab\ncd\n"))
			pt.Insert(1, []byte("\n"))
			return pt
		}, []string{"a", "b", "cd"}},
		{"deleting a newline joins lines", func() *buffer.PieceTable {
			pt := buffer.New([]byte("ab\ncd\n"))
			pt.LineCount() // build the index first, so the delete has to invalidate it
			pt.Delete(2, 1)
			return pt
		}, []string{"abcd"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pt := testCase.build()
			var got []string
			for i := 0; i < pt.LineCount(); i++ {
				got = append(got, pt.Line(i))
			}
			assert.Equal(t, testCase.want, got)
		})
	}
}
