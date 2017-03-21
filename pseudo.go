// pseudo.go implements pseudo3.23.

// NOTES:
// 1. Input is from stdin - c_src#readDimacsFileCreateList.
//    This looks a little cludgy.  main()/Testxxx() should pass in a file
//    handle that may be os.Stdin.
// 2. In RecoverFlow() use gap value based on pseudoCtx.Lowestlabel value.
// 3. All timing/profiling is out in main()/Testxxx - so don't include in this package.
// 4. main() in C source code is really just a test ... implement in pseudo_test.go.

package pseudo

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// global variables
var lowestStrongLabel uint
var highestStrongLabel uint
var adjacencyList []*node
var strongRoots []*root
var arcList []*arc
var labelCount []uint
var numNodes, numArcs, source, sink uint

// local context

type context struct {
	DisplayCut  bool
	DisplayFlow bool
	LowestLabel bool
	FifoBucket  bool
}

// necessary initialization
func init() {
	labelCount = make([]uint, 0)
}

// the arc object

type arc struct {
	from      *node
	to        *node
	flow      uint
	capacity  uint
	direction uint
}

// Initialize a new arc value.
// in-line
// func newArc() *arc {
// 	return &arc{direction: 1}
// }

// the node object

type node struct {
	visited         uint
	numAdjacent     uint
	number          uint
	label           uint
	excess          int
	parent          *node
	childList       *node
	nextScan        *node
	numberOutOfTree uint
	outOfTree       []*arc // was **Arc in C, looking at CreateOutOfTree, we're dealing with a pool of Arc's
	nextarc         uint
	arcToParent     *arc
	next            *node
}

// Newnode returns an initialized node value.
// in-line
// func newNode(n uint) *node {
// 	var u uint
// 	labelCount = append(labelCount, u)
// 	return &node{number: n}
// }

func (n *node) liftAll() {
	var temp *node
	var current = n

	current.nextScan = current.childList
	labelCount[current.label]--
	current.label = numNodes

	for ; current != nil; current = current.parent {
		for current.nextScan != nil {
			temp = current.nextScan
			current.nextScan = current.nextScan.next
			current = temp
			current.nextScan = current.childList

			labelCount[current.label]--
			current.label = numNodes
		}
	}
}

// createOutOfTree allocates arc's for adjacent nodes.
func (n *node) createOutOfTree() {
	n.outOfTree = make([]*arc, n.numAdjacent) // OK if '0' are allocated
}

// addOutOfTreenode
func (n *node) addOutOfTreeNode(out *arc) {
	n.outOfTree[n.numberOutOfTree] = out
	n.numberOutOfTree++
}

// the root object

type root struct {
	start *node
	end   *node
}

//  newRoot is a wrapper on new(root) to mimic source.
// in-line
// func newRoot() *root {
// 	return new(root)
// }

// free reinitializes a root value.
func (r *root) free() {
	r.start = nil
	r.end = nil
}

// addToStrongBucket may be better as a *node method ... need to see usage elsewhere.
func (r *root) addToStrongBucket(n *node) {
	if pseudoCtx.FifoBucket {
		if r.start != nil {
			r.end.next = n
			r.end = n
			n.next = nil
		} else {
			r.start = n
			r.end = n
			n.next = nil
		}
	} else {
		n.next = r.start
		r.start = n
		return
	}
}

// ================ public functions =====================

// ReadDimacsFile reads the input and creates list.
func ReadDimacsFile(fh *os.File) error {
	var i, capacity, numLines, from, to, first, last uint
	var word []byte
	var ch, ch1 byte

	buf := bufio.NewReader(fh)
	var atEOF bool
	for {
		if atEOF {
			break
		}

		line, err := buf.ReadBytes('\n')
		if err != io.EOF {
			return err
		} else if err == io.EOF {
			// ... at EOF with data but no '\n' line termination.
			// While not necessary for os.STDIN; it can happen in a file.
			atEOF = true
		} else {
			// Strip of EOL.
			line = line[:len(line)-1]
		}
		numLines++

		switch line[0] {
		case 'p':
			if _, err := fmt.Sscanf(string(line), "%v %s %d %d", &ch, word, &numNodes, &numArcs); err != nil {
				return err
			}

			adjacencyList = make([]*node, numNodes)
			strongRoots = make([]*root, numNodes)
			labelCount = make([]uint, numNodes)
			arcList = make([]*arc, numArcs)

			var i uint
			for i = 0; i < numNodes; i++ {
				// in-line: strongRoots[i] = newRoot()
				strongRoots[i] = new(root)
				// in-line: adjacencyList[i] = &newNode(i + 1)
				adjacencyList[i] = &node{number: i + 1}
				var u uint
				labelCount = append(labelCount, u)
			}
			for i = 0; i < numArcs; i++ {
				// in-line: arcList[i] = newArc()
				arcList[i] = &arc{direction: 1}
			}
			first = 0
			last = numArcs - 1
		case 'a':
			if _, err := fmt.Scanf(string(line), "%v %d %d %d", &ch, &from, &to, &capacity); err != nil {
				return err
			}
			if (from+to)%2 != 0 {
				arcList[first].from = adjacencyList[from-1]
				arcList[first].to = adjacencyList[to-1]
				arcList[first].capacity = capacity
				first++
			} else {
				arcList[last].from = adjacencyList[from-1]
				arcList[last].to = adjacencyList[to-1]
				arcList[last].capacity = capacity
				last--
			}

			adjacencyList[from-1].numAdjacent++
			adjacencyList[to-1].numAdjacent++
		case 'n':
			if _, err := fmt.Scanf(string(line), "%v  %d %v", &ch, &i, &ch1); err != nil {
				return err
			}
			if ch1 == 's' {
				source = i
			} else if ch1 == 't' {
				sink = i
			} else {
				return fmt.Errorf("Unrecognized character %v on line %d\n", ch1, numLines)
			}
		}
	}

	for i = 0; i < numNodes; i++ {
		adjacencyList[i].createOutOfTree()
	}

	for i = 0; i < numArcs; i++ {
		to = arcList[i].to.number
		from = arcList[i].from.number
		capacity = arcList[i].capacity

		if !(source == to || sink == from || from == to) {
			if source == from && to == sink {
				arcList[i].flow = capacity
			} else if from == source || to != sink {
				adjacencyList[from-1].addOutOfTreeNode(arcList[i])
			} else if to == sink {
				adjacencyList[to-1].addOutOfTreeNode(arcList[i])
			} else {
				adjacencyList[from-1].addOutOfTreeNode(arcList[i])
			}
		}
	}

	return nil
}

func SimpleInitialization() {
}

func PseudoFlowPhaseOne() {
}

// RecoverFlow - internalize setting 'gap' value.
func RecoverFlow() {
}
