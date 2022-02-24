package memdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDbFromFile(t *testing.T) {
	files := []string{
		"../../test_data/test_packages.json.gz",
		"../../test_data/test_packages.json",
	}

	for _, file := range files {
		db, err := LoadDbFromFile(file)
		assert.Nil(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, 666, len(db.PackageNames), "Number of packages don't match")
	}

	brokenFiles := []string{
		"nonsense.gz",
		"nonsense.json",
		"../../test_data/test_packages_broken.json.gz",
		"../../test_data/test_packages_broken.json",
	}

	for _, file := range brokenFiles {
		db, err := LoadDbFromFile(file)
		assert.NotNil(t, err)
		assert.Nil(t, db)
	}
}

func TestLoadDbFromUrl(t *testing.T) {
	urls := []string{"https://github.com/moson-mo/goaurrpc/raw/main/test_data/test_packages.json"}

	for _, url := range urls {
		db, err := LoadDbFromUrl(url)
		assert.Nil(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, 666, len(db.PackageNames), "Number of packages don't match")
	}

	brokenUrls := []string{"https://sdfsdfhahdfagdfgdgdfgdg.agag/raw/main/test_data/test_packages.json"}

	for _, url := range brokenUrls {
		db, err := LoadDbFromUrl(url)
		assert.NotNil(t, err)
		assert.Nil(t, db)
	}

}
