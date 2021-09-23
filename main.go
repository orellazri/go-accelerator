package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

func main() {
	// Parse flags and arguments
	var threads int
	flag.IntVar(&threads, "t", runtime.NumCPU(), "number of threads")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("USAGE: go-accelerator [-t] url")
		os.Exit(1)
	}

	// Initialize download and start
	var startTime = time.Now()
	url := args[0]
	d := Download{url, threads}
	err := d.Go()
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	fmt.Printf("Download took: %.2f seconds\n", time.Since(startTime).Seconds())
}
