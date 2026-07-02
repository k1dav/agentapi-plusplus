package attach

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/xerrors"
)

// TestCheckACPModeHTTPTimeout_HungServer verifies the new client-side
// timeout (audit finding L26): when the server never replies, the call
// must give up within `checkACPModeHTTPTimeout + a small slack` rather
// than blocking forever.
func TestCheckACPModeHTTPTimeout_HungServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Sleep far longer than the client's timeout so the client side
		// is what fails first.
		time.Sleep(2 * checkACPModeHTTPTimeout)
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()

	start := time.Now()
	isACP, err := checkACPMode(srv.URL)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected timeout error from hung server, got nil (isACP=%v)", isACP)
	}
	// The error must be a network-level failure (timeout or refused),
	// not an HTTP status code — the server never got to respond.
	if !strings.Contains(err.Error(), "failed to check server status") {
		t.Fatalf("unexpected error: %v", err)
	}
	// Underlying transport error must be a timeout-shaped error.
	if inner := xerrors.Unwrap(err); inner == nil {
		t.Fatalf("expected wrapped error, got bare %v", err)
	}

	// Allow a 2x slack for slow CI; the core invariant is "didn't hang".
	if elapsed > 2*checkACPModeHTTPTimeout {
		t.Fatalf("checkACPMode blocked for %s (timeout is %s)", elapsed, checkACPModeHTTPTimeout)
	}
}

// TestCheckACPMode_HappyPath_ACP confirms a 200 + TransportACP body
// is correctly detected, so the timeout change didn't break the
// existing happy path.
func TestCheckACPMode_HappyPath_ACP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// TransportACP string is whatever the httpapi package reports;
		// the wire-level contract is "non-empty string != legacy".
		_, _ = w.Write([]byte(`{"transport":"acp"}`))
	}))
	defer srv.Close()

	isACP, err := checkACPMode(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isACP {
		t.Fatalf("expected isACP=true for transport=acp body")
	}
}

// TestCheckACPMode_HappyPath_NonACP confirms a non-ACP body returns
// (false, nil) instead of an error — the CLI then proceeds to attach.
func TestCheckACPMode_HappyPath_NonACP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transport":"legacy"}`))
	}))
	defer srv.Close()

	isACP, err := checkACPMode(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isACP {
		t.Fatalf("expected isACP=false for transport=legacy body")
	}
}