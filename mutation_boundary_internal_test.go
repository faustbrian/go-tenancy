package tenancy

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestInternalValidationBoundariesRejectImpossibleMixedState(t *testing.T) {
	t.Parallel()

	id := MustTenantID("tenant-a")
	reason, err := NewAdministrativeReason("operator", "maintenance", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ScopeFromContext(context.WithValue(context.Background(), contextKey{}, Scope{})); ok {
		t.Fatal("invalid stored scope was accepted")
	}
	if (Scope{kind: ScopeTenant, tenant: id, reason: reason}).Valid() {
		t.Fatal("tenant scope accepted an administrative reason")
	}
	if (Scope{kind: ScopeSystem, tenant: id, reason: reason}).Valid() {
		t.Fatal("system scope accepted a tenant ID")
	}
	if _, err := NewSystemScope(SystemCapability{reason: reason}, Metadata{}); err == nil {
		t.Fatal("invalid capability with a valid reason was accepted")
	}
	for name, invalid := range map[string]AdministrativeReason{
		"actor":   {purpose: "maintenance"},
		"purpose": {actor: "operator"},
	} {
		if invalid.valid() {
			t.Fatalf("reason with invalid %s was accepted", name)
		}
	}
}

func TestInternalTextAndCollectionBoundariesAreInclusive(t *testing.T) {
	t.Parallel()

	entries := make(map[string]string, maxMetadataEntries)
	for index := range maxMetadataEntries {
		entries[string(rune('a'+index%26))+strings.Repeat("x", index/26)] = "value"
	}
	if _, err := NewMetadata(entries); err != nil {
		t.Fatalf("maximum metadata entries error = %v", err)
	}
	if !validMetadataKey(strings.Repeat("a", maxMetadataKey)) {
		t.Fatal("maximum metadata key was rejected")
	}
	if !validPrintable(strings.Repeat("x", maxMetadataValue), maxMetadataValue, false) {
		t.Fatal("maximum printable value was rejected")
	}
	if !validPrintable(" ~", 2, false) {
		t.Fatal("printable ASCII boundaries were rejected")
	}
	for _, value := range []string{"\x1f", "\x7f"} {
		if validPrintable(value, 1, false) {
			t.Fatalf("non-printable boundary %q was accepted", value)
		}
	}
	for _, char := range []byte{'a', 'z', 'A', 'Z', '0', '9'} {
		if !asciiAlphaNumeric(char) {
			t.Fatalf("ASCII boundary %q was rejected", char)
		}
	}
}

func TestGroupWaitPrefersCompletionAcrossReadyTransition(t *testing.T) {
	t.Parallel()

	for iteration := range 128 {
		group := &Group{done: make(chan struct{})}
		waitContext, cancelWait := context.WithCancel(context.Background())
		reached := make(chan struct{})
		proceed := make(chan struct{})
		result := make(chan error, 1)
		go func() {
			result <- group.wait(&blockedDoneContext{
				Context: waitContext,
				reached: reached,
				proceed: proceed,
			})
		}()
		select {
		case <-reached:
		case <-time.After(10 * time.Second):
			t.Fatalf("iteration %d: wait did not reach context selection", iteration)
		}
		close(group.done)
		cancelWait()
		close(proceed)
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("iteration %d: wait error = %v", iteration, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("iteration %d: wait did not return", iteration)
		}
	}
}

func TestGroupGracefulTimeoutEventuallyReleasesOwnedContext(t *testing.T) {
	t.Parallel()

	group, err := NewGroup(context.Background(), GroupOptions{MaxConcurrent: 1})
	if err != nil {
		t.Fatalf("NewGroup() error = %v", err)
	}
	scope, err := NewTenantScope(MustTenantID("tenant-a"), Metadata{})
	if err != nil {
		t.Fatalf("NewTenantScope() error = %v", err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseTask := func() {
		releaseOnce.Do(func() { close(release) })
	}
	defer releaseTask()
	if err := group.Submit(context.Background(), scope, func(context.Context) error {
		close(started)
		<-release
		return nil
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("accepted work did not start")
	}
	waitContext, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	if err := group.Drain(waitContext); !errors.Is(err, context.Canceled) {
		t.Fatalf("Drain(cancelled context) error = %v", err)
	}
	select {
	case <-group.ctx.Done():
		t.Fatal("Drain cancelled accepted work after its wait timed out")
	default:
	}
	releaseTask()
	select {
	case <-group.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("final task completion retained the group-owned context")
	}
}

func TestGroupRemainsOpenAfterCurrentWorkCompletes(t *testing.T) {
	t.Parallel()

	group, err := NewGroup(context.Background(), GroupOptions{MaxConcurrent: 1})
	if err != nil {
		t.Fatalf("NewGroup() error = %v", err)
	}
	scope, err := NewTenantScope(MustTenantID("tenant-a"), Metadata{})
	if err != nil {
		t.Fatalf("NewTenantScope() error = %v", err)
	}
	returned := make(chan struct{})
	if err := group.Submit(context.Background(), scope, func(context.Context) error {
		close(returned)
		return nil
	}); err != nil {
		t.Fatalf("Submit(first) error = %v", err)
	}
	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("first task did not return")
	}

	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for {
		group.mutex.Lock()
		active := group.active
		group.mutex.Unlock()
		if active == 0 {
			break
		}
		select {
		case <-deadline.C:
			t.Fatal("first task did not complete")
		default:
		}
		runtime.Gosched()
	}
	select {
	case <-group.ctx.Done():
		t.Fatal("completing current work cancelled an open group")
	default:
	}
	if err := group.Submit(context.Background(), scope, func(context.Context) error { return nil }); err != nil {
		t.Fatalf("Submit(after current work) error = %v", err)
	}
	drainContext, cancelDrain := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelDrain()
	if err := group.Drain(drainContext); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
}

type blockedDoneContext struct {
	context.Context
	reached chan struct{}
	proceed chan struct{}
	once    sync.Once
}

func (ctx *blockedDoneContext) Done() <-chan struct{} {
	ctx.once.Do(func() {
		close(ctx.reached)
		<-ctx.proceed
	})
	return ctx.Context.Done()
}
