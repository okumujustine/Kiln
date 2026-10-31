package buffer_test

import (
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
