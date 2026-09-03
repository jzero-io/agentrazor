package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestAPIKeyFromRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(APIKeyHeader, " ar-example ")

	key, ok := APIKeyFromRequest(r)
	if !ok || key != "ar-example" {
		t.Fatalf("APIKeyFromRequest() = %q, %v", key, ok)
	}
}

func TestAPIKeyFromRequestDoesNotUseAuthorization(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer ar-example")

	if key, ok := APIKeyFromRequest(r); ok {
		t.Fatalf("APIKeyFromRequest() unexpectedly accepted Authorization value %q", key)
	}
}
