// extract_wordlist parses a wiktionary XML dump into a list of english words.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	//"github.com/pkg/profile"

	"github.com/urfave/cli"
)

var minWordLength int
var maxWordLength int
var includeMixedCase bool

type Word struct {
	word       string
	deps       map[string]bool // any dependencies this word has that must be valid for this word to be valid
	isValid    bool            // true if the word has been validated and is a valid word
	validated  bool            // true if the word has gone through validation. if false, you cannot rely on the value of isValid
	validating bool            // used to catch infinite validation loops
}

// As words are parsed, they are placed in this map to weed out duplicates. The
// map is keyed on the word.
type WordMap map[string]*Word

// As we read in buffers of data, this finds the last page in the buffer. Used
// to send as many <page>..</page> pages as possible at a time to the parsing
// goroutines.
func splitPages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	found := bytes.LastIndex(data, []byte("</page>"))
	if found >= 0 {
		return found + 7, data[0 : found+7], nil

	} else {
		return 0, nil, nil
	}
}

// This method will validate words that have dependencies by verifying
// that at least one dependency is valid. If no dependencies are valid
// this word is invalid.
func (words WordMap) validateWord(elem *Word) (valid, cycle bool) {
	var c bool
	//log.Printf("validating %v", elem.word)

	if elem.validated {
		//log.Printf("vw: (validated) returning %v", elem.isValid)
		return elem.isValid, false
	}

	if elem.validating {
		return false, true
	}

	elem.validating = true
	for p, _ := range elem.deps {
		e, ok := words[p]
		if !ok {
			continue
		}

		elem.isValid, c = words.validateWord(e)
		if c {
			// We cycled, which shouldn't happen
			// but I suppose we'll consider this a valid word anyway
			log.Printf("Cycle detected while validating plurals of '%v'", e.word)
			elem.isValid = true
			break
		}

		if elem.isValid {
			break
		}
	}
	elem.validating = false
	elem.validated = true

	return elem.isValid, false
}

func run(c *cli.Context) error {
	nrCPU := runtime.GOMAXPROCS(0)

	cPage := make(chan []byte, nrCPU*2)
	cWord := make(chan *Word, nrCPU)
	cResults := make(chan WordMap)

	// This is the scanner we'll use to split the XML into
	// blocks of data. Each block will contain one or more
	// groups of <page>..</page> elements. Each block is written
	// to the cPage channel to be parsed by the parsePageBlock goroutine.
	fh := bufio.NewReader(os.Stdin)
	scanner := bufio.NewScanner(fh)
	scanner.Split(splitPages)
	scanner.Buffer(make([]byte, 1024*1024*10), 1024*1024*100)

	for i := 0; i < nrCPU; i++ {
		go parsePageBlock(cPage, cWord)
	}
	go gatherWords(cWord, cResults, nrCPU)

	for scanner.Scan() {
		page := make([]byte, len(scanner.Bytes()))
		copy(page, scanner.Bytes())
		cPage <- page
	}

	if scanner.Err() != nil {
		log.Fatalf("scanner error: %v\n", scanner.Err().Error())
	}

	// Signal there are no more pages to process.
	for i := 0; i < nrCPU; i++ {
		cPage <- nil
	}

	// Get the results from the gatherWords goroutine.
	words := <-cResults

	// Now we'll filter words for length, validate any words that have
	// dependencies, and add all valid words to an array.
	wordList := make([]string, 0, len(words))
	for _, e := range words {
		if v, _ := words.validateWord(e); v {
			if len(e.word) >= minWordLength && (maxWordLength == 0 || len(e.word) <= maxWordLength) {
				wordList = append(wordList, e.word)
			}
		}
	}

	// Sort and output our results.
	sort.Strings(wordList)
	for _, w := range wordList {
		fmt.Println(w)
	}

	return nil
}

func main() {
	//defer profile.Start().Stop()

	cli.AppHelpTemplate = fmt.Sprintf(`%s
Reads XML on stdin and writes parsed words to stdout
`, cli.AppHelpTemplate)

	app := cli.NewApp()
	app.Name = "extract_wordlist"
	app.Usage = "Parse wordlist from an wiktionary XML dump"
	app.HideVersion = true

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "min-length, m",
			Value:       3,
			Usage:       "Minimum length word to output",
			Destination: &minWordLength,
		},
		cli.IntFlag{
			Name:        "max-length, l",
			Value:       7,
			Usage:       "Maximum length word to output",
			Destination: &maxWordLength,
		},
		cli.BoolFlag{
			Name:        "mixed-case, i",
			Usage:       "Include words with upper case letters",
			Destination: &includeMixedCase,
		},
	}

	app.Action = run

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
