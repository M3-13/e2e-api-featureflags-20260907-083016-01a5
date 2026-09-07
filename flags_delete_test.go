package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seedFlag(key string) {
	store.mu.Lock()
	store.flags[key] = Flag{Key: key, Enabled: true, RolloutPercent: 100}
	store.mu.Unlock()
}

func TestDeleteFlag(t *testing.T) {
	store = NewStore()
	seedFlag("myflag")

	handler := newHandler()

	req := httptest.NewRequest(http.MethodDelete, "/flags/myflag", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("first DELETE /flags/myflag: expected 204, got %d", rec.Code)
	}
	store.mu.RLock()
	_, stillPresent := store.flags["myflag"]
	store.mu.RUnlock()
	if stillPresent {
		t.Fatalf("first DELETE /flags/myflag: flag still present after deletion")
	}

	req = httptest.NewRequest(http.MethodDelete, "/flags/myflag", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("second DELETE /flags/myflag: expected 404, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("second DELETE /flags/myflag: invalid JSON: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("second DELETE /flags/myflag: missing error field: %s", rec.Body.String())
	}
}

func TestDeleteUnknownFlag(t *testing.T) {
	store = NewStore()

	handler := newHandler()

	req := httptest.NewRequest(http.MethodDelete, "/flags/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE /flags/missing: expected 404, got %d", rec.Code)
	}
}
