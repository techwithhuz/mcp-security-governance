package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MCPGovernancePolicy defines the governance policy for MCP resources
type MCPGovernancePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MCPGovernancePolicySpec   `json:"spec,omitempty"`
	Status            MCPGovernancePolicyStatus `json:"status,omitempty"`
}

type MCPGovernancePolicySpec struct {
	// RequireAgentGateway enforces that all MCP servers must be behind agentgateway
	RequireAgentGateway bool `json:"requireAgentGateway"`
	// RequireCORS enforces CORS policy on all MCP endpoints
	RequireCORS bool `json:"requireCORS"`
	// RequireJWTAuth enforces JWT authentication on all MCP endpoints
	RequireJWTAuth bool `json:"requireJWTAuth"`
	// RequireRBAC enforces RBAC rules on all MCP tool access
	RequireRBAC bool `json:"requireRBAC"`
	// RequirePromptGuard enforces prompt guard policies on AI backends
	RequirePromptGuard bool `json:"requirePromptGuard"`
	// RequireTLS enforces TLS on all MCP connections
	RequireTLS bool `json:"requireTLS"`
	// RequireRateLimit enforces rate limiting on MCP endpoints
	RequireRateLimit bool `json:"requireRateLimit"`
	// AIAgent configures AI-driven governance scoring using Google ADK alongside algorithmic scoring.
	AIAgent *AIAgentConfig `json:"aiAgent,omitempty"`
	// MaxToolsWarning generates a Warning finding if an MCP server exposes more than this many tools (0 = disabled)
	MaxToolsWarning int `json:"maxToolsWarning,omitempty"`
	// MaxToolsCritical generates a Critical finding if an MCP server exposes more than this many tools (0 = disabled)
	MaxToolsCritical int `json:"maxToolsCritical,omitempty"`
	// ScoringWeights defines the weights for the governance scoring model
	ScoringWeights ScoringWeights `json:"scoringWeights,omitempty"`
	// SeverityPenalties defines the point deductions per severity level
	SeverityPenalties SeverityPenalties `json:"severityPenalties,omitempty"`
	// TargetNamespaces is the list of namespaces to monitor (empty = all namespaces)
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`
	// ExcludeNamespaces is the list of namespaces to exclude from monitoring
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`
	// VerifiedCatalogScoring configures scoring thresholds, category weights, and per-check max scores
	// for the Verified Catalog (MCPServerCatalog inventory) scoring model.
	VerifiedCatalogScoring *VerifiedCatalogScoringConfig `json:"verifiedCatalogScoring,omitempty"`
}

// VerifiedCatalogScoringConfig allows users to customise the Verified Catalog scoring model
// via the MCPGovernancePolicy CRD. All fields are optional — omitted values use built-in defaults.
type VerifiedCatalogScoringConfig struct {
	// Category weights (should sum to 100). Controls how the final composite score is computed.
	SecurityWeight   int `json:"securityWeight,omitempty"`   // default: 50 — weight for transport + deployment
	TrustWeight      int `json:"trustWeight,omitempty"`      // default: 30 — weight for publisher verification
	ComplianceWeight int `json:"complianceWeight,omitempty"` // default: 20 — weight for tool scope + usage

	// Status thresholds — score boundaries for Verified / Unverified / Rejected.
	VerifiedThreshold   int `json:"verifiedThreshold,omitempty"`   // score >= this → "Verified" (default: 70)
	UnverifiedThreshold int `json:"unverifiedThreshold,omitempty"` // score >= this → "Unverified" (default: 50)

	// Per-check max score overrides. Keys are check IDs (e.g. "PUB-001", "SEC-001").
	// Each value is the maximum points awarded for that check.
	CheckMaxScores map[string]int `json:"checkMaxScores,omitempty"`
}

type SeverityPenalties struct {
	Critical int `json:"critical,omitempty"` // default: 40
	High     int `json:"high,omitempty"`     // default: 25
	Medium   int `json:"medium,omitempty"`   // default: 15
	Low      int `json:"low,omitempty"`      // default: 5
}

type ScoringWeights struct {
	AgentGatewayIntegration int `json:"agentGatewayIntegration,omitempty"` // default: 25
	Authentication          int `json:"authentication,omitempty"`          // default: 20
	Authorization           int `json:"authorization,omitempty"`           // default: 15
	CORSPolicy              int `json:"corsPolicy,omitempty"`              // default: 10
	TLSEncryption           int `json:"tlsEncryption,omitempty"`           // default: 10
	PromptGuard             int `json:"promptGuard,omitempty"`             // default: 10
	RateLimit               int `json:"rateLimit,omitempty"`               // default: 5
	ToolScope               int `json:"toolScope,omitempty"`               // default: 5
}

// AIAgentConfig configures the AI-driven governance scoring agent
type AIAgentConfig struct {
	// Enabled toggles AI agent scoring on/off
	Enabled bool `json:"enabled"`
	// Provider selects the LLM provider: "gemini" or "ollama"
	Provider string `json:"provider,omitempty"` // default: "gemini"
	// Model is the model name to use (e.g. "gemini-2.5-flash", "llama3.1", "qwen2.5")
	Model string `json:"model,omitempty"` // default: "gemini-2.5-flash"
	// OllamaEndpoint is the base URL for the Ollama API (only used when provider is "ollama")
	OllamaEndpoint string `json:"ollamaEndpoint,omitempty"` // default: "http://localhost:11434"
	// ScanInterval is the interval between periodic AI evaluations (e.g. "5m", "10m", "1h"). Default: "5m"
	ScanInterval string `json:"scanInterval,omitempty"`
	// ScanEnabled controls whether periodic AI scanning is active. Default: true
	ScanEnabled *bool `json:"scanEnabled,omitempty"`
}

type MCPGovernancePolicyStatus struct {
	Phase              string             `json:"phase,omitempty"`
	ClusterScore       int                `json:"clusterScore,omitempty"`
	LastEvaluationTime *metav1.Time       `json:"lastEvaluationTime,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MCPGovernancePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPGovernancePolicy `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GovernanceEvaluation captures the result of a governance evaluation
type GovernanceEvaluation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GovernanceEvaluationSpec   `json:"spec,omitempty"`
	Status            GovernanceEvaluationStatus `json:"status,omitempty"`
}

type GovernanceEvaluationSpec struct {
	// PolicyRef references the governance policy used
	PolicyRef string `json:"policyRef"`
	// EvaluationScope is cluster, namespace, or resource
	EvaluationScope string `json:"evaluationScope"`
	// TargetRef identifies the evaluated resource
	TargetRef TargetRef `json:"targetRef,omitempty"`
}

type TargetRef struct {
	APIGroup  string `json:"apiGroup"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type GovernanceEvaluationStatus struct {
	Score                   int                      `json:"score"`
	Phase                   string                   `json:"phase,omitempty"`
	Findings                []Finding                `json:"findings,omitempty"`
	ResourceSummary         ResourceSummary          `json:"resourceSummary,omitempty"`
	LastEvaluationTime      *metav1.Time             `json:"lastEvaluationTime,omitempty"`
	ScoreBreakdown          ScoreBreakdown           `json:"scoreBreakdown,omitempty"`
	NamespaceScores         []NamespaceScore         `json:"namespaceScores,omitempty"`
	VerifiedCatalogScores   []VerifiedCatalogScore   `json:"verifiedCatalogScores,omitempty"`
	MCPServerScores         []MCPServerScore         `json:"mcpServerScores,omitempty"`
	FindingsCount           int                      `json:"findingsCount,omitempty"`
}

type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`    // Critical, High, Medium, Low
	Category    string `json:"category"`    // AgentGateway, Authentication, Authorization, CORS, TLS, PromptGuard, RateLimit
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Remediation string `json:"remediation"`
	ResourceRef string `json:"resourceRef,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type ResourceSummary struct {
	// agentgateway resources
	GatewaysFound          int `json:"gatewaysFound"`
	AgentgatewayBackends   int `json:"agentgatewayBackends"`
	AgentgatewayPolicies   int `json:"agentgatewayPolicies"`
	HTTPRoutes             int `json:"httpRoutes"`
	// kagent resources
	KagentAgents           int `json:"kagentAgents"`
	KagentMCPServers       int `json:"kagentMCPServers"`
	KagentRemoteMCPServers int `json:"kagentRemoteMCPServers"`
	// Compliance
	CompliantResources     int `json:"compliantResources"`
	NonCompliantResources  int `json:"nonCompliantResources"`
	TotalMCPEndpoints      int `json:"totalMCPEndpoints"`
	ExposedMCPEndpoints    int `json:"exposedMCPEndpoints"`
}

type ScoreBreakdown struct {
	AgentGatewayScore    int `json:"agentGatewayScore"`
	AuthenticationScore  int `json:"authenticationScore"`
	AuthorizationScore   int `json:"authorizationScore"`
	CORSScore            int `json:"corsScore"`
	TLSScore             int `json:"tlsScore"`
	PromptGuardScore     int `json:"promptGuardScore"`
	RateLimitScore       int `json:"rateLimitScore"`
	ToolScopeScore       int `json:"toolScopeScore"`
}

type NamespaceScore struct {
	Namespace string `json:"namespace"`
	Score     int    `json:"score"`
	Findings  int    `json:"findings"`
}

// VerifiedCatalogScore stores the governance score for an MCPServerCatalog resource
type VerifiedCatalogScore struct {
	CatalogName      string                    `json:"catalogName"`
	Namespace        string                    `json:"namespace"`
	ResourceVersion  string                    `json:"resourceVersion,omitempty"`
	Status           string                    `json:"status"` // "Verified", "Unverified", "Rejected", "Pending"
	CompositeScore   int                       `json:"compositeScore"`
	SecurityScore    int                       `json:"securityScore"`    // 0-100 (transport + deployment)
	TrustScore       int                       `json:"trustScore"`       // 0-100 (publisher verification)
	ComplianceScore  int                       `json:"complianceScore"`  // 0-100 (tool scope + usage)
	Checks           []CatalogScoringCheck     `json:"checks,omitempty"`
	LastScored       *metav1.Time              `json:"lastScored,omitempty"`
}

// CatalogScoringCheck represents a single governance check for Verified Catalog
type CatalogScoringCheck struct {
	ID        string `json:"id"`   // e.g., "PUB-001", "SEC-001", "TOOL-001"
	Name      string `json:"name"`
	Points    int    `json:"points"`    // Points earned
	MaxPoints int    `json:"maxPoints"` // Maximum possible points
}

// MCPServerScore stores the governance score for an MCP server (KagentMCPServer, RemoteMCPServer, etc.)
type MCPServerScore struct {
	Name              string                    `json:"name"`
	Namespace         string                    `json:"namespace"`
	Source            string                    `json:"source"` // "KagentMCPServer", "RemoteMCPServer", "AgentgatewayBackendTarget"
	Status            string                    `json:"status"` // "compliant", "warning", "failing", "critical"
	Score             int                       `json:"score"`
	ScoreBreakdown    MCPServerScoreBreakdown   `json:"scoreBreakdown,omitempty"`
	ToolCount         int                       `json:"toolCount"`
	EffectiveToolCount int                      `json:"effectiveToolCount"`
	RelatedResources  RelatedResourceSummary    `json:"relatedResources,omitempty"`
	CriticalFindings  int                       `json:"criticalFindings"`
	LastEvaluated     *metav1.Time              `json:"lastEvaluated,omitempty"`
}

// MCPServerScoreBreakdown breaks down the score by governance control
type MCPServerScoreBreakdown struct {
	GatewayRouting int `json:"gatewayRouting"`
	Authentication int `json:"authentication"`
	Authorization  int `json:"authorization"`
	TLS            int `json:"tls"`
	CORS           int `json:"cors"`
	RateLimit      int `json:"rateLimit"`
	PromptGuard    int `json:"promptGuard"`
	ToolScope      int `json:"toolScope"`
}

// RelatedResourceSummary summarizes related resources for an MCP server
type RelatedResourceSummary struct {
	Gateways int `json:"gateways"`
	Backends int `json:"backends"`
	Policies int `json:"policies"`
	Routes   int `json:"routes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GovernanceEvaluationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GovernanceEvaluation `json:"items"`
}
