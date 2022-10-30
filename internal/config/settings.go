package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Settings is a data structure holding our configuration data
type Settings struct {
	Port                     int
	AurFileLocation          string
	MaxResults               int
	RefreshInterval          int // in seconds
	RateLimit                int
	LoadFromFile             bool
	RateLimitCleanupInterval int // in seconds
	RateLimitTimeWindow      int // in seconds
	TrustedReverseProxies    []string
	EnableSSL                bool
	CertFile                 string
	KeyFile                  string
	EnableSearchCache        bool
	CacheCleanupInterval     int // in seconds
	CacheExpirationTime      int // in seconds
	LogFile                  string
	EnableMetrics            bool
	EnableAdminApi           bool
	AdminAPIKey              string
}

// DefaultSettings returns the default settings for our server
func DefaultSettings() *Settings {
	s := Settings{
		Port:                     10666,
		AurFileLocation:          "https://aur.archlinux.org/packages-meta-ext-v1.json.gz",
		MaxResults:               5000,
		RefreshInterval:          5 * 60, // refresh every 5 minutes
		RateLimit:                4000,
		LoadFromFile:             false,
		RateLimitCleanupInterval: 10 * 60,
		RateLimitTimeWindow:      24 * 60 * 60,
		TrustedReverseProxies:    []string{"127.0.0.1", "::1"},
		EnableSearchCache:        true,
		CacheCleanupInterval:     60,
		CacheExpirationTime:      180,
		LogFile:                  "",
		EnableMetrics:            true,
		EnableAdminApi:           false,
		AdminAPIKey:              "change-me",
	}
	return &s
}

// LoadFromFile load settings from a file
func LoadFromFile(path string) (*Settings, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	err = json.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}

	// make sure we got sane config data
	err = validateSettings(s)

	return &s, err
}

// validate config settings
func validateSettings(s Settings) error {
	if s.RateLimit == 0 {
		fmt.Printf("Warning: Rate limiting is disabled - RateLimit = 0\n\n")
	}

	errZero := " needs to be specified / greater than 0"
	switch 0 {
	case s.Port:
		return errors.New("config: Port" + errZero)
	case s.MaxResults:
		return errors.New("config: MaxResults" + errZero)
	case s.RefreshInterval:
		return errors.New("config: RefreshInterval" + errZero)
	case s.RateLimitCleanupInterval:
		return errors.New("config: RateLimitCleanupInterval" + errZero)
	case s.RateLimitTimeWindow:
		return errors.New("config: RateLimitTimeWindow" + errZero)
	case s.CacheCleanupInterval:
		return errors.New("config: CacheCleanupInterval" + errZero)
	case s.CacheExpirationTime:
		return errors.New("config: CacheExpirationTime" + errZero)
	}

	return nil
}
