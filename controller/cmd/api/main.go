package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/techwithhuz/mcp-security-governance/controller/pkg/aiagent"
	"github.com/techwithhuz/mcp-security-governance/controller/pkg/discovery"
	"github.com/techwithhuz/mcp-security-governance/controller/pkg/evaluator"
)

var (
	// Version is set at build time via ldflags
	Version = "dev"

	stateMu      sync.RWMutex
	lastResult   *evaluator.EvaluationResult
	lastCluster  *evaluator.ClusterState
	currentState *evaluator.ClusterState
	policy       evaluator.Policy
	discoverer   *discovery.K8sDiscoverer

	// AI agent state
	aiAgent      *aiagent.GovernanceAgent
	lastAIResult *aiagent.AIScoreResult
	aiAgentErr   error // Tracks initialization or last runtime error

	// AI evaluation rate limiting
	aiLastRun     time.Time
	aiBackoff     time.Duration
	aiMinInterval = 5 * time.Minute // default; overridden by policy.AIScanInterval
	aiScanPaused  bool              // runtime pause flag (toggled via API)
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
	lastResult = evaluator.Evaluate(currentState.FilterByNamespaces(policy.TargetNamespaces, policy.ExcludeNamespaces), policy)
	recordTrendPoint(lastResult)
	updatePolicyStatus(policy.Name, lastResult)
	updateEvaluationStatus(policy.Name, lastResult)
	log.Printf("[governance] Initial evaluation. Score: %d, Findings: %d (Policy: AgentGW=%v, CORS=%v, JWT=%v, RBAC=%v, TLS=%v, PromptGuard=%v, RateLimit=%v, AIAgent=%v, TargetNS=%v, ExcludeNS=%v)", 
		lastResult.Score, len(lastResult.Findings),
		policy.RequireAgentGateway, policy.RequireCORS, policy.RequireJWTAuth, 
		policy.RequireRBAC, policy.RequireTLS, policy.RequirePromptGuard, policy.RequireRateLimit,
		policy.EnableAIAgent,
		policy.TargetNamespaces, policy.ExcludeNamespaces)

	// Initialize AI agent if enabled
	if policy.EnableAIAgent {
		initAIAgent(context.Background())
	}

	// Periodic re-evaluation
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			cs := doDiscovery()
			p := loadPolicy()
			res := evaluator.Evaluate(cs.FilterByNamespaces(p.TargetNamespaces, p.ExcludeNamespaces), p)

			stateMu.Lock()
			currentState = cs
			policy = p
			lastCluster = cs
			lastResult = res
			stateMu.Unlock()

			recordTrendPoint(res)
			updatePolicyStatus(p.Name, res)
			updateEvaluationStatus(p.Name, res)
			log.Printf("[governance] Re-evaluated cluster. Score: %d, Findings: %d", res.Score, len(res.Findings))

			// Run AI agent evaluation if enabled
			if p.EnableAIAgent {
				runAIEvaluation(context.Background(), cs, p, res)
			}
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
	mux.HandleFunc("/api/governance/ai-score", handleAIScore)
	mux.HandleFunc("/api/governance/ai-score/refresh", handleAIRefresh)
	mux.HandleFunc("/api/governance/ai-score/toggle", handleAIToggle)

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

// stateSnapshot holds an immutable copy of shared state for safe use in handlers.
type stateSnapshot struct {
	result  *evaluator.EvaluationResult
	cluster *evaluator.ClusterState
	policy  evaluator.Policy
}

// getSnapshot returns a consistent read of the shared state.
func getSnapshot() stateSnapshot {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return stateSnapshot{
		result:  lastResult,
		cluster: lastCluster,
		policy:  policy,
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{"status": "healthy", "version": Version})
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
	snap := getSnapshot()
	if snap.result == nil {
		jsonResponse(w, map[string]interface{}{"score": 0, "grade": "F", "phase": "Unknown"})
		return
	}
	type categoryDetail struct {
		Category    string  `json:"category"`
		Score       int     `json:"score"`
		Weight      int     `json:"weight"`
		Weighted    float64 `json:"weighted"`
		Status      string  `json:"status"`
		InfraAbsent bool    `json:"infraAbsent"`
	}
	// Build explanation of how each category contributes
	pw := snap.policy.Weights
	totalWeight := 0
	var cats []categoryDetail
	
	bd := snap.result.ScoreBreakdown
	
	if snap.policy.RequireAgentGateway {
		totalWeight += pw.AgentGatewayIntegration
		cats = append(cats, categoryDetail{"AgentGateway Compliance", bd.AgentGatewayScore, pw.AgentGatewayIntegration, 0, statusLabel(bd.AgentGatewayScore), bd.InfraAbsent["AgentGateway Compliance"]})
	}
	if snap.policy.RequireJWTAuth {
		totalWeight += pw.Authentication
		cats = append(cats, categoryDetail{"Authentication", bd.AuthenticationScore, pw.Authentication, 0, statusLabel(bd.AuthenticationScore), bd.InfraAbsent["Authentication"]})
	}
	if snap.policy.RequireRBAC {
		totalWeight += pw.Authorization
		cats = append(cats, categoryDetail{"Authorization", bd.AuthorizationScore, pw.Authorization, 0, statusLabel(bd.AuthorizationScore), bd.InfraAbsent["Authorization"]})
	}
	if snap.policy.RequireCORS {
		totalWeight += pw.CORSPolicy
		cats = append(cats, categoryDetail{"CORS", bd.CORSScore, pw.CORSPolicy, 0, statusLabel(bd.CORSScore), bd.InfraAbsent["CORS"]})
	}
	if snap.policy.RequireTLS {
		totalWeight += pw.TLSEncryption
		cats = append(cats, categoryDetail{"TLS", bd.TLSScore, pw.TLSEncryption, 0, statusLabel(bd.TLSScore), bd.InfraAbsent["TLS"]})
	}
	if snap.policy.RequirePromptGuard {
		totalWeight += pw.PromptGuard
		cats = append(cats, categoryDetail{"Prompt Guard", bd.PromptGuardScore, pw.PromptGuard, 0, statusLabel(bd.PromptGuardScore), bd.InfraAbsent["Prompt Guard"]})
	}
	if snap.policy.RequireRateLimit {
		totalWeight += pw.RateLimit
		cats = append(cats, categoryDetail{"Rate Limit", bd.RateLimitScore, pw.RateLimit, 0, statusLabel(bd.RateLimitScore), bd.InfraAbsent["Rate Limit"]})
	}
	if snap.policy.MaxToolsWarning > 0 || snap.policy.MaxToolsCritical > 0 {
		totalWeight += pw.ToolScope
		cats = append(cats, categoryDetail{"Tool Scope", bd.ToolScopeScore, pw.ToolScope, 0, statusLabel(bd.ToolScopeScore), bd.InfraAbsent["Tool Scope"]})
	}
	
	if totalWeight == 0 {
		totalWeight = 100
	}
	
	// Recalculate weighted scores with correct totalWeight
	for i := range cats {
		cats[i].Weighted = float64(cats[i].Score*cats[i].Weight) / float64(totalWeight)
	}

	response := map[string]interface{}{
		"score":      snap.result.Score,
		"grade":      getGrade(snap.result.Score),
		"phase":      getPhase(snap.result.Score),
		"timestamp":  snap.result.Timestamp,
		"categories": cats,
		"severityPenalties": map[string]int{
			"Critical": snap.policy.SeverityPenalties.Critical,
			"High":     snap.policy.SeverityPenalties.High,
			"Medium":   snap.policy.SeverityPenalties.Medium,
			"Low":      snap.policy.SeverityPenalties.Low,
		},
		"explanation": fmt.Sprintf(
			"Score is a weighted average of %d governance categories. Each category is scored 0-100 based on findings (Critical: -%dpts, High: -%dpts, Medium: -%dpts, Low: -%dpts). The final score %d/100 = Grade %s.",
			len(cats), snap.policy.SeverityPenalties.Critical, snap.policy.SeverityPenalties.High, snap.policy.SeverityPenalties.Medium, snap.policy.SeverityPenalties.Low, snap.result.Score, getGrade(snap.result.Score)),
		"aiAgentEnabled": snap.policy.EnableAIAgent,
	}

	// Include AI score if available
	stateMu.RLock()
	if lastAIResult != nil {
		response["aiScore"] = lastAIResult
	}
	stateMu.RUnlock()

	jsonResponse(w, response)
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
	snap := getSnapshot()
	if snap.result == nil {
		jsonResponse(w, map[string]interface{}{
			"findings":   []evaluator.Finding{},
			"total":      0,
			"bySeverity": map[string]int{},
		})
		return
	}
	bySeverity := map[string]int{}
	for _, f := range snap.result.Findings {
		bySeverity[f.Severity]++
	}
	jsonResponse(w, map[string]interface{}{
		"findings":   snap.result.Findings,
		"total":      len(snap.result.Findings),
		"bySeverity": bySeverity,
	})
}

func handleResources(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()
	if snap.result == nil {
		jsonResponse(w, evaluator.ResourceSummary{})
		return
	}
	jsonResponse(w, snap.result.ResourceSummary)
}

func handleNamespaces(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()
	if snap.result == nil {
		jsonResponse(w, map[string]interface{}{"namespaces": []evaluator.NamespaceScore{}})
		return
	}
	jsonResponse(w, map[string]interface{}{"namespaces": snap.result.NamespaceScores})
}

func handleBreakdown(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()
	if snap.result == nil {
		jsonResponse(w, map[string]interface{}{})
		return
	}
	bd := snap.result.ScoreBreakdown
	result := map[string]int{}
	if snap.policy.RequireAgentGateway {
		result["agentGatewayScore"] = bd.AgentGatewayScore
	}
	if snap.policy.RequireJWTAuth {
		result["authenticationScore"] = bd.AuthenticationScore
	}
	if snap.policy.RequireRBAC {
		result["authorizationScore"] = bd.AuthorizationScore
	}
	if snap.policy.RequireCORS {
		result["corsScore"] = bd.CORSScore
	}
	if snap.policy.RequireTLS {
		result["tlsScore"] = bd.TLSScore
	}
	if snap.policy.RequirePromptGuard {
		result["promptGuardScore"] = bd.PromptGuardScore
	}
	if snap.policy.RequireRateLimit {
		result["rateLimitScore"] = bd.RateLimitScore
	}
	if snap.policy.MaxToolsWarning > 0 || snap.policy.MaxToolsCritical > 0 {
		result["toolScopeScore"] = bd.ToolScopeScore
	}
	jsonResponse(w, result)
}

func handleFullEvaluation(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()
	if snap.result == nil {
		http.Error(w, "No evaluation available", http.StatusServiceUnavailable)
		return
	}
	jsonResponse(w, snap.result)
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
	snap := getSnapshot()
	if snap.result == nil || snap.cluster == nil {
		jsonResponse(w, map[string]interface{}{"resources": []ResourceDetail{}})
		return
	}

	// Build a map of resourceRef -> findings
	findingsMap := map[string][]evaluator.Finding{}
	clusterFindings := []evaluator.Finding{} // findings not tied to a specific resource
	for _, f := range snap.result.Findings {
		if f.ResourceRef != "" {
			findingsMap[f.ResourceRef] = append(findingsMap[f.ResourceRef], f)
		} else {
			clusterFindings = append(clusterFindings, f)
		}
	}

	var resources []ResourceDetail

	// AgentGateway Backends
	for _, b := range snap.cluster.AgentgatewayBackends {
		ref := fmt.Sprintf("AgentgatewayBackend/%s/%s", b.Namespace, b.Name)
		rd := buildResourceDetail(ref, "AgentgatewayBackend", b.Name, b.Namespace, findingsMap[ref], snap.policy)
		resources = append(resources, rd)
	}

	// AgentGateway Policies
	for _, p := range snap.cluster.AgentgatewayPolicies {
		ref := fmt.Sprintf("AgentgatewayPolicy/%s/%s", p.Namespace, p.Name)
		rd := buildResourceDetail(ref, "AgentgatewayPolicy", p.Name, p.Namespace, findingsMap[ref], snap.policy)
		resources = append(resources, rd)
	}

	// Gateways
	for _, g := range snap.cluster.Gateways {
		ref := fmt.Sprintf("Gateway/%s/%s", g.Namespace, g.Name)
		rd := buildResourceDetail(ref, "Gateway", g.Name, g.Namespace, findingsMap[ref], snap.policy)
		resources = append(resources, rd)
	}

	// HTTPRoutes
	for _, h := range snap.cluster.HTTPRoutes {
		ref := fmt.Sprintf("HTTPRoute/%s/%s", h.Namespace, h.Name)
		rd := buildResourceDetail(ref, "HTTPRoute", h.Name, h.Namespace, findingsMap[ref], snap.policy)
		resources = append(resources, rd)
	}

	// Kagent Agents
	for _, a := range snap.cluster.KagentAgents {
		ref := fmt.Sprintf("Agent/%s/%s", a.Namespace, a.Name)
		rd := buildResourceDetail(ref, "Agent", a.Name, a.Namespace, findingsMap[ref], snap.policy)
		resources = append(resources, rd)
	}

	// Kagent MCPServers
	for _, m := range snap.cluster.KagentMCPServers {
		ref := fmt.Sprintf("MCPServer/%s/%s", m.Namespace, m.Name)
		rd := buildResourceDetail(ref, "MCPServer", m.Name, m.Namespace, findingsMap[ref], snap.policy)
		resources = append(resources, rd)
	}

	// Kagent RemoteMCPServers
	for _, rm := range snap.cluster.KagentRemoteMCPServers {
		ref := fmt.Sprintf("RemoteMCPServer/%s/%s", rm.Namespace, rm.Name)
		rd := buildResourceDetail(ref, "RemoteMCPServer", rm.Name, rm.Namespace, findingsMap[ref], snap.policy)
		resources = append(resources, rd)
	}

	// Add cluster-wide findings as a virtual resource
	if len(clusterFindings) > 0 {
		rd := buildResourceDetail("cluster-wide", "Cluster", "cluster-wide-policies", "", clusterFindings, snap.policy)
		resources = append(resources, rd)
	}

	jsonResponse(w, map[string]interface{}{
		"resources": resources,
		"total":     len(resources),
	})
}

func buildResourceDetail(ref, kind, name, namespace string, findings []evaluator.Finding, p evaluator.Policy) ResourceDetail {
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
				rd.Score -= p.SeverityPenalties.High
			case "Medium":
				rd.Score -= p.SeverityPenalties.Medium
			case "Low":
				rd.Score -= p.SeverityPenalties.Low
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
var (
	trendMu      sync.RWMutex
	trendHistory []TrendPoint
)

type TrendPoint struct {
	Timestamp string `json:"timestamp"`
	Score     int    `json:"score"`
	Findings  int    `json:"findings"`
	Critical  int    `json:"critical"`
	High      int    `json:"high"`
	Medium    int    `json:"medium"`
	Low       int    `json:"low"`
}

// recordTrendPoint appends a trend point from an evaluation result.
// Called only from the initial setup and the ticker goroutine — never from HTTP handlers.
func recordTrendPoint(res *evaluator.EvaluationResult) {
	if res == nil {
		return
	}
	critical, high, medium, low := 0, 0, 0, 0
	for _, f := range res.Findings {
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

	trendMu.Lock()
	trendHistory = append(trendHistory, TrendPoint{
		Timestamp: res.Timestamp.Format(time.RFC3339),
		Score:     res.Score,
		Findings:  len(res.Findings),
		Critical:  critical,
		High:      high,
		Medium:    medium,
		Low:       low,
	})

	// Keep last 100 trend points
	if len(trendHistory) > 100 {
		trendHistory = trendHistory[len(trendHistory)-100:]
	}
	trendMu.Unlock()
}

func handleTrends(w http.ResponseWriter, r *http.Request) {
	trendMu.RLock()
	trends := make([]TrendPoint, len(trendHistory))
	copy(trends, trendHistory)
	trendMu.RUnlock()

	jsonResponse(w, map[string]interface{}{"trends": trends})
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

// updatePolicyStatus writes the evaluation result back to the MCPGovernancePolicy CR status subresource.
func updatePolicyStatus(policyName string, result *evaluator.EvaluationResult) {
	if discoverer == nil || policyName == "" || result == nil {
		return
	}
	if err := discoverer.UpdatePolicyStatus(context.Background(), policyName, result); err != nil {
		log.Printf("[governance] WARNING: Failed to update policy status: %v", err)
	}
}

// updateEvaluationStatus writes the evaluation result back to GovernanceEvaluation CRs that reference the policy.
func updateEvaluationStatus(policyName string, result *evaluator.EvaluationResult) {
	if discoverer == nil || policyName == "" || result == nil {
		return
	}
	if err := discoverer.UpdateEvaluationStatus(context.Background(), policyName, result); err != nil {
		log.Printf("[governance] WARNING: Failed to update evaluation status: %v", err)
	}
}

// ---------- AI Agent ----------

// initAIAgent initializes the AI governance agent
func initAIAgent(ctx context.Context) {
	stateMu.RLock()
	p := policy
	stateMu.RUnlock()

	provider := p.AIProvider
	if provider == "" {
		provider = "gemini"
	}

	if !aiagent.IsAvailable(provider) {
		log.Printf("[ai-agent] Provider %q not available — AI agent disabled", provider)
		aiAgentErr = fmt.Errorf("AI agent provider %q is not available (check env vars or Ollama endpoint)", provider)
		return
	}

	config := aiagent.AIAgentConfig{
		Provider:       provider,
		Model:          p.AIModel,
		OllamaEndpoint: p.OllamaEndpoint,
	}

	var err error
	aiAgent, err = aiagent.NewGovernanceAgent(ctx, config)
	if err != nil {
		log.Printf("[ai-agent] Failed to initialize AI agent: %v", err)
		aiAgentErr = err
		return
	}

	log.Printf("[ai-agent] AI Governance Agent initialized successfully (provider=%s)", provider)

	// Parse scan interval from policy
	if p.AIScanInterval != "" {
		if d, err := time.ParseDuration(p.AIScanInterval); err == nil && d >= 1*time.Minute {
			aiMinInterval = d
			log.Printf("[ai-agent] Scan interval set to %v", d)
		} else {
			log.Printf("[ai-agent] Invalid scanInterval %q, using default %v", p.AIScanInterval, aiMinInterval)
		}
	}

	// Respect scanEnabled from policy
	stateMu.Lock()
	aiScanPaused = !p.AIScanEnabled
	stateMu.Unlock()
	if aiScanPaused {
		log.Printf("[ai-agent] Periodic scanning is disabled (scanEnabled=false)")
	}

	// Run initial AI evaluation (always runs once regardless of pause)
	stateMu.RLock()
	cs := currentState
	pol := policy
	res := lastResult
	stateMu.RUnlock()

	if cs != nil && res != nil {
		forceRunAIEvaluation(ctx, cs, pol, res)
	}
}

// runAIEvaluation runs the AI agent with rate-limiting and pause checks
func runAIEvaluation(ctx context.Context, state *evaluator.ClusterState, p evaluator.Policy, result *evaluator.EvaluationResult) {
	if aiAgent == nil {
		return
	}

	// Check if scanning is paused
	stateMu.RLock()
	paused := aiScanPaused
	lastRun := aiLastRun
	backoff := aiBackoff
	stateMu.RUnlock()

	if paused {
		return // scanning is paused
	}

	// Rate-limit AI evaluations to avoid burning through API quotas
	interval := aiMinInterval
	if backoff > interval {
		interval = backoff
	}
	if time.Since(lastRun) < interval {
		return // too soon, skip this cycle
	}

	forceRunAIEvaluation(ctx, state, p, result)
}

// forceRunAIEvaluation runs the AI agent immediately, bypassing rate-limit and pause checks.
// Used for initial evaluation and manual refresh via API.
func forceRunAIEvaluation(ctx context.Context, state *evaluator.ClusterState, p evaluator.Policy, result *evaluator.EvaluationResult) {
	if aiAgent == nil {
		return
	}

	log.Printf("[ai-agent] Running AI governance evaluation...")

	stateMu.Lock()
	aiLastRun = time.Now()
	stateMu.Unlock()

	aiResult, err := aiAgent.Evaluate(ctx, state, p, result)
	if err != nil {
		log.Printf("[ai-agent] AI evaluation failed: %v", err)
		stateMu.Lock()
		aiAgentErr = err
		// Exponential backoff: 5m → 10m → 20m, capped at 30m
		if aiBackoff == 0 {
			aiBackoff = aiMinInterval
		} else {
			aiBackoff *= 2
			if aiBackoff > 30*time.Minute {
				aiBackoff = 30 * time.Minute
			}
		}
		log.Printf("[ai-agent] Next retry in %v", aiBackoff)
		stateMu.Unlock()
		return
	}

	stateMu.Lock()
	lastAIResult = aiResult
	aiAgentErr = nil
	aiBackoff = 0 // reset backoff on success
	stateMu.Unlock()

	log.Printf("[ai-agent] AI evaluation complete. AI Score: %d (Grade: %s) vs Algorithmic Score: %d",
		aiResult.Score, aiResult.Grade, result.Score)
}

// handleAIScore returns the AI agent's governance assessment
func handleAIScore(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()

	stateMu.RLock()
	aiResult := lastAIResult
	aiErr := aiAgentErr
	paused := aiScanPaused
	stateMu.RUnlock()

	scanConfig := map[string]interface{}{
		"scanInterval": aiMinInterval.String(),
		"scanPaused":   paused,
	}

	if !snap.policy.EnableAIAgent {
		jsonResponse(w, map[string]interface{}{
			"enabled":    false,
			"scanConfig": scanConfig,
			"message":    "AI agent scoring is not enabled in the governance policy. Set enableAIAgent: true in MCPGovernancePolicy spec.",
		})
		return
	}

	if aiResult == nil {
		errMsg := "AI evaluation has not completed yet"
		if aiErr != nil {
			errMsg = fmt.Sprintf("AI agent error: %v", aiErr)
		}
		jsonResponse(w, map[string]interface{}{
			"enabled":    true,
			"available":  false,
			"scanConfig": scanConfig,
			"message":    errMsg,
		})
		return
	}

	// Include comparison with algorithmic score
	algorithmicScore := 0
	algorithmicGrade := "F"
	if snap.result != nil {
		algorithmicScore = snap.result.Score
		algorithmicGrade = getGrade(snap.result.Score)
	}

	jsonResponse(w, map[string]interface{}{
		"enabled":    true,
		"available":  true,
		"scanConfig": scanConfig,
		"aiScore":    aiResult,
		"comparison": map[string]interface{}{
			"algorithmicScore": algorithmicScore,
			"algorithmicGrade": algorithmicGrade,
			"aiScore":          aiResult.Score,
			"aiGrade":          aiResult.Grade,
			"scoreDifference":  aiResult.Score - algorithmicScore,
		},
	})
}

// handleAIRefresh triggers an immediate AI evaluation (bypasses rate-limit and pause)
func handleAIRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if aiAgent == nil {
		jsonResponse(w, map[string]interface{}{
			"success": false,
			"message": "AI agent is not initialized",
		})
		return
	}

	snap := getSnapshot()
	if snap.cluster == nil || snap.result == nil {
		jsonResponse(w, map[string]interface{}{
			"success": false,
			"message": "Cluster state not yet available, please wait for initial scan",
		})
		return
	}

	go forceRunAIEvaluation(context.Background(), snap.cluster, snap.policy, snap.result)

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": "AI evaluation triggered. Results will be available shortly.",
	})
}

// handleAIToggle toggles the periodic AI scanning on/off
func handleAIToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stateMu.Lock()
	aiScanPaused = !aiScanPaused
	paused := aiScanPaused
	stateMu.Unlock()

	status := "resumed"
	if paused {
		status = "paused"
	}
	log.Printf("[ai-agent] Periodic scanning %s via API", status)

	jsonResponse(w, map[string]interface{}{
		"success":    true,
		"scanPaused": paused,
		"message":    fmt.Sprintf("AI periodic scanning %s", status),
	})
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
