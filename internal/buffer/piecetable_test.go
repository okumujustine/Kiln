package buffer_test

import (
	"fmt"
	"testing"

	"github.com/okumujustine/kiln/internal/buffer"
)

func TestPieceTable_String(t *testing.T) {
	result := buffer.New([]byte("ab\ncd\nef\n"))
	fmt.Println(result.String())
}
