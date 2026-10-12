// Package tenancyjsonrpc propagates tenant scope through a bounded JSON-RPC
// metadata object. It rejects duplicate top-level JSON members before map
// decoding can hide ambiguity and requires an explicit immediate-peer trust
// decision. Nested non-tenant JSON is not recursively duplicate-checked.
// Use New to bind the field, byte limit and trust callback, then Extract, Accept
// or Inject. It does not implement JSON-RPC envelopes, authorization or transport;
// metadata byte slices are borrowed during a call and returned injection bytes
// are independently owned. Trust callbacks are synchronous and not panic-contained.
package tenancyjsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/faustbrian/go-tenancy/v2"
)

const (
	// DefaultMaxMetadataBytes bounds JSON-RPC metadata by default.
	DefaultMaxMetadataBytes = 4096
	// MaximumMetadataBytes is the largest configurable metadata bound.
	MaximumMetadataBytes = 65536
)

var (
	// ErrInvalidOptions reports missing trust or invalid bounds and field names.
	ErrInvalidOptions = errors.New("tenancy jsonrpc: invalid options")
	// ErrInvalidContext reports a nil extraction context.
	ErrInvalidContext = errors.New("tenancy jsonrpc: invalid context")
	// ErrInvalidMetadata reports malformed, non-object, or ambiguous JSON.
	ErrInvalidMetadata = errors.New("tenancy jsonrpc: invalid metadata")
	// ErrOversizedMetadata reports input beyond the configured byte limit.
	ErrOversizedMetadata = errors.New("tenancy jsonrpc: oversized metadata")
)

// Options configure immutable JSON-RPC extraction policy.
type Options struct {
	// Field is the exact case-sensitive top-level tenant member; empty uses tenant_id.
	Field string
	// MaxMetadataBytes bounds both accepted and injected encoded metadata.
	// Zero uses 4096; overrides must be 2..65536 bytes.
	MaxMetadataBytes int
	// Trust is required and owns authenticated immediate-peer propagation policy.
	// It runs after structural decoding, with the borrowed caller context; it does not
	// imply the peer is authorized for a tenant and is not panic-contained.
	Trust func(context.Context) bool
}

// Codec extracts, injects, and installs tenant scope in JSON metadata. Use New;
// the zero value is invalid. Configuration is immutable and may be reused
// concurrently when the trust callback and borrowed inputs support that use.
type Codec struct {
	propagation *tenancy.PropagationCodec
	field       string
	maximum     int
	trust       func(context.Context) bool
}

// New validates extraction trust, field name, and resource bounds.
func New(options Options) (*Codec, error) {
	if options.Trust == nil {
		return nil, ErrInvalidOptions
	}
	maximum := options.MaxMetadataBytes
	if maximum == 0 {
		maximum = DefaultMaxMetadataBytes
	}
	if maximum < 2 || maximum > MaximumMetadataBytes {
		return nil, ErrInvalidOptions
	}
	field := options.Field
	if field == "" {
		field = tenancy.DefaultTenantField
	}
	propagation, err := tenancy.NewPropagationCodec(tenancy.PropagationOptions{Field: field})
	if err != nil {
		return nil, ErrInvalidOptions
	}
	return &Codec{propagation: propagation, field: field, maximum: maximum, trust: options.Trust}, nil
}

// Extract validates a bounded metadata object and accepts its tenant field
// only when Trust approves the supplied immediate-boundary context. It requires
// exactly one string tenant value; other top-level duplicate members are invalid,
// while repeated tenant members receive core duplicate/conflict/count categories.
// It parses before the trust decision and does not check ctx.Err or evaluate
// nested non-tenant field semantics. Caller input must stay unchanged during use.
func (codec *Codec) Extract(ctx context.Context, metadata []byte) (tenancy.Scope, error) {
	if codec == nil || codec.propagation == nil || codec.trust == nil {
		return tenancy.Scope{}, ErrInvalidOptions
	}
	if ctx == nil {
		return tenancy.Scope{}, ErrInvalidContext
	}
	_, values, err := decodeMetadata(metadata, codec.field, codec.maximum)
	if err != nil {
		return tenancy.Scope{}, err
	}
	carrier := tenancy.MapCarrier{codec.field: values}
	return codec.propagation.Extract(carrier, codec.trust(ctx))
}

// Accept extracts tenant scope and derives from ctx without replacing scope.
func (codec *Codec) Accept(ctx context.Context, metadata []byte) (context.Context, error) {
	scope, err := codec.Extract(ctx, metadata)
	if err != nil {
		return nil, err
	}
	return tenancy.WithScope(ctx, scope)
}

// Inject returns a bounded new metadata object containing tenant scope.
// Existing tenant fields, exceptional scopes, and encoded output beyond
// MaxMetadataBytes are rejected. Input must be one valid JSON object (use {} for
// empty metadata), and is not mutated. Non-tenant RawMessage values are retained
// semantically, not byte-for-byte: top-level members are sorted and encoding/json
// may compact whitespace and escape strings. Nested duplicate members are not
// a recursive validation surface. No trust callback is invoked for injection.
func (codec *Codec) Inject(metadata []byte, scope tenancy.Scope) ([]byte, error) {
	if codec == nil || codec.propagation == nil {
		return nil, ErrInvalidOptions
	}
	object, values, err := decodeMetadata(metadata, codec.field, codec.maximum)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		return nil, tenancy.ErrTenantMetadataOverwrite
	}
	carrier := tenancy.MapCarrier{}
	if err := codec.propagation.Inject(carrier, scope); err != nil {
		return nil, err
	}
	encodedTenant, _ := json.Marshal(carrier.Values(codec.field)[0])
	object[codec.field] = encodedTenant
	encoded, _ := json.Marshal(object)
	if len(encoded) > codec.maximum {
		return nil, ErrOversizedMetadata
	}
	return encoded, nil
}

func decodeMetadata(
	metadata []byte,
	field string,
	maximum int,
) (map[string]json.RawMessage, []string, error) {
	if len(metadata) > maximum {
		return nil, nil, ErrOversizedMetadata
	}
	decoder := json.NewDecoder(bytes.NewReader(metadata))
	start, err := decoder.Token()
	if !matchingDelimiter(start, err, json.Delim('{')) {
		return nil, nil, ErrInvalidMetadata
	}
	object := make(map[string]json.RawMessage)
	var tenantValues []string
	for decoder.More() {
		keyToken, tokenErr := decoder.Token()
		key, ok := stringToken(keyToken, tokenErr)
		if !ok {
			return nil, nil, ErrInvalidMetadata
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, nil, ErrInvalidMetadata
		}
		if _, exists := object[key]; exists && key != field {
			return nil, nil, ErrInvalidMetadata
		}
		if key == field {
			var tenantValue string
			if err := json.Unmarshal(value, &tenantValue); err != nil {
				return nil, nil, ErrInvalidMetadata
			}
			tenantValues = append(tenantValues, tenantValue)
		}
		object[key] = append(json.RawMessage(nil), value...)
	}
	end, err := decoder.Token()
	if !matchingDelimiter(end, err, json.Delim('}')) {
		return nil, nil, ErrInvalidMetadata
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, nil, ErrInvalidMetadata
	}
	return object, tenantValues, nil
}

func matchingDelimiter(token json.Token, err error, expected json.Delim) bool {
	return err == nil && token == expected
}

func stringToken(token json.Token, err error) (string, bool) {
	value, ok := token.(string)
	return value, err == nil && ok
}
