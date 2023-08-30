package doc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/moson-mo/goaurrpc/internal/consts"
	"github.com/stretchr/testify/assert"
)

var testUrls = map[string]string{
	"/rpc/swagger":                consts.ContentTypeHtml,
	"/rpc/swagger/":               consts.ContentTypeHtml,
	"/api/swagger":                consts.ContentTypeHtml,
	"/api/swagger/":               consts.ContentTypeHtml,
	"/admin/swagger":              consts.ContentTypeHtml,
	"/admin/swagger/":             consts.ContentTypeHtml,
	"/nonsense/swagger/":          consts.ContentTypeText,
	"/rpc/swagger/openapi.json":   consts.ContentTypeJson,
	"/api/swagger/openapi.json":   consts.ContentTypeJson,
	"/admin/swagger/openapi.json": consts.ContentTypeJson,
	"/rpc/olddoc.html":            consts.ContentTypeHtml,
}

func TestSwagger(t *testing.T) {

	r := chi.NewRouter()

	r.HandleFunc("/rpc/swagger", SwaggerRpcHandler)
	r.HandleFunc("/rpc/swagger/", SwaggerRpcHandler)
	r.HandleFunc("/api/swagger", SwaggerApiHandler)
	r.HandleFunc("/api/swagger/", SwaggerApiHandler)
	r.HandleFunc("/admin/swagger", SwaggerAdminHandler)
	r.HandleFunc("/admin/swagger/", SwaggerAdminHandler)
	r.HandleFunc("/rpc/swagger/openapi.json", SpecRpcHandler)
	r.HandleFunc("/api/swagger/openapi.json", SpecApiHandler)
	r.HandleFunc("/admin/swagger/openapi.json", SpecAdminHandler)
	r.HandleFunc("/rpc/olddoc.html", SpecOldHandler)

	for url, contentType := range testUrls {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", url, nil)
		assert.Nil(t, err, "Could not create GET request")

		r.ServeHTTP(rr, req)

		assert.Equal(t, contentType, rr.Result().Header.Get("Content-Type"), "Wrong Content-Type header")
		if strings.Contains(url, "nonsense") {
			assert.Equal(t, http.StatusNotFound, rr.Result().StatusCode, "Wrong http status code", url)
			assert.Equal(t, "404 page not found\n", rr.Body.String(), "wrong body text", url)
			continue
		}
		assert.Equal(t, http.StatusOK, rr.Result().StatusCode, "Wrong http status code", url)
		assert.Greater(t, rr.Body.Len(), 0, "no body returned", url)
	}
}
