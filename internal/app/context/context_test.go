package appctx

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

const testFetchValue = "hello"

// testAction is a test implementation of Action that records calls and
// optionally returns errors.
type testAction struct {
	desc        string
	executed    bool
	rolledBack  bool
	executeErr  error
	rollbackErr error
	executeFn   func(ctx context.Context) error
	order       *[]string // shared slice to record execution/rollback order
}

func (a *testAction) Execute(ctx context.Context) error {
	if a.executeFn != nil {
		return a.executeFn(ctx)
	}
	if a.executeErr != nil {
		return a.executeErr
	}
	a.executed = true
	if a.order != nil {
		*a.order = append(*a.order, "execute:"+a.desc)
	}
	return nil
}

func (a *testAction) Rollback(ctx context.Context) error {
	a.rolledBack = true
	if a.order != nil {
		*a.order = append(*a.order, "rollback:"+a.desc)
	}
	return a.rollbackErr
}

func (a *testAction) Description() string { return a.desc }

// --- GetOrFetch tests ---

func TestGetOrFetch_CacheMiss(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	calls := 0

	val, err := GetOrFetch(rc, "key", func(_ context.Context) (string, error) {
		calls++
		return testFetchValue, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != testFetchValue {
		t.Fatalf("got %q, want %q", val, testFetchValue)
	}
	if calls != 1 {
		t.Fatalf("fetchFn called %d times, want 1", calls)
	}
}

func TestGetOrFetch_CacheHit(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	calls := 0

	fetchFn := func(_ context.Context) (string, error) {
		calls++
		return testFetchValue, nil
	}

	_, _ = GetOrFetch(rc, "key", fetchFn)
	val, err := GetOrFetch(rc, "key", fetchFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != testFetchValue {
		t.Fatalf("got %q, want %q", val, testFetchValue)
	}
	if calls != 1 {
		t.Fatalf("fetchFn called %d times, want 1", calls)
	}
}

func TestGetOrFetch_CachesErrors(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	calls := 0
	fetchErr := errors.New("fetch failed")

	fetchFn := func(_ context.Context) (string, error) {
		calls++
		return "", fetchErr
	}

	_, _ = GetOrFetch(rc, "key", fetchFn)
	val, err := GetOrFetch(rc, "key", fetchFn)

	if !errors.Is(err, fetchErr) {
		t.Fatalf("got error %v, want %v", err, fetchErr)
	}
	if val != "" {
		t.Fatalf("got %q, want empty string", val)
	}
	if calls != 1 {
		t.Fatalf("fetchFn called %d times, want 1", calls)
	}
}

func TestGetOrFetch_DifferentKeys(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	v1, _ := GetOrFetch(rc, "a", func(_ context.Context) (int, error) { return 1, nil })
	v2, _ := GetOrFetch(rc, "b", func(_ context.Context) (int, error) { return 2, nil })

	if v1 != 1 {
		t.Fatalf("key a: got %d, want 1", v1)
	}
	if v2 != 2 {
		t.Fatalf("key b: got %d, want 2", v2)
	}
}

func TestGetOrFetch_ZeroValue(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	val, err := GetOrFetch(rc, "key", func(_ context.Context) (int, error) { return 0, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Fatalf("got %d, want 0", val)
	}

	// Second call should return cached zero value.
	calls := 0
	val, err = GetOrFetch(rc, "key", func(_ context.Context) (int, error) {
		calls++
		return 99, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Fatalf("got %d, want cached 0", val)
	}
	if calls != 0 {
		t.Fatalf("fetchFn should not be called on cache hit")
	}
}

// --- DataProvider tests ---

func TestDataProvider_Get(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	p := NewDataProvider("todo:1", func(_ context.Context) (string, error) {
		return "buy milk", nil
	})

	val, err := p.Get(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "buy milk" {
		t.Fatalf("got %q, want %q", val, "buy milk")
	}
}

func TestDataProvider_Memoizes(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	calls := 0

	p := NewDataProvider("key", func(_ context.Context) (int, error) {
		calls++
		return 42, nil
	})

	_, _ = p.Get(rc)
	val, _ := p.Get(rc)

	if val != 42 {
		t.Fatalf("got %d, want 42", val)
	}
	if calls != 1 {
		t.Fatalf("fetchFn called %d times, want 1", calls)
	}
}

// --- AddAction / AddGroup tests ---

func TestAddAction_StagesAction(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	err := rc.AddAction(&testAction{desc: "a1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rc.items) != 1 {
		t.Fatalf("got %d items, want 1", len(rc.items))
	}
}

func TestAddAction_AfterCommit(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	_ = rc.Commit(context.Background())

	err := rc.AddAction(&testAction{desc: "late"})
	if !errors.Is(err, ErrAlreadyCommitted) {
		t.Fatalf("got %v, want ErrAlreadyCommitted", err)
	}
}

func TestAddGroup_StagesGroup(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	err := rc.AddGroup(&testAction{desc: "g1"}, &testAction{desc: "g2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rc.items) != 1 {
		t.Fatalf("got %d items, want 1", len(rc.items))
	}
}

func TestAddGroup_AfterCommit(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	_ = rc.Commit(context.Background())

	err := rc.AddGroup(&testAction{desc: "late"})
	if !errors.Is(err, ErrAlreadyCommitted) {
		t.Fatalf("got %v, want ErrAlreadyCommitted", err)
	}
}

// --- Commit tests ---

func TestCommit_EmptyQueue(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	err := rc.Commit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rc.committed {
		t.Fatal("expected committed to be true")
	}
}

func TestCommit_SingleActionSuccess(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	a := &testAction{desc: "a1"}
	_ = rc.AddAction(a)

	err := rc.Commit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.executed {
		t.Fatal("action was not executed")
	}
	if a.rolledBack {
		t.Fatal("action should not be rolled back on success")
	}
}

func TestCommit_MultipleActionsSuccess(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	var order []string
	a1 := &testAction{desc: "a1", order: &order}
	a2 := &testAction{desc: "a2", order: &order}
	a3 := &testAction{desc: "a3", order: &order}

	_ = rc.AddAction(a1)
	_ = rc.AddAction(a2)
	_ = rc.AddAction(a3)

	err := rc.Commit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"execute:a1", "execute:a2", "execute:a3"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestCommit_FailureTriggersRollback(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	var order []string

	a1 := &testAction{desc: "a1", order: &order}
	a2 := &testAction{desc: "a2", order: &order}
	a3 := &testAction{desc: "a3", order: &order, executeErr: errors.New("boom")}

	_ = rc.AddAction(a1)
	_ = rc.AddAction(a2)
	_ = rc.AddAction(a3)

	err := rc.Commit(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	// a3 fails during execute (order tracking skipped because executeErr is returned before order append)
	// but a1 and a2 should be rolled back in reverse order.
	want := []string{"execute:a1", "execute:a2", "rollback:a2", "rollback:a1"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestCommit_RollbackErrorDoesNotStopRollback(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	a1 := &testAction{desc: "a1"}
	a2 := &testAction{desc: "a2", rollbackErr: errors.New("rollback failed")}
	a3 := &testAction{desc: "a3", executeErr: errors.New("boom")}

	_ = rc.AddAction(a1)
	_ = rc.AddAction(a2)
	_ = rc.AddAction(a3)

	_ = rc.Commit(context.Background())

	// Both a1 and a2 should have been rolled back despite a2's rollback error.
	if !a1.rolledBack {
		t.Fatal("a1 should be rolled back")
	}
	if !a2.rolledBack {
		t.Fatal("a2 should be rolled back despite rollback error")
	}
}

func TestCommit_CalledTwice(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	_ = rc.Commit(context.Background())

	err := rc.Commit(context.Background())
	if !errors.Is(err, ErrAlreadyCommitted) {
		t.Fatalf("got %v, want ErrAlreadyCommitted", err)
	}
}

func TestCommit_FailureMarksCommitted(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	_ = rc.AddAction(&testAction{desc: "fail", executeErr: errors.New("boom")})

	_ = rc.Commit(context.Background())

	if !rc.committed {
		t.Fatal("expected committed to be true after failed commit")
	}
}

func TestCommit_ErrorWrapsDescription(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	_ = rc.AddAction(&testAction{desc: "send email", executeErr: errors.New("smtp timeout")})

	err := rc.Commit(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	msg := err.Error()
	if msg != "executing send email: smtp timeout" {
		t.Fatalf("got %q, want error message containing action description", msg)
	}
}

func TestCommit_ExecutionOrder(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	var order []string
	for i := range 5 {
		desc := string(rune('a' + i))
		_ = rc.AddAction(&testAction{desc: desc, order: &order})
	}

	_ = rc.Commit(context.Background())

	want := []string{"execute:a", "execute:b", "execute:c", "execute:d", "execute:e"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestCommit_RollbackOrder(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())
	var order []string

	for i := range 4 {
		desc := string(rune('a' + i))
		a := &testAction{desc: desc, order: &order}
		if i == 3 {
			a.executeErr = errors.New("fail")
		}
		_ = rc.AddAction(a)
	}

	_ = rc.Commit(context.Background())

	// Execute a, b, c; d fails; rollback c, b, a.
	want := []string{
		"execute:a", "execute:b", "execute:c",
		"rollback:c", "rollback:b", "rollback:a",
	}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

// --- ActionGroup tests ---

func TestActionGroup_ParallelExecution(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	var count atomic.Int32
	makeAction := func(desc string) *testAction {
		return &testAction{
			desc: desc,
			executeFn: func(_ context.Context) error {
				count.Add(1)
				return nil
			},
		}
	}

	_ = rc.AddGroup(makeAction("g1"), makeAction("g2"), makeAction("g3"))
	err := rc.Commit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count.Load() != 3 {
		t.Fatalf("got %d executions, want 3", count.Load())
	}
}

func TestActionGroup_PartialFailure(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	succeeded := &testAction{desc: "ok"}
	failed := &testAction{desc: "fail", executeErr: errors.New("boom")}

	// Use executeFn for succeeded to ensure it completes before the failure.
	succeeded.executeFn = func(_ context.Context) error {
		succeeded.executed = true
		return nil
	}

	_ = rc.AddGroup(succeeded, failed)
	err := rc.Commit(context.Background())

	if err == nil {
		t.Fatal("expected error from group")
	}
	if succeeded.executed && !succeeded.rolledBack {
		t.Fatal("successfully executed action in group should be rolled back")
	}
}

func TestActionGroup_CancelsInProgress(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	var ctxCancelled atomic.Bool

	slow := &testAction{
		desc: "slow",
		executeFn: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				ctxCancelled.Store(true)
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	}
	fast := &testAction{desc: "fast-fail", executeErr: errors.New("boom")}

	_ = rc.AddGroup(slow, fast)
	_ = rc.Commit(context.Background())

	// Give the slow action a moment to observe cancellation.
	time.Sleep(50 * time.Millisecond)
	if !ctxCancelled.Load() {
		t.Fatal("expected slow action's context to be canceled")
	}
}

func TestActionGroup_EmptyGroup(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	_ = rc.AddGroup()
	err := rc.Commit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionGroup_Description_Multiple(t *testing.T) {
	t.Parallel()
	g := &actionGroup{
		actions: []Action{
			&testAction{desc: "first"},
			&testAction{desc: "second"},
		},
	}

	desc := g.description()
	want := "action group (2 actions: first, ...)"
	if desc != want {
		t.Fatalf("got %q, want %q", desc, want)
	}
}

func TestActionGroup_Description_Single(t *testing.T) {
	t.Parallel()
	g := &actionGroup{actions: []Action{&testAction{desc: "only"}}}

	if g.description() != "only" {
		t.Fatalf("got %q, want %q", g.description(), "only")
	}
}

func TestActionGroup_Description_Empty(t *testing.T) {
	t.Parallel()
	g := &actionGroup{}

	if g.description() != "empty action group" {
		t.Fatalf("got %q, want %q", g.description(), "empty action group")
	}
}

// --- Mixed action and group tests ---

func TestCommit_ActionAndGroupMixed(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	var count atomic.Int32
	makeCountAction := func(desc string) *testAction {
		return &testAction{
			desc: desc,
			executeFn: func(_ context.Context) error {
				count.Add(1)
				return nil
			},
		}
	}

	_ = rc.AddAction(makeCountAction("a1"))
	_ = rc.AddGroup(makeCountAction("g1"), makeCountAction("g2"))
	_ = rc.AddAction(makeCountAction("a2"))

	err := rc.Commit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count.Load() != 4 {
		t.Fatalf("got %d executions, want 4", count.Load())
	}
}

func TestCommit_GroupFailureRollbacksPriorItems(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	a1 := &testAction{desc: "a1"}
	a2 := &testAction{desc: "a2"}
	groupFail := &testAction{desc: "group-fail", executeErr: errors.New("boom")}

	_ = rc.AddAction(a1)
	_ = rc.AddAction(a2)
	_ = rc.AddGroup(groupFail)

	err := rc.Commit(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	if !a1.rolledBack {
		t.Fatal("a1 should be rolled back")
	}
	if !a2.rolledBack {
		t.Fatal("a2 should be rolled back")
	}
}

// --- Edge case tests ---

func TestNew_ReturnsEmptyContext(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	if len(rc.cache) != 0 {
		t.Fatalf("cache should be empty, got %d entries", len(rc.cache))
	}
	if len(rc.items) != 0 {
		t.Fatalf("items should be empty, got %d entries", len(rc.items))
	}
	if rc.committed {
		t.Fatal("committed should be false")
	}
}

func TestGetOrFetch_WorksAfterCommit(t *testing.T) {
	t.Parallel()
	rc := New(context.Background())

	_, _ = GetOrFetch(rc, "pre", func(_ context.Context) (string, error) {
		return "before", nil
	})
	_ = rc.Commit(context.Background())

	// GetOrFetch should still work after commit (cache is read-only).
	val, err := GetOrFetch(rc, "pre", func(_ context.Context) (string, error) {
		return "should not be called", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "before" {
		t.Fatalf("got %q, want cached value %q", val, "before")
	}

	// New keys can still be fetched after commit.
	val, err = GetOrFetch(rc, "post", func(_ context.Context) (string, error) {
		return "after", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "after" {
		t.Fatalf("got %q, want %q", val, "after")
	}
}
