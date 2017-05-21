// readwriter_test.go

package pseudo

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestReadWriter(t *testing.T) {
	s := NewSession(Context{})

	input, err := os.Open("_data/dimacsMaxf.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output := new(bytes.Buffer)

	if err = s.RunReadWriter(input, output); err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(output.Bytes()))
}
