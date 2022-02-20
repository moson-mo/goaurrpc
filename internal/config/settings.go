package config

import (
	"encoding/json"
	"io/ioutil"
)

// rpc server settings
type Settings struct {
	AurFileLocation string
	MaxResults      int
	RefreshInterval int // in seconds
	RateLimit       int
	LoadFromFile    bool
}

// default settings for our server
func DefaultSettings() *Settings {
	s := Settings{
		AurFileLocation: "https://aur.archlinux.org/packages-meta-ext-v1.json.gz",
		MaxResults:      5000,
		RefreshInterval: 10 * 60, // refresh every 10 minutes
		RateLimit:       5000,
		LoadFromFile:    false,
	}
	return &s
}

// load settings from a file
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
