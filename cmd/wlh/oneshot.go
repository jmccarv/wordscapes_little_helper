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
	wordList = slurp(flagListFile)
	log.Printf("Loaded wordlist in %v", time.Now().Sub(start))

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

	return nil
}
