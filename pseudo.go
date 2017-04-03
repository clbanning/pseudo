// pseudo.go implements pseudo3.23.
// MIT license in accompanying LICENSE file.

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
	"encoding/json"
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

// Context provides optional switches that can be set using Config.
type Context struct {
	DisplayCut  bool
	DisplayFlow bool
	LowestLabel bool
	FifoBucket  bool
	// Stats       bool // always collect stats, reporting just requires call to StatsJSON
}

type statistics struct {
	NumPushes   uint `json:"numPushes"`
	NumMergers  uint `json:"numMergers"`
	NumRelabels uint `json:"numRelabels"`
	NumGaps     uint `json:"numGaps"`
	NumArcScans uint `json:"numArcScans"`
}

var stats statistics

// StatsJSON returns the runtime stats as a JSON object.
func StatsJSON() string {
	j, _ := json.Marshal(stats)
	return string(j)
}

// necessary initialization
func init() {
	labelCount = make([]uint, 0)
}

// ==================== the arc object
type arc struct {
	from      *node
	to        *node
	flow      uint
	capacity  uint
	direction uint
}

// (*arc) pushUpward
// static inline void
func (a *arc) pushUpward(child *node, parent *node, resCap uint) {

	stats.NumPushes++
	// Invalid operation because resCap is type uint and child.excess is as int hence changed reCap to int
	if resCap >= child.excess {
		parent.excess += child.excess
		a.flow += child.excess
		child.excess = 0
		return
	}

	a.direction = 0
	parent.excess += resCap // int and uint
	child.excess -= resCap  // int and uint
	a.flow = a.capacity
	parent.outOfTree[parent.numberOutOfTree] = a
	parent.numberOutOfTree++
	//breakRelationship(parent, child) in c source
	parent.breakRelationship(child)
	if pseudoCtx.LowestLabel {
		lowestStrongLabel = child.label
	}

	//addToStrongBucket(child, &strongRoots[child.label])
	// CLB: note that strongRoots is []*root, so strongRoot[i] is *root.
	child.addToStrongBucket(strongRoots[child.label]) // cannot use type **root as *root hence changed func
}

// (*arc) pushDownward
//static inline void
func (a *arc) pushDownward(child *node, parent *node, flow uint) {

	stats.NumPushes++

	if flow >= child.excess {
		parent.excess += child.excess
		a.flow = child.excess
		child.excess = 0
	}

	a.direction = 1
	child.excess -= flow
	parent.excess += flow
	a.flow = 0
	parent.outOfTree[parent.numberOutOfTree] = a
	parent.numberOutOfTree++
	//breakRelationship(parent, child) in c source
	parent.breakRelationship(child)
	if pseudoCtx.LowestLabel {
		lowestStrongLabel = child.label
	}

	//addToStrongBucket(child, &strongRoots[child.label])
	child.addToStrongBucket(strongRoots[child.label]) // cannot use type **root as *root, changed
	// declaration of addToStrongBucket to **root
}

//Initialize a new arc value.
//in-lined
//func newArc() *arc {
//	return &arc{direction: 1}
//}

// ==================== the node object
type node struct {
	visited         uint
	numAdjacent     uint
	number          uint
	label           uint
	excess          uint
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
// in-lined
// func newNode(n uint) *node {
// 	var u uint
// 	labelCount = append(labelCount, u)
// 	return &node{number: n}
// }

// (*node) liftAll
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

// (*node) createOutOfTree allocates arc's for adjacent nodes.
func (n *node) createOutOfTree() {
	n.outOfTree = make([]*arc, n.numAdjacent) // OK if '0' are allocated
}

// (*node) addOutOfTreenode
func (n *node) addOutOfTreeNode(out *arc) {
	n.outOfTree[n.numberOutOfTree] = out
	n.numberOutOfTree++
}

// (*node) processRoot. 'n' is 'strongRoot' in C source
func (n *node) processRoot() {
	var temp, weakNode *node
	var out *arc
	strongNode := n
	n.nextScan = n.childList

	if out, weakNode = n.findWeakNode(); out != nil {
		weakNode.merge(n, out)
		n.pushExcess()
		return
	}

	n.checkChildren()

	for strongNode != nil {
		for strongNode.nextScan != nil {
			temp = strongNode.nextScan
			strongNode.nextScan = strongNode.nextScan.next
			strongNode = temp
			strongNode.nextScan = strongNode.childList

			if out, weakNode = findWeakNode(); out != nil {
				weakNode.merge(strongNode, out)
				n.pushExcess()
				return
			}

			strongNode.checkChildren()
		}

		if strongNode = strongNode.parent; strongNode != nil {
			strongNode.checkChildren()
		}
	}

	// CLB: note that strongRoots is []*root, so strongRoot[i] is *root.
	n.addToStrongBucket(strongRoots[strongRoot.label])

	if !pseudoCtx.LowestLabel {
		highestStrongLabel++
	}
}

// static void
// merge (Node *parent, Node *child, Arc *newArc)
// (*node) merge. 'n' is 'parent' in C source.
func (n *node) merge(child *node, newArc *arc) {
	var oldArc *arc
	var oldParent *node
	current := child
	newParent := n

	stats.NumMergers++ // unlike C source always calc stats

	for current != nil {
		oldArc = current.arcToParent
		current.arcToParent = newArc
		oldParent = current.parent
		oldParent.breakRelationship(current)
		newParent.addRelationship(current)

		newParent = current
		current = oldParent
		newArc = oldArc
		newArc.direction = 1 - newArc.direction
	}

	current.arcToParent = newArc
	newParent.addRelationship(current)
}

// static void
// pushExcess (Node *strongRoot)
// (*node) pushExcess. 'n' is 'strongRoot' in C source.
func (n *node) pushExcess() {
	var current, parent *node
	var arcToParent *arc
	prevEx := 1

	for current = n; current.excess > 0 && current.parent != nil; current = parent {
		parent = current.parent
		prevEx = parent.excess

		arcToParent = current.arcToParent

		if arcToParent.direction > 0 {
			arcToParent.pushUpward(current, parent, arcToParent.capacity-arcToParent.flow)
		} else {
			arcToParent.pushDownward(current, parent, arcToParent.flow)
		}
	}

	if current.excess > 0 && prevEx <= 0 {
		if pseudoCtx.LowestLabel {
			lowestStrongLabel = current.label
		}
		// CLB: note that strongRoots is []*root, so strongRoot[i] is *root.
		current.addToStrongBucket(strongRoots[current.label]) //type *node does not support indexing was ns[current.label]
	}
}

// static inline void
// breakRelationship (Node *oldParent, Node *child)
// (*node) breakRelationship
func (n *node) breakRelationship(child *node) {
	var current *node
	child.parent = nil

	if n.childList == child {
		n.childList = child.next
		child.next = nil
		return
	}

	for current = n.childList; current.next != child; current = current.next {
		current.next = child.next
		child.next = nil
	}
}

// static inline int
// addRelationship (Node *newParent, Node *child)
// (*node) addRelationship
// CLB: implement as static void function, calling code ignores return value
func (n *node) addRelationship(child *node) {
	child.parent = n
	child.next = n.childList
	n.childList = child
}

// static Arc *
// findWeakNode (Node *strongNode, Node **weakNode)
// (*node) findWeakNode() (*arc, weakNode *node)
// CLB: avoid pointer-to-pointer handling by also returning computed weakNode
func (n *node) findWeakNode() (*arc, *node) {
	var i, size uint
	var out *arc
	var weakNode *node

	size = n.numberOutOfTree

	for i = n.nextarc; i < size; i++ {
		stats.NumArcScans++
		if pseudoCtx.LowestLabel {
			if n.outOfTree[i].to.label == lowestStrongLabel-1 {
				//TODO CHECK SECTION
				n.nextarc = i
				out = n.outOfTree[i]
				weakNode = out.to
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
			if n.outOfTree[i].from.label == (lowestStrongLabel - 1) {
				n.nextarc = i
				out = n.outOfTree[i]
				weakNode = out.from
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
		} else {
			if n.outOfTree[i].to.label == (highestStrongLabel - 1) {
				n.nextarc = i
				out = n.outOfTree[i]
				weakNode = out.to
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
			if n.outOfTree[i].from.label == (highestStrongLabel - 1) {
				n.nextarc = i
				out = n.outOfTree[i]
				weakNode = out.from
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
		}

	}

	n.nextarc = n.numberOutOfTree
	return nil, nil

}

// (*node) checkChildren
func (n *node) checkChildren() {
	for ; n.nextScan != nil; n.nextScan = n.nextScan.next {
		if n.nextScan.label == n.label {
			return
		}
	}

	labelCount[n.label]--
	n.label++
	labelCount[n.label]++
	// Always collect stats
	stats.NumRelabels++
	n.nextarc = 0
}

func (n *node) addToStrongBucket(rootBucket *root) {
	if pseudoCtx.FifoBucket {
		if rootBucket.start != nil {
			rootBucket.end.next = n
			rootBucket.end = n
			n.next = nil
		} else {
			rootBucket.start = n
			rootBucket.end = n
			n.next = nil
		}
	} else {
		n.next = rootBucket.start
		rootBucket.start = n
		return
	}
}

// static void
// sort (Node * current)
func (n *node) sort() {
	if n.numOutOfTree > 1 {
		quickSort(n.outOfTree, 0, n.numOutOfTree-1)
	}
}

// static void
// minisort (Node *current)
func (n *node) minisort() {
	temp := n.outOfTree[n.nextArc]
	var i uint
	size := n.numOutOfTree
	tempflow := temp.flow

	for i := n.nextArc + 1; i < size && tempflow < n.outOfTree[i].flow; i++ {
		n.outOfTree[i-1] = n.outOfTree[i]
	}
	n.outOfTree[i-1] = temp
}

// =================== the root object
type root struct {
	start *node
	end   *node
}

// newRoot is a wrapper on new(root) to mimic source.
// in-lined
// func newRoot() *root {
// 	return new(root)
// }

// free reinitializes a root value.
// CLB: don't need in Go - only used as part of freeMemory in C source
// func (r *root) free() {
// 	r.start = nil
// 	r.end = nil
// }

// ================ public functions =====================

// ReadDimacsFile implements readDimacsFile of C source code.
func ReadDimacsFile(fh *os.File) error {
	var i, numLines, from, to, first, last uint
	var capacity uint
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
			if len(line) == 0 {
				break // nothing more to process
			}
			// ... at EOF with data but no '\n' line termination.
			// While not necessary for os.STDIN; it can happen in a file.
			atEOF = true
		} else {
			// Strip off EOL.
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
				// in-lined: strongRoots[i] = newRoot()
				strongRoots[i] = new(root)
				// in-lined: adjacencyList[i] = &newNode(i + 1)
				adjacencyList[i] = &node{number: i + 1}
				var u uint
				labelCount = append(labelCount, u)
			}
			for i = 0; i < numArcs; i++ {
				// in-lined: arcList[i] = newArc()
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
				return fmt.Errorf("unrecognized character %v on line %d", ch1, numLines)
			}
		case '\n', 'c':
			continue // catches blank lines and "comment" lines - blank lines not in spec.
		default:
			return fmt.Errorf("unknown data: %s", string(line))
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

// SimpleInitialization implements simpleInitialization of C source code.
func SimpleInitialization() {
	var i, size uint
	var tempArc *arc

	size = adjacencyList[source-1].numberOutOfTree
	for i = 0; i < size; i++ {
		tempArc = adjacencyList[source-1].outOfTree[i]
		tempArc.flow = tempArc.capacity
		tempArc.to.excess += tempArc.capacity
	}

	size = adjacencyList[sink-1].numberOutOfTree
	for i = 0; i < size; i++ {
		tempArc = adjacencyList[sink-1].outOfTree[i]
		tempArc.flow = tempArc.capacity
		tempArc.from.excess -= tempArc.capacity
	}

	adjacencyList[source-1].excess = 0
	adjacencyList[sink-1].excess = 0

	for i = 0; i < numNodes; i++ {
		if adjacencyList[i].excess > 0 {
			adjacencyList[i].label = 1
			labelCount[1]++
			adjacencyList[i].addToStrongBucket(strongRoots[1])
		}
	}

	adjacencyList[source-1].label = numNodes
	adjacencyList[sink-1].label = 0
	labelCount[0] = (numNodes - 2) - labelCount[1]
}

// FlowPhaseOne implements pseudoFlowPhaseOne of C source code.
func FlowPhaseOne() {
	var strongRoot *node

	if pseudoCtx.LowestLabel {
		strongRoot = getLowestStrongRoot()
		for ; strongRoot != nil; strongRoot = getLowestStrongRoot() {
			strongRoot.processRoot()
		}
	} else {
		strongRoot = getHighestStrongRoot()
		for ; strongRoot != nil; strongRoot = getHighestStrongRoot() {
			strongRoot.processRoot()
		}
	}
}

// RecoverFlow implements recoverFlow of C source code.
// It internalize setting 'gap' value.
func RecoverFlow() {
}

// Result returns scan of arc/node results in Dimac syntax.
//
// Example for input file "maxflow.net":
//	c <header>
//	c
//	c Dimacs-format maximum flow result file
//	c generated by pseudo.go
//	c
//	c Solution
//	s 15
//	c
//	c SRC DST FLOW
//	f 1 2 5
//	f 1 3 10
//	...
func Result(header string) []string {
	result := []string{
		"c" + " " + header,
		"c",
		"c Dimacs-format maximum flow result file",
		"c generated by pseudo.go",
		"c",
		"c Solution"}

	// add Solution

	// add flows
	result = append(result, "c", "c SRC DST FLOW")

	return result
}

// ======================== quicksort implementation

// static void
// quickSort (Arc **arr, const uint first, const uint last)
func quickSort(arr []*arc, first, last uint) {
	var i, j, x1, x2, x3, pivot, pivotval uint // don't need "mid"
	left, right := first, last
	var swap *arc

	// Bubble sort if 5 elements or less
	if (right - left) <= 5 {
		for i := right; i > left; i++ {
			swap = nil
			for j := left; j < i; j++ {
				if arr[j].flow < arr[j+1].flow {
					swap = arr[j]
					arr[j] = arr[j+1]
					arr[j+1] = swap
				}
			}
			if swap != nil {
				return
			}
		}
		return
	}

	pivot = (first + last) / 2
	x1 = arr[first].flow
	x2 = arr[pivot].flow // was: arr[mid]
	x3 = arr[last].flow

	if x1 <= x2 {
		if x2 > x3 {
			pivot = left
			if x1 <= x3 {
				pivot = right
			}
		}
	} else {
		if x2 <= x3 {
			pivot = right
			if x1 <= x3 {
				pivot = left
			}
		}
	}

	pivotval = arr[pivot].flow
	swap = arr[first]
	arr[first] = arr[pivot]
	arr[pivot] = swap

	left = first + 1

	for left < right {
		if arr[left].flow < pivotval {
			swap = arr[left]
			arr[left] = arr[right]
			arr[right] = swap
			right--
		} else {
			left++
		}
	}

	swap = arr[first]
	arr[first] = arr[left]
	arr[left] = swap

	if first < (left - 1) {
		quickSort(arr, first, left-1)
	}
	if left+1 < last {
		quickSort(arr, left+1, last)
	}
}
