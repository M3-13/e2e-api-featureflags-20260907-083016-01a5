package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func newTestStore() *Store {
	s := NewStore()
	s.flags["feature-a"] = Flag{Key: "feature-a", Enabled: false, RolloutPercent: 50}
	s.flags["zero"] = Flag{Key: "zero", Enabled: true, RolloutPercent: 0}
	s.flags["full"] = Flag{Key: "full", Enabled: false, RolloutPercent: 100}
	return s
}

func doEvaluate(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestEvaluateStability(t *testing.T) {
	s := newTestStore()
	orig := store
	store = s
	defer func() { store = orig }()

	handler := newHandler()
	first := doEvaluate(t, handler, "/flags/feature-a/evaluate?user=alice")
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", first.Code)
	}
	for i := 0; i < 20; i++ {
		rec := doEvaluate(t, handler, "/flags/feature-a/evaluate?user=alice")
		if rec.Code != http.StatusOK {
			t.Fatalf("iteration %d: expected 200, got %d", i, rec.Code)
		}
		if rec.Body.String() != first.Body.String() {
			t.Fatalf("iteration %d: unstable result: %s vs %s", i, rec.Body.String(), first.Body.String())
		}
	}
}

func TestEvaluateRolloutZeroAlwaysFalse(t *testing.T) {
	s := newTestStore()
	orig := store
	store = s
	defer func() { store = orig }()

	handler := newHandler()
	for _, user := range []string{"a", "b", "c", "d", "e"} {
		rec := doEvaluate(t, handler, "/flags/zero/evaluate?user="+user)
		if rec.Code != http.StatusOK {
			t.Fatalf("user %s: expected 200, got %d", user, rec.Code)
		}
		var body map[string]bool
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("user %s: invalid JSON: %v", user, err)
		}
		if body["enabled"] {
			t.Fatalf("user %s: rollout 0 should never be enabled", user)
		}
	}
}

func TestEvaluateRollout100AlwaysTrue(t *testing.T) {
	s := newTestStore()
	orig := store
	store = s
	defer func() { store = orig }()

	handler := newHandler()
	for _, user := range []string{"a", "b", "c", "d", "e"} {
		rec := doEvaluate(t, handler, "/flags/full/evaluate?user="+user)
		if rec.Code != http.StatusOK {
			t.Fatalf("user %s: expected 200, got %d", user, rec.Code)
		}
		var body map[string]bool
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("user %s: invalid JSON: %v", user, err)
		}
		if !body["enabled"] {
			t.Fatalf("user %s: rollout 100 should always be enabled", user)
		}
	}
}

func TestEvaluateDistribution(t *testing.T) {
	s := newTestStore()
	orig := store
	store = s
	defer func() { store = orig }()

	handler := newHandler()
	enabled := 0
	const n = 1000
	for i := 0; i < n; i++ {
		user := "user-" + strconv.Itoa(i)
		rec := doEvaluate(t, handler, "/flags/feature-a/evaluate?user="+user)
		if rec.Code != http.StatusOK {
			t.Fatalf("user %s: expected 200, got %d", user, rec.Code)
		}
		var body map[string]bool
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("user %s: invalid JSON: %v", user, err)
		}
		if body["enabled"] {
			enabled++
		}
	}
	ratio := float64(enabled) / float64(n)
	if ratio < 0.30 || ratio > 0.70 {
		t.Fatalf("rollout 50%%: expected ~50%% enabled over %d users, got %.2f", n, ratio)
	}
}

func TestEvaluateMissingUser(t *testing.T) {
	s := newTestStore()
	orig := store
	store = s
	defer func() { store = orig }()

	handler := newHandler()
	for _, path := range []string{"/flags/feature-a/evaluate", "/flags/feature-a/evaluate?user="} {
		rec := doEvaluate(t, handler, path)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", path, rec.Code)
		}
	}
}

func TestEvaluateUnknownKey(t *testing.T) {
	s := newTestStore()
	orig := store
	store = s
	defer func() { store = orig }()

	handler := newHandler()
	rec := doEvaluate(t, handler, "/flags/does-not-exist/evaluate?user=alice")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
