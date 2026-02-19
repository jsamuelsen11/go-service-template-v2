package health_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/health"
	"github.com/jsamuelsen11/go-service-template-v2/mocks"
)

func TestCheckAll_Empty(t *testing.T) {
	t.Parallel()

	r := health.New()
	results := r.CheckAll(context.Background())

	if results == nil {
		t.Fatal("expected non-nil map, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected empty map, got %d entries", len(results))
	}
}

func TestCheckAll_AllHealthy(t *testing.T) {
	t.Parallel()

	checkerA := mocks.NewMockHealthChecker(t)
	checkerA.EXPECT().Name().Return("db")
	checkerA.EXPECT().HealthCheck(mock.Anything).Return(nil)

	checkerB := mocks.NewMockHealthChecker(t)
	checkerB.EXPECT().Name().Return("cache")
	checkerB.EXPECT().HealthCheck(mock.Anything).Return(nil)

	r := health.New()
	r.Register(checkerA)
	r.Register(checkerB)

	results := r.CheckAll(context.Background())

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results["db"] != nil {
		t.Errorf("db check = %v, want nil", results["db"])
	}
	if results["cache"] != nil {
		t.Errorf("cache check = %v, want nil", results["cache"])
	}
}

func TestCheckAll_MixedHealth(t *testing.T) {
	t.Parallel()

	healthy := mocks.NewMockHealthChecker(t)
	healthy.EXPECT().Name().Return("db")
	healthy.EXPECT().HealthCheck(mock.Anything).Return(nil)

	unhealthyErr := errors.New("connection refused")
	unhealthy := mocks.NewMockHealthChecker(t)
	unhealthy.EXPECT().Name().Return("todo-api")
	unhealthy.EXPECT().HealthCheck(mock.Anything).Return(unhealthyErr)

	r := health.New()
	r.Register(healthy)
	r.Register(unhealthy)

	results := r.CheckAll(context.Background())

	if results["db"] != nil {
		t.Errorf("db check = %v, want nil", results["db"])
	}
	if results["todo-api"] == nil {
		t.Fatal("todo-api check = nil, want error")
	}
	if results["todo-api"].Error() != "connection refused" {
		t.Errorf("todo-api check = %q, want %q", results["todo-api"].Error(), "connection refused")
	}
}

func TestCheckAll_ContextPropagated(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := mocks.NewMockHealthChecker(t)
	checker.EXPECT().Name().Return("todo-api")
	checker.EXPECT().HealthCheck(mock.MatchedBy(func(ctx context.Context) bool {
		return ctx.Err() != nil
	})).Return(context.Canceled)

	r := health.New()
	r.Register(checker)

	results := r.CheckAll(ctx)

	if !errors.Is(results["todo-api"], context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", results["todo-api"])
	}
}

func TestCheckAll_DuplicateNames_LastWriteWins(t *testing.T) {
	t.Parallel()

	first := mocks.NewMockHealthChecker(t)
	first.EXPECT().Name().Return("db")
	first.EXPECT().HealthCheck(mock.Anything).Return(nil)

	secondErr := errors.New("second failure")
	second := mocks.NewMockHealthChecker(t)
	second.EXPECT().Name().Return("db")
	second.EXPECT().HealthCheck(mock.Anything).Return(secondErr)

	r := health.New()
	r.Register(first)
	r.Register(second)

	results := r.CheckAll(context.Background())

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	got, ok := results["db"]
	if !ok {
		t.Fatal(`expected result for key "db", but it was missing`)
	}
	if !errors.Is(got, secondErr) {
		t.Errorf("db check = %v, want %v (from last registered checker)", got, secondErr)
	}
}

func TestCheckAll_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	r := health.New()

	var wg sync.WaitGroup
	const goroutines = 50

	// Half the goroutines register checkers, half call CheckAll.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func() {
				defer wg.Done()
				c := mocks.NewMockHealthChecker(t)
				c.EXPECT().Name().Return("checker").Maybe()
				c.EXPECT().HealthCheck(mock.Anything).Return(nil).Maybe()
				r.Register(c)
			}()
		} else {
			go func() {
				defer wg.Done()
				r.CheckAll(context.Background())
			}()
		}
	}

	wg.Wait()
}
