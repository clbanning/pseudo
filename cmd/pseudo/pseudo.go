// pseudo.go - a command-line program for using the package github.com/clbanning/pseudo
package main

import (
	"flag"
	"fmt"
	"os"

	p "github.com/clbanning/pseudo"
)

func main() {
	var reportStats, reportTimes, stdin bool
	var output string
	flag.BoolVar(&p.PseudoCtx.LowestLabel, "lowestlabel", false, "set LowestLabel == true")
	flag.BoolVar(&p.PseudoCtx.FifoBucket, "fifobuckets", false, "set fifobucket == true")
	flag.BoolVar(&p.PseudoCtx.DisplayCut, "displaycut", false, "report min cut rather than flows")
	flag.BoolVar(&reportStats, "stats", false, "report flow computational metrics")
	flag.BoolVar(&reportTimes, "times", false, "report timer metrics")
	flag.BoolVar(&stdin, "stdin", false, "read data from stdin")
	flag.StringVar(&output, "o", "", "write results to named file")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 && !stdin {
		fmt.Println("no input file specified and 'stdin' flag not set")
		os.Exit(1)
	}
	if stdin {
		args = []string{ "stdin" }
	}

	var fh *os.File
	var err error
	if len(output) == 0 {
		// assume stdout
		fh = os.Stdout
		output = "stdin"
	} else {
		fh, err = os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Printf("unable to open output file: %s, err: %s\n", output, err.Error())
			os.Exit(1)
		}
	}
	defer fh.Close()

	// loop through args and report output
	for _, arg := range args {
		results, err := p.Run(arg)
		fmt.Fprintln(fh) // separate runs with a blank line
		if err != nil {
			fmt.Fprintln(fh, "ERROR -", err)
			continue
		}
		for _, line := range results {
			fmt.Fprintln(fh, line)
		}
		if reportStats {
			fmt.Fprintln(fh, "\nstats:", p.StatsJSON())
		}
		if reportTimes {
			fmt.Fprintln(fh, "\ntimes:", p.TimerJSON())
		}
	}
}
