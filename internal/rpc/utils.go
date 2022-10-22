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
	"depends",
	"makedepends",
	"optdepends",
	"checkdepends",
	"provides",
	"conflicts",
	"replaces",
	"keywords",
	"groups",
}

var ErrCallBack = errors.New("Invalid callback name.")

// Checking the validity of the query parameters
func validateQueryString(values url.Values) error {
	_, hasArg := values["arg"]
	_, hasArgArr := values["arg[]"]

	if values.Get("v") == "" {
		return errors.New("Please specify an API version.")
	}
	if values.Get("v") != "5" {
		return errors.New("Invalid version specified.")
	}
	if values.Get("type") == "" {
		return errors.New("No request type/data specified.")
	}
	if !inSlice(queryTypes, values.Get("type")) {
		return errors.New("Incorrect request type specified.")
	}
	if !inSlice(queryBy, values.Get("by")) {
		return errors.New("Incorrect by field specified.")
	}
	if !hasArg && !hasArgArr && values.Get("by") != "maintainer" {
		return errors.New("No request type/data specified.")
	}
	if ((hasArg && len(values.Get("arg")) < 2) || (hasArgArr && len(values.Get("arg[]")) < 2)) &&
		strings.HasPrefix(values.Get("type"), "search") &&
		values.Get("by") != "maintainer" {
		return errors.New("Query arg too small.")
	}
	if values.Get("callback") != "" {
		match, _ := regexp.MatchString("^[a-zA-Z0-9()_.]{1,128}$", values.Get("callback"))
		if !match {
			return ErrCallBack
		}
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
// we could directly pass PackageInfo to the client as well
// but we want to keep things flexible in case of changes
func convDbPkgToInfoRecord(dbp *db.PackageInfo) InfoRecord {
	ir := InfoRecord{
		ID:             dbp.ID,
		Name:           dbp.Name,
		PackageBaseID:  dbp.PackageBaseID,
		PackageBase:    dbp.PackageBase,
		Version:        dbp.Version,
		Description:    dbp.Description,
		URL:            dbp.URL,
		NumVotes:       dbp.NumVotes,
		Popularity:     dbp.Popularity,
		OutOfDate:      dbp.OutOfDate,
		Maintainer:     dbp.Maintainer,
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

	return ir
}

// converts db.PackageInfo to rpc.SearchRecord
func convDbPkgToSearchRecord(dbp *db.PackageInfo) SearchRecord {
	sr := SearchRecord{
		ID:             dbp.ID,
		PackageBaseID:  dbp.PackageBaseID,
		Description:    dbp.Description,
		FirstSubmitted: dbp.FirstSubmitted,
		LastModified:   dbp.LastModified,
		Maintainer:     dbp.Maintainer,
		Name:           dbp.Name,
		NumVotes:       dbp.NumVotes,
		OutOfDate:      dbp.OutOfDate,
		PackageBase:    dbp.PackageBase,
		Popularity:     dbp.Popularity,
		URL:            dbp.URL,
		URLPath:        dbp.URLPath,
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
