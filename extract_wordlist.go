package main

import (
    "bufio"
    "os"
    "fmt"
    "bytes"
    "encoding/xml"
    "runtime"
    "regexp"
    "sort"
    "strings"
    "flag"
    "log"
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

type Revision struct {
    Text string `xml:"text"`
}

type Page struct {
    Title string `xml:"title"`
    Revisions []Revision `xml:"revision"`
}


type Word struct {
    word string
    pluralOf map[string]bool
    validated bool
    validating bool
}

func parsePage(cPage chan []byte, cResults chan map[string]*Word) {
    words := make(map[string]*Word, 1024)
    var elem *Word
    var ok bool
    var rxValidWord *regexp.Regexp

    whitelist := map[string]bool{"initialism": true, "surname": true}

    if (includeMixedCase) {
        rxValidWord = regexp.MustCompile(`^[a-zA-Z]+$`)
    } else {
        rxValidWord = regexp.MustCompile(`^[a-z]+$`)
    }

    rxIgnore  := regexp.MustCompile(`(?i:initialism of\|[^|]+\|lang=en|surname\|lang=en[^a-z])`)
    rxPlural  := regexp.MustCompile(`plural of\|(\w+)\|lang=en[^a-z]`)
    rxEnglish := regexp.MustCompile(`==English==|Category:(en[^a-z]|English)`)

    wordOk := func(p Page) bool {
        if whitelist[p.Title] {
            return true
        }

        if len(p.Revisions) == 0 {
            return false
        }

        if !rxValidWord.MatchString(p.Title) {
            return false
        }

        rev := p.Revisions[len(p.Revisions)-1]

        if rxIgnore.MatchString(rev.Text) {
            return false
        }

        if rxEnglish.MatchString(rev.Text) {
            return true
        }

        return false
    }

    NEXTPAGE:
    for {
        data := <- cPage
        if data == nil {
            cResults <- words
            return
        }

        decoder := xml.NewDecoder(bytes.NewReader(data))
        decoder.Strict = false

        for {
            // Read tokens from XML document
            t, _ := decoder.Token()
            if t == nil {
                break
            }

            // Check for page start elements
            switch et := t.(type) {
            case xml.StartElement:
                if et.Name.Local == "page" {
                    var p Page
                    // decode this page
                    decoder.DecodeElement(&p, &et)

                    if !wordOk(p) {
                        continue NEXTPAGE
                    }

                    if elem, ok = words[p.Title]; !ok {
                        // This is the first time we've seen this word
                        // so we need to add it to our map
                        //elem = &Word{ p.Title, make(map[string]bool,1), false }
                        elem = &Word{ word: p.Title, pluralOf: make(map[string]bool,1) }
                        words[p.Title] = elem
                    }

                    // If it's a plural we need to track that, but we
                    // still need to add it to our word list since
                    // there are weird words like 'spices' which is
                    // a plural of 'spice' which is a plural of 'spouse'
                    //
                    // Also, there are words like petties that has two
                    // entries, one is 'petties' and one is 'Petties'
                    // with the Uppser case being the plural of a surname
                    rev := p.Revisions[len(p.Revisions)-1]
                    match := rxPlural.FindStringSubmatch(rev.Text)
                    if len(match) > 1 && !strings.EqualFold(elem.word, match[1]) {
                        elem.pluralOf[match[1]] = true
                    }
                }
            }
        }
    }
}


func splitPages(data []byte, atEOF bool) (advance int, token []byte, err error) {
    if (atEOF && len(data) == 0) {
        return 0, nil, nil
    }

    found := bytes.Index(data, []byte("</page>"))
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
    nrCPU = runtime.GOMAXPROCS(0)

    flag.Parse()

    cPage := make(chan []byte, nrCPU)
    cResults := make(chan map[string]*Word)

    fh := bufio.NewReader(os.Stdin)

    scanner := bufio.NewScanner(fh)
    scanner.Split(splitPages)
    scanner.Buffer(make([]byte, 1024*1024), 1024*1024*10)

    for i:= 0; i < nrCPU; i++ {
        go parsePage(cPage, cResults)
    }

    nrFound := 0
    for scanner.Scan() {
        nrFound++

        page := make([]byte, len(scanner.Bytes()))
        copy(page, scanner.Bytes())
        cPage <- page
    }

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
