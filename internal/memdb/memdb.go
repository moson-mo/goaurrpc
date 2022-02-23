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
func LoadDbFromUrl(url string) (*MemoryDB, error) {
	b, err := aur.DownloadPackageData(url)
	if err != nil {
		return nil, err
	}
	return bytesToMemoryDB(b)
}

// constructs MemoryDB struct
func bytesToMemoryDB(b []byte) (*MemoryDB, error) {
	db := MemoryDB{}
	var records []PackageInfo
	err := json.Unmarshal(b, &records)
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
