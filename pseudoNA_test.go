package pseudo

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestRunNAWriter(t *testing.T) {
	s := NewSession(Context{})

	fh, err := os.Open("_data/dimacsMaxf.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	numNodes, numArcs, n, a, err := ParseDimacsReader(fh)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err = s.RunNAWriter(numNodes, numArcs, n, a, &buf); err != nil {
		t.Fatal(err)
	}
	result := string(buf.Bytes())
	if checkNAWriter != result {
		fmt.Println("want:\n", checkNAWriter)
		fmt.Println("got:\n", result)
		t.Fatal()
	}
}

var checkNAWriter = `c Dimacs-format maximum flow result generated by pseudo.go
c 
c Optimal flow using  Hochbaum's PseudoFlow algorithm
c 
c Runtime Configuration -
c Highest label pseudoflow algorithm
c Using LIFO buckets
c 
c Solution checks as feasible
c 
c Solution checks as optimal
c 
c Solution
s 15
c 
c SRC DST FLOW
f 1 2 5
f 2 5 0
f 3 4 5
f 5 6 5
f 4 6 10
f 3 5 5
f 2 4 5
f 1 3 10
`

