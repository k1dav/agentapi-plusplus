package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// passThrough returns a 200 OK with the literal body "ok" so tests
// can assert whether the middleware forwarded the request.
func passThrough() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestAPIKeyAuth_EmptyKeyIsNoOp(t *testing.T) {
	t.Parallel()
	mw := APIKeyAuth("")
	h := mw(passThrough())

	// No header, but the middleware must still forward because no key
	// is configured. This is the "trusted-localhost default" preserved
	// for backward compatibility.
	req := httptest.NewRequest(http.MethodPost, "/anything", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when no key is configured, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", w.Body.String())
	}
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	t.Parallel()
	const key = "secret-token-xyz"
	mw := APIKeyAuth(key)
	h := mw(passThrough())

	req := httptest.NewRequest(http.MethodPost, "/message", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if got := w.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Bearer") {
		t.Fatalf("expected WWW-Authenticate to start with Bearer, got %q", got)
	}
	var body authChallenge
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("challenge body not JSON: %v", err)
	}
	if body.Error != "unauthorized" {
		t.Fatalf("expected error=unauthorized, got %q", body.Error)
	}
}

func TestAPIKeyAuth_WrongScheme(t *testing.T) {
	t.Parallel()
	const key = "secret-token-xyz"
	mw := APIKeyAuth(key)
	h := mw(passThrough())

	req := httptest.NewRequest(http.MethodPost, "/message", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for Basic auth, got %d", w.Code)
	}
}

func TestAPIKeyAuth_WrongKey(t *testing.T) {
	t.Parallel()
	const key = "secret-token-xyz"
	mw := APIKeyAuth(key)
	h := mw(passThrough())

	req := httptest.NewRequest(http.MethodPost, "/message", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong key, got %d", w.Code)
	}
}

func TestAPIKeyAuth_WrongKeyLength(t *testing.T) {
	t.Parallel()
	// Length-mismatch is a real attack vector: subtle.ConstantTimeCompare
	// returns 0 immediately on length mismatch, which leaks length. Our
	// wrapper gates on length first, but that gate is itself a length
	// leak. The important property is that we NEVER return 200 in this
	// case (no shortcut path that skips the compare entirely).
	const key = "secret-token-xyz"
	mw := APIKeyAuth(key)
	h := mw(passThrough())

	req := httptest.NewRequest(http.MethodPost, "/message", nil)
	req.Header.Set("Authorization", "Bearer abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for shorter key, got %d", w.Code)
	}
}

func TestAPIKeyAuth_CorrectKey(t *testing.T) {
	t.Parallel()
	const key = "secret-token-xyz"
	mw := APIKeyAuth(key)
	h := mw(passThrough())

	req := httptest.NewRequest(http.MethodPost, "/message", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct key, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", w.Body.String())
	}
}

func TestAPIKeyAuth_PrefixOnlyHeader(t *testing.T) {
	t.Parallel()
	// "Bearer " with no token must NOT be treated as a valid token.
	const key = "secret-token-xyz"
	mw := APIKeyAuth(key)
	h := mw(passThrough())

	req := httptest.NewRequest(http.MethodPost, "/message", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for prefix-only header, got %d", w.Code)
	}
}

func TestConstantTimeEqual(t *testing.T) {
	t.Parallel()
	// Use a fixed expected length of 16 (the length of `secret-token-xyz`).
	// Each case must produce slices of exactly that length to pass the
	// length gate inside constantTimeEqual.
	const expectedLen = 16
	a := []byte("secret-token-xyz") // 16 bytes
	tests := []struct {
		name     string
		a, b     []byte
		expected bool
	}{
		{"equal", a, a, true},
		{"wrong-bytes-equal-len", []byte("secret-token-abc"), a, false},
		{"a-shorter-rejected", []byte("short"), a, false},
		{"b-shorter-rejected", a, []byte("short"), false},
		{"a-nil-rejected", nil, a, false},
		{"b-nil-rejected", a, nil, false},
		{"same-length-different", []byte("secret-token-ABC"), []byte("secret-token-XYZ"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := constantTimeEqual(tc.a, tc.b, expectedLen)
			if got != tc.expected {
				t.Fatalf("constantTimeEqual(%q,%q,%d) = %v, want %v",
					string(tc.a), string(tc.b), expectedLen, got, tc.expected)
			}
		})
	}
}

// TestConstantTimeEqual_LenGate specifically verifies that slices of
// length 0 (including nil) can NEVER satisfy a non-zero expected
// length. This is the property that prevents the constant-time
// compare from short-circuiting on length and leaking the expected
// length to a timing attacker.
func TestConstantTimeEqual_LenGate(t *testing.T) {
	t.Parallel()
	if constantTimeEqual(nil, nil, 1) {
		t.Fatal("nil/nil must NOT satisfy a non-zero expected length")
	}
	if constantTimeEqual([]byte{}, []byte{}, 1) {
		t.Fatal("empty/empty must NOT satisfy a non-zero expected length")
	}
}
