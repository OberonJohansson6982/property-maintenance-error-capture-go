package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type MaintenanceRequest struct {
	ID         string `json:"id"`
	PropertyID string `json:"property_id"`
	TenantID   string `json:"tenant_id"`
	Summary    string `json:"summary"`
}

type TenantDocument struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Kind     string `json:"kind"`
}

type InspectionReminder struct {
	PropertyID string    `json:"property_id"`
	DueAt      time.Time `json:"due_at"`
}

func needsInspection(r MaintenanceRequest, reminder InspectionReminder, now time.Time) bool {
	return r.PropertyID == reminder.PropertyID && !reminder.DueAt.After(now)
}

func captureFailure(client *InfraiClient, requestID string, err error, request MaintenanceRequest) {
	if client == nil {
		log.Printf("maintenance request failed (error capture unavailable): %v", err)
		return
	}
	payload := map[string]any{
		"title":       "maintenance request failed",
		"message":     err.Error(),
		"level":       "error",
		"fingerprint": []string{"maintenance", request.PropertyID},
		"exception":   fmt.Sprintf("%T: %v", err, err),
		"context":     map[string]any{"request": request},
	}
	if captureErr := client.capture(payload, requestID); captureErr != nil {
		log.Printf("error capture rejected: %v", captureErr)
	}
}

func handler(client *InfraiClient, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/maintenance-requests" {
			http.Error(w, "method or path not found", http.StatusNotFound)
			return
		}
		var request MaintenanceRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		reminder := InspectionReminder{PropertyID: request.PropertyID, DueAt: now().Add(-time.Hour)}
		result := map[string]any{"accepted": true, "inspection_due": needsInspection(request, reminder, now())}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			captureFailure(client, request.ID, err, request)
		}
	})
}

func main() {
	client, err := NewInfraiClient()
	if err != nil {
		log.Print(err)
	}
	log.Println("property service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler(client, time.Now)))
}
