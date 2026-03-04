package models

import (
	"sync"
	"testing"
	"time"
)

func TestNewWatchedAttribute(t *testing.T) {
	w := NewWatchedAttribute("my.key", 100)
	if w.Key != "my.key" {
		t.Errorf("Key = %q, want %q", w.Key, "my.key")
	}
	if !w.Active {
		t.Error("expected Active = true on new watch")
	}
	if w.MaxValues != 100 {
		t.Errorf("MaxValues = %d, want 100", w.MaxValues)
	}
	if w.WatchingSince.IsZero() {
		t.Error("WatchingSince should be set")
	}
	if w.Values == nil {
		t.Error("Values map should be initialized")
	}
}

func TestNewWatchedAttribute_DefaultMaxValues(t *testing.T) {
	w := NewWatchedAttribute("k", 0)
	if w.MaxValues != 10000 {
		t.Errorf("MaxValues = %d, want 10000", w.MaxValues)
	}
}

func TestWatchedAttribute_AddValue(t *testing.T) {
	tests := []struct {
		name               string
		values             []string
		wantUniqueCount    int64
		wantTotalObs       int64
		wantOverflow       bool
		wantCountForValue  map[string]int64
	}{
		{
			name:            "single value multiple times",
			values:          []string{"a", "a", "a"},
			wantUniqueCount: 1,
			wantTotalObs:    3,
			wantOverflow:    false,
			wantCountForValue: map[string]int64{"a": 3},
		},
		{
			name:            "distinct values",
			values:          []string{"x", "y", "z"},
			wantUniqueCount: 3,
			wantTotalObs:    3,
			wantOverflow:    false,
			wantCountForValue: map[string]int64{"x": 1, "y": 1, "z": 1},
		},
		{
			name:            "mix unique and repeated",
			values:          []string{"a", "b", "a", "c", "b"},
			wantUniqueCount: 3,
			wantTotalObs:    5,
			wantOverflow:    false,
			wantCountForValue: map[string]int64{"a": 2, "b": 2, "c": 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWatchedAttribute("k", 10000)
			for _, v := range tc.values {
				w.AddValue(v)
			}
			if w.UniqueCount != tc.wantUniqueCount {
				t.Errorf("UniqueCount = %d, want %d", w.UniqueCount, tc.wantUniqueCount)
			}
			if w.TotalObservations != tc.wantTotalObs {
				t.Errorf("TotalObservations = %d, want %d", w.TotalObservations, tc.wantTotalObs)
			}
			if w.Overflow != tc.wantOverflow {
				t.Errorf("Overflow = %v, want %v", w.Overflow, tc.wantOverflow)
			}
			for val, wantCount := range tc.wantCountForValue {
				if got := w.Values[val]; got != wantCount {
					t.Errorf("Values[%q] = %d, want %d", val, got, wantCount)
				}
			}
		})
	}
}

func TestWatchedAttribute_Overflow(t *testing.T) {
	maxVals := 5
	w := NewWatchedAttribute("k", maxVals)

	// Fill to limit
	for i := 0; i < maxVals; i++ {
		w.AddValue(string(rune('a' + i))) // a, b, c, d, e
	}
	if w.Overflow {
		t.Error("should not overflow before limit")
	}
	if w.UniqueCount != int64(maxVals) {
		t.Errorf("UniqueCount = %d, want %d", w.UniqueCount, maxVals)
	}

	// One more unique value triggers overflow
	w.AddValue("z")
	if !w.Overflow {
		t.Error("expected overflow after exceeding limit")
	}
	// "z" should NOT be in the map
	if _, ok := w.Values["z"]; ok {
		t.Error("value added after overflow should not appear in map")
	}
	// Existing values should still count
	w.AddValue("a")
	if w.Values["a"] != 2 {
		t.Errorf("existing value count after overflow should be 2, got %d", w.Values["a"])
	}
	// TotalObservations keeps going
	if w.TotalObservations != int64(maxVals+2) {
		t.Errorf("TotalObservations = %d, want %d", w.TotalObservations, maxVals+2)
	}
}

func TestWatchedAttribute_InactiveIgnoresValues(t *testing.T) {
	w := NewWatchedAttribute("k", 100)
	w.Active = false
	w.AddValue("anything")
	if w.TotalObservations != 0 {
		t.Errorf("inactive watch should not record observations, got %d", w.TotalObservations)
	}
}

func TestWatchedAttribute_Snapshot(t *testing.T) {
	w := NewWatchedAttribute("snap.key", 100)
	w.AddValue("foo")
	w.AddValue("bar")
	w.AddValue("foo")

	key, vals, unique, total, active, overflow, since := w.Snapshot()
	if key != "snap.key" {
		t.Errorf("key = %q", key)
	}
	if unique != 2 {
		t.Errorf("unique = %d, want 2", unique)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if !active {
		t.Error("active should be true")
	}
	if overflow {
		t.Error("overflow should be false")
	}
	if since.IsZero() {
		t.Error("since should not be zero")
	}
	if vals["foo"] != 2 || vals["bar"] != 1 {
		t.Errorf("unexpected values map: %v", vals)
	}

	// Snapshot copy is independent of original
	delete(vals, "foo")
	if w.Values["foo"] != 2 {
		t.Error("snapshot should be a copy, mutation should not affect original")
	}
}

func TestWatchedAttribute_Concurrency(t *testing.T) {
	w := NewWatchedAttribute("k", 10000)
	const goroutines = 50
	const valuesPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < valuesPerGoroutine; i++ {
				w.AddValue(string(rune('a' + (g*valuesPerGoroutine+i)%26)))
			}
		}()
	}
	wg.Wait()

	if w.TotalObservations != goroutines*valuesPerGoroutine {
		t.Errorf("TotalObservations = %d, want %d", w.TotalObservations, goroutines*valuesPerGoroutine)
	}
	if w.UniqueCount != int64(len(w.Values)) {
		t.Errorf("UniqueCount %d != len(Values) %d", w.UniqueCount, len(w.Values))
	}
}

func TestWatchedAttribute_WatchingSince(t *testing.T) {
	before := time.Now()
	w := NewWatchedAttribute("k", 100)
	after := time.Now()

	if w.WatchingSince.Before(before) || w.WatchingSince.After(after) {
		t.Errorf("WatchingSince %v not between %v and %v", w.WatchingSince, before, after)
	}
}
