// Package tenancypostgres provides explicit PostgreSQL predicates,
// transaction-local tenant settings, and Row-Level Security plans. It never
// rewrites caller queries or installs hidden global state.
// Use Predicate for explicit query filtering, NewRLSPlan for migration data and
// NewManager for transaction-local enforcement over database/sql. The caller
// supplies the registered driver/Connector and deployment credentials.
// Predicates and RLS plans only return migration/query data; callers must install
// and use them with matching Manager settings, restricted roles and policies.
// WithSystem does not bypass RLS or grant database privileges.
//
// Manager borrows a caller-owned Connector and owns each leased connection and
// transaction until return. Callbacks must use only that transaction, not finish
// it themselves or retain it, and must not change the tenant setting. Driver and
// callback errors remain inspectable and may contain sensitive diagnostics;
// identifier redaction does not sanitize those errors. No implicit retry or
// background worker is provided.
package tenancypostgres

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/faustbrian/go-tenancy/v2"
)

const (
	// DefaultSetting is the custom PostgreSQL setting used by Manager and RLS.
	DefaultSetting         = "app.tenant_id"
	maximumIdentifierBytes = 63
	maximumParameter       = 65535
)

var (
	// ErrInvalidIdentifier reports an unsafe PostgreSQL identifier.
	ErrInvalidIdentifier = errors.New("tenancy postgres: invalid identifier")
	// ErrInvalidParameter reports an invalid PostgreSQL placeholder position.
	ErrInvalidParameter = errors.New("tenancy postgres: invalid parameter")
	// ErrInvalidRLSOptions reports incomplete or unsafe RLS configuration.
	ErrInvalidRLSOptions = errors.New("tenancy postgres: invalid RLS options")
)

// QueryPredicate is an explicit tenant predicate and its owned arguments.
type QueryPredicate struct {
	// Clause is a quoted equality fragment, not a complete query; the caller must include
	// it.
	Clause string
	// Arguments is a new caller-owned slice containing the raw tenant string.
	// Align its position with the supplied placeholder and keep it out of diagnostics.
	Arguments []any
}

// Predicate returns a quoted equality fragment and new raw-tenant argument for
// exactly one tenant scope. Column must be an unqualified ASCII identifier of
// 1..63 bytes and parameter a 1-based PostgreSQL position up to 65535. It neither
// executes the query nor authorizes its scope, and callers must place the clause
// and argument in every applicable read or mutation.
func Predicate(scope tenancy.Scope, column string, parameter int) (QueryPredicate, error) {
	if !scope.Valid() || scope.Kind() != tenancy.ScopeTenant {
		return QueryPredicate{}, tenancy.ErrTenantScopeRequired
	}
	quoted, err := quoteIdentifier(column, 1)
	if err != nil {
		return QueryPredicate{}, err
	}
	if parameter < 1 || parameter > maximumParameter {
		return QueryPredicate{}, ErrInvalidParameter
	}
	return QueryPredicate{
		Clause:    quoted + " = $" + fmt.Sprint(parameter),
		Arguments: []any{scope.TenantID().Value()},
	}, nil
}

// RLSOptions identify one table-owned tenant isolation policy.
type RLSOptions struct {
	// Table is a required one- or two-segment ASCII identifier, optionally
	// schema-qualified.
	Table string
	// Column is the required unqualified tenant column identifier.
	Column string
	// Policy is the required restrictive policy name.
	Policy string
	// GrantPolicy is the distinct permissive policy name; empty derives a bounded name
	// from Policy with a SHA-256 suffix. Existing policies are not discovered or audited.
	GrantPolicy string
	// Setting is the custom setting used by both expressions; empty uses DefaultSetting.
	// It must match Manager configuration and use dot-separated ASCII identifiers, at most
	// 64 bytes.
	Setting string
}

// RLSPlan contains explicit migration statements. Apply them through the
// application's migration owner; this package performs no automatic DDL.
type RLSPlan struct {
	// Enable enables RLS on the selected table; it does not execute DDL.
	Enable string
	// Force forces RLS for ordinary table owners, not superusers or BYPASSRLS roles.
	Force string
	// CreateGrant creates the scoped permissive USING/WITH CHECK policy; apply before
	// Create.
	CreateGrant string
	// Create creates the restrictive USING/WITH CHECK policy constraining other permissive
	// grants.
	Create string
	// DropGrant removes the paired permissive policy; apply before Drop during rollback.
	DropGrant string
	// Drop removes the restrictive policy; never leave a broader grant without its
	// restriction.
	Drop string
	// Setting is the validated setting name to share with Manager configuration.
	Setting string
}

// NewRLSPlan validates and quotes identifiers and creates a fail-closed policy
// that treats missing and reset settings as no tenant. It returns a paired
// permissive grant and restrictive constraint with identical USING/WITH CHECK
// expressions. Apply Enable, Force, CreateGrant and Create through a migration
// owner; drop the grant before the restriction. It does not inspect existing
// policies, install roles or prove that an unrestricted SQL caller cannot
// manipulate the configured setting. Privileged roles may bypass RLS.
func NewRLSPlan(options RLSOptions) (RLSPlan, error) {
	table, tableErr := quoteIdentifier(options.Table, 2)
	column, columnErr := quoteIdentifier(options.Column, 1)
	policy, policyErr := quoteIdentifier(options.Policy, 1)
	grantPolicyValue := options.GrantPolicy
	if grantPolicyValue == "" {
		grantPolicyValue = derivedGrantPolicy(options.Policy)
	}
	grantPolicy, grantPolicyErr := quoteIdentifier(grantPolicyValue, 1)
	setting := options.Setting
	if setting == "" {
		setting = DefaultSetting
	}
	if tableErr != nil || columnErr != nil || policyErr != nil || grantPolicyErr != nil ||
		grantPolicyValue == options.Policy || !validSetting(setting) {
		return RLSPlan{}, ErrInvalidRLSOptions
	}
	expression := column + " = NULLIF(current_setting('" + setting + "', true), '')"
	return RLSPlan{
		Enable:      "ALTER TABLE " + table + " ENABLE ROW LEVEL SECURITY",
		Force:       "ALTER TABLE " + table + " FORCE ROW LEVEL SECURITY",
		CreateGrant: "CREATE POLICY " + grantPolicy + " ON " + table + " AS PERMISSIVE USING (" + expression + ") WITH CHECK (" + expression + ")",
		Create:      "CREATE POLICY " + policy + " ON " + table + " AS RESTRICTIVE USING (" + expression + ") WITH CHECK (" + expression + ")",
		DropGrant:   "DROP POLICY IF EXISTS " + grantPolicy + " ON " + table,
		Drop:        "DROP POLICY IF EXISTS " + policy + " ON " + table,
		Setting:     setting,
	}, nil
}

func derivedGrantPolicy(policy string) string {
	const maximumPrefix = maximumIdentifierBytes - len("_grant_") - 16
	prefix := policy[:min(len(policy), maximumPrefix)]
	digest := sha256.Sum256([]byte(policy))
	return prefix + "_grant_" + hex.EncodeToString(digest[:8])
}

func quoteIdentifier(value string, maximumSegments int) (string, error) {
	segments := strings.Split(value, ".")
	if len(segments) == 0 || len(segments) > maximumSegments {
		return "", ErrInvalidIdentifier
	}
	quoted := make([]string, len(segments))
	for index, segment := range segments {
		if !validIdentifierSegment(segment) {
			return "", ErrInvalidIdentifier
		}
		quoted[index] = `"` + segment + `"`
	}
	return strings.Join(quoted, "."), nil
}

func validIdentifierSegment(value string) bool {
	if len(value) == 0 || len(value) > maximumIdentifierBytes ||
		!asciiLetter(value[0]) && value[0] != '_' {
		return false
	}
	for index := 1; index < len(value); index++ {
		char := value[index]
		if !asciiLetter(char) && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func validSetting(value string) bool {
	if len(value) == 0 || len(value) > 64 || !strings.Contains(value, ".") {
		return false
	}
	segments := strings.Split(value, ".")
	for _, segment := range segments {
		if !validIdentifierSegment(segment) {
			return false
		}
	}
	return true
}

func asciiLetter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}
