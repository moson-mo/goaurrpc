package memdb

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/moson-mo/goaurrpc/internal/aur"
)

// LoadDbFromFile loads package data from local JSON file
func LoadDbFromFile(path string, lastmod time.Time) (*MemoryDB, time.Time, error) {
	var b []byte

	file, err := os.Stat(path)
	if err != nil {
		return nil, lastmod, err
	}

	if file.ModTime() == lastmod {
		return nil, lastmod, errors.New("not modified")
	}

	if strings.HasSuffix(path, ".gz") {
		gz, err := os.Open(path)
		if err != nil {
			return nil, lastmod, err
		}
		defer gz.Close()
		r, err := gzip.NewReader(gz)
		if err != nil {
			return nil, lastmod, err
		}
		b, err = io.ReadAll(r)
		if err != nil {
			return nil, lastmod, err
		}
	} else {
		var err error
		b, err = os.ReadFile(path)
		if err != nil {
			return nil, lastmod, err
		}
	}

	memdb, err := bytesToMemoryDB(b)
	if err != nil {
		return nil, lastmod, err
	}

	return memdb, file.ModTime(), nil
}

// LoadDbFromUrl loads package data from web hosted file (packages-meta-ext-v1.json.gz)
func LoadDbFromUrl(url string, lastmod time.Time) (*MemoryDB, time.Time, error) {
	b, newmod, err := aur.DownloadPackageData(url, lastmod)
	if err != nil {
		return nil, lastmod, err
	}
	memdb, err := bytesToMemoryDB(b)
	if err != nil {
		return nil, lastmod, err
	}
	return memdb, newmod, nil
}

// constructs MemoryDB struct
func bytesToMemoryDB(b []byte) (*MemoryDB, error) {
	db := MemoryDB{}
	err := json.Unmarshal(b, &db.PackageSlice)
	if err != nil {
		return nil, err
	}

	db.fillHelperVars()

	return &db, nil
}

// fills some slices we need for search lookups.
func (db *MemoryDB) fillHelperVars() {
	n := len(db.PackageSlice)

	db.PackageMap = make(map[string]*PackageInfo, n)
	db.PackageNames = make([]string, 0, n)
	db.PackageDescriptions = make([]PackageDescription, 0, n)
	db.References = map[string][]*PackageInfo{}
	baseNames := []string{}
	db.SuggestNames = map[byte][]string{}
	db.SuggestBases = map[byte][]string{}

	sort.Slice(db.PackageSlice, func(i, j int) bool {
		return db.PackageSlice[i].Name < db.PackageSlice[j].Name
	})

	for i, pkg := range db.PackageSlice {
		db.PackageMap[pkg.Name] = pkg
		db.PackageNames = append(db.PackageNames, pkg.Name)
		baseNames = append(baseNames, pkg.PackageBase)
		db.PackageDescriptions = append(db.PackageDescriptions, PackageDescription{Name: pkg.Name, Description: strings.ToLower(pkg.Description)})
		if len(pkg.Name) > 0 {
			db.SuggestNames[pkg.Name[0]] = append(db.SuggestNames[pkg.Name[0]], pkg.Name)
		}

		// depends
		for _, ref := range pkg.Depends {
			sref := "dep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// makedepends
		for _, ref := range pkg.MakeDepends {
			sref := "mdep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// optdepends
		for _, ref := range pkg.OptDepends {
			sref := "odep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// checkdepends
		for _, ref := range pkg.CheckDepends {
			sref := "cdep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// provides
		for _, ref := range pkg.Provides {
			sref := "pro-" + stripRef(ref)
			if ref != pkg.Name {
				db.References[sref] = append(db.References[sref], db.PackageSlice[i])
			}
		}
		// conflicts
		for _, ref := range pkg.Conflicts {
			sref := "con-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// replaces
		for _, ref := range pkg.Replaces {
			sref := "rep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// groups
		for _, ref := range pkg.Groups {
			sref := "grp-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// keywords
		for _, ref := range pkg.Keywords {
			sref := "key-" + strings.ToLower(stripRef(ref))
			db.References[sref] = append(db.References[sref], db.PackageSlice[i])
		}
		// maintainer
		maintainer := "m-" + strings.ToLower(pkg.Maintainer)
		db.References[maintainer] = append(db.References[maintainer], db.PackageSlice[i])
		// submitter
		submitter := "s-" + strings.ToLower(pkg.Submitter)
		db.References[submitter] = append(db.References[submitter], db.PackageSlice[i])
		// comaintainers
		for _, com := range pkg.CoMaintainers {
			com = "com-" + strings.ToLower(com)
			db.References[com] = append(db.References[com], db.PackageSlice[i])
		}
	}

	for _, base := range distinctStringSlice(baseNames) {
		if len(base) > 0 {
			db.SuggestBases[base[0]] = append(db.SuggestBases[base[0]], base)
		}
	}
}

func stripRef(ref string) string {
	ret := strings.Split(ref, ">")[0]
	ret = strings.Split(ret, "<")[0]
	ret = strings.Split(ret, ":")[0]
	ret = strings.Split(ret, "=")[0]
	return ret
}

func distinctStringSlice(s []string) []string {
	keys := make(map[string]bool)
	dist := []string{}

	for _, entry := range s {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			dist = append(dist, entry)
		}
	}
	return dist
}
