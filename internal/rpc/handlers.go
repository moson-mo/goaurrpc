package rpc

import (
	"net/url"
	"strings"

	"github.com/moson-mo/goaurrpc/internal/metrics"
)

// construct result for "info" calls
func (s *server) rpcInfo(values url.Values) RpcResult {
	rr := RpcResult{
		Type: "multiinfo",
	}

	packages := getArgumentList(values)

	for _, p := range packages {
		if dbp, ok := s.memDB.PackageMap[p]; ok {
			rr.Results = append(rr.Results, convDbPkgToInfoRecord(&dbp))
			rr.Resultcount++
		}
	}
	return rr
}

// construct result for "search" calls
func (s *server) rpcSearch(values url.Values) (RpcResult, bool) {
	rr := RpcResult{
		Type: values.Get("type"),
	}

	// get from search cache
	key := values.Encode()
	if s.settings.EnableSearchCache {
		s.mutCache.RLock()
		res, found := s.searchCache[key]
		s.mutCache.RUnlock()
		if found {
			// update cache hits metric
			metrics.CacheHits.Inc()

			return res.Result, false
		}
	}

	// search
	by := getBy(values)
	arg := getArgument(values)
	found, cache := s.search(arg, by)

	for _, pkg := range found {
		rr.Results = append(rr.Results, convDbPkgToSearchRecord(&pkg))
	}

	return rr, cache
}

// construct result for "suggest" calls
func (s *server) rpcSuggest(values url.Values, pkgBase bool) []string {
	found := []string{}
	search := getArgument(values)

	count := 0

	var searchBase []string
	if pkgBase {
		searchBase = s.memDB.PackageBaseNames
	} else {
		searchBase = s.memDB.PackageNames
	}

	for _, p := range searchBase {
		if strings.HasPrefix(p, search) {
			found = append(found, p)
			count++
			if count == 100 {
				break
			}
		}

		/*
			we can bail out if the first character is not matching anymore since our list is sorted
			this can be optimized further but it's probably not even worth it
		*/
		if len(search) > 0 && len(p) > 0 {
			if search[0] < p[0] {
				break
			}
		}
	}
	return found
}
