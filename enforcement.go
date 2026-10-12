package tenancy

import (
	"context"
	"errors"
)

// ErrTenantMismatch reports cross-tenant use without disclosing either ID.
var ErrTenantMismatch = errors.New("tenancy: tenant mismatch")

// TenantAsserter is the small consumer-facing application and persistence
// boundary for operations that require one expected tenant.
type TenantAsserter interface {
	// AssertTenant rejects absent, exceptional or mismatched tenant scope.
	// Implementations own any additional authorization, cancellation and error policy.
	AssertTenant(context.Context, TenantID) error
}

// TenantAssertFunc adapts a function to TenantAsserter.
type TenantAssertFunc func(context.Context, TenantID) error

// AssertTenant calls the adapted function without validation, error rewriting
// or panic containment. A nil function panics; callers must supply an implementation.
func (assert TenantAssertFunc) AssertTenant(ctx context.Context, expected TenantID) error {
	return assert(ctx, expected)
}

// AssertTenant fails closed unless ctx is tenant-bound to expected.
func AssertTenant(ctx context.Context, expected TenantID) error {
	actual, err := RequireTenant(ctx)
	if err != nil {
		return err
	}
	if !expected.Valid() || !actual.Equal(expected) {
		return ErrTenantMismatch
	}
	return nil
}

// AssertScope fails closed unless ctx contains exactly expected scope,
// including metadata and administrative reason. It does not authorize that scope
// or check whether the context has been canceled.
func AssertScope(ctx context.Context, expected Scope) error {
	actual, err := RequireScope(ctx)
	if err != nil {
		return err
	}
	if !expected.Valid() || !actual.Equal(expected) {
		return ErrConflictingScope
	}
	return nil
}
