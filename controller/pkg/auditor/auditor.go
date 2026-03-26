// Package auditor provides structured audit logging for governance evaluations.
// Every Evaluate() call emits AuditEvent JSON lines to stdout, which can be
// picked up by any log aggregator (Loki, Fluentd, CloudWatch, Splunk, etc.).
//
// Tier 2 #16 — Audit logging pipeline
package auditor

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// EventType classifies the kind of audit event.
type EventType string

const (
	EventTypeEvaluation  EventType = "EVALUATION"
	EventTypeFinding     EventType = "FINDING"
	EventTypeScoreChange EventType = "SCORE_CHANGE"
	EventTypePolicy      EventType = "POLICY"
)

// AuditEvent is a single structured log entry emitted on stdout as JSON.
// All fields are JSON-serialised so log aggregators can index them without parsing.
type AuditEvent struct {
	// When the event was recorded (RFC3339 UTC)
	Timestamp time.Time `json:"timestamp"`

	// EventType is one of EVALUATION | FINDING | SCORE_CHANGE | POLICY
	EventType EventType `json:"eventType"`

	// EvaluationID ties all events from a single Evaluate() call together
	EvaluationID string `json:"evaluationId"`

	// Cluster context
	ClusterName string `json:"clusterName,omitempty"`

	// MCP server context (populated for per-server events)
	MCPServerName      string `json:"mcpServerName,omitempty"`
	MCPServerNamespace string `json:"mcpServerNamespace,omitempty"`

	// Finding details (populated for EventTypeFinding)
	FindingID       string `json:"findingId,omitempty"`
	FindingSeverity string `json:"findingSeverity,omitempty"` // Critical | High | Medium | Low
	FindingCategory string `json:"findingCategory,omitempty"`

	// Score details (populated for EventTypeScoreChange)
	PreviousScore  int            `json:"previousScore,omitempty"`
	NewScore       int            `json:"newScore,omitempty"`
	ScoreBreakdown map[string]int `json:"scoreBreakdown,omitempty"`

	// Policy context (populated for EventTypePolicy)
	PolicyName string `json:"policyName,omitempty"`

	// Action taken: CREATED | UPDATED | REMEDIATED
	Action string `json:"action,omitempty"`

	// Human-readable summary
	Message string `json:"message"`
}

// Logger writes structured audit events to stdout as newline-delimited JSON.
// It is safe for concurrent use.
type Logger struct {
	enabled     bool
	clusterName string
}

// NewLogger creates a new audit Logger.
// When enabled is false all Log* calls are no-ops (zero cost in production).
func NewLogger(clusterName string, enabled bool) *Logger {
	return &Logger{
		enabled:     enabled,
		clusterName: clusterName,
	}
}

// NewEvaluationID generates a short unique ID for one Evaluate() call.
func NewEvaluationID() string {
	return fmt.Sprintf("eval-%d-%04x", time.Now().Unix(), rand.Intn(0xffff)) //nolint:gosec
}

// LogEvaluation records the start or completion of a governance evaluation.
func (l *Logger) LogEvaluation(evaluationID, message string) {
	if !l.enabled {
		return
	}
	l.emit(AuditEvent{
		Timestamp:    time.Now().UTC(),
		EventType:    EventTypeEvaluation,
		EvaluationID: evaluationID,
		ClusterName:  l.clusterName,
		Action:       "CREATED",
		Message:      message,
	})
}

// LogFinding records a single governance finding detected during evaluation.
func (l *Logger) LogFinding(evaluationID, findingID, severity, category, mcpName, mcpNamespace, message string) {
	if !l.enabled {
		return
	}
	l.emit(AuditEvent{
		Timestamp:          time.Now().UTC(),
		EventType:          EventTypeFinding,
		EvaluationID:       evaluationID,
		ClusterName:        l.clusterName,
		MCPServerName:      mcpName,
		MCPServerNamespace: mcpNamespace,
		FindingID:          findingID,
		FindingSeverity:    severity,
		FindingCategory:    category,
		Action:             "CREATED",
		Message:            message,
	})
}

// LogScoreChange records a per-server score computed during evaluation.
func (l *Logger) LogScoreChange(evaluationID, mcpName, mcpNamespace string, previousScore, newScore int, breakdown map[string]int, message string) {
	if !l.enabled {
		return
	}
	l.emit(AuditEvent{
		Timestamp:          time.Now().UTC(),
		EventType:          EventTypeScoreChange,
		EvaluationID:       evaluationID,
		ClusterName:        l.clusterName,
		MCPServerName:      mcpName,
		MCPServerNamespace: mcpNamespace,
		PreviousScore:      previousScore,
		NewScore:           newScore,
		ScoreBreakdown:     breakdown,
		Action:             "UPDATED",
		Message:            message,
	})
}

// LogPolicy records when a governance policy is applied during evaluation.
func (l *Logger) LogPolicy(evaluationID, policyName, message string) {
	if !l.enabled {
		return
	}
	l.emit(AuditEvent{
		Timestamp:    time.Now().UTC(),
		EventType:    EventTypePolicy,
		EvaluationID: evaluationID,
		ClusterName:  l.clusterName,
		PolicyName:   policyName,
		Action:       "APPLIED",
		Message:      message,
	})
}

// emit serialises the event as JSON and writes it to stdout.
// The [AUDIT] prefix makes it easy to grep in mixed log streams.
func (l *Logger) emit(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Printf("[AUDIT] %s\n", string(data))
}
