package doc

import (
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed openapi_rpc.json
var openApiRpc []byte

//go:embed openapi_api.json
var openApiApi []byte

//go:embed openapi_admin.json
var openApiAdmin []byte

// swagger-ui
const swaggerUi = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta
	name="description"
	content="SwaggerUI"
  />
  <title>goaurrpc - SwaggerUI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui-bundle.js" crossorigin></script>
<script>
  window.onload = () => {
	window.ui = SwaggerUIBundle({
	  url: '/%s/openapi.json',
	  dom_id: '#swagger-ui',
	});
  };
</script>
</body>
</html>
`

// SwaggerRpcHandler handles calls to the swagger-ui of /rpc
func SwaggerRpcHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	fmt.Fprintf(w, swaggerUi, "rpc")
}

// SwaggerApiHandler handles calls to the swagger-ui of /api
func SwaggerApiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	fmt.Fprintf(w, swaggerUi, "api")
}

// SwaggerAdminHandler handles calls to the swagger-ui of /admin
func SwaggerAdminHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	fmt.Fprintf(w, swaggerUi, "admin")
}

// SpecRpcHandler handles calls to the openapi spec for /rpc
func SpecRpcHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(openApiRpc)
}

// SpecApiHandler handles calls to the openapi spec for /api
func SpecApiHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(openApiApi)
}

// SpecAdminHandler handles calls to the openapi spec for /admin
func SpecAdminHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(openApiAdmin)
}
