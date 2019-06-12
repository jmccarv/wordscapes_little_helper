package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"
)

var flagServeHTTP bool
var flagLetters string
var flagTemplate string
var flagListFile string
var flagHost string

var wordList map[int][]string

func main() {
	//defer profile.Start().Stop()

	cli.AppHelpTemplate = fmt.Sprintf(`%s
Searches a wordlist for words matching a template using a specified set of letters
AKA a cheat program for the wordscapes game
`, cli.AppHelpTemplate)

	app := cli.NewApp()
	app.Name = "Wordscapes Little Helper"
	app.Usage = "Search a wordlist or something"
	app.HideVersion = true

	app.Commands = []cli.Command{
		{
			Name:    "serve",
			Aliases: []string{"s"},
			Usage:   "Serve requests over HTTP",
			Action:  serveHTTP,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "host, l",
					Usage:       "Host to listen on, may include port",
					Value:       "localhost:8080",
					EnvVar:      "WLH_HOST",
					Destination: &flagHost,
				},
			},
		},
		{
			Name:    "find",
			Aliases: []string{"f"},
			Usage:   "Oneshot search",
			Action:  oneshotSearch,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "template, t",
					Usage:       "Template to search for, non-alpha for any letter, ex: 'a...' to find all four letter words that start with 'a'",
					Destination: &flagTemplate,
				},
				cli.StringFlag{
					Name:        "letters, l",
					Usage:       "Available letters to use to make words, ex: 'ebsls' might be used to make the word 'bless'",
					Destination: &flagLetters,
				},
			},
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "wordlist, w",
			Usage:       "Read wordlist from `FILE`, one word per line",
			Value:       "/dev/stdin",
			EnvVar:      "WLH_WORDLIST",
			Destination: &flagListFile,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
