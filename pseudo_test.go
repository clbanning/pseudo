// pseudo_test.go - test cases.
// The principle test is C_source/pseudo.c#main()

package pseudo

import (
	"fmt"
	"os"
	"testing"
	// "time"
)

/*
   cat dimacsMaxf.txt
   p max 6 8
   n 1 s
   n 6 t
   a 1 2 5
   a 1 3 15
   a 2 4 5
   a 2 5 5
   a 3 4 5
   a 3 5 5
   a 4 6 15
   a 5 6 5
*/
func TestReadDimacsFile(t *testing.T) {
	fh, err := os.Open("_data/dimacsMaxf.txt")
	if err != nil {
		t.Fatal(err)
	}
	err = ReadDimacsFile(fh)
	if err != nil {
		t.Fatal(err)
	}

	// check some allocations made by 'p' record
	if numNodes != uint(6) {
		fmt.Println("numNodes != 6 :", numNodes)
		t.Fatal()
	}
	if numArcs != uint(8) {
		fmt.Println("numArcs != 8 :", numArcs)
		t.Fatal()
	}
	if uint(len(adjacencyList)) != numNodes {
		fmt.Println("len(adjacencyList):", len(adjacencyList), "numNodes:", numNodes)
		t.Fatal()
	}
	if uint(len(strongRoots)) != numNodes {
		fmt.Println("len(strongRoots):", len(strongRoots), "numNodes:", numNodes)
		t.Fatal()
	}
	if uint(len(labelCount)) != numNodes {
		fmt.Println("len(labelCount):", len(labelCount), "numNodes:", numNodes)
		t.Fatal()
	}
	if uint(len(arcList)) != numArcs {
		fmt.Println("len(arcList):", len(arcList), "numNodes:", numNodes)
		t.Fatal()
	}

	// check values set by 'n' records
	if source != uint(1) {
		fmt.Println("source != 1 :", source)
		t.Fatal()
	}
	if sink != uint(6) {
		fmt.Println("sink != 6 :", source)
		t.Fatal()
	}

	// check arc record parsing
	checkVals := []uint{5, 15, 5, 5, 5, 5, 15, 5}
	for k, v := range arcList{
		if checkVals[k] != v.capacity {
			fmt.Println(k, "- want:", checkVals[k], "got:", v.capacity)
			t.Fatal()
		}
	}
}


func TestRun(t *testing.T) {
       PseudoCtx.LowestLabel = false
       PseudoCtx.FifoBucket = true

       results, err := Run("_data/dimacsMaxf.txt")
       if err != nil {
              t.Fatal(err)
       }

       fmt.Printf("Results = %v", results)
}
