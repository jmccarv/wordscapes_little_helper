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
)

var minWordLength int
var maxWordLength int

func init() {
    const (
        defaultMinLength = 1
        defaultMaxLength = 0
        minLengthUsage = "Minimum length word to output"
        maxLengthUsage = "Maximum length word to output (default 0 - no maximum)"
    )

    flag.IntVar(&minWordLength, "min-length", defaultMinLength, minLengthUsage)
    flag.IntVar(&minWordLength, "m", defaultMinLength, minLengthUsage+" (short)")
    flag.IntVar(&maxWordLength, "max-length", defaultMaxLength, maxLengthUsage)
    flag.IntVar(&maxWordLength, "l", defaultMaxLength, maxLengthUsage+" (short)")
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
    plural map[string]string
}

func parsePage(cPage chan []byte, cResults chan parseResults) {
    var res parseResults
    res.word = make([]string, 0, 1024)
    res.plural = make(map[string]string)

    whitelist := map[string]bool{"initialism": true, "surname": true}

    rxValidWord := regexp.MustCompile(`(?i:^[a-z]+$)`)
    rxIgnore  := regexp.MustCompile(`(?i:initialism|surname\|lang=en)`)
    rxPlural  := regexp.MustCompile(`plural of\|(\w+)\|lang=en`)
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

                    if whitelist[p.Title] {
                        res.word = append(res.word, p.Title)
                        continue NEXTPAGE
                    }

                    if len(p.Revisions) == 0 {
                        continue NEXTPAGE
                    }

                    if !rxValidWord.MatchString(p.Title) {
                        continue NEXTPAGE
                    }

                    rev := p.Revisions[len(p.Revisions)-1]

                    if rxIgnore.MatchString(rev.Text) {
                        continue NEXTPAGE
                    }

                    word := strings.ToLower(p.Title)

                    match := rxPlural.FindStringSubmatch(rev.Text)
                    if len(match) > 0 {
                        p := strings.ToLower(match[1])
                        if p != word {
                            res.plural[word] = p
                            continue NEXTPAGE
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

    cPage := make(chan []byte, nrCPU*2)
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
    plural := make(map[string]string, 1024)

    for i := 0; i < nrCPU; i++ {
        res := <- cResults

        for _, w := range(res.word) {
            wordMap[w] = true
        }

        for k,v := range(res.plural) {
            plural[k] = v
        }
    }

    for k,v := range(plural) {
        if (wordMap[v]) {
            wordMap[k] = true
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
