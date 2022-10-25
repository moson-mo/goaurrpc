package memdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadDbFromFile(t *testing.T) {
	files := []string{
		"../../test_data/test_packages.json.gz",
		"../../test_data/test_packages.json",
	}

	for _, file := range files {
		db, _, err := LoadDbFromFile(file, time.Time{})
		assert.Nil(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, 666, len(db.PackageNames), "Number of packages don't match")
	}

	// modified test
	_, mod, _ := LoadDbFromFile(files[0], time.Time{})
	assert.NotEqual(t, mod, time.Time{}, "Modified date should be different")
	_, nmod, err := LoadDbFromFile(files[0], mod)
	assert.NotNil(t, err)
	assert.Equal(t, nmod, mod)

	brokenFiles := []string{
		"nonsense.gz",
		"nonsense.json",
		"../../test_data/test_packages_broken.json.gz",
		"../../test_data/test_packages_broken.json",
	}

	for _, file := range brokenFiles {
		db, _, err := LoadDbFromFile(file, time.Time{})
		assert.NotNil(t, err)
		assert.Nil(t, db)
	}
}

func TestLoadDbFromUrl(t *testing.T) {
	urls := []string{"https://github.com/moson-mo/goaurrpc/raw/main/test_data/test_packages.json"}

	for _, url := range urls {
		db, _, err := LoadDbFromUrl(url, time.Time{})
		assert.Nil(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, 666, len(db.PackageNames), "Number of packages don't match")
	}

	brokenUrls := []string{"https://sdfsdfhahdfagdfgdgdfgdg.agag/raw/main/test_data/test_packages.json"}

	for _, url := range brokenUrls {
		db, _, err := LoadDbFromUrl(url, time.Time{})
		assert.NotNil(t, err)
		assert.Nil(t, db)
	}
}

func TestBytesToMemory(t *testing.T) {
	db, err := bytesToMemoryDB([]byte("nonsense"))
	assert.Nil(t, db)
	assert.NotNil(t, err)

	db, err = bytesToMemoryDB([]byte("[{\"Name\":\"testpkg\"}]"))
	assert.NotNil(t, db)
	assert.Nil(t, err)
}
