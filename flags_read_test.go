package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func resetStore(t *testing.T, flags map[string]Flag) {
	t.Helper()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.flags = make(map[string]Flag)
	for k, v := range flags {
		store.flags[k] = v
	}
}

func TestListFlagsReturnsAllFlags(t *testing.T) {
	resetStore(t, map[string]Flag{
		"dark-mode":  {Key: "dark-mode", Enabled: true},
		"beta-dash":  {Key: "beta-dash", Enabled: false, Description: "new dashboard"},
		"alpha-test": {Key: "alpha-test", Enabled: true, RolloutPercent: 50},
	})

	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /flags: expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("GET /flags: expected application/json content type, got %q", ct)
	}
	var flags []Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &flags); err != nil {
		t.Fatalf("GET /flags: invalid JSON: %v", err)
	}
	if len(flags) != 3 {
		t.Fatalf("GET /flags: expected 3 flags, got %d", len(flags))
	}

	seen := map[string]Flag{}
	for _, f := range flags {
		seen[f.Key] = f
	}
	if f, ok := seen["dark-mode"]; !ok || !f.Enabled {
		t.Fatalf("GET /flags: missing or wrong dark-mode flag: %+v", flags)
	}
	if f, ok := seen["beta-dash"]; !ok || f.Enabled || f.Description != "new dashboard" {
		t.Fatalf("GET /flags: missing or wrong beta-dash flag: %+v", flags)
	}
	if f, ok := seen["alpha-test"]; !ok || f.RolloutPercent != 50 {
		t.Fatalf("GET /flags: missing or wrong alpha-test flag: %+v", flags)
	}
}

func TestListFlagsEmptyArray(t *testing.T) {
	resetStore(t, nil)

	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /flags: expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "[]\n" {
		t.Fatalf("GET /flags: expected empty JSON array, got %q", body)
	}
}

func TestGetFlagReturnsFlag(t *testing.T) {
	resetStore(t, map[string]Flag{
		"dark-mode": {Key: "dark-mode", Enabled: true, Description: "toggle theme"},
	})

	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/flags/dark-mode", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /flags/dark-mode: expected 200, got %d", rec.Code)
	}
	var f Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatalf("GET /flags/dark-mode: invalid JSON: %v", err)
	}
	if f.Key != "dark-mode" || !f.Enabled || f.Description != "toggle theme" {
		t.Fatalf("GET /flags/dark-mode: unexpected flag: %+v", f)
	}
}

func TestGetFlagNotFound(t *testing.T) {
	resetStore(t, nil)

	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/flags/unknown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /flags/unknown: expected 404, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("GET /flags/unknown: expected application/json content type, got %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET /flags/unknown: invalid JSON: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("GET /flags/unknown: missing error field: %s", rec.Body.String())
	}
}
