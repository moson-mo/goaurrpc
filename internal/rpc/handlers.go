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
