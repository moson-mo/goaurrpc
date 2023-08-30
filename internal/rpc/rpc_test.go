package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/moson-mo/goaurrpc/internal/config"
	"github.com/moson-mo/goaurrpc/internal/consts"
	"github.com/stretchr/testify/suite"
)

type responseResult struct {
	expected    string
	contentType string
}

type RpcTestSuite struct {
	suite.Suite
	srv                      *server
	httpSrv                  *http.Server
	ExpectedRpcResults       map[string]responseResult
	ExpectedArgumentsList    map[*url.Values][]string
	ExpectedArguments        map[*url.Values]string
	ExpectedRateLimit        string
	ExpectedAdminResultsPOST map[string]string
	ExpectedAdminResultsGET  map[string]responseResult
}

var conf = config.Settings{
	Port:                     10667,
	AurFileLocation:          "../../test_data/test_packages.json",
	MaxResults:               5000,
	RefreshInterval:          600,
	RateLimit:                4000,
	RateLimitCleanupInterval: 600,
	LoadFromFile:             true,
	TrustedReverseProxies:    []string{"127.0.0.1", "::1"},
	EnableSearchCache:        true,
	CacheCleanupInterval:     60,
	CacheExpirationTime:      300,
	RateLimitTimeWindow:      86400,
	LogFile:                  "/tmp/log.tst",
	EnableMetrics:            true,
	EnableAdminApi:           true,
	AdminAPIKey:              "test",
}
var confBroken = config.Settings{
	Port:                     99999,
	AurFileLocation:          "../../test_data/test_packages.json",
	MaxResults:               5000,
	RefreshInterval:          600,
	RateLimit:                4000,
	RateLimitCleanupInterval: 600,
	LoadFromFile:             true,
	TrustedReverseProxies:    []string{"127.0.0.1", "::1"},
	EnableSearchCache:        true,
	CacheCleanupInterval:     60,
	CacheExpirationTime:      300,
	RateLimitTimeWindow:      86400,
	LogFile:                  "/tmp/log.tst",
	EnableMetrics:            true,
	EnableAdminApi:           true,
	AdminAPIKey:              "test",
}

// setup our test suite
func (suite *RpcTestSuite) SetupSuite() {
	fmt.Println(">>> Setting up RPC test suite")

	suite.ExpectedRpcResults = map[string]responseResult{
		"/rpc?v=5&type=info&arg=attest":                        {`{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=info&arg=australia":                     {`{"resultcount":1,"results":[{"CheckDepends":["signally"],"Conflicts":["reline"],"Depends":["overs","ella"],"Description":null,"FirstSubmitted":1644749268,"ID":60415,"Keywords":[],"LastModified":1644749268,"License":[],"Maintainer":null,"MakeDepends":["brendans"],"Name":"australia","NumVotes":40,"OptDepends":["anesthetics: for chippendale","cytologists: for arianism"],"OutOfDate":null,"PackageBase":"australia","PackageBaseID":60415,"Popularity":0,"Provides":["helpfulness","haiphong","tethered"],"Replaces":["predators","heavyweights"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/australia.tar.gz","Version":"3.11.48-5"}],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc.php?v=5&type=info&arg=attest":                    {`{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc.php/?v=5&type=info&arg=attest":                   {`{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc/v5/info/attest":                                  {`{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=multiinfo&arg=attest":                   {`{"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=info&arg=doesnotexist":                  {`{"resultcount":0,"results":[],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=info&arg=x":                             {`{"resultcount":0,"results":[],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=info&arg":                               {`{"resultcount":0,"results":[],"type":"multiinfo","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&arg=attest":                      {`{"resultcount":6,"results":[{"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"LastModified":1644749267,"Maintainer":"violate","Name":"attest","NumVotes":42,"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"},{"Description":"This is a desciptive text for package attestation","FirstSubmitted":1644749269,"ID":75661,"LastModified":1644749269,"Maintainer":null,"Name":"attestation","NumVotes":39,"OutOfDate":null,"PackageBase":"attestation","PackageBaseID":75661,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","Version":"4.18.64-2"},{"Description":"This is a desciptive text for package attestations","FirstSubmitted":1644749269,"ID":74902,"LastModified":1644749269,"Maintainer":"gilchrists","Name":"attestations","NumVotes":44,"OutOfDate":null,"PackageBase":"attestations","PackageBaseID":74902,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Version":"4.9-9"},{"Description":"This is a desciptive text for package attested","FirstSubmitted":1644749268,"ID":71241,"LastModified":1644749268,"Maintainer":null,"Name":"attested","NumVotes":45,"OutOfDate":null,"PackageBase":"attested","PackageBaseID":71241,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","Version":"0.2.33-2"},{"Description":"This is a desciptive text for package attesting","FirstSubmitted":1644749268,"ID":67658,"LastModified":1644749268,"Maintainer":"amorphousness","Name":"attesting","NumVotes":51,"OutOfDate":null,"PackageBase":"attesting","PackageBaseID":67658,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Version":"1.14.65-10"},{"Description":"This is a desciptive text for package attests","FirstSubmitted":1644749268,"ID":42783,"LastModified":1644749268,"Maintainer":"injudicious","Name":"attests","NumVotes":48,"OutOfDate":null,"PackageBase":"attests","PackageBaseID":42783,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Version":"8.13.74-4"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&arg=ATTEST":                      {`{"resultcount":6,"results":[{"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"LastModified":1644749267,"Maintainer":"violate","Name":"attest","NumVotes":42,"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"},{"Description":"This is a desciptive text for package attestation","FirstSubmitted":1644749269,"ID":75661,"LastModified":1644749269,"Maintainer":null,"Name":"attestation","NumVotes":39,"OutOfDate":null,"PackageBase":"attestation","PackageBaseID":75661,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","Version":"4.18.64-2"},{"Description":"This is a desciptive text for package attestations","FirstSubmitted":1644749269,"ID":74902,"LastModified":1644749269,"Maintainer":"gilchrists","Name":"attestations","NumVotes":44,"OutOfDate":null,"PackageBase":"attestations","PackageBaseID":74902,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Version":"4.9-9"},{"Description":"This is a desciptive text for package attested","FirstSubmitted":1644749268,"ID":71241,"LastModified":1644749268,"Maintainer":null,"Name":"attested","NumVotes":45,"OutOfDate":null,"PackageBase":"attested","PackageBaseID":71241,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","Version":"0.2.33-2"},{"Description":"This is a desciptive text for package attesting","FirstSubmitted":1644749268,"ID":67658,"LastModified":1644749268,"Maintainer":"amorphousness","Name":"attesting","NumVotes":51,"OutOfDate":null,"PackageBase":"attesting","PackageBaseID":67658,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Version":"1.14.65-10"},{"Description":"This is a desciptive text for package attests","FirstSubmitted":1644749268,"ID":42783,"LastModified":1644749268,"Maintainer":"injudicious","Name":"attests","NumVotes":48,"OutOfDate":null,"PackageBase":"attests","PackageBaseID":42783,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Version":"8.13.74-4"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&arg=at":                          {`{"error":"Too many package results.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc/v5/search/blablabla":                             {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backbite BLABLABLA","FirstSubmitted":1644749267,"ID":11182,"LastModified":1644749267,"Maintainer":"guayaquils","Name":"backbite","NumVotes":44,"OutOfDate":null,"PackageBase":"backbite","PackageBaseID":11182,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backbite.tar.gz","Version":"9.7.90-4"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest&arg=at":                         {`["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest&arg=":                           {`["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest&arg=attest":                     {`["attest","attestation","attestations","attested","attesting","attests"]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest&arg=test":                       {`[]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest-pkgbase&arg=attest":             {`["attest","attestation","attestations","attested","attesting","attests"]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest-pkgbase&arg=aTTest":             {`["attest","attestation","attestations","attested","attesting","attests"]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest-pkgbase&arg=at":                 {`["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest-pkgbase&arg=":                   {`["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]`, consts.ContentTypeJson},
		"/rpc?v=5&type=suggest-pkgbase&arg=test":               {`[]`, consts.ContentTypeJson},
		"/rpc?v=5&type=opensearch-suggest&arg=at":              {`["at",["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest&arg=":                {`["",["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest&arg=attest":          {`["attest",["attest","attestation","attestations","attested","attesting","attests"]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest&arg=test":            {`["test",[]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest-pkgbase&arg=attest":  {`["attest",["attest","attestation","attestations","attested","attesting","attests"]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest-pkgbase&arg=aTTest":  {`["aTTest",["attest","attestation","attestations","attested","attesting","attests"]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest-pkgbase&arg=at":      {`["at",["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest-pkgbase&arg=":        {`["",["attest","attestation","attestations","attested","attesting","attests","attic","atticas","attics","attila","attire","attired","attires","attitude","attitudes","attitudinal","attitudinize","attitudinized","attitudinizing","attlee"]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=opensearch-suggest-pkgbase&arg=test":    {`["test",[]]`, consts.ContentTypeOSS},
		"/rpc?v=5&type=search&by=depends&arg=chrystals":        {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attics","FirstSubmitted":1644749267,"ID":6877,"LastModified":1644749267,"Maintainer":"supergrasses","Name":"attics","NumVotes":42,"OutOfDate":null,"PackageBase":"attics","PackageBaseID":6877,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attics.tar.gz","Version":"8.5-10"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc/v5/search/chrystals?by=depends":                  {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attics","FirstSubmitted":1644749267,"ID":6877,"LastModified":1644749267,"Maintainer":"supergrasses","Name":"attics","NumVotes":42,"OutOfDate":null,"PackageBase":"attics","PackageBaseID":6877,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attics.tar.gz","Version":"8.5-10"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=depends&arg=x":                {`{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=optdepends&arg=bhopal":        {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attired","FirstSubmitted":1644749267,"ID":28970,"LastModified":1644749267,"Maintainer":"backtalks","Name":"attired","NumVotes":51,"OutOfDate":null,"PackageBase":"attired","PackageBaseID":28970,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attired.tar.gz","Version":"9.2-4"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=optdepends&arg=x":             {`{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=makedepends&arg=constructive": {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attitudinized","FirstSubmitted":1644749269,"ID":73196,"LastModified":1644749269,"Maintainer":"ptolemies","Name":"attitudinized","NumVotes":46,"OutOfDate":null,"PackageBase":"attitudinized","PackageBaseID":73196,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attitudinized.tar.gz","Version":"2.15-6"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=makedepends&arg=x":            {`{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=checkdepends&arg=amphibian":   {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attire","FirstSubmitted":1644749268,"ID":70252,"LastModified":1644749268,"Maintainer":"punish","Name":"attire","NumVotes":51,"OutOfDate":null,"PackageBase":"attire","PackageBaseID":70252,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attire.tar.gz","Version":"0.14.38-2"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=checkdepends&arg=x":           {`{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc/?v=5&type=search&by=provides&arg=scrumpy":        {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backspacing","FirstSubmitted":1644749268,"ID":59993,"LastModified":1644749268,"Maintainer":"starlings","Name":"backspacing","NumVotes":50,"OutOfDate":null,"PackageBase":"backspacing","PackageBaseID":59993,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backspacing.tar.gz","Version":"7.4.48-8"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc/?v=5&type=search&by=provides&arg=awfulness":      {`{"resultcount":2,"results":[{"Description":"This is a desciptive text for package backyard","FirstSubmitted":1644749267,"ID":17402,"LastModified":1644749267,"Maintainer":"comers","Name":"backyard","NumVotes":43,"OutOfDate":null,"PackageBase":"backyard","PackageBaseID":17402,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backyard.tar.gz","Version":"4.18-3"},{"Description":"This is a desciptive text for package awfulness","FirstSubmitted":1644749267,"ID":25750,"LastModified":1644749267,"Maintainer":"fatalists","Name":"awfulness","NumVotes":50,"OutOfDate":null,"PackageBase":"awfulness","PackageBaseID":25750,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/awfulness.tar.gz","Version":"3.7-5"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc/?v=5&type=search&by=conflicts&arg=hope":          {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backyard","FirstSubmitted":1644749267,"ID":17402,"LastModified":1644749267,"Maintainer":"comers","Name":"backyard","NumVotes":43,"OutOfDate":null,"PackageBase":"backyard","PackageBaseID":17402,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backyard.tar.gz","Version":"4.18-3"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc/?v=5&type=search&by=replaces&arg=spangled":       {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backspace","FirstSubmitted":1644749268,"ID":51569,"LastModified":1644749268,"Maintainer":"tariff","Name":"backspace","NumVotes":52,"OutOfDate":null,"PackageBase":"backspace","PackageBaseID":51569,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backspace.tar.gz","Version":"0.2.60-5"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc/?v=5&type=search&by=keywords&arg=nonsense":       {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backwoodsmans","FirstSubmitted":1644749267,"ID":6308,"LastModified":1644749267,"Maintainer":"sss","Name":"backwoodsmans","NumVotes":46,"OutOfDate":null,"PackageBase":"backwoodsmans","PackageBaseID":6308,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backwoodsmans.tar.gz","Version":"9.8.61-4"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc/?v=5&type=search&by=groups&arg=nothing":          {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backwoodsmans","FirstSubmitted":1644749267,"ID":6308,"LastModified":1644749267,"Maintainer":"sss","Name":"backwoodsmans","NumVotes":46,"OutOfDate":null,"PackageBase":"backwoodsmans","PackageBaseID":6308,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backwoodsmans.tar.gz","Version":"9.8.61-4"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc/?v=5&type=search&by=nonsense&arg=nothing":        {`{"error":"Incorrect by field specified.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=name&arg=attest":              {`{"resultcount":6,"results":[{"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"LastModified":1644749267,"Maintainer":"violate","Name":"attest","NumVotes":42,"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"},{"Description":"This is a desciptive text for package attestation","FirstSubmitted":1644749269,"ID":75661,"LastModified":1644749269,"Maintainer":null,"Name":"attestation","NumVotes":39,"OutOfDate":null,"PackageBase":"attestation","PackageBaseID":75661,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","Version":"4.18.64-2"},{"Description":"This is a desciptive text for package attestations","FirstSubmitted":1644749269,"ID":74902,"LastModified":1644749269,"Maintainer":"gilchrists","Name":"attestations","NumVotes":44,"OutOfDate":null,"PackageBase":"attestations","PackageBaseID":74902,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Version":"4.9-9"},{"Description":"This is a desciptive text for package attested","FirstSubmitted":1644749268,"ID":71241,"LastModified":1644749268,"Maintainer":null,"Name":"attested","NumVotes":45,"OutOfDate":null,"PackageBase":"attested","PackageBaseID":71241,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","Version":"0.2.33-2"},{"Description":"This is a desciptive text for package attesting","FirstSubmitted":1644749268,"ID":67658,"LastModified":1644749268,"Maintainer":"amorphousness","Name":"attesting","NumVotes":51,"OutOfDate":null,"PackageBase":"attesting","PackageBaseID":67658,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Version":"1.14.65-10"},{"Description":"This is a desciptive text for package attests","FirstSubmitted":1644749268,"ID":42783,"LastModified":1644749268,"Maintainer":"injudicious","Name":"attests","NumVotes":48,"OutOfDate":null,"PackageBase":"attests","PackageBaseID":42783,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Version":"8.13.74-4"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=name&arg=x":                   {`{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=maintainer&arg=mistrustful":   {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attitudinize","FirstSubmitted":1644749268,"ID":64246,"LastModified":1644749268,"Maintainer":"mistrustful","Name":"attitudinize","NumVotes":42,"OutOfDate":null,"PackageBase":"attitudinize","PackageBaseID":64246,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attitudinize.tar.gz","Version":"7.17.87-9"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=maintainer&arg=gibbered":      {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backlog","FirstSubmitted":1644749267,"ID":29230,"LastModified":1644749267,"Maintainer":"giBBered","Name":"backlog","NumVotes":53,"OutOfDate":null,"PackageBase":"backlog","PackageBaseID":29230,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backlog.tar.gz","Version":"0.12.52-5"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=maintainer&arg=gIbbeRed":      {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backlog","FirstSubmitted":1644749267,"ID":29230,"LastModified":1644749267,"Maintainer":"giBBered","Name":"backlog","NumVotes":53,"OutOfDate":null,"PackageBase":"backlog","PackageBaseID":29230,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backlog.tar.gz","Version":"0.12.52-5"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=maintainer&arg=x":             {`{"resultcount":0,"results":[],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=maintainer&arg=":              {`{"error":"Too many package results.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=maintainer":                   {`{"error":"Too many package results.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=msearch&arg=mistrustful":                {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package attitudinize","FirstSubmitted":1644749268,"ID":64246,"LastModified":1644749268,"Maintainer":"mistrustful","Name":"attitudinize","NumVotes":42,"OutOfDate":null,"PackageBase":"attitudinize","PackageBaseID":64246,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/attitudinize.tar.gz","Version":"7.17.87-9"}],"type":"msearch","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=nonsense&arg=bla":                       {`{"error":"Incorrect request type specified.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=4&type=search&arg=bla":                         {`{"error":"Invalid version specified.","resultcount":0,"results":[],"type":"error","version":4}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&arg=a":                           {`{"error":"Query arg too small.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?type=search&arg=a":                               {`{"error":"Please specify an API version.","resultcount":0,"results":[],"type":"error","version":null}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search":                                 {`{"error":"No request type/data specified.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&arg=bla":                                     {`{"error":"No request type/data specified.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=info&arg=attest&callback=test":          {`/**/test({"resultcount":1,"results":[{"CheckDepends":["acyclovir","severals"],"Conflicts":["georginas","craw","lift"],"Description":"This is a desciptive text for package attest","FirstSubmitted":1644749267,"ID":25746,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"violate","MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"Name":"attest","NumVotes":42,"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"OutOfDate":null,"PackageBase":"attest","PackageBaseID":25746,"Popularity":0,"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Replaces":["brutishness","messaged","abut"],"URL":null,"URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Version":"2.11.73-4"}],"type":"multiinfo","version":5})`, "text/javascript"},
		"/rpc?v=5&type=info&arg=attest&callback=test[":         {`{"error":"Invalid callback name.","resultcount":0,"results":[],"type":"error","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=submitter&arg=someone":        {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backwash","FirstSubmitted":1644749267,"ID":22442,"LastModified":1644749267,"Maintainer":"unrepresented","Name":"backwash","NumVotes":34,"OutOfDate":null,"PackageBase":"backwash","PackageBaseID":22442,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backwash.tar.gz","Version":"6.0.38-1"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=comaintainers&arg=testmaint":  {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backwash","FirstSubmitted":1644749267,"ID":22442,"LastModified":1644749267,"Maintainer":"unrepresented","Name":"backwash","NumVotes":34,"OutOfDate":null,"PackageBase":"backwash","PackageBaseID":22442,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backwash.tar.gz","Version":"6.0.38-1"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=search&by=comaintainers&arg=testMaint2": {`{"resultcount":1,"results":[{"Description":"This is a desciptive text for package backwash","FirstSubmitted":1644749267,"ID":22442,"LastModified":1644749267,"Maintainer":"unrepresented","Name":"backwash","NumVotes":34,"OutOfDate":null,"PackageBase":"backwash","PackageBaseID":22442,"Popularity":0,"URL":null,"URLPath":"/cgit/aur.git/snapshot/backwash.tar.gz","Version":"6.0.38-1"}],"type":"search","version":5}`, consts.ContentTypeJson},
		"/rpc?v=5&type=info&arg=backwash":                      {`{"resultcount":1,"results":[{"CoMaintainers":["testmaint","testmaint2"],"CheckDepends":["transpiring","parquet"],"Conflicts":["corries","workday"],"Depends":["incorrect","splashed"],"Description":"This is a desciptive text for package backwash","FirstSubmitted":1644749267,"ID":22442,"Keywords":[],"LastModified":1644749267,"License":[],"Maintainer":"unrepresented","MakeDepends":["wish","crudities"],"Name":"backwash","NumVotes":34,"OptDepends":["used: for mourners","testicles: for psephologists"],"OutOfDate":null,"PackageBase":"backwash","PackageBaseID":22442,"Popularity":0,"Provides":["lifes","coimbatore","entreats","bodleian"],"Replaces":["anatomists","grammars"],"Submitter":"someone","URL":null,"URLPath":"/cgit/aur.git/snapshot/backwash.tar.gz","Version":"6.0.38-1"}],"type":"multiinfo","version":5}`, consts.ContentTypeJson},

		// v6
		"/api/v6/info/attest":                               {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/ATTEST":                               {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/name/attest":                          {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/name/ATTEST":                          {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/provides/attest":                      {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/provides/ATTEST":                      {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/replaces/reintroducing":               {`{"resultcount":1,"results":[{"Name":"baccarats","Description":"This is a desciptive text for package baccarats","Version":"4.3.85-5","PackageBase":"baccarats","URLPath":"/cgit/aur.git/snapshot/baccarats.tar.gz","Maintainer":"unmasks","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":53,"Depends":["gassers","bong","poos"],"MakeDepends":["eructation","outhouses","unloose"],"OptDepends":["newss: for misshaped","brighton: for guys","lacs: for adorer"],"CheckDepends":["flagmans","rump","herbivorous"],"Provides":["britches","stoppered","heshvan"],"Conflicts":["tomatoes","pyramids","presaging","transgressions"],"Replaces":["reintroducing","dogwoods","delphinium","anaerobically","aftermaths"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/conflicts/tomatoes":                   {`{"resultcount":1,"results":[{"Name":"baccarats","Description":"This is a desciptive text for package baccarats","Version":"4.3.85-5","PackageBase":"baccarats","URLPath":"/cgit/aur.git/snapshot/baccarats.tar.gz","Maintainer":"unmasks","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":53,"Depends":["gassers","bong","poos"],"MakeDepends":["eructation","outhouses","unloose"],"OptDepends":["newss: for misshaped","brighton: for guys","lacs: for adorer"],"CheckDepends":["flagmans","rump","herbivorous"],"Provides":["britches","stoppered","heshvan"],"Conflicts":["tomatoes","pyramids","presaging","transgressions"],"Replaces":["reintroducing","dogwoods","delphinium","anaerobically","aftermaths"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/maintainer/gibbered":                  {`{"resultcount":1,"results":[{"Name":"backlog","Description":"This is a desciptive text for package backlog","Version":"0.12.52-5","PackageBase":"backlog","URLPath":"/cgit/aur.git/snapshot/backlog.tar.gz","Maintainer":"giBBered","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":53,"Depends":["ebonys","pasquales","mahlers"],"MakeDepends":["orangutans"],"CheckDepends":["wristwatch","insensitivity"],"Conflicts":["reichs","procrastinators","culmination"],"Replaces":["ramifications","utilizes","cheerfulnesss"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/submitter/someone":                    {`{"resultcount":1,"results":[{"Name":"backwash","Description":"This is a desciptive text for package backwash","Version":"6.0.38-1","PackageBase":"backwash","URLPath":"/cgit/aur.git/snapshot/backwash.tar.gz","Maintainer":"unrepresented","Submitter":"someone","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":34,"Depends":["incorrect","splashed"],"MakeDepends":["wish","crudities"],"OptDepends":["used: for mourners","testicles: for psephologists"],"CheckDepends":["transpiring","parquet"],"Provides":["lifes","coimbatore","entreats","bodleian"],"Conflicts":["corries","workday"],"Replaces":["anatomists","grammars"],"CoMaintainers":["testmaint","testmaint2"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/comaintainers/testmaint":              {`{"resultcount":1,"results":[{"Name":"backwash","Description":"This is a desciptive text for package backwash","Version":"6.0.38-1","PackageBase":"backwash","URLPath":"/cgit/aur.git/snapshot/backwash.tar.gz","Maintainer":"unrepresented","Submitter":"someone","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":34,"Depends":["incorrect","splashed"],"MakeDepends":["wish","crudities"],"OptDepends":["used: for mourners","testicles: for psephologists"],"CheckDepends":["transpiring","parquet"],"Provides":["lifes","coimbatore","entreats","bodleian"],"Conflicts":["corries","workday"],"Replaces":["anatomists","grammars"],"CoMaintainers":["testmaint","testmaint2"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/groups/nothing":                       {`{"resultcount":1,"results":[{"Name":"backwoodsmans","Description":"This is a desciptive text for package backwoodsmans","Version":"9.8.61-4","PackageBase":"backwoodsmans","URLPath":"/cgit/aur.git/snapshot/backwoodsmans.tar.gz","Maintainer":"sss","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":46,"Depends":["castaways","means"],"MakeDepends":["sleazy"],"OptDepends":["woos: for musky"],"CheckDepends":["prawn","domiciliary"],"Provides":["afforesting","novartis"],"Conflicts":["pinochles"],"Replaces":["quays","airbase","smooth"],"Groups":["nothing"],"Keywords":["nonsense"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/depends/chrystals":                    {`{"resultcount":1,"results":[{"Name":"attics","Description":"This is a desciptive text for package attics","Version":"8.5-10","PackageBase":"attics","URLPath":"/cgit/aur.git/snapshot/attics.tar.gz","Maintainer":"supergrasses","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"Depends":["chrystals","joblessness","coast","preteens","philter"],"MakeDepends":["unsuited","coiled","reputes","eugenia","exhumed"],"OptDepends":["coverts: for avocational"],"CheckDepends":["peopled","whiskeys","benedicts"],"Provides":["disturber","appraising"],"Conflicts":["overthrew","infusion","prospectors","plumpness","prefabbing","coroner","swedishs"],"Replaces":["tailspin","hombre","meridian","dockers","safaried"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/optdepends/bhopal":                    {`{"resultcount":1,"results":[{"Name":"attired","Description":"This is a desciptive text for package attired","Version":"9.2-4","PackageBase":"attired","URLPath":"/cgit/aur.git/snapshot/attired.tar.gz","Maintainer":"backtalks","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":51,"MakeDepends":["horatio","nuanced","deistic"],"OptDepends":["bhopal: for turgenevs","contagious: for quiescent"],"CheckDepends":["velcros"],"Provides":["stepchildrens","earthy","christian"],"Conflicts":["cockier","pounced"],"Replaces":["transitivenesss"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/makedepends/constructive":             {`{"resultcount":1,"results":[{"Name":"attitudinized","Description":"This is a desciptive text for package attitudinized","Version":"2.15-6","PackageBase":"attitudinized","URLPath":"/cgit/aur.git/snapshot/attitudinized.tar.gz","Maintainer":"ptolemies","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":46,"MakeDepends":["constructive"],"OptDepends":["massiveness: for paranoids"],"Provides":["trichinosis"],"Replaces":["brasses"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/checkdepends/amphibian":               {`{"resultcount":1,"results":[{"Name":"attire","Description":"This is a desciptive text for package attire","Version":"0.14.38-2","PackageBase":"attire","URLPath":"/cgit/aur.git/snapshot/attire.tar.gz","Maintainer":"punish","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"CheckDepends":["amphibian"],"Replaces":["nontechnical"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/keywords/nonsense":                    {`{"resultcount":1,"results":[{"Name":"backwoodsmans","Description":"This is a desciptive text for package backwoodsmans","Version":"9.8.61-4","PackageBase":"backwoodsmans","URLPath":"/cgit/aur.git/snapshot/backwoodsmans.tar.gz","Maintainer":"sss","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":46,"Depends":["castaways","means"],"MakeDepends":["sleazy"],"OptDepends":["woos: for musky"],"CheckDepends":["prawn","domiciliary"],"Provides":["afforesting","novartis"],"Conflicts":["pinochles"],"Replaces":["quays","airbase","smooth"],"Groups":["nothing"],"Keywords":["nonsense"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info/nonsense/blubb":                       {`{"error":"Incorrect by field specified.","resultcount":0,"results":[],"type":"error","version":6}`, consts.ContentTypeJson},
		"/api/v6/info?by=provides&arg=attest":               {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info?by=name&arg=attest":                   {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info?arg=attest":                           {`{"resultcount":1,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info?by=provides&arg=attest&arg=awfulness": {`{"resultcount":3,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"awfulness","Description":"This is a desciptive text for package awfulness","Version":"3.7-5","PackageBase":"awfulness","URLPath":"/cgit/aur.git/snapshot/awfulness.tar.gz","Maintainer":"fatalists","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":50,"Depends":["cays","ifs","coaxed"],"CheckDepends":["cunt","anathematized"],"Provides":["sellerss"],"Conflicts":["functionalist","accessorizes","agree","altered"]},{"Name":"backyard","Description":"This is a desciptive text for package backyard","Version":"4.18-3","PackageBase":"backyard","URLPath":"/cgit/aur.git/snapshot/backyard.tar.gz","Maintainer":"comers","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":43,"MakeDepends":["aztecs","mouthe","stochastic"],"OptDepends":["amorphously: for angela"],"CheckDepends":["peptide","enthusiastic","daft"],"Provides":["awfulness","brindled","streaming","haifas"],"Conflicts":["hope","doppelganger"],"Replaces":["sigma","nuclear"]}],"type":"multiinfo","version":6}`, consts.ContentTypeJson},
		"/api/v6/info?by=provides":                          {`{"error":"No request data specified.","resultcount":0,"results":[],"type":"error","version":6}`, consts.ContentTypeJson},

		"/api/v6/search/atTest":                    {`{"resultcount":6,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/at test":                   {`{"resultcount":6,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/at te st":                  {`{"resultcount":7,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]},{"Name":"baathist","Description":"This is a desciptive text for package baathist","Version":"2.2-2","PackageBase":"baathist","URLPath":"/cgit/aur.git/snapshot/baathist.tar.gz","Maintainer":"claque","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":46,"Depends":["leukocyte","convincingly","milds"],"MakeDepends":["ignominious","wildcatting","standee","spca"],"OptDepends":["reggie: for moister","lists: for intuitivenesss","jetports: for unprincipled"],"Provides":["contusion","sauntering"],"Conflicts":["inured","masterminds","subdues","gienah","rehearsing"],"Replaces":["vested","appreciation","towlines"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/attest":               {`{"resultcount":6,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name-desc/attest":          {`{"resultcount":6,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/contains/attest":      {`{"resultcount":6,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/starts-with/attest":   {`{"resultcount":6,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/nonsense/attest":           {`{"error":"Incorrect by field specified.","resultcount":0,"results":[],"type":"error","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/nonsense/nonsense/attest":  {`{"error":"Incorrect by field specified.","resultcount":0,"results":[],"type":"error","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/nonsense/attest":      {`{"error":"Incorrect mode specified.","resultcount":0,"results":[],"type":"error","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/starts-with/nonsense": {`{"resultcount":0,"results":[],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/contains/non":         {`{"resultcount":2,"results":[{"Name":"avignon","Description":"This is a desciptive text for package avignon","Version":"1.1-8","PackageBase":"avignon","URLPath":"/cgit/aur.git/snapshot/avignon.tar.gz","Maintainer":"energizing","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":62,"Depends":["ear","alberta","sheltons"],"MakeDepends":["yucca","shaves","stricter"],"OptDepends":["exoplanet: for keewatins","fig: for chaucer","pavlovians: for satchels","unsubscribes: for subjugate","scott: for premature","toothier: for chamomiles"],"CheckDepends":["ramons","akivas"],"Provides":["scrumping","degas","requiems","appositive"],"Conflicts":["mobsters","backbiters","nausea","coma"],"Replaces":["reweaving","texts","suwanees","thievings","footlights","bouts"]},{"Name":"avignons","Description":"This is a desciptive text for package avignons","Version":"3.14-9","PackageBase":"avignons","URLPath":"/cgit/aur.git/snapshot/avignons.tar.gz","Maintainer":"effaced","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":48,"MakeDepends":["packinghouses"],"OptDepends":["degeneration: for cheerlessness","deon: for cowardly","conscienceless: for gradualism"],"CheckDepends":["foment","latitude","cadillac","idolatresses"],"Provides":["resistivity","disbursals"],"Conflicts":["quechua","derivations","scientifically"],"Replaces":["skycaps","wrings","suitability"]}],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/starts-with/test":     {`{"resultcount":0,"results":[],"type":"search","version":6}`, consts.ContentTypeJson},
		"/api/v6/search/name/contains/test":        {`{"resultcount":7,"results":[{"Name":"attest","Description":"This is a desciptive text for package attest","Version":"2.11.73-4","PackageBase":"attest","URLPath":"/cgit/aur.git/snapshot/attest.tar.gz","Maintainer":"violate","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":42,"MakeDepends":["answerable","ingrained","circumscribed","crust","landsats","emptier"],"OptDepends":["lowermost: for unanswered","racquetballs: for ornaments","slit: for dichotomy"],"CheckDepends":["acyclovir","severals"],"Provides":["superber","acupuncture","destination","rota","shoeshine"],"Conflicts":["georginas","craw","lift"],"Replaces":["brutishness","messaged","abut"]},{"Name":"attestation","Description":"This is a desciptive text for package attestation","Version":"4.18.64-2","PackageBase":"attestation","URLPath":"/cgit/aur.git/snapshot/attestation.tar.gz","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":39,"Depends":["damson","nearer","friar"],"OptDepends":["ungenerous: for matt"],"CheckDepends":["uptick","zeitgeist","surprising","pin"],"Provides":["invidiousness","canoeists","hobart","pugnaciousness"],"Conflicts":["sabre","manganese"],"Replaces":["eructs","dantons"]},{"Name":"attestations","Description":"This is a desciptive text for package attestations","Version":"4.9-9","PackageBase":"attestations","URLPath":"/cgit/aur.git/snapshot/attestations.tar.gz","Maintainer":"gilchrists","FirstSubmitted":1644749269,"LastModified":1644749269,"NumVotes":44,"Depends":["vivariums"],"MakeDepends":["hyperthyroidisms","moleskin"],"OptDepends":["breaded: for beasley","biopsy: for scylla"],"CheckDepends":["vigils","eschewing"],"Provides":["earnestness","conveyor","axiom"],"Conflicts":["obnoxiousness","bugging"],"Replaces":["dogcart","gorgon"]},{"Name":"attested","Description":"This is a desciptive text for package attested","Version":"0.2.33-2","PackageBase":"attested","URLPath":"/cgit/aur.git/snapshot/attested.tar.gz","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":45,"Depends":["rupees","reattempted"],"MakeDepends":["pettifogged","referendum","buchanans","intravenously"],"OptDepends":["momentary: for primeval"],"CheckDepends":["nouakchotts","plasterer","gamier","perished"],"Provides":["dumpster","embroiderys","dispersed","inglorious","outdid","counterattacked"],"Conflicts":["arbitrators","nadines","smiths","riotous"],"Replaces":["kennan"]},{"Name":"attesting","Description":"This is a desciptive text for package attesting","Version":"1.14.65-10","PackageBase":"attesting","URLPath":"/cgit/aur.git/snapshot/attesting.tar.gz","Maintainer":"amorphousness","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":51,"Depends":["overcasts","jingles","josie","facepalm"],"MakeDepends":["kit","dados","witnessing","votes"],"OptDepends":["crayfishes: for parsonages","deactivated: for refugee","bedtimes: for fleeing"],"CheckDepends":["expansiveness"],"Provides":["trigonometrys","overturns","giggling","scone","memorial"],"Conflicts":["pertussis","emf","penning"],"Replaces":["contents","bibliophiles","spiritual","constitute"]},{"Name":"attests","Description":"This is a desciptive text for package attests","Version":"8.13.74-4","PackageBase":"attests","URLPath":"/cgit/aur.git/snapshot/attests.tar.gz","Maintainer":"injudicious","FirstSubmitted":1644749268,"LastModified":1644749268,"NumVotes":48,"Depends":["surveyor"],"MakeDepends":["watchmaker","fringing","packsaddles","enjoy"],"OptDepends":["headphones: for naphthalenes"],"CheckDepends":["phones","headily"],"Provides":["hungry","placket"],"Conflicts":["tangibles","taxon","lawmaking"],"Replaces":["editing","refortifies","tabbies"]},{"Name":"augustest","Description":"This is a desciptive text for package augustest","Version":"5.9-4","PackageBase":"augustest","URLPath":"/cgit/aur.git/snapshot/augustest.tar.gz","Maintainer":"liquidation","FirstSubmitted":1644749267,"LastModified":1644749267,"NumVotes":50,"CheckDepends":["gadabouts"],"Replaces":["tiresias"]}],"type":"search","version":6}`, consts.ContentTypeJson},

		"/api/v6/info?by=provides&arg=awfulness&arg=rollerblades&arg=pastime&arg=bliss&arg=idleness&arg=ambushed&arg=spectate&arg=retakes&arg=tradeswoman&arg=belfries": {`{"error":"Too many package results.","resultcount":0,"results":[],"type":"error","version":6}`, consts.ContentTypeJson},
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

	suite.ExpectedAdminResultsGET = map[string]responseResult{
		"/admin/settings/aur-file-location":           {`Current setting for 'AurFileLocation' is '../../test_data/test_packages.json'`, consts.ContentTypeText},
		"/admin/settings/max-results":                 {`Current setting for 'MaxResults' is '5000'`, consts.ContentTypeText},
		"/admin/settings/refresh-interval":            {`Current setting for 'RefreshInterval' is '600'`, consts.ContentTypeText},
		"/admin/settings/rate-limit":                  {`Current setting for 'RateLimit' is '4000'`, consts.ContentTypeText},
		"/admin/settings/rate-limit-cleanup-interval": {`Current setting for 'RateLimitCleanupInterval' is '600'`, consts.ContentTypeText},
		"/admin/settings/rate-limit-time-window":      {`Current setting for 'RateLimitTimeWindow' is '86400'`, consts.ContentTypeText},
		"/admin/settings/cache-cleanup-interval":      {`Current setting for 'CacheCleanupInterval' is '60'`, consts.ContentTypeText},
		"/admin/settings/cache-expiration-time":       {`Current setting for 'CacheExpirationTime' is '300'`, consts.ContentTypeText},
		"/admin/settings/enable-search-cache":         {`Current setting for 'EnableSearchCache' is 'true'`, consts.ContentTypeText},
		"/admin/settings":                             {"{\n\t\"Port\": 10667,\n\t\"AurFileLocation\": \"../../test_data/test_packages.json\",\n\t\"MaxResults\": 5000,\n\t\"RefreshInterval\": 600,\n\t\"RateLimit\": 4000,\n\t\"LoadFromFile\": true,\n\t\"RateLimitCleanupInterval\": 600,\n\t\"RateLimitTimeWindow\": 86400,\n\t\"TrustedReverseProxies\": [\n\t\t\"127.0.0.1\",\n\t\t\"::1\"\n\t],\n\t\"EnableSSL\": false,\n\t\"CertFile\": \"\",\n\t\"KeyFile\": \"\",\n\t\"EnableSearchCache\": true,\n\t\"CacheCleanupInterval\": 60,\n\t\"CacheExpirationTime\": 300,\n\t\"LogFile\": \"/tmp/log.tst\",\n\t\"EnableMetrics\": true,\n\t\"EnableAdminApi\": true,\n\t\"AdminAPIKey\": \"test\"\n}", consts.ContentTypeJson},
		"/admin/settings/":                            {"{\n\t\"Port\": 10667,\n\t\"AurFileLocation\": \"../../test_data/test_packages.json\",\n\t\"MaxResults\": 5000,\n\t\"RefreshInterval\": 600,\n\t\"RateLimit\": 4000,\n\t\"LoadFromFile\": true,\n\t\"RateLimitCleanupInterval\": 600,\n\t\"RateLimitTimeWindow\": 86400,\n\t\"TrustedReverseProxies\": [\n\t\t\"127.0.0.1\",\n\t\t\"::1\"\n\t],\n\t\"EnableSSL\": false,\n\t\"CertFile\": \"\",\n\t\"KeyFile\": \"\",\n\t\"EnableSearchCache\": true,\n\t\"CacheCleanupInterval\": 60,\n\t\"CacheExpirationTime\": 300,\n\t\"LogFile\": \"/tmp/log.tst\",\n\t\"EnableMetrics\": true,\n\t\"EnableAdminApi\": true,\n\t\"AdminAPIKey\": \"test\"\n}", consts.ContentTypeJson},
	}

	suite.ExpectedAdminResultsPOST = map[string]string{
		"/admin/run-job/wipe-cache":         `Wiped search-cache (0 entries removed)`,
		"/admin/run-job/wipe-ratelimits":    `Wiped rate-limits (0 entries removed)`,
		"/admin/run-job/cleanup-cache":      `Cleaned up search-cache`,
		"/admin/run-job/cleanup-ratelimits": `Cleaned up rate-limits`,
		"/admin/run-job/nonsense":           `Job not found`,

		"/admin/settings/aur-file-location?value=xyz":         `Changed 'AurFileLocation' from '../../test_data/test_packages.json' to 'xyz'`,
		"/admin/settings/aur-file-location":                   `Need new value: ?value=...`,
		"/admin/settings/max-results?value=1":                 `Changed 'MaxResults' from '5000' to '1'`,
		"/admin/settings/max-results?value=0":                 `Value can not be 0`,
		"/admin/settings/max-results?value=x":                 `strconv.Atoi: parsing "x": invalid syntax`,
		"/admin/settings/max-results":                         `Need new value: ?value=...`,
		"/admin/settings/refresh-interval?value=1":            `Changed 'RefreshInterval' from '600' to '1'`,
		"/admin/settings/refresh-interval?value=x":            `strconv.Atoi: parsing "x": invalid syntax`,
		"/admin/settings/refresh-interval":                    `Need new value: ?value=...`,
		"/admin/settings/rate-limit?value=1":                  `Changed 'RateLimit' from '4000' to '1'`,
		"/admin/settings/rate-limit?value=0":                  "Changed 'RateLimit' from '4000' to '0'\nWARNING: Rate limit is disabled",
		"/admin/settings/rate-limit?value=x":                  `strconv.Atoi: parsing "x": invalid syntax`,
		"/admin/settings/rate-limit":                          `Need new value: ?value=...`,
		"/admin/settings/rate-limit-cleanup-interval?value=1": `Changed 'RateLimitCleanupInterval' from '600' to '1'`,
		"/admin/settings/rate-limit-cleanup-interval?value=x": `strconv.Atoi: parsing "x": invalid syntax`,
		"/admin/settings/rate-limit-cleanup-interval":         `Need new value: ?value=...`,
		"/admin/settings/rate-limit-time-window?value=1":      `Changed 'RateLimitTimeWindow' from '86400' to '1'`,
		"/admin/settings/rate-limit-time-window?value=x":      `strconv.Atoi: parsing "x": invalid syntax`,
		"/admin/settings/rate-limit-time-window":              `Need new value: ?value=...`,
		"/admin/settings/cache-cleanup-interval?value=1":      `Changed 'CacheCleanupInterval' from '60' to '1'`,
		"/admin/settings/cache-cleanup-interval?value=x":      `strconv.Atoi: parsing "x": invalid syntax`,
		"/admin/settings/cache-cleanup-interval":              `Need new value: ?value=...`,
		"/admin/settings/cache-expiration-time?value=1":       `Changed 'CacheExpirationTime' from '300' to '1'`,
		"/admin/settings/cache-expiration-time?value=x":       `strconv.Atoi: parsing "x": invalid syntax`,
		"/admin/settings/cache-expiration-time":               `Need new value: ?value=...`,
		"/admin/settings/enable-search-cache?value=false":     `Changed 'EnableSearchCache' from 'true' to 'false'`,
		"/admin/settings/enable-search-cache?value=x":         `strconv.ParseBool: parsing "x": invalid syntax`,
		"/admin/settings/enable-search-cache":                 `Need new value: ?value=...`,
		"/admin/settings/nonsense":                            `Setting not found`,
		"/admin/settings":                                     `Setting not found`,
		"/admin/settings/":                                    `Setting not found`,
	}

	var err error
	suite.srv, err = New(conf, true, true, "")
	suite.Nil(err, "Could not create rpc server")
	suite.srv.setupRoutes()

	// start webserver for some http tests
	modTime := time.Now().UTC()
	l, err := net.Listen("tcp", "127.0.0.1:10668")
	suite.Nil(err, err)
	suite.httpSrv = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cmodTime, _ := http.ParseTime(r.Header.Get("If-Modified-Since"))
			mod := r.URL.Query().Get("nomod") == ""

			if mod && (modTime.Equal(cmodTime) || cmodTime.Before(modTime)) {
				w.WriteHeader(304)
				return
			}

			if mod {
				w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
			}

			b, _ := os.ReadFile(conf.AurFileLocation)
			w.Write(b)
		}),
	}

	go suite.httpSrv.Serve(l)
}

// run before each test
func (suite *RpcTestSuite) SetupTest() {
	// reset settings
	suite.srv.conf = conf
	suite.srv.lastRefresh = time.Time{}
	suite.srv.reloadData()
}

// cleanup
func (suite *RpcTestSuite) TearDownSuite() {
	log := suite.srv.conf.LogFile
	_, err := os.Stat(log)
	if log != "" && err == nil {
		err = os.Remove(log)
		suite.Nil(err)
	}

	suite.httpSrv.Shutdown(context.TODO())

	fmt.Println(">>> RPC tests completed")
}

// test function returning a list of arguments
func (suite *RpcTestSuite) TestGetArgumentList() {
	for k, v := range suite.ExpectedArgumentsList {
		test := getArgsList(*k)
		suite.Equal(v, test)
	}
}

// test function returning a single argument
func (suite *RpcTestSuite) TestGetArgumentSingle() {
	for k, v := range suite.ExpectedArguments {
		test := getArg(*k)
		suite.Equal(v, test)
	}
}

// test handlers
func (suite *RpcTestSuite) TestRpcHandlers() {
	suite.srv.conf.MaxResults = 10

	for i := 0; i < 2; i++ {
		// get requests
		for k, v := range suite.ExpectedRpcResults {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", k, nil)
			suite.Nil(err, "Could not create GET request")

			suite.srv.router.ServeHTTP(rr, req)
			suite.Equal(v.expected, rr.Body.String(), "Input: "+k)
			suite.Equal(v.contentType, rr.Result().Header.Get("Content-Type"), "Input: "+k)
		}

		// post requests
		for k, v := range suite.ExpectedRpcResults {
			rr := httptest.NewRecorder()
			reader := &strings.Reader{}
			if len(strings.Split(k, "?")) > 1 {
				reader = strings.NewReader(strings.Split(k, "?")[1])
			}
			req, err := http.NewRequest("POST", strings.Split(k, "?")[0], reader)
			req.Header.Add("Content-Type", consts.ContentTypeForm)
			suite.Nil(err, "Could not create POST request")

			suite.srv.router.ServeHTTP(rr, req)
			suite.Equal(v.expected, rr.Body.String(), "Input: "+k)
			suite.Equal(v.contentType, rr.Result().Header.Get("Content-Type"), "Input: "+k)
		}

		// lets disable search cache and rate limit for the next iteration
		suite.srv.conf.EnableSearchCache = false
		suite.srv.conf.RateLimit = 0
	}
}

// test /admin handlers
func (suite *RpcTestSuite) TestAdminHandlers() {
	// test wrong key
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/admin/settings/xyz", nil)
	req.Header.Add("APIKey", "wrong")

	suite.srv.router.ServeHTTP(rr, req)
	suite.Nil(err, "Could not create GET request")
	suite.Equal("Unauthorized", rr.Body.String(), "Should return 'Unauthorized'")
	suite.Equal(http.StatusUnauthorized, rr.Result().StatusCode)
	suite.Equal(consts.ContentTypeText, rr.Result().Header.Get("Content-Type"))

	// get requests
	for k, v := range suite.ExpectedAdminResultsGET {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", k, nil)
		req.Header.Add("APIKey", "test")
		suite.Nil(err, "Could not create GET request")

		suite.srv.router.ServeHTTP(rr, req)
		suite.Equal(v.expected, rr.Body.String(), "Input: "+k)
		suite.Equal(v.contentType, rr.Result().Header.Get("Content-Type"), "Input: "+k)
	}

	// post requests
	for k, v := range suite.ExpectedAdminResultsPOST {
		suite.srv.conf.RateLimit = 4000

		rr := httptest.NewRecorder()
		req, err := http.NewRequest("POST", k, nil)
		req.Header.Add("Content-Type", consts.ContentTypeForm)
		req.Header.Add("APIKey", "test")
		suite.Nil(err, "Could not create POST request")

		suite.srv.router.ServeHTTP(rr, req)
		suite.Equal(v, rr.Body.String(), "Input: "+k)
		suite.Equal(consts.ContentTypeText, rr.Result().Header.Get("Content-Type"), "Input: "+k)
	}

	// data reload - ok
	suite.srv.lastRefresh = time.Time{}
	suite.srv.conf.AurFileLocation = conf.AurFileLocation
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/admin/run-job/reload-data", nil)
	req.Header.Add("APIKey", "test")

	suite.srv.router.ServeHTTP(rr, req)
	suite.Nil(err, "Could not create POST request")
	suite.Equal("Successfully reloaded data", rr.Body.String(), "Should return 'Successfully reloaded data'")
	suite.Equal(http.StatusAccepted, rr.Result().StatusCode)
	suite.Equal(consts.ContentTypeText, rr.Result().Header.Get("Content-Type"))

	// data reload, not changed
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/admin/run-job/reload-data", nil)
	req.Header.Add("APIKey", "test")

	suite.srv.router.ServeHTTP(rr, req)
	suite.Nil(err, "Could not create POST request")
	suite.Equal("Reload skipped. Data has not changed", rr.Body.String(), "Should return 'Reload skipped. Data has not changed'")
	suite.Equal(http.StatusAccepted, rr.Result().StatusCode)
	suite.Equal(consts.ContentTypeText, rr.Result().Header.Get("Content-Type"))

	// data reload - fail
	suite.srv.conf.AurFileLocation = "nonsense"
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/admin/run-job/reload-data", nil)
	req.Header.Add("APIKey", "test")

	suite.srv.router.ServeHTTP(rr, req)
	suite.Nil(err, "Could not create POST request")
	suite.Equal("stat nonsense: no such file or directory", rr.Body.String(), "Should return 'stat nonsense: no such file or directory'")
	suite.Equal(http.StatusInternalServerError, rr.Result().StatusCode)
	suite.Equal(consts.ContentTypeText, rr.Result().Header.Get("Content-Type"))
}

// test rate limit
func (suite *RpcTestSuite) TestRateLimit() {
	suite.srv.conf.RateLimit = 1

	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.handleRequest).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit)
			suite.Equal(consts.ContentTypeHtml, rr.Result().Header.Get("Content-Type"))
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
			suite.Equal(consts.ContentTypeJson, rr.Result().Header.Get("Content-Type"))
		}
	}

	// with X-Real-IP
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Add("X-Real-IP", "test_rate_limit_real_ip")
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.handleRequest).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit, "request number: ", i)
			suite.Equal(consts.ContentTypeHtml, rr.Result().Header.Get("Content-Type"))
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
			suite.Equal(consts.ContentTypeJson, rr.Result().Header.Get("Content-Type"))
		}
	}

	// with X-Forwarded-For
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Add("X-Forwarded-For", "test_rate_limit_x_forwarded")
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.handleRequest).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit, "request number: ", i)
			suite.Equal(consts.ContentTypeHtml, rr.Result().Header.Get("Content-Type"))
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
			suite.Equal(consts.ContentTypeJson, rr.Result().Header.Get("Content-Type"))
		}
	}

	// with X-Forwarded-For (multiple)
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/rpc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Add("X-Forwarded-For", "test_rate_limit_x_forwarded_multi, bla, blubb")
		suite.Nil(err, "Could not create request")

		http.HandlerFunc(suite.srv.handleRequest).ServeHTTP(rr, req)
		if i == 0 {
			suite.NotEqual(rr.Body.String(), suite.ExpectedRateLimit, "request number: ", i)
			suite.Equal(consts.ContentTypeHtml, rr.Result().Header.Get("Content-Type"))
		} else if i > 0 {
			suite.Equal(rr.Body.String(), suite.ExpectedRateLimit)
			suite.Equal(consts.ContentTypeJson, rr.Result().Header.Get("Content-Type"))
		}
	}
}

// test create server
func (suite *RpcTestSuite) TestListen() {
	suite.srv.conf.RateLimitCleanupInterval = 1
	suite.srv.conf.RefreshInterval = 1
	suite.srv.conf.CacheCleanupInterval = 1
	suite.srv.conf.CacheExpirationTime = 1
	suite.srv.lastRefresh = time.Time{}

	go func() {
		err := suite.srv.Listen()
		suite.Equal(http.ErrServerClosed, err)
	}()

	suite.srv.mutLimit.Lock()
	suite.srv.rateLimits["test"] = RateLimit{WindowStart: time.Now().AddDate(0, 0, -2), Requests: 1}
	suite.srv.mutLimit.Unlock()
	suite.srv.mutCache.Lock()
	suite.srv.searchCache["test"] = CacheEntry{}
	suite.srv.mutCache.Unlock()
	time.Sleep(1200 * time.Millisecond)
	suite.srv.mutLimit.Lock()
	suite.Empty(suite.srv.rateLimits) // check if rate limit got removed
	suite.srv.mutLimit.Unlock()
	suite.srv.mutCache.Lock()
	suite.Empty(suite.srv.searchCache)
	suite.srv.mutCache.Unlock()
	time.Sleep(1200 * time.Millisecond)
	suite.srv.Stop()
	srv, err := New(confBroken, false, false, "")
	suite.Nil(err)
	suite.NotNil(srv.Listen())
}

// test data reload
func (suite *RpcTestSuite) TestReload() {
	// since we reload data before each test, this should throw "not modified"
	err := suite.srv.reloadData()
	suite.Equal("not modified", err.Error(), "Should be \"not modified\"")

	// should reload data
	suite.srv.lastRefresh = time.Time{}
	err = suite.srv.reloadData()
	suite.Nil(err, "Error reloading data")

	modTime := time.Now().UTC()

	// reload from http
	suite.srv.conf.AurFileLocation = "http://127.0.0.1:10668/test_packages.json"
	suite.srv.conf.LoadFromFile = false
	suite.srv.lastRefresh = modTime.Add(time.Hour * -1)
	err = suite.srv.reloadData()
	suite.NotNil(err, "Error should not be nil")
	suite.Equal("not modified", err.Error(), "Should be \"not modified\"")

	suite.srv.lastRefresh = time.Now().UTC().Add(time.Second * 1)
	err = suite.srv.reloadData()
	suite.Nil(err, err)

	// test for servers not providing "Last-Modified" header
	suite.srv.conf.AurFileLocation = "http://127.0.0.1:10668/test_packages.json?nomod=1"
	suite.srv.conf.LoadFromFile = false
	suite.srv.lastRefresh = modTime.Add(time.Hour * -1)
	err = suite.srv.reloadData()
	suite.Nil(err, err)
}

// purposefully crash reload function
func (suite *RpcTestSuite) TestBrokenReload() {
	suite.srv.conf.AurFileLocation = "x"
	suite.NotNil(suite.srv.reloadData(), "Should return an error")
}

// test stats
func (suite *RpcTestSuite) TestStats() {
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/rpc/stats", nil)
	suite.Nil(err, "Could not create GET request")

	suite.srv.router.ServeHTTP(rr, req)
	suite.Equal(200, rr.Result().StatusCode)
	suite.Equal(consts.ContentTypeHtml, rr.Result().Header.Get("Content-Type"))
}

// run our tests
func TestRPCTestSuite(t *testing.T) {
	suite.Run(t, new(RpcTestSuite))
}
