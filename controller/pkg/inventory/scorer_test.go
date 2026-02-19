package inventory

import (
	"testing"
)

func TestGradeFromScore(t *testing.T) {
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
		got := GradeFromScore(tt.score)
		if got != tt.want {
			t.Errorf("GradeFromScore(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestStatusFromScore(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "Verified"},
		{70, "Verified"},
		{69, "Unverified"},
		{50, "Unverified"},
		{49, "Rejected"},
		{30, "Rejected"},
		{29, "Rejected"},
		{0, "Rejected"},
	}
	for _, tt := range tests {
		got := StatusFromScore(tt.score)
		if got != tt.want {
			t.Errorf("StatusFromScore(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

// TestScoreCatalog_FullyVerified tests a resource with all fields populated —
// should score near 100 and get grade A.
func TestScoreCatalog_FullyVerified(t *testing.T) {
	res := &VerifiedResource{
		Name:            "test-mcp-server",
		Namespace:       "default",
		SourceKind:      "MCPServer",
		SourceName:      "my-mcp",
		SourceNamespace: "default",
		Environment:     "production",
		Cluster:         "prod-cluster",
		ManagementType:  "external",
		Transport:       "streamable-http",
		RemoteURL:       "https://mcp.example.com/sse",
		Published:       true,
		DeploymentReady: true,
		Version:         "v1.2.3",
		ToolCount:       5,
		ToolNames:       []string{"tool1", "tool2", "tool3", "tool4", "tool5"},
		UsedByAgents: []AgentUsage{
			{Name: "agent1", Namespace: "default"},
		},
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	if score.Score != 100 {
		t.Errorf("expected score 100 for fully verified resource, got %d", score.Score)
	}
	if score.Grade != "A" {
		t.Errorf("expected grade A, got %s", score.Grade)
	}
	if score.Status != "Verified" {
		t.Errorf("expected status 'Verified', got %s", score.Status)
	}
	if len(score.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d: %+v", len(score.Findings), score.Findings)
	}
	if score.ChecksPassed != score.ChecksTotal {
		t.Errorf("expected all %d checks to pass, only %d passed", score.ChecksTotal, score.ChecksPassed)
	}
	if score.ChecksTotal != 10 {
		t.Errorf("expected 10 checks, got %d", score.ChecksTotal)
	}
}

// TestScoreCatalog_MinimalResource tests a resource with almost no fields —
// should get a low score with multiple findings.
func TestScoreCatalog_MinimalResource(t *testing.T) {
	res := &VerifiedResource{
		Name:      "bare-server",
		Namespace: "default",
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	if score.Score >= 70 {
		t.Errorf("minimal resource should score < 70, got %d", score.Score)
	}
	if score.Grade == "A" || score.Grade == "B" {
		t.Errorf("minimal resource should not get grade A or B, got %s", score.Grade)
	}
	if len(score.Findings) == 0 {
		t.Error("expected findings for minimal resource, got none")
	}
}

// TestScoreCatalog_InsecureHTTP tests that an HTTP (non-TLS) remote URL
// generates a finding.
func TestScoreCatalog_InsecureHTTP(t *testing.T) {
	res := &VerifiedResource{
		Name:            "insecure-server",
		Namespace:       "default",
		SourceKind:      "RemoteMCPServer",
		SourceName:      "remote-mcp",
		SourceNamespace: "default",
		Environment:     "staging",
		Cluster:         "staging-cluster",
		ManagementType:  "external",
		Transport:       "streamable-http",
		RemoteURL:       "http://mcp.internal:8080/sse", // HTTP — not HTTPS
		Published:       true,
		DeploymentReady: true,
		Version:         "v1.0.0",
		ToolCount:       3,
		UsedByAgents: []AgentUsage{
			{Name: "agent-staging", Namespace: "default"},
		},
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	// Should lose 10 points for SEC-002
	if score.Score >= 100 {
		t.Errorf("insecure HTTP should score < 100, got %d", score.Score)
	}
	// SEC-002 should fail
	found := false
	for _, c := range score.Checks {
		if c.ID == "SEC-002" {
			if c.Passed {
				t.Error("SEC-002 should fail for HTTP URL")
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("SEC-002 check not found in results")
	}

	// Should have a finding for TLS
	foundFinding := false
	for _, f := range score.Findings {
		if f.Category == "transport" {
			foundFinding = true
			break
		}
	}
	if !foundFinding {
		t.Error("expected transport finding for insecure HTTP")
	}
}

// TestScoreCatalog_TooManyTools tests the blast radius check when tool count
// exceeds the warning and critical thresholds.
func TestScoreCatalog_TooManyTools(t *testing.T) {
	// Above critical threshold (default 20)
	res := &VerifiedResource{
		Name:            "swiss-army-server",
		Namespace:       "default",
		SourceKind:      "MCPServer",
		SourceName:      "big-mcp",
		SourceNamespace: "default",
		Environment:     "prod",
		Cluster:         "prod-cluster",
		ManagementType:  "managed",
		Transport:       "streamable-http",
		RemoteURL:       "https://big-mcp.example.com",
		Published:       true,
		DeploymentReady: true,
		Version:         "v2.0.0",
		ToolCount:       25, // exceeds critical=20
		UsedByAgents: []AgentUsage{
			{Name: "agent1", Namespace: "default"},
		},
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	// TOOL-001 should fail (score=0)
	for _, c := range score.Checks {
		if c.ID == "TOOL-001" {
			if c.Score != 0 {
				t.Errorf("TOOL-001 should score 0 for 25 tools, got %d", c.Score)
			}
			if c.Passed {
				t.Error("TOOL-001 should not pass for 25 tools")
			}
			break
		}
	}

	// Between warning (10) and critical (20)
	res.ToolCount = 15
	score2 := ScoreCatalog(res, DefaultScoringPolicy())
	for _, c := range score2.Checks {
		if c.ID == "TOOL-001" {
			expectedScore := 15 / 2 // maxSc / 2 = 7
			if c.Score != expectedScore {
				t.Errorf("TOOL-001 should score %d for 15 tools, got %d", expectedScore, c.Score)
			}
			if !c.Passed {
				t.Error("TOOL-001 should still pass for 15 tools (partial credit)")
			}
			break
		}
	}
}

// TestScoreCatalog_StdioTransport tests a local stdio-based server
// (no remote URL, no TLS needed).
func TestScoreCatalog_StdioTransport(t *testing.T) {
	res := &VerifiedResource{
		Name:            "local-server",
		Namespace:       "default",
		SourceKind:      "MCPServer",
		SourceName:      "local-mcp",
		SourceNamespace: "default",
		Environment:     "dev",
		Cluster:         "dev-cluster",
		ManagementType:  "external",
		Transport:       "stdio",
		PackageImage:    "ghcr.io/example/mcp:v1.0.0",
		Published:       true,
		DeploymentReady: true,
		Version:         "v1.0.0",
		ToolCount:       3,
		UsedByAgents: []AgentUsage{
			{Name: "local-agent", Namespace: "default"},
		},
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	// SEC-001 should give 10 for stdio (less than 15 for HTTP)
	for _, c := range score.Checks {
		if c.ID == "SEC-001" {
			if c.Score != 10 {
				t.Errorf("SEC-001 should score 10 for stdio, got %d", c.Score)
			}
			break
		}
	}
	// SEC-002 should give full score (no remote URL)
	for _, c := range score.Checks {
		if c.ID == "SEC-002" {
			if c.Score != 10 {
				t.Errorf("SEC-002 should score 10 when no remote URL, got %d", c.Score)
			}
			break
		}
	}
}

// TestScoreCatalog_NoAgentUsage tests that an MCP server with no agent
// consumers still gets partial credit.
func TestScoreCatalog_NoAgentUsage(t *testing.T) {
	res := &VerifiedResource{
		Name:            "orphaned-server",
		Namespace:       "default",
		SourceKind:      "MCPServer",
		SourceName:      "unused-mcp",
		SourceNamespace: "default",
		Environment:     "prod",
		Cluster:         "prod-cluster",
		ManagementType:  "external",
		Transport:       "streamable-http",
		RemoteURL:       "https://unused-mcp.example.com",
		Published:       true,
		DeploymentReady: true,
		Version:         "v1.0.0",
		ToolCount:       5,
		UsedByAgents:    nil, // no consumers
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	// USE-001 should give partial credit (5/10) but still pass
	for _, c := range score.Checks {
		if c.ID == "USE-001" {
			if c.Score != 5 {
				t.Errorf("USE-001 should give 5 for no agents, got %d", c.Score)
			}
			if !c.Passed {
				t.Error("USE-001 should still pass (partial credit)")
			}
			break
		}
	}
}

// TestScoreCatalog_UnversionedLatest tests that "latest" version does not pass.
func TestScoreCatalog_UnversionedLatest(t *testing.T) {
	res := &VerifiedResource{
		Name:      "unversioned",
		Namespace: "default",
		Version:   "latest",
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	for _, c := range score.Checks {
		if c.ID == "DEP-003" {
			if c.Passed {
				t.Error("DEP-003 should fail for 'latest' version")
			}
			if c.Score != 0 {
				t.Errorf("DEP-003 should score 0 for 'latest', got %d", c.Score)
			}
			break
		}
	}
}

// TestScoreCatalog_CustomPolicy tests that custom policy thresholds are honoured.
func TestScoreCatalog_CustomPolicy(t *testing.T) {
	res := &VerifiedResource{
		Name:            "custom-server",
		Namespace:       "default",
		SourceKind:      "MCPServer",
		SourceName:      "custom-mcp",
		SourceNamespace: "default",
		Environment:     "prod",
		Cluster:         "prod",
		ManagementType:  "managed",
		Transport:       "streamable-http",
		RemoteURL:       "https://custom-mcp.example.com",
		Published:       true,
		DeploymentReady: true,
		Version:         "v1.0.0",
		ToolCount:       8,
		UsedByAgents: []AgentUsage{
			{Name: "agent1", Namespace: "default"},
		},
	}

	// With default policy (warn=10), 8 tools should be fine (full 15)
	score1 := ScoreCatalog(res, DefaultScoringPolicy())
	for _, c := range score1.Checks {
		if c.ID == "TOOL-001" {
			if c.Score != 15 {
				t.Errorf("TOOL-001 should score 15 with default policy for 8 tools, got %d", c.Score)
			}
			break
		}
	}

	// With stricter policy (warn=5, crit=10), 8 tools should be between warn & crit → maxSc/2 = 7
	strictPolicy := ScoringPolicy{MaxToolsWarning: 5, MaxToolsCritical: 10}
	score2 := ScoreCatalog(res, strictPolicy)
	for _, c := range score2.Checks {
		if c.ID == "TOOL-001" {
			expectedScore := 15 / 2 // maxSc / 2 = 7
			if c.Score != expectedScore {
				t.Errorf("TOOL-001 should score %d with strict policy for 8 tools, got %d", expectedScore, c.Score)
			}
			break
		}
	}
}

// TestScoreCatalog_PartialEnvironmentLabels tests that having only one of
// environment or cluster still gives partial credit.
func TestScoreCatalog_PartialEnvironmentLabels(t *testing.T) {
	res := &VerifiedResource{
		Name:        "partial-labels",
		Namespace:   "default",
		Environment: "staging",
		// Cluster intentionally left empty
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())

	for _, c := range score.Checks {
		if c.ID == "PUB-002" {
			if c.Score != 5 {
				t.Errorf("PUB-002 should score 5 for partial labels, got %d", c.Score)
			}
			if !c.Passed {
				t.Error("PUB-002 should pass with partial credit")
			}
			break
		}
	}
}

// TestScoreCatalog_SSETransport tests SSE transport scoring.
func TestScoreCatalog_SSETransport(t *testing.T) {
	res := &VerifiedResource{
		Name:      "sse-server",
		Namespace: "default",
		Transport: "sse",
	}

	score := ScoreCatalog(res, DefaultScoringPolicy())
	for _, c := range score.Checks {
		if c.ID == "SEC-001" {
			if c.Score != 12 {
				t.Errorf("SEC-001 should score 12 for SSE transport, got %d", c.Score)
			}
			break
		}
	}
}

// TestScoreCatalog_AllChecksPresent ensures all 10 expected checks are returned.
func TestScoreCatalog_AllChecksPresent(t *testing.T) {
	res := &VerifiedResource{Name: "any", Namespace: "default"}
	score := ScoreCatalog(res, DefaultScoringPolicy())

	expectedIDs := []string{
		"PUB-001", "PUB-002", "PUB-003",
		"SEC-001", "SEC-002",
		"DEP-001", "DEP-002", "DEP-003",
		"TOOL-001",
		"USE-001",
	}
	idSet := make(map[string]bool)
	for _, c := range score.Checks {
		idSet[c.ID] = true
	}
	for _, id := range expectedIDs {
		if !idSet[id] {
			t.Errorf("expected check %s not found in results", id)
		}
	}
}

// TestDefaultScoringPolicy verifies default thresholds.
func TestDefaultScoringPolicy(t *testing.T) {
	p := DefaultScoringPolicy()
	if p.MaxToolsWarning != 10 {
		t.Errorf("default MaxToolsWarning = %d, want 10", p.MaxToolsWarning)
	}
	if p.MaxToolsCritical != 20 {
		t.Errorf("default MaxToolsCritical = %d, want 20", p.MaxToolsCritical)
	}
	if p.SecurityWeight != 50 {
		t.Errorf("default SecurityWeight = %d, want 50", p.SecurityWeight)
	}
	if p.TrustWeight != 30 {
		t.Errorf("default TrustWeight = %d, want 30", p.TrustWeight)
	}
	if p.ComplianceWeight != 20 {
		t.Errorf("default ComplianceWeight = %d, want 20", p.ComplianceWeight)
	}
	if p.VerifiedThreshold != 70 {
		t.Errorf("default VerifiedThreshold = %d, want 70", p.VerifiedThreshold)
	}
	if p.UnverifiedThreshold != 50 {
		t.Errorf("default UnverifiedThreshold = %d, want 50", p.UnverifiedThreshold)
	}
}

// TestStatusFromScoreWithThresholds verifies custom threshold scoring.
func TestStatusFromScoreWithThresholds(t *testing.T) {
	tests := []struct {
		score     int
		verified  int
		unverified int
		want      string
	}{
		{80, 70, 50, "Verified"},
		{60, 70, 50, "Unverified"},
		{40, 70, 50, "Rejected"},
		{90, 90, 80, "Verified"},
		{85, 90, 80, "Unverified"},
		{70, 90, 80, "Rejected"},
	}
	for _, tt := range tests {
		got := StatusFromScoreWithThresholds(tt.score, tt.verified, tt.unverified)
		if got != tt.want {
			t.Errorf("StatusFromScoreWithThresholds(%d, %d, %d) = %q, want %q",
				tt.score, tt.verified, tt.unverified, got, tt.want)
		}
	}
}

// TestScoreCatalog_CustomCheckMaxScores verifies that per-check max score overrides work.
func TestScoreCatalog_CustomCheckMaxScores(t *testing.T) {
	res := &VerifiedResource{
		Name:            "custom-max-server",
		Namespace:       "default",
		SourceKind:      "MCPServer",
		SourceName:      "custom-mcp",
		SourceNamespace: "default",
		Environment:     "prod",
		Cluster:         "prod",
		ManagementType:  "managed",
		Transport:       "streamable-http",
		RemoteURL:       "https://custom-mcp.example.com",
		Published:       true,
		DeploymentReady: true,
		Version:         "v1.0.0",
		ToolCount:       5,
		UsedByAgents: []AgentUsage{
			{Name: "agent1", Namespace: "default"},
		},
	}

	// Override PUB-001 max score to 20
	policy := DefaultScoringPolicy()
	policy.CheckMaxScores = map[string]int{"PUB-001": 20}
	score := ScoreCatalog(res, policy)

	for _, c := range score.Checks {
		if c.ID == "PUB-001" {
			if c.MaxScore != 20 {
				t.Errorf("PUB-001 MaxScore should be 20, got %d", c.MaxScore)
			}
			if c.Score != 20 {
				t.Errorf("PUB-001 Score should be 20 (passed), got %d", c.Score)
			}
			break
		}
	}
}

// TestScoreCatalog_CustomThresholds verifies custom verified/unverified thresholds.
func TestScoreCatalog_CustomThresholds(t *testing.T) {
	res := &VerifiedResource{
		Name:            "threshold-test",
		Namespace:       "default",
		SourceKind:      "MCPServer",
		SourceName:      "test-mcp",
		SourceNamespace: "default",
		Environment:     "prod",
		Cluster:         "prod",
		ManagementType:  "managed",
		Transport:       "streamable-http",
		RemoteURL:       "https://test.example.com",
		Published:       true,
		DeploymentReady: true,
		Version:         "v1.0.0",
		ToolCount:       5,
		UsedByAgents: []AgentUsage{
			{Name: "agent1", Namespace: "default"},
		},
	}

	// With strict thresholds (verified=90, unverified=80)
	policy := DefaultScoringPolicy()
	policy.VerifiedThreshold = 90
	policy.UnverifiedThreshold = 80

	score := ScoreCatalog(res, policy)
	// Score is 100 with all checks passing → should still be "Verified"
	if score.Status != "Verified" {
		t.Errorf("Expected Verified with score %d and threshold 90, got %s", score.Score, score.Status)
	}
}
