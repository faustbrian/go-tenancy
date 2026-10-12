// Package metrics is the exact telemetry sink used by the tenancy analyzer
// fixture. It deliberately accepts any value so the analyzer, not the fixture
// type signature, owns the high-cardinality rejection.
package metrics

// Label is the inert exact sink used for analyzer matching. It performs no
// recording or I/O; its permissive signature leaves rejection to the analyzer.
func Label(any) {}
