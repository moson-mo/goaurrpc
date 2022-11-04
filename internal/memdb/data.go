package memdb

// MemoryDB is a data structe which holds our package data
type MemoryDB struct {
	PackageMap          map[string]*PackageInfo
	PackageNames        []string
	SuggestNames        map[byte][]string
	SuggestBases        map[byte][]string
	PackageSlice        []*PackageInfo
	PackageDescriptions []PackageDescription
	References          map[string][]*PackageInfo
}

// PackageInfo is a data structure holding data for a single package
type PackageInfo struct {
	ID             int
	Name           string
	PackageBaseID  int
	PackageBase    string
	Version        string
	Description    string
	URL            string
	NumVotes       int
	Popularity     float64
	OutOfDate      int
	Maintainer     string
	Submitter      string
	FirstSubmitted int
	LastModified   int
	URLPath        string
	MakeDepends    []string
	License        []string
	Depends        []string
	Conflicts      []string
	Provides       []string
	Keywords       []string
	OptDepends     []string
	CheckDepends   []string
	Replaces       []string
	Groups         []string
	CoMaintainers  []string
	Arg            string
}

type PackageDescription struct {
	Name        string
	Description string
}
