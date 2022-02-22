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
