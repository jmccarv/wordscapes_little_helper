package main

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
