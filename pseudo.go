// pseudo.go implements pseudo3.23 support functions.
// See pseudo/cmd for CLI app.

package pseudo

import (
	"fmt"
	"json"
	// "time"

	"github.com/clbanning/checkjson"
)

// global variables
var numNodes uint
var numArcs uint
var source uint
var sink uint
var lowestStrongLabel uint
var highestStrongLabel uint
var adjacencyList *Node
var strongRoots *Root
var labelCount []uint // index'd to len(Nodes), grows with NewNode
var arcList *Arc

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

var pseudoCtx context

// Need a config to parse a JSON file with runtime settings.
// This is called by an init() perhaps with a default file name of "./pseudo.json"
// or "pseudo.config".  The init() can also handle CLI flags to override default
// settings.
func Config(file string) error {
	// read file into an array of JSON objects
	objs, err := checkjson.ReadJSONFile(file)
	if err != nil {
		return fmt.Errorf("config file: %s - %s", file, err.Error())
	}

	// get a JSON object that has "config":"pseudo" key:value pair
	type config struct {
		Config string
	}
	var ctxset bool // make sure just one pseudo config entry
	for n, obj := range objs {
		c := new(config)
		// unmarshal the object - and try and retrule a meaningful error
		if err := json.Unmarshal(obj, c); err != nil {
			return fmt.Errorf("parsing config file: %s entry: %d - %s",
				fn, n+1, checkjson.ResolveJSONError(obj, err).Error())
		}
		switch strings.ToLower(c.Config) {
		case "pseudo":
			if ctxset {
				return fmt.Errorf("duplicate 'pseudo' entry in config file: %s entry: %d", file, n)
			}
			if err := checkjson.Validate(obj, pseudoCtx); err != nil {
				return fmt.Errorf("checking pseudo config JSON object: %s", err)
			}
			if err := json.Unmarshal(obj, &pseudoCtx); err != nil {
				return fmt.Errorf("config file: %s - %s", file, err)
			}
			ctxset = true
		default:
			// return fmt.Errorf("unknown config option in config file: %s entry: %d", file, n+1)
			// for now, just ignore stuff we're not interested in
		}
	}
	if !ctxset {
		return fmt.Errorf("no pseudo config object in %s", file)
	}
	return nil
}

// the Arc object

type Arc struct {
	from      *Node
	to        *Node
	flow      uint
	capacity  uint
	direction uint
}

// Initialize a new Arc value.
func NewArc() *Arc {
	return &Arc{direction: 1}
}

// the Node object

type Node struct {
	visited         uint
	numAdjacent     uint
	number          uint
	label           uint
	excess          int
	parent          *Node
	childList       *Node
	nextScan        *Node
	numberOutOfTree uint
	outOfTree       []*Arc // was **Arc in C, looking at CreateOutOfTree, we're dealing with a pool of Arc's
	nextArc         uint
	arcToParent     *Arc
	next            *Node
}

// NewNode returns an initialized Node value.
func NewNode(n uint) *Node {
	labelCount = append(labelCount, uint{})
	return &Node{number: n}
}

func (n *Node) LiftAll() {
	var temp *Node
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

// CreateOutOfTree allocates Arc's for adjacent Nodes.
func (n *Node) CreateOutOfTree() {
	n.outOfTree = make([]*Arc, n.numAdjacent) // OK if '0' are allocated
	// runtime handles mallocs and panics on OOM; you'll get a stack trace
	/*
		if (nd->numAdjacent)
			if ((nd->outOfTree = (Arc **) malloc (nd->numAdjacent * sizeof (Arc *))) == NULL)
			{
				printf ("%s Line %d: Out of memory\n", __FILE__, __LINE__);
				exit (1);
			}
		}
	*/
}

// AddOutOfTreeNode
func (n *Node) AddOutOfTreeNode(out *Arc) {
	n.outOfTree[n.numOutOfTree] = out
	n.numOutOfTree++
}

// the Root object

type Root struct {
	start *Node
	end   *Node
}

//  NewRoot is a wrapper on new(Root) to mimic source.
func NewRoot() *Root {
	return new(Root)
}

// Free reinitializes a Root value.
func (r *Root) Free() {
	r.start = nil
	r.end = nil
}

// AddToStrongBucket may be better as a *Node method ... need to see usage elsewhere.
func (r *Root) AddToStrongBucket(newRoot *Node) {
	if pseudoCtx.FifoBucket {
		if r.start != nil {
			r.end.next = newRoot
			r.end = newRoot
			newRoot.next = nil
		} else {
			r.start = newRoot
			r.end = newRoot
			newRoot.next = nil
		}
	} else {
		newRoot.next = r.start
		r.start = newRoot
		return
	}
}
