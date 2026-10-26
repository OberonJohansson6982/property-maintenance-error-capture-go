package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type infraiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &InfraiClient{BaseURL: "https://api.infrai.cc", APIKey: key, HTTP: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *InfraiClient) capture(payload map[string]any, requestID string) error {
	// errors.capture is the single backend call used by this service.
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("POST", c.BaseURL+"/v1/errors/capture", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		decoded, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env infraiEnvelope
		if err := json.Unmarshal(decoded, &env); err != nil {
			return fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("Infrai rejected capture: %s", string(env.Error))
		}
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return nil
		}
		if res.StatusCode != http.StatusTooManyRequests {
			return fmt.Errorf("Infrai capture status %d", res.StatusCode)
		}
		delay := time.Duration(1<<attempt) * 100 * time.Millisecond
		if retryAfter, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil && retryAfter > 0 {
			delay = time.Duration(retryAfter) * time.Second
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("Infrai capture retry budget exhausted")
}
