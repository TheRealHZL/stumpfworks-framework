package middleware

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CORSOptions defines an explicit browser cross-origin policy. An application
// that does not install this middleware exposes no CORS headers.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// CORS builds strict CORS middleware from an exact origin allowlist.
func CORS(options CORSOptions) (func(http.Handler) http.Handler, error) {
	origins := make(map[string]struct{}, len(options.AllowedOrigins))
	wildcard := false
	for _, origin := range options.AllowedOrigins {
		if origin == "*" {
			wildcard = true
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, errors.New("CORS origins must be exact HTTP(S) origins")
		}
		origins[origin] = struct{}{}
	}
	if wildcard && options.AllowCredentials {
		return nil, errors.New("credentialed CORS cannot use a wildcard origin")
	}
	methods := make(map[string]struct{}, len(options.AllowedMethods))
	methodList := make([]string, 0, len(options.AllowedMethods))
	for _, method := range options.AllowedMethods {
		method = strings.ToUpper(method)
		if !validToken(method) {
			return nil, errors.New("CORS method is invalid")
		}
		methods[method] = struct{}{}
		methodList = append(methodList, method)
	}
	headers := make(map[string]struct{}, len(options.AllowedHeaders))
	for _, header := range options.AllowedHeaders {
		if !validToken(header) {
			return nil, errors.New("CORS header is invalid")
		}
		headers[strings.ToLower(header)] = struct{}{}
	}
	for _, header := range options.ExposedHeaders {
		if !validToken(header) {
			return nil, errors.New("CORS exposed header is invalid")
		}
	}
	if options.MaxAge < 0 || options.MaxAge > 24*time.Hour {
		return nil, errors.New("CORS max age must be between zero and 24 hours")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			_, allowed := origins[origin]
			allowed = allowed || wildcard
			w.Header().Add("Vary", "Origin")
			if !allowed {
				if isPreflight(r) {
					w.Header().Set("Cache-Control", "no-store")
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			allowOrigin := origin
			if wildcard {
				allowOrigin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			if options.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if !isPreflight(r) {
				if len(options.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(options.ExposedHeaders, ", "))
				}
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			requestedMethod := strings.ToUpper(r.Header.Get("Access-Control-Request-Method"))
			if _, ok := methods[requestedMethod]; !ok {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
				header = strings.TrimSpace(strings.ToLower(header))
				if header == "" {
					continue
				}
				if _, ok := headers[header]; !ok {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methodList, ", "))
			if len(options.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(options.AllowedHeaders, ", "))
			}
			if options.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.FormatInt(int64(options.MaxAge/time.Second), 10))
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}, nil
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}

func validToken(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z') && !(character >= 'A' && character <= 'Z') && !(character >= '0' && character <= '9') && !strings.ContainsRune("!#$%&'*+-.^_`|~", character) {
			return false
		}
	}
	return true
}
