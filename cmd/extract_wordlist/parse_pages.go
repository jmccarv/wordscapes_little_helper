package main

import (
	"bytes"
	"regexp"
	"strings"
)

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

	// Words to ignore/skip
	rxIgnore := regexp.MustCompile(`(initialism|archaic spelling) of\|[^|]+\|lang=en|surname\|lang=en[^a-z]`)

	// We're only interested in English words for this list.
	rxEnglish := regexp.MustCompile(`==English==|Category:(en[^a-z]|English)`)

	// Find dependencies. At least one dependency must be valid for a word to be valid.
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

		// A nil block signals the end of input and that there's no more work
		// to do. Time to exit. We send a nil word along to the gather coroutine
		// so it knows to finish up.
		if block == nil {
			cWord <- nil
			return
		}

		// Each word is stored in a separate wiki <page>..</page>
		// The word itself is the <title>word</title> of the page
		// There may be one or more <revision>s that hold <text>.
		// We search the <text>..</text> to know if the word is valid.
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
				word:      word,
				deps:      make(map[string]bool, 1),
				validated: true,
				isValid:   true,
			}

			// If it's a plural we need to track that, but we
			// still need to add it to our word list since
			// there are weird words like 'spices' which is
			// a plural of 'spice' which is a plural of 'spouse'
			//
			// Also, there are words like petties that has two
			// entries, one is 'petties' and one is 'Petties'
			// with the Uppser case being the plural of a surname
			//
			// FIXME: Handle multiple matches of this regex
			d := rxDep.Find(text[0])
			if d != nil {
				w := rxDepWord.FindSubmatch(d)

				if w == nil {
					continue
				}

				if !strings.EqualFold(elem.word, string(w[0])) {
					elem.deps[string(w[1])] = true
					elem.validated = false
					elem.isValid = false
				}
			}

			cWord <- elem
		}
	}
}
