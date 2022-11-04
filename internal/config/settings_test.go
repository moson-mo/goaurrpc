package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromFile(t *testing.T) {
	s, err := LoadFromFile("../../sample.conf")
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, DefaultSettings(), s, "Sample does not equal defaults")

	s, err = LoadFromFile("../../test_data/test_broken.conf")
	assert.NotNil(t, err)
	assert.Nil(t, s)

	s, err = LoadFromFile("../../test_data/doesnotexist")
	assert.NotNil(t, err)
	assert.Nil(t, s)
}

func TestValidateSettings(t *testing.T) {
	// ok
	err := validateSettings(*DefaultSettings())
	assert.Nil(t, err)

	// errors
	s := Settings{}
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.Port = 1
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.MaxResults = 1
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.RefreshInterval = 1
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.RateLimitCleanupInterval = 1
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.RateLimitTimeWindow = 1
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.CacheCleanupInterval = 1
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.CacheExpirationTime = 1
	err = validateSettings(s)
	assert.NotNil(t, err)

	s.MaxArgsStringComparison = 1
	err = validateSettings(s)
	assert.Nil(t, err)
}
