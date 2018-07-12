// pseudo.go - a concurrency safe implementation of Hochbaum's pseudoflow algorithm.
// Copyright (c) 2017 C. L. Banning (clbanning@gmail.com).  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pseudo is a concurrency safe implementation of Hochbaum's pseudoflow algorithm.
// It derives from an original port of pseudo3.23 from C to Go that can be used
// from https://github.com/clbanning/pseudo/v1.2.
//
// The way to use this package is to create a runtime session with the desired
// Context - s := NewSession(Context{}) uses the default Context. Then call s.Run
// or s.RunJSON to get the results for a data set.
// Internal processing statistics and timings are available after s.Run is
// called with s.StatsJSON and s.TimerJSON.
//
// The default output looks like this:
//	c Data: _data/dimacsMaxf.txt
//	c
//	c Dimacs-format maximum flow result generated by pseudo.go
//	c
//	c Optimal flow using  Hochbaum's PseudoFlow algorithm
//	c
//	c Runtime Configuration -
//	c Highest label pseudoflow algorithm
//	c Using LIFO buckets
//	c
//	c Solution checks as feasible
//	c
//	c Solution checks as optimal
//	c
//	c Solution
//	s 15
//	c
//	c SRC DST FLOW
//	f 1 2 5
//	f 2 5 0
//	f 3 4 5
//	f 5 6 5
//	f 4 6 10
//	f 3 5 5
//	f 2 4 5
//	f 1 3 10
//
// An output option is to report the minimum cut rather than the flows.
// Calling s.Run with Context{DisplayCut:true} produces the following result.
//	c Data: _data/dimacsMaxf.txt
//	c
//	c Dimacs-format maximum flow result generated by pseudo.go
//	c
//	c Optimal flow using  Hochbaum's PseudoFlow algorithm
//	c
//	c Runtime Configuration -
//	c Highest label pseudoflow algorithm
//	c Using LIFO buckets
//	c
//	c Solution checks as feasible
//	c
//	c Solution checks as optimal
//	c
//	c Solution
//	s 15
//	c
//	c
//	c Nodes in source set of min s-t cut:
//	n 1
//	n 3
//
package pseudo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Session is the runtime container.
type Session struct {
	// the runtime context
	ctx Context
	// global variables
	lowestStrongLabel               uint
	highestStrongLabel              uint
	adjacencyList                   []*node
	strongRoots                     []*root
	arcList                         []*arc
	labelCount                      []uint
	numNodes, numArcs, source, sink uint
	// stats and timer
	stats statistics
	times timer
}

// Context provides optional switches that can be used to configure
// the Session runtime.
type Context struct {
	LowestLabel bool
	FifoBuckets bool
	DisplayCut  bool // report minimun cut set instead of graph flows
}

// statistics
type statistics struct {
	Pushes   uint `json:"pushes"`
	Mergers  uint `json:"mergers"`
	Relabels uint `json:"relabels"`
	Gaps     uint `json:"gaps"`
	ArcScans uint `json:"arcScans"`
}

// timing info in case someone wants it as in C source main()
type timer struct {
	start, readfile, initialize, flow, recflow time.Time
}

// NewSession returns a pseudo Session initialized to the specified Context.
// specified by 'c'. Examples:
//	s := NewSession(Context{})                                 // use default runtime settings
//	s := NewSession(Context{LowestLabel:true,DisplayCut:true}) // use LowestLabel logic and output the minimum cut
//
func NewSession(c Context) *Session {
	s := &Session{ctx: c}
	if s.ctx.LowestLabel {
		s.lowestStrongLabel = 1
	} else {
		s.highestStrongLabel = 1
	}
	return s
}

// ConfigJSON returns the runtime context settings as a JSON object.
func (s *Session) ConfigJSON() string {
	j, _ := json.Marshal(s.ctx)
	return string(j)
}

// StatsJSON returns the runtime stats as a JSON object.
func (s *Session) StatsJSON() string {
	j, _ := json.Marshal(s.stats)
	return string(j)
}

// TimerJSON returns timings of the 4 processing steps of Run -
// readDimacsFile, simpleInitialization, flowPhaseOne, and recoverFlow.
// Note: the file initialization and result marshaling times are not
// included in result.
func (s *Session) TimerJSON() string {
	data := struct {
		ReadDimacsFile       string `json:"readDimacsFile"`
		SimpleInitialization string `json:"simpleInitialization"`
		FlowPhaseOne         string `json:"flowPhaseOne"`
		RecoverFlow          string `json:"recoverFlow"`
		Total                string `json:"total"`
	}{
		s.times.readfile.Sub(s.times.start).String(),
		s.times.initialize.Sub(s.times.readfile).String(),
		s.times.flow.Sub(s.times.initialize).String(),
		s.times.recflow.Sub(s.times.flow).String(),
		s.times.recflow.Sub(s.times.start).String(),
	}
	j, _ := json.Marshal(data)
	return string(j)
}

// ==================== the arc object
type arc struct {
	from      *node
	to        *node
	flow      int // in source: uint
	capacity  int // in source: uint
	direction uint
}

// static inline void
// pushUpward (Arc *currentArc, Node *child, Node *parent, const uint resCap)
func (s *Session) pushUpward(a *arc, child, parent *node, resCap int) {
	s.stats.Pushes++
	if resCap >= child.excess {
		parent.excess += child.excess
		a.flow += child.excess
		child.excess = 0
		return
	}

	a.direction = 0
	parent.excess += resCap
	child.excess -= resCap
	a.flow = a.capacity
	parent.outOfTree[parent.numberOutOfTree] = a
	parent.numberOutOfTree++
	parent.breakRelationship(child)
	if s.ctx.LowestLabel {
		s.lowestStrongLabel = child.label
	}

	s.addToStrongBucket(child, s.strongRoots[child.label])
}

//static inline void
// pushDownward (Arc *currentArc, Node *child, Node *parent, uint flow)
func (s *Session) pushDownward(a *arc, child, parent *node, flow int) {
	s.stats.Pushes++

	if flow >= child.excess {
		parent.excess += child.excess
		a.flow -= child.excess
		child.excess = 0
		return
	}

	a.direction = 1
	child.excess -= flow
	parent.excess += flow
	a.flow = 0
	parent.outOfTree[parent.numberOutOfTree] = a
	parent.numberOutOfTree++
	parent.breakRelationship(child)
	if s.ctx.LowestLabel {
		s.lowestStrongLabel = child.label
	}

	s.addToStrongBucket(child, s.strongRoots[child.label])
}

// ==================== the node object
type node struct {
	arcToParent     *arc
	childList       *node
	excess          int
	label           uint
	next            *node
	nextArc         uint
	nextScan        *node
	numAdjacent     uint
	number          uint
	numberOutOfTree uint
	outOfTree       []*arc // was **Arc in C, looking at createOutOfTree, we're dealing with a pool of Arc's
	parent          *node
	visited         uint
}

// make sure everything gets allocated
func (s *Session) newNode(number uint) *node {
	return &node{
		number: number,
		// outOfTree: make([]*arc, int(s.numArcs)),
	}
}

// #ifdef LOWEST_LABEL
// static Node *
// getLowestStrongRoot (void)
func (s *Session) getLowestStrongRoot() *node {
	var i uint
	var strongRoot *node

	if s.lowestStrongLabel == 0 {
		for s.strongRoots[0].start != nil {
			strongRoot = s.strongRoots[0].start
			s.strongRoots[0].start = strongRoot.next
			strongRoot.next = nil
			strongRoot.label = uint(1)

			s.labelCount[0]--
			s.labelCount[1]++
			s.stats.Relabels++

			s.addToStrongBucket(strongRoot, s.strongRoots[strongRoot.label])
		}
		s.lowestStrongLabel = 1
	}

	for i = s.lowestStrongLabel; i < s.numNodes; i++ {
		if s.strongRoots[i].start != nil {
			s.lowestStrongLabel = i

			if s.labelCount[i-1] == 0 {
				s.stats.Gaps++
				return nil
			}

			strongRoot = s.strongRoots[i].start
			s.strongRoots[i].start = strongRoot.next
			strongRoot.next = nil
			return strongRoot
		}
	}

	s.lowestStrongLabel = s.numNodes
	return nil
}

// static Node *
// getHighestStrongRoot (void)
func (s *Session) getHighestStrongRoot() *node {
	var i uint
	strongRoot := s.newNode(0)

	for i = s.highestStrongLabel; i > 0; i-- {

		if s.strongRoots[i].start != nil {
			s.highestStrongLabel = i
			if s.labelCount[i-1] > 0 {
				strongRoot = s.strongRoots[i].start
				s.strongRoots[i].start = strongRoot.next
				strongRoot.next = nil
				return strongRoot
			}

			for s.strongRoots[i].start != nil {
				s.stats.Gaps++
				strongRoot = s.strongRoots[i].start
				s.strongRoots[i].start = strongRoot.next
				s.liftAll(strongRoot)
			}
		}
	}

	if s.strongRoots[0].start == nil {
		return nil
	}

	for s.strongRoots[0].start != nil {
		strongRoot = s.strongRoots[0].start
		s.strongRoots[0].start = strongRoot.next
		strongRoot.label = 1

		s.labelCount[0]--
		s.labelCount[1]++
		s.stats.Relabels++

		s.addToStrongBucket(strongRoot, s.strongRoots[strongRoot.label])
	}

	s.highestStrongLabel = 1

	strongRoot = s.strongRoots[1].start
	s.strongRoots[1].start = strongRoot.next
	strongRoot.next = nil

	return strongRoot
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

// static void
// processRoot (Node *strongRoot)
// (*node) processRoot. 'n' is 'strongRoot' in C source
func (s *Session) processRoot(n *node) {

	var temp, weakNode *node
	var out *arc
	strongNode := n
	n.nextScan = n.childList

	if out, weakNode = s.findWeakNode(n); out != nil {
		s.merge(weakNode, strongNode, out)
		s.pushExcess(n)
		return
	}

	s.checkChildren(n)

	for strongNode != nil {
		for strongNode.nextScan != nil {
			temp = strongNode.nextScan
			strongNode.nextScan = strongNode.nextScan.next
			strongNode = temp
			strongNode.nextScan = strongNode.childList

			if out, weakNode = s.findWeakNode(strongNode); out != nil {
				s.merge(weakNode, strongNode, out)
				s.pushExcess(n)
				return
			}

			s.checkChildren(strongNode)
		}

		if strongNode = strongNode.parent; strongNode != nil {
			s.checkChildren(strongNode)
		}
	}

	s.addToStrongBucket(n, s.strongRoots[n.label])

	if !s.ctx.LowestLabel {
		s.highestStrongLabel++
	}
}

// static void
// merge (Node *parent, Node *child, Arc *newArc)
// (*node) merge. 'n' is 'parent' in C source.
func (s *Session) merge(n, child *node, newArc *arc) {
	var oldArc *arc
	var oldParent *node
	current := child
	newParent := n

	s.stats.Mergers++ // unlike C source always calc stats

	for current.parent != nil {
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
func (s *Session) pushExcess(n *node) {
	var current, parent *node
	var arcToParent *arc
	prevEx := 1

	for current = n; current.excess != 0 && current.parent != nil && current.arcToParent != nil; current = parent {
		parent = current.parent
		prevEx = parent.excess

		arcToParent = current.arcToParent

		if arcToParent.direction != 0 {
			s.pushUpward(arcToParent, current, parent, arcToParent.capacity-arcToParent.flow)
		} else {
			s.pushDownward(arcToParent, current, parent, arcToParent.flow)
		}
	}

	if current.excess > 0 && prevEx <= 0 {
		if s.ctx.LowestLabel {
			s.lowestStrongLabel = current.label
		}
		s.addToStrongBucket(current, s.strongRoots[current.label])
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
	}

	current.next = child.next
	child.next = nil
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
func (s *Session) findWeakNode(n *node) (*arc, *node) {
	var i, size uint
	var out *arc
	var weakNode *node

	size = n.numberOutOfTree

	for i = n.nextArc; i < size; i++ {
		s.stats.ArcScans++
		if s.ctx.LowestLabel {
			if n.outOfTree[i].to.label == s.lowestStrongLabel-1 {
				n.nextArc = i
				out = n.outOfTree[i]
				weakNode = out.to
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
			if n.outOfTree[i].from.label == s.lowestStrongLabel-1 {
				n.nextArc = i
				out = n.outOfTree[i]
				weakNode = out.from
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
		} else {
			if n.outOfTree[i].to.label == s.highestStrongLabel-1 {
				n.nextArc = i
				out = n.outOfTree[i]
				weakNode = out.to
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
			if n.outOfTree[i].from.label == s.highestStrongLabel-1 {
				n.nextArc = i
				out = n.outOfTree[i]
				weakNode = out.from
				n.numberOutOfTree--
				n.outOfTree[i] = n.outOfTree[n.numberOutOfTree]
				return out, weakNode
			}
		}
	}

	n.nextArc = n.numberOutOfTree
	return nil, nil

}

// (*node) checkChildren
func (s *Session) checkChildren(n *node) {
	for ; n.nextScan != nil; n.nextScan = n.nextScan.next {
		if n.nextScan.label == n.label {
			return
		}
	}

	s.labelCount[n.label]--
	n.label++
	s.labelCount[n.label]++

	s.stats.Relabels++ // Always collect stats

	n.nextArc = 0
}

// static void
// liftAll (Node *rootNode)
// node.liftAll()
func (s *Session) liftAll(n *node) {
	var temp *node
	current := n

	current.nextScan = current.childList

	s.labelCount[current.label]--
	current.label = s.numNodes

	for ; current != nil; current = current.parent {
		for current.nextScan != nil {
			temp = current.nextScan
			current.nextScan = current.nextScan.next
			current = temp
			current.nextScan = current.childList

			s.labelCount[current.label]--
			current.label = s.numNodes
		}
	}
}

func (s *Session) addToStrongBucket(n *node, rootBucket *root) {
	if s.ctx.FifoBuckets {
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
	if n.numberOutOfTree > uint(1) {
		quickSort(n.outOfTree, 0, n.numberOutOfTree-1)
	}
}

// static void
// minisort (Node *current)
func (n *node) minisort() {
	temp := n.outOfTree[n.nextArc]
	var i uint
	size := n.numberOutOfTree
	tempflow := temp.flow

	for i = n.nextArc + 1; i < size && tempflow < n.outOfTree[i].flow; i++ {
		n.outOfTree[i-1] = n.outOfTree[i]
	}
	n.outOfTree[i-1] = temp
}

// static void
// decompose (Node *excessNode, const uint source, uint *iteration)
// CLB: would prefer node.decompose(source) iteration, but keep mainline logic the same
//  node.decompose(source uint, interation *uint)
func (n *node) decompose(source uint, iteration *uint) {
	current := n
	var tempArc *arc
	bottleneck := n.excess

	for ; current.number != source && current.visited < *iteration; current = tempArc.from {
		current.visited = *iteration
		tempArc = current.outOfTree[current.nextArc]

		if tempArc.flow < bottleneck {
			bottleneck = tempArc.flow
		}
	}

	if current.number == source {
		n.excess -= bottleneck
		current = n

		for current.number != source {
			tempArc = current.outOfTree[current.nextArc]
			tempArc.flow -= bottleneck

			if tempArc.flow != 0 {
				current.minisort()
			} else {
				current.nextArc++
			}
			current = tempArc.from
		}
		return
	}

	*iteration++

	bottleneck = current.outOfTree[current.nextArc].flow
	for current.visited < *iteration {
		current.visited = *iteration
		tempArc = current.outOfTree[current.nextArc]

		if tempArc.flow < bottleneck {
			bottleneck = tempArc.flow
		}
		current = tempArc.from
	}

	*iteration++

	for current.visited < *iteration {
		current.visited = *iteration

		tempArc = current.outOfTree[current.nextArc]
		tempArc.flow -= bottleneck

		if tempArc.flow != 0 {
			current.minisort()
			current = tempArc.from
		} else {
			current.nextArc++
			current = tempArc.from
		}
	}
}

// =================== the root object
// allocations are in-line, as needed
type root struct {
	start *node
	end   *node
}

// ========================== functions implementing solution logic ============================

// static void
// checkOptimality (const uint gap)
// Internalize "gap" as in RecoverFlow.
func (s *Session) checkOptimality(w io.Writer) error {
	// setting gap value is taken out of main() in C source code
	var gap uint
	if s.ctx.LowestLabel {
		gap = s.lowestStrongLabel
	} else {
		gap = s.numNodes
	}

	var i uint
	var mincut int
	// in source: excess := make([]uint, numNodes)
	excess := make([]int, s.numNodes)

	check := true
	var err error
	for i = 0; i < s.numArcs; i++ {
		if s.arcList[i].from.label >= gap && s.arcList[i].to.label < gap {
			mincut += s.arcList[i].capacity
		}
		if s.arcList[i].flow > s.arcList[i].capacity || s.arcList[i].flow < 0 {
			check = false
			if _, err = w.Write([]byte(fmt.Sprintf("c Capacity constraint violated on arc (%d, %d). Flow = %d, capacity = %d\n",
				s.arcList[i].from.number,
				s.arcList[i].to.number,
				s.arcList[i].flow,
				s.arcList[i].capacity))); err != nil {
				return err
			}
		}
		excess[s.arcList[i].from.number-1] -= s.arcList[i].flow
		excess[s.arcList[i].to.number-1] += s.arcList[i].flow
	}
	for i = 0; i < s.numNodes; i++ {
		if i != s.source-1 && i != s.sink-1 {
			if excess[i] != 0 {
				check = false
				if _, err = w.Write([]byte(fmt.Sprintf("c Flow balance constraint violated in node %d. Excess = %d\n",
					i+1,
					excess[i]))); err != nil {
					return err
				}
			}
		}
	}
	if check {
		if _, err = w.Write([]byte("c \nc Solution checks as feasible\n")); err != nil {
			return err
		}
	}

	check = true
	if excess[s.sink-1] != mincut {
		check = false
		if _, err = w.Write([]byte("c \nc Flow is not optimal - max flow does not equal min cut\n")); err != nil {
			return err
		}
	}
	if check {
		if _, err = w.Write([]byte("c \nc Solution checks as optimal\nc \nc Solution\n")); err != nil {
			return err
		}
		if _, err = w.Write([]byte(fmt.Sprintf("s %d\n", mincut))); err != nil {
			return err
		}
	}

	return nil
}

// static void
// displayCut (const uint gap)
func (s *Session) displayCut(w io.Writer) error {
	var gap uint
	if s.ctx.LowestLabel {
		gap = s.lowestStrongLabel
	} else {
		gap = s.numNodes
	}

	var err error
	if _, err = w.Write([]byte("c Nodes in source set of min s-t cut:\n")); err != nil {
		return err
	}

	for i := uint(0); i < s.numNodes; i++ {
		if s.adjacencyList[i].label >= gap {
			if _, err = w.Write([]byte(fmt.Sprintf("n %d\n", s.adjacencyList[i].number))); err != nil {
				return err
			}
		}
	}

	return nil
}

// static void
// displayFlow (void)
// C_source uses "a SRC DST FLOW" format; however, the examples we have,
// e.g., http://lpsolve.sourceforge.net/5.5/DIMACS_asn.htm, use
// "f SRC DST FLOW" format.  Here we use the latter, since we can
// then use the examples as test cases.
func (s *Session) displayFlow(w io.Writer) error {
	var err error
	for i := uint(0); i < s.numArcs; i++ {
		if _, err = w.Write([]byte(fmt.Sprintf("f %d %d %d\n",
			s.arcList[i].from.number,
			s.arcList[i].to.number,
			s.arcList[i].flow))); err != nil {
			return err
		}
	}

	return nil
}

// ReadDimacsFile implements readDimacsFile of C source code.
func (s *Session) readDimacsFile(r io.Reader) error {
	var i, numLines, from, to, first, last uint
	var capacity int
	var ch1 string

	buf := bufio.NewReader(r)
	var atEOF bool
	var n uint64
	var haveSource, haveSink bool
	for {
		if atEOF {
			break
		}

		line, err := buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		} else if err == io.EOF {
			if len(bytes.TrimSpace(line)) == 0 {
				break // nothing more to process
			}
			// ... at EOF with data but no '\n' line termination.
			// While not necessary for os.Stdin; it can happen in a file.
			atEOF = true
		} else {
			// Strip off EOL and white space
			line = bytes.TrimSpace(line[:len(line)-1])
			if len(line) == 0 {
				continue // skip empty lines
			}
		}
		numLines++

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
		switch line[0] {
		case 'p':
			vals := strings.Fields(string(line))
			if len(vals) != 4 {
				return fmt.Errorf("p entry doesn't have 3 values, has: %d", len(vals))
			}
			n, err = strconv.ParseUint(vals[2], 10, 64)
			if err != nil {
				return err
			}
			s.numNodes = uint(n)
			n, err = strconv.ParseUint(vals[3], 10, 64)
			if err != nil {
				return err
			}
			s.numArcs = uint(n)

			s.adjacencyList = make([]*node, s.numNodes)
			s.strongRoots = make([]*root, s.numNodes)
			s.labelCount = make([]uint, s.numNodes)
			s.arcList = make([]*arc, s.numArcs)

			var i uint
			for i = 0; i < s.numNodes; i++ {
				s.strongRoots[i] = &root{} // newRoot()
				s.adjacencyList[i] = s.newNode(uint(i + 1))
			}
			for i = 0; i < s.numArcs; i++ {
				s.arcList[i] = &arc{direction: 1} // newArc(1)
			}
			first = 0
			last = s.numArcs - 1
		case 'a':
			vals := strings.Fields(string(line))
			if len(vals) != 4 {
				return fmt.Errorf("a entry doesn't have 3 values, has: %d", len(vals))
			}
			n, err = strconv.ParseUint(vals[1], 10, 64)
			if err != nil {
				return err
			}
			from = uint(n)
			n, err = strconv.ParseUint(vals[2], 10, 64)
			if err != nil {
				return err
			}
			to = uint(n)
			n, err = strconv.ParseUint(vals[3], 10, 64)
			if err != nil {
				return err
			}
			capacity = int(n)

			// What's the point of loading arcList this way?
			// 	(1+3)%2 = 0 --> arcList[first]
			// 	(1+2)%2 = 1 --> arcList[last]
			if (from+to)%2 != 0 {
				s.arcList[first].from = s.adjacencyList[from-1]
				s.arcList[first].to = s.adjacencyList[to-1]
				s.arcList[first].capacity = capacity
				first++
			} else {
				s.arcList[last].from = s.adjacencyList[from-1]
				s.arcList[last].to = s.adjacencyList[to-1]
				s.arcList[last].capacity = capacity
				last--
			}

			s.adjacencyList[from-1].numAdjacent++
			s.adjacencyList[to-1].numAdjacent++
		case 'n':
			vals := strings.Fields(string(line))
			if len(vals) != 3 {
				return fmt.Errorf("n entry doesn't have 2 values, has: %d", len(vals))
			}
			n, err = strconv.ParseUint(vals[1], 10, 64)
			if err != nil {
				return err
			}
			i = uint(n)
			ch1 = vals[2]

			if ch1 == "s" {
				if haveSource {
					return fmt.Errorf("muliple 's' n lines")
				}
				s.source = i
				haveSource = true
			} else if ch1 == "t" {
				if haveSink {
					return fmt.Errorf("multiple 't' n lines")
				}
				s.sink = i
				haveSink = true
			} else {
				return fmt.Errorf("unrecognized character %s on line %d", ch1, numLines)
			}
		case 'c':
			continue // catches "comment" lines
		default:
			return fmt.Errorf("unknown data: %s", string(line))
		}
	}

	for i = 0; i < s.numNodes; i++ {
		s.adjacencyList[i].createOutOfTree()
	}

	for i = 0; i < s.numArcs; i++ {
		to = s.arcList[i].to.number
		from = s.arcList[i].from.number
		capacity = s.arcList[i].capacity

		if !(s.source == to || s.sink == from || from == to) {
			if s.source == from && to == s.sink {
				s.arcList[i].flow = capacity
			} else if from == s.source || to != s.sink {
				s.adjacencyList[from-1].addOutOfTreeNode(s.arcList[i])
			} else if to == s.sink {
				s.adjacencyList[to-1].addOutOfTreeNode(s.arcList[i])
			} else {
				s.adjacencyList[from-1].addOutOfTreeNode(s.arcList[i])
			}
		}
	}

	return nil
}

// SimpleInitialization implements simpleInitialization of C source code.
func (s *Session) simpleInitialization() {
	var i, size uint
	var tempArc *arc

	size = s.adjacencyList[s.source-1].numberOutOfTree
	for i = 0; i < size; i++ {
		tempArc = s.adjacencyList[s.source-1].outOfTree[i]
		tempArc.flow = tempArc.capacity
		tempArc.to.excess += tempArc.capacity
	}

	size = s.adjacencyList[s.sink-1].numberOutOfTree
	for i = 0; i < size; i++ {
		tempArc = s.adjacencyList[s.sink-1].outOfTree[i]
		tempArc.flow = tempArc.capacity
		tempArc.from.excess -= tempArc.capacity
	}

	s.adjacencyList[s.source-1].excess = 0
	s.adjacencyList[s.sink-1].excess = 0

	for i = 0; i < s.numNodes; i++ {
		if s.adjacencyList[i].excess > 0 {
			s.adjacencyList[i].label = 1
			s.labelCount[1]++
			s.addToStrongBucket(s.adjacencyList[i], s.strongRoots[1])
		}
	}

	s.adjacencyList[s.source-1].label = s.numNodes
	s.adjacencyList[s.sink-1].label = 0
	s.labelCount[0] = (s.numNodes - 2) - s.labelCount[1]
}

// FlowPhaseOne implements pseudoFlowPhase1 of C source code.
func (s *Session) flowPhaseOne() {
	var strongRoot *node

	if s.ctx.LowestLabel {
		strongRoot = s.getLowestStrongRoot()
		for ; strongRoot != nil; strongRoot = s.getLowestStrongRoot() {
			s.processRoot(strongRoot)
		}
	} else {
		strongRoot = s.getHighestStrongRoot()
		for ; strongRoot != nil; strongRoot = s.getHighestStrongRoot() {
			s.processRoot(strongRoot)
		}
	}
}

// static void
// recoverFlow (const uint gap)
// RecoverFlow implements recoverFlow of C source code.
// It internalizes setting 'gap' value.
func (s *Session) recoverFlow() {
	// setting gap value is taken out of main() in C source code
	var gap uint
	if s.ctx.LowestLabel {
		gap = s.lowestStrongLabel
	} else {
		gap = s.numNodes
	}

	var i, j uint
	iteration := uint(1)
	var tempArc *arc
	var tempNode *node

	for i = 0; i < s.adjacencyList[s.sink-1].numberOutOfTree; i++ {
		tempArc = s.adjacencyList[s.sink-1].outOfTree[i]
		if tempArc.from.excess < 0 {
			if tempArc.from.excess+tempArc.flow < 0 {
				tempArc.from.excess += tempArc.flow
				tempArc.flow = 0
			} else {
				tempArc.flow = tempArc.from.excess + tempArc.flow
				tempArc.from.excess = 0
			}
		}
	}

	for i = 0; i < s.adjacencyList[s.source-1].numberOutOfTree; i++ {
		tempArc = s.adjacencyList[s.source-1].outOfTree[i]
		tempArc.to.addOutOfTreeNode(tempArc)
	}

	s.adjacencyList[s.source-1].excess = 0
	s.adjacencyList[s.sink-1].excess = 0

	for i = 0; i < s.numNodes; i++ {
		tempNode = s.adjacencyList[i]
		if i == s.source-1 || i == s.sink-1 {
			continue
		}

		if tempNode.label >= gap {
			tempNode.nextArc = 0
			if tempNode.parent != nil && tempNode.arcToParent.flow != 0 {
				tempNode.arcToParent.to.addOutOfTreeNode(tempNode.arcToParent)
			}

			for j = 0; j < tempNode.numberOutOfTree; j++ {
				if tempNode.outOfTree[j].flow == 0 {
					tempNode.numberOutOfTree--
					tempNode.outOfTree[j] = tempNode.outOfTree[tempNode.numberOutOfTree]
					j--
				}
			}

			tempNode.sort()
		}
	}

	for i = 0; i < s.numNodes; i++ {
		tempNode = s.adjacencyList[i]
		for tempNode.excess > 0 {
			iteration++
			tempNode.decompose(s.source, &iteration)
		}
	}
}

// Result returns scan of arc/node results in Dimac syntax.
//
// Example for input file "maxflow.net":
//	c <header>
//	c
//	c Dimacs-format maximum flow result generated by pseudo.go
//	c
//	c Optimal flow using Hochbaum's PseudoFlow algorithm"
//	c
//	c Runtime Configuration:
//	c Lowest label pseudoflow algorithm
//	c Using LIFO buckets
//	c
//	c Solution checks as feasible.
//	c
//	c Solution checks as optimal
//	c Solution
//	s 15
//	c
//	c SRC DST FLOW
//	f 1 2 5
//	f 1 3 10
//	...
func (s *Session) result(w io.Writer, header string) error {
	// header and runtime config info
	ret := [][]byte{
		[]byte("c " + header + "\n"),
		[]byte("c \n"),
		[]byte("c Dimacs-format maximum flow result generated by pseudo.go\n"),
		[]byte("c \n"),
		[]byte("c Optimal flow using  Hochbaum's PseudoFlow algorithm\n"),
		[]byte("c \n"),
		[]byte("c Runtime Configuration -\n")}

	var err error
	for i, v := range ret {
		if len(header) == 0 && i < 2 {
			continue
		}
		if _, err = w.Write(v); err != nil {
			return err
		}
	}

	var line []byte
	if s.ctx.LowestLabel {
		line = []byte("c Lowest label pseudoflow algorithm\n")
	} else {
		line = []byte("c Highest label pseudoflow algorithm\n")
	}
	if _, err = w.Write(line); err != nil {
		return err
	}

	if s.ctx.FifoBuckets {
		line = []byte("c Using FIFO buckets\n")
	} else {
		line = []byte("c Using LIFO buckets\n")
	}
	if _, err = w.Write(line); err != nil {
		return err
	}

	// add Solution
	if err = s.checkOptimality(w); err != nil {
		return nil
	}
	if _, err = w.Write([]byte("c \n")); err != nil {
		return err
	}

	// add cut nodes
	if s.ctx.DisplayCut {
		if err = s.displayCut(w); err != nil {
			return err
		}
	} else {
		// add flows
		if _, err = w.Write([]byte("c SRC DST FLOW\n")); err != nil {
			return err
		}
		if err = s.displayFlow(w); err != nil {
			return err
		}
	}

	return nil
}

// ================ public functions =====================

// Run takes an input file and returns the optimal flow if
// possible. 'input' is the (relative) path name for the data
// file; if input == "stdin" then os.Stdin is read. Optional
// 'header' is a header to be written on the first comment
// line of the output; by default the first output line will
// be "c Data: <input>".
func (s *Session) Run(input string, header ...string) ([]string, error) {
	var fh *os.File
	var err error
	if strings.ToLower(input) == "stdin" {
		fh = os.Stdin
	} else {
		fh, err = os.Open(input)
		if err != nil {
			return nil, err
		}
	}
	defer fh.Close()

	if len(header) == 0 {
		header = append(header, "Data: "+input)
	}
	return s.RunReader(fh, header...)
}

// RunReader is Run but takes an io.Reader to process the input rather than
// an input file.
func (s *Session) RunReader(r io.ReadCloser, header ...string) ([]string, error) {
	w := new(bytes.Buffer)
	if err := s.RunReadWriter(r, w, header...); err != nil {
		return nil, err
	}

	// extract the result
	ret := make([]string, 0)
	for {
		l, err := w.ReadBytes('\n')
		if err == io.EOF {
			break // all lines will be NL terminated
		}
		if err != nil {
			return ret, err
		}
		ret = append(ret, string(l[:len(l)-1]))
	}

	return ret, nil
}

// RunReadWriter supports large data set output to a predefined io.Writer.
//	...
//	s := NewSession(Context{})
//	input, _ := os.Open(<input_file>)
//	defer input.Close()
//	output, _ := os.OpenFile(<output_file>, os.CREAT|os.TRUNC|os.WRONLY, 0600)
//	defer output.Close()
//
//	if err := s.RunReadWriter(input, output); err != nil {
//		// handle error
//	}
//	// result is in output_file
//
func (s *Session) RunReadWriter(r io.ReadCloser, w io.Writer, header ...string) error {
	// always reinitialize stats - might be making
	// sucessive calls to Run
	s.stats = statistics{}

	// implement C source main()
	// load the data ...
	s.times.start = time.Now()
	if err := s.readDimacsFile(r); err != nil {
		r.Close()
		return err
	}
	// might be large file, don't keep it open
	r.Close()

	return s.process(w, header...)
}

// process handles processing dimacs data. Split out to support s.RunNA.
func (s *Session) process(w io.Writer, header ...string) error {
	// find the solution ...
	s.times.readfile = time.Now()
	s.simpleInitialization()
	s.times.initialize = time.Now()
	s.flowPhaseOne()
	s.times.flow = time.Now()
	s.recoverFlow()
	s.times.recflow = time.Now()

	// results might have custom header comment
	var h string
	if len(header) > 0 {
		h = header[0]
	}
	return s.result(w, h)
}

// RunJSON returns the results of Run as a JSON object. This
// is useful if you want to return results to a JS app.
func (s *Session) RunJSON(input string, header ...string) ([]byte, error) {
	r, err := s.Run(input, header...)
	if err != nil {
		return nil, err
	}

	return json.Marshal(r)
}

// RunReaderJSON returns the results of Run as a JSON object. This
// is useful if you want to return results to a JS app.
func (s *Session) RunReaderJSON(r io.ReadCloser, header ...string) ([]byte, error) {
	res, err := s.RunReader(r, header...)
	if err != nil {
		return nil, err
	}

	return json.Marshal(res)
}

// ======================== quicksort implementation

// static void
// quickSort (Arc **arr, const uint first, const uint last)
// CLB: **Arc value is []*arc; slices manipulate the backing array
func quickSort(arr []*arc, first, last uint) {
	left, right := first, last
	var swap *arc

	// Bubble sort if 5 elements or less
	if (right - left) <= 5 {
		for i := right; i > left; i-- {
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

	pivot := (first + last) / 2
	x1 := arr[first].flow
	x2 := arr[pivot].flow // was: arr[mid]
	x3 := arr[last].flow

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

	pivotval := arr[pivot].flow
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
