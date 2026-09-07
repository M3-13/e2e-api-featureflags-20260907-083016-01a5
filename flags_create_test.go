package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateFlagReturns201WithFlagBody(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"dark-mode","enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	var flag Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if flag.Key != "dark-mode" {
		t.Fatalf("expected key dark-mode, got %q", flag.Key)
	}
	if !flag.Enabled {
		t.Fatalf("expected enabled true, got false")
	}
	if flag.RolloutPercent != 0 {
		t.Fatalf("expected default rollout_percent 0, got %d", flag.RolloutPercent)
	}
}

func TestCreateFlagDefaultsRolloutPercent(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"opt","enabled":true,"description":"d"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var flag Flag
	_ = json.Unmarshal(rec.Body.Bytes(), &flag)
	if flag.RolloutPercent != 0 {
		t.Fatalf("expected default rollout_percent 0, got %d", flag.RolloutPercent)
	}
	if flag.Description != "d" {
		t.Fatalf("expected description d, got %q", flag.Description)
	}
}

func TestCreateFlagDuplicateKeyReturns409(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	body := `{"key":"dup","enabled":true}`
	first := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(body))
	first.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, first)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first POST expected 201, got %d", rec1.Code)
	}

	second := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(body))
	second.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, second)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec2.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("error body not JSON: %v", err)
	}
	if parsed["error"] == "" {
		t.Fatalf("expected error field, got %s", rec2.Body.String())
	}
}

func TestCreateFlagMissingKeyReturns400(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagEmptyKeyReturns400(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"","enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagMissingEnabledReturns400(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"no-enabled"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagRolloutPercentOutOfRange(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	for _, rp := range []string{"150", "-1"} {
		body := `{"key":"r","enabled":true,"rollout_percent":` + rp + `}`
		req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("rollout_percent=%s: expected 400, got %d", rp, rec.Code)
		}
	}
}

func TestCreateFlagInvalidJSONReturns400(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagWrongContentTypeReturns415(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"x","enabled":true}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rec.Code)
	}
}

func TestCreateFlagBodyTooLargeReturns413(t *testing.T) {
	store = NewStore()
	handler := newHandler()

	big := `{"key":"x","enabled":true,"description":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("error body not JSON: %v", err)
	}
	if parsed["error"] == "" {
		t.Fatalf("expected error field, got %s", rec.Body.String())
	}
}
