```go
package api

import (
	"net/http"
)

// CORSConfig holds the configuration for CORS
type CORSConfig struct {
	AllowedOrigins []string
}

// CORS middleware to handle CORS requests
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == origin {
			return true
		}
	}
	return false
}
```

!!internal/api/cors_test.go!!
```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	allowedOrigins := []string{"http://example.com", "http://anotherdomain.com"}
	cors := CORS(CORSConfig{AllowedOrigins: allowedOrigins})

	tests := []struct {
		origin       string
		expectedCode int
	}{
		{"http://example.com", http.StatusOK},
		{"http://anotherdomain.com", http.StatusOK},
		{"http://notallowed.com", http.StatusOK}, // Should still work but not set the header
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", "http://localhost", nil)
		req.Header.Set("Origin", test.origin)
		rec := httptest.NewRecorder()

		cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)

		if rec.Code != test.expectedCode {
			t.Errorf("Expected status %d but got %d for origin %s", test.expectedCode, rec.Code, test.origin)
		}

		if test.origin == "http://example.com" || test.origin == "http://anotherdomain.com" {
			if rec.Header().Get("Access-Control-Allow-Origin") != test.origin {
				t.Errorf("Expected Access-Control-Allow-Origin to be %s but got %s", test.origin, rec.Header().Get("Access-Control-Allow-Origin"))
			}
		} else {
			if rec.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Errorf("Expected no Access-Control-Allow-Origin header for not allowed origin but got %s", rec.Header().Get("Access-Control-Allow-Origin"))
			}
		}
	}
}
```