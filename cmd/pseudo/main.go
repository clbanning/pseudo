// pseudo.go - a command-line program for using the package github.com/clbanning/pseudo
// NOTE: input and output default to os.Stdin/os.Stdout.
// 
// $ go build -o pseudo main.go
// $ cat input_file | pseudo             # read input_file from stdin and write results to stdout
// $ pseudo input_file                   # read input_file and write results to stdout
// $ pseudo input_file1 input_file2      # successively read input files and write results to stdout
// $ pseudo -o output_file input_file    # read input_file and write results to output_file
//
// Command-line switches - lowestlabel, fifobuckets, displaycut - toggle runtime context values.
// 
package main

import (
	"flag"
	"fmt"
	"os"

	p "github.com/clbanning/pseudo"
)

func main() {
	var lowestlabel, fifobuckets, displaycut bool
	var output string
	var in, out *os.File
	var err error

	// command-line switches/options
	flag.BoolVar(&lowestlabel, "lowestlabel", false, "set LowestLabel == true")
	flag.BoolVar(&fifobuckets, "fifobuckets", false, "set fifobucket == true")
	flag.BoolVar(&displaycut, "displaycut", false, "report min cut rather than flows")
	flag.StringVar(&output, "o", "", "write results to named file")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"stdin"}
	}

	if len(output) == 0 {
		// assume stdout
		out = os.Stdout
	} else {
		out, err = os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to open output file: %s, err: %s\n", output, err.Error())
			os.Exit(1)
		}
	}
	defer out.Close()

	// loop through args and report output
	s := p.NewSession(p.Context{lowestlabel, fifobuckets, displaycut})
	for i, arg := range args {
		if arg == "stdin" {
			in = os.Stdin
		} else {
			in, err = os.Open(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR - unable to open input file: %s, err: %s\n", arg, err.Error())
				continue
			}
		}
		if err = s.RunReadWriter(in, out); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR - processing input: %s, error: %s\n", arg, err.Error())
		}
		if i != len(args)-1 {
			fmt.Fprintf(out, "\n") // separate runs with a blank line
		}
	}
}
