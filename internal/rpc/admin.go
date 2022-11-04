package rpc

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/moson-mo/goaurrpc/internal/config"
)

// middleware for authentication (API key)
func (s *server) rpcAdminMiddleware(hf http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("APIKey")

		// check api key
		if key != s.settings.AdminAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		hf.ServeHTTP(w, r)
	})
}

// handles jobs
func (s *server) rpcAdminJobsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	switch name {
	case "reload-data":
		err := s.reloadData()
		if err != nil {
			if err.Error() == "not modified" {
				sendAdminOk("Reload skipped. Data has not changed", w)
				return
			}
			sendAdminError(err.Error(), w)
			return
		}
		sendAdminOk("Successfully reloaded data", w)
	case "wipe-cache":
		numEntries := s.wipeSearchCache()
		sendAdminOk("Wiped search-cache ("+strconv.Itoa(numEntries)+" entries removed)", w)
	case "wipe-ratelimits":
		numEntries := s.wipeRateLimits()
		sendAdminOk("Wiped rate-limits ("+strconv.Itoa(numEntries)+" entries removed)", w)
	case "cleanup-cache":
		s.cleanupSearchCache()
		sendAdminOk("Cleaned up search-cache", w)
	case "cleanup-ratelimits":
		s.cleanupRateLimits()
		sendAdminOk("Cleaned up rate-limits", w)
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Job not found"))
	}
}

// handles settings
func (s *server) rpcAdminSettingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := strings.ToLower(vars["name"])
	isPost := r.Method == "POST"
	value := r.URL.Query().Get("value")

	// update and return settings in JSON format
	if strings.TrimRight(r.URL.Path, "/") == "/admin/settings" && !isPost {
		sendSettings(s.settings, w)
		return
	}

	// update / send individual option
	s.sendChangeOption(name, value, isPost, w)
}

// converts query param to int. Don't allow 0
func convValueToInt(value string) (int, error) {
	ival, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	if ival == 0 {
		return 0, errors.New("Value can not be 0")
	}

	return ival, nil
}

// send settings in JSON format
func sendSettings(settings config.Settings, w http.ResponseWriter) {
	b, err := json.MarshalIndent(settings, "", "\t")
	if err != nil {
		sendAdminError(err.Error(), w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *server) sendChangeOption(name, value string, isPost bool, w http.ResponseWriter) {
	s.LogVerbose("Admin initiated change of setting '" + name + "' to '" + value + "'")

	// get/set individual option
	switch name {
	case "aur-file-location":
		pval := s.settings.AurFileLocation
		if isPost {
			if value != "" {
				s.settings.AurFileLocation = value
				sendAdminOk("Changed 'AurFileLocation' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'AurFileLocation' is '"+pval+"'", w)
	case "max-results":
		pval := strconv.Itoa(s.settings.MaxResults)
		if isPost {
			if value != "" {
				ival, err := convValueToInt(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.MaxResults = ival
				sendAdminOk("Changed 'MaxResults' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'MaxResults' is '"+pval+"'", w)
	case "refresh-interval":
		pval := strconv.Itoa(s.settings.RefreshInterval)
		if isPost {
			if value != "" {
				ival, err := convValueToInt(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.RefreshInterval = ival
				sendAdminOk("Changed 'RefreshInterval' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'RefreshInterval' is '"+pval+"'", w)
	case "rate-limit":
		pval := strconv.Itoa(s.settings.RateLimit)
		if isPost {
			if value != "" {
				ival, err := strconv.Atoi(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.RateLimit = ival
				warning := ""
				if ival == 0 {
					warning = "\nWARNING: Rate limit is disabled"
				}
				sendAdminOk("Changed 'RateLimit' from '"+pval+"' to '"+value+"'"+warning, w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'RateLimit' is '"+pval+"'", w)
	case "rate-limit-cleanup-interval":
		pval := strconv.Itoa(s.settings.RateLimitCleanupInterval)
		if isPost {
			if value != "" {
				ival, err := convValueToInt(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.RateLimitCleanupInterval = ival
				sendAdminOk("Changed 'RateLimitCleanupInterval' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'RateLimitCleanupInterval' is '"+pval+"'", w)
	case "rate-limit-time-window":
		pval := strconv.Itoa(s.settings.RateLimitTimeWindow)
		if isPost {
			if value != "" {
				ival, err := convValueToInt(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.RateLimitTimeWindow = ival
				sendAdminOk("Changed 'RateLimitTimeWindow' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'RateLimitTimeWindow' is '"+pval+"'", w)
	case "cache-cleanup-interval":
		pval := strconv.Itoa(s.settings.CacheCleanupInterval)
		if isPost {
			if value != "" {
				ival, err := convValueToInt(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.CacheCleanupInterval = ival
				sendAdminOk("Changed 'CacheCleanupInterval' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'CacheCleanupInterval' is '"+pval+"'", w)
	case "cache-expiration-time":
		pval := strconv.Itoa(s.settings.CacheExpirationTime)
		if isPost {
			if value != "" {
				ival, err := convValueToInt(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.CacheExpirationTime = ival
				sendAdminOk("Changed 'CacheExpirationTime' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'CacheExpirationTime' is '"+pval+"'", w)
	case "enable-search-cache":
		pval := strconv.FormatBool(s.settings.EnableSearchCache)
		if isPost {
			if value != "" {
				bval, err := strconv.ParseBool(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.EnableSearchCache = bval
				sendAdminOk("Changed 'EnableSearchCache' from '"+pval+"' to '"+strconv.FormatBool(s.settings.EnableSearchCache)+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'EnableSearchCache' is '"+pval+"'", w)
	case "max-args-string-comparison":
		pval := strconv.Itoa(s.settings.MaxArgsStringComparison)
		if isPost {
			if value != "" {
				ival, err := convValueToInt(value)
				if err != nil {
					sendAdminError(err.Error(), w)
					return
				}
				s.settings.MaxArgsStringComparison = ival
				sendAdminOk("Changed 'MaxArgsStringComparison' from '"+pval+"' to '"+value+"'", w)
			} else {
				sendAdminError("Need new value: ?value=...", w)
			}
			return
		}
		sendAdminOk("Current setting for 'MaxArgsStringComparison' is '"+pval+"'", w)
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Setting not found"))
	}
}

// returns error result
func sendAdminError(message string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(message))
}

// returns OK result
func sendAdminOk(message string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(message))
}
