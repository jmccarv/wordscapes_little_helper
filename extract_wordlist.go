package main

import (
    "bufio"
    "os"
    "fmt"
    "bytes"
    "runtime"
    "regexp"
    "sort"
    "strings"
    "flag"
    "log"
    //"github.com/pkg/profile"
)

var minWordLength    int
var maxWordLength    int
var includeMixedCase bool
var nrCPU int

func init() {
    const (
        defaultMinLength = 1
        defaultMaxLength = 0
        defaultIncludeMixedCase = false
        minLengthUsage = "Minimum length word to output"
        maxLengthUsage = "Maximum length word to output (default 0 - no maximum)"
        includeMixedCaseUsage = "Include words with upper case letters (results will be folded to lower case)"
    )

    flag.IntVar(&minWordLength, "min-length", defaultMinLength, minLengthUsage)
    flag.IntVar(&minWordLength, "m", defaultMinLength, minLengthUsage+" (short)")
    flag.IntVar(&maxWordLength, "max-length", defaultMaxLength, maxLengthUsage)
    flag.IntVar(&maxWordLength, "l", defaultMaxLength, maxLengthUsage+" (short)")
    flag.BoolVar(&includeMixedCase, "mixed-case", defaultIncludeMixedCase, includeMixedCaseUsage)
    flag.BoolVar(&includeMixedCase, "i", defaultIncludeMixedCase, includeMixedCaseUsage+" (short)")
}

type Word struct {
    word string
    pluralOf map[string]bool
    validated bool
    validating bool
}

// I had previously used the xml module to parse pages more "correctly"
// This not-at-all correct parser is much faster
// On my test system the runtime of this program on the wiktionary dump
// ran in 1.5 minutes using xml and about .5 minutes this way
func parsePageBlock(cPage chan []byte, cResults chan map[string]*Word) {
    words := make(map[string]*Word, 1024)
    var elem *Word
    var ok bool

    tagTitle := []byte("<title>")
    tagTitleEnd := []byte("</title>")
    tagRevision := []byte("<revision>")
    tagText := []byte("<text")
    tagTextEnd := []byte("</text>")

    var rxValidWord *regexp.Regexp
    if (includeMixedCase) {
        rxValidWord = regexp.MustCompile(`^[a-zA-Z]+$`)
    } else {
        rxValidWord = regexp.MustCompile(`^[a-z]+$`)
    }

    rxIgnore  := regexp.MustCompile(`initialism of\|[^|]+\|lang=en|surname\|lang=en[^a-z]`)
    rxPlural  := regexp.MustCompile(`plural of\|(\w+)\|lang=en[^a-z]`)
    rxEnglish := regexp.MustCompile(`==English==|Category:(en[^a-z]|English)`)

    wordOk := func(word string, text []byte) bool {
        if rxIgnore.Match(text) {
            return false
        }

        if rxEnglish.Match(text) {
            return true
        }

        return false
    }

    for {
        block := <- cPage
        if block == nil {
            cResults <- words
            return
        }

        NEXTPAGE:
        for _,data := range(bytes.SplitAfter(block, []byte("</page>"))) {
            title := bytes.SplitN(data, tagTitle, 2)
            if len(title) != 2 {
                continue NEXTPAGE
            }

            title = bytes.SplitN(title[1], tagTitleEnd, 2)
            if len(title) != 2 {
                continue NEXTPAGE
            }

            word := string(title[0])

            if !rxValidWord.MatchString(word) {
                continue NEXTPAGE
            }

            revisions := bytes.Split(data, tagRevision)
            if len(revisions) < 1 {
                continue NEXTPAGE
            }

            text := bytes.SplitN(revisions[len(revisions)-1], tagText, 2)
            if len(text) != 2 {
                continue NEXTPAGE
            }

            text = bytes.SplitN(text[1], tagTextEnd, 2)
            if len(text) != 2 {
                continue NEXTPAGE
            }

            if !wordOk(word, text[0]) {
                continue NEXTPAGE
            }

            if elem, ok = words[word]; !ok {
                // This is the first time we've seen this word
                // so we need to add it to our map
                //elem = &Word{ p.Title, make(map[string]bool,1), false }
                elem = &Word{ word: word, pluralOf: make(map[string]bool,1) }
                words[word] = elem
            }

            // If it's a plural we need to track that, but we
            // still need to add it to our word list since
            // there are weird words like 'spices' which is
            // a plural of 'spice' which is a plural of 'spouse'
            //
            // Also, there are words like petties that has two
            // entries, one is 'petties' and one is 'Petties'
            // with the Uppser case being the plural of a surname
            match := rxPlural.FindSubmatch(text[0])
            if len(match) > 1 && !strings.EqualFold(elem.word, string(match[1])) {
                elem.pluralOf[string(match[1])] = true
            }
        }
    }
}


func splitPages(data []byte, atEOF bool) (advance int, token []byte, err error) {
    if (atEOF && len(data) == 0) {
        return 0, nil, nil
    }

    found := bytes.LastIndex(data, []byte("</page>"))
    if (found >= 0) {
        return found+7, data[0:found+7], nil

    } else {
        return 0, nil, nil
    }
}

func consolidateWords(cResults chan map[string]*Word) map[string]*Word {
    words := make(map[string]*Word, 1024)

    for i := 0; i < nrCPU; i++ {
        ret := <- cResults

        for w,e := range(ret) {
            if elem, ok := words[w]; ok {
                // Copy any plurals from ret to our existing word words[k]
                for k,_ := range(e.pluralOf) {
                    elem.pluralOf[k] = true
                }

            } else {
                // Copy element e to our words map
                words[w] = e
            }

            if len(words[w].pluralOf) == 0 {
                words[w].validated = true
            }
        }
    }

    //fmt.Printf("cw: count: %v\n", len(words))
    return words
}

func validateWord(w string, words map[string]*Word) (cycle bool) {
    //var elem *Word
    elem, ok := words[w]
    if !ok {
        return
    }

    if (elem.validating) {
        return true
    }

    isValid := len(elem.pluralOf) == 0

    elem.validating = true
    for p,_ := range(elem.pluralOf) {
        //fmt.Printf("Validating plural %v => %v\n", w, p)
        if validateWord(p, words) {
            // We cycled, which shouldn't happen
            // but I suppose we'll consider this a valid word anyway
            log.Printf("Cycle detected while validating %v", w)
            isValid =true
            break
        }

        if _, ok := words[p]; ok {
            isValid = true
            break
        }
    }
    elem.validating = false

    if isValid {
        elem.validated = true
    } else {
        //fmt.Printf("Removing invalid plural %v\n", w)
        delete(words, w)
    }

    return
}

// This will remove any plural words whose base
// words aren't valid (not in the list)
// This isn't as simple as it seems :(
func validateWords(words map[string]*Word) map[string]*Word {
    for k,_ := range(words) {
        validateWord(k, words)
    }
    return words
}

func main() {
    //defer profile.Start().Stop()
    nrCPU = runtime.GOMAXPROCS(0)

    flag.Parse()

    cPage := make(chan []byte, nrCPU*2)
    cResults := make(chan map[string]*Word)

    fh := bufio.NewReader(os.Stdin)

    scanner := bufio.NewScanner(fh)
    scanner.Split(splitPages)
    scanner.Buffer(make([]byte, 1024*1024*10), 1024*1024*100)

    for i:= 0; i < nrCPU; i++ {
        go parsePageBlock(cPage, cResults)
    }

    nrFound := 0
    emptyCount := 0
    for scanner.Scan() {
        nrFound++

        page := make([]byte, len(scanner.Bytes()))
        copy(page, scanner.Bytes())
        if len(cPage) == 0 {
            emptyCount++
        }
        cPage <- page
    }

    log.Printf("Channel was empty %v times out of %v blocks (empty %.2f%%)\n", emptyCount, nrFound, (float64(emptyCount)/float64(nrFound))*100.0)

    if scanner.Err() != nil {
        log.Fatalf("scanner error: %v\n", scanner.Err().Error())
    }

    // Signal there are no more pages to process, this will cause the
    // worders to write their results to the cResults channel
    for i := 0; i < nrCPU; i++ {
        cPage <- nil
    }

    // Consolidate all parsed words from the goroutines into one map
    //words := validateWords(consolidateWords(cResults))
    words := make(map[string]bool, 1024)
    for w,_ := range(validateWords(consolidateWords(cResults))) {
        words[strings.ToLower(w)] = true
    }

    wordList := make([]string, 0, len(words))
    for k,_ := range(words) {
        wordList = append(wordList, k)
    }

    sort.Strings(wordList)
    for _,w := range(wordList) {
        if len(w) >= minWordLength && (maxWordLength == 0 || len(w) <= maxWordLength) {
            fmt.Println(w)
        }
    }
}
