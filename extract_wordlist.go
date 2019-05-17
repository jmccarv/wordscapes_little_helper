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
    //"log"
)

var minWordLength    int
var maxWordLength    int
var includeMixedCase bool

func init() {
    const (
        defaultMinLength = 1
        defaultMaxLength = 0
        defaultIncludeMixedCase = false
        minLengthUsage = "Minimum length word to output"
        maxLengthUsage = "Maximum length word to output (default 0 - no maximum)"
        includeMixedCaseUsage = "Include words with upper case letters"
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

type parseResults struct {
    word []string
    plural map[string][]string
}

func parsePage(cPage chan []byte, cResults chan parseResults) {
    var res parseResults
    res.word = make([]string, 0, 1024)
    res.plural = make(map[string][]string, 1024)

    whitelist := map[string]bool{"initialism": true, "surname": true}

    rxValidWord := regexp.MustCompile(`^[a-z]+$`)
    rxIgnore  := regexp.MustCompile(`(?i:initialism|surname\|lang=en[^a-z])`)
    rxPlural  := regexp.MustCompile(`plural of\|(\w+)\|lang=en[^a-z]`)
    rxEnglish := regexp.MustCompile(`==English==|Category:(en[^a-z]|English)`)

    NEXTPAGE:
    for {
        data := <- cPage
        if data == nil {
            cResults <- res
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

                    word := p.Title
                    if includeMixedCase {
                        word = strings.ToLower(word)
                    }

                    if whitelist[word] {
                        res.word = append(res.word, word)
                        continue NEXTPAGE
                    }

                    if len(p.Revisions) == 0 {
                        continue NEXTPAGE
                    }

                    if !rxValidWord.MatchString(word) {
                        continue NEXTPAGE
                    }

                    rev := p.Revisions[len(p.Revisions)-1]

                    if rxIgnore.MatchString(rev.Text) {
                        continue NEXTPAGE
                    }


                    // If it's a plural we need to track that, but we
                    // still need to add it to our word list since
                    // there are weird words like 'spices' which is
                    // a plural of 'spice' which is a plural of 'spouse'
                    //
                    // Also, there are words like petties that has two
                    // entries, one is 'petties' and one is 'Petties'
                    // with the Uppser case being the plural of a surname
                    match := rxPlural.FindStringSubmatch(rev.Text)
                    if len(match) > 0 {
                        p := strings.ToLower(match[1])
                        if p != word {
                            res.plural[word] = append(res.plural[word], p)
                        }
                    }

                    if rxEnglish.MatchString(rev.Text) {
                        res.word = append(res.word, word)
                        continue NEXTPAGE
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


func main() {
    nrCPU := runtime.GOMAXPROCS(0)

    flag.Parse()

    cPage := make(chan []byte, nrCPU)
    cResults := make(chan parseResults)

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
        fmt.Printf("scanner error: %v\n", scanner.Err().Error())
    }

    for i := 0; i < nrCPU; i++ {
        cPage <- nil
    }

    wordMap := make(map[string]bool, 1024)
    plural := make(map[string][]string, 1024)

    for i := 0; i < nrCPU; i++ {
        res := <- cResults

        for _, w := range(res.word) {
            wordMap[w] = true
        }

        for k,v := range(res.plural) {
            plural[k] = append(plural[k], v...)
        }
    }

    for k,v := range(plural) {
        // Remove any plural words whose base word didn't make it into the list
        // A plural may have multiple base words, if any of those base words
        // made it to the list so should the plural form
        del := true
        for _, z := range(v) {
            if wordMap[z] {
                del = false
                break
            }
        }

        if del {
            delete(wordMap, k)
        }
    }


    words := make([]string, 0, len(wordMap))
    for k,_ := range(wordMap) {
        words = append(words, k)
    }

    sort.Strings(words)
    for _,w := range(words) {
        if len(w) >= minWordLength && (maxWordLength == 0 || len(w) <= maxWordLength) {
            fmt.Println(w)
        }
    }
}
