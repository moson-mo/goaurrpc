package memdb

import (
	"encoding/json"
	"io/ioutil"

	"github.com/moson-mo/goaurrpc-poc/internal/aur"
)

// loads package data from local JSON file
func LoadDbFromFile(path string) (*MemoryDB, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytesToMemoryDB(&b)
}

// loads package data from web hosted file (packages-meta-ext-v1.json.gz)
func LoadDbFromUrl(url string) (*MemoryDB, error) {
	b, err := aur.DownloadPackageData(url)
	if err != nil {
		return nil, err
	}
	return bytesToMemoryDB(&b)
}

// constructs MemoryDB struct
func bytesToMemoryDB(b *[]byte) (*MemoryDB, error) {
	db := MemoryDB{
		RateLimits: make(map[string]RateLimit),
	}
	var records []PackageInfo
	err := json.Unmarshal(*b, &records)
	if err != nil {
		return nil, err
	}

	db.Packages = make(map[string]PackageInfo, len(records))

	for _, pkg := range records {
		db.Packages[pkg.Name] = pkg
		db.PackageNames = append(db.PackageNames, pkg.Name)
	}
	return &db, nil
}
