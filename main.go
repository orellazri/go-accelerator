package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type Download struct {
	url     string
	threads int
}

func (d Download) DownloadSection(num int, start int, end int) error {
	fmt.Printf("Downloading section number %d...\n", num)

	req, err := http.NewRequest("GET", d.url, nil)
	req.Header.Set("User-Agent", "Go Accelerator")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("section-%d.tmp", num), b, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (d Download) MergeSections() error {
	outputFile, err := os.OpenFile("output", os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	for i := 0; i < d.threads; i++ {
		fmt.Printf("Merging section number %d...\n", i)
		b, err := ioutil.ReadFile(fmt.Sprintf("section-%d.tmp", i))
		if err != nil {
			return err
		}
		n, err := outputFile.Write(b)
		if err != nil {
			return err
		}
		if n != len(b) {
			return err
		}

		os.Remove(fmt.Sprintf("section-%d.tmp", i))
	}

	return nil
}

func (d Download) Go() error {
	// Make a request to get the size of the file in bytes
	fmt.Println("Collecting information...")
	req, err := http.NewRequest("GET", d.url, nil)
	req.Header.Set("User-Agent", "Go Accelerator")
	req.Header.Set("Range", "bytes=1-")
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf("bad response status code: %d", resp.StatusCode)
	}

	sizeInBytes, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	fmt.Printf("File size is: %dMB\n", sizeInBytes/1e+6)

	// Download the file
	sectionSize := sizeInBytes / d.threads
	var sections = make([][2]int, d.threads) // for example: [[0,10], [11, 21], [22, 32], [33, 43]...]

	var wg sync.WaitGroup

	for i := range sections {
		i := i

		// Start byte
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		// End byte
		if i < d.threads-1 {
			sections[i][1] = sections[i][0] + sectionSize
		} else {
			sections[i][1] = sizeInBytes - 1
		}

		wg.Add(1)
		go func() {
			d.DownloadSection(i, sections[i][0], sections[i][1])
			wg.Done()
		}()
	}

	wg.Wait()

	// Merge sections
	d.MergeSections()

	return nil
}

func main() {
	// Parse flags and arguments
	var threads int
	flag.IntVar(&threads, "t", 10, "number of threads")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("USAGE: go-accelerator [-t] url")
		os.Exit(1)
	}

	// Initialize download and start
	url := args[0]
	d := Download{url, threads}
	err := d.Go()
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}
}
