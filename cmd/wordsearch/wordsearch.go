package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli"
)

var flagServeHTTP bool
var flagLetters string
var flagTemplate string
var flagListFile string

var wordList map[int][]string

func search(w http.ResponseWriter, req *http.Request) {
	//io.WriteString(w, fmt.
	//fmt.Fprintf(w, "%v\n%v\n", req.Method, req.URL.RawQuery)
	//io.WriteString(w, "Test\n")

	w.Header().Add("Access-Control-Allow-Origin", "*")

	query, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		log.Print(err)
		return
	}

	if len(query["letters"]) < 1 {
		log.Print("Invalid letters")
		return
	}
	letters := strings.ToLower(query["letters"][0])

	if len(query["template"]) < 1 {
		log.Print("Invalid template")
		return
	}
	template := strings.ToLower(query["template"][0])

	if len(letters) < 1 || len(template) < 1 || len(template) > len(letters) {
		log.Print("Invalid parameters")
		return
	}

	start := time.Now()

	// Generate array of characters that are availabe
	var letterTab [256]int
	for _, l := range letters {
		letterTab[l]++
	}

	json.NewEncoder(w).Encode(findWords(letterTab, wordList, template, letters))

	log.Printf("Search time: %v\n", time.Now().Sub(start))
}

func run() {
	//nrcpu := runtime.GOMAXPROCS(0)

	flag.Parse()

	start := time.Now()
	wordList = slurp(flagListFile)
	log.Printf("Loaded wordlist in %v", time.Now().Sub(start))

	if flagServeHTTP {
		http.HandleFunc("/search", search)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}

	// Not running in server mode, do the search from command line flags

	template := flagTemplate
	letters := flagLetters
	if len(template) < 1 || len(letters) < 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	start = time.Now()

	// Generate array of characters that are availabe
	var letterTab [256]int
	for _, l := range letters {
		letterTab[l]++
	}

	for _, word := range findWords(letterTab, wordList, template, letters) {
		fmt.Println(word)
	}
	log.Printf("Search time: %v\n", time.Now().Sub(start))

}

func findWords(letterTab [256]int, wordList map[int][]string, template, letters string) []string {
	found := false
	searchLen := len(template)
	ret := make([]string, 0)

	log.Printf("Searching %v words for %v letter long words from letters '%v' which fit template |%v|\n", len(wordList[searchLen]), searchLen, letters, template)

	for _, word := range wordList[searchLen] {
		if len(word) != searchLen {
			continue
		}

		l := letterTab
		found = true
		for i := 0; i < searchLen; i++ {
			w := word[i]

			if l[w] == 0 {
				found = false
				break
			}

			t := template[i]
			if t >= 'a' && t <= 'z' && t != w {
				found = false
				break
			}

			l[w]--
		}

		if found {
			ret = append(ret, word)
		}
	}

	return ret
}

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

func main() {
	//defer profile.Start().Stop()

	cli.AppHelpTemplate = fmt.Sprintf(`%s
Searches a wordlist for words matching a template using a specified set of letters
AKA a cheat program for the wordscapes game
`, cli.AppHelpTemplate)

	app := cli.NewApp()
	app.Name = "wordsearch"
	app.Usage = "Search a wordlist or something"
	app.HideVersion = true

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "serve-http, d",
			Usage:       "Listen for requests on HTTP",
			Destination: &flagServeHTTP,
		},
		cli.StringFlag{
			Name:        "wordlist, w",
			Usage:       "Wordlist file to read, one word per line",
			Value:       "/dev/stdin",
			EnvVar:      "WORDSEARCH_WORDLIST",
			Destination: &flagListFile,
		},
		cli.StringFlag{
			Name:        "template, t",
			Usage:       "Template to search for, spaces for any letter, ex: 'a...' to find all four letter words that start with 'a'",
			Destination: &flagTemplate,
		},
		cli.StringFlag{
			Name:        "letters, l",
			Usage:       "Available letters to use to make words, ex: 'ebsls' might be used to make the word 'bless'",
			Destination: &flagLetters,
		},
	}

	app.Action = run

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
