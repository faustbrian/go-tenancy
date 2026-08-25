// Package adapter is the only reviewed construction boundary in the tenancy
// analyzer fixture.
package adapter

import (
	auditmemory "github.com/faustbrian/go-audit/memory"
	cachememory "github.com/faustbrian/go-cache/backend/memory"
	"github.com/faustbrian/go-queue"
	telemetryotlp "github.com/faustbrian/go-telemetry/otlp"
	workflowpostgres "github.com/faustbrian/go-workflow/postgres"
)

// Construct proves that the exact reviewed adapter exception remains narrow.
func Construct() {
	_, _ = cachememory.New(cachememory.Config{})
	_, _ = auditmemory.New(auditmemory.Config{})
	_, _ = queue.NewQueue()
	_, _ = workflowpostgres.New(nil, workflowpostgres.Config{})
	_, _ = telemetryotlp.NewTraceExporter(nil, telemetryotlp.Config{})
}
