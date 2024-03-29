package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
)

var tmpl *template.Template

type TmplBox struct {
	Name  string
	Value string
}
type wlhState struct {
	Tmpl    []TmplBox
	Letters string
	Results []string
}

func logger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s requested %s", r.RemoteAddr, r.URL)
		h.ServeHTTP(w, r)
	})
}

func stateFromReq(req *http.Request) (wlhState, error) {
	var state wlhState

	if err := req.ParseForm(); err != nil {
		return state, err
	}
	//log.Printf("%v\n%v\n%v\n", req.Method, req.URL.RawQuery, req.Form)

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("t%d", i)
		if v, ok := req.Form[name]; ok {
			// boxes should contain only a single character
			l := v[0]
			if len(l) > 1 {
				l = l[:1]
			}
			state.Tmpl = append(state.Tmpl, TmplBox{Name: name, Value: l})
		} else {
			break
		}
	}
	state.Letters = req.Form.Get("Letters")

	return state, nil
}

func search(w http.ResponseWriter, req *http.Request) {
	doReq(w, req, tmpl.Lookup("results.tmpl"), nil)
}

func boxRemove(w http.ResponseWriter, req *http.Request) {
	doReq(w, req, nil, func(s wlhState) wlhState {
		if len(s.Tmpl) > 3 {
			s.Tmpl = s.Tmpl[:len(s.Tmpl)-1]
		}
		for i := range s.Tmpl {
			s.Tmpl[i].Value = ""
		}
		return s
	})
}

func boxAdd(w http.ResponseWriter, req *http.Request) {
	doReq(w, req, nil, func(s wlhState) wlhState {
		for i := range s.Tmpl {
			s.Tmpl[i].Value = ""
		}
		if len(s.Tmpl) < 10 {
			s.Tmpl = append(s.Tmpl, TmplBox{Name: fmt.Sprintf("t%d", len(s.Tmpl))})
		}
		return s
	})
}

func clear(w http.ResponseWriter, req *http.Request) {
	doReq(w, req, nil, func(s wlhState) wlhState {
		s.Letters = ""
		for i := range s.Tmpl {
			s.Tmpl[i].Value = ""
		}
		return s
	})
}

func doSearch(state wlhState) wlhState {
	template := ""
	letters := strings.ToLower(state.Letters)
	for _, v := range state.Tmpl {
		if v.Value == "" {
			v.Value = " "
		}
		template += strings.ToLower(v.Value)
	}

	if len(letters) < 1 || len(template) < 1 || len(template) > len(letters) {
		log.Println("Invalid parameters")
		return state
	}

	start := time.Now()

	state.Results = findWords(wordList, freqList, template, letters)
	log.Println("Found", len(state.Results), "words")

	log.Printf("Search time: %v\n", time.Now().Sub(start))
	return state
}

func doReq(w http.ResponseWriter, req *http.Request, t *template.Template, mut func(wlhState) wlhState) {
	//log.Printf("%+v\n", req)

	state, err := stateFromReq(req)
	if err != nil {
		log.Println(err)
	}

	if mut != nil {
		state = mut(state)
	}
	state = doSearch(state)

	if t == nil {
		t = tmpl.Lookup("page.tmpl")
	}
	err = t.Execute(w, state)
	if err != nil {
		log.Println(err)
	}
}

func serveHTTP(c *cli.Context) error {
	var state wlhState

	start := time.Now()
	wordList = readWordList(flagListFile)
	freqList = readFreqList(flagFreqFile)
	log.Printf("Loaded wordlists in %v", time.Now().Sub(start))

	for i := 0; i < 4; i++ {
		state.Tmpl = append(state.Tmpl, TmplBox{Name: "t" + strconv.Itoa(i)})
	}

	var err error
	tmpl, err = template.ParseGlob("tmpl/*.tmpl")
	if err != nil {
		panic(err)
	}

	h := http.NewServeMux()

	h.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		err = tmpl.ExecuteTemplate(w, "index.tmpl", state)
		if err != nil {
			log.Println(err)
		}
	})

	h.HandleFunc("/search/", search)
	h.HandleFunc("/box/remove/", boxRemove)
	h.HandleFunc("/box/add/", boxAdd)
	h.HandleFunc("/clear/", clear)

	log.Printf("Listening on %v", flagHost)
	return http.ListenAndServe(flagHost, logger(h))
}
