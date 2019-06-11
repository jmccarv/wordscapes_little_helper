package main

// Goroutine to gather words as they're parsed from XML and
// add them to a results map. If a word has no dependencies
// it can be marked as valid.
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

			if len(elem.deps) != 0 {
				elem.validated = false
				elem.isValid = false
			}

		} else {
			// Copy element e to our words map
			words[elem.word] = elem
		}

		//log.Printf("gw: %v dep count %v", elem.word, len(elem.deps))
	}
}
