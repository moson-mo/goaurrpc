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

	s, err = LoadFromFile("../../test_data/test_errors.conf")
	assert.NotNil(t, err)
	assert.Nil(t, s)
}
