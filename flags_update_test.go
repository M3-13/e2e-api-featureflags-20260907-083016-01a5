package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedUpdateFlag(t *testing.T, f Flag) {
	t.Helper()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.flags[f.Key] = f
}

func TestUpdateFlagOnlyEnabled(t *testing.T) {
	seedUpdateFlag(t, Flag{Key: "myflag", Enabled: true, Description: "original", RolloutPercent: 50})

	handler := newHandler()
	req := httptest.NewRequest(http.MethodPut, "/flags/myflag", strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if got.Enabled {
		t.Fatalf("expected enabled=false, got true")
	}
	if got.Description != "original" {
		t.Fatalf("description changed: %q", got.Description)
	}
	if got.RolloutPercent != 50 {
		t.Fatalf("rollout_percent changed: %d", got.RolloutPercent)
	}
	if got.Key != "myflag" {
		t.Fatalf("key changed: %q", got.Key)
	}
}

func TestUpdateFlagUnknownKey(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodPut, "/flags/does-not-exist", strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateFlagRolloutOutOfRange(t *testing.T) {
	seedUpdateFlag(t, Flag{Key: "myflag", Enabled: true, RolloutPercent: 50})

	handler := newHandler()
	req := httptest.NewRequest(http.MethodPut, "/flags/myflag", strings.NewReader(`{"rollout_percent":150}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagInvalidJSON(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodPut, "/flags/myflag", strings.NewReader(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagWrongContentType(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodPut, "/flags/myflag", strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rec.Code)
	}
}

func TestUpdateFlagBodyTooLarge(t *testing.T) {
	handler := newHandler()
	body := `{"description":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPut, "/flags/myflag", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}
