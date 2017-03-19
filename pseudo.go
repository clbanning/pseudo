// pseudo.go implements pseudo3.23 support functions.
// See pseudo/cmd for CLI app.

// NOTES:
// 1. Input is from stdin - c_src#readDimacsFileCreateList.
//    This looks a little cludgy.  main() should pass in a file
//    handle that may be os.Stdin.
// 2. Replace timer logic with more convential profiling logic.
// 3. In RecoverFlow() use gap value based on pseudoCtx.Lowestlabel value.
// 4. All timing/profiling is out in main() - so don't include in this package.

package pseudo

import (
	"fmt"
	"os"
)

// global variables
var numnodes uint
var numarcs uint
var source uint
var sink uint
var lowestStrongLabel uint
var highestStrongLabel uint
var adjacencyList *node
var strongroots *root
var labelCount []uint // index'd to len(nodes), grows with Newnode
var arcList *arc

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
func newarc() *arc {
	return &arc{direction: 1}
}

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
func newNode(n uint) *node {
	labelCount = append(labelCount, uint{})
	return &node{number: n}
}

func (n *node) liftAll() {
	var temp *node
	var current = n

	current.nextScan = current.childList
	labelCount[current.label]--
	current.label = numnodes

	for ; current != nil; current = current.parent {
		for current.nextScan != nil {
			temp = current.nextScan
			current.nextScan = current.nextScan.next
			current = temp
			current.nextScan = current.childList

			labelCount[current.label]--
			current.label = numnodes
		}
	}
}

// createOutOfTree allocates arc's for adjacent nodes.
func (n *node) createOutOfTree() {
	n.outOfTree = make([]*arc, n.numAdjacent) // OK if '0' are allocated
	// runtime handles mallocs and panics on OOM; you'll get a stack trace
	/*
		if (nd->numAdjacent)
			if ((nd->outOfTree = (arc **) malloc (nd->numAdjacent * sizeof (Arc *))) == NULL)
			{
				printf ("%s Line %d: Out of memory\n", __FILE__, __LINE__);
				exit (1);
			}
		}
	*/
}

// addOutOfTreenode
func (n *node) addOutOfTreenode(out *arc) {
	n.outOfTree[n.numOutOfTree] = out
	n.numOutOfTree++
}

// the root object

type root struct {
	start *node
	end   *node
}

//  Newroot is a wrapper on new(root) to mimic source.
func newRoot() *root {
	return new(root)
}

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
	return nil
}

func SimpleInitialization() {
}

func PseudoFlowPhaseOne() {
}

// RecoverFlow - internalize setting 'gap' value.
func RecoverFlow() {
}
