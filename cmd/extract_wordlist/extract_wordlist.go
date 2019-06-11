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
	deps       map[string]bool
	isValid    bool
	validated  bool
	validating bool
}

type WordMap map[string]*Word

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

	// Set up the scanner
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
	words := <-cResults

	wordList := make([]string, 0, len(words))
	for _, e := range words {
		if v, _ := words.validateWord(e); v {
			if len(e.word) >= minWordLength && (maxWordLength == 0 || len(e.word) <= maxWordLength) {
				wordList = append(wordList, e.word)
			}
		}
	}

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
