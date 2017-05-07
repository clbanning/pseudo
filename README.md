<h2>pseudo - a Go implementation of pseudoflow algorithm</h2>

A Go implementation of Hochbaum's PseudoFlow algorithm as [implemented here in C][c_ref] that is safe for concurrent (server) use.

<h2>Documentation</h2>

... is in the [usual place][docs].

<h2>Version 1.2</h2>

The original port of the C source code is in the subdirectory [v1.2][v1.2]. It is used in the example command-line program [here][cmdline].  It is fine for command-line or utility programs, but it is not safe for concurrent use in a server.

<h2>Status</h2>

Release 2.0.  (Note: could do with more testing with larger data sets.)


[c_ref]: http://riot.ieor.berkeley.edu/Applications/Pseudoflow/maxflow.html
[docs]: https://godoc.org/github.com/clbanning/pseudo
[v1.2]: https://github.com/clbanning/pseudo/v1.2
[cmdline]: https://github.com/clbanning/pseudo/cmd/pseudo
