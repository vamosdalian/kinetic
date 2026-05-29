package clusterauth

import (
	"errors"
	"testing"
	"time"
)

func verifyHeaders(secret string, headers map[string]string, method, path string, body []byte, now time.Time) error {
	return Verify(secret, headers[HeaderTimestamp], headers[HeaderSignature], method, path, body, now, DefaultTolerance)
}

func TestVerify_RoundTrip(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	body := []byte(`{"hello":"world"}`)
	headers := Headers("top-secret", "POST", "/api/internal/nodes/n1/task-events", body, now)

	if err := verifyHeaders("top-secret", headers, "POST", "/api/internal/nodes/n1/task-events", body, now); err != nil {
		t.Fatalf("expected valid signature, got %v", err)
	}
}

func TestVerify_EmptySecretRoundTrip(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	headers := Headers("", "GET", "/api/internal/nodes/n1/stream", nil, now)

	if err := verifyHeaders("", headers, "GET", "/api/internal/nodes/n1/stream", nil, now); err != nil {
		t.Fatalf("empty secret should still verify uniformly, got %v", err)
	}
}

func TestVerify_MissingHeaders(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	if err := Verify("s", "", "", "GET", "/p", nil, now, DefaultTolerance); !errors.Is(err, ErrMissingSignature) {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}

func TestVerify_TimestampSkew(t *testing.T) {
	signedAt := time.Unix(1_700_000_000, 0)
	headers := Headers("s", "GET", "/p", nil, signedAt)
	tooLate := signedAt.Add(DefaultTolerance + time.Minute)

	if err := verifyHeaders("s", headers, "GET", "/p", nil, tooLate); !errors.Is(err, ErrTimestampSkew) {
		t.Fatalf("expected ErrTimestampSkew, got %v", err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	headers := Headers("correct", "POST", "/p", []byte("x"), now)

	if err := verifyHeaders("wrong", headers, "POST", "/p", []byte("x"), now); !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got %v", err)
	}
}

func TestVerify_TamperedRequest(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	headers := Headers("s", "POST", "/api/internal/nodes/n1/task-events", []byte(`{"type":"finished"}`), now)

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{"body", "POST", "/api/internal/nodes/n1/task-events", []byte(`{"type":"failed"}`)},
		{"path", "POST", "/api/internal/nodes/n2/task-events", []byte(`{"type":"finished"}`)},
		{"method", "GET", "/api/internal/nodes/n1/task-events", []byte(`{"type":"finished"}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := verifyHeaders("s", headers, tc.method, tc.path, tc.body, now); !errors.Is(err, ErrSignatureMismatch) {
				t.Fatalf("tampered %s should fail, got %v", tc.name, err)
			}
		})
	}
}
