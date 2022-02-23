package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	if !hasArg && !hasArgArr {
		return errors.New("No request type/data specified.")
	}
	if (hasArg && len(values.Get("arg")) < 2) || (hasArgArr && len(values.Get("arg[]")) < 2) {
		return errors.New("Query arg too small.")
	}

	return nil
}

// get a string slice with all arguments that have been passed
func getArgumentList(values url.Values) []string {
	var args []string
	if values.Get("arg") != "" {
		args = append(args, values.Get("arg"))
	} else {
		args = append(args, values["arg[]"]...)
	}
	return args
}

// get a single argument
func getArgument(values url.Values) string {
	if values.Get("arg") != "" {
		return values.Get("arg")
	}
	return values.Get("arg[]")
}

// generate JSON error and return to client
func writeError(code int, message string, version int, w http.ResponseWriter) {
	e := RpcResult{
		Error:   message,
		Type:    "error",
		Results: make([]interface{}, 0),
		Version: null.NewInt(int64(version), version != 0),
	}
	b, err := json.Marshal(e)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "This should not happen")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}

// generate JSON string from RpcResult and return to client
func writeResult(result *RpcResult, w http.ResponseWriter) {
	// set number of records
	if result.Resultcount == 0 {
		result.Results = make([]interface{}, 0)
	}
	b, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "This should not happen")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
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
