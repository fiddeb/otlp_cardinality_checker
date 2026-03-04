package memory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

func newTestStore(maxWatched int) *Store {
	return NewWithAutoTemplate(false, maxWatched)
}

// ---------------------------------------------------------------------------
// WatchAttribute
// ---------------------------------------------------------------------------

func TestWatchAttribute_HappyPath(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	if err := s.WatchAttribute(ctx, "http.method"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWatchAttribute_EmptyKey(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	if err := s.WatchAttribute(ctx, ""); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestWatchAttribute_Idempotent(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)

	if err := s.WatchAttribute(ctx, "k1"); err != nil {
		t.Fatal(err)
	}
	// Second call on same key must succeed and not double-count toward limit.
	if err := s.WatchAttribute(ctx, "k1"); err != nil {
		t.Fatalf("idempotent re-watch failed: %v", err)
	}
	list, _ := s.ListWatchedAttributes(ctx)
	if len(list) != 1 {
		t.Errorf("expected 1 watched attribute, got %d", len(list))
	}
}

func TestWatchAttribute_LimitEnforced(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(3)

	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		if err := s.WatchAttribute(ctx, key); err != nil {
			t.Fatalf("watch %q failed: %v", key, err)
		}
	}

	err := s.WatchAttribute(ctx, "z")
	if err == nil {
		t.Fatal("expected limit error, got nil")
	}
}

// ---------------------------------------------------------------------------
// UnwatchAttribute
// ---------------------------------------------------------------------------

func TestUnwatchAttribute_DeactivatesButPreservesEntry(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	_ = s.WatchAttribute(ctx, "k1")
	_ = s.StoreAttributeValue(ctx, "k1", "v1", "metrics", "")
	if err := s.UnwatchAttribute(ctx, "k1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Entry must still exist.
	w, err := s.GetWatchedAttribute(ctx, "k1")
	if err != nil {
		t.Fatalf("entry removed after unwatch: %v", err)
	}
	// Must be inactive.
	_, _, _, _, active, _, _ := w.Snapshot()
	if active {
		t.Error("expected inactive after unwatch")
	}
	// Values must be preserved.
	_, vals, _, _, _, _, _ := w.Snapshot()
	if len(vals) == 0 {
		t.Error("values must be preserved after unwatch")
	}
}

func TestUnwatchAttribute_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	err := s.UnwatchAttribute(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, models.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestUnwatchAttribute_EmptyKey(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	if err := s.UnwatchAttribute(ctx, ""); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestWatchAttribute_RewatchPreservesValues(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	_ = s.WatchAttribute(ctx, "env")
	_ = s.StoreAttributeValue(ctx, "env", "prod", "metrics", "")
	_ = s.StoreAttributeValue(ctx, "env", "staging", "metrics", "")
	_ = s.UnwatchAttribute(ctx, "env")

	// Re-watch must succeed and values must still be there.
	if err := s.WatchAttribute(ctx, "env"); err != nil {
		t.Fatalf("re-watch failed: %v", err)
	}
	w, _ := s.GetWatchedAttribute(ctx, "env")
	_, _, uniqueCount, totalObs, active, _, _ := w.Snapshot()
	if !active {
		t.Error("expected active after re-watch")
	}
	if uniqueCount != 2 || totalObs != 2 {
		t.Errorf("expected 2 unique / 2 total, got %d / %d", uniqueCount, totalObs)
	}
}

func TestWatchAttribute_InactiveDoesNotCountTowardLimit(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(2)
	_ = s.WatchAttribute(ctx, "k1")
	_ = s.WatchAttribute(ctx, "k2")
	_ = s.UnwatchAttribute(ctx, "k1") // k1 inactive, frees active slot

	// Adding a new key must succeed because only 1 active watch exists.
	if err := s.WatchAttribute(ctx, "k3"); err != nil {
		t.Errorf("expected success when inactive entry frees slot: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetWatchedAttribute
// ---------------------------------------------------------------------------

func TestGetWatchedAttribute_Found(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	_ = s.WatchAttribute(ctx, "k1")
	w, err := s.GetWatchedAttribute(ctx, "k1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil || w.Key != "k1" {
		t.Errorf("unexpected attribute: %+v", w)
	}
}

func TestGetWatchedAttribute_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	_, err := s.GetWatchedAttribute(ctx, "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, models.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListWatchedAttributes
// ---------------------------------------------------------------------------

func TestListWatchedAttributes_Empty(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	list, err := s.ListWatchedAttributes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0, got %d", len(list))
	}
}

func TestListWatchedAttributes_Multiple(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	keys := []string{"a", "b", "c"}
	for _, k := range keys {
		_ = s.WatchAttribute(ctx, k)
	}
	list, err := s.ListWatchedAttributes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != len(keys) {
		t.Errorf("expected %d entries, got %d", len(keys), len(list))
	}
}

// ---------------------------------------------------------------------------
// MergeWatchedAttribute
// ---------------------------------------------------------------------------

func TestMergeWatchedAttribute_NilNoOp(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	if err := s.MergeWatchedAttribute(ctx, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeWatchedAttribute_InsertsAsInactive(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	w := models.NewWatchedAttribute("env", 100)
	w.AddValue("prod")
	w.SetActive(true) // will be reset by merge

	if err := s.MergeWatchedAttribute(ctx, w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.GetWatchedAttribute(ctx, "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _, uniqueCount, _, active, _, _ := got.Snapshot()
	if active {
		t.Error("merged attribute must be inactive")
	}
	if uniqueCount == 0 {
		t.Error("merged values must be preserved")
	}
}

// ---------------------------------------------------------------------------
// Hot path: StoreAttributeValue feeds watched attribute
// ---------------------------------------------------------------------------

func TestStoreAttributeValue_FeedsWatchedAttribute(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	_ = s.WatchAttribute(ctx, "http.status_code")

	_ = s.StoreAttributeValue(ctx, "http.status_code", "200", "metrics", "")
	_ = s.StoreAttributeValue(ctx, "http.status_code", "200", "metrics", "")
	_ = s.StoreAttributeValue(ctx, "http.status_code", "404", "metrics", "")

	w, err := s.GetWatchedAttribute(ctx, "http.status_code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _, uniqueCount, totalObs, _, _, _ := w.Snapshot()
	if totalObs != 3 {
		t.Errorf("expected 3 total observations, got %d", totalObs)
	}
	if uniqueCount != 2 {
		t.Errorf("expected 2 unique values, got %d", uniqueCount)
	}
}

func TestStoreAttributeValue_UnwatchedKeyIgnored(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)

	_ = s.StoreAttributeValue(ctx, "some.other.key", "value", "metrics", "")

	list, _ := s.ListWatchedAttributes(ctx)
	if len(list) != 0 {
		t.Errorf("expected 0 watched entries, got %d", len(list))
	}
}

func TestStoreAttributeValue_InactiveWatchIgnored(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)

	// Merge an inactive attribute.
	w := models.NewWatchedAttribute("env", 100)
	w.SetActive(false)
	_ = s.MergeWatchedAttribute(ctx, w)

	_ = s.StoreAttributeValue(ctx, "env", "staging", "metrics", "")

	got, _ := s.GetWatchedAttribute(ctx, "env")
	_, _, uniqueCount, totalObs, _, _, _ := got.Snapshot()
	if totalObs != 0 || uniqueCount != 0 {
		t.Errorf("inactive watch must not record values: totalObs=%d uniqueCount=%d", totalObs, uniqueCount)
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestWatchAttribute_ConcurrentStoreAttributeValue(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)
	_ = s.WatchAttribute(ctx, "region")

	const goroutines = 50
	const writes = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < writes; j++ {
				_ = s.StoreAttributeValue(ctx, "region", "us-east", "metrics", "")
			}
		}()
	}
	wg.Wait()

	w, err := s.GetWatchedAttribute(ctx, "region")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _, _, totalObs, _, _, _ := w.Snapshot()
	expected := int64(goroutines * writes)
	if totalObs != expected {
		t.Errorf("expected %d total observations, got %d", expected, totalObs)
	}
}
