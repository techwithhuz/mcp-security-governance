package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/techwithhuz/mcp-security-governance/controller/pkg/discovery"
	"github.com/techwithhuz/mcp-security-governance/controller/pkg/evaluator"
)

var (
	lastResult   *evaluator.EvaluationResult
	lastCluster  *evaluator.ClusterState
	currentState *evaluator.ClusterState
	policy       evaluator.Policy
	discoverer   *discovery.K8sDiscoverer
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	// Try to create a real Kubernetes discoverer
	var err error
	discoverer, err = discovery.NewK8sDiscoverer()
	if err != nil {
		log.Printf("[governance-api] WARNING: Could not create K8s discoverer: %v", err)
		log.Printf("[governance-api] Falling back to simulated cluster state")
		discoverer = nil
	} else {
		log.Printf("[governance-api] Connected to Kubernetes cluster — using real discovery")
	}

	// Initial discovery and evaluation
	currentState = doDiscovery()
	policy = loadPolicy()
	lastCluster = currentState
	lastResult = evaluator.Evaluate(currentState, policy)
	log.Printf("[governance] Initial evaluation. Score: %d, Findings: %d (Policy: AgentGW=%v, CORS=%v, JWT=%v, RBAC=%v, TLS=%v, PromptGuard=%v, RateLimit=%v)", 
		lastResult.Score, len(lastResult.Findings),
		policy.RequireAgentGateway, policy.RequireCORS, policy.RequireJWTAuth, 
		policy.RequireRBAC, policy.RequireTLS, policy.RequirePromptGuard, policy.RequireRateLimit)

	// Periodic re-evaluation
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			currentState = doDiscovery()
			policy = loadPolicy() // Reload policy in case it changed
			lastCluster = currentState
			lastResult = evaluator.Evaluate(currentState, policy)
			log.Printf("[governance] Re-evaluated cluster. Score: %d, Findings: %d", lastResult.Score, len(lastResult.Findings))
		}
	}()

	// API routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/governance/score", handleScore)
	mux.HandleFunc("/api/governance/findings", handleFindings)
	mux.HandleFunc("/api/governance/resources", handleResources)
	mux.HandleFunc("/api/governance/namespaces", handleNamespaces)
	mux.HandleFunc("/api/governance/breakdown", handleBreakdown)
	mux.HandleFunc("/api/governance/evaluation", handleFullEvaluation)
	mux.HandleFunc("/api/governance/trends", handleTrends)
	mux.HandleFunc("/api/governance/resources/detail", handleResourceDetail)

	// CORS middleware
	handler := corsMiddleware(mux)

	log.Printf("[governance-api] Starting on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{"status": "healthy", "version": "0.1.0"})
}

func getGrade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 70:
		return "B"
	case score >= 50:
		return "C"
	case score >= 30:
		return "D"
	default:
		return "F"
	}
}

func getPhase(score int) string {
	switch {
	case score >= 90:
		return "Compliant"
	case score >= 70:
		return "PartiallyCompliant"
	case score >= 50:
		return "NonCompliant"
	default:
		return "Critical"
	}
}

func handleScore(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		jsonResponse(w, map[string]interface{}{"score": 0, "grade": "F", "phase": "Unknown"})
		return
	}
	type categoryDetail struct {
		Category string  `json:"category"`
		Score    int     `json:"score"`
		Weight   int     `json:"weight"`
		Weighted float64 `json:"weighted"`
		Status   string  `json:"status"`
	}
	// Build explanation of how each category contributes
	pw := policy.Weights
	totalWeight := 0
	var cats []categoryDetail
	
	bd := lastResult.ScoreBreakdown
	
	if policy.RequireAgentGateway {
		totalWeight += pw.AgentGatewayIntegration
		cats = append(cats, categoryDetail{"AgentGateway Compliance", bd.AgentGatewayScore, pw.AgentGatewayIntegration, float64(bd.AgentGatewayScore*pw.AgentGatewayIntegration) / float64(pw.AgentGatewayIntegration), statusLabel(bd.AgentGatewayScore)})
	}
	if policy.RequireJWTAuth {
		totalWeight += pw.Authentication
		cats = append(cats, categoryDetail{"Authentication", bd.AuthenticationScore, pw.Authentication, float64(bd.AuthenticationScore*pw.Authentication) / float64(pw.Authentication), statusLabel(bd.AuthenticationScore)})
	}
	if policy.RequireRBAC {
		totalWeight += pw.Authorization
		cats = append(cats, categoryDetail{"Authorization", bd.AuthorizationScore, pw.Authorization, float64(bd.AuthorizationScore*pw.Authorization) / float64(pw.Authorization), statusLabel(bd.AuthorizationScore)})
	}
	if policy.RequireCORS {
		totalWeight += pw.CORSPolicy
		cats = append(cats, categoryDetail{"CORS", bd.CORSScore, pw.CORSPolicy, float64(bd.CORSScore*pw.CORSPolicy) / float64(pw.CORSPolicy), statusLabel(bd.CORSScore)})
	}
	if policy.RequireTLS {
		totalWeight += pw.TLSEncryption
		cats = append(cats, categoryDetail{"TLS", bd.TLSScore, pw.TLSEncryption, float64(bd.TLSScore*pw.TLSEncryption) / float64(pw.TLSEncryption), statusLabel(bd.TLSScore)})
	}
	if policy.RequirePromptGuard {
		totalWeight += pw.PromptGuard
		cats = append(cats, categoryDetail{"Prompt Guard", bd.PromptGuardScore, pw.PromptGuard, float64(bd.PromptGuardScore*pw.PromptGuard) / float64(pw.PromptGuard), statusLabel(bd.PromptGuardScore)})
	}
	if policy.RequireRateLimit {
		totalWeight += pw.RateLimit
		cats = append(cats, categoryDetail{"Rate Limit", bd.RateLimitScore, pw.RateLimit, float64(bd.RateLimitScore*pw.RateLimit) / float64(pw.RateLimit), statusLabel(bd.RateLimitScore)})
	}
	if policy.MaxToolsWarning > 0 || policy.MaxToolsCritical > 0 {
		totalWeight += pw.ToolScope
		cats = append(cats, categoryDetail{"Tool Scope", bd.ToolScopeScore, pw.ToolScope, float64(bd.ToolScopeScore*pw.ToolScope) / float64(pw.ToolScope), statusLabel(bd.ToolScopeScore)})
	}
	
	if totalWeight == 0 {
		totalWeight = 100
	}
	
	// Recalculate weighted scores with correct totalWeight
	for i := range cats {
		cats[i].Weighted = float64(cats[i].Score*cats[i].Weight) / float64(totalWeight)
	}
	jsonResponse(w, map[string]interface{}{
		"score":      lastResult.Score,
		"grade":      getGrade(lastResult.Score),
		"phase":      getPhase(lastResult.Score),
		"timestamp":  lastResult.Timestamp,
		"categories": cats,
		"explanation": fmt.Sprintf(
			"Score is a weighted average of %d governance categories. Each category is scored 0-100 based on findings (Critical: -%dpts, High: -%dpts, Medium: -%dpts, Low: -%dpts). The final score %d/100 = Grade %s.",
			len(cats), policy.SeverityPenalties.Critical, policy.SeverityPenalties.High, policy.SeverityPenalties.Medium, policy.SeverityPenalties.Low, lastResult.Score, getGrade(lastResult.Score)),
	})
}

func statusLabel(score int) string {
	switch {
	case score >= 90:
		return "passing"
	case score >= 70:
		return "warning"
	case score >= 50:
		return "failing"
	default:
		return "critical"
	}
}

func handleFindings(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		jsonResponse(w, map[string]interface{}{
			"findings":   []evaluator.Finding{},
			"total":      0,
			"bySeverity": map[string]int{},
		})
		return
	}
	bySeverity := map[string]int{}
	for _, f := range lastResult.Findings {
		bySeverity[f.Severity]++
	}
	jsonResponse(w, map[string]interface{}{
		"findings":   lastResult.Findings,
		"total":      len(lastResult.Findings),
		"bySeverity": bySeverity,
	})
}

func handleResources(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		jsonResponse(w, evaluator.ResourceSummary{})
		return
	}
	jsonResponse(w, lastResult.ResourceSummary)
}

func handleNamespaces(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		jsonResponse(w, map[string]interface{}{"namespaces": []evaluator.NamespaceScore{}})
		return
	}
	jsonResponse(w, map[string]interface{}{"namespaces": lastResult.NamespaceScores})
}

func handleBreakdown(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		jsonResponse(w, map[string]interface{}{})
		return
	}
	bd := lastResult.ScoreBreakdown
	result := map[string]int{}
	if policy.RequireAgentGateway {
		result["agentGatewayScore"] = bd.AgentGatewayScore
	}
	if policy.RequireJWTAuth {
		result["authenticationScore"] = bd.AuthenticationScore
	}
	if policy.RequireRBAC {
		result["authorizationScore"] = bd.AuthorizationScore
	}
	if policy.RequireCORS {
		result["corsScore"] = bd.CORSScore
	}
	if policy.RequireTLS {
		result["tlsScore"] = bd.TLSScore
	}
	if policy.RequirePromptGuard {
		result["promptGuardScore"] = bd.PromptGuardScore
	}
	if policy.RequireRateLimit {
		result["rateLimitScore"] = bd.RateLimitScore
	}
	if policy.MaxToolsWarning > 0 || policy.MaxToolsCritical > 0 {
		result["toolScopeScore"] = bd.ToolScopeScore
	}
	jsonResponse(w, result)
}

func handleFullEvaluation(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		http.Error(w, "No evaluation available", http.StatusServiceUnavailable)
		return
	}
	jsonResponse(w, lastResult)
}

// ResourceDetail groups findings per individual resource
type ResourceDetail struct {
	ResourceRef string             `json:"resourceRef"`
	Kind        string             `json:"kind"`
	Name        string             `json:"name"`
	Namespace   string             `json:"namespace"`
	Status      string             `json:"status"`
	Score       int                `json:"score"`
	Findings    []evaluator.Finding `json:"findings"`
	Critical    int                `json:"critical"`
	High        int                `json:"high"`
	Medium      int                `json:"medium"`
	Low         int                `json:"low"`
}

func handleResourceDetail(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil || lastCluster == nil {
		jsonResponse(w, map[string]interface{}{"resources": []ResourceDetail{}})
		return
	}

	// Build a map of resourceRef -> findings
	findingsMap := map[string][]evaluator.Finding{}
	clusterFindings := []evaluator.Finding{} // findings not tied to a specific resource
	for _, f := range lastResult.Findings {
		if f.ResourceRef != "" {
			findingsMap[f.ResourceRef] = append(findingsMap[f.ResourceRef], f)
		} else {
			clusterFindings = append(clusterFindings, f)
		}
	}

	var resources []ResourceDetail

	// AgentGateway Backends
	for _, b := range lastCluster.AgentgatewayBackends {
		ref := fmt.Sprintf("AgentgatewayBackend/%s/%s", b.Namespace, b.Name)
		rd := buildResourceDetail(ref, "AgentgatewayBackend", b.Name, b.Namespace, findingsMap[ref])
		resources = append(resources, rd)
	}

	// AgentGateway Policies
	for _, p := range lastCluster.AgentgatewayPolicies {
		ref := fmt.Sprintf("AgentgatewayPolicy/%s/%s", p.Namespace, p.Name)
		rd := buildResourceDetail(ref, "AgentgatewayPolicy", p.Name, p.Namespace, findingsMap[ref])
		resources = append(resources, rd)
	}

	// Gateways
	for _, g := range lastCluster.Gateways {
		ref := fmt.Sprintf("Gateway/%s/%s", g.Namespace, g.Name)
		rd := buildResourceDetail(ref, "Gateway", g.Name, g.Namespace, findingsMap[ref])
		resources = append(resources, rd)
	}

	// HTTPRoutes
	for _, h := range lastCluster.HTTPRoutes {
		ref := fmt.Sprintf("HTTPRoute/%s/%s", h.Namespace, h.Name)
		rd := buildResourceDetail(ref, "HTTPRoute", h.Name, h.Namespace, findingsMap[ref])
		resources = append(resources, rd)
	}

	// Kagent Agents
	for _, a := range lastCluster.KagentAgents {
		ref := fmt.Sprintf("Agent/%s/%s", a.Namespace, a.Name)
		rd := buildResourceDetail(ref, "Agent", a.Name, a.Namespace, findingsMap[ref])
		resources = append(resources, rd)
	}

	// Kagent MCPServers
	for _, m := range lastCluster.KagentMCPServers {
		ref := fmt.Sprintf("MCPServer/%s/%s", m.Namespace, m.Name)
		rd := buildResourceDetail(ref, "MCPServer", m.Name, m.Namespace, findingsMap[ref])
		resources = append(resources, rd)
	}

	// Kagent RemoteMCPServers
	for _, rm := range lastCluster.KagentRemoteMCPServers {
		ref := fmt.Sprintf("RemoteMCPServer/%s/%s", rm.Namespace, rm.Name)
		rd := buildResourceDetail(ref, "RemoteMCPServer", rm.Name, rm.Namespace, findingsMap[ref])
		resources = append(resources, rd)
	}

	// Add cluster-wide findings as a virtual resource
	if len(clusterFindings) > 0 {
		rd := buildResourceDetail("cluster-wide", "Cluster", "cluster-wide-policies", "", clusterFindings)
		resources = append(resources, rd)
	}

	jsonResponse(w, map[string]interface{}{
		"resources": resources,
		"total":     len(resources),
	})
}

func buildResourceDetail(ref, kind, name, namespace string, findings []evaluator.Finding) ResourceDetail {
	rd := ResourceDetail{
		ResourceRef: ref,
		Kind:        kind,
		Name:        name,
		Namespace:   namespace,
		Findings:    findings,
		Score:       100,
	}
	if rd.Findings == nil {
		rd.Findings = []evaluator.Finding{}
	}

	// Count severity occurrences
	for _, f := range findings {
		switch f.Severity {
		case "Critical":
			rd.Critical++
		case "High":
			rd.High++
		case "Medium":
			rd.Medium++
		case "Low":
			rd.Low++
		}
	}

	// Calculate score: if any Critical findings exist, score = 0
	// Otherwise deduct per severity using policy-configured penalties
	if rd.Critical > 0 {
		rd.Score = 0
	} else {
		for _, f := range findings {
			switch f.Severity {
			case "High":
				rd.Score -= policy.SeverityPenalties.High
			case "Medium":
				rd.Score -= policy.SeverityPenalties.Medium
			case "Low":
				rd.Score -= policy.SeverityPenalties.Low
			}
		}
		if rd.Score < 0 {
			rd.Score = 0
		}
	}

	switch {
	case len(findings) == 0:
		rd.Status = "compliant"
	case rd.Critical > 0:
		rd.Status = "critical"
	case rd.High > 0:
		rd.Status = "failing"
	case rd.Medium > 0:
		rd.Status = "warning"
	default:
		rd.Status = "info"
	}
	return rd
}

// Trend data - in production this would use persistent storage
var trendHistory []TrendPoint

type TrendPoint struct {
	Timestamp string `json:"timestamp"`
	Score     int    `json:"score"`
	Findings  int    `json:"findings"`
	Critical  int    `json:"critical"`
	High      int    `json:"high"`
	Medium    int    `json:"medium"`
	Low       int    `json:"low"`
}

func handleTrends(w http.ResponseWriter, r *http.Request) {
	if lastResult != nil {
		critical, high, medium, low := 0, 0, 0, 0
		for _, f := range lastResult.Findings {
			switch f.Severity {
			case "Critical":
				critical++
			case "High":
				high++
			case "Medium":
				medium++
			case "Low":
				low++
			}
		}
		trendHistory = append(trendHistory, TrendPoint{
			Timestamp: lastResult.Timestamp.Format(time.RFC3339),
			Score:     lastResult.Score,
			Findings:  len(lastResult.Findings),
			Critical:  critical,
			High:      high,
			Medium:    medium,
			Low:       low,
		})

		// Keep last 100 trend points
		if len(trendHistory) > 100 {
			trendHistory = trendHistory[len(trendHistory)-100:]
		}
	}
	jsonResponse(w, map[string]interface{}{"trends": trendHistory})
}

// doDiscovery uses real K8s discovery if available, otherwise falls back to simulated data
func doDiscovery() *evaluator.ClusterState {
	if discoverer != nil {
		return discoverer.DiscoverClusterState(context.Background())
	}
	return discoverClusterState()
}

// loadPolicy loads the MCPGovernancePolicy from the cluster or returns default
func loadPolicy() evaluator.Policy {
	if discoverer != nil {
		if policy := discoverer.DiscoverGovernancePolicy(context.Background()); policy != nil {
			return *policy
		}
	}
	log.Printf("[governance] Using default policy")
	return evaluator.DefaultPolicy()
}

// discoverClusterState is the fallback simulated discovery
// Used when the controller is running outside a Kubernetes cluster
func discoverClusterState() *evaluator.ClusterState {
	state := &evaluator.ClusterState{
		Namespaces: []string{"default", "agentgateway-system", "kagent", "mcp-apps"},

		// Simulate: agentgateway Gateway exists but may lack policies
		Gateways: []evaluator.GatewayResource{
			{
				Name:             "agentgateway-proxy",
				Namespace:        "agentgateway-system",
				GatewayClassName: "agentgateway",
				Programmed:       true,
				Listeners: []evaluator.ListenerInfo{
					{Name: "http", Port: 80, Protocol: "HTTP"},
				},
			},
		},

		// Some MCP backends configured
		AgentgatewayBackends: []evaluator.AgentgatewayBackendResource{
			{
				Name:        "github-mcp-backend",
				Namespace:   "agentgateway-system",
				BackendType: "mcp",
				HasTLS:      false,
				MCPTargets: []evaluator.MCPTargetInfo{
					{
						Name:     "github-mcp",
						Host:     "mcp-github-server.mcp-apps.svc.cluster.local",
						Port:     80,
						Protocol: "StreamableHTTP",
						HasAuth:  false,
						HasRBAC:  false,
					},
				},
			},
			{
				Name:        "fetch-mcp-backend",
				Namespace:   "agentgateway-system",
				BackendType: "mcp",
				HasTLS:      false,
				MCPTargets: []evaluator.MCPTargetInfo{
					{
						Name:     "fetch-mcp",
						Host:     "mcp-website-fetcher.default.svc.cluster.local",
						Port:     80,
						Protocol: "SSE",
						HasAuth:  false,
						HasRBAC:  false,
					},
				},
			},
			{
				Name:        "openai-backend",
				Namespace:   "agentgateway-system",
				BackendType: "ai",
				HasTLS:      true,
			},
		},

		// Limited policies - missing some security controls
		AgentgatewayPolicies: []evaluator.AgentgatewayPolicyResource{
			{
				Name:      "basic-auth",
				Namespace: "agentgateway-system",
				HasJWT:    true,
				JWTMode:   "Optional", // Intentionally weak for demo
				HasCORS:   false,
				HasCSRF:   false,
				HasRBAC:   false,
				HasRateLimit: false,
				HasPromptGuard: false,
				TargetRefs: []evaluator.PolicyTargetRef{
					{Group: "gateway.networking.k8s.io", Kind: "Gateway", Name: "agentgateway-proxy"},
				},
			},
		},

		HTTPRoutes: []evaluator.HTTPRouteResource{
			{
				Name:          "mcp-github",
				Namespace:     "agentgateway-system",
				ParentGateway: "agentgateway-proxy",
				BackendRefs:   []string{"github-mcp-backend"},
				HasCORSFilter: false,
			},
			{
				Name:          "mcp-fetcher",
				Namespace:     "agentgateway-system",
				ParentGateway: "agentgateway-proxy",
				BackendRefs:   []string{"fetch-mcp-backend"},
				HasCORSFilter: false,
			},
		},

		// kagent resources
		KagentAgents: []evaluator.KagentAgentResource{
			{
				Name:      "k8s-agent",
				Namespace: "kagent",
				Type:      "Declarative",
				Ready:     true,
				Tools: []evaluator.KagentToolRef{
					{Type: "McpServer", Kind: "RemoteMCPServer", Name: "kagent-tool-server", ToolNames: []string{"k8s_get_resources"}},
				},
			},
			{
				Name:      "fetch-agent",
				Namespace: "kagent",
				Type:      "Declarative",
				Ready:     true,
				Tools: []evaluator.KagentToolRef{
					{Type: "McpServer", Kind: "MCPServer", Name: "mcp-website-fetcher", ToolNames: []string{"fetch"}},
				},
			},
		},

		KagentMCPServers: []evaluator.KagentMCPServerResource{
			{
				Name:      "mcp-website-fetcher",
				Namespace: "kagent",
				Transport: "stdio",
				Port:      3000,
			},
			{
				Name:      "unrouted-mcp-server",
				Namespace: "mcp-apps",
				Transport: "sse",
				Port:      8080,
			},
		},

		KagentRemoteMCPServers: []evaluator.KagentRemoteMCPServerResource{
			{
				Name:      "kagent-tool-server",
				Namespace: "kagent",
				URL:       "http://kagent-tool-server.kagent.svc:3000",
			},
		},

		Services: []evaluator.ServiceResource{
			{
				Name:        "mcp-website-fetcher",
				Namespace:   "default",
				AppProtocol: "kgateway.dev/mcp",
				Ports:       []int{80},
				IsMCP:       true,
			},
			{
				Name:        "standalone-mcp-svc",
				Namespace:   "mcp-apps",
				AppProtocol: "kgateway.dev/mcp",
				Ports:       []int{8080},
				IsMCP:       true,
			},
		},
	}

	return state
}

func init() {
	// Suppress unused import warning
	_ = fmt.Sprintf
}
