package rpc

import (
	"strings"

	db "github.com/moson-mo/goaurrpc/internal/memdb"
)

// searches and returns found packages from our DB
func (s *server) search(arg, by string) ([]db.PackageInfo, bool) {
	found := []db.PackageInfo{}
	cache := false

	// perform search according to the "by" parameter
	switch by {
	case "name":
		cache = true
		for _, name := range s.memDB.PackageNames {
			if strings.Contains(name, arg) {
				found = append(found, s.memDB.PackageMap[name])
			}
		}
	case "maintainer":
		for _, pkg := range s.memDB.PackageSlice {
			if pkg.Maintainer.ValueOrZero() == arg {
				found = append(found, pkg)
			}
		}
	case "depends":
		if pkgs, f := s.memDB.References["dep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	case "makedepends":
		if pkgs, f := s.memDB.References["mdep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	case "optdepends":
		if pkgs, f := s.memDB.References["odep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	case "checkdepends":
		if pkgs, f := s.memDB.References["cdep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	case "provides":
		if pkgs, f := s.memDB.References["pro-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
		if pkg, f := s.memDB.PackageMap[arg]; f {
			found = append(found, pkg)
		}
	case "conflicts":
		if pkgs, f := s.memDB.References["con-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	case "replaces":
		if pkgs, f := s.memDB.References["rep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	case "keywords":
		if pkgs, f := s.memDB.References["key-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	case "groups":
		if pkgs, f := s.memDB.References["grp-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, *pkg)
			}
		}
	default:
		cache = true
		for _, pkg := range s.memDB.PackageDescriptions {
			if strings.Contains(pkg.Name, arg) || strings.Contains(pkg.Description.String, arg) {
				found = append(found, s.memDB.PackageMap[pkg.Name])
			}
		}
	}

	return found, cache
}
