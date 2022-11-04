package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	db "github.com/moson-mo/goaurrpc/internal/memdb"
	"github.com/moson-mo/goaurrpc/internal/metrics"
	"gopkg.in/guregu/null.v4"
)

// allowed query types
var queryTypes = []string{
	"info",
	"multiinfo",
	"search",
	"search-info",
	"msearch",
	"suggest",
	"suggest-pkgbase",
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

var ErrCallBack = errors.New("Invalid callback name.")

// Checking the validity of the query parameters
func validateValues(values url.Values, maxStringComp int) error {
	_, hasArg := values["arg"]
	multiArgs, hasArgArr := values["arg[]"]
	v := values.Get("v")
	t := values.Get("type")
	by := values.Get("by")

	if values.Get("v") == "" {
		return errors.New("Please specify an API version.")
	}
	if v != "5" && v != "6" {
		return errors.New("Invalid version specified.")
	}
	if t == "" {
		return errors.New("No request type/data specified.")
	}
	if !inSlice(queryTypes, t) {
		return errors.New("Incorrect request type specified.")
	}
	if !inSlice(queryBy, by) {
		return errors.New("Incorrect by field specified.")
	}
	if !hasArg && !hasArgArr && by != "maintainer" {
		return errors.New("No request type/data specified.")
	}
	if ((hasArg && len(values.Get("arg")) < 2) || (hasArgArr && len(values.Get("arg[]")) < 2)) &&
		strings.HasPrefix(t, "search") &&
		by != "maintainer" {
		return errors.New("Query arg too small.")
	}
	if values.Get("callback") != "" {
		match, _ := regexp.MatchString("^[a-zA-Z0-9()_.]{1,128}$", values.Get("callback"))
		if !match {
			return ErrCallBack
		}
	}
	if strings.Contains(t, "search") &&
		(by == "" || strings.HasPrefix(by, "name")) &&
		len(multiArgs) > maxStringComp {
		return errors.New("Exceeded maximum number of arguments for search query")
	}

	return nil
}

// get a string slice with all arguments that have been passed
func getArgumentList(values url.Values) []string {
	var args []string
	if values.Get("arg") != "" {
		args = append(args, strings.ToLower(values.Get("arg")))
	} else {
		for _, arg := range values["arg[]"] {
			args = append(args, strings.ToLower(arg))
		}
	}
	return args
}

// get a single argument
func getArgument(values url.Values) string {
	if values.Get("arg") != "" {
		return strings.ToLower(values.Get("arg"))
	}
	return strings.ToLower(values.Get("arg[]"))
}

// get search "by" parameter
func getBy(values url.Values) string {
	// if not specified use name and description for search
	by := "name-desc"
	if values.Get("by") != "" {
		by = values.Get("by")
	}

	// if type is msearch we search by maintainer
	if values.Get("type") == "msearch" {
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
		w.Header().Set("Content-Type", "text/javascript")
		w.WriteHeader(code)
		fmt.Fprintf(w, "/**/%s(%s)", callback, string(b))
		return
	}
	w.Header().Set("Content-Type", "application/json")
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
func convDbPkgToInfoRecord(dbp *db.PackageInfo, isV6 bool) InfoRecord {
	ir := InfoRecord{
		Name:           dbp.Name,
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
		Arg:            dbp.Arg,
	}
	/*
		for some reason Keywords and License should be returned
		as empty JSON arrays rather than being omitted
	*/
	if ir.Keywords == nil {
		ir.Keywords = []string{}
	}
	if ir.License == nil {
		ir.License = []string{}
	}

	if !isV6 {
		ir.ID = dbp.ID
		ir.PackageBaseID = dbp.PackageBaseID
		ir.Arg = ""
		ir.Submitter = ""
		ir.CoMaintainers = nil
	}

	return ir
}

// converts db.PackageInfo to rpc.SearchRecord
func convDbPkgToSearchRecord(dbp *db.PackageInfo, isV6 bool) SearchRecord {
	sr := SearchRecord{
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
		Arg:            dbp.Arg,
	}

	if !isV6 {
		sr.ID = dbp.ID
		sr.PackageBaseID = dbp.PackageBaseID
		sr.Arg = ""
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

func UniqueStrings(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		if _, found := keys[entry]; !found {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
