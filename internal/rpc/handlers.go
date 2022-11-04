package rpc

import (
	"net/url"
	"strings"
	"time"

	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"github.com/moson-mo/goaurrpc/internal/metrics"
)

// construct result for "info" calls
func (s *server) rpcInfo(values url.Values) RpcResult {
	rr := RpcResult{
		Type: "multiinfo",
	}

	packages := getArgumentList(values)
	isV6 := values.Get("v") == "6"

	for _, p := range packages {
		if dbp, ok := s.memDB.PackageMap[p]; ok {
			rr.Results = append(rr.Results, convDbPkgToInfoRecord(dbp, isV6))
			rr.Resultcount++
		}
	}
	return rr
}

// construct result for "search" calls
func (s *server) rpcSearch(values url.Values) RpcResult {
	rr := RpcResult{
		Type: values.Get("type"),
	}

	by := getBy(values)
	foundAll := map[string][]db.PackageInfo{}
	search := getArgumentList(values)
	version := values.Get("v")
	isV6 := version == "6"
	isSearchType := (rr.Type == "search" || rr.Type == "msearch")

	// maintainer search
	if len(search) == 0 && by == "maintainer" {
		search = append(search, "")
	}

	for _, arg := range search {
		cacheKey := version + "-" + by + "-" + arg

		// check in cache
		if s.settings.EnableSearchCache {
			s.mutCache.RLock()
			res, f := s.searchCache[cacheKey]
			s.mutCache.RUnlock()
			if f {
				// update cache hits metric
				metrics.CacheHits.Inc()

				foundAll[arg] = res.Entry
				rr.Resultcount += res.ResultCount
				if (rr.Resultcount) > s.settings.MaxResults {
					return rr
				}
				continue
			}
		}

		// search for packages
		found, cache := s.search(arg, by, isV6)
		lenFound := len(found)
		rr.Resultcount += lenFound

		if lenFound < s.settings.MaxResults {
			foundAll[arg] = found
			if cache {
				s.addToCache(found, cacheKey, lenFound)
			}

		} else if cache {
			s.addToCache(nil, cacheKey, lenFound)
		}
		if (rr.Resultcount) > s.settings.MaxResults {
			return rr
		}

	}

	// Convert to Search- or InfoResult based on version and type
	for _, arg := range search {
		for _, pkg := range foundAll[arg] {
			if !isV6 || isSearchType {
				rr.Results = append(rr.Results, convDbPkgToSearchRecord(&pkg, isV6))
			} else {
				rr.Results = append(rr.Results, convDbPkgToInfoRecord(&pkg, isV6))
			}
		}
	}

	return rr
}

// construct result for "suggest" calls
func (s *server) rpcSuggest(values url.Values, pkgBase bool) []string {
	var searchBase []string
	found := []string{}
	search := getArgument(values)
	if len(search) == 0 {
		searchBase = s.memDB.PackageNames
	} else {
		bucket := search[0]

		if pkgBase {
			searchBase = s.memDB.SuggestBases[bucket]
		} else {
			searchBase = s.memDB.SuggestNames[bucket]
		}
	}

	count := 0

	for _, p := range searchBase {
		if strings.HasPrefix(p, search) {
			found = append(found, p)
			count++
			if count == 20 {
				break
			}
		}
	}
	return found
}

// add search results to cache. Don't store if exceeding max limit
func (s *server) addToCache(packages []db.PackageInfo, key string, resultCount int) {
	if !s.settings.EnableSearchCache {
		return
	}
	s.mutCache.Lock()
	defer s.mutCache.Unlock()
	s.searchCache[key] = CacheEntry{Entry: packages, TimeAdded: time.Now(), ResultCount: resultCount}
}
