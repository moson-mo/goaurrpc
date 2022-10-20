package memdb

import (
	"gopkg.in/guregu/null.v4"
)

// MemoryDB is a data structe which holds our package data
type MemoryDB struct {
	PackageMap          map[string]PackageInfo
	PackageNames        []string
	SuggestNames        map[byte][]string
	SuggestBases        map[byte][]string
	PackageSlice        []PackageInfo
	PackageDescriptions []PackageDescription
	References          map[string][]*PackageInfo
}

// PackageInfo is a data structure holding data for a single package
type PackageInfo struct {
	ID             int         `json:"ID"`
	Name           string      `json:"Name"`
	PackageBaseID  int         `json:"PackageBaseID"`
	PackageBase    string      `json:"PackageBase"`
	Version        string      `json:"Version"`
	Description    null.String `json:"Description"`
	URL            null.String `json:"URL"`
	NumVotes       int         `json:"NumVotes"`
	Popularity     float64     `json:"Popularity"`
	OutOfDate      null.Int    `json:"OutOfDate"`
	Maintainer     null.String `json:"Maintainer"`
	FirstSubmitted int         `json:"FirstSubmitted"`
	LastModified   int         `json:"LastModified"`
	URLPath        null.String `json:"URLPath"`
	MakeDepends    []string    `json:"MakeDepends"`
	License        []string    `json:"License"`
	Depends        []string    `json:"Depends"`
	Conflicts      []string    `json:"Conflicts"`
	Provides       []string    `json:"Provides"`
	Keywords       []string    `json:"Keywords"`
	OptDepends     []string    `json:"OptDepends"`
	CheckDepends   []string    `json:"CheckDepends"`
	Replaces       []string    `json:"Replaces"`
	Groups         []string    `json:"Groups"`
}

type PackageDescription struct {
	Name        string      `json:"Name"`
	Description null.String `json:"Description"`
}
