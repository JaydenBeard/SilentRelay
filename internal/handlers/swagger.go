package handlers

import (
	_ "embed"
	"io"
	"log"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec string

// SwaggerUIHTML returns the HTML page for Swagger UI
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SilentRelay API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css">
    <link rel="icon" type="image/png" href="https://silentrelay.com.au/favicon.ico">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info .title { color: #3b4151; }
        .swagger-ui .info .title small.version-stamp { background-color: #7d8492; }
        .swagger-ui .scheme-container { background: #fff; box-shadow: 0 1px 2px 0 rgba(0,0,0,.15); }
        /* Custom header */
        .custom-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px 40px;
            display: flex;
            align-items: center;
            gap: 20px;
        }
        .custom-header h1 { margin: 0; font-size: 24px; font-weight: 600; }
        .custom-header p { margin: 5px 0 0 0; opacity: 0.9; font-size: 14px; }
        .custom-header .logo { font-size: 32px; }
    </style>
</head>
<body>
    <div class="custom-header">
        <div class="logo">🔐</div>
        <div>
            <h1>SilentRelay API</h1>
            <p>Secure End-to-End Encrypted Messaging</p>
        </div>
    </div>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/api/docs/openapi.yaml",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                persistAuthorization: true,
                displayRequestDuration: true,
                filter: true,
                syntaxHighlight: {
                    activate: true,
                    theme: "monokai"
                }
            });
        };
    </script>
</body>
</html>`

// SwaggerUI serves the Swagger UI interface
func SwaggerUI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, swaggerUIHTML); err != nil {
			log.Printf("Warning: failed to write swagger UI: %v", err)
		}
	}
}

// OpenAPISpec serves the embedded OpenAPI specification
func OpenAPISpec(_ string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, openAPISpec); err != nil {
			log.Printf("Warning: failed to write OpenAPI spec: %v", err)
		}
	}
}
