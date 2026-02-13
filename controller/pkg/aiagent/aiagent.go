package aiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"

	"github.com/techwithhuz/mcp-security-governance/controller/pkg/evaluator"
)

const appName = "mcp-governance-agent"

// AIScoreResult holds the AI agent's governance assessment
type AIScoreResult struct {
	Score       int               `json:"score"`
	Grade       string            `json:"grade"`
	Reasoning   string            `json:"reasoning"`
	Risks       []RiskAssessment  `json:"risks"`
	Suggestions []string          `json:"suggestions"`
	Timestamp   time.Time         `json:"timestamp"`
}

// RiskAssessment represents an AI-identified risk
type RiskAssessment struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// GovernanceAgent wraps the ADK agent for governance scoring
type GovernanceAgent struct {
	agent  agent.Agent
	runner *runner.Runner
	sessService session.Service
}

// ---------- Tool Input/Output Types ----------

// GetClusterStateArgs is empty — the tool needs no input
type GetClusterStateArgs struct{}

// ClusterStateSummary is the output of the get_cluster_state tool
type ClusterStateSummary struct {
	TotalNamespaces          int      `json:"totalNamespaces"`
	Namespaces               []string `json:"namespaces"`
	GatewaysCount            int      `json:"gatewaysCount"`
	AgentgatewayBackends     int      `json:"agentgatewayBackends"`
	AgentgatewayPolicies     int      `json:"agentgatewayPolicies"`
	HTTPRoutes               int      `json:"httpRoutes"`
	KagentAgents             int      `json:"kagentAgents"`
	KagentMCPServers         int      `json:"kagentMCPServers"`
	KagentRemoteMCPServers   int      `json:"kagentRemoteMCPServers"`
	ServicesCount            int      `json:"servicesCount"`
	HasAgentGateway          bool     `json:"hasAgentGateway"`
	GatewaysProgrammed       int      `json:"gatewaysProgrammed"`
	MCPBackendsWithTLS       int      `json:"mcpBackendsWithTLS"`
	MCPBackendsWithoutTLS    int      `json:"mcpBackendsWithoutTLS"`
	PoliciesWithJWT          int      `json:"policiesWithJWT"`
	PoliciesWithCORS         int      `json:"policiesWithCORS"`
	PoliciesWithRBAC         int      `json:"policiesWithRBAC"`
	PoliciesWithRateLimit    int      `json:"policiesWithRateLimit"`
	PoliciesWithPromptGuard  int      `json:"policiesWithPromptGuard"`
	UnroutedMCPServers       int      `json:"unroutedMCPServers"`
}

type GetFindingsArgs struct{}

type FindingsSummary struct {
	TotalFindings  int                       `json:"totalFindings"`
	BySeverity     map[string]int            `json:"bySeverity"`
	ByCategory     map[string]int            `json:"byCategory"`
	CriticalItems  []FindingBrief            `json:"criticalItems"`
	HighItems      []FindingBrief            `json:"highItems"`
}

type FindingBrief struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	ResourceRef string `json:"resourceRef,omitempty"`
}

type GetPolicyArgs struct{}

type PolicySummary struct {
	Name                string            `json:"name"`
	RequireAgentGateway bool              `json:"requireAgentGateway"`
	RequireCORS         bool              `json:"requireCORS"`
	RequireJWTAuth      bool              `json:"requireJWTAuth"`
	RequireRBAC         bool              `json:"requireRBAC"`
	RequirePromptGuard  bool              `json:"requirePromptGuard"`
	RequireTLS          bool              `json:"requireTLS"`
	RequireRateLimit    bool              `json:"requireRateLimit"`
	MaxToolsWarning     int               `json:"maxToolsWarning"`
	MaxToolsCritical    int               `json:"maxToolsCritical"`
	TargetNamespaces    []string          `json:"targetNamespaces"`
	ExcludeNamespaces   []string          `json:"excludeNamespaces"`
	Weights             map[string]int    `json:"weights"`
}

type GetAlgorithmicScoreArgs struct{}

type AlgorithmicScoreSummary struct {
	OverallScore    int            `json:"overallScore"`
	Grade           string         `json:"grade"`
	CategoryScores  map[string]int `json:"categoryScores"`
}

type GetResourceDetailsArgs struct{}

type ResourceDetailsSummary struct {
	Gateways             []ResourceBrief `json:"gateways"`
	AgentgatewayBackends []BackendBrief  `json:"agentgatewayBackends"`
	AgentgatewayPolicies []PolicyBrief   `json:"agentgatewayPolicies"`
	KagentAgents         []AgentBrief    `json:"kagentAgents"`
	MCPServers           []MCPBrief      `json:"mcpServers"`
	RemoteMCPServers     []RemoteMCPBrief `json:"remoteMCPServers"`
}

type ResourceBrief struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
}

type BackendBrief struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	BackendType string `json:"backendType"`
	HasTLS      bool   `json:"hasTLS"`
	HasAuth     bool   `json:"hasAuth"`
	MCPTargets  int    `json:"mcpTargets"`
}

type PolicyBrief struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	HasJWT         bool   `json:"hasJWT"`
	JWTMode        string `json:"jwtMode,omitempty"`
	HasCORS        bool   `json:"hasCORS"`
	HasRBAC        bool   `json:"hasRBAC"`
	HasRateLimit   bool   `json:"hasRateLimit"`
	HasPromptGuard bool   `json:"hasPromptGuard"`
}

type AgentBrief struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	ToolCount int    `json:"toolCount"`
	Ready     bool   `json:"ready"`
}

type MCPBrief struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Transport string `json:"transport"`
}

type RemoteMCPBrief struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	URL       string `json:"url"`
	ToolCount int    `json:"toolCount"`
}

// ---------- Shared state for tools ----------

// evaluationContext holds the data that tools will access during agent execution.
// It is set before each agent invocation.
var evalCtx struct {
	state    *evaluator.ClusterState
	policy   evaluator.Policy
	findings []evaluator.Finding
	result   *evaluator.EvaluationResult
}

// ---------- Tool handler functions ----------

func handleGetClusterState(_ tool.Context, _ GetClusterStateArgs) (ClusterStateSummary, error) {
	state := evalCtx.state
	if state == nil {
		return ClusterStateSummary{}, fmt.Errorf("no cluster state available")
	}

	summary := ClusterStateSummary{
		TotalNamespaces:        len(state.Namespaces),
		Namespaces:             state.Namespaces,
		GatewaysCount:          len(state.Gateways),
		AgentgatewayBackends:   len(state.AgentgatewayBackends),
		AgentgatewayPolicies:   len(state.AgentgatewayPolicies),
		HTTPRoutes:             len(state.HTTPRoutes),
		KagentAgents:           len(state.KagentAgents),
		KagentMCPServers:       len(state.KagentMCPServers),
		KagentRemoteMCPServers: len(state.KagentRemoteMCPServers),
		ServicesCount:          len(state.Services),
	}

	// Check for agentgateway
	for _, gw := range state.Gateways {
		if gw.GatewayClassName == "agentgateway" {
			summary.HasAgentGateway = true
			if gw.Programmed {
				summary.GatewaysProgrammed++
			}
		}
	}

	// Check backend security
	for _, b := range state.AgentgatewayBackends {
		if b.BackendType == "mcp" {
			if b.HasTLS {
				summary.MCPBackendsWithTLS++
			} else {
				summary.MCPBackendsWithoutTLS++
			}
		}
	}

	// Check policy features
	for _, p := range state.AgentgatewayPolicies {
		if p.HasJWT {
			summary.PoliciesWithJWT++
		}
		if p.HasCORS {
			summary.PoliciesWithCORS++
		}
		if p.HasRBAC {
			summary.PoliciesWithRBAC++
		}
		if p.HasRateLimit {
			summary.PoliciesWithRateLimit++
		}
		if p.HasPromptGuard {
			summary.PoliciesWithPromptGuard++
		}
	}

	// Count unrouted MCP servers
	for _, mcp := range state.KagentMCPServers {
		routed := false
		for _, b := range state.AgentgatewayBackends {
			if b.BackendType == "mcp" {
				for _, t := range b.MCPTargets {
					expectedHost := fmt.Sprintf("%s.%s.svc.cluster.local", mcp.Name, mcp.Namespace)
					if t.Host == expectedHost || t.Host == mcp.Name {
						routed = true
						break
					}
				}
			}
			if routed {
				break
			}
		}
		if !routed {
			summary.UnroutedMCPServers++
		}
	}

	return summary, nil
}

func handleGetFindings(_ tool.Context, _ GetFindingsArgs) (FindingsSummary, error) {
	findings := evalCtx.findings
	summary := FindingsSummary{
		TotalFindings: len(findings),
		BySeverity:    make(map[string]int),
		ByCategory:    make(map[string]int),
	}

	for _, f := range findings {
		summary.BySeverity[f.Severity]++
		summary.ByCategory[f.Category]++

		brief := FindingBrief{
			ID:          f.ID,
			Category:    f.Category,
			Title:       f.Title,
			Severity:    f.Severity,
			ResourceRef: f.ResourceRef,
		}

		switch f.Severity {
		case evaluator.SeverityCritical:
			summary.CriticalItems = append(summary.CriticalItems, brief)
		case evaluator.SeverityHigh:
			summary.HighItems = append(summary.HighItems, brief)
		}
	}

	return summary, nil
}

func handleGetPolicy(_ tool.Context, _ GetPolicyArgs) (PolicySummary, error) {
	p := evalCtx.policy
	return PolicySummary{
		Name:                p.Name,
		RequireAgentGateway: p.RequireAgentGateway,
		RequireCORS:         p.RequireCORS,
		RequireJWTAuth:      p.RequireJWTAuth,
		RequireRBAC:         p.RequireRBAC,
		RequirePromptGuard:  p.RequirePromptGuard,
		RequireTLS:          p.RequireTLS,
		RequireRateLimit:    p.RequireRateLimit,
		MaxToolsWarning:     p.MaxToolsWarning,
		MaxToolsCritical:    p.MaxToolsCritical,
		TargetNamespaces:    p.TargetNamespaces,
		ExcludeNamespaces:   p.ExcludeNamespaces,
		Weights: map[string]int{
			"agentGateway":   p.Weights.AgentGatewayIntegration,
			"authentication": p.Weights.Authentication,
			"authorization":  p.Weights.Authorization,
			"cors":           p.Weights.CORSPolicy,
			"tls":            p.Weights.TLSEncryption,
			"promptGuard":    p.Weights.PromptGuard,
			"rateLimit":      p.Weights.RateLimit,
			"toolScope":      p.Weights.ToolScope,
		},
	}, nil
}

func handleGetAlgorithmicScore(_ tool.Context, _ GetAlgorithmicScoreArgs) (AlgorithmicScoreSummary, error) {
	res := evalCtx.result
	if res == nil {
		return AlgorithmicScoreSummary{}, fmt.Errorf("no evaluation result available")
	}

	grade := "F"
	switch {
	case res.Score >= 90:
		grade = "A"
	case res.Score >= 70:
		grade = "B"
	case res.Score >= 50:
		grade = "C"
	case res.Score >= 30:
		grade = "D"
	}

	return AlgorithmicScoreSummary{
		OverallScore: res.Score,
		Grade:        grade,
		CategoryScores: map[string]int{
			"agentGateway":   res.ScoreBreakdown.AgentGatewayScore,
			"authentication": res.ScoreBreakdown.AuthenticationScore,
			"authorization":  res.ScoreBreakdown.AuthorizationScore,
			"cors":           res.ScoreBreakdown.CORSScore,
			"tls":            res.ScoreBreakdown.TLSScore,
			"promptGuard":    res.ScoreBreakdown.PromptGuardScore,
			"rateLimit":      res.ScoreBreakdown.RateLimitScore,
			"toolScope":      res.ScoreBreakdown.ToolScopeScore,
		},
	}, nil
}

func handleGetResourceDetails(_ tool.Context, _ GetResourceDetailsArgs) (ResourceDetailsSummary, error) {
	state := evalCtx.state
	if state == nil {
		return ResourceDetailsSummary{}, fmt.Errorf("no cluster state available")
	}

	var summary ResourceDetailsSummary

	for _, gw := range state.Gateways {
		status := "Not Programmed"
		if gw.Programmed {
			status = "Programmed"
		}
		summary.Gateways = append(summary.Gateways, ResourceBrief{
			Name:      gw.Name,
			Namespace: gw.Namespace,
			Status:    status,
		})
	}

	for _, b := range state.AgentgatewayBackends {
		summary.AgentgatewayBackends = append(summary.AgentgatewayBackends, BackendBrief{
			Name:        b.Name,
			Namespace:   b.Namespace,
			BackendType: b.BackendType,
			HasTLS:      b.HasTLS,
			HasAuth:     b.HasAuth,
			MCPTargets:  len(b.MCPTargets),
		})
	}

	for _, p := range state.AgentgatewayPolicies {
		summary.AgentgatewayPolicies = append(summary.AgentgatewayPolicies, PolicyBrief{
			Name:           p.Name,
			Namespace:      p.Namespace,
			HasJWT:         p.HasJWT,
			JWTMode:        p.JWTMode,
			HasCORS:        p.HasCORS,
			HasRBAC:        p.HasRBAC,
			HasRateLimit:   p.HasRateLimit,
			HasPromptGuard: p.HasPromptGuard,
		})
	}

	for _, a := range state.KagentAgents {
		summary.KagentAgents = append(summary.KagentAgents, AgentBrief{
			Name:      a.Name,
			Namespace: a.Namespace,
			Type:      a.Type,
			ToolCount: len(a.Tools),
			Ready:     a.Ready,
		})
	}

	for _, m := range state.KagentMCPServers {
		summary.MCPServers = append(summary.MCPServers, MCPBrief{
			Name:      m.Name,
			Namespace: m.Namespace,
			Transport: m.Transport,
		})
	}

	for _, rm := range state.KagentRemoteMCPServers {
		summary.RemoteMCPServers = append(summary.RemoteMCPServers, RemoteMCPBrief{
			Name:      rm.Name,
			Namespace: rm.Namespace,
			URL:       rm.URL,
			ToolCount: rm.ToolCount,
		})
	}

	return summary, nil
}

// ---------- Agent construction ----------

const governanceInstruction = `You are an expert MCP (Model Context Protocol) Security Governance AI Agent.
Your job is to analyze the current state of a Kubernetes cluster's MCP infrastructure and provide a security governance score from 0 to 100.

You have access to the following tools:
- get_cluster_state: Returns a summary of all discovered Kubernetes resources (gateways, backends, policies, MCP servers, agents, etc.)
- get_findings: Returns the list of governance findings/violations discovered by the algorithmic evaluator
- get_policy: Returns the current governance policy configuration (what security requirements are enabled)
- get_algorithmic_score: Returns the current algorithmic (rule-based) governance score and per-category breakdown
- get_resource_details: Returns detailed information about each individual resource in the cluster

**Your Evaluation Process:**
1. First, call get_cluster_state to understand what infrastructure exists
2. Call get_findings to see what violations have been detected
3. Call get_policy to understand what security requirements are enabled
4. Call get_algorithmic_score to see the rule-based score for reference
5. Call get_resource_details if you need more details about specific resources

**Scoring Guidelines:**
- Score 90-100 (Grade A): Fully compliant. All required security controls are in place, properly configured, and covering all resources.
- Score 70-89 (Grade B): Mostly compliant. Core security controls exist but some gaps or weak configurations remain.
- Score 50-69 (Grade C): Partially compliant. Some security controls exist but significant gaps or misconfigurations are present.
- Score 30-49 (Grade D): Mostly non-compliant. Major security controls are missing or critically misconfigured.
- Score 0-29 (Grade F): Critical non-compliance. Essential security infrastructure is missing or fundamentally broken.

**Key Factors to Consider:**
- Is agentgateway deployed and routing ALL MCP traffic? (Most critical)
- Is JWT authentication configured in Strict mode?
- Is tool-level RBAC (CEL-based authorization) configured?
- Is TLS enabled on all backend connections?
- Are MCP servers exposed without going through agentgateway?
- Is CORS configured to prevent cross-origin attacks?
- Is rate limiting configured to prevent abuse?
- Is prompt guard configured for AI backends?
- Are there excessive tools on any MCP server (tool sprawl)?

**Important:** You may agree or disagree with the algorithmic score. The algorithmic score is purely penalty-based, while you should consider the holistic security posture, threat landscape, and practical risk. For example:
- If only minor Medium/Low findings exist but all critical infrastructure is in place, you might score higher than the algorithm.
- If the cluster has systemic architectural issues (e.g., no gateway at all), you might score even lower than the algorithm.

**Response Format:**
You MUST respond with ONLY a valid JSON object (no markdown, no code blocks, no extra text) in this exact format:
{
  "score": <integer 0-100>,
  "grade": "<A|B|C|D|F>",
  "reasoning": "<2-4 sentence explanation of your overall assessment>",
  "risks": [
    {
      "category": "<category name>",
      "severity": "<Critical|High|Medium|Low>",
      "description": "<brief description of the risk>",
      "impact": "<potential business/security impact>"
    }
  ],
  "suggestions": [
    "<actionable suggestion 1>",
    "<actionable suggestion 2>"
  ]
}`

// AIAgentConfig holds AI agent configuration from the governance policy
type AIAgentConfig struct {
	Provider       string // "gemini" or "ollama"
	Model          string // e.g. "gemini-2.5-flash", "llama3.1", "qwen2.5"
	OllamaEndpoint string // e.g. "http://localhost:11434"
}

// NewGovernanceAgent creates a new AI governance agent.
// For Gemini provider: requires GOOGLE_API_KEY environment variable.
// For Ollama provider: requires a running Ollama instance.
func NewGovernanceAgent(ctx context.Context, config AIAgentConfig) (*GovernanceAgent, error) {
	// Default provider and model
	provider := config.Provider
	if provider == "" {
		provider = "gemini"
	}
	modelName := config.Model

	var llmModel adkmodel.LLM
	var err error

	switch provider {
	case "ollama":
		if modelName == "" {
			modelName = "llama3.1"
		}
		endpoint := config.OllamaEndpoint
		if endpoint == "" {
			endpoint = os.Getenv("OLLAMA_HOST")
		}
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		llmModel = NewOllamaModel(modelName, endpoint)
		log.Printf("[ai-agent] Using Ollama model: %s at %s", modelName, endpoint)

	case "gemini":
		if modelName == "" {
			modelName = "gemini-2.5-flash"
		}
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GOOGLE_API_KEY environment variable is required for Gemini provider")
		}
		llmModel, err = gemini.NewModel(ctx, modelName, &genai.ClientConfig{
			APIKey: apiKey,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini model: %w", err)
		}
		log.Printf("[ai-agent] Using Gemini model: %s", modelName)

	default:
		return nil, fmt.Errorf("unsupported AI provider: %q (supported: gemini, ollama)", provider)
	}

	// Create function tools
	clusterStateTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_cluster_state",
			Description: "Returns a summary of all discovered Kubernetes resources related to MCP governance, including gateways, backends, policies, MCP servers, agents, services, and their security configuration status.",
		},
		handleGetClusterState,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_cluster_state tool: %w", err)
	}

	findingsTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_findings",
			Description: "Returns all governance findings (violations) discovered by the rule-based evaluator, grouped by severity and category. Includes critical and high severity item details.",
		},
		handleGetFindings,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_findings tool: %w", err)
	}

	policyTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_policy",
			Description: "Returns the current governance policy configuration including which security requirements are enabled (agentgateway, CORS, JWT, RBAC, TLS, etc.), scoring weights, and namespace filters.",
		},
		handleGetPolicy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_policy tool: %w", err)
	}

	algorithmicScoreTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_algorithmic_score",
			Description: "Returns the current rule-based (algorithmic) governance score and per-category score breakdown for reference. The AI agent can use this as a baseline but should form its own independent assessment.",
		},
		handleGetAlgorithmicScore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_algorithmic_score tool: %w", err)
	}

	resourceDetailsTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_resource_details",
			Description: "Returns detailed information about each individual resource in the cluster, including gateway programming status, backend TLS/auth configuration, policy security features, agent tool counts, and MCP server details.",
		},
		handleGetResourceDetails,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_resource_details tool: %w", err)
	}

	// Create the LLM agent
	a, err := llmagent.New(llmagent.Config{
		Name:        "mcp_governance_agent",
		Model:       llmModel,
		Description: "An AI agent that evaluates the security governance posture of MCP infrastructure in a Kubernetes cluster.",
		Instruction: governanceInstruction,
		Tools: []tool.Tool{
			clusterStateTool,
			findingsTool,
			policyTool,
			algorithmicScoreTool,
			resourceDetailsTool,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create governance agent: %w", err)
	}

	// Create session service and runner
	sessService := session.InMemoryService()

	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          a,
		SessionService: sessService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	return &GovernanceAgent{
		agent:       a,
		runner:      r,
		sessService: sessService,
	}, nil
}

// Evaluate runs the AI agent to produce a governance score
func (g *GovernanceAgent) Evaluate(ctx context.Context, state *evaluator.ClusterState, policy evaluator.Policy, result *evaluator.EvaluationResult) (*AIScoreResult, error) {
	// Set the evaluation context for tools to access
	evalCtx.state = state
	evalCtx.policy = policy
	evalCtx.findings = result.Findings
	evalCtx.result = result

	// Create a new session for this evaluation
	userID := "governance-controller"
	sessResp, err := g.sessService.Create(ctx, &session.CreateRequest{
		AppName: appName,
		UserID:  userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	sessionID := sessResp.Session.ID()

	// Send the evaluation prompt
	userMsg := &genai.Content{
		Parts: []*genai.Part{
			genai.NewPartFromText("Analyze the current MCP security governance posture of this Kubernetes cluster. Use all available tools to gather information about the cluster state, findings, policy, and current scores. Then provide your independent AI-driven governance assessment as a JSON response."),
		},
		Role: "user",
	}

	// Run the agent and collect the final response
	var finalText string
	for event, err := range g.runner.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{}) {
		if err != nil {
			log.Printf("[ai-agent] Error during agent execution: %v", err)
			continue
		}
		if event.IsFinalResponse() {
			// Extract text from the final response
			if event.Content != nil {
				for _, part := range event.Content.Parts {
					if part.Text != "" {
						finalText += part.Text
					}
				}
			}
		}
	}

	if finalText == "" {
		return nil, fmt.Errorf("AI agent produced no response")
	}

	// Parse the AI agent's JSON response
	aiResult, err := parseAIResponse(finalText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI agent response: %w (raw: %s)", err, truncateString(finalText, 500))
	}

	// Clean up the session
	_ = g.sessService.Delete(ctx, &session.DeleteRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})

	aiResult.Timestamp = time.Now()
	return aiResult, nil
}

// parseAIResponse extracts and parses the JSON from the AI agent's response
func parseAIResponse(text string) (*AIScoreResult, error) {
	// Try to find JSON in the response (the agent might wrap it in markdown code blocks)
	cleaned := text

	// Strip markdown code block if present
	if idx := strings.Index(cleaned, "```json"); idx != -1 {
		cleaned = cleaned[idx+7:]
		if endIdx := strings.Index(cleaned, "```"); endIdx != -1 {
			cleaned = cleaned[:endIdx]
		}
	} else if idx := strings.Index(cleaned, "```"); idx != -1 {
		cleaned = cleaned[idx+3:]
		if endIdx := strings.Index(cleaned, "```"); endIdx != -1 {
			cleaned = cleaned[:endIdx]
		}
	}

	// Try to find JSON object boundaries
	cleaned = strings.TrimSpace(cleaned)
	if !strings.HasPrefix(cleaned, "{") {
		// Try to find JSON object in the text
		start := strings.Index(cleaned, "{")
		if start == -1 {
			return nil, fmt.Errorf("no JSON object found in response")
		}
		cleaned = cleaned[start:]
	}

	// Find the matching closing brace
	braceCount := 0
	endIdx := -1
	for i, ch := range cleaned {
		if ch == '{' {
			braceCount++
		} else if ch == '}' {
			braceCount--
			if braceCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}
	if endIdx > 0 {
		cleaned = cleaned[:endIdx]
	}

	var result AIScoreResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Validate score range
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 100 {
		result.Score = 100
	}

	// Ensure grade is set correctly based on score
	switch {
	case result.Score >= 90:
		result.Grade = "A"
	case result.Score >= 70:
		result.Grade = "B"
	case result.Score >= 50:
		result.Grade = "C"
	case result.Score >= 30:
		result.Grade = "D"
	default:
		result.Grade = "F"
	}

	return &result, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// IsAvailable checks if the AI agent can be initialized.
// For Gemini: requires GOOGLE_API_KEY env var.
// For Ollama: always available (assumes Ollama is running).
func IsAvailable(provider string) bool {
	switch provider {
	case "ollama":
		return true // Ollama availability is checked at connection time
	case "gemini", "":
		return os.Getenv("GOOGLE_API_KEY") != ""
	default:
		return false
	}
}
