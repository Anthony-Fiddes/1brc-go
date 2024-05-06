package main

import (
	"flag"
	"log"
	"os"
	"path"
	"runtime/pprof"
)

func main() {
	pprofPath := flag.String("cpuprofile", "", "If supplied, will generate a pprof file at the given path")
	revision := flag.Int("revision", 0, "Which revision to run. Defaults to 0")

	flag.Parse()

	type Revision func(measurementsPath string)
	revisions := []Revision{r0, r1, r2, r3, r4, r5, r6}
	if *revision >= len(revisions) || *revision < 0 {
		log.SetFlags(0)
		log.Printf("revision must be between %d and %d", 0, len(revisions)-1)
		log.Println()
		flag.Usage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) != 1 {
		log.SetFlags(0)
		log.Printf(
			"%s takes one argument, the path to the file containing the measurements to use.",
			path.Base(os.Args[0]),
		)
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	measurementsPath := args[0]

	if *pprofPath != "" {
		file, err := os.Create(*pprofPath)
		if err != nil {
			log.Fatalf("Could not create the cpuprofile at '%s'", *pprofPath)
		}
		defer file.Close()
		pprof.StartCPUProfile(file)
		defer pprof.StopCPUProfile()
	}

	revisions[*revision](measurementsPath)
}
