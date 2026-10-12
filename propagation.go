package tenancy

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// DefaultTenantField is the transport-neutral metadata field name.
const DefaultTenantField = "tenant_id"

var (
	// ErrInvalidPropagation reports invalid codec or carrier configuration.
	ErrInvalidPropagation = errors.New("tenancy: invalid propagation")
	// ErrTenantMetadataMissing reports absent required tenant metadata.
	ErrTenantMetadataMissing = errors.New("tenancy: tenant metadata missing")
	// ErrTenantMetadataDuplicate reports repeated identical metadata.
	ErrTenantMetadataDuplicate = errors.New("tenancy: tenant metadata duplicate")
	// ErrTenantMetadataConflicting reports repeated distinct metadata.
	ErrTenantMetadataConflicting = errors.New("tenancy: tenant metadata conflicting")
	// ErrTenantMetadataOversized reports too many values at one boundary.
	ErrTenantMetadataOversized = errors.New("tenancy: tenant metadata oversized")
	// ErrTenantMetadataUntrusted reports metadata from an untrusted boundary.
	ErrTenantMetadataUntrusted = errors.New("tenancy: tenant metadata untrusted")
	// ErrTenantMetadataOverwrite reports injection into a populated field.
	ErrTenantMetadataOverwrite = errors.New("tenancy: tenant metadata overwrite")
)

// Carrier is an explicit metadata boundary. Values must return an immutable,
// transport-bounded view or copy and no more than nine values; Set must replace
// the field with exactly one value.
type Carrier interface {
	// Values returns field values without retaining a mutable view owned by another
	// concurrent caller. Custom carriers must cap allocation/count and byte resources
	// before returning; the codec cannot preempt or undo carrier work.
	Values(string) []string
	// Set replaces the field with exactly one raw tenant value. Implementations
	// own synchronization; Inject is a check followed by Set, not an atomic operation.
	Set(string, string)
}

// MapCarrier is a mutable caller-owned in-memory carrier for message metadata and
// adapters. Values returns a copy, but map assignment/aliases share mutable
// storage: callers must coordinate writes. It does not enforce transport count
// or byte bounds itself. A nil map cannot be written through Set.
type MapCarrier map[string][]string

// Values returns an independently owned copy of a field's values.
func (carrier MapCarrier) Values(field string) []string {
	return append([]string(nil), carrier[field]...)
}

// Set replaces field with exactly one value without validating either string.
// A nil map panics; callers own mutation synchronization and transport bounds.
func (carrier MapCarrier) Set(field, value string) {
	carrier[field] = []string{value}
}

// PropagationOptions configure an immutable tenant field name.
type PropagationOptions struct {
	// Field is the case-sensitive metadata key; empty selects DefaultTenantField.
	// It uses the bounded core metadata-key alphabet, not every transport header syntax.
	Field string
}

// PropagationCodec extracts and injects tenant-bound scope without deciding
// whether an inbound carrier is trusted. Construct it with NewPropagationCodec;
// a nil receiver or zero codec is invalid. An initialized codec is immutable and
// can be reused concurrently with independently synchronized carriers.
type PropagationCodec struct {
	field string
}

// NewPropagationCodec validates and copies codec configuration.
func NewPropagationCodec(options PropagationOptions) (*PropagationCodec, error) {
	field := options.Field
	if field == "" {
		field = DefaultTenantField
	}
	if !validMetadataKey(field) {
		return nil, fmt.Errorf("%w: field", ErrInvalidPropagation)
	}
	return &PropagationCodec{field: field}, nil
}

// Extract parses exactly one tenant value after the immediate boundary has
// explicitly established trust. Presence alone never establishes trust.
// Missing, more-than-eight, conflicting and identical duplicate values are
// classified before trust refusal; one trusted value is then parsed. Neither
// carrier panics nor caller trust decisions are contained or authenticated here.
// Extracted scopes contain empty optional metadata.
func (codec *PropagationCodec) Extract(carrier Carrier, trusted bool) (Scope, error) {
	if codec == nil || nilLike(carrier) || !validMetadataKey(codec.field) {
		return Scope{}, ErrInvalidPropagation
	}
	values := carrier.Values(codec.field)
	if len(values) == 0 {
		return Scope{}, ErrTenantMetadataMissing
	}
	if len(values) > 8 {
		return Scope{}, ErrTenantMetadataOversized
	}
	if len(values) > 1 {
		for _, value := range values[1:] {
			if value != values[0] {
				return Scope{}, ErrTenantMetadataConflicting
			}
		}
		return Scope{}, ErrTenantMetadataDuplicate
	}
	if !trusted {
		return Scope{}, ErrTenantMetadataUntrusted
	}
	id, err := ParseTenantID(values[0])
	if err != nil {
		return Scope{}, err
	}
	return NewTenantScope(id, Metadata{})
}

// Inject writes one raw tenant value and refuses any existing field value or
// exceptional scope. It does not transmit optional Metadata or audit reasons.
// Values checking and Set are separate calls; callers own carrier mutation
// serialization, resource limits and panic/error policy.
func (codec *PropagationCodec) Inject(carrier Carrier, scope Scope) error {
	if codec == nil || nilLike(carrier) || !validMetadataKey(codec.field) {
		return ErrInvalidPropagation
	}
	if !scope.Valid() || scope.Kind() != ScopeTenant {
		return ErrTenantScopeRequired
	}
	if len(carrier.Values(codec.field)) != 0 {
		return ErrTenantMetadataOverwrite
	}
	carrier.Set(codec.field, scope.TenantID().Value())
	return nil
}

// Accept extracts trusted tenant metadata and installs it without replacing an
// existing context scope.
func (codec *PropagationCodec) Accept(
	ctx context.Context,
	carrier Carrier,
	trusted bool,
) (context.Context, error) {
	scope, err := codec.Extract(carrier, trusted)
	if err != nil {
		return nil, err
	}
	return WithScope(ctx, scope)
}

// InjectFromContext requires and injects tenant-bound context scope.
func (codec *PropagationCodec) InjectFromContext(carrier Carrier, ctx context.Context) error {
	scope, err := RequireScope(ctx)
	if err != nil {
		return ErrTenantScopeRequired
	}
	return codec.Inject(carrier, scope)
}

func nilLike(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
