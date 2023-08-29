package rpc

import (
	"context"
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
	"github.com/moson-mo/goaurrpc/internal/consts"
	"github.com/moson-mo/goaurrpc/internal/doc"
	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"github.com/moson-mo/goaurrpc/internal/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
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
	conf        config.Settings
	stop        chan os.Signal
	rateLimits  map[string]RateLimit
	searchCache map[string]CacheEntry
	verbose     bool
	veryVerbose bool
	ver         string
	lastRefresh time.Time
	router      chi.Router
}

// New creates a new server and immediately loads package data into memory
func New(settings config.Settings, verbose, vverbose bool, version string) (*server, error) {
	s := server{
		rateLimits:  make(map[string]RateLimit),
		searchCache: make(map[string]CacheEntry),
		stop:        make(chan os.Signal, 1),
		verbose:     verbose,
		veryVerbose: vverbose,
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

	s.conf = settings

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

	// set up router
	s.setupRoutes()

	srv := http.Server{
		Addr:    ":" + strconv.Itoa(s.conf.Port),
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
	if s.conf.EnableSSL {
		return srv.ListenAndServeTLS(s.conf.CertFile, s.conf.KeyFile)
	}
	return srv.ListenAndServe()
}

// Stop stops the server
func (s *server) Stop() {
	s.stop <- os.Interrupt
}

// set up our routes
func (s *server) setupRoutes() {
	// routes
	s.router = chi.NewRouter()

	s.router.HandleFunc("/rpc", s.handleRequest)
	s.router.HandleFunc("/rpc/", s.handleRequest)
	s.router.HandleFunc("/rpc.php", s.handleRequest)  // deprecated
	s.router.HandleFunc("/rpc.php/", s.handleRequest) // deprecated
	s.router.HandleFunc("/rpc/stats", s.handleStats)

	// v5 with url paths
	s.router.HandleFunc("/rpc/v{version}/{type}/{arg}", s.handleRequest)
	s.router.HandleFunc("/rpc/v{version}/{type}", s.handleRequest)

	// v6
	s.router.HandleFunc("/api", s.handleRequest)
	s.router.HandleFunc("/api/", s.handleRequest)
	s.router.HandleFunc("/api/v{version}/{type}/{by}/{mode}/{arg}", s.handleRequest)
	s.router.HandleFunc("/api/v{version}/{type}/{by}/{arg}", s.handleRequest)
	s.router.HandleFunc("/api/v{version}/{type}/{arg}", s.handleRequest)
	s.router.HandleFunc("/api/v{version}/{type}", s.handleRequest)

	// metrics
	if s.conf.EnableMetrics {
		metrics.RegisterMetrics()
		s.router.Handle("/metrics", promhttp.Handler())
	}

	// admin api
	if s.conf.EnableAdminApi {
		s.router.Handle("/admin/run-job/{name}", s.adminMiddleware(s.handleAdminJobs))
		s.router.Handle("/admin/settings/{name}", s.adminMiddleware(s.handleAdminSettings))
		s.router.Handle("/admin/settings", s.adminMiddleware(s.handleAdminSettings))
		s.router.Handle("/admin/settings/", s.adminMiddleware(s.handleAdminSettings))
	}

	// swagger
	s.router.HandleFunc("/rpc/swagger", doc.SwaggerRpcHandler)
	s.router.HandleFunc("/rpc/swagger/", doc.SwaggerRpcHandler)
	s.router.HandleFunc("/api/swagger", doc.SwaggerApiHandler)
	s.router.HandleFunc("/api/swagger/", doc.SwaggerApiHandler)
	if s.conf.EnableAdminApi {
		s.router.HandleFunc("/admin/swagger", doc.SwaggerAdminHandler)
		s.router.HandleFunc("/admin/swagger/", doc.SwaggerAdminHandler)
	}
	s.router.HandleFunc("/rpc/openapi.json", doc.SpecRpcHandler)
	s.router.HandleFunc("/api/openapi.json", doc.SpecApiHandler)
	s.router.HandleFunc("/admin/openapi.json", doc.SpecAdminHandler)
	s.router.HandleFunc("/rpc/olddoc.html", doc.SpecOldHandler)
}

// handles client connections
func (s *server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// response time metrics
	timer := prometheus.NewTimer(metrics.HttpDuration.WithLabelValues())
	defer timer.ObserveDuration()

	// get clients IP address
	ip := getRealIP(r, s.conf.TrustedReverseProxies)
	s.LogVeryVerbose("Client connected:", ip, "->", "["+r.Method+"]", r.URL)

	// get API parameters
	params := s.composeParameters(r)

	rtype := params.Get("type")
	by := getBy(params)
	version := params.Get("v")
	verInt, _ := strconv.Atoi(version)
	callback := params.Get("callback")
	mode := params.Get("mode")
	arg := getArg(params)
	args := getArgsList(params)
	isV6 := verInt == 6
	cacheKey := params.Encode()

	// rate limit check
	if s.isRateLimited(ip) {
		// update rate limited metric
		metrics.RateLimited.Inc()

		s.LogVerbose("Client reached rate limit:", ip, "-", "User-Agent:", r.UserAgent())
		writeError(429, "Rate limit reached", verInt, "", w)
		return
	}

	// if we don't get any parameters, return documentation
	if len(params) == 0 {
		http.Redirect(w, r, r.URL.Path+"/swagger", http.StatusFound)
		return
	}

	// validate our parameters
	err := validateParameters(params)
	errCode := 200
	if isV6 {
		errCode = 400
	}
	if err != nil {
		if errors.Is(err, ErrCallBack) {
			writeError(errCode, err.Error(), verInt, "", w)
			return
		}
		writeError(errCode, err.Error(), verInt, callback, w)
		return
	}

	// update requests metric
	metrics.Requests.WithLabelValues(r.Method, rtype, by).Inc()

	// handle suggest calls
	if rtype == "suggest" || rtype == "suggest-pkgbase" || rtype == "opensearch-suggest" || rtype == "opensearch-suggest-pkgbase" {
		s.mut.RLock()
		results := s.getSuggestResult(arg, (rtype == "suggest-pkgbase" || rtype == "opensearch-suggest-pkgbase"))
		s.mut.RUnlock()

		var b []byte
		var err error
		if rtype == "opensearch-suggest" || rtype == "opensearch-suggest-pkgbase" {
			b, err = json.Marshal([]interface{}{params.Get("arg"), results})
		} else {
			b, err = json.Marshal(results)
		}

		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "")
			return
		}

		if rtype == "opensearch-suggest" || rtype == "opensearch-suggest-pkgbase" {
			w.Header().Set("Content-Type", consts.ContentTypeOpenSearchSuggestion)
		} else {
			w.Header().Set("Content-Type", consts.ContentTypeJson)
		}

		w.Write(b)
		return
	}

	// handle info / search calls
	result := RpcResult{}
	cache := false
	s.mut.RLock()
	switch rtype {
	case "info", "multiinfo":
		result = s.getInfoResult(by, args, isV6)
	case "search", "msearch":
		result, cache = s.getSearchResult(rtype, by, mode, arg, cacheKey, isV6)
	}
	s.mut.RUnlock()

	// don't return data if we exceed max number of results
	if result.Resultcount > s.conf.MaxResults {
		result.Error = "Too many package results."
		result.Resultcount = 0
		result.Results = nil
		result.Type = "error"
	}

	// add to search cache
	if cache {
		s.addToCache(result, params.Encode())
	}

	// set version number
	result.Version = null.NewInt(int64(verInt), verInt != 0)

	// return JSON to client
	writeResult(&result, callback, w)
}

// get API parameters from url query/form or path
func (s *server) composeParameters(r *http.Request) url.Values {
	// check if we got a GET or POST request
	var params url.Values
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		r.ParseForm()
		params = r.PostForm
	}

	// set parameters from path variables
	vp := chi.URLParam(r, "version")
	tp := chi.URLParam(r, "type")
	ap := chi.URLParam(r, "arg")
	bp := chi.URLParam(r, "by")
	mp := chi.URLParam(r, "mode")

	if vp != "" {
		params.Set("v", vp)
	}
	if tp != "" {
		params.Set("type", tp)
	}
	if ap != "" {
		params.Set("arg", ap)
	}
	if bp != "" {
		params.Set("by", bp)
	}
	if mp != "" {
		params.Set("mode", mp)
	}

	return params
}

// check if rate limit is reached. Create / update the record.
func (s *server) isRateLimited(ip string) bool {
	s.mutLimit.Lock()
	defer s.mutLimit.Unlock()

	// RateLimit of 0 -> Skip check
	if s.conf.RateLimit == 0 {
		return false
	}

	la, ok := s.rateLimits[ip]
	if ok {
		la.Requests++
		s.rateLimits[ip] = la
		if la.Requests > s.conf.RateLimit {
			return true
		}
	} else {
		s.LogVeryVerbose("Rate limit added:", ip)
		s.rateLimits[ip] = RateLimit{
			Requests:    1,
			WindowStart: time.Now(),
		}
	}
	return false
}

// add search results to cache.
func (s *server) addToCache(result RpcResult, key string) {
	if !s.conf.EnableSearchCache {
		return
	}
	s.mutCache.Lock()
	defer s.mutCache.Unlock()
	s.searchCache[key] = CacheEntry{Result: result, TimeAdded: time.Now()}
}
