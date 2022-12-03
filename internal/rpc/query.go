package rpc

import (
	"sort"
	"strings"

	"github.com/moson-mo/goaurrpc/internal/metrics"
)

// construct result for "info" calls
func (s *server) getInfoResult(by string, args []string, isV6 bool) RpcResult {
	rr := RpcResult{
		Type: "multiinfo",
	}

	if !isV6 || isV6 && (by == "name" || by == "") {
		for _, pkg := range args {
			if dbp, ok := s.memDB.PackageMap[pkg]; ok {
				if isV6 {
					rr.Results = append(rr.Results, convDbPkgToPackageData(dbp))
				} else {
					rr.Results = append(rr.Results, convDbPkgToInfoRecord(dbp))
				}

				rr.Resultcount++
			}
		}
	} else {
		// we need to return a unique list of packages ordered by name
		uniquePackages := map[string]bool{}
		for _, arg := range args {
			found, _ := s.search(arg, by, "name", isV6)
			for _, pkg := range found {
				uniquePackages[pkg] = true
			}

			// we can bail out if we got more packages than our maximum
			if len(uniquePackages) > s.conf.MaxResults {
				rr.Resultcount = len(uniquePackages)
				return rr
			}
		}

		// get sorted list
		packages := make([]string, 0, len(uniquePackages))
		for pkg := range uniquePackages {
			packages = append(packages, pkg)
		}

		sort.Strings(packages)

		// compose results
		for _, pkg := range packages {
			rr.Results = append(rr.Results, convDbPkgToPackageData(s.memDB.PackageMap[pkg]))
			rr.Resultcount++
		}
	}

	return rr
}

// construct result for "search" calls
func (s *server) getSearchResult(rtype, by, mode, arg, cacheKey string, isV6 bool) (RpcResult, bool) {
	rr := RpcResult{
		Type: rtype,
	}

	// get from search cache
	if s.conf.EnableSearchCache {
		s.mutCache.RLock()
		res, found := s.searchCache[cacheKey]
		s.mutCache.RUnlock()
		if found {
			// update cache hits metric
			metrics.CacheHits.Inc()

			return res.Result, false
		}
	}

	// search
	found, cache := s.search(arg, by, mode, isV6)

	for _, pkg := range found {
		if isV6 {
			rr.Results = append(rr.Results, convDbPkgToPackageData(s.memDB.PackageMap[pkg]))
		} else {
			rr.Results = append(rr.Results, convDbPkgToSearchRecord(s.memDB.PackageMap[pkg]))
		}
		rr.Resultcount++
	}

	return rr, cache
}

// construct result for "suggest" calls
func (s *server) getSuggestResult(arg string, pkgBase bool) []string {
	var searchBase []string
	found := []string{}
	if len(arg) == 0 {
		searchBase = s.memDB.PackageNames
	} else {
		bucket := arg[0]

		if pkgBase {
			searchBase = s.memDB.SuggestBases[bucket]
		} else {
			searchBase = s.memDB.SuggestNames[bucket]
		}
	}

	count := 0

	for _, p := range searchBase {
		if strings.HasPrefix(p, arg) {
			found = append(found, p)
			count++
			if count == 20 {
				break
			}
		}
	}
	return found
}
