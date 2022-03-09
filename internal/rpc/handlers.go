package rpc

import (
	"net/url"
	"sort"
	"strings"

	db "github.com/moson-mo/goaurrpc/internal/memdb"
)

// construct result for "info" calls
func (s *server) rpcInfo(values url.Values) RpcResult {
	rr := RpcResult{
		Type: "multiinfo",
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
		Type: values.Get("type"),
	}

	by := getBy(values)
	found := []db.PackageInfo{}
	search := getArgument(values)

	// perform search according to the "by" parameter
	switch by {
	case "name":
		for _, name := range s.memDB.PackageNames {
			if strings.Contains(name, search) {
				found = append(found, s.memDB.Packages[name])
			}
		}
	case "maintainer":
		for _, pkg := range s.memDB.PackageInfos {
			if pkg.Maintainer.ValueOrZero() == search {
				found = append(found, pkg)
			}
		}
	case "depends":
		for _, pkg := range s.memDB.PackageInfos {
			if inSlice(pkg.Depends, search) {
				found = append(found, pkg)
			}
		}
	case "makedepends":
		for _, pkg := range s.memDB.PackageInfos {
			if inSlice(pkg.MakeDepends, search) {
				found = append(found, pkg)
			}
		}
	case "optdepends":
		for _, pkg := range s.memDB.PackageInfos {
			if sliceContainsBeginsWith(pkg.OptDepends, search) {
				found = append(found, pkg)
			}
		}
	case "checkdepends":
		for _, pkg := range s.memDB.PackageInfos {
			if inSlice(pkg.CheckDepends, search) {
				found = append(found, pkg)
			}
		}
	default:
		for _, pkg := range s.memDB.PackageDescriptions {
			if strings.Contains(pkg.Name, search) || strings.Contains(pkg.Description.String, search) {
				found = append(found, s.memDB.Packages[pkg.Name])
			}
		}
	}
	sort.Slice(found, func(i, j int) bool {
		return found[i].Name < found[j].Name
	})
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
			if count == 20 {
				break
			}
		}

		/*
			we can bail out if the first character is not matching anymore since our list is sorted
			this can be optimized further but it's probably not even worth it
		*/
		if len(search) > 0 && len(p) > 0 {
			if search[0] != p[0] {
				break
			}
		}
	}
	return found
}
