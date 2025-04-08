package api

import (
	"alertctl/internal/types"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AlertManagerClient struct {
	BaseURL    string
	APIVersion string
	HTTPClient *http.Client
	Timeout    time.Duration
}

func NewAlertManagerClient(baseURL, apiVersion string) *AlertManagerClient {
	return &AlertManagerClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIVersion: apiVersion,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *AlertManagerClient) doRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s, response: %s", resp.Status, string(errorBody))
	}

	return resp, nil
}

func (c *AlertManagerClient) GetAlerts() ([]types.Alert, error) {
	endpoint := fmt.Sprintf("/api/%s/alerts", c.APIVersion)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if c.APIVersion == "v1" {
		var response types.AlertResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode v1 response: %v", err)
		}
		return response.Data, nil
	}

	var alerts []types.Alert
	if err := json.Unmarshal(body, &alerts); err != nil {
		return nil, fmt.Errorf("failed to decode v2 response: %v", err)
	}

	// Обогащаем алерты для единообразного представления
	for i := range alerts {
		if alerts[i].StartsAt.IsZero() {
			alerts[i].StartsAt = time.Now()
		}
		if alerts[i].EndsAt.IsZero() {
			alerts[i].EndsAt = time.Now().Add(24 * time.Hour)
		}
	}

	return alerts, nil
}

func (c *AlertManagerClient) GetSilences() ([]types.Silence, error) {
	endpoint := fmt.Sprintf("/api/%s/silences", c.APIVersion)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var silences []types.Silence
	if err := json.NewDecoder(resp.Body).Decode(&silences); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return silences, nil
}

func (c *AlertManagerClient) CreateSilence(silence types.Silence) (string, error) {
	// Устанавливаем значения по умолчанию
	if silence.StartsAt.IsZero() {
		silence.StartsAt = time.Now()
	}
	if silence.EndsAt.IsZero() {
		silence.EndsAt = time.Now().Add(1 * time.Hour)
	}
	if silence.CreatedBy == "" {
		silence.CreatedBy = "alertctl"
	}

	endpoint := fmt.Sprintf("/api/%s/silences", c.APIVersion)
	body, err := json.Marshal(silence)
	if err != nil {
		return "", fmt.Errorf("failed to marshal silence: %v", err)
	}

	resp, err := c.doRequest("POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		SilenceID string `json:"silenceID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	return result.SilenceID, nil
}

func (c *AlertManagerClient) DeleteSilence(id string) error {
	endpoint := fmt.Sprintf("/api/%s/silence/%s", c.APIVersion, id)
	resp, err := c.doRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
