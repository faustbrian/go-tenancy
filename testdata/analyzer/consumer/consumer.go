// Package consumer deliberately bypasses every configured tenancy boundary.
package consumer

import (
	"context"

	auditmemory "github.com/faustbrian/go-audit/memory"
	cachememory "github.com/faustbrian/go-cache/backend/memory"
	"github.com/faustbrian/go-queue"
	telemetryotlp "github.com/faustbrian/go-telemetry/otlp"
	"github.com/faustbrian/go-tenancy"
	"github.com/faustbrian/go-tenancy/testdata/analyzer/metrics"
	workflowpostgres "github.com/faustbrian/go-workflow/postgres"
)

// Bypass contains one direct call for every policy-owned negative fixture.
func Bypass(tenant tenancy.TenantID) {
	_, _ = cachememory.New(cachememory.Config{})
	_, _ = auditmemory.New(auditmemory.Config{})
	_, _ = queue.NewQueue()
	_, _ = workflowpostgres.New(nil, workflowpostgres.Config{})
	_, _ = telemetryotlp.NewTraceExporter(nil, telemetryotlp.Config{})
	metrics.Label(tenant)
	_ = context.Background()
}
