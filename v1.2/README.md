<h2>pseudo - a Go implementation of pseudoflow algorithm</h2>

A Go implementation of Hochbaum's PseudoFlow algorithm as [implemented here in C][c_ref].

<h2>Documentation</h2>

... are in the [usual place][docs].

<h2>Example</h2>

A sample command-line program that supports most options is in ../cmd/pseudo.

<h2>Status</h2>

Release 1.2.  (Note: could do with more testing with larger data sets.)

<h2>TODO</h2>

Refactor to make it safe for concurrency; then package can be used in a server.

- Wrap Context, statistics, timer and globals in a Session.
- NewSession(c Context) initializes a session.
- (s *Session) Run() executes the logic within a Session.

[c_ref]: http://riot.ieor.berkeley.edu/Applications/Pseudoflow/maxflow.html
[docs]: https://godoc.org/github.com/clbanning/pseudo
