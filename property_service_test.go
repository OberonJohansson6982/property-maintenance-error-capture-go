package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type failedResponseWriter struct {
	header http.Header
}

func (w *failedResponseWriter) Header() http.Header { return w.header }
func (w *failedResponseWriter) WriteHeader(int)     {}
func (w *failedResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("response write failed")
}

func TestHandlerCapturesResponseFailure(t *testing.T) {
	var client *InfraiClient
	captured := false
	if os.Getenv("INFRAI_LIVE_VERIFY") == "1" {
		var err error
		client, err = NewInfraiClient()
		if err != nil {
			t.Fatal(err)
		}
	} else {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			captured = true
			if r.URL.Path != "/v1/errors/capture" || r.Method != http.MethodPost {
				t.Errorf("unexpected capture request: %s %s", r.Method, r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "response write failed") {
				t.Errorf("missing backend failure in capture: %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"data":{}}`))
		}))
		defer server.Close()
		client = &InfraiClient{BaseURL: server.URL, HTTP: server.Client()}
	}

	writer := &failedResponseWriter{header: make(http.Header)}
	request := httptest.NewRequest(http.MethodPost, "/maintenance-requests", strings.NewReader(`{"id":"mr-capture-verify","property_id":"p-1","tenant_id":"t-4","summary":"leaking tap"}`))
	handler(client, time.Now).ServeHTTP(writer, request)
	if os.Getenv("INFRAI_LIVE_VERIFY") != "1" && !captured {
		t.Fatal("backend failure was not captured")
	}
}

func TestNeedsInspection(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		request  MaintenanceRequest
		reminder InspectionReminder
		want     bool
	}{
		{"same property and due", MaintenanceRequest{PropertyID: "p-1"}, InspectionReminder{PropertyID: "p-1", DueAt: now}, true},
		{"same property but future", MaintenanceRequest{PropertyID: "p-1"}, InspectionReminder{PropertyID: "p-1", DueAt: now.Add(time.Hour)}, false},
		{"different property", MaintenanceRequest{PropertyID: "p-1"}, InspectionReminder{PropertyID: "p-2", DueAt: now.Add(-time.Hour)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsInspection(tc.request, tc.reminder, now); got != tc.want {
				t.Fatalf("needsInspection() = %v, want %v", got, tc.want)
			}
		})
	}
}
