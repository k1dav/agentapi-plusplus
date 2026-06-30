package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// authChallenge is the JSON body returned for a 401 Unauthorized response.
// Centralized so the wire format is stable for SDK consumers.
type authChallenge struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// APIKeyAuth returns an HTTP middleware that requires a matching
// `Authorization: Bearer <key>` header on every request.
//
// The comparison uses crypto/subtle.ConstantTimeCompare to defeat
// timing-side-channel attacks against the key. An empty expectedKey
// disables the middleware (returns the handler unchanged) so that
// the trusted-localhost default remains backward compatible.
//
// On rejection the middleware writes a 401 Unauthorized with a
// machine-readable JSON body and the canonical `WWW-Authenticate`
// header so HTTP clients and SDKs can recover cleanly.
func APIKeyAuth(expectedKey string) func(next http.Handler) http.Handler {
	// Fast path: nothing configured → return handler untouched.
	if expectedKey == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	// Pre-compute a fixed-size copy of the expected key bytes so the
	// constant-time compare operates on equal-length slices. The
	// length gate inside constantTimeEqual prevents the fast-path
	// 0 return from subtle.ConstantTimeCompare (on length mismatch)
	// from leaking length information to a timing attacker.
	expected := []byte(expectedKey)
	expectedLen := len(expected)
	challengeBody := authChallenge{
		Error:   "unauthorized",
		Message: "Missing or invalid Authorization header. Provide `Authorization: Bearer <api_key>`.",
	}
	challengeJSON, _ := json.Marshal(challengeBody)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			// Header must be of the form "Bearer <token>".
			const prefix = "Bearer "
			if len(authz) <= len(prefix) || authz[:len(prefix)] != prefix {
				writeUnauthorized(w, challengeJSON)
				return
			}
			provided := []byte(authz[len(prefix):])
			if !constantTimeEqual(provided, expected, expectedLen) {
				writeUnauthorized(w, challengeJSON)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HumaAPIKeyAuth returns a huma middleware that enforces API-key auth.
// Use this on huma.Operation.Middlewares for fine-grained per-route
// gating (e.g., only on mutating routes like POST /message, DELETE /messages).
// An empty expectedKey makes this a no-op so the trusted-localhost
// default remains backward compatible.
func HumaAPIKeyAuth(expectedKey string) func(huma.Context, func(huma.Context)) {
	if expectedKey == "" {
		return func(ctx huma.Context, next func(huma.Context)) {
			next(ctx)
		}
	}
	expected := []byte(expectedKey)
	expectedLen := len(expected)
	challengeBody := authChallenge{
		Error:   "unauthorized",
		Message: "Missing or invalid Authorization header. Provide `Authorization: Bearer <api_key>`.",
	}
	challengeJSON, _ := json.Marshal(challengeBody)

	return func(ctx huma.Context, next func(huma.Context)) {
		authz := ctx.Header("Authorization")
		const prefix = "Bearer "
		if len(authz) <= len(prefix) || authz[:len(prefix)] != prefix {
			writeHumaUnauthorized(ctx, challengeJSON)
			return
		}
		provided := []byte(authz[len(prefix):])
		if !constantTimeEqual(provided, expected, expectedLen) {
			writeHumaUnauthorized(ctx, challengeJSON)
			return
		}
		next(ctx)
	}
}

// writeUnauthorized responds with 401 + the canonical challenge body
// on a plain http.ResponseWriter (router-level middleware).
func writeUnauthorized(w http.ResponseWriter, body []byte) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="agentapi"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write(body)
}

// writeHumaUnauthorized responds with 401 + the canonical challenge body
// on a huma.Context (operation-level middleware). Huma's Context interface
// uses SetHeader/SetStatus/BodyWriter instead of the standard library
// ResponseWriter surface, so we adapt to that contract here.
func writeHumaUnauthorized(ctx huma.Context, body []byte) {
	ctx.SetHeader("WWW-Authenticate", `Bearer realm="agentapi"`)
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetStatus(http.StatusUnauthorized)
	_, _ = ctx.BodyWriter().Write(body)
}

// constantTimeEqual compares two byte slices in constant time, gated
// on equal length to the expected buffer. Returns true iff len(a)
// equals expectedLen AND a equals b. The length gate prevents the
// fast-path 0 return from subtle.ConstantTimeCompare (on length
// mismatch) from leaking length information to a timing attacker.
func constantTimeEqual(a, b []byte, expectedLen int) bool {
	if len(a) != expectedLen || len(b) != expectedLen {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
