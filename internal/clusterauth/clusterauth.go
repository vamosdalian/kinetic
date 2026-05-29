// Package clusterauth implements the shared-secret HMAC scheme used to
// authenticate worker -> controller traffic on the /api/internal/* endpoints.
//
// The same secret (config field "cluster_secret", env KINETIC_CLUSTER_SECRET)
// is configured on both the controller and every worker. Requests are signed
// over their timestamp, method, path and body, so the raw secret never travels
// on the wire and captured signatures cannot be reused for a different request.
//
// The verification path is unconditional: an empty secret is a valid (but
// public) key, meaning anyone who implements this scheme can talk to an
// unconfigured controller. Setting a non-empty secret on both sides is what
// turns the internal API into an authenticated channel.
package clusterauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	// HeaderTimestamp carries the unix-second timestamp the request was signed at.
	HeaderTimestamp = "X-Kinetic-Timestamp"
	// HeaderSignature carries the base64 (raw URL) HMAC-SHA256 signature.
	HeaderSignature = "X-Kinetic-Signature"
	// DefaultTolerance is the allowed clock skew between worker and controller.
	DefaultTolerance = 5 * time.Minute
)

var (
	// ErrMissingSignature is returned when the signing headers are absent.
	ErrMissingSignature = errors.New("clusterauth: missing signature headers")
	// ErrInvalidTimestamp is returned when the timestamp header is malformed.
	ErrInvalidTimestamp = errors.New("clusterauth: invalid timestamp")
	// ErrTimestampSkew is returned when the timestamp is outside the tolerance window.
	ErrTimestampSkew = errors.New("clusterauth: timestamp outside tolerance")
	// ErrSignatureMismatch is returned when the signature does not match.
	ErrSignatureMismatch = errors.New("clusterauth: signature mismatch")
)

func canonical(timestamp, method, path string, body []byte) string {
	sum := sha256.Sum256(body)
	return strings.Join([]string{
		timestamp,
		strings.ToUpper(method),
		path,
		hex.EncodeToString(sum[:]),
	}, "\n")
}

func sign(secret []byte, timestamp, method, path string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(canonical(timestamp, method, path, body)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Headers returns the signing headers a client must attach to a request.
func Headers(secret, method, path string, body []byte, now time.Time) map[string]string {
	ts := strconv.FormatInt(now.Unix(), 10)
	return map[string]string{
		HeaderTimestamp: ts,
		HeaderSignature: sign([]byte(secret), ts, method, path, body),
	}
}

// Verify checks the signature of an incoming request. It always runs, even when
// the secret is empty, so the code path is uniform; an empty secret simply means
// the signature is computable by anyone.
func Verify(secret, timestamp, signature, method, path string, body []byte, now time.Time, tolerance time.Duration) error {
	if timestamp == "" || signature == "" {
		return ErrMissingSignature
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrInvalidTimestamp
	}
	if tolerance <= 0 {
		tolerance = DefaultTolerance
	}
	delta := now.Unix() - ts
	if delta < 0 {
		delta = -delta
	}
	if time.Duration(delta)*time.Second > tolerance {
		return ErrTimestampSkew
	}
	expected := sign([]byte(secret), timestamp, method, path, body)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return ErrSignatureMismatch
	}
	return nil
}
