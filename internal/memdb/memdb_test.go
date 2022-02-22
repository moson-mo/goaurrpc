package memdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDbFromFile(t *testing.T) {
	files := []string{"../../test_data/test_packages.json.gz", "../../test_data/test_packages.json"}

	for _, file := range files {
		db, err := LoadDbFromFile(file)
		assert.Nil(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, 666, len(db.PackageNames), "Number of packages don't match")
	}
}
