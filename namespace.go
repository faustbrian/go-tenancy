package tenancy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

const (
	minimumNamespaceKeyBytes = 32
	maximumNamespaceKeyBytes = 1024
	maximumNamespacePart     = 4096
	namespaceVersion         = "tenancy-namespace-v2"
)

var (
	// ErrInvalidNamespaceKey reports a key outside the 32..1024-byte length bounds.
	ErrInvalidNamespaceKey = errors.New("tenancy: invalid namespace key")
	// ErrInvalidNamespaceInput reports invalid scope, domain, or logical key.
	ErrInvalidNamespaceInput = errors.New("tenancy: invalid namespace input")
)

// NamespaceDomain separates independently owned isolation key spaces.
type NamespaceDomain string

const (
	// NamespaceCache isolates cache entries.
	NamespaceCache NamespaceDomain = "cache"
	// NamespaceIdempotency isolates idempotency records.
	NamespaceIdempotency NamespaceDomain = "idempotency"
	// NamespaceRateLimit isolates rate-limit counters.
	NamespaceRateLimit NamespaceDomain = "rate-limit"
	// NamespaceSearch isolates search indexes and documents.
	NamespaceSearch NamespaceDomain = "search"
	// NamespaceQueue isolates queue routing and deduplication keys.
	NamespaceQueue NamespaceDomain = "queue"
	// NamespaceScheduler isolates scheduled work.
	NamespaceScheduler NamespaceDomain = "scheduler"
	// NamespaceEvent isolates event streams and event identities.
	NamespaceEvent NamespaceDomain = "event"
	// NamespaceWorkflow isolates workflow executions.
	NamespaceWorkflow NamespaceDomain = "workflow"
	// NamespaceTelemetry produces opaque telemetry attributes.
	NamespaceTelemetry NamespaceDomain = "telemetry"
)

// NamespaceEncoder creates opaque, collision-resistant, versioned keys. It
// owns a copy of its HMAC-SHA-256 key and is safe for concurrent use. It retains
// that key for its lifetime and offers no key erasure or built-in rotation.
// Opaque deterministic names are not encrypted records, authorization tokens
// or an enforcement boundary for separately held provider clients. The zero
// value cannot encode; construct an encoder with NewNamespaceEncoder.
type NamespaceEncoder struct {
	key []byte
}

// NewNamespaceEncoder validates length (32..1024 bytes) and copies the key.
// It does not assess entropy or generate randomness; applications must supply
// securely generated secret key material and own rotation/dual-read migration.
// The input must remain unchanged during construction.
func NewNamespaceEncoder(key []byte) (*NamespaceEncoder, error) {
	if len(key) < minimumNamespaceKeyBytes || len(key) > maximumNamespaceKeyBytes {
		return nil, ErrInvalidNamespaceKey
	}
	return &NamespaceEncoder{key: append([]byte(nil), key...)}, nil
}

// Encode composes scope, domain, and key with length-delimited HMAC input. Raw
// tenant IDs and logical keys are not embedded as raw fields in the returned name.
// The HMAC input is length-prefixed version tenancy-namespace-v2, one scope-kind
// byte, and length-prefixed raw tenant ID, domain and logical key; lengths are
// four-byte big-endian. Metadata and administrative reasons do not affect names.
// Logical keys may contain arbitrary bytes but must be 1..4096 bytes. Only the
// exported domains are accepted. Output is tn2_ plus 64 lowercase hex digits.
// Repeated inputs produce repeatable names, allowing equality correlation.
// Secrecy requires key custody, and collision resistance is not a mathematical
// uniqueness guarantee.
func (encoder *NamespaceEncoder) Encode(scope Scope, domain NamespaceDomain, key string) (string, error) {
	if encoder == nil || len(encoder.key) < minimumNamespaceKeyBytes ||
		!scope.Valid() || !domain.valid() || len(key) == 0 || len(key) > maximumNamespacePart {
		return "", ErrInvalidNamespaceInput
	}
	digest := hmac.New(sha256.New, encoder.key)
	writeNamespacePart(digest, namespaceVersion)
	digest.Write([]byte{byte(scope.Kind())})
	writeNamespacePart(digest, scope.TenantID().Value())
	writeNamespacePart(digest, string(domain))
	writeNamespacePart(digest, key)
	return "tn2_" + hex.EncodeToString(digest.Sum(nil)), nil
}

func (domain NamespaceDomain) valid() bool {
	switch domain {
	case NamespaceCache, NamespaceIdempotency, NamespaceRateLimit,
		NamespaceSearch, NamespaceQueue, NamespaceScheduler, NamespaceEvent,
		NamespaceWorkflow, NamespaceTelemetry:
		return true
	default:
		return false
	}
}

type namespaceWriter interface {
	Write([]byte) (int, error)
}

func writeNamespacePart(writer namespaceWriter, value string) {
	var size [4]byte
	// #nosec G115 -- Encode admits only tenant IDs <=128 and parts <=4096 bytes.
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = writer.Write([]byte(value))
}
