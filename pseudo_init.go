package pseudo

type SessionInitializer struct {
	session *Session
	first   uint
	last    uint
}

func NewSessionInitializer(session *Session) *SessionInitializer {
	return &SessionInitializer{
		session: session,
	}
}

func (si *SessionInitializer) Init(numNodes, numArcs uint) {
	s := si.session

	s.numNodes = numNodes
	s.numArcs = numArcs

	s.adjacencyList = make([]*node, numNodes)
	s.strongRoots = make([]*root, numNodes)
	s.labelCount = make([]uint, numNodes)
	s.arcList = make([]*arc, numArcs)

	var i uint
	for i = 0; i < numNodes; i++ {
		s.strongRoots[i] = &root{} // newRoot()
		s.adjacencyList[i] = s.newNode(uint(i + 1))
	}
	for i = 0; i < numArcs; i++ {
		s.arcList[i] = &arc{direction: 1} // newArc(1)
	}
	si.first = 0
	si.last = numArcs - 1
}

func (si *SessionInitializer) SetSource(source uint) {
	si.session.source = source
}

func (si *SessionInitializer) SetSink(sink uint) {
	si.session.sink = sink
}

func (si *SessionInitializer) AddArc(from, to uint, capacity int) {
	s := si.session

	// What's the point of loading arcList this way?
	// 	(1+3)%2 = 0 --> arcList[first]
	// 	(1+2)%2 = 1 --> arcList[last]
	if (from+to)%2 != 0 {
		s.arcList[si.first].from = s.adjacencyList[from-1]
		s.arcList[si.first].to = s.adjacencyList[to-1]
		s.arcList[si.first].capacity = capacity
		si.first++
	} else {
		s.arcList[si.last].from = s.adjacencyList[from-1]
		s.arcList[si.last].to = s.adjacencyList[to-1]
		s.arcList[si.last].capacity = capacity
		si.last--
	}

	s.adjacencyList[from-1].numAdjacent++
	s.adjacencyList[to-1].numAdjacent++
}

func (si *SessionInitializer) Complete() {
	s := si.session

	for i := 0; i < int(s.numNodes); i++ {
		s.adjacencyList[i].createOutOfTree()
	}

	for i := 0; i < int(s.numArcs); i++ {

		to := s.arcList[i].to.number
		from := s.arcList[i].from.number
		capacity := s.arcList[i].capacity

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
}
