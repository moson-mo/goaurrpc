package rpc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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

// setup our test suite
func (suite *RpcTestSuite) SetupSuite() {
	fmt.Println(">>> Setting up RPC test suite")

	conf := config.Settings{
		Port:            10666,
		AurFileLocation: "../../test_data/test_packages.json.gz",
		MaxResults:      5000,
		RefreshInterval: 600,
		RateLimit:       4000,
		LoadFromFile:    true,
	}

	suite.ExpectedRpcResults = map[string]string{
		"/rpc?v=5&type=info&arg=attest":    `{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`,
		"/rpc?v=5&type=search&arg=attest":  `{"resultcount":6,"results":[{"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"LastModified":1644749267,"Maintainer":"violate","Name":"attest","NumVotes":42,"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"},{"Description":"This is a desciptive text for package attestation","FirstSubmitted":1644749269,"ID":75661,"LastModified":1644749269,"Maintainer":null,"Name":"attestation","NumVotes":39,"OutOfDate":null,"PackageBase":"attestation","PackageBaseID":75661,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","Version":"4.18.64-2"},{"Description":"This is a desciptive text for package attestations","FirstSubmitted":1644749269,"ID":74902,"LastModified":1644749269,"Maintainer":"gilchrists","Name":"attestations","NumVotes":44,"OutOfDate":null,"PackageBase":"attestations","PackageBaseID":74902,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Version":"4.9-9"},{"Description":"This is a desciptive text for package attested","FirstSubmitted":1644749268,"ID":71241,"LastModified":1644749268,"Maintainer":null,"Name":"attested","NumVotes":45,"OutOfDate":null,"PackageBase":"attested","PackageBaseID":71241,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","Version":"0.2.33-2"},{"Description":"This is a desciptive text for package attesting","FirstSubmitted":1644749268,"ID":67658,"LastModified":1644749268,"Maintainer":"amorphousness","Name":"attesting","NumVotes":51,"OutOfDate":null,"PackageBase":"attesting","PackageBaseID":67658,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Version":"1.14.65-10"},{"Description":"This is a desciptive text for package attests","FirstSubmitted":1644749268,"ID":42783,"LastModified":1644749268,"Maintainer":"injudicious","Name":"attests","NumVotes":48,"OutOfDate":null,"PackageBase":"attests","PackageBaseID":42783,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Version":"8.13.74-4"}],"type":"search","version":5}`,
		"/rpc?v=5&type=suggest&arg=attest": `["attest","attestation","attestations","attested","attesting","attests"]`,
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

	suite.ExpectedRateLimit = `{"error":"Rate limit reached","resultcount":0,"results":null,"type":"error","version":0}`

	var err error
	suite.srv, err = New(conf)
	suite.Nil(err, "Could not create rpc server")
}

// run before each test
func (suite *RpcTestSuite) SetupTest() {
	// reset rate limit
	suite.srv.RateLimits = map[string]RateLimit{}
	suite.srv.settings.RateLimit = 4000
}

func (suite *RpcTestSuite) TearDownSuite() {
	fmt.Println(">>> RPC tests completed")
}

func (suite *RpcTestSuite) TestGetArgumentList() {
	for k, v := range suite.ExpectedArgumentsList {
		test := getArgumentList(*k)
		suite.Equal(v, test)
	}
}

func (suite *RpcTestSuite) TestGetArgumentSingle() {
	for k, v := range suite.ExpectedArguments {
		test := getArgument(*k)
		suite.Equal(v, test)
	}
}

func (suite *RpcTestSuite) TestRpcHandlers() {
	for k, v := range suite.ExpectedRpcResults {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", k, nil)
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.rpcHandler).ServeHTTP(rr, req)
		suite.Equal(v, rr.Body.String(), "Input: "+k)
	}
}

func (suite *RpcTestSuite) TestRateLimit() {
	suite.srv.settings.RateLimit = 1

	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.rpcHandler).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit)
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
		}
	}
}

func TestRPCTestSuite(t *testing.T) {
	suite.Run(t, new(RpcTestSuite))
}
