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
	v1alpha1 "github.com/techwithhuz/mcp-security-governance/controller/pkg/apis/governance/v1alpha1"
	"github.com/techwithhuz/mcp-security-governance/controller/pkg/discovery"
	"github.com/techwithhuz/mcp-security-governance/controller/pkg/evaluator"
	"github.com/techwithhuz/mcp-security-governance/controller/pkg/inventory"
	"github.com/techwithhuz/mcp-security-governance/controller/pkg/watcher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Governance scan state
	lastScanTime  time.Time  // when the last governance scan completed
	scanInterval  = 5 * time.Minute // default; overridden by policy.ScanInterval
	scanMode      = "watch" // "watch" (reconcile on change) or "poll" (periodic timer)

	// Resource watcher (reconcile-based scanning)
	resourceWatcher *watcher.ResourceWatcher

	// Inventory watcher — watches MCPServerCatalog from Agent Registry
	// and scores each one with a Verified Score (publisher, transport, deployment, tools, usage)
	inventoryWatcher *inventory.Watcher
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

	// Parse scan interval from policy (used as resync period for watcher fallback)
	if policy.ScanInterval != "" {
		if d, err := time.ParseDuration(policy.ScanInterval); err == nil && d >= 30*time.Second {
			scanInterval = d
		} else {
			log.Printf("[governance] Invalid scanInterval %q, using default %v", policy.ScanInterval, scanInterval)
		}
	}
	lastScanTime = time.Now()

	// Start resource watcher (reconcile on change) or fall back to periodic polling
	if discoverer != nil {
		w, err := watcher.New(watcher.Config{
			DynamicClient: discoverer.DynamicClient(),
			Reconcile: func(reason string) {
				doPeriodicScan()
			},
			Debounce:     3 * time.Second,
			ResyncPeriod: scanInterval, // full resync as safety net
		})
		if err != nil {
			log.Printf("[governance] WARNING: Failed to create resource watcher: %v — falling back to polling", err)
			scanMode = "poll"
			startPollingLoop()
		} else {
			resourceWatcher = w
			scanMode = "watch"
			log.Printf("[governance] Watch mode enabled — reconciling on resource changes (resync every %v)", scanInterval)
			go resourceWatcher.Start(context.Background())
		}
	} else {
		scanMode = "poll"
		log.Printf("[governance] No K8s connection — using polling mode (every %v)", scanInterval)
		startPollingLoop()
	}

	// Start inventory watcher — watches MCPServerCatalog from Agent Registry
	// and scores each resource with a Verified Score on add/update/delete.
	// No polling needed: the controller reconciliation loop handles it.
	if discoverer != nil {
		invPolicy := inventory.ScoringPolicy{
			MaxToolsWarning:  policy.MaxToolsWarning,
			MaxToolsCritical: policy.MaxToolsCritical,
		}
		// Apply verifiedCatalogScoring overrides from governance policy CR
		if vcs, ok := policy.VerifiedCatalogScoring.(*v1alpha1.VerifiedCatalogScoringConfig); ok && vcs != nil {
			if vcs.SecurityWeight > 0 {
				invPolicy.SecurityWeight = vcs.SecurityWeight
			}
			if vcs.TrustWeight > 0 {
				invPolicy.TrustWeight = vcs.TrustWeight
			}
			if vcs.ComplianceWeight > 0 {
				invPolicy.ComplianceWeight = vcs.ComplianceWeight
			}
			if vcs.VerifiedThreshold > 0 {
				invPolicy.VerifiedThreshold = vcs.VerifiedThreshold
			}
			if vcs.UnverifiedThreshold > 0 {
				invPolicy.UnverifiedThreshold = vcs.UnverifiedThreshold
			}
			if len(vcs.CheckMaxScores) > 0 {
				invPolicy.CheckMaxScores = vcs.CheckMaxScores
			}
		}
		iw, err := inventory.NewWatcher(inventory.WatcherConfig{
			DynamicClient: discoverer.DynamicClient(),
			Policy: invPolicy,
			Namespace: "", // watch all namespaces
			OnChange: func() {
				// Log when inventory verified scores change
				log.Printf("[inventory] Verified resources updated — scores reconciled")
			},
		})
		if err != nil {
			log.Printf("[governance] WARNING: Failed to create inventory watcher: %v", err)
		} else {
			inventoryWatcher = iw
			log.Printf("[governance] Inventory watcher enabled — scoring MCPServerCatalog resources on change")
			go inventoryWatcher.Start(context.Background())
		}
	} else {
		log.Printf("[governance] No K8s connection — inventory watcher disabled")
	}

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
	// MCP-server-centric endpoints
	mux.HandleFunc("/api/governance/mcp-servers", handleMCPServers)
	mux.HandleFunc("/api/governance/mcp-servers/summary", handleMCPServerSummary)
	mux.HandleFunc("/api/governance/mcp-servers/detail", handleMCPServerDetail)
	// Scan management endpoints
	mux.HandleFunc("/api/governance/scan/refresh", handleRefreshScan)
	mux.HandleFunc("/api/governance/scan/status", handleScanStatus)
	// Inventory verified score endpoints
	mux.HandleFunc("/api/governance/inventory/verified", handleInventoryVerified)
	mux.HandleFunc("/api/governance/inventory/summary", handleInventorySummary)
	mux.HandleFunc("/api/governance/inventory/detail", handleInventoryDetail)

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
	stateMu.RLock()
	scanTime := lastScanTime
	interval := scanInterval
	mode := scanMode
	stateMu.RUnlock()

	resp := map[string]interface{}{
		"status":       "healthy",
		"version":      Version,
		"lastScanTime": scanTime.Format(time.RFC3339),
		"scanInterval": interval.String(),
		"scanMode":     mode,
	}

	// Include watcher stats if in watch mode
	if mode == "watch" && resourceWatcher != nil {
		stats := resourceWatcher.Stats()
		resp["watcher"] = map[string]interface{}{
			"activeWatches":  stats.ActiveWatches,
			"totalGVRs":      stats.TotalGVRs,
			"eventCount":     stats.EventCount,
			"reconcileCount": stats.ReconcileCount,
			"lastEvent":      stats.LastEvent.Format(time.RFC3339),
			"lastReconcile":  stats.LastReconcile.Format(time.RFC3339),
		}
	}

	// Include inventory watcher stats
	if inventoryWatcher != nil {
		iStats := inventoryWatcher.Stats()
		iSummary := inventoryWatcher.GetSummary()
		resp["inventory"] = map[string]interface{}{
			"enabled":        true,
			"resourceCount":  iStats.ResourceCount,
			"eventCount":     iStats.EventCount,
			"reconcileCount": iStats.ReconcileCount,
			"lastEvent":      iStats.LastEvent.Format(time.RFC3339),
			"lastReconcile":  iStats.LastReconcile.Format(time.RFC3339),
			"averageScore":   iSummary.AverageScore,
			"verifiedCount":  iSummary.VerifiedCount,
			"warningCount":   iSummary.WarningCount,
		}
	}

	jsonResponse(w, resp)
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
	type serverContribution struct {
		Name  string `json:"name"`
		Score int    `json:"score"`
		Grade string `json:"grade"`
	}
	type categoryDetail struct {
		Category    string               `json:"category"`
		Score       int                  `json:"score"`
		Weight      int                  `json:"weight"`
		Weighted    float64              `json:"weighted"`
		Status      string               `json:"status"`
		InfraAbsent bool                 `json:"infraAbsent"`
		Servers     []serverContribution `json:"servers"` // per-server scores for this category
	}

	// Mapping from category display name to MCPServerScoreBreakdown field getter
	type catDef struct {
		Name     string
		Required bool
		Weight   int
		ClScore  int
		GetScore func(v evaluator.MCPServerView) int
	}

	pw := snap.policy.Weights
	bd := snap.result.ScoreBreakdown

	allCats := []catDef{
		{"AgentGateway Compliance", snap.policy.RequireAgentGateway, pw.AgentGatewayIntegration, bd.AgentGatewayScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.GatewayRouting }},
		{"Authentication", snap.policy.RequireJWTAuth, pw.Authentication, bd.AuthenticationScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.Authentication }},
		{"Authorization", snap.policy.RequireRBAC, pw.Authorization, bd.AuthorizationScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.Authorization }},
		{"CORS", snap.policy.RequireCORS, pw.CORSPolicy, bd.CORSScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.CORS }},
		{"TLS", snap.policy.RequireTLS, pw.TLSEncryption, bd.TLSScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.TLS }},
		{"Prompt Guard", snap.policy.RequirePromptGuard, pw.PromptGuard, bd.PromptGuardScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.PromptGuard }},
		{"Rate Limit", snap.policy.RequireRateLimit, pw.RateLimit, bd.RateLimitScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.RateLimit }},
		{"Tool Scope", snap.policy.MaxToolsWarning > 0 || snap.policy.MaxToolsCritical > 0, pw.ToolScope, bd.ToolScopeScore,
			func(v evaluator.MCPServerView) int { return v.ScoreBreakdown.ToolScope }},
	}

	totalWeight := 0
	var cats []categoryDetail

	for _, c := range allCats {
		if !c.Required {
			continue
		}
		totalWeight += c.Weight

		// Collect per-server scores for this category
		var servers []serverContribution
		for _, v := range snap.result.MCPServerViews {
			s := c.GetScore(v)
			servers = append(servers, serverContribution{
				Name:  v.Name,
				Score: s,
				Grade: getGrade(s),
			})
		}

		cats = append(cats, categoryDetail{
			Category:    c.Name,
			Score:       c.ClScore,
			Weight:      c.Weight,
			Status:      statusLabel(c.ClScore),
			InfraAbsent: bd.InfraAbsent[c.Name],
			Servers:     servers,
		})
	}

	if totalWeight == 0 {
		totalWeight = 100
	}

	// Recalculate weighted scores with correct totalWeight
	for i := range cats {
		cats[i].Weighted = float64(cats[i].Score*cats[i].Weight) / float64(totalWeight)
	}

	numServers := len(snap.result.MCPServerViews)
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
			"Score is a weighted average of %d governance categories. Each category score is the average across %d MCP server(s). The final score %d/100 = Grade %s.",
			len(cats), numServers, snap.result.Score, getGrade(snap.result.Score)),
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

	// Populate verified catalog scores from inventory watcher if available
	if inventoryWatcher != nil {
		verifiedResources := inventoryWatcher.GetResources()
		verifiedScores := make([]v1alpha1.VerifiedCatalogScore, 0, len(verifiedResources))

		for _, vr := range verifiedResources {
			score := v1alpha1.VerifiedCatalogScore{
				CatalogName:     vr.CatalogName,
				Namespace:       vr.Namespace,
				ResourceVersion: vr.ResourceVersion,
				Status:          vr.VerifiedScore.Status,
				CompositeScore:  vr.VerifiedScore.Score,
				SecurityScore:   vr.VerifiedScore.SecurityScore,
				TrustScore:      vr.VerifiedScore.TrustScore,
				ComplianceScore: vr.VerifiedScore.ComplianceScore,
			}

			// Convert VerifiedCheck to CatalogScoringCheck
			for _, check := range vr.VerifiedScore.Checks {
				score.Checks = append(score.Checks, v1alpha1.CatalogScoringCheck{
					ID:        check.ID,
					Name:      check.Name,
					Points:    check.Score,
					MaxPoints: check.MaxScore,
				})
			}

			// Set LastScored timestamp
			if !vr.LastScored.IsZero() {
				metaTime := metav1.NewTime(vr.LastScored)
				score.LastScored = &metaTime
			}

			verifiedScores = append(verifiedScores, score)
		}

		result.VerifiedCatalogScores = verifiedScores
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

// ---------- MCP Server Endpoints ----------

// doPeriodicScan runs a full governance scan and updates all state
func doPeriodicScan() {
	cs := doDiscovery()
	p := loadPolicy()
	res := evaluator.Evaluate(cs.FilterByNamespaces(p.TargetNamespaces, p.ExcludeNamespaces), p)

	stateMu.Lock()
	currentState = cs
	policy = p
	lastCluster = cs
	lastResult = res
	lastScanTime = time.Now()
	stateMu.Unlock()

	recordTrendPoint(res)
	updatePolicyStatus(p.Name, res)
	updateEvaluationStatus(p.Name, res)
	log.Printf("[governance] Scan complete. Score: %d, Findings: %d, MCP Servers: %d", res.Score, len(res.Findings), len(res.MCPServerViews))

	// Run AI agent evaluation if enabled
	if p.EnableAIAgent {
		runAIEvaluation(context.Background(), cs, p, res)
	}
}

// startPollingLoop starts a traditional ticker-based scan loop.
// Used as fallback when the resource watcher cannot be created.
func startPollingLoop() {
	go func() {
		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()
		for range ticker.C {
			doPeriodicScan()
		}
	}()
}

// handleRefreshScan triggers an on-demand governance scan (POST only)
func handleRefreshScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[governance] On-demand scan triggered via API")
	doPeriodicScan()

	stateMu.RLock()
	result := lastResult
	scanTime := lastScanTime
	stateMu.RUnlock()

	jsonResponse(w, map[string]interface{}{
		"status":       "completed",
		"score":        result.Score,
		"findings":     len(result.Findings),
		"mcpServers":   len(result.MCPServerViews),
		"lastScanTime": scanTime.Format(time.RFC3339),
	})
}

// handleScanStatus returns the current scan status and interval
func handleScanStatus(w http.ResponseWriter, r *http.Request) {
	stateMu.RLock()
	scanTime := lastScanTime
	interval := scanInterval
	p := policy
	mode := scanMode
	stateMu.RUnlock()

	resp := map[string]interface{}{
		"lastScanTime":     scanTime.Format(time.RFC3339),
		"scanInterval":     interval.String(),
		"scanIntervalSpec": p.ScanInterval,
		"scanMode":         mode,
	}

	if mode == "watch" {
		resp["description"] = "Reconcile-on-change mode: the controller watches Kubernetes resources and re-evaluates governance scores within seconds of any change."
		if resourceWatcher != nil {
			stats := resourceWatcher.Stats()
			resp["watcher"] = stats
		}
	} else {
		nextScan := scanTime.Add(interval)
		resp["nextScanTime"] = nextScan.Format(time.RFC3339)
		resp["description"] = "Polling mode: the controller periodically scans the cluster at a fixed interval."
	}

	jsonResponse(w, resp)
}

// handleMCPServers returns all MCP server views with their scores and related resources
func handleMCPServers(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()
	if snap.result == nil {
		jsonResponse(w, map[string]interface{}{
			"servers": []evaluator.MCPServerView{},
			"summary": evaluator.MCPServerSummary{},
		})
		return
	}
	jsonResponse(w, map[string]interface{}{
		"servers": snap.result.MCPServerViews,
		"summary": snap.result.MCPServerSummary,
	})
}

// handleMCPServerSummary returns only the cluster-level MCP server summary
func handleMCPServerSummary(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()
	if snap.result == nil {
		jsonResponse(w, evaluator.MCPServerSummary{})
		return
	}
	jsonResponse(w, snap.result.MCPServerSummary)
}

// handleMCPServerDetail returns detailed info for a single MCP server by ID
func handleMCPServerDetail(w http.ResponseWriter, r *http.Request) {
	snap := getSnapshot()
	if snap.result == nil {
		http.Error(w, "No evaluation available", http.StatusServiceUnavailable)
		return
	}

	serverID := r.URL.Query().Get("id")
	if serverID == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	for _, view := range snap.result.MCPServerViews {
		if view.ID == serverID {
			jsonResponse(w, view)
			return
		}
	}

	http.Error(w, "MCP server not found", http.StatusNotFound)
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

// ---------- Inventory Verified Score Endpoints ----------

// handleInventoryVerified returns all MCPServerCatalog entries with their Verified Scores.
func handleInventoryVerified(w http.ResponseWriter, r *http.Request) {
	if inventoryWatcher == nil {
		jsonResponse(w, map[string]interface{}{
			"enabled":   false,
			"message":   "Inventory watcher is not running — no Kubernetes connection or MCPServerCatalog CRD not available",
			"resources": []interface{}{},
			"summary":   inventory.VerifiedSummary{},
		})
		return
	}

	resources := inventoryWatcher.GetResources()
	summary := inventoryWatcher.GetSummary()

	jsonResponse(w, map[string]interface{}{
		"enabled":   true,
		"resources": resources,
		"summary":   summary,
		"total":     len(resources),
	})
}

// handleInventorySummary returns only the cluster-level verified summary.
func handleInventorySummary(w http.ResponseWriter, r *http.Request) {
	if inventoryWatcher == nil {
		jsonResponse(w, map[string]interface{}{
			"enabled": false,
			"summary": inventory.VerifiedSummary{},
		})
		return
	}

	summary := inventoryWatcher.GetSummary()
	jsonResponse(w, map[string]interface{}{
		"enabled": true,
		"summary": summary,
	})
}

// handleInventoryDetail returns detailed verified score for a single MCPServerCatalog.
func handleInventoryDetail(w http.ResponseWriter, r *http.Request) {
	if inventoryWatcher == nil {
		http.Error(w, "Inventory watcher not running", http.StatusServiceUnavailable)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	if namespace == "" || name == "" {
		http.Error(w, "Missing 'namespace' and 'name' query parameters", http.StatusBadRequest)
		return
	}

	res, found := inventoryWatcher.GetResource(namespace, name)
	if !found {
		http.Error(w, fmt.Sprintf("MCPServerCatalog %s/%s not found in verified resources", namespace, name), http.StatusNotFound)
		return
	}

	jsonResponse(w, res)
}
