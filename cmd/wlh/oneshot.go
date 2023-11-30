package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli"
)

func oneshotSearch(c *cli.Context) error {
	//nrcpu := runtime.GOMAXPROCS(0)

	start := time.Now()
	wordList = readWordList(flagListFile)
	freqList = readFreqList(flagFreqFile)
	log.Printf("Loaded wordlists in %v", time.Now().Sub(start))

	template := flagTemplate
	letters := flagLetters
	if len(template) < 1 || len(letters) < 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	start = time.Now()

	for _, word := range findWords(wordList, freqList, template, letters) {
		fmt.Println(word)
	}
	log.Printf("Search time: %v\n", time.Now().Sub(start))

	return nil
}
