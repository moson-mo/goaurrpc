package rpc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/moson-mo/goaurrpc-poc/internal/config"
	db "github.com/moson-mo/goaurrpc-poc/internal/memdb"
)

// API server struct
type server struct {
	memDB    *db.MemoryDB
	mut      sync.RWMutex
	mutLimit sync.RWMutex
	settings *config.Settings
}

// Creates a new server and immediately loads package data into memory
func New(settings *config.Settings) (*server, error) {
	s := server{
		mut: sync.RWMutex{},
	}
	s.settings = settings
	var err error
	fmt.Println("Loading package data...")
	s.mut.Lock()
	defer s.mut.Unlock()

	/*
		use local file for extensive testing -> s.memDB, err = db.LoadDbFromFile("packages.json")
		we don't want to stress the aur server
	*/
	s.memDB, err = db.LoadDbFromUrl(settings.AurFileUrl)
	//s.memDB, err = db.LoadDbFromFile("packages.json")
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
		for ip, rl := range s.memDB.RateLimits {
			if time.Since(rl.WindowStart).Hours() > 23 {
				delete(s.memDB.RateLimits, ip)
				fmt.Println("Removed rate limit for", ip)
			}
		}
	}()

	// Listen for requests on /rpc
	http.HandleFunc("/rpc", s.rpcHandler)
	return http.ListenAndServe(":10666", nil)
}

// handles client connections
func (s *server) rpcHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Client connected:", r.RemoteAddr, "->", r.URL)

	// rate limit check
	if s.isRateLimited(r) {
		writeError(429, "Rate limit reached", w)
		return
	}

	qstr := r.URL.Query()
	t := qstr.Get("type")

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
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	la, ok := s.memDB.RateLimits[ip]
	if ok {
		la.Requests++
		s.memDB.RateLimits[ip] = la
		if la.Requests > s.settings.RateLimit {
			return true
		}
	} else {
		fmt.Println("Rate limit added", ip)
		s.memDB.RateLimits[ip] = db.RateLimit{
			Requests:    1,
			WindowStart: time.Now(),
		}
	}
	return false
}

// generate JSON error and return to client
func writeError(code int, message string, w http.ResponseWriter) {
	w.WriteHeader(code)
	e := RpcResult{
		Error: message,
		Type:  "error",
	}
	b, err := json.Marshal(e)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "This should not happen")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

// generate JSON string from RpcResult and return to client
func writeResult(result *RpcResult, w http.ResponseWriter) {
	b, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "This should not happen")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

// construct result for "info" calls
func (s *server) rpcInfo(values url.Values) RpcResult {
	rr := RpcResult{
		Type:    "multiinfo",
		Version: 5,
	}
	packages := getArgumentList(values)

	for _, p := range packages {
		if dbp, ok := s.memDB.Packages[p]; ok {
			ir := InfoRecord{
				ID:             dbp.ID,
				Name:           dbp.Name,
				PackageBaseID:  dbp.PackageBaseID,
				PackageBase:    dbp.PackageBase,
				Version:        dbp.Version,
				Description:    dbp.Description,
				URL:            dbp.URL,
				NumVotes:       dbp.NumVotes,
				Popularity:     dbp.Popularity,
				OutOfDate:      dbp.OutOfDate,
				Maintainer:     dbp.Maintainer,
				FirstSubmitted: dbp.FirstSubmitted,
				LastModified:   dbp.LastModified,
				URLPath:        dbp.URLPath,
				MakeDepends:    dbp.MakeDepends,
				License:        dbp.License,
				Depends:        dbp.Depends,
				Conflicts:      dbp.Conflicts,
				Provides:       dbp.Provides,
				Keywords:       dbp.Keywords,
				OptDepends:     dbp.OptDepends,
				CheckDepends:   dbp.CheckDepends,
				Replaces:       dbp.Replaces,
				Groups:         dbp.Groups,
			}

			/*
				for some reason Keywords and License should be returned
				as empty JSON arrays rather than being omitted
			*/
			if ir.Keywords == nil {
				ir.Keywords = []string{}
			}
			if ir.License == nil {
				ir.License = []string{}
			}
			rr.Results = append(rr.Results, ir)
		}
	}
	return rr
}

// construct result for "search" calls
func (s *server) rpcSearch(values url.Values) RpcResult {
	rr := RpcResult{
		Type:    "search",
		Version: 5,
	}

	// if not specified use name and description for search
	var by = "name-desc"
	if values.Get("by") != "" {
		by = values.Get("by")
	}

	// if type is msearch we search by maintainer
	if values.Get("type") == "msearch" {
		by = "maintainer"
		rr.Type = "msearch"
	}

	found := []db.PackageInfo{}
	search := getArgument(values)

	// perform search according to the "by" parameter
	switch by {
	case "name":
		for k, pkg := range s.memDB.Packages {
			if k == search {
				found = append(found, pkg)
			}
		}
	case "maintainer":
		for _, pkg := range s.memDB.Packages {
			if pkg.Maintainer.ValueOrZero() == search {
				found = append(found, pkg)
			}
		}
	case "depends":
		for _, pkg := range s.memDB.Packages {
			if sliceContains(pkg.Depends, search) {
				found = append(found, pkg)
			}
		}
	case "makedepends":
		for _, pkg := range s.memDB.Packages {
			if sliceContains(pkg.MakeDepends, search) {
				found = append(found, pkg)
			}
		}
	case "optdepends":
		for _, pkg := range s.memDB.Packages {
			if sliceContains(pkg.OptDepends, search) {
				found = append(found, pkg)
			}
		}
	case "checkdepends":
		for _, pkg := range s.memDB.Packages {
			if sliceContains(pkg.CheckDepends, search) {
				found = append(found, pkg)
			}
		}
	default:
		for _, pkg := range s.memDB.Packages {
			if strings.Contains(pkg.Name, search) || strings.Contains(pkg.Description.ValueOrZero(), search) {
				found = append(found, pkg)
			}
		}
	}
	for _, pkg := range found {
		sr := SearchRecord{
			Description:    pkg.Description,
			FirstSubmitted: pkg.FirstSubmitted,
			ID:             pkg.ID,
			LastModified:   pkg.LastModified,
			Maintainer:     pkg.Maintainer,
			Name:           pkg.Name,
			NumVotes:       pkg.NumVotes,
			OutOfDate:      pkg.OutOfDate,
			PackageBase:    pkg.PackageBase,
			PackageBaseID:  pkg.PackageBaseID,
			Popularity:     pkg.Popularity,
			URL:            pkg.URL,
			URLPath:        pkg.URLPath,
			Version:        pkg.Version,
		}
		rr.Results = append(rr.Results, sr)
	}
	return rr
}

// construct result for "suggest" calls
func (s *server) rpcSuggest(values url.Values, pkgBase bool) []string {
	found := []string{}
	search := getArgument(values)

	/*
		see if we can optimize type "suggest-pkgbase"
		right now we iterate through all packages,
		sort them and return the first 20

		For "suggest" we can bail out after 20 found packages
		since our data is already ordered in our PackageNames slice
	*/
	if pkgBase {
		for _, v := range s.memDB.Packages {
			if strings.Contains(v.PackageBase, search) {
				found = append(found, v.Name)
			}
		}

		sort.Strings(found)
		if len(found) > 20 {
			return found[:20]
		}

	} else {
		count := 0
		for _, p := range s.memDB.PackageNames {
			if strings.Contains(p, search) {
				found = append(found, p)
				count++
				if count == 20 {
					break
				}
			}
		}
	}
	return found
}

// load data from file/url
func (s *server) reloadData() error {
	/*
		use local file for extensive testing -> ptr, err := db.LoadDbFromFile("packages.json")
		we don't want to stress the aur server
	*/
	ptr, err := db.LoadDbFromUrl(s.settings.AurFileUrl)
	//ptr, err := db.LoadDbFromFile("packages.json")
	s.mut.Lock()
	defer s.mut.Unlock()
	s.memDB = ptr
	if err != nil {
		return err
	}
	return nil
}
