package types

import (
	"encoding/json"
	"time"
)

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations,omitempty"`
	StartsAt    time.Time         `json:"startsAt,omitempty"`
	EndsAt      time.Time         `json:"endsAt,omitempty"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty"`

	// Для совместимости с API v1 и v2
	Status      string       `json:"status,omitempty"` // v1
	AlertStatus *AlertStatus `json:"status,omitempty"` // v2
}

type AlertStatus struct {
	State       string   `json:"state"`
	SilencedBy  []string `json:"silencedBy"`
	InhibitedBy []string `json:"inhibitedBy"`
	MutedBy     []string `json:"mutedBy"`
}

func (a *Alert) UnmarshalJSON(data []byte) error {
	type Alias Alert
	aux := &struct {
		*Alias
		RawStatus json.RawMessage `json:"status"`
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Обработка поля status для API v2
	if aux.RawStatus != nil {
		var status AlertStatus
		if err := json.Unmarshal(aux.RawStatus, &status); err == nil {
			a.AlertStatus = &status
		} else {
			// Попробуем распарсить как строку (для v1)
			var statusStr string
			if err := json.Unmarshal(aux.RawStatus, &statusStr); err == nil {
				a.Status = statusStr
			}
		}
	}

	return nil
}

func (a Alert) GetState() string {
	// Приоритет у API v2
	if a.AlertStatus != nil {
		// Если state явно задан
		if a.AlertStatus.State != "" {
			return a.AlertStatus.State
		}
		// Проверяем silenced/muted состояния
		if len(a.AlertStatus.SilencedBy) > 0 || len(a.AlertStatus.MutedBy) > 0 {
			return "silenced"
		}
		// Проверяем inhibited состояния
		if len(a.AlertStatus.InhibitedBy) > 0 {
			return "inhibited"
		}
	}

	// Fallback на API v1
	if a.Status != "" {
		return a.Status
	}

	// Значение по умолчанию
	return "active"
}

type Silence struct {
	ID        string         `json:"id,omitempty"`
	Status    string         `json:"status,omitempty"` // v1
	StatusV2  *SilenceStatus `json:"status,omitempty"` // v2
	Matchers  []Matcher      `json:"matchers"`
	StartsAt  time.Time      `json:"startsAt"`
	EndsAt    time.Time      `json:"endsAt"`
	CreatedBy string         `json:"createdBy"`
	Comment   string         `json:"comment"`
}

type SilenceStatus struct {
	State string `json:"state"`
}

func (s Silence) GetState() string {
	if s.StatusV2 != nil && s.StatusV2.State != "" {
		return s.StatusV2.State
	}
	if s.Status != "" {
		return s.Status
	}
	return "active"
}

type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
}

type AlertResponse struct {
	Status    string  `json:"status"`
	Data      []Alert `json:"data"`
	ErrorType string  `json:"errorType,omitempty"`
	Error     string  `json:"error,omitempty"`
}
