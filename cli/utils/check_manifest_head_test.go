package utils

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/scheme/reg"
	"github.com/regclient/regclient/types/errs"
	"github.com/regclient/regclient/types/ref"
)

// newCheckManifestHeadTestClient builds a regclient pointed at the given
// httptest.Server over plain HTTP (no TLS, no credentials, no real registry
// involved), with retries disabled so a stubbed status code is returned to
// CheckManifestHead after exactly one HTTP round trip instead of the several
// seconds of exponential backoff regclient defaults to for 429/5xx.
func newCheckManifestHeadTestClient(t *testing.T, serverURL string) (*regclient.RegClient, ref.Ref) {
	t.Helper()

	hostname := strings.TrimPrefix(serverURL, "http://")
	host := config.Host{
		Name:     hostname,
		Hostname: hostname,
		TLS:      config.TLSDisabled,
	}
	client := regclient.New(
		regclient.WithConfigHost(host),
		regclient.WithRegOpts(
			reg.WithRetryLimit(1),
			reg.WithDelay(time.Millisecond, time.Millisecond),
		),
	)

	imageName := fmt.Sprintf("%s/check-manifest-head-test:abc123", hostname)
	r, err := ref.New(imageName)
	if err != nil {
		t.Fatalf("ref.New(%q) failed: %s", imageName, err)
	}

	return client, r
}

// TestCheckManifestHead drives CheckManifestHead against a stub httptest.Server
// for each of the status codes a registry can plausibly return, to confirm
// they are classified correctly with errors.Is against regclient/types/errs.
// None of these need a real registry or credentials, and none of them should
// sleep for regclient's default retry backoff (see
// newCheckManifestHeadTestClient), so this stays fast enough for the default
// unit-test pass.
func TestCheckManifestHead(t *testing.T) {
	t.Run("404 not found is not an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client, r := newCheckManifestHeadTestClient(t, server.URL)
		exists, err := CheckManifestHead("abc123", r, client)

		if err != nil {
			t.Errorf("expected a nil error for a 404, got: %s", err)
		}
		if exists {
			t.Errorf("expected exists to be false for a 404")
		}
	})

	t.Run("401 unauthorized is classified and returned as an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// An auth-type the client has no handler for (only "basic",
			// "bearer" and "jwt" are supported) forces the auth challenge to
			// be rejected as unauthorized rather than retried, which is what
			// surfaces errs.ErrHTTPUnauthorized here rather than a lower
			// level challenge-parsing error.
			w.Header().Set("WWW-Authenticate", `Unsupported realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		client, r := newCheckManifestHeadTestClient(t, server.URL)
		exists, err := CheckManifestHead("abc123", r, client)

		if exists {
			t.Errorf("expected exists to be false for a 401")
		}
		if err == nil {
			t.Fatal("expected a non-nil error for a 401")
		}
		if !errors.Is(err, errs.ErrHTTPUnauthorized) {
			t.Errorf("expected errors.Is(err, errs.ErrHTTPUnauthorized), got: %s", err)
		}
		if errors.Is(err, errs.ErrNotFound) {
			t.Errorf("a 401 must not be classified as errs.ErrNotFound: %s", err)
		}
	})

	t.Run("429 rate limit is classified and returned as an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		client, r := newCheckManifestHeadTestClient(t, server.URL)
		exists, err := CheckManifestHead("abc123", r, client)

		if exists {
			t.Errorf("expected exists to be false for a 429")
		}
		if err == nil {
			t.Fatal("expected a non-nil error for a 429")
		}
		if !errors.Is(err, errs.ErrHTTPRateLimit) {
			t.Errorf("expected errors.Is(err, errs.ErrHTTPRateLimit), got: %s", err)
		}
		if errors.Is(err, errs.ErrNotFound) {
			t.Errorf("a 429 must not be classified as errs.ErrNotFound: %s", err)
		}
	})

	t.Run("500 internal server error falls back to the raw error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, r := newCheckManifestHeadTestClient(t, server.URL)
		exists, err := CheckManifestHead("abc123", r, client)

		if exists {
			t.Errorf("expected exists to be false for a 500")
		}
		if err == nil {
			t.Fatal("expected a non-nil error for a 500")
		}
		if errors.Is(err, errs.ErrNotFound) {
			t.Errorf("a 500 must not be classified as errs.ErrNotFound: %s", err)
		}
		if errors.Is(err, errs.ErrHTTPUnauthorized) {
			t.Errorf("a 500 must not be classified as errs.ErrHTTPUnauthorized: %s", err)
		}
		if errors.Is(err, errs.ErrHTTPRateLimit) {
			t.Errorf("a 500 must not be classified as errs.ErrHTTPRateLimit: %s", err)
		}
		// Falls into the "anything else" bucket: the underlying transport
		// error is still returned so the caller has the raw detail, but it's
		// only a generic errs.ErrHTTPStatus, not one of the specific
		// sentinels handled above.
		if !errors.Is(err, errs.ErrHTTPStatus) {
			t.Errorf("expected errors.Is(err, errs.ErrHTTPStatus), got: %s", err)
		}
	})
}
