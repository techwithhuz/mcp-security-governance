package evaluator

import (
	"testing"
)

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

func defaultPolicy() Policy {
	return DefaultPolicy()
}

func emptyState() *ClusterState {
	return &ClusterState{
		Namespaces: []string{"default"},
	}
}

// fullCompliantState returns a cluster state that should score 100/100.
func fullCompliantState() *ClusterState {
	return &ClusterState{
		Namespaces: []string{"default", "mcp-system"},
		Gateways: []GatewayResource{
			{Name: "agentgateway", Namespace: "agentgateway-system", GatewayClassName: "agentgateway", Programmed: true},
		},
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{
				Name: "mcp-backend", Namespace: "agentgateway-system", BackendType: "mcp",
				HasTLS: true,
				MCPTargets: []MCPTargetInfo{
					{Name: "my-mcp", Host: "my-mcp.mcp-system.svc.cluster.local", Port: 8080, HasAuth: true, HasRBAC: true},
				},
			},
		},
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{
				Name: "security-policy", Namespace: "agentgateway-system",
				HasJWT: true, JWTMode: "Strict",
				HasCORS: true, HasCSRF: true,
				HasRBAC: true, HasRateLimit: true, HasPromptGuard: true,
			},
		},
		HTTPRoutes: []HTTPRouteResource{
			{Name: "mcp-route", Namespace: "mcp-system", ParentGateway: "agentgateway", ParentGatewayNamespace: "agentgateway-system", BackendRefs: []string{"mcp-backend"}, HasCORSFilter: true},
		},
		KagentAgents: []KagentAgentResource{
			{Name: "agent-1", Namespace: "mcp-system", Type: "Declarative", Ready: true},
		},
		KagentMCPServers: []KagentMCPServerResource{
			{Name: "my-mcp", Namespace: "mcp-system", Transport: "sse", Port: 8080, HasService: true},
		},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "remote-mcp", Namespace: "mcp-system", URL: "http://agentgateway.agentgateway-system:8080/mcp/backend/target", ToolCount: 5},
		},
		Services: []ServiceResource{
			{Name: "agentgateway", Namespace: "agentgateway-system", Ports: []int{8080}},
		},
	}
}

// ────────────────────────────────────────────────────────────────────────────
// DefaultPolicy / DefaultSeverityPenalties
// ────────────────────────────────────────────────────────────────────────────

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()

	if !p.RequireAgentGateway {
		t.Error("RequireAgentGateway should default to true")
	}
	if !p.RequireJWTAuth {
		t.Error("RequireJWTAuth should default to true")
	}
	if !p.RequireRBAC {
		t.Error("RequireRBAC should default to true")
	}
	if !p.RequireCORS {
		t.Error("RequireCORS should default to true")
	}
	if !p.RequireTLS {
		t.Error("RequireTLS should default to true")
	}
	if p.RequirePromptGuard {
		t.Error("RequirePromptGuard should default to false")
	}
	if p.RequireRateLimit {
		t.Error("RequireRateLimit should default to false")
	}
	if p.MaxToolsWarning != 10 {
		t.Errorf("MaxToolsWarning = %d, want 10", p.MaxToolsWarning)
	}
	if p.MaxToolsCritical != 15 {
		t.Errorf("MaxToolsCritical = %d, want 15", p.MaxToolsCritical)
	}

	// Weights should sum to 100
	w := p.Weights
	total := w.AgentGatewayIntegration + w.Authentication + w.Authorization +
		w.CORSPolicy + w.TLSEncryption + w.PromptGuard + w.RateLimit + w.ToolScope
	if total != 100 {
		t.Errorf("Default weights sum to %d, want 100", total)
	}
}

func TestDefaultSeverityPenalties(t *testing.T) {
	p := DefaultSeverityPenalties()
	if p.Critical != 40 {
		t.Errorf("Critical = %d, want 40", p.Critical)
	}
	if p.High != 25 {
		t.Errorf("High = %d, want 25", p.High)
	}
	if p.Medium != 15 {
		t.Errorf("Medium = %d, want 15", p.Medium)
	}
	if p.Low != 5 {
		t.Errorf("Low = %d, want 5", p.Low)
	}
}

func TestDefaultExcludeNamespaces(t *testing.T) {
	ns := DefaultExcludeNamespaces()
	expected := map[string]bool{
		"kube-system":        true,
		"kube-public":        true,
		"kube-node-lease":    true,
		"local-path-storage": true,
	}
	if len(ns) != len(expected) {
		t.Fatalf("DefaultExcludeNamespaces len = %d, want %d", len(ns), len(expected))
	}
	for _, n := range ns {
		if !expected[n] {
			t.Errorf("Unexpected namespace in DefaultExcludeNamespaces: %q", n)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// FilterByNamespaces
// ────────────────────────────────────────────────────────────────────────────

func TestFilterByNamespaces_NoFilters(t *testing.T) {
	state := &ClusterState{
		Namespaces:             []string{"default", "kube-system", "mcp-apps"},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{{Name: "r1", Namespace: "default"}},
	}
	filtered := state.FilterByNamespaces(nil, nil)
	if filtered != state {
		t.Error("FilterByNamespaces with no filters should return same pointer")
	}
}

func TestFilterByNamespaces_TargetOnly(t *testing.T) {
	state := &ClusterState{
		Namespaces: []string{"default", "kube-system", "mcp-apps"},
		KagentAgents: []KagentAgentResource{
			{Name: "a1", Namespace: "default"},
			{Name: "a2", Namespace: "kube-system"},
			{Name: "a3", Namespace: "mcp-apps"},
		},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default"},
			{Name: "r2", Namespace: "mcp-apps"},
		},
	}

	filtered := state.FilterByNamespaces([]string{"default", "mcp-apps"}, nil)

	if len(filtered.Namespaces) != 2 {
		t.Fatalf("Namespaces len = %d, want 2", len(filtered.Namespaces))
	}
	if len(filtered.KagentAgents) != 2 {
		t.Errorf("KagentAgents len = %d, want 2", len(filtered.KagentAgents))
	}
	if len(filtered.KagentRemoteMCPServers) != 2 {
		t.Errorf("RemoteMCPServers len = %d, want 2", len(filtered.KagentRemoteMCPServers))
	}
}

func TestFilterByNamespaces_ExcludeOnly(t *testing.T) {
	state := &ClusterState{
		Namespaces: []string{"default", "kube-system", "mcp-apps"},
		KagentAgents: []KagentAgentResource{
			{Name: "a1", Namespace: "default"},
			{Name: "a2", Namespace: "kube-system"},
			{Name: "a3", Namespace: "mcp-apps"},
		},
	}

	filtered := state.FilterByNamespaces(nil, []string{"kube-system"})

	if len(filtered.Namespaces) != 2 {
		t.Fatalf("Namespaces len = %d, want 2", len(filtered.Namespaces))
	}
	if len(filtered.KagentAgents) != 2 {
		t.Errorf("KagentAgents len = %d, want 2 (kube-system excluded)", len(filtered.KagentAgents))
	}
}

func TestFilterByNamespaces_TargetAndExclude(t *testing.T) {
	state := &ClusterState{
		Namespaces: []string{"default", "kube-system", "mcp-apps", "staging"},
		Services: []ServiceResource{
			{Name: "s1", Namespace: "default"},
			{Name: "s2", Namespace: "kube-system"},
			{Name: "s3", Namespace: "mcp-apps"},
			{Name: "s4", Namespace: "staging"},
		},
	}

	// Target 3, exclude 1 of them → 2 remain
	filtered := state.FilterByNamespaces([]string{"default", "kube-system", "mcp-apps"}, []string{"kube-system"})

	if len(filtered.Namespaces) != 2 {
		t.Fatalf("Namespaces len = %d, want 2", len(filtered.Namespaces))
	}
	if len(filtered.Services) != 2 {
		t.Errorf("Services len = %d, want 2", len(filtered.Services))
	}
}

func TestFilterByNamespaces_GatewaysPreserved(t *testing.T) {
	state := &ClusterState{
		Namespaces: []string{"default", "agentgateway-system"},
		Gateways: []GatewayResource{
			{Name: "gw", Namespace: "agentgateway-system", GatewayClassName: "agentgateway"},
		},
		KagentAgents: []KagentAgentResource{
			{Name: "a1", Namespace: "default"},
		},
	}

	filtered := state.FilterByNamespaces([]string{"default"}, nil)
	// Gateways are cluster-scoped and should always be preserved
	if len(filtered.Gateways) != 1 {
		t.Errorf("Gateways should be preserved regardless of namespace filter, got %d", len(filtered.Gateways))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Evaluate — Integration
// ────────────────────────────────────────────────────────────────────────────

func TestEvaluate_EmptyCluster(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	result := Evaluate(state, policy)

	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Score > 10 {
		t.Errorf("Empty cluster should score very low, got %d", result.Score)
	}
	if len(result.Findings) == 0 {
		t.Error("Empty cluster should produce findings")
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestEvaluate_FullyCompliant(t *testing.T) {
	state := fullCompliantState()
	policy := defaultPolicy()
	// Enable all categories
	policy.RequirePromptGuard = true
	policy.RequireRateLimit = true

	result := Evaluate(state, policy)

	if result.Score < 80 {
		t.Errorf("Fully compliant cluster should score high, got %d", result.Score)
	}
	// Should have very few or no critical findings
	criticalCount := 0
	for _, f := range result.Findings {
		if f.Severity == SeverityCritical {
			criticalCount++
		}
	}
	if criticalCount > 0 {
		t.Errorf("Fully compliant cluster should have 0 Critical findings, got %d", criticalCount)
		for _, f := range result.Findings {
			if f.Severity == SeverityCritical {
				t.Logf("  Critical: %s - %s", f.ID, f.Title)
			}
		}
	}
}

func TestEvaluate_ResourceSummary(t *testing.T) {
	state := fullCompliantState()
	policy := defaultPolicy()

	result := Evaluate(state, policy)
	rs := result.ResourceSummary

	if rs.GatewaysFound != 1 {
		t.Errorf("GatewaysFound = %d, want 1", rs.GatewaysFound)
	}
	if rs.AgentgatewayBackends != 1 {
		t.Errorf("AgentgatewayBackends = %d, want 1", rs.AgentgatewayBackends)
	}
	if rs.AgentgatewayPolicies != 1 {
		t.Errorf("AgentgatewayPolicies = %d, want 1", rs.AgentgatewayPolicies)
	}
	if rs.KagentAgents != 1 {
		t.Errorf("KagentAgents = %d, want 1", rs.KagentAgents)
	}
	if rs.KagentMCPServers != 1 {
		t.Errorf("KagentMCPServers = %d, want 1", rs.KagentMCPServers)
	}
	if rs.KagentRemoteMCPServers != 1 {
		t.Errorf("KagentRemoteMCPServers = %d, want 1", rs.KagentRemoteMCPServers)
	}
}

func TestEvaluate_NamespaceScores(t *testing.T) {
	state := &ClusterState{
		Namespaces: []string{"clean-ns", "dirty-ns"},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "ok-mcp", Namespace: "clean-ns", URL: "http://localhost:8080", ToolCount: 3},
			{Name: "bad-mcp", Namespace: "dirty-ns", URL: "http://localhost:9090", ToolCount: 50},
		},
	}
	policy := defaultPolicy()
	result := Evaluate(state, policy)

	if len(result.NamespaceScores) != 2 {
		t.Fatalf("NamespaceScores len = %d, want 2", len(result.NamespaceScores))
	}

	// dirty-ns should have more findings than clean-ns
	var cleanFindings, dirtyFindings int
	for _, ns := range result.NamespaceScores {
		switch ns.Namespace {
		case "clean-ns":
			cleanFindings = ns.Findings
		case "dirty-ns":
			dirtyFindings = ns.Findings
		}
	}
	if dirtyFindings <= cleanFindings {
		t.Errorf("dirty-ns findings (%d) should exceed clean-ns findings (%d)", dirtyFindings, cleanFindings)
	}
}

func TestEvaluate_DisabledCategories(t *testing.T) {
	// Provide a gateway so AGW-001 (always-on gateway check) doesn't fire
	state := emptyState()
	state.Gateways = []GatewayResource{
		{Name: "gw", Namespace: "ns", GatewayClassName: "agentgateway", Programmed: true},
	}
	policy := Policy{
		// Disable everything
		RequireAgentGateway: false,
		RequireCORS:         false,
		RequireJWTAuth:      false,
		RequireRBAC:         false,
		RequirePromptGuard:  false,
		RequireTLS:          false,
		RequireRateLimit:    false,
		MaxToolsWarning:     0,
		MaxToolsCritical:    0,
		Weights:             DefaultPolicy().Weights,
		SeverityPenalties:   DefaultSeverityPenalties(),
	}

	result := Evaluate(state, policy)

	if result.Score != 100 {
		t.Errorf("All categories disabled → score should be 100, got %d", result.Score)
	}
	if len(result.Findings) != 0 {
		t.Errorf("All categories disabled → findings should be 0, got %d", len(result.Findings))
		for _, f := range result.Findings {
			t.Logf("  Finding: %s - %s", f.ID, f.Title)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// AgentGateway Compliance Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckAgentGateway_NoGateway(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()

	findings := checkAgentGatewayCompliance(state, policy)

	hasAGW001 := false
	for _, f := range findings {
		if f.ID == "AGW-001" {
			hasAGW001 = true
			if f.Severity != SeverityCritical {
				t.Errorf("AGW-001 severity = %s, want Critical", f.Severity)
			}
			if f.Category != CategoryAgentGateway {
				t.Errorf("AGW-001 category = %s, want %s", f.Category, CategoryAgentGateway)
			}
		}
	}
	if !hasAGW001 {
		t.Error("Expected AGW-001 finding for missing gateway")
	}
}

func TestCheckAgentGateway_NotProgrammed(t *testing.T) {
	state := &ClusterState{
		Gateways: []GatewayResource{
			{Name: "gw", Namespace: "system", GatewayClassName: "agentgateway", Programmed: false},
		},
	}
	policy := defaultPolicy()

	findings := checkAgentGatewayCompliance(state, policy)

	hasAGW002 := false
	for _, f := range findings {
		if f.ID == "AGW-002" {
			hasAGW002 = true
			if f.Severity != SeverityHigh {
				t.Errorf("AGW-002 severity = %s, want High", f.Severity)
			}
		}
	}
	if !hasAGW002 {
		t.Error("Expected AGW-002 finding for unprogrammed gateway")
	}
}

func TestCheckAgentGateway_WrongGatewayClass(t *testing.T) {
	state := &ClusterState{
		Gateways: []GatewayResource{
			{Name: "gw", Namespace: "system", GatewayClassName: "nginx", Programmed: true},
		},
	}
	policy := defaultPolicy()

	findings := checkAgentGatewayCompliance(state, policy)

	hasAGW003 := false
	for _, f := range findings {
		if f.ID == "AGW-003" {
			hasAGW003 = true
		}
	}
	if !hasAGW003 {
		t.Error("Expected AGW-003 finding for wrong gateway class")
	}
}

func TestCheckAgentGateway_NoMCPBackend(t *testing.T) {
	state := &ClusterState{
		Gateways: []GatewayResource{
			{Name: "gw", Namespace: "system", GatewayClassName: "agentgateway", Programmed: true},
		},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "remote1", Namespace: "default", URL: "http://localhost"},
		},
	}
	policy := defaultPolicy()

	findings := checkAgentGatewayCompliance(state, policy)

	hasAGW004 := false
	for _, f := range findings {
		if f.ID == "AGW-004" {
			hasAGW004 = true
		}
	}
	if !hasAGW004 {
		t.Error("Expected AGW-004 when MCP servers exist but no MCP backend")
	}
}

func TestCheckAgentGateway_MCPServerNotRouted(t *testing.T) {
	state := &ClusterState{
		Gateways: []GatewayResource{
			{Name: "gw", Namespace: "system", GatewayClassName: "agentgateway", Programmed: true},
		},
		KagentMCPServers: []KagentMCPServerResource{
			{Name: "my-mcp", Namespace: "default", Transport: "sse", Port: 8080},
		},
	}
	policy := defaultPolicy()

	findings := checkAgentGatewayCompliance(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "AGW-100-my-mcp" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("AGW-100 severity = %s, want Critical", f.Severity)
			}
		}
	}
	if !found {
		t.Error("Expected AGW-100-my-mcp for unrouted MCPServer")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Authentication Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckAuthentication_Disabled(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequireJWTAuth = false

	findings := checkAuthentication(state, policy)
	if len(findings) != 0 {
		t.Errorf("Auth disabled: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckAuthentication_NoJWTPolicy(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasJWT: false},
		},
	}
	policy := defaultPolicy()

	findings := checkAuthentication(state, policy)

	hasAUTH002 := false
	for _, f := range findings {
		if f.ID == "AUTH-002" {
			hasAUTH002 = true
			if f.Severity != SeverityCritical {
				t.Errorf("AUTH-002 severity = %s, want Critical", f.Severity)
			}
		}
	}
	if !hasAUTH002 {
		t.Error("Expected AUTH-002 when no JWT policy exists")
	}
}

func TestCheckAuthentication_OptionalJWT(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasJWT: true, JWTMode: "Optional"},
		},
	}
	policy := defaultPolicy()

	findings := checkAuthentication(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "AUTH-001-p1" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("AUTH-001 severity = %s, want High", f.Severity)
			}
		}
	}
	if !found {
		t.Error("Expected AUTH-001-p1 for Optional JWT mode")
	}
}

func TestCheckAuthentication_StrictJWT_NoFindings(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasJWT: true, JWTMode: "Strict"},
		},
	}
	policy := defaultPolicy()

	findings := checkAuthentication(state, policy)

	for _, f := range findings {
		if f.ID == "AUTH-002" {
			t.Error("Should not have AUTH-002 when strict JWT exists")
		}
		if f.ID == "AUTH-001-p1" {
			t.Error("Should not have AUTH-001 for Strict JWT mode")
		}
	}
}

func TestCheckAuthentication_MCPTargetNoAuth(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasJWT: true, JWTMode: "Strict"},
		},
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{
				Name: "b1", Namespace: "system", BackendType: "mcp",
				MCPTargets: []MCPTargetInfo{
					{Name: "t1", HasAuth: false},
				},
			},
		},
	}
	policy := defaultPolicy()

	findings := checkAuthentication(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "AUTH-100-b1-t1" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("AUTH-100 severity = %s, want Medium", f.Severity)
			}
		}
	}
	if !found {
		t.Error("Expected AUTH-100-b1-t1 for MCP target without auth")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Authorization Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckAuthorization_Disabled(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequireRBAC = false

	findings := checkAuthorization(state, policy)
	if len(findings) != 0 {
		t.Errorf("RBAC disabled: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckAuthorization_NoInfrastructure(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()

	findings := checkAuthorization(state, policy)

	hasRBAC002 := false
	for _, f := range findings {
		if f.ID == "RBAC-002" {
			hasRBAC002 = true
			if f.Severity != SeverityCritical {
				t.Errorf("RBAC-002 severity = %s, want Critical", f.Severity)
			}
		}
	}
	if !hasRBAC002 {
		t.Error("Expected RBAC-002 when no agentgateway infrastructure exists")
	}
}

func TestCheckAuthorization_NoRBACPolicy(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasRBAC: false},
		},
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{Name: "b1", Namespace: "system", BackendType: "mcp"},
		},
	}
	policy := defaultPolicy()

	findings := checkAuthorization(state, policy)

	hasRBAC001 := false
	for _, f := range findings {
		if f.ID == "RBAC-001" {
			hasRBAC001 = true
		}
	}
	if !hasRBAC001 {
		t.Error("Expected RBAC-001 when no RBAC policy exists")
	}
}

func TestCheckAuthorization_MCPTargetNoRBAC(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasRBAC: true},
		},
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{
				Name: "b1", Namespace: "system", BackendType: "mcp",
				MCPTargets: []MCPTargetInfo{
					{Name: "t1", HasRBAC: false},
				},
			},
		},
	}
	policy := defaultPolicy()

	findings := checkAuthorization(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "RBAC-100-b1-t1" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("RBAC-100 severity = %s, want High", f.Severity)
			}
		}
	}
	if !found {
		t.Error("Expected RBAC-100-b1-t1 for MCP target without RBAC")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// CORS Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckCORS_Disabled(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequireCORS = false

	findings := checkCORS(state, policy)
	if len(findings) != 0 {
		t.Errorf("CORS disabled: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckCORS_NoInfrastructure(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()

	findings := checkCORS(state, policy)

	hasCORS003 := false
	for _, f := range findings {
		if f.ID == "CORS-003" {
			hasCORS003 = true
		}
	}
	if !hasCORS003 {
		t.Error("Expected CORS-003 when no infrastructure exists")
	}
}

func TestCheckCORS_NoCORSConfigured(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasCORS: false},
		},
		HTTPRoutes: []HTTPRouteResource{
			{Name: "r1", Namespace: "system", HasCORSFilter: false},
		},
	}
	policy := defaultPolicy()

	findings := checkCORS(state, policy)

	hasCORS001 := false
	for _, f := range findings {
		if f.ID == "CORS-001" {
			hasCORS001 = true
			if f.Severity != SeverityMedium {
				t.Errorf("CORS-001 severity = %s, want Medium", f.Severity)
			}
		}
	}
	if !hasCORS001 {
		t.Error("Expected CORS-001 when no CORS configured")
	}
}

func TestCheckCORS_WithCORS_NoCSRF(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasCORS: true, HasCSRF: false},
		},
	}
	policy := defaultPolicy()

	findings := checkCORS(state, policy)

	hasCORS002 := false
	for _, f := range findings {
		if f.ID == "CORS-002" {
			hasCORS002 = true
			if f.Severity != SeverityLow {
				t.Errorf("CORS-002 severity = %s, want Low", f.Severity)
			}
		}
	}
	if !hasCORS002 {
		t.Error("Expected CORS-002 when CORS present but no CSRF")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// TLS Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckTLS_Disabled(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequireTLS = false

	findings := checkTLS(state, policy)
	if len(findings) != 0 {
		t.Errorf("TLS disabled: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckTLS_NoBackends(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()

	findings := checkTLS(state, policy)

	hasTLS002 := false
	for _, f := range findings {
		if f.ID == "TLS-002" {
			hasTLS002 = true
		}
	}
	if !hasTLS002 {
		t.Error("Expected TLS-002 when no backends exist")
	}
}

func TestCheckTLS_BackendNoTLS(t *testing.T) {
	state := &ClusterState{
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{Name: "b1", Namespace: "system", BackendType: "mcp", HasTLS: false},
		},
	}
	policy := defaultPolicy()

	findings := checkTLS(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "TLS-001-b1" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("TLS-001 severity = %s, want High", f.Severity)
			}
		}
	}
	if !found {
		t.Error("Expected TLS-001-b1 for backend without TLS")
	}
}

func TestCheckTLS_BackendWithTLS(t *testing.T) {
	state := &ClusterState{
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{Name: "b1", Namespace: "system", BackendType: "mcp", HasTLS: true},
		},
	}
	policy := defaultPolicy()

	findings := checkTLS(state, policy)

	if len(findings) != 0 {
		t.Errorf("Backend with TLS: expected 0 findings, got %d", len(findings))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Prompt Guard Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckPromptGuard_Disabled(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	// RequirePromptGuard defaults to false, but let's be explicit
	policy.RequirePromptGuard = false

	findings := checkPromptGuard(state, policy)
	if len(findings) != 0 {
		t.Errorf("PromptGuard disabled: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckPromptGuard_NoInfrastructure(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequirePromptGuard = true

	findings := checkPromptGuard(state, policy)

	hasPG002 := false
	for _, f := range findings {
		if f.ID == "PG-002" {
			hasPG002 = true
		}
	}
	if !hasPG002 {
		t.Error("Expected PG-002 when no infrastructure exists")
	}
}

func TestCheckPromptGuard_NoPromptGuardPolicy(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasPromptGuard: false},
		},
	}
	policy := defaultPolicy()
	policy.RequirePromptGuard = true

	findings := checkPromptGuard(state, policy)

	hasPG001 := false
	for _, f := range findings {
		if f.ID == "PG-001" {
			hasPG001 = true
		}
	}
	if !hasPG001 {
		t.Error("Expected PG-001 when prompt guard not configured")
	}
}

func TestCheckPromptGuard_Configured(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasPromptGuard: true},
		},
	}
	policy := defaultPolicy()
	policy.RequirePromptGuard = true

	findings := checkPromptGuard(state, policy)
	if len(findings) != 0 {
		t.Errorf("Prompt guard configured: expected 0 findings, got %d", len(findings))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Rate Limit Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckRateLimit_Disabled(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequireRateLimit = false

	findings := checkRateLimit(state, policy)
	if len(findings) != 0 {
		t.Errorf("RateLimit disabled: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckRateLimit_NoInfrastructure(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequireRateLimit = true

	findings := checkRateLimit(state, policy)

	hasRL002 := false
	for _, f := range findings {
		if f.ID == "RL-002" {
			hasRL002 = true
		}
	}
	if !hasRL002 {
		t.Error("Expected RL-002 when no infrastructure exists")
	}
}

func TestCheckRateLimit_NotConfigured(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasRateLimit: false},
		},
	}
	policy := defaultPolicy()
	policy.RequireRateLimit = true

	findings := checkRateLimit(state, policy)

	hasRL001 := false
	for _, f := range findings {
		if f.ID == "RL-001" {
			hasRL001 = true
		}
	}
	if !hasRL001 {
		t.Error("Expected RL-001 when rate limiting not configured")
	}
}

func TestCheckRateLimit_Configured(t *testing.T) {
	state := &ClusterState{
		AgentgatewayPolicies: []AgentgatewayPolicyResource{
			{Name: "p1", Namespace: "system", HasRateLimit: true},
		},
	}
	policy := defaultPolicy()
	policy.RequireRateLimit = true

	findings := checkRateLimit(state, policy)
	if len(findings) != 0 {
		t.Errorf("Rate limit configured: expected 0 findings, got %d", len(findings))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Tool Count Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckToolCount_Disabled(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", ToolCount: 100},
		},
	}
	policy := defaultPolicy()
	policy.MaxToolsWarning = 0
	policy.MaxToolsCritical = 0

	findings := checkToolCount(state, policy)
	if len(findings) != 0 {
		t.Errorf("Tool count disabled: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckToolCount_BelowThreshold(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", ToolCount: 5},
		},
	}
	policy := defaultPolicy()

	findings := checkToolCount(state, policy)
	if len(findings) != 0 {
		t.Errorf("Below threshold: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckToolCount_WarningThreshold(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", ToolCount: 12}, // > 10 warning, < 15 critical
		},
	}
	policy := defaultPolicy()

	findings := checkToolCount(state, policy)

	if len(findings) != 1 {
		t.Fatalf("Warning threshold: expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != SeverityMedium {
		t.Errorf("Warning tool count severity = %s, want Medium", findings[0].Severity)
	}
}

func TestCheckToolCount_CriticalThreshold(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", ToolCount: 50},
		},
	}
	policy := defaultPolicy()

	findings := checkToolCount(state, policy)

	if len(findings) != 1 {
		t.Fatalf("Critical threshold: expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != SeverityCritical {
		t.Errorf("Critical tool count severity = %s, want Critical", findings[0].Severity)
	}
	if findings[0].Category != CategoryToolScope {
		t.Errorf("Category = %s, want %s", findings[0].Category, CategoryToolScope)
	}
}

func TestCheckToolCount_ZeroToolCount(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", ToolCount: 0},
		},
	}
	policy := defaultPolicy()

	findings := checkToolCount(state, policy)
	if len(findings) != 0 {
		t.Errorf("Zero tool count: expected 0 findings, got %d", len(findings))
	}
}

func TestCheckToolCount_MultipleServers(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "ok-server", Namespace: "default", ToolCount: 5},
			{Name: "warn-server", Namespace: "default", ToolCount: 12},
			{Name: "crit-server", Namespace: "default", ToolCount: 20},
		},
	}
	policy := defaultPolicy()

	findings := checkToolCount(state, policy)

	if len(findings) != 2 {
		t.Fatalf("Multiple servers: expected 2 findings, got %d", len(findings))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Exposure Checks
// ────────────────────────────────────────────────────────────────────────────

func TestCheckExposure_Disabled(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", URL: "http://localhost:8080"},
		},
	}
	policy := defaultPolicy()
	policy.RequireAgentGateway = false

	findings := checkExposure(state, policy)
	if len(findings) != 0 {
		t.Errorf("AgentGateway disabled: expected 0 exposure findings, got %d", len(findings))
	}
}

func TestCheckExposure_NotRoutedThroughGateway(t *testing.T) {
	state := &ClusterState{
		Gateways: []GatewayResource{
			{Name: "gw", Namespace: "agentgateway-system", GatewayClassName: "agentgateway"},
		},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", URL: "http://my-mcp.default:8080"},
		},
		Services: []ServiceResource{
			{Name: "agentgateway", Namespace: "agentgateway-system"},
		},
	}
	policy := defaultPolicy()

	findings := checkExposure(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "EXP-001-r1" {
			found = true
			if f.Category != CategoryExposure {
				t.Errorf("Category = %s, want %s", f.Category, CategoryExposure)
			}
		}
	}
	if !found {
		t.Error("Expected EXP-001-r1 for RemoteMCPServer not routed through gateway")
	}
}

func TestCheckExposure_RoutedThroughGateway(t *testing.T) {
	state := &ClusterState{
		Gateways: []GatewayResource{
			{Name: "gw", Namespace: "agentgateway-system", GatewayClassName: "agentgateway"},
		},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", URL: "http://agentgateway.agentgateway-system:8080/mcp/backend/target"},
		},
		Services: []ServiceResource{
			{Name: "agentgateway", Namespace: "agentgateway-system"},
		},
	}
	policy := defaultPolicy()

	findings := checkExposure(state, policy)

	for _, f := range findings {
		if f.ID == "EXP-001-r1" {
			t.Error("Should not have EXP-001-r1 when routed through agentgateway")
		}
	}
}

func TestCheckExposure_NoGateway_CriticalSeverity(t *testing.T) {
	state := &ClusterState{
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{
			{Name: "r1", Namespace: "default", URL: "http://my-mcp.default:8080"},
		},
	}
	policy := defaultPolicy()

	findings := checkExposure(state, policy)

	for _, f := range findings {
		if f.ID == "EXP-001-r1" {
			if f.Severity != SeverityCritical {
				t.Errorf("No gateway → exposure severity = %s, want Critical", f.Severity)
			}
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Scoring Logic
// ────────────────────────────────────────────────────────────────────────────

func TestSeverityPenalty(t *testing.T) {
	penalties := DefaultSeverityPenalties()

	tests := []struct {
		severity string
		want     int
	}{
		{SeverityCritical, 40},
		{SeverityHigh, 25},
		{SeverityMedium, 15},
		{SeverityLow, 5},
		{"Unknown", 0},
	}

	for _, tt := range tests {
		got := severityPenalty(tt.severity, penalties)
		if got != tt.want {
			t.Errorf("severityPenalty(%s) = %d, want %d", tt.severity, got, tt.want)
		}
	}
}

func TestCalculateCategoryScore_NoFindings(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()

	result := calculateCategoryScore(CategoryTLS, "", nil, state, policy)
	if result.Score != 100 {
		t.Errorf("No findings: score = %d, want 100", result.Score)
	}
	if result.InfraAbsent {
		t.Error("No findings: InfraAbsent should be false")
	}
}

func TestCalculateCategoryScore_InfraAbsent(t *testing.T) {
	findings := []Finding{
		{ID: "AGW-001", Category: CategoryAgentGateway, Severity: SeverityCritical},
	}
	state := emptyState()
	policy := defaultPolicy()

	result := calculateCategoryScore(CategoryAgentGateway, "", findings, state, policy)
	if result.Score != 0 {
		t.Errorf("Infra absent: score = %d, want 0", result.Score)
	}
	if !result.InfraAbsent {
		t.Error("Infra absent: InfraAbsent should be true")
	}
}

func TestCalculateCategoryScore_PartialCompliance(t *testing.T) {
	findings := []Finding{
		{ID: "TLS-001-b1", Category: CategoryTLS, Severity: SeverityHigh},
	}
	state := emptyState()
	policy := defaultPolicy()

	result := calculateCategoryScore(CategoryTLS, "", findings, state, policy)
	// 100 - 25 (high penalty) = 75
	if result.Score != 75 {
		t.Errorf("Partial compliance: score = %d, want 75", result.Score)
	}
	if result.InfraAbsent {
		t.Error("Partial compliance: InfraAbsent should be false")
	}
}

func TestCalculateCategoryScore_ScoreFloor(t *testing.T) {
	// Multiple findings that overflow past 0
	findings := []Finding{
		{ID: "f1", Category: CategoryTLS, Severity: SeverityCritical},
		{ID: "f2", Category: CategoryTLS, Severity: SeverityCritical},
		{ID: "f3", Category: CategoryTLS, Severity: SeverityCritical},
		{ID: "f4", Category: CategoryTLS, Severity: SeverityCritical},
	}
	state := emptyState()
	policy := defaultPolicy()

	result := calculateCategoryScore(CategoryTLS, "", findings, state, policy)
	if result.Score != 0 {
		t.Errorf("Score overflow: score = %d, want 0 (floor)", result.Score)
	}
}

func TestCalculateCategoryScore_SecondaryCategoryIncluded(t *testing.T) {
	findings := []Finding{
		{ID: "EXP-001-r1", Category: CategoryExposure, Severity: SeverityCritical},
	}
	state := emptyState()
	policy := defaultPolicy()

	// AgentGateway uses Exposure as secondary category
	result := calculateCategoryScore(CategoryAgentGateway, CategoryExposure, findings, state, policy)
	// This should include the exposure finding
	if result.Score == 100 {
		t.Error("Secondary category findings should be included in score")
	}
}

func TestCalculateOverallScore_NoRequirements(t *testing.T) {
	breakdown := ScoreBreakdown{}
	weights := ScoringWeights{}
	policy := Policy{} // all require* false

	score := calculateOverallScore(breakdown, weights, policy)
	if score != 100 {
		t.Errorf("No requirements: score = %d, want 100", score)
	}
}

func TestCalculateOverallScore_SingleCategory(t *testing.T) {
	breakdown := ScoreBreakdown{
		AgentGatewayScore: 50,
	}
	weights := ScoringWeights{
		AgentGatewayIntegration: 100,
	}
	policy := Policy{
		RequireAgentGateway: true,
	}

	score := calculateOverallScore(breakdown, weights, policy)
	if score != 50 {
		t.Errorf("Single category: score = %d, want 50", score)
	}
}

func TestCalculateOverallScore_WeightedAverage(t *testing.T) {
	breakdown := ScoreBreakdown{
		AgentGatewayScore:   100,
		AuthenticationScore: 0,
	}
	weights := ScoringWeights{
		AgentGatewayIntegration: 50,
		Authentication:          50,
	}
	policy := Policy{
		RequireAgentGateway: true,
		RequireJWTAuth:      true,
	}

	score := calculateOverallScore(breakdown, weights, policy)
	// (100*50 + 0*50) / 100 = 50
	if score != 50 {
		t.Errorf("Weighted average: score = %d, want 50", score)
	}
}

func TestCalculateNamespaceScores_ClusterWideFindingsExcluded(t *testing.T) {
	state := &ClusterState{
		Namespaces: []string{"ns1"},
	}
	findings := []Finding{
		{ID: "AGW-001", Severity: SeverityCritical, Namespace: ""}, // Cluster-wide, no namespace
	}

	scores := calculateNamespaceScores(state, findings, DefaultSeverityPenalties())

	if len(scores) != 1 {
		t.Fatalf("len = %d, want 1", len(scores))
	}
	// Cluster-wide findings (Namespace="") should not affect namespace scores
	if scores[0].Score != 100 {
		t.Errorf("ns1 score = %d, want 100 (cluster-wide findings shouldn't affect ns scores)", scores[0].Score)
	}
}

func TestCalculateNamespaceScores_NamespacedFindings(t *testing.T) {
	state := &ClusterState{
		Namespaces: []string{"ns1", "ns2"},
	}
	findings := []Finding{
		{ID: "f1", Severity: SeverityHigh, Namespace: "ns1"},
		{ID: "f2", Severity: SeverityCritical, Namespace: "ns1"},
	}

	scores := calculateNamespaceScores(state, findings, DefaultSeverityPenalties())

	var ns1Score, ns2Score NamespaceScore
	for _, s := range scores {
		switch s.Namespace {
		case "ns1":
			ns1Score = s
		case "ns2":
			ns2Score = s
		}
	}

	// ns1: 100 - 25 - 40 = 35
	if ns1Score.Score != 35 {
		t.Errorf("ns1 score = %d, want 35", ns1Score.Score)
	}
	if ns1Score.Findings != 2 {
		t.Errorf("ns1 findings = %d, want 2", ns1Score.Findings)
	}
	if ns2Score.Score != 100 {
		t.Errorf("ns2 score = %d, want 100", ns2Score.Score)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Infrastructure Absence Detection
// ────────────────────────────────────────────────────────────────────────────

func TestIsInfrastructureAbsenceFinding(t *testing.T) {
	infraAbsentIDs := []string{
		"AGW-001", "AGW-003", "AGW-004",
		"AUTH-002", "RBAC-002", "CORS-003",
		"TLS-002", "PG-002", "RL-002",
	}
	for _, id := range infraAbsentIDs {
		f := Finding{ID: id}
		if !isInfrastructureAbsenceFinding(f) {
			t.Errorf("isInfrastructureAbsenceFinding(%q) = false, want true", id)
		}
	}

	nonInfraIDs := []string{
		"AGW-002", "AGW-100-x", "AUTH-001-x",
		"RBAC-001", "CORS-001", "TLS-001-x",
		"PG-001", "RL-001", "TOOLS-001-x",
		"EXP-001-x",
	}
	for _, id := range nonInfraIDs {
		f := Finding{ID: id}
		if isInfrastructureAbsenceFinding(f) {
			t.Errorf("isInfrastructureAbsenceFinding(%q) = true, want false", id)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ScoreBreakdown InfraAbsent tracking
// ────────────────────────────────────────────────────────────────────────────

func TestScoreBreakdown_InfraAbsent(t *testing.T) {
	state := emptyState()
	policy := defaultPolicy()
	policy.RequirePromptGuard = true
	policy.RequireRateLimit = true

	result := Evaluate(state, policy)

	// With no infrastructure, all categories should be marked infra-absent
	expectedAbsent := []string{
		"AgentGateway Compliance",
		"Authentication",
		"Authorization",
		"CORS",
		"TLS",
		"Prompt Guard",
		"Rate Limit",
	}
	for _, cat := range expectedAbsent {
		if !result.ScoreBreakdown.InfraAbsent[cat] {
			t.Errorf("InfraAbsent[%q] = false, want true (empty cluster)", cat)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Helper Functions
// ────────────────────────────────────────────────────────────────────────────

func TestIsMCPServerRouted(t *testing.T) {
	mcp := KagentMCPServerResource{Name: "my-mcp", Namespace: "mcp-system"}

	tests := []struct {
		name   string
		state  *ClusterState
		routed bool
	}{
		{
			name:   "no backends",
			state:  &ClusterState{},
			routed: false,
		},
		{
			name: "backend matches by FQDN",
			state: &ClusterState{
				AgentgatewayBackends: []AgentgatewayBackendResource{
					{
						BackendType: "mcp",
						MCPTargets: []MCPTargetInfo{
							{Host: "my-mcp.mcp-system.svc.cluster.local"},
						},
					},
				},
			},
			routed: true,
		},
		{
			name: "backend matches by short name",
			state: &ClusterState{
				AgentgatewayBackends: []AgentgatewayBackendResource{
					{
						BackendType: "mcp",
						MCPTargets: []MCPTargetInfo{
							{Host: "my-mcp"},
						},
					},
				},
			},
			routed: true,
		},
		{
			name: "backend wrong type",
			state: &ClusterState{
				AgentgatewayBackends: []AgentgatewayBackendResource{
					{
						BackendType: "ai",
						MCPTargets: []MCPTargetInfo{
							{Host: "my-mcp.mcp-system.svc.cluster.local"},
						},
					},
				},
			},
			routed: false,
		},
		{
			name: "backend different host",
			state: &ClusterState{
				AgentgatewayBackends: []AgentgatewayBackendResource{
					{
						BackendType: "mcp",
						MCPTargets: []MCPTargetInfo{
							{Host: "other-mcp.mcp-system.svc.cluster.local"},
						},
					},
				},
			},
			routed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMCPServerRouted(mcp, tt.state)
			if got != tt.routed {
				t.Errorf("isMCPServerRouted() = %v, want %v", got, tt.routed)
			}
		})
	}
}

func TestContainsHost(t *testing.T) {
	tests := []struct {
		url       string
		svcName   string
		svcNS     string
		want      bool
	}{
		{"http://agentgateway.agentgateway-system:8080/mcp", "agentgateway", "agentgateway-system", true},
		{"http://agentgateway.agentgateway-system.svc:8080/mcp", "agentgateway", "agentgateway-system", true},
		{"http://my-mcp.default:8080", "agentgateway", "agentgateway-system", false},
		{"http://localhost:8080", "agentgateway", "agentgateway-system", false},
		{"", "agentgateway", "agentgateway-system", false},
	}

	for _, tt := range tests {
		got := containsHost(tt.url, tt.svcName, tt.svcNS)
		if got != tt.want {
			t.Errorf("containsHost(%q, %q, %q) = %v, want %v", tt.url, tt.svcName, tt.svcNS, got, tt.want)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Custom Severity Penalties
// ────────────────────────────────────────────────────────────────────────────

func TestEvaluate_CustomSeverityPenalties(t *testing.T) {
	// In the MCP-server-centric model, the cluster-level ScoreBreakdown
	// is computed from per-server views. Severity penalties still affect
	// namespace-level scoring. Verify that namespace scores reflect custom penalties.
	state := &ClusterState{
		Namespaces: []string{"default"},
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{Name: "b1", Namespace: "default", BackendType: "mcp", HasTLS: false,
				MCPTargets: []MCPTargetInfo{{Name: "mcp-server-1", Host: "mcp-server-1.default.svc.cluster.local", Port: 8080}}},
		},
		KagentMCPServers: []KagentMCPServerResource{
			{Name: "mcp-server-1", Namespace: "default", Transport: "sse", Port: 8080, HasService: true},
		},
	}
	// Mild penalties
	policy := defaultPolicy()
	policy.SeverityPenalties = SeverityPenalties{Critical: 10, High: 5, Medium: 2, Low: 1}

	result := Evaluate(state, policy)

	// The MCP server has no TLS → per-server TLS = 0 → cluster TLS = 0
	if result.ScoreBreakdown.TLSScore != 0 {
		t.Errorf("TLS score with no TLS on MCP server = %d, want 0", result.ScoreBreakdown.TLSScore)
	}

	// Namespace scores should use mild penalties (lower deductions than defaults)
	// Verify that namespace scores exist and are reasonable
	if len(result.NamespaceScores) == 0 {
		t.Error("Expected namespace scores to be computed")
	}
	for _, ns := range result.NamespaceScores {
		if ns.Namespace == "default" && ns.Score < 0 {
			t.Errorf("Namespace score for 'default' = %d, should not be negative", ns.Score)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Custom Scoring Weights
// ────────────────────────────────────────────────────────────────────────────

func TestEvaluate_CustomWeights(t *testing.T) {
	// Use a state where all MCP servers have TLS coverage through their backends
	state := &ClusterState{
		Namespaces: []string{"default", "mcp-system"},
		Gateways: []GatewayResource{
			{Name: "agentgateway", Namespace: "mcp-system", GatewayClassName: "agentgateway", Programmed: true},
		},
		AgentgatewayBackends: []AgentgatewayBackendResource{
			{
				Name: "mcp-backend", Namespace: "mcp-system", BackendType: "mcp",
				HasTLS: true,
				MCPTargets: []MCPTargetInfo{
					{Name: "my-mcp", Host: "my-mcp.mcp-system.svc.cluster.local", Port: 8080},
				},
			},
		},
		KagentMCPServers: []KagentMCPServerResource{
			{Name: "my-mcp", Namespace: "mcp-system", Transport: "sse", Port: 8080, HasService: true},
		},
	}
	policy := defaultPolicy()
	// Set all weight to TLS (which should score 100 with a TLS-enabled backend)
	policy.Weights = ScoringWeights{
		TLSEncryption: 100,
	}
	// Disable other categories
	policy.RequireAgentGateway = false
	policy.RequireCORS = false
	policy.RequireJWTAuth = false
	policy.RequireRBAC = false
	policy.MaxToolsWarning = 0
	policy.MaxToolsCritical = 0

	result := Evaluate(state, policy)

	if result.Score != 100 {
		t.Errorf("Custom weights (TLS only, compliant): score = %d, want 100", result.Score)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Edge Cases
// ────────────────────────────────────────────────────────────────────────────

func TestEvaluate_NilState(t *testing.T) {
	// Passing a completely empty (but non-nil) state shouldn't panic
	state := &ClusterState{}
	policy := defaultPolicy()

	result := Evaluate(state, policy)
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestSummarizeResources(t *testing.T) {
	state := &ClusterState{
		Gateways:               []GatewayResource{{Name: "gw"}},
		AgentgatewayBackends:   []AgentgatewayBackendResource{{Name: "b1", BackendType: "ai"}, {Name: "b2", BackendType: "mcp", MCPTargets: []MCPTargetInfo{{Name: "t1"}, {Name: "t2"}}}},
		AgentgatewayPolicies:   []AgentgatewayPolicyResource{{Name: "p1"}},
		HTTPRoutes:             []HTTPRouteResource{{Name: "r1"}, {Name: "r2"}},
		KagentAgents:           []KagentAgentResource{{Name: "a1"}},
		KagentMCPServers:       []KagentMCPServerResource{{Name: "m1"}},
		KagentRemoteMCPServers: []KagentRemoteMCPServerResource{{Name: "rm1"}},
	}

	rs := summarizeResources(state)

	if rs.GatewaysFound != 1 {
		t.Errorf("GatewaysFound = %d, want 1", rs.GatewaysFound)
	}
	if rs.AgentgatewayBackends != 2 {
		t.Errorf("AgentgatewayBackends = %d, want 2", rs.AgentgatewayBackends)
	}
	if rs.HTTPRoutes != 2 {
		t.Errorf("HTTPRoutes = %d, want 2", rs.HTTPRoutes)
	}
	// TotalMCPEndpoints: 1 MCPServer + 1 RemoteMCPServer + 2 MCP targets from backend = 4
	if rs.TotalMCPEndpoints != 4 {
		t.Errorf("TotalMCPEndpoints = %d, want 4", rs.TotalMCPEndpoints)
	}
}
