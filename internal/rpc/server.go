package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/moson-mo/goaurrpc/internal/config"
	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"gopkg.in/guregu/null.v4"
)

// API server struct
type server struct {
	memDB      *db.MemoryDB
	mut        sync.RWMutex
	mutLimit   sync.RWMutex
	settings   config.Settings
	stop       chan os.Signal
	RateLimits map[string]RateLimit
}

// New creates a new server and immediately loads package data into memory
func New(settings config.Settings) (*server, error) {
	s := server{
		RateLimits: make(map[string]RateLimit),
		stop:       make(chan os.Signal, 1),
	}
	signal.Notify(s.stop, os.Interrupt)

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

// Listen creates a rest API endpoint and starts listening for requests
func (s *server) Listen() error {
	// start period tasks
	s.startJobs()

	// routes
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.rpcHandler)

	srv := http.Server{
		Addr:    ":" + strconv.Itoa(s.settings.Port),
		Handler: mux,
	}

	// shut down if we get the interrupt signal
	go func() {
		<-s.stop

		fmt.Println("Server is shutting down...")
		srv.Shutdown(context.Background())
	}()

	// Listen for requests
	return srv.ListenAndServe()
}

func (s *server) Stop() {
	s.stop <- os.Interrupt
}

// handles client connections
func (s *server) rpcHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Client connected:", r.RemoteAddr, "->", r.URL)

	// check if got a GET or POST request
	var qstr url.Values
	if r.Method == "GET" {
		qstr = r.URL.Query()
	} else {
		r.ParseForm()
		qstr = r.PostForm
	}
	t := qstr.Get("type")
	v := qstr.Get("v")
	version, _ := strconv.Atoi(v)

	// rate limit check
	if s.isRateLimited(r) {
		writeError(429, "Rate limit reached", version, w)
		return
	}

	// if we don't get any query parameters, return documentation
	if len(qstr) == 0 {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		fmt.Fprintln(w, doc)
		return
	}

	// validate query parameters
	err := validateQueryString(qstr)
	if err != nil {
		writeError(200, err.Error(), version, w)
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

	// don't return data if we exceed max number of results
	result.Resultcount = len(result.Results)
	if result.Resultcount > s.settings.MaxResults {
		result.Error = "Too many package results."
		result.Resultcount = 0
		result.Results = nil
		result.Type = "error"
	}

	// set version number
	result.Version = null.NewInt(int64(version), version != 0)

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

// start go-routines for periodic tasks
func (s *server) startJobs() {
	// starts a go routine that continuesly refreshes the package data
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

	// starts a go routine that removes rate limits if older than 24h
	go func() {
		time.Sleep(time.Duration(s.settings.RateLimitCleanupInterval) * time.Second)
		s.mutLimit.Lock()
		for ip, rl := range s.RateLimits {
			if time.Since(rl.WindowStart).Hours() > 23 {
				delete(s.RateLimits, ip)
				fmt.Println("Removed rate limit for", ip)
			}
		}
		s.mutLimit.Unlock()
	}()
}
