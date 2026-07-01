package main

import (
	"testing"

	"github.com/WangYihang/tranco-go-package"
)

func TestTrancoListCacheGetSet(t *testing.T) {
	cache := newTrancoListCache(2)

	a := &tranco.TrancoList{ID: "a"}
	cache.set("2024-01-01", a)

	got, ok := cache.get("2024-01-01")
	if !ok || got != a {
		t.Fatalf("get(2024-01-01) = (%v, %v), want (%v, true)", got, ok, a)
	}

	if _, ok := cache.get("missing"); ok {
		t.Error("get(missing) should report ok=false")
	}
}

func TestTrancoListCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newTrancoListCache(2)

	first := &tranco.TrancoList{ID: "first"}
	second := &tranco.TrancoList{ID: "second"}
	third := &tranco.TrancoList{ID: "third"}

	cache.set("2024-01-01", first)
	cache.set("2024-01-02", second)

	// Touch the first entry so it's now more recently used than the second.
	if _, ok := cache.get("2024-01-01"); !ok {
		t.Fatal("expected 2024-01-01 to still be cached")
	}

	// Inserting a third entry over capacity should evict 2024-01-02 (the
	// least recently used), not 2024-01-01.
	cache.set("2024-01-03", third)

	if _, ok := cache.get("2024-01-02"); ok {
		t.Error("2024-01-02 should have been evicted as least recently used")
	}
	if got, ok := cache.get("2024-01-01"); !ok || got != first {
		t.Errorf("2024-01-01 should still be cached, got (%v, %v)", got, ok)
	}
	if got, ok := cache.get("2024-01-03"); !ok || got != third {
		t.Errorf("2024-01-03 should be cached, got (%v, %v)", got, ok)
	}
}

func TestTrancoListCacheSetOverwritesExisting(t *testing.T) {
	cache := newTrancoListCache(2)

	original := &tranco.TrancoList{ID: "original"}
	updated := &tranco.TrancoList{ID: "updated"}

	cache.set("2024-01-01", original)
	cache.set("2024-01-01", updated)

	got, ok := cache.get("2024-01-01")
	if !ok || got != updated {
		t.Errorf("get(2024-01-01) = (%v, %v), want (%v, true)", got, ok, updated)
	}
}
