package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"
)

type FileSystem http.Dir

// The go implementation will display a directory's contents,
// we don't want that.
func (fs FileSystem) Open(path string) (http.File, error) {
	d := http.Dir(fs)
	log.Printf("Open(): Looking for path %v", path)
	f, err := d.Open(path)
	if err != nil {
		return nil, err
	}

	log.Printf("Open(): Opened...")
	// Only allow directories if they contain an 'index.html'
	s, err := f.Stat()
	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		log.Printf("Open(): In directory %v, looking for index %v", path, index)
		if _, err = d.Open(index); err != nil {
			log.Printf("Open(): Got an error: %v", err)
			return nil, err
		}
	}

	log.Printf("Open(): Returning %V", f)
	return f, nil
}

// Read a list of words into a map of slices of words
// keyed on word length
func slurp(fn string) map[int][]string {
	ret := make(map[int][]string)

	fh, err := os.Open(fn)
	if err != nil {
		log.Fatal("Failed to open list file: %v", err)
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		l := len(scanner.Text())
		if ret[l] == nil {
			ret[l] = make([]string, 1)
		}
		ret[l] = append(ret[l], scanner.Text())
	}

	return ret
}
