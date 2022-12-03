package rpc

import (
	"strings"
)

// searches and returns found packages from our DB
func (s *server) search(arg, by, mode string, v6 bool) ([]string, bool) {
	found := []string{}
	cache := false
	terms := []string{arg}
	if v6 {
		terms = strings.Split(arg, " ")
	}

	compFunc := strings.Contains
	if mode == "starts-with" {
		compFunc = strings.HasPrefix
	}

	// perform search according to the "by" parameter
	switch by {
	case "name":
		cache = true
		for _, name := range s.memDB.PackageNames {
			f := true
			for _, term := range terms {
				if !compFunc(name, term) {
					f = false
					break
				}
			}
			if f {
				found = append(found, name)
			}
		}
	case "maintainer":
		if pkgs, f := s.memDB.References["m-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "submitter":
		if pkgs, f := s.memDB.References["s-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "depends":
		if pkgs, f := s.memDB.References["dep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "makedepends":
		if pkgs, f := s.memDB.References["mdep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "optdepends":
		if pkgs, f := s.memDB.References["odep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "checkdepends":
		if pkgs, f := s.memDB.References["cdep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "provides":
		if pkgs, f := s.memDB.References["pro-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
		if pkg, f := s.memDB.PackageMap[arg]; f {
			found = append(found, pkg.Name)
		}
	case "conflicts":
		if pkgs, f := s.memDB.References["con-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "replaces":
		if pkgs, f := s.memDB.References["rep-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "keywords":
		if pkgs, f := s.memDB.References["key-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "groups":
		if pkgs, f := s.memDB.References["grp-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	case "comaintainers":
		if pkgs, f := s.memDB.References["com-"+arg]; f {
			for _, pkg := range pkgs {
				found = append(found, pkg.Name)
			}
		}
	default:
		cache = true
		for _, pkg := range s.memDB.PackageDescriptions {
			f := true
			for _, term := range terms {
				if !compFunc(pkg.Name, term) && !compFunc(pkg.Description, term) {
					f = false
					break
				}
			}
			if f {
				found = append(found, pkg.Name)
			}
		}
	}

	return found, cache
}
