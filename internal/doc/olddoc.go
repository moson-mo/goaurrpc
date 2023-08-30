package doc

import (
	_ "embed"
	"net/http"

	"github.com/moson-mo/goaurrpc/internal/consts"
)

//go:embed olddoc.html
var oldDoc []byte

// SpecOldHandler handles calls and returns the old documentation
func SpecOldHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", consts.ContentTypeHtml)
	w.Write(oldDoc)
}
