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
}
