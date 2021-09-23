package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Download struct {
	url     string
	threads int
}

func (d Download) DownloadSection(num int, start int, end int) error {
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
	// Get output file name from url
	outputFilename := d.url[strings.LastIndex(d.url, "/")+1:]

	// Create output file
	outputFile, err := os.OpenFile(outputFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Merge section files
	for i := 0; i < d.threads; i++ {
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
	sizeInMB := sizeInBytes / 1e+6

	// Download the file
	sectionSize := sizeInBytes / d.threads
	var sections = make([][2]int, d.threads) // for example: [[0,10], [11, 21], [22, 32], [33, 43]...]

	fmt.Println("Downloading...")
	var progressWidth = 60
	var progress = 0
	fmt.Printf("[>%s] 0/%dMB", strings.Repeat(" ", progressWidth), sizeInMB)

	var wg sync.WaitGroup
	var progressMutex sync.Mutex
	for i := range sections {
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
		go func(i int) {
			defer wg.Done()

			d.DownloadSection(i, sections[i][0], sections[i][1])

			// Update progress bar
			progressMutex.Lock()
			progress++
			progressWidthRatio := int((float32(progress) / float32(d.threads)) * float32(progressWidth))
			downloadedInMB := (progress * sectionSize) / 1e+6
			fmt.Printf("\r[%s>%s] %d/%dMB", strings.Repeat("=", progressWidthRatio), strings.Repeat(" ", progressWidth-progressWidthRatio), downloadedInMB, sizeInMB)
			progressMutex.Unlock()
		}(i)
	}

	wg.Wait()

	fmt.Println()

	// Merge sections
	fmt.Println("Merging sections...")
	d.MergeSections()

	fmt.Println("Done.")

	return nil
}
