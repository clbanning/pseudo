// pseudo_test.go - test cases.
// The principle test is C_source/pseudo.c#main()

package pseudo

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
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
	checkVals := map[string]int{ "1_2":5, "1_3":15, "2_4":5, "2_5":5, "3_4":5, "3_5":5, "4_6":15, "5_6":5}
	for k, v := range arcList {
		ck := strconv.Itoa(int(v.from.number))+"_"+strconv.Itoa(int(v.to.number))
		if vcap, ok := checkVals[ck]; !ok {
			fmt.Println("unknown ck:", ck)
			t.Fatal()
		} else if vcap != v.capacity {
			fmt.Println(k, "- want:", checkVals[ck], "got:", v.capacity)
			t.Fatal()
		}
	}
}

func TestRunCase1(t *testing.T) {
	PseudoCtx.LowestLabel = false
	PseudoCtx.FifoBucket = false

	results, err := Run("_data/dimacsMaxf.txt")
	if err != nil {
		t.Fatal(err)
	}

	fh, _ := os.Open("_data/dimacsMaxf.txt")
	defer fh.Close()
	input, _ := ioutil.ReadAll(fh)
	fmt.Println("input:")
	fmt.Println(string(input))

	for _, v := range results {
		fmt.Println(v)
	}

	fmt.Println("\nstats:", StatsJSON())
}
