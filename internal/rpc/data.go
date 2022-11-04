package rpc

import (
	"time"

	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"gopkg.in/guregu/null.v4"
)

// RpcResult is a data structure that is being sent back
type RpcResult struct {
	Error       string        `json:"error,omitempty"`
	Resultcount int           `json:"resultcount"`
	Results     []interface{} `json:"results"`
	Type        string        `json:"type"`
	Version     null.Int      `json:"version"`
}

// InfoRecord is a data structure for "search" API calls (results)
type InfoRecord struct {
	CheckDepends   []string    `json:"CheckDepends,omitempty"`
	CoMaintainers  []string    `json:"CoMaintainers,omitempty"`
	Conflicts      []string    `json:"Conflicts,omitempty"`
	Depends        []string    `json:"Depends,omitempty"`
	Description    null.String `json:"Description"`
	FirstSubmitted int         `json:"FirstSubmitted"`
	Groups         []string    `json:"Groups,omitempty"`
	ID             int         `json:"ID,omitempty"`
	Keywords       []string    `json:"Keywords"`
	LastModified   int         `json:"LastModified"`
	License        []string    `json:"License"`
	Maintainer     null.String `json:"Maintainer"`
	MakeDepends    []string    `json:"MakeDepends,omitempty"`
	Name           string      `json:"Name"`
	NumVotes       int         `json:"NumVotes"`
	OptDepends     []string    `json:"OptDepends,omitempty"`
	OutOfDate      null.Int    `json:"OutOfDate"`
	PackageBase    string      `json:"PackageBase"`
	PackageBaseID  int         `json:"PackageBaseID,omitempty"`
	Popularity     float64     `json:"Popularity"`
	Provides       []string    `json:"Provides,omitempty"`
	Replaces       []string    `json:"Replaces,omitempty"`
	Submitter      string      `json:"Submitter,omitempty"`
	URL            null.String `json:"URL"`
	URLPath        null.String `json:"URLPath"`
	Version        string      `json:"Version"`
	Arg            string      `json:"arg,omitempty"`
}

// SearchRecord is a data structure for "info" API calls (results)
type SearchRecord struct {
	Description    null.String `json:"Description"`
	FirstSubmitted int         `json:"FirstSubmitted"`
	ID             int         `json:"ID,omitempty"`
	LastModified   int         `json:"LastModified"`
	Maintainer     null.String `json:"Maintainer"`
	Name           string      `json:"Name"`
	NumVotes       int         `json:"NumVotes"`
	OutOfDate      null.Int    `json:"OutOfDate"`
	PackageBase    string      `json:"PackageBase"`
	PackageBaseID  int         `json:"PackageBaseID,omitempty"`
	Popularity     float64     `json:"Popularity"`
	URL            null.String `json:"URL"`
	URLPath        null.String `json:"URLPath"`
	Version        string      `json:"Version"`
	Arg            string      `json:"arg,omitempty"`
}

// RateLimit holds data for the rate limit checking
type RateLimit struct {
	Requests    int
	WindowStart time.Time
}

type CacheEntry struct {
	Entry       []db.PackageInfo
	TimeAdded   time.Time
	ResultCount int
}
