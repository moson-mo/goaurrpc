package rpc

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/moson-mo/goaurrpc/internal/consts"
	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"github.com/moson-mo/goaurrpc/internal/metrics"

	"github.com/goccy/go-json"
	"gopkg.in/guregu/null.v4"
)

// allowed query types
var queryTypes = []string{
	"info",
	"multiinfo",
	"search",
	"msearch",
	"suggest",
	"suggest-pkgbase",
	"opensearch-suggest",
	"opensearch-suggest-pkgbase",
}

// allowed "by" values
var queryBy = []string{
	"",
	"name",
	"name-desc",
	"maintainer",
	"submitter",
	"depends",
	"makedepends",
	"optdepends",
	"checkdepends",
	"provides",
	"conflicts",
	"replaces",
	"keywords",
	"groups",
	"comaintainers",
}

// allowed "mode" values
var queryMode = []string{
	"",
	"contains",
	"starts-with",
}

var ErrCallBack = errors.New("Invalid callback name.")

// Checking the validity of the query parameters
func validateParameters(params url.Values) error {
	arg, hasArg := params["arg"]
	_, hasArgArr := params["arg[]"]
	v := params.Get("v")
	t := params.Get("type")
	by := params.Get("by")
	m := params.Get("mode")

	if v == "" {
		return errors.New("Please specify an API version.")
	}
	if v != "5" && v != "6" {
		return errors.New("Invalid version specified.")
	}
	if strings.ToLower(t) == "" {
		return errors.New("No request type/data specified.")
	}
	if !inSlice(queryTypes, t) {
		return errors.New("Incorrect request type specified.")
	}
	if !inSlice(queryBy, by) {
		return errors.New("Incorrect by field specified.")
	}
	if !inSlice(queryMode, m) {
		return errors.New("Incorrect mode specified.")
	}
	if v == "6" && len(arg) == 0 {
		return errors.New("No request data specified.")
	}
	if !hasArg && !hasArgArr && by != "maintainer" {
		return errors.New("No request type/data specified.")
	}
	if ((hasArg && len(params.Get("arg")) < 2) || (hasArgArr && len(params.Get("arg[]")) < 2)) &&
		strings.HasPrefix(t, "search") &&
		by != "maintainer" {
		return errors.New("Query arg too small.")
	}
	if params.Get("callback") != "" {
		match, _ := regexp.MatchString("^[a-zA-Z0-9()_.]{1,128}$", params.Get("callback"))
		if !match {
			return ErrCallBack
		}
	}

	return nil
}

// get a string slice with all arguments that have been passed
func getArgsList(params url.Values) []string {
	var args []string

	if params.Get("v") == "6" {
		for _, arg := range params["arg"] {
			args = append(args, strings.ToLower(arg))
		}
		return args
	}
	if params.Get("arg") != "" {
		args = append(args, strings.ToLower(params.Get("arg")))
	} else {
		for _, arg := range params["arg[]"] {
			args = append(args, strings.ToLower(arg))
		}
	}
	return args
}

// get a single argument
func getArg(params url.Values) string {
	if params.Get("arg") != "" {
		return strings.ToLower(params.Get("arg"))
	}
	return strings.ToLower(params.Get("arg[]"))
}

// get search "by" parameter
func getBy(params url.Values) string {
	// if not specified use name and description for search
	by := "name-desc"
	if params.Get("by") != "" {
		by = params.Get("by")
	}

	// in case we did not get a by parameter with an info call, set name as by
	if params.Get("type") == "info" && params.Get("by") == "" {
		by = "name"
	}

	// if type is msearch we search by maintainer
	if params.Get("type") == "msearch" {
		by = "maintainer"
	}

	return by
}

// generate JSON error and return to client
func writeError(code int, message string, version int, callback string, w http.ResponseWriter) {
	e := RpcResult{
		Error:   message,
		Type:    "error",
		Results: make([]interface{}, 0),
		Version: null.NewInt(int64(version), version != 0),
	}

	b, _ := json.Marshal(e)

	sendResult(code, callback, b, w)

	// update request errors metric
	metrics.RequestErrors.WithLabelValues(e.Error).Inc()
}

// generate JSON string from RpcResult and return to client
func writeResult(result *RpcResult, callback string, w http.ResponseWriter) {
	// set number of records
	if result.Resultcount == 0 {
		result.Results = make([]interface{}, 0)
	}
	b, _ := json.Marshal(result)

	sendResult(200, callback, b, w)

	// update request size metrics
	metrics.ResponseSize.WithLabelValues(result.Type).Observe(float64(len(b)))
}

// sends data to client
func sendResult(code int, callback string, b []byte, w http.ResponseWriter) {
	if callback != "" {
		w.Header().Set("Content-Type", consts.ContentTypeJS)
		w.WriteHeader(code)
		fmt.Fprintf(w, "/**/%s(%s)", callback, string(b))
		return
	}
	w.Header().Set("Content-Type", consts.ContentTypeJson)
	w.WriteHeader(code)
	w.Write(b)
}

// get the client IP-Address. If behind a reverse proxy, obtain it from the X-Real-IP header
func getRealIP(r *http.Request, trustedProxies []string) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	realIP := r.Header.Get("X-Real-IP")
	fwdIP := r.Header.Get("X-Forwarded-For")
	isProxyTrusted := inSlice(trustedProxies, ip)

	if realIP != "" && isProxyTrusted {
		return realIP
	}
	if fwdIP != "" && isProxyTrusted {
		ips := strings.Split(fwdIP, ", ")
		if len(ips) > 1 {
			return ips[0]
		}
		return fwdIP
	}
	return ip
}

// converts db.PackageInfo to rpc.InfoRecord
func convDbPkgToInfoRecord(dbp *db.PackageInfo) InfoRecord {
	ir := InfoRecord{
		ID:             dbp.ID,
		Name:           dbp.Name,
		PackageBaseID:  dbp.PackageBaseID,
		PackageBase:    dbp.PackageBase,
		Version:        dbp.Version,
		Description:    null.NewString(dbp.Description, dbp.Description != ""),
		URL:            null.NewString(dbp.URL, dbp.URL != ""),
		NumVotes:       dbp.NumVotes,
		Popularity:     dbp.Popularity,
		OutOfDate:      null.NewInt(int64(dbp.OutOfDate), dbp.OutOfDate != 0),
		Maintainer:     null.NewString(dbp.Maintainer, dbp.Maintainer != ""),
		Submitter:      dbp.Submitter,
		FirstSubmitted: dbp.FirstSubmitted,
		LastModified:   dbp.LastModified,
		URLPath:        null.NewString(dbp.URLPath, dbp.URLPath != ""),
		MakeDepends:    dbp.MakeDepends,
		License:        dbp.License,
		Depends:        dbp.Depends,
		Conflicts:      dbp.Conflicts,
		Provides:       dbp.Provides,
		Keywords:       dbp.Keywords,
		OptDepends:     dbp.OptDepends,
		CheckDepends:   dbp.CheckDepends,
		Replaces:       dbp.Replaces,
		Groups:         dbp.Groups,
		CoMaintainers:  dbp.CoMaintainers,
	}
	/*
		for some reason Keywords and License should be returned
		as empty JSON arrays rather than being "null"
	*/
	if ir.Keywords == nil {
		ir.Keywords = []string{}
	}
	if ir.License == nil {
		ir.License = []string{}
	}

	return ir
}

// converts db.PackageInfo to rpc.PackageData
func convDbPkgToPackageData(dbp *db.PackageInfo) PackageData {
	ir := PackageData{
		Name:           dbp.Name,
		PackageBase:    dbp.PackageBase,
		Version:        dbp.Version,
		Description:    dbp.Description,
		URL:            dbp.URL,
		NumVotes:       dbp.NumVotes,
		Popularity:     dbp.Popularity,
		OutOfDate:      dbp.OutOfDate,
		Maintainer:     dbp.Maintainer,
		Submitter:      dbp.Submitter,
		FirstSubmitted: dbp.FirstSubmitted,
		LastModified:   dbp.LastModified,
		URLPath:        dbp.URLPath,
		MakeDepends:    dbp.MakeDepends,
		License:        dbp.License,
		Depends:        dbp.Depends,
		Conflicts:      dbp.Conflicts,
		Provides:       dbp.Provides,
		Keywords:       dbp.Keywords,
		OptDepends:     dbp.OptDepends,
		CheckDepends:   dbp.CheckDepends,
		Replaces:       dbp.Replaces,
		Groups:         dbp.Groups,
		CoMaintainers:  dbp.CoMaintainers,
	}

	return ir
}

// converts db.PackageInfo to rpc.SearchRecord
func convDbPkgToSearchRecord(dbp *db.PackageInfo) SearchRecord {
	sr := SearchRecord{
		ID:             dbp.ID,
		PackageBaseID:  dbp.PackageBaseID,
		Description:    null.NewString(dbp.Description, dbp.Description != ""),
		FirstSubmitted: dbp.FirstSubmitted,
		LastModified:   dbp.LastModified,
		Maintainer:     null.NewString(dbp.Maintainer, dbp.Maintainer != ""),
		Name:           dbp.Name,
		NumVotes:       dbp.NumVotes,
		OutOfDate:      null.NewInt(int64(dbp.OutOfDate), dbp.OutOfDate != 0),
		PackageBase:    dbp.PackageBase,
		Popularity:     dbp.Popularity,
		URL:            null.NewString(dbp.URL, dbp.URL != ""),
		URLPath:        null.NewString(dbp.URLPath, dbp.URLPath != ""),
		Version:        dbp.Version,
	}

	return sr
}

func inSlice(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
