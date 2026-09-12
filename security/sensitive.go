// Package security contains small, reviewed security-policy helpers.
package security

import "strings"

var sensitiveKeys = []string{"password", "passwd", "secret", "token", "authorization", "cookie", "api_key", "database_url", "dsn", "private_key"}

// IsSensitiveKey reports whether a structured field name conventionally holds
// credentials or secret material.
func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), ".", "_"))
	for _, sensitive := range sensitiveKeys {
		if normalized == sensitive || strings.HasSuffix(normalized, "_"+sensitive) {
			return true
		}
	}
	return false
}
