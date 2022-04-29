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

	"gopkg.in/guregu/null.v4"
)

// The allowed query types
var queryTypes = []string{
	"info",
	"multiinfo",
	"search",
	"msearch",
	"suggest",
	"suggest-pkgbase",
}

var CallBackError = errors.New("Invalid callback name.")

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
			return CallBackError
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
}

// generate JSON string from RpcResult and return to client
func writeResult(result *RpcResult, callback string, w http.ResponseWriter) {
	// set number of records
	if result.Resultcount == 0 {
		result.Results = make([]interface{}, 0)
	}
	b, _ := json.Marshal(result)
	sendResult(200, callback, b, w)
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
	ipp := r.Header.Get("X-Real-IP")
	if ipp != "" && inSlice(trustedProxies, ip) {
		ip = ipp
	}
	return ip
}

func inSlice(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func sliceContainsBeginsWith(s []string, e string) bool {
	for _, a := range s {
		if strings.HasPrefix(a, e) {
			return true
		}
	}
	return false
}
