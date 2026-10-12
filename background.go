package tenancy

import (
	"context"
	"errors"
	"sync"
)

const maximumGroupConcurrency = 1024

var (
	// ErrInvalidGroup reports invalid ownership, limits, or receiver state.
	ErrInvalidGroup = errors.New("tenancy: invalid background group")
	// ErrGroupClosed reports submission after graceful close or shutdown begins.
	ErrGroupClosed = errors.New("tenancy: background group closed")
)

// GroupOptions define bounded concurrency and task error ownership. HandleError
// may be invoked concurrently by independently completing tasks.
type GroupOptions struct {
	// MaxConcurrent bounds active operations plus their error handling, from 1 to 1024.
	// There is no unbounded internal queue; Submit waits for capacity.
	MaxConcurrent int
	// HandleError optionally consumes each nonnil task error synchronously in its task
	// goroutine, outside library locks. Nil discards task errors; lifecycle methods do not
	// return them. It must permit concurrent calls, return promptly and not join its own
	// Group; errors and panics are not sanitized or contained.
	HandleError func(Scope, error)
}

// Group owns every goroutine it starts. Submit is bounded and cancellable;
// callers must invoke Drain for graceful completion or Shutdown for
// cancellation. Concurrent lifecycle calls join one terminal state while each
// call observes its own wait context, and any Shutdown cancels the shared task
// context. The mutex owns intake/active/terminal state and the semaphore owns
// capacity. Scope and cancellation state belong to individual task contexts,
// never an ambient global tenant. Cancellation is cooperative: operation and
// error-handler panics are not recovered, and Shutdown cannot force code to return.
// Use NewGroup; the zero value is invalid. Submit and lifecycle methods support
// concurrent calls but do not supply safety for callback-owned state.
type Group struct {
	ctx         context.Context
	cancel      context.CancelFunc
	semaphore   chan struct{}
	handleError func(Scope, error)

	mutex      sync.Mutex
	active     int
	closed     bool
	doneClosed bool
	done       chan struct{}
}

// NewGroup derives task lifetime from parent and validates a hard concurrency
// bound. Parent must contain no installed scope, including ScopeUnscoped, so one
// submitted tenant cannot become another task's base identity. Its cancellation
// and deadline own group lifetime; the group owns the derived cancellation
// function. Call Drain or Shutdown to finish ownership even after a wait timeout.
func NewGroup(parent context.Context, options GroupOptions) (*Group, error) {
	if parent == nil || options.MaxConcurrent <= 0 || options.MaxConcurrent > maximumGroupConcurrency {
		return nil, ErrInvalidGroup
	}
	if _, scoped := ScopeFromContext(parent); scoped {
		return nil, ErrInvalidGroup
	}
	ctx, cancel := context.WithCancel(parent)
	return &Group{
		ctx: ctx, cancel: cancel, semaphore: make(chan struct{}, options.MaxConcurrent),
		handleError: options.HandleError, done: make(chan struct{}),
	}, nil
}

// Submit validates and installs scope synchronously before acquiring bounded
// capacity. The task preserves submitCtx values, deadline, and cancellation
// while also remaining owned and cancellable by the group lifetime. A nil error
// acknowledges admission, not completion or task success; errors are delivered
// only to HandleError. Conflicting scope, invalid inputs and observed cancellation
// refuse admission. No task is retried, and calls/operations must not wait for
// their own group's completion.
func (group *Group) Submit(
	submitCtx context.Context,
	scope Scope,
	operation func(context.Context) error,
) error {
	if group == nil || group.ctx == nil || submitCtx == nil {
		return ErrInvalidGroup
	}
	if group.isClosed() {
		return ErrGroupClosed
	}
	if !scope.Valid() || operation == nil {
		return ErrInvalidOperation
	}
	scopedSubmitCtx, err := WithScope(submitCtx, scope)
	switch err {
	case nil:
	default:
		return err
	}
	select {
	case <-submitCtx.Done():
		return submitCtx.Err()
	default:
	}
	select {
	case <-group.ctx.Done():
		return group.stopError()
	default:
	}
	select {
	case group.semaphore <- struct{}{}:
	case <-submitCtx.Done():
		return submitCtx.Err()
	case <-group.ctx.Done():
		return group.stopError()
	}
	switch err := submitCtx.Err(); err {
	case nil:
	default:
		<-group.semaphore
		return err
	}
	switch err := group.ctx.Err(); err {
	case nil:
	default:
		<-group.semaphore
		return group.stopError()
	}

	group.mutex.Lock()
	if group.closed {
		group.mutex.Unlock()
		<-group.semaphore
		return ErrGroupClosed
	}
	group.active++
	group.mutex.Unlock()

	go group.run(scopedSubmitCtx, scope, operation)
	return nil
}

// Drain stops new submissions and waits for active tasks without cancelling
// them. Waiting observes ctx. Drain is safe for concurrent and repeated calls;
// once accepted work completes, it releases the group-owned task context. A
// concurrent Shutdown cancels shared tasks while Drain continues waiting for
// the same terminal state. A wait timeout leaves intake closed and accepted work
// active; eventual completion still releases the task context. Drain reports wait
// status only, not task errors, and cannot force an uncooperative task to return.
func (group *Group) Drain(ctx context.Context) error {
	if group == nil || group.ctx == nil || ctx == nil {
		return ErrInvalidGroup
	}
	group.beginClose()
	if err := group.wait(ctx); err != nil {
		return err
	}
	group.cancel()
	return nil
}

// Close delegates to Drain and preserves its graceful completion behavior.
//
// Deprecated: use Drain.
func (group *Group) Close(ctx context.Context) error {
	return group.Drain(ctx)
}

// Shutdown stops new submissions, cancels active tasks, and waits for their
// return. Waiting observes ctx, while task cancellation uses the group context.
// Concurrent and repeated lifecycle calls join the same terminal state; any
// Shutdown makes an active Drain or Close forceful by cancelling shared tasks.
// A wait timeout does not mean tasks have returned; call again to join them.
// Once terminal completion wins, repeated lifecycle waits return nil even with
// a canceled wait context. Task errors are not returned by lifecycle methods.
func (group *Group) Shutdown(ctx context.Context) error {
	if group == nil || group.ctx == nil || ctx == nil {
		return ErrInvalidGroup
	}
	group.beginClose()
	group.cancel()
	return group.wait(ctx)
}

func (group *Group) run(
	submitCtx context.Context,
	scope Scope,
	operation func(context.Context) error,
) {
	defer group.complete()
	taskContext, cancel := context.WithCancel(submitCtx)
	stopGroupCancellation := context.AfterFunc(group.ctx, cancel)
	defer func() {
		stopGroupCancellation()
		cancel()
	}()
	err := operation(taskContext)
	if err != nil && group.handleError != nil {
		group.handleError(scope, err)
	}
}

func (group *Group) complete() {
	<-group.semaphore
	group.mutex.Lock()
	group.active--
	group.closeDoneLocked()
	terminal := group.closed && group.active == 0
	group.mutex.Unlock()
	if terminal {
		group.cancel()
	}
}

func (group *Group) beginClose() {
	group.mutex.Lock()
	group.closed = true
	group.closeDoneLocked()
	group.mutex.Unlock()
}

func (group *Group) closeDoneLocked() {
	if group.closed && group.active == 0 && !group.doneClosed {
		close(group.done)
		group.doneClosed = true
	}
}

func (group *Group) wait(ctx context.Context) error {
	select {
	case <-group.done:
		return nil
	default:
	}
	select {
	case <-group.done:
		return nil
	case <-ctx.Done():
		select {
		case <-group.done:
			return nil
		default:
			return ctx.Err()
		}
	}
}

func (group *Group) isClosed() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.closed
}

func (group *Group) stopError() error {
	if group.isClosed() {
		return ErrGroupClosed
	}
	return group.ctx.Err()
}
