package domain

import (
	"encoding/json"
	"time"
)

type (
	EventType string
)

const (
	EventTypeSecurity      EventType = "trivy.security"
	EventTypeObservability EventType = "observability"
)

type NotificationEnvelope struct {
	EventID   string          `json:"event_id"`
	Source    string          `json:"source"`
	EventType EventType       `json:"event_type"`
	Severity  string          `json:"severity"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}
