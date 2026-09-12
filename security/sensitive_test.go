package security

import "testing"

func TestIsSensitiveKey(t *testing.T) {
	for _, key := range []string{"password", "db-password", "oauth.access_token", "API_KEY", "client_secret", "private_key"} {
		if !IsSensitiveKey(key) {
			t.Errorf("did not classify %q", key)
		}
	}
	for _, key := range []string{"component", "actor", "token_count", "key_id"} {
		if IsSensitiveKey(key) {
			t.Errorf("misclassified %q", key)
		}
	}
}
