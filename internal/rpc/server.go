package rpc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/moson-mo/goaurrpc/internal/config"
	db "github.com/moson-mo/goaurrpc/internal/memdb"
)

// API server struct
type server struct {
	memDB      *db.MemoryDB
	mut        sync.RWMutex
	mutLimit   sync.RWMutex
	settings   config.Settings
	RateLimits map[string]RateLimit
}

// Creates a new server and immediately loads package data into memory
func New(settings config.Settings) (*server, error) {
	s := server{
		RateLimits: make(map[string]RateLimit),
	}
	s.settings = settings

	// load data
	fmt.Println("Loading package data...")
	err := s.reloadData()
	if err != nil {
		return nil, err
	}
	fmt.Println("Loaded package data.")
	return &s, nil
}

// Creates a rest API endpoint and starts listening for requests
func (s *server) Listen() error {
	// Starts a go routine that continuesly updates refreshes the package data
	go func() {
		for {
			time.Sleep(time.Duration(s.settings.RefreshInterval) * time.Second)
			fmt.Println("Reloading package data...")
			start := time.Now()
			err := s.reloadData()
			if err != nil {
				fmt.Println("Error reloading data: ", err)
				break
			}
			elapsed := time.Since(start)
			fmt.Println("Successfully reloaded package data in ", elapsed.Milliseconds(), " ms")
		}
	}()

	// remove rate limits if older than 24h
	go func() {
		time.Sleep(5 * time.Minute)
		s.mutLimit.Lock()
		for ip, rl := range s.RateLimits {
			if time.Since(rl.WindowStart).Hours() > 23 {
				delete(s.RateLimits, ip)
				fmt.Println("Removed rate limit for", ip)
			}
		}
		s.mutLimit.Unlock()
	}()

	// Listen for requests on /rpc
	http.HandleFunc("/rpc", s.rpcHandler)
	return http.ListenAndServe(":"+strconv.Itoa(s.settings.Port), nil)
}

// handles client connections
func (s *server) rpcHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Client connected:", r.RemoteAddr, "->", r.URL)

	// rate limit check
	if s.isRateLimited(r) {
		writeError(429, "Rate limit reached", w)
		return
	}

	// check if got a GET or POST request
	var qstr url.Values
	if r.Method == "GET" {
		qstr = r.URL.Query()
	} else {
		r.ParseForm()
		qstr = r.PostForm
	}
	t := qstr.Get("type")

	// if we don't get any query parameters, return documentation
	if len(qstr) == 0 {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		fmt.Fprintln(w, doc)
		return
	}

	// validate query parameters
	err := validateQueryString(qstr)
	if err != nil {
		writeError(200, err.Error(), w)
		return
	}

	// handle suggest calls
	if t == "suggest" || t == "suggest-pkgbase" {
		s.mut.RLock()
		defer s.mut.RUnlock()
		b, err := json.Marshal(s.rpcSuggest(qstr, (t == "suggest-pkgbase")))
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "")
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
		return
	}

	// handle info / search calls
	var result RpcResult
	s.mut.RLock()
	switch t {
	case "info":
		result = s.rpcInfo(qstr)
	case "multiinfo":
		result = s.rpcInfo(qstr)
	case "search":
		result = s.rpcSearch(qstr)
	case "msearch":
		result = s.rpcSearch(qstr)
	default:
		result = RpcResult{}
	}
	s.mut.RUnlock()

	// set number of records
	result.Resultcount = len(result.Results)
	if result.Resultcount == 0 {
		result.Results = make([]interface{}, 0)
	}

	// don't return data if we exceed max number of results
	if result.Resultcount > s.settings.MaxResults {
		result.Error = "Too many package results."
		result.Resultcount = 0
		result.Results = nil
		result.Type = "error"
	}

	// return JSON to client
	writeResult(&result, w)
}

// check if rate limit is reached. Create / update the record.
func (s *server) isRateLimited(r *http.Request) bool {
	s.mutLimit.Lock()
	defer s.mutLimit.Unlock()

	// RateLimit of 0 -> Skip check
	if s.settings.RateLimit == 0 {
		return false
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	la, ok := s.RateLimits[ip]
	if ok {
		la.Requests++
		s.RateLimits[ip] = la
		if la.Requests > s.settings.RateLimit {
			return true
		}
	} else {
		fmt.Println("Rate limit added", ip)
		s.RateLimits[ip] = RateLimit{
			Requests:    1,
			WindowStart: time.Now(),
		}
	}
	return false
}

// load data from file/url
func (s *server) reloadData() error {
	/*
		use local file for extensive testing -> ptr, err := db.LoadDbFromFile("packages.json")
		we don't want to stress the aur server
	*/
	var ptr *db.MemoryDB
	var err error
	if s.settings.LoadFromFile {
		ptr, err = db.LoadDbFromFile(s.settings.AurFileLocation)
		if err != nil {
			return err
		}
	} else {
		ptr, err = db.LoadDbFromUrl(s.settings.AurFileLocation)
		if err != nil {
			return err
		}
	}
	s.mut.Lock()
	defer s.mut.Unlock()
	s.memDB = ptr
	return nil
}
