package rpc

import (
	"net/url"
	"strings"
	"time"

	db "github.com/moson-mo/goaurrpc/internal/memdb"
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
func (s *server) rpcSearch(values url.Values) RpcResult {
	rr := RpcResult{
		Type: values.Get("type"),
	}

	by := getBy(values)
	foundAll := map[string][]db.PackageInfo{}
	search := getArgumentList(values)
	isSearchType := (rr.Type == "search" || rr.Type == "msearch")

	// maintainer search
	if len(search) == 0 && by == "maintainer" {
		search = append(search, "")
	}

	for _, arg := range search {
		cacheKey := by + "-" + arg

		// check in cache
		if s.settings.EnableSearchCache && len(arg) < 1024 {
			s.mutCache.RLock()
			res, f := s.searchCache[cacheKey]
			s.mutCache.RUnlock()
			if f {
				foundAll[arg] = res.Entry
				rr.Resultcount += res.ResultCount
				if (rr.Resultcount) > s.settings.MaxResults {
					return rr
				}
				continue
			}
		}

		// search for packages
		found := s.search(arg, by)
		lenFound := len(found)
		rr.Resultcount += lenFound

		if lenFound < s.settings.MaxResults {
			foundAll[arg] = found
			s.addToCache(found, cacheKey, lenFound)
		} else {
			s.addToCache(nil, cacheKey, lenFound)
		}
		if (rr.Resultcount) > s.settings.MaxResults {
			return rr
		}

	}

	// Convert to Search- or InfoResult based on version and type
	for _, arg := range search {
		for _, pkg := range foundAll[arg] {
			if isSearchType {
				rr.Results = append(rr.Results, convDbPkgToSearchRecord(&pkg))
			} else {
				rr.Results = append(rr.Results, convDbPkgToInfoRecord(&pkg))
			}
		}
	}

	return rr
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

// add search results to cache. Don't store if exceeding max limit
func (s *server) addToCache(packages []db.PackageInfo, key string, resultCount int) {
	if !s.settings.EnableSearchCache {
		return
	}
	s.mutCache.Lock()
	defer s.mutCache.Unlock()
	s.searchCache[key] = CacheEntry{Entry: packages, TimeAdded: time.Now(), ResultCount: resultCount}
}
