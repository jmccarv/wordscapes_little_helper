package main

import (
	"log"
)

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
