package doc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

var testUrls = []string{
	"/rpc/swagger",
	"/rpc/swagger/",
	"/admin/swagger",
	"/admin/swagger/",
	"/nonsense/swagger/",
	"/rpc/swagger/openapi.json",
	"/admin/swagger/openapi.json",
	"/rpc/olddoc.html",
}

func TestSwagger(t *testing.T) {

	r := mux.NewRouter()

	r.HandleFunc("/rpc/swagger", SwaggerRpcHandler)
	r.HandleFunc("/rpc/swagger/", SwaggerRpcHandler)
	r.HandleFunc("/admin/swagger", SwaggerAdminHandler)
	r.HandleFunc("/admin/swagger/", SwaggerAdminHandler)
	r.HandleFunc("/rpc/swagger/openapi.json", SpecRpcHandler)
	r.HandleFunc("/admin/swagger/openapi.json", SpecAdminHandler)
	r.HandleFunc("/rpc/olddoc.html", SpecOldHandler)

	for _, url := range testUrls {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", url, nil)
		assert.Nil(t, err, "Could not create GET request")

		r.ServeHTTP(rr, req)

		if strings.Contains(url, "nonsense") {
			assert.Equal(t, http.StatusNotFound, rr.Result().StatusCode, "Wrong http status code", url)
			assert.Equal(t, "404 page not found\n", rr.Body.String(), "wrong body text", url)
			continue
		}
		assert.Equal(t, http.StatusOK, rr.Result().StatusCode, "Wrong http status code", url)
		assert.Greater(t, rr.Body.Len(), 0, "no body returned", url)
	}
}
