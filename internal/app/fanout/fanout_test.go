package fanout_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/app/fanout"
)

func TestRun_EmptyItems(t *testing.T) {
	t.Parallel()

	results := fanout.Run(context.Background(), 5, []int{}, func(_ context.Context, _ int) (string, error) {
		t.Fatal("fn should not be called for empty items")
		return "", nil
	})

	if results == nil {
		t.Fatal("expected non-nil slice for empty items")
	}
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
}

func TestRun_AllSucceed(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3, 4, 5}

	results := fanout.Run(context.Background(), 3, items, func(_ context.Context, n int) (int, error) {
		return n * 10, nil
	})

	if len(results) != len(items) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(items))
	}

	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
		want := items[i] * 10
		if r.Value != want {
			t.Errorf("results[%d].Value = %d, want %d", i, r.Value, want)
		}
	}
}

func TestRun_PartialFailure(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	items := []int{1, 2, 3}

	results := fanout.Run(context.Background(), 3, items, func(_ context.Context, n int) (int, error) {
		if n == 2 {
			return 0, errBoom
		}
		return n * 10, nil
	})

	// Item 0 (value 1): success
	if results[0].Err != nil || results[0].Value != 10 {
		t.Errorf("results[0] = {%d, %v}, want {10, nil}", results[0].Value, results[0].Err)
	}

	// Item 1 (value 2): failure
	if !errors.Is(results[1].Err, errBoom) {
		t.Errorf("results[1].Err = %v, want %v", results[1].Err, errBoom)
	}

	// Item 2 (value 3): success
	if results[2].Err != nil || results[2].Value != 30 {
		t.Errorf("results[2] = {%d, %v}, want {30, nil}", results[2].Value, results[2].Err)
	}
}

func TestRun_PreservesInputOrder(t *testing.T) {
	t.Parallel()

	// Items with varying delays to encourage out-of-order completion.
	items := []time.Duration{
		30 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
	}

	results := fanout.Run(context.Background(), 3, items, func(_ context.Context, d time.Duration) (time.Duration, error) {
		time.Sleep(d)
		return d, nil
	})

	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v", i, r.Err)
		}
		if r.Value != items[i] {
			t.Errorf("results[%d].Value = %v, want %v", i, r.Value, items[i])
		}
	}
}

func TestRun_BoundedConcurrency(t *testing.T) {
	t.Parallel()

	const maxWorkers = 3
	const totalItems = 15

	var peak atomic.Int32
	var active atomic.Int32

	items := make([]int, totalItems)
	for i := range items {
		items[i] = i
	}

	results := fanout.Run(context.Background(), maxWorkers, items, func(_ context.Context, _ int) (int, error) {
		cur := active.Add(1)
		defer active.Add(-1)

		// Track peak concurrency with CAS loop.
		for {
			p := peak.Load()
			if cur <= p || peak.CompareAndSwap(p, cur) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
		return 0, nil
	})

	if len(results) != totalItems {
		t.Fatalf("got %d results, want %d", len(results), totalItems)
	}
	if p := peak.Load(); p > maxWorkers {
		t.Fatalf("peak concurrency %d exceeded maxWorkers %d", p, maxWorkers)
	}
}

func TestRun_ContextCancellation_BeforeAcquire(t *testing.T) {
	t.Parallel()

	// Use 1 worker with 3 items. Cancel while items are waiting for the semaphore.
	ctx, cancel := context.WithCancel(context.Background())

	items := []int{1, 2, 3}
	var started atomic.Int32

	results := fanout.Run(ctx, 1, items, func(ctx context.Context, n int) (int, error) {
		started.Add(1)
		if n == 1 {
			// First item: cancel context then block briefly so others see cancellation.
			cancel()
			time.Sleep(50 * time.Millisecond)
		}
		return n, nil
	})

	// At least one item should have a context.Canceled error.
	var canceledCount int
	for _, r := range results {
		if errors.Is(r.Err, context.Canceled) {
			canceledCount++
		}
	}

	if canceledCount == 0 {
		t.Error("expected at least one result with context.Canceled error")
	}
}

func TestRun_ContextCancellation_DuringExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	items := []int{1}

	results := fanout.Run(ctx, 1, items, func(ctx context.Context, _ int) (int, error) {
		cancel()
		// fn should see the canceled context.
		return 0, ctx.Err()
	})

	if !errors.Is(results[0].Err, context.Canceled) {
		t.Errorf("results[0].Err = %v, want context.Canceled", results[0].Err)
	}
}

func TestRun_MaxWorkersExceedsItems(t *testing.T) {
	t.Parallel()

	items := []int{1, 2}

	results := fanout.Run(context.Background(), 100, items, func(_ context.Context, n int) (int, error) {
		return n * 2, nil
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Value != 2 || results[1].Value != 4 {
		t.Errorf("results = [%d, %d], want [2, 4]", results[0].Value, results[1].Value)
	}
}
