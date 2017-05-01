<h2>pseudo - a Go implementation of pseudoflow algorithm</h2>

A Go implementation of Hochbaum's PseudoFlow algorithm as [implemented here in C][c_ref].

<h2>Status</h2>
Passes 1st of 4 test cases. Other test cases that need to be run:

- LowestLevel == true, FifoBucket == false
- LowestLevel == false, FifoBucket == true
- LowestLevel == true, FifoBucket == true


[c_ref]: http://riot.ieor.berkeley.edu/Applications/Pseudoflow/maxflow.html
