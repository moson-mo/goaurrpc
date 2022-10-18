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

	"github.com/gorilla/mux"
	"github.com/moson-mo/goaurrpc/internal/config"
	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"github.com/moson-mo/goaurrpc/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	router      *mux.Router
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
	s.Log("Loading package data...")
	start := time.Now()
	err := s.reloadData()
	if err != nil {
		return nil, err
	}
	s.Log("Loaded package data in", time.Since(start).Milliseconds(), "ms.")
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
	s.router = mux.NewRouter()

	s.router.HandleFunc("/rpc", s.rpcHandler)
	s.router.HandleFunc("/rpc/", s.rpcHandler)
	s.router.HandleFunc("/rpc.php", s.rpcHandler)  // should have been removed ?! but aurweb is answering
	s.router.HandleFunc("/rpc.php/", s.rpcHandler) // should have been removed ?! but aurweb is answering
	s.router.HandleFunc("/rpc/stats", s.rpcStatsHandler)

	// v5 with url paths
	s.router.HandleFunc("/rpc/v{version}/{type}/{name}", s.rpcHandler)
	s.router.HandleFunc("/rpc/v{version}/{type}", s.rpcHandler)

	// metrics
	if s.settings.EnableMetrics {
		metrics.RegisterMetrics()
		s.router.Handle("/metrics", promhttp.Handler())
	}

	srv := http.Server{
		Addr:    ":" + strconv.Itoa(s.settings.Port),
		Handler: s.router,
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
	// response time metrics
	timer := prometheus.NewTimer(metrics.HttpDuration.WithLabelValues())
	defer timer.ObserveDuration()

	// get clients IP address
	ip := getRealIP(r, s.settings.TrustedReverseProxies)
	s.LogVerbose("Client connected:", ip, "->", "["+r.Method+"]", r.URL)

	// check if we got a GET or POST request
	var values url.Values
	if r.Method == "GET" {
		values = r.URL.Query()
	} else {
		r.ParseForm()
		values = r.PostForm
	}

	// override query string if we have path variables
	vars := mux.Vars(r)
	if len(vars) > 0 {
		values.Set("v", vars["version"])
		values.Set("type", vars["type"])
		if vars["name"] != "" {
			values.Set("arg", vars["name"])
		}
	}

	t := values.Get("type")
	by := values.Get("by")
	v := values.Get("v")
	version, _ := strconv.Atoi(v)
	c := values.Get("callback")

	// update requests metric
	metrics.Requests.WithLabelValues(r.Method, t, by).Inc()

	// rate limit check
	if s.isRateLimited(ip) {
		// update rate limited metric
		metrics.RateLimited.WithLabelValues().Inc()

		s.LogVerbose("Client reached rate limit: ", ip)
		writeError(429, "Rate limit reached", version, "", w)
		return
	}

	// if we don't get any query parameters, return documentation
	if len(values) == 0 {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		fmt.Fprintln(w, doc)
		return
	}

	// validate query parameters
	err := validateQueryString(values)
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
		b, err := json.Marshal(s.rpcSuggest(values, (t == "suggest-pkgbase")))
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
	s.mut.RLock()
	switch t {
	case "info", "multiinfo":
		result = s.rpcInfo(values)
	case "search", "msearch":
		result = s.rpcSearch(values)
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

	// add to search cache
	s.addToCache(result, values.Encode())

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

// add search results to cache.
func (s *server) addToCache(result RpcResult, key string) {
	if !s.settings.EnableSearchCache {
		return
	}
	s.mutCache.Lock()
	defer s.mutCache.Unlock()
	s.searchCache[key] = CacheEntry{Result: result, TimeAdded: time.Now()}
}
