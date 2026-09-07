package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz: expected 200, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET /healthz: invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("GET /healthz: expected status ok, got %q", body["status"])
	}
	if strings.Contains(rec.Body.String(), "flags") {
		t.Fatalf("GET /healthz: leaked store contents: %s", rec.Body.String())
	}
}

func TestErrorFormatHasNoInternalDetails(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
	body := rec.Body.String()
	for _, leaked := range []string{"goroutine", ".go:", "panic", "runtime error"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("error response leaked internal details (%q): %s", leaked, body)
		}
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("error response is not valid JSON: %v", err)
	}
	if _, ok := parsed["error"]; !ok {
		t.Fatalf("error response missing error field: %s", body)
	}
}

func TestNoCORSHeader(t *testing.T) {
	handler := newHandler()
	for _, path := range []string{"/healthz", "/flags"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if h := rec.Header().Get("Access-Control-Allow-Origin"); h != "" {
			t.Fatalf("%s: unexpected Access-Control-Allow-Origin header %q", path, h)
		}
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	handler := recoveryMiddleware(panicking)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("recovery response is not valid JSON: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("recovery response missing error field: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("recovery leaked panic detail: %s", rec.Body.String())
	}
}

func TestLoggingOmitsQueryString(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/flags/myflag/evaluate?user=alice", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "/flags/myflag/evaluate") {
		t.Fatalf("log is missing the request path: %s", out)
	}
	if strings.Contains(out, "alice") || strings.Contains(out, "user=") {
		t.Fatalf("log leaked the query string: %s", out)
	}
}

func TestLoggingRecordsErrorStatus(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	handler := newHandler()
	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	out := buf.String()
	if !strings.Contains(out, "POST /flags 400 ") {
		t.Fatalf("log did not record the 400 status for POST /flags: %q", out)
	}
	if strings.Contains(out, "POST /flags 200 ") {
		t.Fatalf("log recorded 200 for an error response: %q", out)
	}
}

func TestNotFoundReturnsJSON(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("404 body is not JSON: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("404 body missing error field: %s", rec.Body.String())
	}
}

func TestMethodNotAllowedReturnsJSON(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("405 body is not JSON: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("405 body missing error field: %s", rec.Body.String())
	}
}
