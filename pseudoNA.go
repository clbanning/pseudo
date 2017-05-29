// pseudoNA.go - package extension for passing preprocessed Dimacs file.

package pseudo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// N is the dimacs 'n' entry
type N struct {
	Val  uint
	Node string
}

// A is the dimacs 'a' entry
type A struct {
	From     uint
	To       uint
	Capacity int
}

// RunNAWriter solves optimal flow given slices of 'n' and 'a' dimacs entries.
func (s *Session) RunNAWriter(numNodes, numArcs uint, nodes []N, arcs []A, w io.Writer, header ...string) error {
	if err := s.loadNA(numNodes, numArcs, nodes, arcs); err != nil {
		return err
	}
	return s.process(w, header...)
}

func (s *Session) loadNA(nn, na uint, n []N, a []A) error {
	s.numNodes, s.numArcs = nn, na

	// allocate & initialize storage
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

	// process N values
	if len(n) != 2 {
		return fmt.Errorf("want 2 N vals, have %d", len(n))
	}
	var haveSrc, haveSink bool
	for _, v := range n {
		if v.Node == "s" {
			s.source = v.Val
			haveSrc = true
		} else if v.Node == "t" {
			s.sink = v.Val
			haveSink = true
		} else {
			return fmt.Errorf("unrecognized character %s in N.Node value", v.Node)
		}
	}
	// check if there are 2 source or sink values
	if haveSrc && !haveSink {
		return fmt.Errorf("N slice does not include a sink - N.Node == t - value")
	}
	if !haveSrc && haveSink {
		return fmt.Errorf("N slice does not include a source - N.Node == s - value")
	}

	// process A values
	first := uint(0)
	last := s.numArcs - 1
	for _, v := range a {
		if (v.From+v.To)%2 != 0 {
			s.arcList[first].from = s.adjacencyList[v.From-1]
			s.arcList[first].to = s.adjacencyList[v.To-1]
			s.arcList[first].capacity = v.Capacity
			first++
		} else {
			s.arcList[last].from = s.adjacencyList[v.From-1]
			s.arcList[last].to = s.adjacencyList[v.To-1]
			s.arcList[last].capacity = v.Capacity
			last--
		}

		s.adjacencyList[v.From-1].numAdjacent++
		s.adjacencyList[v.To-1].numAdjacent++
	}

	// finish initialization
	for i = 0; i < s.numNodes; i++ {
		s.adjacencyList[i].createOutOfTree()
	}

	var from, to uint
	var capacity int
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

// ParseDimacsReader generates input data for s.RunNAWriter. It is generally for tests.
func ParseDimacsReader(r io.Reader) (uint, uint, []N, []A, error) {
	var numNodes, numArcs uint
	n := []N{}
	a := []A{}

	var i, from, to uint
	var capacity int
	var ch1 string

	buf := bufio.NewReader(r)
	var atEOF bool
	var num uint64
	for {
		if atEOF {
			break
		}

		line, err := buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return numNodes, numArcs, n, a, err
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

		switch line[0] {
		case 'p':
			vals := strings.Fields(string(line))
			if len(vals) != 4 {
				return numNodes, numArcs, n, a, fmt.Errorf("p entry doesn't have 3 values, has: %d", len(vals))
			}
			num, err = strconv.ParseUint(vals[2], 10, 64)
			if err != nil {
				return numNodes, numArcs, n, a, err
			}
			numNodes = uint(num)
			num, err = strconv.ParseUint(vals[3], 10, 64)
			if err != nil {
				return numNodes, numArcs, n, a, err
			}
			numArcs = uint(num)
		case 'a':
			vals := strings.Fields(string(line))
			if len(vals) != 4 {
				return numNodes, numArcs, n, a, fmt.Errorf("a entry doesn't have 3 values, has: %d", len(vals))
			}
			num, err = strconv.ParseUint(vals[1], 10, 64)
			if err != nil {
				return numNodes, numArcs, n, a, err
			}
			from = uint(num)
			num, err = strconv.ParseUint(vals[2], 10, 64)
			if err != nil {
				return numNodes, numArcs, n, a, err
			}
			to = uint(num)
			num, err = strconv.ParseUint(vals[3], 10, 64)
			if err != nil {
				return numNodes, numArcs, n, a, err
			}
			capacity = int(num)
			a = append(a, A{from, to, capacity})
		case 'n':
			vals := strings.Fields(string(line))
			if len(vals) != 3 {
				return numNodes, numArcs, n, a, fmt.Errorf("n entry doesn't have 2 values, has: %d", len(vals))
			}
			num, err = strconv.ParseUint(vals[1], 10, 64)
			if err != nil {
				return numNodes, numArcs, n, a, err
			}
			i = uint(num)
			ch1 = vals[2]
			n = append(n, N{i, ch1})
		case 'c':
			continue // catches "comment" lines
		default:
			return numNodes, numArcs, n, a, fmt.Errorf("unknown data: %s", string(line))
		}
	}

	return numNodes, numArcs, n, a, nil
}
