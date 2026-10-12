package tenancy

import (
	"context"
	"errors"
)

const (
	maximumIterationPageSize = 1000
	maximumIterationTenants  = 1_000_000
	maximumCursorBytes       = 256
)

// ErrInvalidIteration reports an unsafe administrative iteration contract.
var ErrInvalidIteration = errors.New("tenancy: invalid administrative iteration")

// TenantPage is one bounded page from a resumable tenant source. NextCursor is
// empty only when the source is complete.
type TenantPage struct {
	// Tenants contains at most the requested PageSize validated IDs. The source owns
	// its backing storage and must not mutate it while that page is being processed.
	Tenants []TenantID
	// NextCursor is the source-owned stable snapshot continuation, at most 256 bytes;
	// empty is terminal. It is not authenticated or cryptographically bound here.
	NextCursor string
}

// TenantPager is the consumer-defined source for bounded administrative work.
type TenantPager interface {
	// ListTenants reads the supplied base context, cursor and positive page ceiling.
	// The source must preserve snapshot ordering across retries/resume, honor cancellation
	// and bound its own resources; the library cannot prove those properties.
	ListTenants(context.Context, string, int) (TenantPage, error)
}

// ResumeToken identifies an exact page position. Offset is the next tenant in
// the page at Cursor, allowing retry after partial-page failure.
type ResumeToken struct {
	// Cursor selects a stable source page, at most 256 bytes; empty starts at the first
	// page.
	Cursor string
	// Offset is the nonnegative index of the next tenant within the selected page.
	Offset int
}

// AdministrativeAudit records system intent before each tenant operation.
// Implementations own durability and redaction. Returning an error prevents
// that attempt's operation but cannot undo prior side effects. Audit callbacks
// run synchronously and must honor cancellation; errors/panics propagate unchanged.
type AdministrativeAudit func(context.Context, AdministrativeReason, TenantID) error

// IterationOptions bound work and make audit integration mandatory.
type IterationOptions struct {
	// PageSize is the requested positive page ceiling, at most 1000.
	PageSize int
	// MaxTenants caps successful operations in this invocation, from 1 to 1000000.
	// At most MaxTenants+1 pages are requested; it is not an aggregate concurrency limit.
	MaxTenants int
	// Resume identifies where to continue against the same source snapshot; zero starts
	// anew.
	Resume ResumeToken
	// Audit is required and runs before every attempted tenant operation, including
	// retries.
	// It owns durable attribution and sanitized diagnostics, not the library.
	Audit AdministrativeAudit
}

// IterationResult reports this run's progress and exact continuation point.
type IterationResult struct {
	// Processed counts operations that returned nil during this invocation only.
	Processed int
	// Resume is the next operation position, or the failed tenant position on audit/work
	// error.
	// Failures may have side effects already; resume requires idempotent operations.
	Resume ResumeToken
	// Complete is true only when the observed source ends, not merely when MaxTenants is
	// reached.
	Complete bool
}

// IterateTenants executes bounded sequential work from an unscoped base
// context. system must be an explicit system scope. Audit callbacks receive a
// system-scoped child; operations receive a fresh tenant-scoped child derived
// from the original base. Base must have no installed Scope, not ScopeUnscoped.
// The pager receives the base context, not the system child. Identity does not
// persist in the library between iterations, but callers must not retain child
// contexts or resources. Duplicate tenants/cursor cycles are rejected within
// this invocation. Audit/source/work errors and panics propagate unchanged;
// successful side effects before failure are not undone or implicitly retried.
func IterateTenants(
	ctx context.Context,
	system Scope,
	source TenantPager,
	options IterationOptions,
	operation func(context.Context, TenantID) error,
) (IterationResult, error) {
	if !validIteration(ctx, system, source, options, operation) {
		return IterationResult{}, ErrInvalidIteration
	}
	systemContext, _ := WithScope(ctx, system)

	result := IterationResult{Resume: options.Resume}
	cursor := options.Resume.Cursor
	offset := options.Resume.Offset
	seen := make(map[TenantID]struct{})
	seenCursors := make(map[string]struct{})
	pages := 0
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if _, repeated := seenCursors[cursor]; repeated || pages == options.MaxTenants+1 {
			return result, ErrInvalidIteration
		}
		seenCursors[cursor] = struct{}{}
		pages++
		page, pageErr := source.ListTenants(ctx, cursor, options.PageSize)
		if pageErr != nil {
			return result, pageErr
		}
		if err := validateTenantPage(page, cursor, offset, options.PageSize, seen); err != nil {
			return result, err
		}
		for index := offset; index < len(page.Tenants); index++ {
			tenant := page.Tenants[index]
			result.Resume = ResumeToken{Cursor: cursor, Offset: index}
			if err := options.Audit(systemContext, system.AdministrativeReason(), tenant); err != nil {
				return result, err
			}
			if err := ctx.Err(); err != nil {
				return result, err
			}
			tenantScope, _ := NewTenantScope(tenant, Metadata{})
			if err := RunScoped(ctx, tenantScope, func(scoped context.Context) error {
				return operation(scoped, tenant)
			}); err != nil {
				return result, err
			}
			result.Processed++
			if result.Processed == options.MaxTenants {
				return boundedIterationResult(result, cursor, index+1, page), nil
			}
		}
		if page.NextCursor == "" {
			result.Resume = ResumeToken{}
			result.Complete = true
			return result, nil
		}
		cursor = page.NextCursor
		offset = 0
		result.Resume = ResumeToken{Cursor: cursor}
	}
}

func validIteration(
	ctx context.Context,
	system Scope,
	source TenantPager,
	options IterationOptions,
	operation func(context.Context, TenantID) error,
) bool {
	if ctx == nil || !system.Valid() || system.Kind() != ScopeSystem || nilLike(source) ||
		operation == nil || options.Audit == nil || options.PageSize <= 0 ||
		options.PageSize > maximumIterationPageSize || options.MaxTenants <= 0 ||
		options.MaxTenants > maximumIterationTenants || options.Resume.Offset < 0 ||
		len(options.Resume.Cursor) > maximumCursorBytes {
		return false
	}
	_, scoped := ScopeFromContext(ctx)
	return !scoped
}

func validateTenantPage(
	page TenantPage,
	cursor string,
	offset int,
	pageSize int,
	seen map[TenantID]struct{},
) error {
	if len(page.Tenants) > pageSize || offset > len(page.Tenants) ||
		len(page.NextCursor) > maximumCursorBytes ||
		page.NextCursor != "" && page.NextCursor == cursor {
		return ErrInvalidIteration
	}
	for _, tenant := range page.Tenants {
		if !tenant.Valid() {
			return ErrInvalidIteration
		}
		if _, exists := seen[tenant]; exists {
			return ErrInvalidIteration
		}
		seen[tenant] = struct{}{}
	}
	return nil
}

func boundedIterationResult(
	result IterationResult,
	cursor string,
	nextOffset int,
	page TenantPage,
) IterationResult {
	if nextOffset < len(page.Tenants) {
		result.Resume = ResumeToken{Cursor: cursor, Offset: nextOffset}
		return result
	}
	if page.NextCursor != "" {
		result.Resume = ResumeToken{Cursor: page.NextCursor}
		return result
	}
	result.Resume = ResumeToken{}
	result.Complete = true
	return result
}
