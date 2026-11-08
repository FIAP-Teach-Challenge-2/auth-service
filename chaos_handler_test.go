package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChaosHandlerTogglesFault(t *testing.T) {
	app := &App{Fault: &FaultSwitch{}}

	steps := []struct {
		method     string
		wantStatus int
		wantFault  bool
	}{
		{http.MethodPost, http.StatusOK, true},
		{http.MethodGet, http.StatusOK, true},
		{http.MethodDelete, http.StatusOK, false},
		{http.MethodPut, http.StatusMethodNotAllowed, false},
	}

	for _, step := range steps {
		rec := httptest.NewRecorder()
		app.chaosHandler(rec, httptest.NewRequest(step.method, "/admin/chaos", nil))

		if rec.Code != step.wantStatus {
			t.Fatalf("%s: expected status %d, got %d", step.method, step.wantStatus, rec.Code)
		}
		if app.Fault.Enabled() != step.wantFault {
			t.Fatalf("%s: expected fault %v, got %v", step.method, step.wantFault, app.Fault.Enabled())
		}
	}
}

func TestValidateReturns500WhenFaultEnabled(t *testing.T) {
	app := &App{Fault: &FaultSwitch{}}
	app.Fault.Enable()

	req := httptest.NewRequest(http.MethodGet, "/validate", nil)
	req.Header.Set("Authorization", "Bearer tm_key_any")
	rec := httptest.NewRecorder()
	app.validateKeyHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}
