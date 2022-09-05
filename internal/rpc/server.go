package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	memDB       *db.MemoryDB
	mut         sync.RWMutex
	mutLimit    sync.RWMutex
	mutCache    sync.RWMutex
	settings    config.Settings
	stop        chan os.Signal
	rateLimits  map[string]RateLimit
	searchCache map[string]CacheEntry
	lastmod     string
	verbose     bool
	ver         string
	lastRefresh time.Time
}

// New creates a new server and immediately loads package data into memory
func New(settings config.Settings, verbose bool, version string) (*server, error) {
	s := server{
		rateLimits:  make(map[string]RateLimit),
		searchCache: make(map[string]CacheEntry),
		stop:        make(chan os.Signal, 1),
		verbose:     verbose,
		ver:         version,
	}

	// prep logging
	if settings.LogFile != "" {
		f, err := os.OpenFile(settings.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		log.SetOutput(f)
	}
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("| ")

	signal.Notify(s.stop, os.Interrupt)

	s.settings = settings

	// load data
	s.LogVerbose("Loading package data...")
	err := s.reloadData()
	if err != nil {
		return nil, err
	}
	s.LogVerbose("Loaded package data.")
	s.Log("Server started. Ready for client connections...")
	return &s, nil
}

// Listen creates a rest API endpoint and starts listening for requests
func (s *server) Listen() error {
	wg := sync.WaitGroup{}
	shutdown := make(chan struct{})
	// start period tasks
	s.startJobs(shutdown, &wg)

	// routes
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.rpcHandler)
	mux.HandleFunc("/rpc/", s.rpcHandler)
	mux.HandleFunc("/rpc/info", s.rpcInfoHandler)

	srv := http.Server{
		Addr:    ":" + strconv.Itoa(s.settings.Port),
		Handler: mux,
	}

	// shut down if we get the interrupt signal
	go func() {
		<-s.stop
		s.Log("Server is shutting down...")
		close(shutdown)

		wg.Wait()
		srv.Shutdown(context.Background())
	}()

	// Listen for requests
	if s.settings.EnableSSL {
		return srv.ListenAndServeTLS(s.settings.CertFile, s.settings.KeyFile)
	}
	return srv.ListenAndServe()
}

// Stop stops the server
func (s *server) Stop() {
	s.stop <- os.Interrupt
}

// handles client connections
func (s *server) rpcHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRealIP(r, s.settings.TrustedReverseProxies)
	s.LogVerbose("Client connected:", ip, "->", "["+r.Method+"]", r.URL)

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
	c := qstr.Get("callback")

	// rate limit check
	if s.isRateLimited(ip) {
		s.LogVerbose("Client reached rate limit: ", ip)
		writeError(429, "Rate limit reached", version, "", w)
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
		if errors.Is(err, ErrCallBack) {
			writeError(200, err.Error(), version, "", w)
			return
		}
		writeError(200, err.Error(), version, c, w)
		return
	}

	// handle suggest calls
	if t == "suggest" || t == "suggest-pkgbase" {
		s.mut.RLock()
		b, err := json.Marshal(s.rpcSuggest(qstr, (t == "suggest-pkgbase")))
		s.mut.RUnlock()
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
	addCache := false
	s.mut.RLock()
	switch t {
	case "info", "multiinfo":
		result = s.rpcInfo(qstr)
	case "search", "msearch":
		result = s.rpcSearch(qstr)
		addCache = true
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

	// add to cache
	if s.settings.EnableSearchCache && addCache {
		s.mutCache.Lock()
		s.searchCache[qstr.Encode()] = CacheEntry{Result: result, TimeAdded: time.Now()}
		s.mutCache.Unlock()
	}

	// set version number
	result.Version = null.NewInt(int64(version), version != 0)

	// return JSON to client
	writeResult(&result, c, w)
}

// check if rate limit is reached. Create / update the record.
func (s *server) isRateLimited(ip string) bool {
	s.mutLimit.Lock()
	defer s.mutLimit.Unlock()

	// RateLimit of 0 -> Skip check
	if s.settings.RateLimit == 0 {
		return false
	}

	la, ok := s.rateLimits[ip]
	if ok {
		la.Requests++
		s.rateLimits[ip] = la
		if la.Requests > s.settings.RateLimit {
			return true
		}
	} else {
		s.LogVerbose("Rate limit added:", ip)
		s.rateLimits[ip] = RateLimit{
			Requests:    1,
			WindowStart: time.Now(),
		}
	}
	return false
}
