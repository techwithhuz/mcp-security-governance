package evaluator

import (
	"fmt"
	"time"

	v1alpha1 "github.com/techwithhuz/mcp-security-governance/controller/pkg/apis/governance/v1alpha1"
)

// Severity levels for governance findings
const (
	SeverityCritical = "Critical"
	SeverityHigh     = "High"
	SeverityMedium   = "Medium"
	SeverityLow      = "Low"
)

// Categories for governance findings
const (
	CategoryAgentGateway    = "AgentGateway"
	CategoryAuthentication  = "Authentication"
	CategoryAuthorization   = "Authorization"
	CategoryCORS            = "CORS"
	CategoryTLS             = "TLS"
	CategoryPromptGuard     = "PromptGuard"
	CategoryRateLimit       = "RateLimit"
	CategoryExposure        = "Exposure"
	CategoryToolScope       = "ToolScope"
)

// ClusterState holds discovered Kubernetes resource state
type ClusterState struct {
	// agentgateway resources (agentgateway.dev/v1alpha1)
	Gateways             []GatewayResource
	AgentgatewayBackends []AgentgatewayBackendResource
	AgentgatewayPolicies []AgentgatewayPolicyResource
	HTTPRoutes           []HTTPRouteResource

	// kagent resources (kagent.dev/v1alpha1 / v1alpha2)
	KagentAgents           []KagentAgentResource
	KagentMCPServers       []KagentMCPServerResource
	KagentRemoteMCPServers []KagentRemoteMCPServerResource

	// Standard K8s
	Services   []ServiceResource
	Namespaces []string
}

// FilterByNamespaces returns a new ClusterState containing only resources whose
// namespace is in the given list. Cluster-scoped resources (Gateways) are kept
// as-is. If targetNamespaces is empty, the original state is returned unchanged
// (after applying excludeNamespaces if provided).
func (s *ClusterState) FilterByNamespaces(targetNamespaces, excludeNamespaces []string) *ClusterState {
	if len(targetNamespaces) == 0 && len(excludeNamespaces) == 0 {
		return s
	}

	// Build the allowed set: if targetNamespaces is specified, use it as the
	// include list; otherwise start with all discovered namespaces.
	allowed := make(map[string]bool)
	if len(targetNamespaces) > 0 {
		for _, ns := range targetNamespaces {
			allowed[ns] = true
		}
	} else {
		for _, ns := range s.Namespaces {
			allowed[ns] = true
		}
	}

	// Remove excluded namespaces
	for _, ns := range excludeNamespaces {
		delete(allowed, ns)
	}

	filtered := &ClusterState{
		// Keep cluster-scoped resources
		Gateways: s.Gateways,
	}

	// Filter namespaces list
	for _, ns := range s.Namespaces {
		if allowed[ns] {
			filtered.Namespaces = append(filtered.Namespaces, ns)
		}
	}

	// Filter namespaced resources
	for _, r := range s.AgentgatewayBackends {
		if allowed[r.Namespace] {
			filtered.AgentgatewayBackends = append(filtered.AgentgatewayBackends, r)
		}
	}
	for _, r := range s.AgentgatewayPolicies {
		if allowed[r.Namespace] {
			filtered.AgentgatewayPolicies = append(filtered.AgentgatewayPolicies, r)
		}
	}
	for _, r := range s.HTTPRoutes {
		if allowed[r.Namespace] {
			filtered.HTTPRoutes = append(filtered.HTTPRoutes, r)
		}
	}
	for _, r := range s.KagentAgents {
		if allowed[r.Namespace] {
			filtered.KagentAgents = append(filtered.KagentAgents, r)
		}
	}
	for _, r := range s.KagentMCPServers {
		if allowed[r.Namespace] {
			filtered.KagentMCPServers = append(filtered.KagentMCPServers, r)
		}
	}
	for _, r := range s.KagentRemoteMCPServers {
		if allowed[r.Namespace] {
			filtered.KagentRemoteMCPServers = append(filtered.KagentRemoteMCPServers, r)
		}
	}
	for _, r := range s.Services {
		if allowed[r.Namespace] {
			filtered.Services = append(filtered.Services, r)
		}
	}

	return filtered
}

// ---- agentgateway resource representations ----

type GatewayResource struct {
	Name             string
	Namespace        string
	GatewayClassName string
	Listeners        []ListenerInfo
	Programmed       bool
}

type ListenerInfo struct {
	Name     string
	Port     int
	Protocol string
}

type AgentgatewayBackendResource struct {
	Name      string
	Namespace string
	BackendType string // "ai", "mcp", "static", "dynamicForwardProxy"
	MCPTargets  []MCPTargetInfo
	HasAuth     bool
	HasTLS      bool
}

type MCPTargetInfo struct {
	Name     string
	Host     string
	Port     int
	Protocol string // "StreamableHTTP", "SSE"
	HasAuth  bool
	HasRBAC  bool
}

type AgentgatewayPolicyResource struct {
	Name       string
	Namespace  string
	TargetRefs []PolicyTargetRef
	HasJWT     bool
	HasCORS    bool
	HasCSRF    bool
	HasExtAuth bool
	HasRateLimit bool
	HasRBAC    bool
	HasPromptGuard bool
	JWTMode    string // "Strict", "Optional", "Permissive"
	AllowedTools []string // Tool names extracted from authorization CEL matchExpressions
}

type PolicyTargetRef struct {
	Group string
	Kind  string
	Name  string
}

type HTTPRouteResource struct {
	Name                   string
	Namespace              string
	ParentGateway          string
	ParentGatewayNamespace string // defaults to route's own namespace if empty
	BackendRefs            []string
	HasCORSFilter          bool
	Paths                  []string // Extracted path values from rules (e.g., ["/ro", "/rw"])
}

// ---- kagent resource representations ----

type KagentAgentResource struct {
	Name      string
	Namespace string
	Type      string // "Declarative", "BYO"
	Tools     []KagentToolRef
	Ready     bool
}

type KagentToolRef struct {
	Type       string // "McpServer", "Agent"
	Kind       string // "RemoteMCPServer", "MCPServer", or k8s Service
	Name       string
	ToolNames  []string
}

type KagentMCPServerResource struct {
	Name      string
	Namespace string
	Transport string // "stdio", "sse", "streamablehttp"
	Port      int
	HasService bool
}

type KagentRemoteMCPServerResource struct {
	Name      string
	Namespace string
	URL       string
	ToolCount int
	ToolNames []string
}

type ServiceResource struct {
	Name        string
	Namespace   string
	AppProtocol string
	Ports       []int
	IsMCP       bool
}

// Finding represents a governance finding
type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Remediation string `json:"remediation"`
	ResourceRef string `json:"resourceRef,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

// EvaluationResult holds the complete evaluation output
type EvaluationResult struct {
	Score           int
	ScoreBreakdown  ScoreBreakdown
	Findings        []Finding
	ResourceSummary ResourceSummary
	NamespaceScores []NamespaceScore
	Timestamp       time.Time
	// MCP-server-centric views
	MCPServerViews       []MCPServerView             `json:"mcpServerViews"`
	MCPServerSummary     MCPServerSummary            `json:"mcpServerSummary"`
	VerifiedCatalogScores []v1alpha1.VerifiedCatalogScore `json:"verifiedCatalogScores,omitempty"`
}

type ScoreBreakdown struct {
	AgentGatewayScore   int `json:"agentGatewayScore"`
	AuthenticationScore int `json:"authenticationScore"`
	AuthorizationScore  int `json:"authorizationScore"`
	CORSScore           int `json:"corsScore"`
	TLSScore            int `json:"tlsScore"`
	PromptGuardScore    int `json:"promptGuardScore"`
	RateLimitScore      int `json:"rateLimitScore"`
	ToolScopeScore      int `json:"toolScopeScore"`
	// InfraAbsent tracks which categories scored 0 due to missing infrastructure
	// (as opposed to penalty overflow). Key = category display name.
	InfraAbsent map[string]bool `json:"infraAbsent,omitempty"`
}

// categoryScoreResult is the internal return type from calculateCategoryScore.
type categoryScoreResult struct {
	Score       int
	InfraAbsent bool // true when score=0 is caused by missing infrastructure
}

type ResourceSummary struct {
	GatewaysFound          int `json:"gatewaysFound"`
	AgentgatewayBackends   int `json:"agentgatewayBackends"`
	AgentgatewayPolicies   int `json:"agentgatewayPolicies"`
	HTTPRoutes             int `json:"httpRoutes"`
	KagentAgents           int `json:"kagentAgents"`
	KagentMCPServers       int `json:"kagentMCPServers"`
	KagentRemoteMCPServers int `json:"kagentRemoteMCPServers"`
	CompliantResources     int `json:"compliantResources"`
	NonCompliantResources  int `json:"nonCompliantResources"`
	TotalMCPEndpoints      int `json:"totalMCPEndpoints"`
	ExposedMCPEndpoints    int `json:"exposedMCPEndpoints"`
}

type NamespaceScore struct {
	Namespace string `json:"namespace"`
	Score     int    `json:"score"`
	Findings  int    `json:"findings"`
}

// Policy holds the governance policy configuration
type Policy struct {
	Name                string   // Name of the MCPGovernancePolicy CR (for status updates)
	RequireAgentGateway bool
	RequireCORS         bool
	RequireJWTAuth      bool
	RequireRBAC         bool
	RequirePromptGuard  bool
	RequireTLS          bool
	RequireRateLimit    bool
	EnableAIAgent       bool     // If true, use AI agent for governance scoring alongside algorithmic scoring
	AIProvider          string   // LLM provider: "gemini" or "ollama" (default: "gemini")
	AIModel             string   // Model name (e.g. "gemini-2.5-flash", "llama3.1")
	OllamaEndpoint      string   // Ollama API endpoint (default: "http://localhost:11434")
	AIScanInterval      string   // Interval between AI evaluations (e.g. "5m", "10m", "1h"); default: "5m"
	AIScanEnabled       bool     // Whether periodic AI scanning is active (default: true)
	ScanInterval        string   // Interval between governance scans (e.g. "5m", "10m", "1h"); default: "5m"
	MaxToolsWarning     int // If MCP server has more than this many tools, generate Warning
	MaxToolsCritical    int // If MCP server has more than this many tools, generate Critical
	TargetNamespaces    []string // If non-empty, only evaluate resources in these namespaces
	ExcludeNamespaces   []string // Namespaces to exclude from evaluation (e.g. kube-system)
	Weights             ScoringWeights
	SeverityPenalties   SeverityPenalties
	VerifiedCatalogScoring interface{} // *v1alpha1.VerifiedCatalogScoringConfig (stored as interface to avoid circular imports)
}

// SeverityPenalties defines how many points are deducted per finding severity
type SeverityPenalties struct {
	Critical int // default: 40
	High     int // default: 25
	Medium   int // default: 15
	Low      int // default: 5
}

type ScoringWeights struct {
	AgentGatewayIntegration int
	Authentication          int
	Authorization           int
	CORSPolicy              int
	TLSEncryption           int
	PromptGuard             int
	RateLimit               int
	ToolScope               int
}

// DefaultExcludeNamespaces returns the list of system namespaces that should
// be excluded from scanning by default.
func DefaultExcludeNamespaces() []string {
	return []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"local-path-storage",
	}
}

func DefaultPolicy() Policy {
	return Policy{
		RequireAgentGateway: true,
		RequireCORS:         true,
		RequireJWTAuth:      true,
		RequireRBAC:         true,
		RequirePromptGuard:  false,
		RequireTLS:          true,
		RequireRateLimit:    false,
		MaxToolsWarning:     10,
		MaxToolsCritical:    15,
		ExcludeNamespaces:   DefaultExcludeNamespaces(),
		Weights: ScoringWeights{
			AgentGatewayIntegration: 25,
			Authentication:          20,
			Authorization:           15,
			CORSPolicy:              10,
			TLSEncryption:           10,
			PromptGuard:             10,
			RateLimit:               5,
			ToolScope:               5,
		},
		SeverityPenalties: DefaultSeverityPenalties(),
	}
}

// DefaultSeverityPenalties returns the default penalty values
func DefaultSeverityPenalties() SeverityPenalties {
	return SeverityPenalties{
		Critical: 40,
		High:     25,
		Medium:   15,
		Low:      5,
	}
}

// Evaluate runs the full governance evaluation against the cluster state
func Evaluate(state *ClusterState, policy Policy) *EvaluationResult {
	result := &EvaluationResult{
		Timestamp: time.Now(),
	}

	// 1. Discover and summarize resources
	result.ResourceSummary = summarizeResources(state)

	// 2. Run all governance checks
	result.Findings = append(result.Findings, checkAgentGatewayCompliance(state, policy)...)
	result.Findings = append(result.Findings, checkAuthentication(state, policy)...)
	result.Findings = append(result.Findings, checkAuthorization(state, policy)...)
	result.Findings = append(result.Findings, checkCORS(state, policy)...)
	result.Findings = append(result.Findings, checkTLS(state, policy)...)
	result.Findings = append(result.Findings, checkPromptGuard(state, policy)...)
	result.Findings = append(result.Findings, checkRateLimit(state, policy)...)
	result.Findings = append(result.Findings, checkExposure(state, policy)...)
	result.Findings = append(result.Findings, checkToolCount(state, policy)...)

	// 3. Calculate scores
	result.ScoreBreakdown = calculateScores(state, result.Findings, policy)
	result.Score = calculateOverallScore(result.ScoreBreakdown, policy.Weights, policy)

	// 4. Namespace-level scores
	result.NamespaceScores = calculateNamespaceScores(state, result.Findings, policy.SeverityPenalties)

	// 5. Count compliant vs non-compliant
	for _, f := range result.Findings {
		if f.Severity == SeverityCritical || f.Severity == SeverityHigh {
			result.ResourceSummary.NonCompliantResources++
		}
	}
	totalResources := result.ResourceSummary.AgentgatewayBackends +
		result.ResourceSummary.KagentMCPServers +
		result.ResourceSummary.KagentAgents +
		result.ResourceSummary.KagentRemoteMCPServers
	result.ResourceSummary.CompliantResources = totalResources - result.ResourceSummary.NonCompliantResources
	if result.ResourceSummary.CompliantResources < 0 {
		result.ResourceSummary.CompliantResources = 0
	}

	// 6. Build MCP-server-centric views
	result.MCPServerViews = BuildMCPServerViews(state, result.Findings, policy)
	result.MCPServerSummary = BuildMCPServerSummary(result.MCPServerViews)

	// 7. Sync: remove findings that were suppressed by MCP-server-level correlation
	// so the Findings tab and Resource Inventory stay consistent with MCP Server views.
	suppressed := ComputeSuppressedFindingIDs(result.MCPServerViews, result.Findings)
	if len(suppressed) > 0 {
		result.Findings = FilterFindings(result.Findings, suppressed)

		// Recalculate compliant/non-compliant counts with the filtered findings
		result.ResourceSummary.NonCompliantResources = 0
		for _, f := range result.Findings {
			if f.Severity == SeverityCritical || f.Severity == SeverityHigh {
				result.ResourceSummary.NonCompliantResources++
			}
		}
		totalRes := result.ResourceSummary.AgentgatewayBackends +
			result.ResourceSummary.KagentMCPServers +
			result.ResourceSummary.KagentAgents +
			result.ResourceSummary.KagentRemoteMCPServers
		result.ResourceSummary.CompliantResources = totalRes - result.ResourceSummary.NonCompliantResources
		if result.ResourceSummary.CompliantResources < 0 {
			result.ResourceSummary.CompliantResources = 0
		}
	}

	// 8. Recompute cluster-level ScoreBreakdown from MCP-server-centric views
	// so the overview dashboard is consistent with per-server scores.
	result.ScoreBreakdown = aggregateBreakdownFromMCPViews(result.MCPServerViews, policy)
	result.Score = calculateOverallScore(result.ScoreBreakdown, policy.Weights, policy)

	return result
}

func summarizeResources(state *ClusterState) ResourceSummary {
	totalMCP := len(state.KagentMCPServers) + len(state.KagentRemoteMCPServers)
	for _, b := range state.AgentgatewayBackends {
		if b.BackendType == "mcp" {
			totalMCP += len(b.MCPTargets)
		}
	}

	return ResourceSummary{
		GatewaysFound:          len(state.Gateways),
		AgentgatewayBackends:   len(state.AgentgatewayBackends),
		AgentgatewayPolicies:   len(state.AgentgatewayPolicies),
		HTTPRoutes:             len(state.HTTPRoutes),
		KagentAgents:           len(state.KagentAgents),
		KagentMCPServers:       len(state.KagentMCPServers),
		KagentRemoteMCPServers: len(state.KagentRemoteMCPServers),
		TotalMCPEndpoints:      totalMCP,
	}
}

// ---------- GOVERNANCE CHECKS ----------

func checkAgentGatewayCompliance(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	// CRITICAL: No agentgateway Gateway found
	if len(state.Gateways) == 0 {
		findings = append(findings, Finding{
			ID:          "AGW-001",
			Severity:    SeverityCritical,
			Category:    CategoryAgentGateway,
			Title:       "No agentgateway Gateway detected",
			Description: "No Gateway resource with gatewayClassName 'agentgateway' was found in the cluster. All MCP communication must be routed through agentgateway.",
			Impact:      "MCP servers and agents have no centralized security enforcement point. All MCP traffic is ungoverned.",
			Remediation: "Deploy an agentgateway Gateway: kubectl apply -f gateway.yaml with gatewayClassName: agentgateway",
			Timestamp:   ts,
		})
	}

	// Check for agentgateway class specifically
	hasAgentGatewayClass := false
	for _, gw := range state.Gateways {
		if gw.GatewayClassName == "agentgateway" {
			hasAgentGatewayClass = true
			if !gw.Programmed {
				findings = append(findings, Finding{
					ID:          "AGW-002",
					Severity:    SeverityHigh,
					Category:    CategoryAgentGateway,
					Title:       fmt.Sprintf("agentgateway '%s' is not programmed", gw.Name),
					Description: fmt.Sprintf("Gateway '%s/%s' exists but is not in Programmed state. MCP traffic routing may be disrupted.", gw.Namespace, gw.Name),
					Impact:      "MCP traffic cannot be properly routed through agentgateway enforcement point.",
					Remediation: "Check agentgateway controller logs and verify the Gateway resource status.",
					ResourceRef: fmt.Sprintf("Gateway/%s/%s", gw.Namespace, gw.Name),
					Namespace:   gw.Namespace,
					Timestamp:   ts,
				})
			}
		}
	}
	if !hasAgentGatewayClass && len(state.Gateways) > 0 {
		findings = append(findings, Finding{
			ID:          "AGW-003",
			Severity:    SeverityCritical,
			Category:    CategoryAgentGateway,
			Title:       "No Gateway using agentgateway GatewayClass",
			Description: "Gateway resources exist but none use the 'agentgateway' GatewayClass. MCP governance requires agentgateway as the control plane.",
			Impact:      "MCP traffic is not being processed by the agentgateway data plane.",
			Remediation: "Create a Gateway with gatewayClassName: agentgateway",
			Timestamp:   ts,
		})
	}

	// CRITICAL: kagent MCP servers not routed through agentgateway
	if policy.RequireAgentGateway {
		for _, mcp := range state.KagentMCPServers {
			routed := isMCPServerRouted(mcp, state)
			if !routed {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("AGW-100-%s", mcp.Name),
					Severity:    SeverityCritical,
					Category:    CategoryAgentGateway,
					Title:       fmt.Sprintf("MCPServer '%s' bypasses agentgateway", mcp.Name),
					Description: fmt.Sprintf("kagent MCPServer '%s/%s' is deployed but has no AgentgatewayBackend or HTTPRoute routing traffic through agentgateway.", mcp.Namespace, mcp.Name),
					Impact:      "This MCP server operates outside governance. No authentication, authorization, rate limiting, or observability is applied.",
					Remediation: "Create an AgentgatewayBackend with mcp targets pointing to this server's Service, and an HTTPRoute to route through agentgateway.",
					ResourceRef: fmt.Sprintf("MCPServer/%s/%s", mcp.Namespace, mcp.Name),
					Namespace:   mcp.Namespace,
					Timestamp:   ts,
				})
			}
		}

		// Check kagent agents referencing MCP tools not through agentgateway
		for _, agent := range state.KagentAgents {
			for _, tool := range agent.Tools {
				if tool.Type == "McpServer" && tool.Kind == "MCPServer" {
					// MCPServer tools should ideally route through agentgateway
					// Add discovery label check
					findings = append(findings, Finding{
						ID:          fmt.Sprintf("AGW-200-%s-%s", agent.Name, tool.Name),
						Severity:    SeverityMedium,
						Category:    CategoryAgentGateway,
						Title:       fmt.Sprintf("Agent '%s' uses MCPServer '%s' - verify agentgateway routing", agent.Name, tool.Name),
						Description: fmt.Sprintf("Agent '%s/%s' references MCPServer '%s'. Ensure the kagent.dev/discovery=disabled label is set and traffic routes through agentgateway.", agent.Namespace, agent.Name, tool.Name),
						Impact:      "If MCP traffic bypasses agentgateway, security policies are not enforced.",
						Remediation: "Add kagent.dev/discovery=disabled label to MCPServer and configure AgentgatewayBackend routing.",
						ResourceRef: fmt.Sprintf("Agent/%s/%s", agent.Namespace, agent.Name),
						Namespace:   agent.Namespace,
						Timestamp:   ts,
					})
				}
			}
		}
	}

	// No AgentgatewayBackend with MCP type
	mcpBackends := 0
	for _, b := range state.AgentgatewayBackends {
		if b.BackendType == "mcp" {
			mcpBackends++
		}
	}
	if mcpBackends == 0 && (len(state.KagentMCPServers) > 0 || len(state.KagentRemoteMCPServers) > 0) {
		findings = append(findings, Finding{
			ID:          "AGW-004",
			Severity:    SeverityHigh,
			Category:    CategoryAgentGateway,
			Title:       "No MCP-type AgentgatewayBackend configured",
			Description: "MCP servers exist in the cluster but no AgentgatewayBackend of type 'mcp' is configured to route their traffic.",
			Impact:      "MCP servers are accessible directly without agentgateway governance.",
			Remediation: "Create AgentgatewayBackend resources with spec.mcp.targets pointing to your MCP server Services.",
			Timestamp:   ts,
		})
	}

	return findings
}

func checkAuthentication(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	if !policy.RequireJWTAuth {
		return findings
	}

	hasJWTPolicy := false
	for _, p := range state.AgentgatewayPolicies {
		if p.HasJWT {
			hasJWTPolicy = true
			if p.JWTMode == "Optional" || p.JWTMode == "Permissive" {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("AUTH-001-%s", p.Name),
					Severity:    SeverityHigh,
					Category:    CategoryAuthentication,
					Title:       fmt.Sprintf("JWT auth mode is '%s' on policy '%s'", p.JWTMode, p.Name),
					Description: fmt.Sprintf("AgentgatewayPolicy '%s/%s' has JWT authentication in '%s' mode. This allows unauthenticated requests.", p.Namespace, p.Name, p.JWTMode),
					Impact:      "MCP endpoints accept requests without valid JWT tokens, allowing unauthorized access.",
					Remediation: "Set jwtAuthentication.mode to 'Strict' in the AgentgatewayPolicy.",
					ResourceRef: fmt.Sprintf("AgentgatewayPolicy/%s/%s", p.Namespace, p.Name),
					Namespace:   p.Namespace,
					Timestamp:   ts,
				})
			}
		}
	}

	if !hasJWTPolicy {
		findings = append(findings, Finding{
			ID:          "AUTH-002",
			Severity:    SeverityCritical,
			Category:    CategoryAuthentication,
			Title:       "No JWT authentication configured",
			Description: "No AgentgatewayPolicy with JWT authentication was found. All MCP endpoints are unauthenticated.",
			Impact:      "Any client can access MCP tools without presenting valid credentials.",
			Remediation: "Create an AgentgatewayPolicy with traffic.jwtAuthentication targeting your Gateway or HTTPRoutes.",
			Timestamp:   ts,
		})
	}

	// Check MCP backends for MCP-level authentication
	for _, b := range state.AgentgatewayBackends {
		if b.BackendType == "mcp" {
			for _, t := range b.MCPTargets {
				if !t.HasAuth {
					findings = append(findings, Finding{
						ID:          fmt.Sprintf("AUTH-100-%s-%s", b.Name, t.Name),
						Severity:    SeverityMedium,
						Category:    CategoryAuthentication,
						Title:       fmt.Sprintf("MCP target '%s' in backend '%s' has no MCP-level auth", t.Name, b.Name),
						Description: fmt.Sprintf("AgentgatewayBackend '%s' MCP target '%s' does not configure MCP-spec authentication (OAuth/OIDC).", b.Name, t.Name),
						Impact:      "MCP-level authentication is not enforced. Relies solely on transport-level auth.",
						Remediation: "Configure backend.mcp.authentication with provider (Auth0/Keycloak) and issuer in the AgentgatewayBackend or AgentgatewayPolicy.",
						ResourceRef: fmt.Sprintf("AgentgatewayBackend/%s/%s", b.Namespace, b.Name),
						Namespace:   b.Namespace,
						Timestamp:   ts,
					})
				}
			}
		}
	}

	return findings
}

func checkAuthorization(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	if !policy.RequireRBAC {
		return findings
	}

	// If no agentgateway infrastructure exists at all, authorization cannot be enforced
	if len(state.AgentgatewayPolicies) == 0 && len(state.AgentgatewayBackends) == 0 {
		findings = append(findings, Finding{
			ID:          "RBAC-002",
			Severity:    SeverityCritical,
			Category:    CategoryAuthorization,
			Title:       "No agentgateway infrastructure for authorization enforcement",
			Description: "RBAC is required by policy but no AgentgatewayPolicies or AgentgatewayBackends exist. Authorization cannot be enforced without agentgateway infrastructure.",
			Impact:      "All MCP tool access is completely unrestricted — no role-based access control is possible.",
			Remediation: "Deploy agentgateway with Gateway, AgentgatewayBackend, and AgentgatewayPolicy resources with authorization rules.",
			Timestamp:   ts,
		})
		return findings
	}

	hasRBACPolicy := false
	for _, p := range state.AgentgatewayPolicies {
		if p.HasRBAC {
			hasRBACPolicy = true
		}
	}

	// Check MCP backend tool-level authorization
	for _, b := range state.AgentgatewayBackends {
		if b.BackendType == "mcp" {
			for _, t := range b.MCPTargets {
				if !t.HasRBAC {
					findings = append(findings, Finding{
						ID:          fmt.Sprintf("RBAC-100-%s-%s", b.Name, t.Name),
						Severity:    SeverityHigh,
						Category:    CategoryAuthorization,
						Title:       fmt.Sprintf("No CEL-based tool access control on MCP target '%s'", t.Name),
						Description: fmt.Sprintf("MCP target '%s' in AgentgatewayBackend '%s' has no authorization.matchExpressions for tool-level access control.", t.Name, b.Name),
						Impact:      "All authenticated users can access all tools on this MCP server without restriction.",
						Remediation: "Add backend.mcp.authorization with CEL matchExpressions like 'jwt.sub == \"admin\" && mcp.tool.name == \"sensitive_tool\"' to the AgentgatewayPolicy targeting this backend.",
						ResourceRef: fmt.Sprintf("AgentgatewayBackend/%s/%s", b.Namespace, b.Name),
						Namespace:   b.Namespace,
						Timestamp:   ts,
					})
				}
			}
		}
	}

	if !hasRBACPolicy && len(state.AgentgatewayBackends) > 0 {
		findings = append(findings, Finding{
			ID:          "RBAC-001",
			Severity:    SeverityHigh,
			Category:    CategoryAuthorization,
			Title:       "No authorization policies configured",
			Description: "No AgentgatewayPolicy with authorization rules was found. MCP tool access is unrestricted.",
			Impact:      "Any authenticated user can access any MCP tool without role-based restrictions.",
			Remediation: "Create an AgentgatewayPolicy with traffic.authorization targeting your MCP backends.",
			Timestamp:   ts,
		})
	}

	return findings
}

func checkCORS(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	if !policy.RequireCORS {
		return findings
	}

	// If no agentgateway infrastructure exists, CORS cannot be enforced
	if len(state.AgentgatewayPolicies) == 0 && len(state.HTTPRoutes) == 0 {
		findings = append(findings, Finding{
			ID:          "CORS-003",
			Severity:    SeverityHigh,
			Category:    CategoryCORS,
			Title:       "No agentgateway infrastructure for CORS enforcement",
			Description: "CORS policy is required but no AgentgatewayPolicies or HTTPRoutes exist. CORS headers cannot be enforced without agentgateway infrastructure.",
			Impact:      "Browser-based MCP clients have no cross-origin protection.",
			Remediation: "Deploy agentgateway with CORS configuration in AgentgatewayPolicy or HTTPRoute CORS filters.",
			Timestamp:   ts,
		})
		return findings
	}

	hasCORS := false
	for _, p := range state.AgentgatewayPolicies {
		if p.HasCORS {
			hasCORS = true
		}
	}
	for _, r := range state.HTTPRoutes {
		if r.HasCORSFilter {
			hasCORS = true
		}
	}

	if !hasCORS {
		findings = append(findings, Finding{
			ID:          "CORS-001",
			Severity:    SeverityMedium,
			Category:    CategoryCORS,
			Title:       "No CORS policy configured for MCP endpoints",
			Description: "No AgentgatewayPolicy or HTTPRoute with CORS configuration was found. MCP endpoints may be vulnerable to cross-origin attacks.",
			Impact:      "Browser-based MCP clients may be susceptible to cross-site request forgery.",
			Remediation: "Add a CORS filter to your HTTPRoute or create an AgentgatewayPolicy with traffic.cors configuration.",
			Timestamp:   ts,
		})
	}

	// Check for CSRF protection
	hasCSRF := false
	for _, p := range state.AgentgatewayPolicies {
		if p.HasCSRF {
			hasCSRF = true
		}
	}
	if !hasCSRF && hasCORS {
		findings = append(findings, Finding{
			ID:          "CORS-002",
			Severity:    SeverityLow,
			Category:    CategoryCORS,
			Title:       "CSRF protection not configured alongside CORS",
			Description: "CORS is configured but no CSRF protection (AgentgatewayPolicy traffic.csrf) was found.",
			Impact:      "Cross-site request forgery attacks may still be possible.",
			Remediation: "Add traffic.csrf configuration to your AgentgatewayPolicy.",
			Timestamp:   ts,
		})
	}

	return findings
}

func checkTLS(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	if !policy.RequireTLS {
		return findings
	}

	// If no agentgateway backends exist, TLS cannot be enforced on backend connections
	if len(state.AgentgatewayBackends) == 0 {
		findings = append(findings, Finding{
			ID:          "TLS-002",
			Severity:    SeverityHigh,
			Category:    CategoryTLS,
			Title:       "No agentgateway backends for TLS enforcement",
			Description: "TLS is required by policy but no AgentgatewayBackends exist. TLS encryption cannot be enforced on MCP traffic without agentgateway infrastructure.",
			Impact:      "MCP traffic is not encrypted — data in transit is exposed.",
			Remediation: "Deploy agentgateway with AgentgatewayBackend resources configured with TLS.",
			Timestamp:   ts,
		})
		return findings
	}

	for _, b := range state.AgentgatewayBackends {
		if !b.HasTLS {
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("TLS-001-%s", b.Name),
				Severity:    SeverityHigh,
				Category:    CategoryTLS,
				Title:       fmt.Sprintf("Backend '%s' does not enforce TLS", b.Name),
				Description: fmt.Sprintf("AgentgatewayBackend '%s/%s' does not configure TLS for backend connections.", b.Namespace, b.Name),
				Impact:      "MCP traffic between agentgateway and backend MCP servers is unencrypted.",
				Remediation: "Configure policies.tls in the AgentgatewayBackend or attach an AgentgatewayPolicy with backend TLS settings.",
				ResourceRef: fmt.Sprintf("AgentgatewayBackend/%s/%s", b.Namespace, b.Name),
				Namespace:   b.Namespace,
				Timestamp:   ts,
			})
		}
	}

	return findings
}

func checkPromptGuard(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	if !policy.RequirePromptGuard {
		return findings
	}

	// If no agentgateway policies exist, prompt guard cannot be enforced
	if len(state.AgentgatewayPolicies) == 0 {
		findings = append(findings, Finding{
			ID:          "PG-002",
			Severity:    SeverityHigh,
			Category:    CategoryPromptGuard,
			Title:       "No agentgateway infrastructure for prompt guard enforcement",
			Description: "Prompt guard is required by policy but no AgentgatewayPolicies exist. Prompt injection and sensitive data detection cannot be enforced without agentgateway.",
			Impact:      "LLM requests and responses are not inspected for sensitive data or prompt injection attacks.",
			Remediation: "Deploy agentgateway with AgentgatewayPolicy resources configured with prompt guard rules.",
			Timestamp:   ts,
		})
		return findings
	}

	hasPromptGuard := false
	for _, p := range state.AgentgatewayPolicies {
		if p.HasPromptGuard {
			hasPromptGuard = true
		}
	}

	if !hasPromptGuard {
		findings = append(findings, Finding{
			ID:          "PG-001",
			Severity:    SeverityMedium,
			Category:    CategoryPromptGuard,
			Title:       "No prompt guard policies configured",
			Description: "No AgentgatewayPolicy with prompt guard (regex matching, content moderation) was found for AI backends.",
			Impact:      "LLM requests may contain sensitive data (credit cards, SSNs) without detection or masking.",
			Remediation: "Add backend.ai.promptGuard with request/response regex rules or OpenAI moderation to your AgentgatewayPolicy.",
			Timestamp:   ts,
		})
	}

	return findings
}

func checkRateLimit(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	if !policy.RequireRateLimit {
		return findings
	}

	// If no agentgateway policies exist, rate limiting cannot be enforced
	if len(state.AgentgatewayPolicies) == 0 {
		findings = append(findings, Finding{
			ID:          "RL-002",
			Severity:    SeverityHigh,
			Category:    CategoryRateLimit,
			Title:       "No agentgateway infrastructure for rate limit enforcement",
			Description: "Rate limiting is required by policy but no AgentgatewayPolicies exist. Rate limiting cannot be enforced without agentgateway.",
			Impact:      "MCP endpoints have no request rate controls — vulnerable to abuse and resource exhaustion.",
			Remediation: "Deploy agentgateway with AgentgatewayPolicy resources configured with rate limit rules.",
			Timestamp:   ts,
		})
		return findings
	}

	hasRateLimit := false
	for _, p := range state.AgentgatewayPolicies {
		if p.HasRateLimit {
			hasRateLimit = true
		}
	}

	if !hasRateLimit {
		findings = append(findings, Finding{
			ID:          "RL-001",
			Severity:    SeverityMedium,
			Category:    CategoryRateLimit,
			Title:       "No rate limiting configured for MCP endpoints",
			Description: "No AgentgatewayPolicy with rate limiting was found. MCP endpoints are vulnerable to abuse.",
			Impact:      "Unbounded request rates to MCP tools may lead to resource exhaustion or cost overruns.",
			Remediation: "Add traffic.rateLimit with local or global rate limiting rules to your AgentgatewayPolicy.",
			Timestamp:   ts,
		})
	}

	return findings
}

func checkToolCount(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	// Skip if thresholds are not configured (both 0 means disabled)
	if policy.MaxToolsWarning <= 0 && policy.MaxToolsCritical <= 0 {
		return findings
	}

	// Check RemoteMCPServers for excessive tool counts
	for _, rms := range state.KagentRemoteMCPServers {
		if rms.ToolCount <= 0 {
			continue // No discovered tools or not available
		}

		if policy.MaxToolsCritical > 0 && rms.ToolCount > policy.MaxToolsCritical {
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("TOOLS-001-%s", rms.Name),
				Severity:    SeverityCritical,
				Category:    CategoryToolScope,
				Title:       fmt.Sprintf("RemoteMCPServer '%s' exposes %d tools (threshold: %d)", rms.Name, rms.ToolCount, policy.MaxToolsCritical),
				Description: fmt.Sprintf("RemoteMCPServer '%s/%s' has %d discovered tools, exceeding the critical threshold of %d. Excessive tool exposure increases the attack surface and makes authorization harder to manage.", rms.Namespace, rms.Name, rms.ToolCount, policy.MaxToolsCritical),
				Impact:      "Large tool surface increases risk of unauthorized tool invocation, prompt injection via tool descriptions, and makes least-privilege access control impractical.",
				Remediation: fmt.Sprintf("Split the MCP server into smaller, focused servers with fewer tools. Consider using toolNames in agent tool references to limit exposed tools to only those needed. Target: ≤%d tools per server.", policy.MaxToolsCritical),
				ResourceRef: fmt.Sprintf("RemoteMCPServer/%s/%s", rms.Namespace, rms.Name),
				Namespace:   rms.Namespace,
				Timestamp:   ts,
			})
		} else if policy.MaxToolsWarning > 0 && rms.ToolCount > policy.MaxToolsWarning {
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("TOOLS-001-%s", rms.Name),
				Severity:    SeverityMedium,
				Category:    CategoryToolScope,
				Title:       fmt.Sprintf("RemoteMCPServer '%s' exposes %d tools (threshold: %d)", rms.Name, rms.ToolCount, policy.MaxToolsWarning),
				Description: fmt.Sprintf("RemoteMCPServer '%s/%s' has %d discovered tools, exceeding the warning threshold of %d. Consider splitting into focused MCP servers.", rms.Namespace, rms.Name, rms.ToolCount, policy.MaxToolsWarning),
				Impact:      "Moderately large tool surface may make authorization management complex and increases potential attack vectors.",
				Remediation: fmt.Sprintf("Review the tools exposed by this MCP server and consider splitting into focused servers with ≤%d tools each.", policy.MaxToolsWarning),
				ResourceRef: fmt.Sprintf("RemoteMCPServer/%s/%s", rms.Namespace, rms.Name),
				Namespace:   rms.Namespace,
				Timestamp:   ts,
			})
		}
	}

	return findings
}

func checkExposure(state *ClusterState, policy Policy) []Finding {
	var findings []Finding
	ts := time.Now().Format(time.RFC3339)

	// Check RemoteMCPServers — their URLs should route through agentgateway
	if policy.RequireAgentGateway {
		// Determine severity: if no agentgateway exists at all, it's Critical
		hasAgentGateway := false
		for _, gw := range state.Gateways {
			if gw.GatewayClassName == "agentgateway" {
				hasAgentGateway = true
				break
			}
		}
		exposureSeverity := SeverityHigh
		if !hasAgentGateway {
			exposureSeverity = SeverityCritical
		}

		for _, rms := range state.KagentRemoteMCPServers {
			// Check if the RemoteMCPServer URL points through agentgateway
			routedThroughGateway := false
			for _, gw := range state.Gateways {
				if gw.GatewayClassName == "agentgateway" {
					// Check if the URL references the agentgateway service
					for _, svc := range state.Services {
						if svc.Name == "agentgateway" || svc.Name == gw.Name {
							// URL should contain the agentgateway service name
							if containsHost(rms.URL, svc.Name, svc.Namespace) {
								routedThroughGateway = true
							}
						}
					}
				}
			}

			if !routedThroughGateway {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("EXP-001-%s", rms.Name),
					Severity:    exposureSeverity,
					Category:    CategoryExposure,
					Title:       fmt.Sprintf("RemoteMCPServer '%s' not routed through agentgateway", rms.Name),
					Description: fmt.Sprintf("RemoteMCPServer '%s/%s' has URL '%s' which does not point to agentgateway. MCP traffic should be routed through agentgateway for governance enforcement.", rms.Namespace, rms.Name, rms.URL),
					Impact:      "MCP tool calls bypass agentgateway governance — no authentication, authorization, or rate limiting is applied.",
					Remediation: "Update the RemoteMCPServer URL to point to the agentgateway service endpoint (e.g., http://agentgateway.agentgateway-system:8080/mcp/<backend-name>/<target>).",
					ResourceRef: fmt.Sprintf("RemoteMCPServer/%s/%s", rms.Namespace, rms.Name),
					Namespace:   rms.Namespace,
					Timestamp:   ts,
				})
			}
		}
	}

	return findings
}

// containsHost checks if a URL references a K8s service by name pattern
func containsHost(url, svcName, svcNamespace string) bool {
	patterns := []string{
		fmt.Sprintf("%s.%s", svcName, svcNamespace),
		fmt.Sprintf("%s.%s.svc", svcName, svcNamespace),
	}
	for _, p := range patterns {
		if len(url) > 0 && len(p) > 0 && stringContains(url, p) {
			return true
		}
	}
	return false
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------- SCORING ----------

func calculateScores(state *ClusterState, findings []Finding, policy Policy) ScoreBreakdown {
	breakdown := ScoreBreakdown{
		InfraAbsent: make(map[string]bool),
	}

	apply := func(name string, r categoryScoreResult, target *int) {
		*target = r.Score
		if r.InfraAbsent {
			breakdown.InfraAbsent[name] = true
		}
	}

	if policy.RequireAgentGateway {
		apply("AgentGateway Compliance", calculateCategoryScore(CategoryAgentGateway, CategoryExposure, findings, state, policy), &breakdown.AgentGatewayScore)
	}
	if policy.RequireJWTAuth {
		apply("Authentication", calculateCategoryScore(CategoryAuthentication, "", findings, state, policy), &breakdown.AuthenticationScore)
	}
	if policy.RequireRBAC {
		apply("Authorization", calculateCategoryScore(CategoryAuthorization, "", findings, state, policy), &breakdown.AuthorizationScore)
	}
	if policy.RequireCORS {
		apply("CORS", calculateCategoryScore(CategoryCORS, "", findings, state, policy), &breakdown.CORSScore)
	}
	if policy.RequireTLS {
		apply("TLS", calculateCategoryScore(CategoryTLS, "", findings, state, policy), &breakdown.TLSScore)
	}
	if policy.RequirePromptGuard {
		apply("Prompt Guard", calculateCategoryScore(CategoryPromptGuard, "", findings, state, policy), &breakdown.PromptGuardScore)
	}
	if policy.RequireRateLimit {
		apply("Rate Limit", calculateCategoryScore(CategoryRateLimit, "", findings, state, policy), &breakdown.RateLimitScore)
	}
	if policy.MaxToolsWarning > 0 || policy.MaxToolsCritical > 0 {
		apply("Tool Scope", calculateCategoryScore(CategoryToolScope, "", findings, state, policy), &breakdown.ToolScopeScore)
	}

	return breakdown
}

// calculateCategoryScore calculates score for a category.
// If no findings exist = fully compliant = 100.
// If ANY Critical finding exists for "no infrastructure" = 0 (nothing is deployed).
// Otherwise, deduct from 100 based on findings severity (partial compliance).
func calculateCategoryScore(primaryCategory, secondaryCategory string, findings []Finding, state *ClusterState, policy Policy) categoryScoreResult {
	// Collect findings for this category
	var categoryFindings []Finding
	for _, f := range findings {
		if f.Category == primaryCategory || (secondaryCategory != "" && f.Category == secondaryCategory) {
			categoryFindings = append(categoryFindings, f)
		}
	}

	// No findings = fully compliant
	if len(categoryFindings) == 0 {
		return categoryScoreResult{Score: 100}
	}

	// Check if any finding indicates total absence of infrastructure.
	// These are the "no infrastructure" findings that mean score = 0.
	for _, f := range categoryFindings {
		if isInfrastructureAbsenceFinding(f) {
			return categoryScoreResult{Score: 0, InfraAbsent: true}
		}
	}

	// Partial compliance: infrastructure exists but has issues
	penalty := 0
	for _, f := range categoryFindings {
		penalty += severityPenalty(f.Severity, policy.SeverityPenalties)
	}

	score := 100 - penalty
	if score < 0 {
		return categoryScoreResult{Score: 0}
	}
	return categoryScoreResult{Score: score}
}

// isInfrastructureAbsenceFinding returns true for findings that indicate
// the required infrastructure is completely missing (not just misconfigured).
// When infrastructure is absent, the category score should be 0.
func isInfrastructureAbsenceFinding(f Finding) bool {
	// These IDs represent "no infrastructure at all" findings
	switch f.ID {
	case "AGW-001",  // No agentgateway Gateway detected
		"AGW-003",   // No Gateway using agentgateway GatewayClass
		"AGW-004",   // No MCP-type AgentgatewayBackend configured
		"AUTH-002",  // No JWT authentication configured
		"RBAC-002",  // No agentgateway infrastructure for authorization
		"CORS-003",  // No agentgateway infrastructure for CORS
		"TLS-002",   // No agentgateway backends for TLS
		"PG-002",    // No agentgateway infrastructure for prompt guard
		"RL-002":    // No agentgateway infrastructure for rate limiting
		return true
	default:
		return false
	}
}

func calculateOverallScore(breakdown ScoreBreakdown, weights ScoringWeights, policy Policy) int {
	totalWeight := 0
	weightedScore := 0

	if policy.RequireAgentGateway {
		totalWeight += weights.AgentGatewayIntegration
		weightedScore += breakdown.AgentGatewayScore * weights.AgentGatewayIntegration
	}
	if policy.RequireJWTAuth {
		totalWeight += weights.Authentication
		weightedScore += breakdown.AuthenticationScore * weights.Authentication
	}
	if policy.RequireRBAC {
		totalWeight += weights.Authorization
		weightedScore += breakdown.AuthorizationScore * weights.Authorization
	}
	if policy.RequireCORS {
		totalWeight += weights.CORSPolicy
		weightedScore += breakdown.CORSScore * weights.CORSPolicy
	}
	if policy.RequireTLS {
		totalWeight += weights.TLSEncryption
		weightedScore += breakdown.TLSScore * weights.TLSEncryption
	}
	if policy.RequirePromptGuard {
		totalWeight += weights.PromptGuard
		weightedScore += breakdown.PromptGuardScore * weights.PromptGuard
	}
	if policy.RequireRateLimit {
		totalWeight += weights.RateLimit
		weightedScore += breakdown.RateLimitScore * weights.RateLimit
	}
	if policy.MaxToolsWarning > 0 || policy.MaxToolsCritical > 0 {
		totalWeight += weights.ToolScope
		weightedScore += breakdown.ToolScopeScore * weights.ToolScope
	}

	if totalWeight == 0 {
		return 100 // no requirements = fully compliant
	}

	return weightedScore / totalWeight
}

// aggregateBreakdownFromMCPViews computes the cluster-level ScoreBreakdown
// by averaging the per-server MCPServerScoreBreakdown across all MCP server views.
// This ensures the overview dashboard is consistent with the MCP-server-centric scores.
func aggregateBreakdownFromMCPViews(views []MCPServerView, policy Policy) ScoreBreakdown {
	n := len(views)
	if n == 0 {
		// No MCP servers discovered — mark all required categories as infra-absent
		infraAbsent := make(map[string]bool)
		if policy.RequireAgentGateway {
			infraAbsent["AgentGateway Compliance"] = true
			infraAbsent["Agent Gateway"] = true
		}
		if policy.RequireJWTAuth {
			infraAbsent["Authentication"] = true
		}
		if policy.RequireRBAC {
			infraAbsent["Authorization"] = true
		}
		if policy.RequireTLS {
			infraAbsent["TLS"] = true
		}
		if policy.RequireCORS {
			infraAbsent["CORS"] = true
		}
		if policy.RequireRateLimit {
			infraAbsent["Rate Limit"] = true
		}
		if policy.RequirePromptGuard {
			infraAbsent["Prompt Guard"] = true
		}
		bd := ScoreBreakdown{}
		if len(infraAbsent) > 0 {
			bd.InfraAbsent = infraAbsent
		}
		return bd
	}

	var sumGW, sumAuth, sumAuthz, sumTLS, sumCORS, sumRL, sumPG, sumTool int
	for _, v := range views {
		sumGW += v.ScoreBreakdown.GatewayRouting
		sumAuth += v.ScoreBreakdown.Authentication
		sumAuthz += v.ScoreBreakdown.Authorization
		sumTLS += v.ScoreBreakdown.TLS
		sumCORS += v.ScoreBreakdown.CORS
		sumRL += v.ScoreBreakdown.RateLimit
		sumPG += v.ScoreBreakdown.PromptGuard
		sumTool += v.ScoreBreakdown.ToolScope
	}

	bd := ScoreBreakdown{
		AgentGatewayScore:   sumGW / n,
		AuthenticationScore: sumAuth / n,
		AuthorizationScore:  sumAuthz / n,
		TLSScore:            sumTLS / n,
		CORSScore:           sumCORS / n,
		RateLimitScore:      sumRL / n,
		PromptGuardScore:    sumPG / n,
		ToolScopeScore:      sumTool / n,
	}

	// Determine InfraAbsent: a category is "infra absent" at cluster level
	// only if ALL servers have score 0 for that category.
	infraAbsent := make(map[string]bool)
	categoryChecks := []struct {
		name     string
		required bool
		score    int
	}{
		{"Agent Gateway", policy.RequireAgentGateway, bd.AgentGatewayScore},
		{"Authentication", policy.RequireJWTAuth, bd.AuthenticationScore},
		{"Authorization", policy.RequireRBAC, bd.AuthorizationScore},
		{"TLS Encryption", policy.RequireTLS, bd.TLSScore},
		{"CORS Policy", policy.RequireCORS, bd.CORSScore},
		{"Rate Limiting", policy.RequireRateLimit, bd.RateLimitScore},
		{"Prompt Guard", policy.RequirePromptGuard, bd.PromptGuardScore},
	}
	for _, c := range categoryChecks {
		if c.required && c.score == 0 {
			// Check if ALL servers scored 0 (true infrastructure absence)
			allZero := true
			for _, v := range views {
				var s int
				switch c.name {
				case "Agent Gateway":
					s = v.ScoreBreakdown.GatewayRouting
				case "Authentication":
					s = v.ScoreBreakdown.Authentication
				case "Authorization":
					s = v.ScoreBreakdown.Authorization
				case "TLS Encryption":
					s = v.ScoreBreakdown.TLS
				case "CORS Policy":
					s = v.ScoreBreakdown.CORS
				case "Rate Limiting":
					s = v.ScoreBreakdown.RateLimit
				case "Prompt Guard":
					s = v.ScoreBreakdown.PromptGuard
				}
				if s > 0 {
					allZero = false
					break
				}
			}
			if allZero {
				infraAbsent[c.name] = true
			}
		}
	}
	if len(infraAbsent) > 0 {
		bd.InfraAbsent = infraAbsent
	}

	return bd
}

func calculateNamespaceScores(state *ClusterState, findings []Finding, penalties SeverityPenalties) []NamespaceScore {
	nsFindings := make(map[string][]Finding)
	for _, f := range findings {
		if f.Namespace != "" {
			nsFindings[f.Namespace] = append(nsFindings[f.Namespace], f)
		}
	}

	var scores []NamespaceScore
	for _, ns := range state.Namespaces {
		score := 100
		nf := nsFindings[ns]
		for _, f := range nf {
			score -= severityPenalty(f.Severity, penalties)
		}
		if score < 0 {
			score = 0
		}
		scores = append(scores, NamespaceScore{
			Namespace: ns,
			Score:     score,
			Findings:  len(nf),
		})
	}

	return scores
}

func severityPenalty(severity string, penalties SeverityPenalties) int {
	switch severity {
	case SeverityCritical:
		return penalties.Critical
	case SeverityHigh:
		return penalties.High
	case SeverityMedium:
		return penalties.Medium
	case SeverityLow:
		return penalties.Low
	default:
		return 0
	}
}

// Helper to check if a kagent MCPServer is routed through agentgateway
func isMCPServerRouted(mcp KagentMCPServerResource, state *ClusterState) bool {
	for _, b := range state.AgentgatewayBackends {
		if b.BackendType == "mcp" {
			for _, t := range b.MCPTargets {
				// Match by host pattern: <name>.<namespace>.svc.cluster.local
				expectedHost := fmt.Sprintf("%s.%s.svc.cluster.local", mcp.Name, mcp.Namespace)
				if t.Host == expectedHost || t.Host == mcp.Name {
					return true
				}
			}
		}
	}
	return false
}
