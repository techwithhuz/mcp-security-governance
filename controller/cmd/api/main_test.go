package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/techwithhuz/mcp-security-governance/controller/pkg/evaluator"
)

// ────────────────────────────────────────────────────────────────────────────
// Test Setup Helpers
// ────────────────────────────────────────────────────────────────────────────

func setupTestState(result *evaluator.EvaluationResult, cluster *evaluator.ClusterState, p evaluator.Policy) {
	stateMu.Lock()
	defer stateMu.Unlock()
	lastResult = result
	lastCluster = cluster
	policy = p
}

func setupNilState() {
	stateMu.Lock()
	defer stateMu.Unlock()
	lastResult = nil
	lastCluster = nil
	policy = evaluator.DefaultPolicy()
}

func sampleResult() *evaluator.EvaluationResult {
	return &evaluator.EvaluationResult{
		Score:     72,
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		Findings: []evaluator.Finding{
			{ID: "AGW-001", Severity: "Critical", Category: "AgentGateway", Title: "No gateway"},
			{ID: "AUTH-002", Severity: "Critical", Category: "Authentication", Title: "No JWT"},
			{ID: "CORS-001", Severity: "Medium", Category: "CORS", Title: "No CORS"},
			{ID: "TLS-001-b1", Severity: "High", Category: "TLS", Title: "No TLS on b1", ResourceRef: "AgentgatewayBackend/system/b1", Namespace: "system"},
		},
		ScoreBreakdown: evaluator.ScoreBreakdown{
			AgentGatewayScore:   0,
			AuthenticationScore: 0,
			AuthorizationScore:  100,
			CORSScore:           85,
			TLSScore:            75,
			PromptGuardScore:    0,
			RateLimitScore:      0,
			InfraAbsent: map[string]bool{
				"AgentGateway Compliance": true,
				"Authentication":          true,
			},
		},
		ResourceSummary: evaluator.ResourceSummary{
			GatewaysFound:          1,
			AgentgatewayBackends:   2,
			AgentgatewayPolicies:   1,
			KagentAgents:           3,
			KagentRemoteMCPServers: 2,
			TotalMCPEndpoints:      4,
		},
		NamespaceScores: []evaluator.NamespaceScore{
			{Namespace: "default", Score: 100, Findings: 0},
			{Namespace: "system", Score: 75, Findings: 1},
		},
	}
}

func sampleCluster() *evaluator.ClusterState {
	return &evaluator.ClusterState{
		Namespaces: []string{"default", "system"},
		Gateways: []evaluator.GatewayResource{
			{Name: "gw", Namespace: "system", GatewayClassName: "agentgateway"},
		},
		AgentgatewayBackends: []evaluator.AgentgatewayBackendResource{
			{Name: "b1", Namespace: "system", BackendType: "mcp"},
		},
		AgentgatewayPolicies: []evaluator.AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system"},
		},
		KagentAgents: []evaluator.KagentAgentResource{
			{Name: "a1", Namespace: "default"},
		},
		KagentRemoteMCPServers: []evaluator.KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default"},
		},
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Grade & Phase Helpers
// ────────────────────────────────────────────────────────────────────────────

func TestGetGrade(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "A"},
		{95, "A"},
		{90, "A"},
		{89, "B"},
		{70, "B"},
		{69, "C"},
		{50, "C"},
		{49, "D"},
		{30, "D"},
		{29, "F"},
		{0, "F"},
	}
	for _, tt := range tests {
		got := getGrade(tt.score)
		if got != tt.want {
			t.Errorf("getGrade(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestGetPhase(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "Compliant"},
		{90, "Compliant"},
		{89, "PartiallyCompliant"},
		{70, "PartiallyCompliant"},
		{69, "NonCompliant"},
		{50, "NonCompliant"},
		{49, "Critical"},
		{0, "Critical"},
	}
	for _, tt := range tests {
		got := getPhase(tt.score)
		if got != tt.want {
			t.Errorf("getPhase(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "passing"},
		{90, "passing"},
		{89, "warning"},
		{70, "warning"},
		{69, "failing"},
		{50, "failing"},
		{49, "critical"},
		{0, "critical"},
	}
	for _, tt := range tests {
		got := statusLabel(tt.score)
		if got != tt.want {
			t.Errorf("statusLabel(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Health Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)

	if body["status"] != "healthy" {
		t.Errorf("status = %q, want 'healthy'", body["status"])
	}
	if body["version"] == "" {
		t.Error("version should not be empty")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Score Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleScore_NilResult(t *testing.T) {
	setupNilState()
	req := httptest.NewRequest("GET", "/api/governance/score", nil)
	w := httptest.NewRecorder()

	handleScore(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	if body["score"].(float64) != 0 {
		t.Errorf("score = %v, want 0", body["score"])
	}
	if body["grade"] != "F" {
		t.Errorf("grade = %v, want F", body["grade"])
	}
}

func TestHandleScore_WithResult(t *testing.T) {
	setupTestState(sampleResult(), sampleCluster(), evaluator.DefaultPolicy())
	req := httptest.NewRequest("GET", "/api/governance/score", nil)
	w := httptest.NewRecorder()

	handleScore(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	if body["score"].(float64) != 72 {
		t.Errorf("score = %v, want 72", body["score"])
	}
	if body["grade"] != "B" {
		t.Errorf("grade = %v, want B", body["grade"])
	}
	if body["phase"] != "PartiallyCompliant" {
		t.Errorf("phase = %v, want PartiallyCompliant", body["phase"])
	}

	cats, ok := body["categories"].([]interface{})
	if !ok || len(cats) == 0 {
		t.Error("categories should be a non-empty array")
	}

	if body["explanation"] == nil || body["explanation"] == "" {
		t.Error("explanation should be present")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Findings Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleFindings_NilResult(t *testing.T) {
	setupNilState()
	req := httptest.NewRequest("GET", "/api/governance/findings", nil)
	w := httptest.NewRecorder()

	handleFindings(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	if body["total"].(float64) != 0 {
		t.Errorf("total = %v, want 0", body["total"])
	}
}

func TestHandleFindings_WithResult(t *testing.T) {
	setupTestState(sampleResult(), sampleCluster(), evaluator.DefaultPolicy())
	req := httptest.NewRequest("GET", "/api/governance/findings", nil)
	w := httptest.NewRecorder()

	handleFindings(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	total := int(body["total"].(float64))
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}

	findings := body["findings"].([]interface{})
	if len(findings) != 4 {
		t.Errorf("findings len = %d, want 4", len(findings))
	}

	bySeverity := body["bySeverity"].(map[string]interface{})
	if int(bySeverity["Critical"].(float64)) != 2 {
		t.Errorf("Critical count = %v, want 2", bySeverity["Critical"])
	}
	if int(bySeverity["High"].(float64)) != 1 {
		t.Errorf("High count = %v, want 1", bySeverity["High"])
	}
	if int(bySeverity["Medium"].(float64)) != 1 {
		t.Errorf("Medium count = %v, want 1", bySeverity["Medium"])
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Resources Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleResources_NilResult(t *testing.T) {
	setupNilState()
	req := httptest.NewRequest("GET", "/api/governance/resources", nil)
	w := httptest.NewRecorder()

	handleResources(w, req)

	var body evaluator.ResourceSummary
	json.NewDecoder(w.Body).Decode(&body)

	if body.GatewaysFound != 0 {
		t.Errorf("GatewaysFound = %d, want 0", body.GatewaysFound)
	}
}

func TestHandleResources_WithResult(t *testing.T) {
	setupTestState(sampleResult(), sampleCluster(), evaluator.DefaultPolicy())
	req := httptest.NewRequest("GET", "/api/governance/resources", nil)
	w := httptest.NewRecorder()

	handleResources(w, req)

	var body evaluator.ResourceSummary
	json.NewDecoder(w.Body).Decode(&body)

	if body.GatewaysFound != 1 {
		t.Errorf("GatewaysFound = %d, want 1", body.GatewaysFound)
	}
	if body.AgentgatewayBackends != 2 {
		t.Errorf("AgentgatewayBackends = %d, want 2", body.AgentgatewayBackends)
	}
	if body.KagentAgents != 3 {
		t.Errorf("KagentAgents = %d, want 3", body.KagentAgents)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Namespaces Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleNamespaces_NilResult(t *testing.T) {
	setupNilState()
	req := httptest.NewRequest("GET", "/api/governance/namespaces", nil)
	w := httptest.NewRecorder()

	handleNamespaces(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	ns := body["namespaces"].([]interface{})
	if len(ns) != 0 {
		t.Errorf("namespaces len = %d, want 0", len(ns))
	}
}

func TestHandleNamespaces_WithResult(t *testing.T) {
	setupTestState(sampleResult(), sampleCluster(), evaluator.DefaultPolicy())
	req := httptest.NewRequest("GET", "/api/governance/namespaces", nil)
	w := httptest.NewRecorder()

	handleNamespaces(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	ns := body["namespaces"].([]interface{})
	if len(ns) != 2 {
		t.Errorf("namespaces len = %d, want 2", len(ns))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Breakdown Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleBreakdown_NilResult(t *testing.T) {
	setupNilState()
	req := httptest.NewRequest("GET", "/api/governance/breakdown", nil)
	w := httptest.NewRecorder()

	handleBreakdown(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	if len(body) != 0 {
		t.Errorf("breakdown should be empty when no result, got %d keys", len(body))
	}
}

func TestHandleBreakdown_WithResult(t *testing.T) {
	setupTestState(sampleResult(), sampleCluster(), evaluator.DefaultPolicy())
	req := httptest.NewRequest("GET", "/api/governance/breakdown", nil)
	w := httptest.NewRecorder()

	handleBreakdown(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	// Default policy enables: AgentGateway, Auth, AuthZ, CORS, TLS, ToolScope (6 categories)
	// PromptGuard and RateLimit are disabled by default
	if _, ok := body["agentGatewayScore"]; !ok {
		t.Error("breakdown should include agentGatewayScore")
	}
	if _, ok := body["authenticationScore"]; !ok {
		t.Error("breakdown should include authenticationScore")
	}
	if _, ok := body["tlsScore"]; !ok {
		t.Error("breakdown should include tlsScore")
	}
	// PromptGuard and RateLimit disabled by default
	if _, ok := body["promptGuardScore"]; ok {
		t.Error("breakdown should NOT include promptGuardScore (disabled)")
	}
	if _, ok := body["rateLimitScore"]; ok {
		t.Error("breakdown should NOT include rateLimitScore (disabled)")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Full Evaluation Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleFullEvaluation_NilResult(t *testing.T) {
	setupNilState()
	req := httptest.NewRequest("GET", "/api/governance/evaluation", nil)
	w := httptest.NewRecorder()

	handleFullEvaluation(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestHandleFullEvaluation_WithResult(t *testing.T) {
	setupTestState(sampleResult(), sampleCluster(), evaluator.DefaultPolicy())
	req := httptest.NewRequest("GET", "/api/governance/evaluation", nil)
	w := httptest.NewRecorder()

	handleFullEvaluation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	if body["Score"].(float64) != 72 {
		t.Errorf("Score = %v, want 72", body["Score"])
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Resource Detail Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleResourceDetail_NilResult(t *testing.T) {
	setupNilState()
	req := httptest.NewRequest("GET", "/api/governance/resources/detail", nil)
	w := httptest.NewRecorder()

	handleResourceDetail(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	resources := body["resources"].([]interface{})
	if len(resources) != 0 {
		t.Errorf("resources len = %d, want 0", len(resources))
	}
}

func TestHandleResourceDetail_WithResult(t *testing.T) {
	setupTestState(sampleResult(), sampleCluster(), evaluator.DefaultPolicy())
	req := httptest.NewRequest("GET", "/api/governance/resources/detail", nil)
	w := httptest.NewRecorder()

	handleResourceDetail(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	resources := body["resources"].([]interface{})
	if len(resources) == 0 {
		t.Error("resources should not be empty")
	}

	total := int(body["total"].(float64))
	if total != len(resources) {
		t.Errorf("total = %d, want %d", total, len(resources))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// buildResourceDetail
// ────────────────────────────────────────────────────────────────────────────

func TestBuildResourceDetail_NoFindings(t *testing.T) {
	p := evaluator.DefaultPolicy()
	rd := buildResourceDetail("ref", "Gateway", "gw", "ns", nil, p)

	if rd.Score != 100 {
		t.Errorf("score = %d, want 100", rd.Score)
	}
	if rd.Status != "compliant" {
		t.Errorf("status = %q, want 'compliant'", rd.Status)
	}
	if rd.Critical != 0 || rd.High != 0 || rd.Medium != 0 || rd.Low != 0 {
		t.Error("severity counts should all be 0")
	}
	if rd.Findings == nil {
		t.Error("Findings should be initialized to empty slice, not nil")
	}
}

func TestBuildResourceDetail_CriticalFinding(t *testing.T) {
	p := evaluator.DefaultPolicy()
	findings := []evaluator.Finding{
		{Severity: "Critical", Category: "AgentGateway"},
	}

	rd := buildResourceDetail("ref", "Gateway", "gw", "ns", findings, p)

	if rd.Score != 0 {
		t.Errorf("score = %d, want 0 (critical finding)", rd.Score)
	}
	if rd.Status != "critical" {
		t.Errorf("status = %q, want 'critical'", rd.Status)
	}
	if rd.Critical != 1 {
		t.Errorf("Critical = %d, want 1", rd.Critical)
	}
}

func TestBuildResourceDetail_HighFinding(t *testing.T) {
	p := evaluator.DefaultPolicy()
	findings := []evaluator.Finding{
		{Severity: "High", Category: "TLS"},
	}

	rd := buildResourceDetail("ref", "Backend", "b1", "ns", findings, p)

	// 100 - 25 (default High penalty) = 75
	if rd.Score != 75 {
		t.Errorf("score = %d, want 75", rd.Score)
	}
	// buildResourceDetail status: High -> "failing"
	if rd.Status != "failing" {
		t.Errorf("status = %q, want 'failing'", rd.Status)
	}
	if rd.High != 1 {
		t.Errorf("High = %d, want 1", rd.High)
	}
}

func TestBuildResourceDetail_MediumFinding(t *testing.T) {
	p := evaluator.DefaultPolicy()
	findings := []evaluator.Finding{
		{Severity: "Medium", Category: "CORS"},
	}

	rd := buildResourceDetail("ref", "Route", "r1", "ns", findings, p)

	// 100 - 15 = 85
	if rd.Score != 85 {
		t.Errorf("score = %d, want 85", rd.Score)
	}
	// buildResourceDetail status: Medium -> "warning"
	if rd.Status != "warning" {
		t.Errorf("status = %q, want 'warning'", rd.Status)
	}
	if rd.Medium != 1 {
		t.Errorf("Medium = %d, want 1", rd.Medium)
	}
}

func TestBuildResourceDetail_LowFinding(t *testing.T) {
	p := evaluator.DefaultPolicy()
	findings := []evaluator.Finding{
		{Severity: "Low", Category: "CORS"},
	}

	rd := buildResourceDetail("ref", "Route", "r1", "ns", findings, p)

	// 100 - 5 = 95
	if rd.Score != 95 {
		t.Errorf("score = %d, want 95", rd.Score)
	}
	// buildResourceDetail status: Low -> "info"
	if rd.Status != "info" {
		t.Errorf("status = %q, want 'info'", rd.Status)
	}
	if rd.Low != 1 {
		t.Errorf("Low = %d, want 1", rd.Low)
	}
}

func TestBuildResourceDetail_ScoreFloor(t *testing.T) {
	p := evaluator.DefaultPolicy()
	// 5 High findings: 5 * 25 = 125, should floor at 0
	findings := []evaluator.Finding{
		{Severity: "High"},
		{Severity: "High"},
		{Severity: "High"},
		{Severity: "High"},
		{Severity: "High"},
	}

	rd := buildResourceDetail("ref", "Backend", "b1", "ns", findings, p)

	if rd.Score != 0 {
		t.Errorf("score = %d, want 0 (floor)", rd.Score)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Trend Recording
// ────────────────────────────────────────────────────────────────────────────

func TestRecordTrendPoint_Nil(t *testing.T) {
	trendMu.Lock()
	trendHistory = nil
	trendMu.Unlock()

	recordTrendPoint(nil)

	trendMu.RLock()
	defer trendMu.RUnlock()
	if len(trendHistory) != 0 {
		t.Errorf("nil result should not add trend point, got %d", len(trendHistory))
	}
}

func TestRecordTrendPoint_AddsPoint(t *testing.T) {
	trendMu.Lock()
	trendHistory = nil
	trendMu.Unlock()

	result := &evaluator.EvaluationResult{
		Score:     50,
		Timestamp: time.Now(),
		Findings: []evaluator.Finding{
			{Severity: "Critical"},
			{Severity: "High"},
			{Severity: "Medium"},
			{Severity: "Low"},
			{Severity: "Low"},
		},
	}
	recordTrendPoint(result)

	trendMu.RLock()
	defer trendMu.RUnlock()

	if len(trendHistory) != 1 {
		t.Fatalf("len = %d, want 1", len(trendHistory))
	}
	tp := trendHistory[0]
	if tp.Score != 50 {
		t.Errorf("Score = %d, want 50", tp.Score)
	}
	if tp.Findings != 5 {
		t.Errorf("Findings = %d, want 5", tp.Findings)
	}
	if tp.Critical != 1 {
		t.Errorf("Critical = %d, want 1", tp.Critical)
	}
	if tp.High != 1 {
		t.Errorf("High = %d, want 1", tp.High)
	}
	if tp.Medium != 1 {
		t.Errorf("Medium = %d, want 1", tp.Medium)
	}
	if tp.Low != 2 {
		t.Errorf("Low = %d, want 2", tp.Low)
	}
}

func TestRecordTrendPoint_MaxHistory(t *testing.T) {
	trendMu.Lock()
	trendHistory = nil
	trendMu.Unlock()

	// Add 110 points — should trim to last 100
	for i := 0; i < 110; i++ {
		recordTrendPoint(&evaluator.EvaluationResult{
			Score:     i,
			Timestamp: time.Now(),
		})
	}

	trendMu.RLock()
	defer trendMu.RUnlock()

	if len(trendHistory) != 100 {
		t.Errorf("len = %d, want 100 (max)", len(trendHistory))
	}
	// First entry should be score=10 (items 0-9 trimmed)
	if trendHistory[0].Score != 10 {
		t.Errorf("first score = %d, want 10", trendHistory[0].Score)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Trends Endpoint
// ────────────────────────────────────────────────────────────────────────────

func TestHandleTrends_Empty(t *testing.T) {
	trendMu.Lock()
	trendHistory = nil
	trendMu.Unlock()

	req := httptest.NewRequest("GET", "/api/governance/trends", nil)
	w := httptest.NewRecorder()

	handleTrends(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	trends := body["trends"].([]interface{})
	if len(trends) != 0 {
		t.Errorf("trends len = %d, want 0", len(trends))
	}
}

func TestHandleTrends_WithData(t *testing.T) {
	trendMu.Lock()
	trendHistory = []TrendPoint{
		{Timestamp: "2026-02-11T12:00:00Z", Score: 50, Findings: 5},
		{Timestamp: "2026-02-11T12:00:30Z", Score: 55, Findings: 4},
	}
	trendMu.Unlock()

	req := httptest.NewRequest("GET", "/api/governance/trends", nil)
	w := httptest.NewRecorder()

	handleTrends(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	trends := body["trends"].([]interface{})
	if len(trends) != 2 {
		t.Errorf("trends len = %d, want 2", len(trends))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// CORS Middleware
// ────────────────────────────────────────────────────────────────────────────

func TestCORSMiddleware_Headers(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS Allow-Origin header missing")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS Allow-Methods header missing")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("CORS Allow-Headers header missing")
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // should not reach here
	}))

	req := httptest.NewRequest("OPTIONS", "/api/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("OPTIONS status = %d, want 200", w.Code)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// JSON Response Helper
// ────────────────────────────────────────────────────────────────────────────

func TestJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	jsonResponse(w, map[string]string{"key": "value"})

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want 'application/json'", w.Header().Get("Content-Type"))
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["key"] != "value" {
		t.Errorf("body['key'] = %q, want 'value'", body["key"])
	}
}
