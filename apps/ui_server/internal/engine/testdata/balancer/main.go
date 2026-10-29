package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	input := flag.String("c", "", "")
	output := flag.String("out", "", "")
	optimize := flag.Bool("substatOptim", false, "")
	flag.Parse()
	data, err := os.ReadFile(*input)
	if err != nil || !*optimize {
		fmt.Fprintln(os.Stderr, "invalid optimizer invocation")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "optimizer started")
	switch string(data) {
	case "fail":
		fmt.Fprintln(os.Stderr, "invalid config")
		os.Exit(1)
	case "wait":
		time.Sleep(30 * time.Second)
	}
	if err := os.WriteFile(*output, append(data, []byte("-balanced")...), 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
