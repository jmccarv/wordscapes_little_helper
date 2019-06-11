package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/urfave/cli"
)

func logger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s requested %s", r.RemoteAddr, r.URL)
		h.ServeHTTP(w, r)
	})
}

func search(w http.ResponseWriter, req *http.Request) {
	//fmt.Fprintf(w, "%v\n%v\n", req.Method, req.URL.RawQuery)

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

func serveHTTP(c *cli.Context) error {
	start := time.Now()
	wordList = slurp(flagListFile)
	log.Printf("Loaded wordlist in %v", time.Now().Sub(start))

	h := http.NewServeMux()
	h.Handle("/", http.FileServer(FileSystem("www/html")))
	h.Handle("/js/", http.FileServer(FileSystem("www")))
	h.HandleFunc("/api/search/", search)

	log.Printf("Listening on %v", flagHost)
	log.Fatal(http.ListenAndServe(flagHost, logger(h)))

	return nil
}
