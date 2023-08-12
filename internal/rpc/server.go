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

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/moson-mo/goaurrpc/internal/config"
	"github.com/moson-mo/goaurrpc/internal/doc"
	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"github.com/moson-mo/goaurrpc/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/guregu/null.v4"

	"net/http/pprof"
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

	// set up router
	s.setupRoutes()

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

// set up our routes
func (s *server) setupRoutes() {
	// routes
	s.router = chi.NewRouter()

	s.router.HandleFunc("/rpc", s.rpcHandler)
	s.router.HandleFunc("/rpc/", s.rpcHandler)
	s.router.HandleFunc("/rpc.php", s.rpcHandler)  // deprecated
	s.router.HandleFunc("/rpc.php/", s.rpcHandler) // deprecated
	s.router.HandleFunc("/rpc/stats", s.rpcStatsHandler)

	// v5 with url paths
	s.router.HandleFunc("/rpc/v{version}/{type}/{name}", s.rpcHandler)
	s.router.HandleFunc("/rpc/v{version}/{type}", s.rpcHandler)

	// metrics
	if s.settings.EnableMetrics {
		metrics.RegisterMetrics()
		s.router.Handle("/metrics", promhttp.Handler())
	}

	// admin api
	if s.settings.EnableAdminApi {
		s.router.Handle("/admin/run-job/{name}", s.rpcAdminMiddleware(s.rpcAdminJobsHandler))
		s.router.Handle("/admin/settings/{name}", s.rpcAdminMiddleware(s.rpcAdminSettingsHandler))
		s.router.Handle("/admin/settings", s.rpcAdminMiddleware(s.rpcAdminSettingsHandler))
		s.router.Handle("/admin/settings/", s.rpcAdminMiddleware(s.rpcAdminSettingsHandler))
	}

	// swagger
	s.router.HandleFunc("/rpc/swagger", doc.SwaggerRpcHandler)
	s.router.HandleFunc("/rpc/swagger/", doc.SwaggerRpcHandler)
	if s.settings.EnableAdminApi {
		s.router.HandleFunc("/admin/swagger", doc.SwaggerAdminHandler)
		s.router.HandleFunc("/admin/swagger/", doc.SwaggerAdminHandler)
	}
	s.router.HandleFunc("/rpc/openapi.json", doc.SpecRpcHandler)
	s.router.HandleFunc("/admin/openapi.json", doc.SpecAdminHandler)
	s.router.HandleFunc("/rpc/olddoc.html", doc.SpecOldHandler)

	// pprof
	s.router.HandleFunc("/debug/pprof/", pprof.Index)
	s.router.HandleFunc("/debug/pprof/heap", pprof.Index)
	s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// handles client connections
func (s *server) rpcHandler(w http.ResponseWriter, r *http.Request) {
	// response time metrics
	timer := prometheus.NewTimer(metrics.HttpDuration.WithLabelValues())
	defer timer.ObserveDuration()

	// get clients IP address
	ip := getRealIP(r, s.settings.TrustedReverseProxies)
	s.LogVeryVerbose("Client connected:", ip, "->", "["+r.Method+"]", r.URL)

	// check if we got a GET or POST request
	var values url.Values
	if r.Method == "GET" {
		values = r.URL.Query()
	} else {
		r.ParseForm()
		values = r.PostForm
	}

	// override query string if we have path variables
	vp := chi.URLParam(r, "version")
	tp := chi.URLParam(r, "type")
	np := chi.URLParam(r, "name")

	if vp != "" {
		values.Set("v", vp)
	}
	if tp != "" {
		values.Set("type", tp)
	}
	if np != "" {
		values.Set("arg", np)
	}

	// get API parameters
	t := values.Get("type")
	by := values.Get("by")
	if by == "" {
		by = "name-desc"
	}
	v := values.Get("v")
	version, _ := strconv.Atoi(v)
	c := values.Get("callback")

	// rate limit check
	if s.isRateLimited(ip) {
		// update rate limited metric
		metrics.RateLimited.Inc()

		s.LogVerbose("Client reached rate limit:", ip, "-", "User-Agent:", r.UserAgent())
		writeError(429, "Rate limit reached", version, "", w)
		return
	}

	// if we don't get any query parameters, return documentation
	if len(values) == 0 {
		http.Redirect(w, r, "/rpc/swagger", http.StatusFound)
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

	// update requests metric
	metrics.Requests.WithLabelValues(r.Method, t, by).Inc()

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
	result := RpcResult{}
	cache := false
	s.mut.RLock()
	switch t {
	case "info", "multiinfo":
		result = s.rpcInfo(values)
	case "search", "msearch":
		result, cache = s.rpcSearch(values)
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
	if cache {
		s.addToCache(result, values.Encode())
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
	if !s.settings.EnableSearchCache {
		return
	}
	s.mutCache.Lock()
	defer s.mutCache.Unlock()
	s.searchCache[key] = CacheEntry{Result: result, TimeAdded: time.Now()}
}
