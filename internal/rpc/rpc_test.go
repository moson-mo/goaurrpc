package rpc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/moson-mo/goaurrpc/internal/config"
	"github.com/stretchr/testify/suite"
)

type RpcTestSuite struct {
	suite.Suite
	srv                   *server
	ExpectedRpcResults    map[string]string
	ExpectedArgumentsList map[*url.Values][]string
	ExpectedArguments     map[*url.Values]string
	ExpectedRateLimit     string
}

var conf = config.Settings{
	Port:                  10666,
	AurFileLocation:       "../../test_data/test_packages.json",
	MaxResults:            5000,
	RefreshInterval:       600,
	RateLimit:             4000,
	LoadFromFile:          true,
	TrustedReverseProxies: []string{"127.0.0.1", "::1"},
	EnableSearchCache:     true,
	RateLimitTimeWindow:   86400,
	LogFile:               "log.tst",
	EnableMetrics:         true,
}

// setup our test suite
func (suite *RpcTestSuite) SetupSuite() {
	fmt.Println(">>> Setting up RPC test suite")

	suite.ExpectedRpcResults = map[string]string{
		"/rpc?v=5&type=info&arg=attest":                        `{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`,
		"/rpc.php?v=5&type=info&arg=attest":                    `{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`,
		"/rpc.php/?v=5&type=info&arg=attest":                   `{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`,
		"/rpc/v5/info/attest":                                  `{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`,
		"/rpc?v=5&type=multiinfo&arg=attest":                   `{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`,
		"/rpc?v=5&type=info&arg=doesnotexist":                  `{"resultcount":0,"results":[],"type":"multiinfo","version":5}`,
		"/rpc?v=5&type=info&arg=x":                             `{"resultcount":0,"results":[],"type":"multiinfo","version":5}`,
		"/rpc?v=5&type=info&arg":                               `{"resultcount":0,"results":[],"type":"multiinfo","version":5}`,
		"/rpc?v=5&type=search&arg=attest":                      `{"resultcount":6,"results":[{"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"LastModified":1644749267,"Maintainer":"violate","Name":"attest","NumVotes":42,"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"},{"Description":"This is a desciptive text for package attestation","FirstSubmitted":1644749269,"ID":75661,"LastModified":1644749269,"Maintainer":null,"Name":"attestation","NumVotes":39,"OutOfDate":null,"PackageBase":"attestation","PackageBaseID":75661,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","Version":"4.18.64-2"},{"Description":"This is a desciptive text for package attestations","FirstSubmitted":1644749269,"ID":74902,"LastModified":1644749269,"Maintainer":"gilchrists","Name":"attestations","NumVotes":44,"OutOfDate":null,"PackageBase":"attestations","PackageBaseID":74902,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Version":"4.9-9"},{"Description":"This is a desciptive text for package attested","FirstSubmitted":1644749268,"ID":71241,"LastModified":1644749268,"Maintainer":null,"Name":"attested","NumVotes":45,"OutOfDate":null,"PackageBase":"attested","PackageBaseID":71241,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","Version":"0.2.33-2"},{"Description":"This is a desciptive text for package attesting","FirstSubmitted":1644749268,"ID":67658,"LastModified":1644749268,"Maintainer":"amorphousness","Name":"attesting","NumVotes":51,"OutOfDate":null,"PackageBase":"attesting","PackageBaseID":67658,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Version":"1.14.65-10"},{"Description":"This is a desciptive text for package attests","FirstSubmitted":1644749268,"ID":42783,"LastModified":1644749268,"Maintainer":"injudicious","Name":"attests","NumVotes":48,"OutOfDate":null,"PackageBase":"attests","PackageBaseID":42783,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Version":"8.13.74-4"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&arg=at":                          `{"error":"Too many package results.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=suggest&arg=at":                         `["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee","attlees","attn","attorney","attorneys","attract","attractable","attractant","attractants","attracted","attracting","attraction","attractions","attractively","attractiveness","attractivenesss","attracts","attribute","attributed","attributes","attributions","attributive","attributives","attrition","attritions","attucks","attuckss","attunes","attuning","atty","atv","atwitter","atwoods","atypical"]`,
		"/rpc?v=5&type=suggest&arg=attest":                     `["attest","attestation","attestations","attested","attesting","attests"]`,
		"/rpc?v=5&type=suggest&arg=test":                       `[]`,
		"/rpc?v=5&type=suggest-pkgbase&arg=attest":             `["attest","attestation","attestations","attested","attesting","attests"]`,
		"/rpc?v=5&type=suggest-pkgbase&arg=aTTest":             `["attest","attestation","attestations","attested","attesting","attests"]`,
		"/rpc?v=5&type=suggest-pkgbase&arg=at":                 `["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee","attlees","attn","attorney","attorneys","attract","attractable","attractant","attractants","attracted","attracting","attraction","attractions","attractively","attractiveness","attractivenesss","attracts","attribute","attributed","attributes","attributions","attributive","attributives","attrition","attritions","attucks","attuckss","attunes","attuning","atty","atv","atwitter","atwoods","atypical"]`,
		"/rpc?v=5&type=suggest-pkgbase&arg=test":               `[]`,
		"/rpc?v=5&type=search&by=depends&arg=chrystals":        `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attics","FirstSubmitted":1644749267,"ID":6877,"LastModified":1644749267,"Maintainer":"supergrasses","Name":"attics","NumVotes":42,"OutOfDate":null,"PackageBase":"attics","PackageBaseID":6877,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attics.tar.gz","Version":"8.5-10"}],"type":"search","version":5}`,
		"/rpc/v5/search/chrystals?by=depends":                  `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attics","FirstSubmitted":1644749267,"ID":6877,"LastModified":1644749267,"Maintainer":"supergrasses","Name":"attics","NumVotes":42,"OutOfDate":null,"PackageBase":"attics","PackageBaseID":6877,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attics.tar.gz","Version":"8.5-10"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=depends&arg=x":                `{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=search&by=optdepends&arg=bhopal":        `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attired","FirstSubmitted":1644749267,"ID":28970,"LastModified":1644749267,"Maintainer":"backtalks","Name":"attired","NumVotes":51,"OutOfDate":null,"PackageBase":"attired","PackageBaseID":28970,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attired.tar.gz","Version":"9.2-4"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=optdepends&arg=x":             `{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=search&by=makedepends&arg=constructive": `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attitudinized","FirstSubmitted":1644749269,"ID":73196,"LastModified":1644749269,"Maintainer":"ptolemies","Name":"attitudinized","NumVotes":46,"OutOfDate":null,"PackageBase":"attitudinized","PackageBaseID":73196,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attitudinized.tar.gz","Version":"2.15-6"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=makedepends&arg=x":            `{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=search&by=checkdepends&arg=amphibian":   `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attire","FirstSubmitted":1644749268,"ID":70252,"LastModified":1644749268,"Maintainer":"punish","Name":"attire","NumVotes":51,"OutOfDate":null,"PackageBase":"attire","PackageBaseID":70252,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attire.tar.gz","Version":"0.14.38-2"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=checkdepends&arg=x":           `{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc/?v=5&type=search&by=provides&arg=scrumpy":        `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backspacing","FirstSubmitted":1644749268,"ID":59993,"LastModified":1644749268,"Maintainer":"starlings","Name":"backspacing","NumVotes":50,"OutOfDate":null,"PackageBase":"backspacing","PackageBaseID":59993,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backspacing.tar.gz","Version":"7.4.48-8"}],"type":"search","version":5}`,
		"/rpc/?v=5&type=search&by=provides&arg=awfulness":      `{"resultcount":2,"results":[{"Description":"This is a desciptive text for package backyard","FirstSubmitted":1644749267,"ID":17402,"LastModified":1644749267,"Maintainer":"comers","Name":"backyard","NumVotes":43,"OutOfDate":null,"PackageBase":"backyard","PackageBaseID":17402,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backyard.tar.gz","Version":"4.18-3"},{"Description":"This is a desciptive text for package awfulness","FirstSubmitted":1644749267,"ID":25750,"LastModified":1644749267,"Maintainer":"fatalists","Name":"awfulness","NumVotes":50,"OutOfDate":null,"PackageBase":"awfulness","PackageBaseID":25750,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/awfulness.tar.gz","Version":"3.7-5"}],"type":"search","version":5}`,
		"/rpc/?v=5&type=search&by=conflicts&arg=hope":          `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backyard","FirstSubmitted":1644749267,"ID":17402,"LastModified":1644749267,"Maintainer":"comers","Name":"backyard","NumVotes":43,"OutOfDate":null,"PackageBase":"backyard","PackageBaseID":17402,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backyard.tar.gz","Version":"4.18-3"}],"type":"search","version":5}`,
		"/rpc/?v=5&type=search&by=replaces&arg=spangled":       `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backspace","FirstSubmitted":1644749268,"ID":51569,"LastModified":1644749268,"Maintainer":"tariff","Name":"backspace","NumVotes":52,"OutOfDate":null,"PackageBase":"backspace","PackageBaseID":51569,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backspace.tar.gz","Version":"0.2.60-5"}],"type":"search","version":5}`,
		"/rpc/?v=5&type=search&by=keywords&arg=nonsense":       `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backwoodsmans","FirstSubmitted":1644749267,"ID":6308,"LastModified":1644749267,"Maintainer":"sss","Name":"backwoodsmans","NumVotes":46,"OutOfDate":null,"PackageBase":"backwoodsmans","PackageBaseID":6308,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backwoodsmans.tar.gz","Version":"9.8.61-4"}],"type":"search","version":5}`,
		"/rpc/?v=5&type=search&by=groups&arg=nothing":          `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backwoodsmans","FirstSubmitted":1644749267,"ID":6308,"LastModified":1644749267,"Maintainer":"sss","Name":"backwoodsmans","NumVotes":46,"OutOfDate":null,"PackageBase":"backwoodsmans","PackageBaseID":6308,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backwoodsmans.tar.gz","Version":"9.8.61-4"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=name&arg=attest":              `{"resultcount":6,"results":[{"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"LastModified":1644749267,"Maintainer":"violate","Name":"attest","NumVotes":42,"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"},{"Description":"This is a desciptive text for package attestation","FirstSubmitted":1644749269,"ID":75661,"LastModified":1644749269,"Maintainer":null,"Name":"attestation","NumVotes":39,"OutOfDate":null,"PackageBase":"attestation","PackageBaseID":75661,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","Version":"4.18.64-2"},{"Description":"This is a desciptive text for package attestations","FirstSubmitted":1644749269,"ID":74902,"LastModified":1644749269,"Maintainer":"gilchrists","Name":"attestations","NumVotes":44,"OutOfDate":null,"PackageBase":"attestations","PackageBaseID":74902,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Version":"4.9-9"},{"Description":"This is a desciptive text for package attested","FirstSubmitted":1644749268,"ID":71241,"LastModified":1644749268,"Maintainer":null,"Name":"attested","NumVotes":45,"OutOfDate":null,"PackageBase":"attested","PackageBaseID":71241,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","Version":"0.2.33-2"},{"Description":"This is a desciptive text for package attesting","FirstSubmitted":1644749268,"ID":67658,"LastModified":1644749268,"Maintainer":"amorphousness","Name":"attesting","NumVotes":51,"OutOfDate":null,"PackageBase":"attesting","PackageBaseID":67658,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Version":"1.14.65-10"},{"Description":"This is a desciptive text for package attests","FirstSubmitted":1644749268,"ID":42783,"LastModified":1644749268,"Maintainer":"injudicious","Name":"attests","NumVotes":48,"OutOfDate":null,"PackageBase":"attests","PackageBaseID":42783,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Version":"8.13.74-4"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=name&arg=x":                   `{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=search&by=maintainer&arg=mistrustful":   `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attitudinize","FirstSubmitted":1644749268,"ID":64246,"LastModified":1644749268,"Maintainer":"mistrustful","Name":"attitudinize","NumVotes":42,"OutOfDate":null,"PackageBase":"attitudinize","PackageBaseID":64246,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attitudinize.tar.gz","Version":"7.17.87-9"}],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=maintainer&arg=x":             `{"resultcount":0,"results":[],"type":"search","version":5}`,
		"/rpc?v=5&type=search&by=maintainer&arg=":              `{"error":"Too many package results.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=search&by=maintainer":                   `{"error":"Too many package results.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=msearch&arg=mistrustful":                `{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attitudinize","FirstSubmitted":1644749268,"ID":64246,"LastModified":1644749268,"Maintainer":"mistrustful","Name":"attitudinize","NumVotes":42,"OutOfDate":null,"PackageBase":"attitudinize","PackageBaseID":64246,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attitudinize.tar.gz","Version":"7.17.87-9"}],"type":"msearch","version":5}`,
		"/rpc?v=5&type=nonsense&arg=bla":                       `{"error":"Incorrect request type specified.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=4&type=search&arg=bla":                         `{"error":"Invalid version specified.","resultcount":0,"results":[],"type":"error","version":4}`,
		"/rpc?v=5&type=search&arg=a":                           `{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?type=search&arg=a":                               `{"error":"Please specify an API version.","resultcount":0,"results":[],"type":"error","version":null}`,
		"/rpc?v=5&type=search":                                 `{"error":"No request type/data specified.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&arg=bla":                                     `{"error":"No request type/data specified.","resultcount":0,"results":[],"type":"error","version":5}`,
		"/rpc?v=5&type=info&arg=attest&callback=test":          `/**/test({"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5})`,
		"/rpc?v=5&type=info&arg=attest&callback=test[":         `{"error":"Invalid callback name.","resultcount":0,"results":[],"type":"error","version":5}`,
	}

	suite.ExpectedArgumentsList = map[*url.Values][]string{
		{"arg": {"test1"}}:                                       {"test1"},
		{"arg": {"test1", "test2"}}:                              {"test1"},
		{"arg[]": {"test1"}}:                                     {"test1"},
		{"arg[]": {"test1", "test2"}}:                            {"test1", "test2"},
		{"arg": {"test1"}, "arg[]": {"test2", "test3"}}:          {"test1"},
		{"arg": {"test1", "test2"}, "arg[]": {"test3", "test4"}}: {"test1"},
		{"x": {"test1", "test2"}}:                                nil,
	}

	suite.ExpectedArguments = map[*url.Values]string{
		{"arg": {"test1"}}:                                       "test1",
		{"arg": {"test1", "test2"}}:                              "test1",
		{"arg[]": {"test1"}}:                                     "test1",
		{"arg[]": {"test1", "test2"}}:                            "test1",
		{"arg": {"test1"}, "arg[]": {"test2", "test3"}}:          "test1",
		{"arg": {"test1", "test2"}, "arg[]": {"test3", "test4"}}: "test1",
		{"x": {"test1", "test2"}}:                                "",
	}

	suite.ExpectedRateLimit = `{"error":"Rate limit reached","resultcount":0,"results":[],"type":"error","version":null}`

	var err error
	suite.srv, err = New(conf, false, "")
	suite.Nil(err, "Could not create rpc server")
	suite.srv.verbose = true
}

// run before each test
func (suite *RpcTestSuite) SetupTest() {
	// reset settings
	suite.srv.settings = conf
	suite.srv.reloadData()
}

// cleanup
func (suite *RpcTestSuite) TearDownSuite() {
	log := suite.srv.settings.LogFile
	_, err := os.Stat(log)
	if log != "" && err == nil {
		err = os.Remove(log)
		suite.Nil(err)
	}
	fmt.Println(">>> RPC tests completed")
}

// test function returning a list of arguments
func (suite *RpcTestSuite) TestGetArgumentList() {
	for k, v := range suite.ExpectedArgumentsList {
		test := getArgumentList(*k)
		suite.Equal(v, test)
	}
}

// test function returning a single argument
func (suite *RpcTestSuite) TestGetArgumentSingle() {
	for k, v := range suite.ExpectedArguments {
		test := getArgument(*k)
		suite.Equal(v, test)
	}
}

// test handlers
func (suite *RpcTestSuite) TestRpcHandlers() {
	suite.srv.settings.MaxResults = 10

	for i := 0; i < 2; i++ {
		// get requests
		for k, v := range suite.ExpectedRpcResults {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", k, nil)
			suite.Nil(err, "Could not create GET request")

			suite.srv.router.ServeHTTP(rr, req)
			suite.Equal(v, rr.Body.String(), "Input: "+k)
		}

		// post requests
		for k, v := range suite.ExpectedRpcResults {
			rr := httptest.NewRecorder()
			reader := &strings.Reader{}
			if len(strings.Split(k, "?")) > 1 {
				reader = strings.NewReader(strings.Split(k, "?")[1])
			}
			req, err := http.NewRequest("POST", strings.Split(k, "?")[0], reader)
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			suite.Nil(err, "Could not create POST request")

			suite.srv.router.ServeHTTP(rr, req)
			suite.Equal(v, rr.Body.String(), "Input: "+k)
		}
	}
}

// test rate limit
func (suite *RpcTestSuite) TestRateLimit() {
	suite.srv.settings.RateLimit = 1

	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.rpcHandler).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit)
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
		}
	}

	// with X-Real-IP
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Add("X-Real-IP", "test_rate_limit_real_ip")
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.rpcHandler).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit, "request number: ", i)
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
		}
	}

	// with X-Forwarded-For
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Add("X-Forwarded-For", "test_rate_limit_x_forwarded")
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.rpcHandler).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit, "request number: ", i)
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
		}
	}

	// with X-Forwarded-For (multiple)
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Add("X-Forwarded-For", "test_rate_limit_x_forwarded_multi, bla, blubb")
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.rpcHandler).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit, "request number: ", i)
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
		}
	}
}

// test create server
func (suite *RpcTestSuite) TestListen() {
	suite.srv.settings.RateLimitCleanupInterval = 1
	suite.srv.settings.RefreshInterval = 1
	suite.srv.settings.CacheCleanupInterval = 1
	suite.srv.settings.CacheExpirationTime = 1

	go func() {
		err := suite.srv.Listen()
		suite.Equal(http.ErrServerClosed, err)
	}()

	suite.srv.rateLimits["test"] = RateLimit{WindowStart: time.Now().AddDate(0, 0, -2), Requests: 1}
	suite.srv.searchCache["test"] = CacheEntry{}
	time.Sleep(1000 * time.Millisecond)
	suite.srv.settings.AurFileLocation = "https://github.com/moson-mo/goaurrpc/raw/main/test_data/test_packages.json"
	suite.srv.settings.LoadFromFile = false
	time.Sleep(1000 * time.Millisecond)
	suite.Empty(suite.srv.rateLimits) // check if rate limit got removed
	suite.Empty(suite.srv.searchCache)
	suite.srv.Stop()

	suite.srv.settings.Port = 99999 // use impossible port to trigger an error
	suite.NotNil(suite.srv.Listen())
}

// purposefully crash reload function
func (suite *RpcTestSuite) TestBrokenReload() {
	suite.srv.settings.AurFileLocation = "x"
	suite.NotNil(suite.srv.reloadData(), "Should return an error")
}

// test stats
func (suite *RpcTestSuite) TestStats() {
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/rpc/stats", nil)
	suite.Nil(err, "Could not create GET request")

	suite.srv.router.ServeHTTP(rr, req)
	suite.Equal(200, rr.Result().StatusCode)
}

// run our tests
func TestRPCTestSuite(t *testing.T) {
	suite.Run(t, new(RpcTestSuite))
}
