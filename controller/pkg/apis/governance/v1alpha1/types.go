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
	Score              int               `json:"score"`
	Phase              string            `json:"phase,omitempty"`
	Findings           []Finding         `json:"findings,omitempty"`
	ResourceSummary    ResourceSummary   `json:"resourceSummary,omitempty"`
	LastEvaluationTime *metav1.Time      `json:"lastEvaluationTime,omitempty"`
	ScoreBreakdown     ScoreBreakdown    `json:"scoreBreakdown,omitempty"`
	NamespaceScores    []NamespaceScore  `json:"namespaceScores,omitempty"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GovernanceEvaluationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GovernanceEvaluation `json:"items"`
}
