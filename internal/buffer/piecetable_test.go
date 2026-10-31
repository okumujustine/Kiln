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
		{name: "Nil Inpu", input: nil, want: ""},
	}

	for _, testCase := range testCases {
		assert.Equal(t, testCase.want, buffer.New(testCase.input).String())
	}

}
