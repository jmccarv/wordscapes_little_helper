package main

import (
	"cmp"
	"log"
	"slices"
)

func findWords(wordList map[int][]string, freqList map[string]int, template, letters string) []string {
	found := false
	searchLen := len(template)
	ret := make([]string, 0)

	for _, l := range template {
		if l > 255 {
			log.Printf("Invalid character '%v' in template, aborting search\n", l)
			return []string{}
		}
	}

	var letterTab [256]int
	for _, l := range letters {
		if l > 256 {
			log.Printf("Invalid character '%v' in candidate letters, aborting search\n", l)
			return []string{}
		}
		letterTab[l]++
	}

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

	slices.SortFunc(ret, func(a, b string) int {
		af := freqList[a]
		bf := freqList[b]
		if af == bf {
			return cmp.Compare(a, b)
		}
		return cmp.Compare(af, bf) * -1
	})

	return ret
}
