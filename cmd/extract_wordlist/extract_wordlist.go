package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
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

// I had previously used the xml module to parse pages more "correctly"
// This not-at-all correct parser is much faster
// On my test system the runtime of this program on the wiktionary dump
// ran in 1.5 minutes using xml and about .5 minutes this way
func parsePageBlock(cPage chan []byte, cWord chan *Word) {
	var rxValidWord *regexp.Regexp

	if includeMixedCase {
		rxValidWord = regexp.MustCompile(`^[a-zA-Z]+$`)
	} else {
		rxValidWord = regexp.MustCompile(`^[a-z]+$`)
	}

	rxIgnore := regexp.MustCompile(`(initialism|archaic spelling) of\|[^|]+\|lang=en|surname\|lang=en[^a-z]`)
	rxEnglish := regexp.MustCompile(`==English==|Category:(en[^a-z]|English)`)
	//rxDep := regexp.MustCompile(`{{(plural|alternative form) of.*\|lang=en[^a-z].*?}}`)
	rxDep := regexp.MustCompile(`{{plural of.*\|lang=en[^a-z].*?}}`)
	rxDepWord := regexp.MustCompile(`\|(\w+)(\||}})`)

	wordOk := func(word string, text []byte) bool {
		if !rxEnglish.Match(text) {
			return false
		}

		if rxIgnore.Match(text) {
			return false
		}

		return true
	}

	for {
		block := <-cPage
		if block == nil {
			cWord <- nil
			return
		}

		for _, data := range bytes.SplitAfter(block, []byte("</page>")) {
			title := bytes.SplitN(data, []byte("<title>"), 2)
			if len(title) != 2 {
				continue
			}

			title = bytes.SplitN(title[1], []byte("</title>"), 2)
			if len(title) != 2 {
				continue
			}

			word := string(title[0])

			if !rxValidWord.MatchString(word) {
				continue
			}

			revisions := bytes.Split(data, []byte("<revision>"))
			if len(revisions) < 1 {
				continue
			}

			text := bytes.SplitN(revisions[len(revisions)-1], []byte("<text"), 2)
			if len(text) != 2 {
				continue
			}

			text = bytes.SplitN(text[1], []byte("</text>"), 2)
			if len(text) != 2 {
				continue
			}

			if !wordOk(word, text[0]) {
				continue
			}

			elem := &Word{
				word: word,
				deps: make(map[string]bool, 1),
			}

			// If it's a plural we need to track that, but we
			// still need to add it to our word list since
			// there are weird words like 'spices' which is
			// a plural of 'spice' which is a plural of 'spouse'
			//
			// Also, there are words like petties that has two
			// entries, one is 'petties' and one is 'Petties'
			// with the Uppser case being the plural of a surname
			d := rxDep.Find(text[0])
			if d != nil {
				w := rxDepWord.FindSubmatch(d)

				if w == nil {
					continue
				}

				if !strings.EqualFold(elem.word, string(w[0])) {
					elem.deps[string(w[1])] = true
				}
			}

			cWord <- elem
		}
	}
}

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

func gatherWords(cWord chan *Word, cResults chan WordMap, nrCPU int) {
	words := make(WordMap)

	for {
		elem := <-cWord

		if elem == nil {
			nrCPU--
			if nrCPU == 0 {
				cResults <- words
				return
			}

			continue
		}

		if e, ok := words[elem.word]; ok {
			// Copy any plurals to the existing word
			// and use the existing element
			for k, _ := range elem.deps {
				e.deps[k] = true
			}
			elem = e

		} else {
			// Copy element e to our words map
			words[elem.word] = elem
		}

		//log.Printf("gw: %v dep count %v", elem.word, len(elem.deps))
		if len(elem.deps) == 0 {
			elem.validated = true
			elem.isValid = true
		} else {
			elem.validated = false
		}
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
