package rpc

import (
	"errors"
	"net/url"
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
	if !sliceContains(queryTypes, values.Get("type")) {
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
		for _, v := range values["arg[]"] {
			args = append(args, v)
		}
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

func sliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func mapContains(s map[string]string, e string) bool {
	for a, _ := range s {
		if a == e {
			return true
		}
	}
	return false
}
