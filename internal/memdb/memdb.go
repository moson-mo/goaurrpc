package memdb

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/moson-mo/goaurrpc/internal/aur"
)

// LoadDbFromFile loads package data from local JSON file
func LoadDbFromFile(path string) (*MemoryDB, error) {
	var b []byte
	if strings.HasSuffix(path, ".gz") {
		gz, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		r, err := gzip.NewReader(gz)
		if err != nil {
			return nil, err
		}
		b, err = ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		b, err = ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}

	return bytesToMemoryDB(b)
}

// LoadDbFromUrl loads package data from web hosted file (packages-meta-ext-v1.json.gz)
func LoadDbFromUrl(url string, lastmod string) (*MemoryDB, string, error) {
	b, lastmod, err := aur.DownloadPackageData(url, lastmod)
	if err != nil {
		return nil, "", err
	}
	memdb, err := bytesToMemoryDB(b)
	if err != nil {
		return nil, "", err
	}
	return memdb, lastmod, nil
}

// constructs MemoryDB struct
func bytesToMemoryDB(b []byte) (*MemoryDB, error) {
	db := MemoryDB{}
	err := json.Unmarshal(b, &db.PackageInfos)
	if err != nil {
		return nil, err
	}

	n := len(db.PackageInfos)

	db.Packages = make(map[string]PackageInfo, n)
	db.PackageNames = make([]string, 0, n)
	db.PackageDescriptions = make([]PackageDescription, 0, n)
	baseNames := make([]string, 0, n)

	for _, pkg := range db.PackageInfos {
		db.Packages[pkg.Name] = pkg
		db.PackageNames = append(db.PackageNames, pkg.Name)
		baseNames = append(baseNames, pkg.PackageBase)
		db.PackageDescriptions = append(db.PackageDescriptions, PackageDescription{Name: pkg.Name, Description: pkg.Description})
	}

	db.PackageBaseNames = distinctStringSlice(baseNames)

	return &db, nil
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
