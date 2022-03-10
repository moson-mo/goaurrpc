package config

import (
	"encoding/json"
	"io/ioutil"
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
	}
	return &s
}

// LoadFromFile load settings from a file
func LoadFromFile(path string) (*Settings, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	err = json.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
