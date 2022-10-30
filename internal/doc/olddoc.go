package doc

import (
	_ "embed"
	"net/http"
)

//go:embed olddoc.html
var oldDoc []byte

// SpecOldHandler handles calls and returns the old documentation
func SpecOldHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	w.Write(oldDoc)
}
