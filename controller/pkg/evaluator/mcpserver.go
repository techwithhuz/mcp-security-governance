package evaluator

import (
	"fmt"
	"strings"
)

// MCPServerView represents a unified view of an MCP server and all related resources.
// This is the primary entity in the MCP-centric governance model.
type MCPServerView struct {
	// Identity
	ID        string `json:"id"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Source    string `json:"source"` // "KagentMCPServer", "KagentRemoteMCPServer", "AgentgatewayBackendTarget", "Service"

	// MCP Server details
	Transport string   `json:"transport,omitempty"`
	URL       string   `json:"url,omitempty"`
	Port      int      `json:"port,omitempty"`
	ToolCount int      `json:"toolCount"`
	ToolNames []string `json:"toolNames"`

	// Effective tools (after policy enforcement)
	EffectiveToolCount int                         `json:"effectiveToolCount"`
	EffectiveToolNames []string                    `json:"effectiveToolNames,omitempty"`
	HasToolRestriction bool                        `json:"hasToolRestriction"`
	ToolsByRoute       map[string][]string         `json:"toolsByRoute,omitempty"` // Route name -> allowed tools for that route
	ToolsByPolicy      map[string]map[string][]string `json:"toolsByPolicy,omitempty"` // Route name -> Policy name -> allowed tools
	PathTools          map[string][]string         `json:"pathTools,omitempty"` // Path label (e.g., "/ro", "/rw") -> allowed tools

	// Related resources (populated by correlation)
	RelatedBackends  []RelatedResource `json:"relatedBackends"`
	RelatedPolicies  []RelatedResource `json:"relatedPolicies"`
	RelatedRoutes    []RelatedResource `json:"relatedRoutes"`
	RelatedGateways  []RelatedResource `json:"relatedGateways"`
	RelatedAgents    []RelatedResource `json:"relatedAgents"`
	RelatedServices  []RelatedResource `json:"relatedServices"`

	// Security posture (derived from related resources)
	RoutedThroughGateway bool   `json:"routedThroughGateway"`
	HasTLS               bool   `json:"hasTLS"`
	HasAuth              bool   `json:"hasAuth"`
	HasJWT               bool   `json:"hasJWT"`
	JWTMode              string `json:"jwtMode,omitempty"`
	HasRBAC              bool   `json:"hasRBAC"`
	HasCORS              bool   `json:"hasCORS"`
	HasRateLimit         bool   `json:"hasRateLimit"`
	HasPromptGuard       bool   `json:"hasPromptGuard"`

	// Scoring
	Score             int                      `json:"score"`
	Grade             string                   `json:"grade"`
	Status            string                   `json:"status"` // "compliant", "warning", "failing", "critical"
	Findings          []Finding                `json:"findings"`
	ScoreBreakdown    MCPServerScoreBreakdown  `json:"scoreBreakdown"`
	ScoreExplanations []ScoreExplanation       `json:"scoreExplanations"`
}

// RelatedResource is a reference to a Kubernetes resource related to an MCP server.
type RelatedResource struct {
	Kind      string                 `json:"kind"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Status    string                 `json:"status"` // "healthy", "warning", "critical", "missing"
	Details   map[string]interface{} `json:"details,omitempty"`
}

// MCPServerScoreBreakdown is the per-category score for a single MCP server.
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

// ScoreExplanation describes how a single security control score was calculated.
type ScoreExplanation struct {
	Category    string   `json:"category"`
	Score       int      `json:"score"`
	MaxScore    int      `json:"maxScore"`
	Status      string   `json:"status"`      // "pass", "partial", "fail", "not-required"
	Reasons     []string `json:"reasons"`      // What contributes to the current score
	Suggestions []string `json:"suggestions"`  // What could improve the score
	Sources     []string `json:"sources"`      // Which resources provide this control
}

// MCPServerSummary is the cluster-level summary of all MCP servers.
type MCPServerSummary struct {
	TotalMCPServers int `json:"totalMCPServers"`
	RoutedServers   int `json:"routedServers"`
	UnroutedServers int `json:"unroutedServers"`
	SecuredServers  int `json:"securedServers"`
	AtRiskServers   int `json:"atRiskServers"`
	CriticalServers int `json:"criticalServers"`
	TotalTools      int `json:"totalTools"`
	ExposedTools    int `json:"exposedTools"`
	AverageScore    int `json:"averageScore"`
}

// BuildMCPServerViews correlates cluster state into MCP-server-centric views.
func BuildMCPServerViews(state *ClusterState, findings []Finding, policy Policy) []MCPServerView {
	var views []MCPServerView

	// 1. Kagent MCPServers
	for _, mcp := range state.KagentMCPServers {
		view := MCPServerView{
			ID:        fmt.Sprintf("KagentMCPServer/%s/%s", mcp.Namespace, mcp.Name),
			Name:      mcp.Name,
			Namespace: mcp.Namespace,
			Source:    "KagentMCPServer",
			Transport: mcp.Transport,
			Port:      mcp.Port,
		}
		correlateMCPServer(&view, state, findings, policy)
		views = append(views, view)
	}

	// 2. Kagent RemoteMCPServers
	for _, rms := range state.KagentRemoteMCPServers {
		view := MCPServerView{
			ID:        fmt.Sprintf("KagentRemoteMCPServer/%s/%s", rms.Namespace, rms.Name),
			Name:      rms.Name,
			Namespace: rms.Namespace,
			Source:    "KagentRemoteMCPServer",
			URL:       rms.URL,
			ToolCount: rms.ToolCount,
			ToolNames: rms.ToolNames,
		}
		correlateMCPServer(&view, state, findings, policy)
		views = append(views, view)
	}

	return views
}

// correlateMCPServer finds all resources related to this MCP server and computes its score.
func correlateMCPServer(view *MCPServerView, state *ClusterState, findings []Finding, policy Policy) {
	// --- Find related backends ---
	for _, b := range state.AgentgatewayBackends {
		if b.BackendType != "mcp" {
			continue
		}
		for _, t := range b.MCPTargets {
			if matchesMCPTarget(view, t) {
				view.RelatedBackends = append(view.RelatedBackends, RelatedResource{
					Kind:      "AgentgatewayBackend",
					Name:      b.Name,
					Namespace: b.Namespace,
					Status:    mcpBackendStatus(b),
					Details: map[string]interface{}{
						"backendType": b.BackendType,
						"hasTLS":      b.HasTLS,
						"targetName":  t.Name,
						"hasAuth":     t.HasAuth,
						"hasRBAC":     t.HasRBAC,
					},
				})
				if b.HasTLS {
					view.HasTLS = true
				}
				if t.HasAuth {
					view.HasAuth = true
				}
				if t.HasRBAC {
					view.HasRBAC = true
				}
			}
		}
	}

	// --- Find related HTTPRoutes ---
	// An HTTPRoute relates to this MCP server if:
	//   a) it references one of the server's related backends, OR
	//   b) it directly references the MCP server's service name as a backendRef
	addedRoutes := map[string]bool{}
	for _, route := range state.HTTPRoutes {
		matched := false
		routeUsesAGWBackend := false

		for _, backendRef := range route.BackendRefs {
			// Check if the route references one of our related backends
			for _, rb := range view.RelatedBackends {
				if rb.Name == backendRef {
					matched = true
					routeUsesAGWBackend = true
				}
			}
			// Check if the route directly references this MCP server's service
			if backendRef == view.Name {
				matched = true
				// Check if this backendRef is actually an AgentgatewayBackend
				for _, b := range state.AgentgatewayBackends {
					if b.Name == backendRef {
						routeUsesAGWBackend = true
					}
				}
			}
		}

		if matched && !addedRoutes[route.Name] {
			addedRoutes[route.Name] = true
			view.RelatedRoutes = append(view.RelatedRoutes, RelatedResource{
				Kind:      "HTTPRoute",
				Name:      route.Name,
				Namespace: route.Namespace,
				Status:    mcpRouteStatus(route),
				Details: map[string]interface{}{
					"parentGateway":          route.ParentGateway,
					"parentGatewayNamespace": route.ParentGatewayNamespace,
					"hasCORSFilter":          route.HasCORSFilter,
					"usesAGWBackend":         routeUsesAGWBackend,
					"paths":                  route.Paths,
				},
			})
			if route.HasCORSFilter {
				view.HasCORS = true
			}
			// If this route uses an agentgateway backend and has a parent gateway,
			// it confirms the MCP server is routed through agentgateway
			if routeUsesAGWBackend {
				view.RoutedThroughGateway = true
			}
		}
	}

	// --- Find related Gateways ---
	// Only include gateways that are explicitly referenced as parentGateway
	// by this MCP server's related routes. We match by name AND namespace.
	// The parentRef namespace defaults to the route's own namespace per Gateway API spec.
	type gwKey struct{ name, ns string }
	gwKeys := map[gwKey]bool{}
	for _, route := range view.RelatedRoutes {
		parentName, _ := route.Details["parentGateway"].(string)
		if parentName != "" {
			// Use explicit parentGatewayNamespace if set, otherwise default to route's namespace
			parentNS, _ := route.Details["parentGatewayNamespace"].(string)
			if parentNS == "" {
				parentNS = route.Namespace
			}
			gwKeys[gwKey{parentName, parentNS}] = true
		}
	}
	for _, gw := range state.Gateways {
		if gwKeys[gwKey{gw.Name, gw.Namespace}] {
			view.RelatedGateways = append(view.RelatedGateways, RelatedResource{
				Kind:      "Gateway",
				Name:      gw.Name,
				Namespace: gw.Namespace,
				Status:    mcpGatewayStatus(gw),
				Details: map[string]interface{}{
					"gatewayClassName": gw.GatewayClassName,
					"programmed":       gw.Programmed,
				},
			})
			if gw.GatewayClassName == "agentgateway" && len(view.RelatedBackends) > 0 {
				view.RoutedThroughGateway = true
			}
		}
	}

	// --- Find related Policies ---
	// A policy is related to this MCP server if its targetRef points to one of
	// the server's related gateways or routes. We compare by name AND namespace
	// (the targetRef namespace defaults to the policy's own namespace per Gateway API).
	for _, p := range state.AgentgatewayPolicies {
		related := false
		for _, tr := range p.TargetRefs {
			// Determine the effective namespace for this targetRef
			// (Gateway API: defaults to the policy's own namespace)
			trNS := p.Namespace
			for _, gw := range view.RelatedGateways {
				if tr.Kind == "Gateway" && tr.Name == gw.Name && trNS == gw.Namespace {
					related = true
				}
			}
			for _, rt := range view.RelatedRoutes {
				if tr.Kind == "HTTPRoute" && tr.Name == rt.Name && trNS == rt.Namespace {
					related = true
				}
			}
			for _, rb := range view.RelatedBackends {
				if tr.Kind == "AgentgatewayBackend" && tr.Name == rb.Name && trNS == rb.Namespace {
					related = true
				}
			}
		}
		// If no target refs, assume cluster-wide policy
		if len(p.TargetRefs) == 0 {
			related = true
		}
		if related {
			view.RelatedPolicies = append(view.RelatedPolicies, RelatedResource{
				Kind:      "AgentgatewayPolicy",
				Name:      p.Name,
				Namespace: p.Namespace,
				Status:    mcpPolicyStatus(p),
				Details: map[string]interface{}{
					"hasJWT":        p.HasJWT,
					"jwtMode":       p.JWTMode,
					"hasCORS":       p.HasCORS,
					"hasRBAC":       p.HasRBAC,
					"hasRateLimit":  p.HasRateLimit,
					"hasPromptGuard": p.HasPromptGuard,
					"allowedTools":  p.AllowedTools,
				},
			})
			if p.HasJWT {
				view.HasJWT = true
				view.JWTMode = p.JWTMode
			}
			if p.HasCORS {
				view.HasCORS = true
			}
			if p.HasRBAC {
				view.HasRBAC = true
			}
			if p.HasRateLimit {
				view.HasRateLimit = true
			}
			if p.HasPromptGuard {
				view.HasPromptGuard = true
			}
			// Collect allowed tools from authorization policies
			if len(p.AllowedTools) > 0 {
				view.HasToolRestriction = true
				view.EffectiveToolNames = append(view.EffectiveToolNames, p.AllowedTools...)
			}
		}
	}

	// Back-annotate HTTPRoute details with policy-level CORS
	// If a related policy has CORS, mark the route as covered
	policyCORS := false
	for _, p := range view.RelatedPolicies {
		if hasCORS, ok := p.Details["hasCORS"].(bool); ok && hasCORS {
			policyCORS = true
			break
		}
	}
	if policyCORS {
		for i := range view.RelatedRoutes {
			view.RelatedRoutes[i].Details["hasCORSFromPolicy"] = true
		}
	}

	// Compute effective tool count based on policy restrictions
	if view.HasToolRestriction && len(view.RelatedPolicies) > 0 {
		// When multiple policies restrict tools (e.g., for different path rules in the same HTTPRoute),
		// they may restrict different sets of tools:
		// - /ro path policy: allows 10 read-only tools
		// - /rw path policy: allows 10 different read-write tools
		//
		// Instead of counting all unique tools (20), we should count tools restrictively:
		// For each related HTTPRoute, take the SMALLEST tool set across all policies for that route.
		// This represents the most restrictive exposure for that route.
		// Then take the MAXIMUM across all routes (the least restrictive route).
		//
		// Also populate ToolsByRoute and ToolsByPolicy for UI display of per-route and per-policy tool restrictions.
		
		type policyToolInfo struct {
			name  string
			tools map[string]bool
		}
		toolSetsByRoute := make(map[string][]policyToolInfo)
		
		for _, pResource := range view.RelatedPolicies {
			allowedTools, ok := pResource.Details["allowedTools"].([]string)
			if !ok || len(allowedTools) == 0 {
				continue
			}
			// Convert to set
			toolSet := make(map[string]bool)
			for _, tool := range allowedTools {
				toolSet[tool] = true
			}
			
			// Map policy to its related routes
			for _, route := range view.RelatedRoutes {
				routeKey := route.Name
				toolSetsByRoute[routeKey] = append(toolSetsByRoute[routeKey], policyToolInfo{
					name:  pResource.Name,
					tools: toolSet,
				})
			}
		}
		
		// Initialize maps
		view.ToolsByRoute = make(map[string][]string)
		view.ToolsByPolicy = make(map[string]map[string][]string)
		view.PathTools = make(map[string][]string)
		
		// For each route, track tools by policy and find the most restrictive set
		var mostOpenToolSet map[string]bool
		for routeName, policyToolInfos := range toolSetsByRoute {
			if len(policyToolInfos) == 0 {
				continue
			}
			
			// Initialize route map in ToolsByPolicy
			view.ToolsByPolicy[routeName] = make(map[string][]string)
			
			// Find the route resource to get its actual paths
			var routeResource *HTTPRouteResource
			for _, route := range view.RelatedRoutes {
				if route.Name == routeName {
					for i := range state.HTTPRoutes {
						if state.HTTPRoutes[i].Name == routeName {
							routeResource = &state.HTTPRoutes[i]
							break
						}
					}
					break
				}
			}
			
			// Store tools for each policy in this route
			var minSet map[string]bool
			for i, pti := range policyToolInfos {
				var toolList []string
				for tool := range pti.tools {
					toolList = append(toolList, tool)
				}
				view.ToolsByPolicy[routeName][pti.name] = toolList
				
				// Map tools to actual HTTPRoute path (if available)
				// Each policy corresponds to one rule in the HTTPRoute in order
				if routeResource != nil && i < len(routeResource.Paths) {
					pathValue := routeResource.Paths[i]
					view.PathTools[pathValue] = toolList
				}
				
				// Track the most restrictive (smallest) set for this route
				if minSet == nil || len(pti.tools) < len(minSet) {
					minSet = pti.tools
				}
			}
			
			// Store tools for this route (the most restrictive set)
			var routeTools []string
			for tool := range minSet {
				routeTools = append(routeTools, tool)
			}
			view.ToolsByRoute[routeName] = routeTools
			
			// Take the largest of all the "most restrictive" sets
			// (represents the least restrictive route)
			if mostOpenToolSet == nil || len(minSet) > len(mostOpenToolSet) {
				mostOpenToolSet = minSet
			}
		}
		
		var effective []string
		if mostOpenToolSet != nil {
			for tool := range mostOpenToolSet {
				effective = append(effective, tool)
			}
		} else {
			// Fallback: deduplicate union if no clear route mapping
			seen := make(map[string]bool)
			for _, t := range view.EffectiveToolNames {
				if !seen[t] {
					seen[t] = true
					effective = append(effective, t)
				}
			}
		}
		
		view.EffectiveToolNames = effective
		view.EffectiveToolCount = len(effective)
	} else {
		// No policy restriction — effective = total discovered
		view.EffectiveToolCount = view.ToolCount
		view.EffectiveToolNames = view.ToolNames
	}

	// --- Find related Agents ---
	for _, agent := range state.KagentAgents {
		for _, tool := range agent.Tools {
			if tool.Name == view.Name {
				view.RelatedAgents = append(view.RelatedAgents, RelatedResource{
					Kind:      "Agent",
					Name:      agent.Name,
					Namespace: agent.Namespace,
					Status:    mcpAgentStatus(agent),
					Details: map[string]interface{}{
						"type":  agent.Type,
						"ready": agent.Ready,
						"tools": tool.ToolNames,
					},
				})
				// Only populate tool info from agents if the server itself didn't have discoveredTools
				if view.ToolCount == 0 && len(tool.ToolNames) > 0 {
					view.ToolNames = append(view.ToolNames, tool.ToolNames...)
					view.ToolCount = len(view.ToolNames)
				}
			}
		}
	}

	// --- Find related Services ---
	for _, svc := range state.Services {
		if svc.Name == view.Name && svc.Namespace == view.Namespace {
			view.RelatedServices = append(view.RelatedServices, RelatedResource{
				Kind:      "Service",
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Status:    "healthy",
				Details: map[string]interface{}{
					"appProtocol": svc.AppProtocol,
					"isMCP":       svc.IsMCP,
					"ports":       svc.Ports,
				},
			})
		}
	}

	// Also check exposure via URL for RemoteMCPServers
	if view.Source == "KagentRemoteMCPServer" && !view.RoutedThroughGateway {
		for _, gw := range state.Gateways {
			if gw.GatewayClassName == "agentgateway" {
				for _, svc := range state.Services {
					if (svc.Name == "agentgateway" || svc.Name == gw.Name) && containsHost(view.URL, svc.Name, svc.Namespace) {
						view.RoutedThroughGateway = true
					}
				}
			}
		}
	}

	// Ensure nil slices become empty arrays in JSON
	ensureNonNilSlices(view)

	// --- Collect findings for this MCP server ---
	view.Findings = collectMCPServerFindings(view, findings)
	if view.Findings == nil {
		view.Findings = []Finding{}
	}

	// --- Score this MCP server ---
	scoreMCPServer(view, policy)
}

// ensureNonNilSlices makes sure all slice fields are non-nil (for clean JSON encoding).
func ensureNonNilSlices(view *MCPServerView) {
	if view.RelatedBackends == nil {
		view.RelatedBackends = []RelatedResource{}
	}
	if view.RelatedPolicies == nil {
		view.RelatedPolicies = []RelatedResource{}
	}
	if view.RelatedRoutes == nil {
		view.RelatedRoutes = []RelatedResource{}
	}
	if view.RelatedGateways == nil {
		view.RelatedGateways = []RelatedResource{}
	}
	if view.RelatedAgents == nil {
		view.RelatedAgents = []RelatedResource{}
	}
	if view.RelatedServices == nil {
		view.RelatedServices = []RelatedResource{}
	}
	if view.ToolNames == nil {
		view.ToolNames = []string{}
	}
	if view.EffectiveToolNames == nil {
		view.EffectiveToolNames = []string{}
	}
}

// collectMCPServerFindings gathers findings relevant to this MCP server.
func collectMCPServerFindings(view *MCPServerView, allFindings []Finding) []Finding {
	var result []Finding

	// Collect all resource refs that belong to this MCP server
	refs := map[string]bool{}
	refs[view.ID] = true
	// Alternate refs
	switch view.Source {
	case "KagentMCPServer":
		refs[fmt.Sprintf("MCPServer/%s/%s", view.Namespace, view.Name)] = true
	case "KagentRemoteMCPServer":
		refs[fmt.Sprintf("RemoteMCPServer/%s/%s", view.Namespace, view.Name)] = true
	}
	for _, r := range view.RelatedBackends {
		refs[fmt.Sprintf("AgentgatewayBackend/%s/%s", r.Namespace, r.Name)] = true
	}
	for _, r := range view.RelatedRoutes {
		refs[fmt.Sprintf("HTTPRoute/%s/%s", r.Namespace, r.Name)] = true
	}
	for _, r := range view.RelatedGateways {
		refs[fmt.Sprintf("Gateway/%s/%s", r.Namespace, r.Name)] = true
	}
	for _, r := range view.RelatedPolicies {
		refs[fmt.Sprintf("AgentgatewayPolicy/%s/%s", r.Namespace, r.Name)] = true
	}
	for _, r := range view.RelatedAgents {
		refs[fmt.Sprintf("Agent/%s/%s", r.Namespace, r.Name)] = true
	}

	for _, f := range allFindings {
		// If this MCP server has tool restriction via policy, suppress the raw TOOLS-001 finding
		// since the effective tool count (after policy enforcement) is what matters
		if view.HasToolRestriction && strings.HasPrefix(f.ID, "TOOLS-001-") && strings.Contains(f.ID, view.Name) {
			continue
		}

		// Suppress EXP-001 if the MCP server is actually routed through gateway
		// (the cluster-level evaluator checks URL-based routing, but correlation
		// may have detected routing via backend/gateway/route association)
		if view.RoutedThroughGateway && strings.HasPrefix(f.ID, "EXP-001-") && strings.Contains(f.ID, view.Name) {
			continue
		}

		// Suppress AUTH-100 (MCP-level auth) if the MCP server has JWT auth from policy
		// JWT at the gateway/policy level provides transport-level auth which is sufficient
		if view.HasJWT && strings.HasPrefix(f.ID, "AUTH-100-") && strings.Contains(f.ID, view.Name) {
			continue
		}

		// Suppress RBAC-100 (MCP-level RBAC) if the MCP server has RBAC from policy
		// Policy-level authorization with CEL expressions provides tool-level access control
		if view.HasRBAC && strings.HasPrefix(f.ID, "RBAC-100-") && strings.Contains(f.ID, view.Name) {
			continue
		}

		// Match by resource ref
		if f.ResourceRef != "" && refs[f.ResourceRef] {
			result = append(result, f)
			continue
		}
		// Match by name in finding ID (e.g., "AGW-100-my-mcp", "EXP-001-my-mcp")
		if strings.Contains(f.ID, view.Name) {
			result = append(result, f)
			continue
		}
		// Cluster-wide findings (no resource ref) that affect all MCP servers
		if f.ResourceRef == "" && isClusterWideFinding(f) {
			result = append(result, f)
		}
	}

	return result
}

func isClusterWideFinding(f Finding) bool {
	switch f.ID {
	case "AGW-001", "AGW-003", "AGW-004", "AUTH-002", "CORS-001", "CORS-002",
		"CORS-003", "RL-001", "RL-002", "RBAC-001", "RBAC-002", "PG-001", "PG-002",
		"TLS-002":
		return true
	}
	return false
}

// scoreMCPServer calculates the governance score for a single MCP server.
func scoreMCPServer(view *MCPServerView, policy Policy) {
	bd := MCPServerScoreBreakdown{
		GatewayRouting: 100,
		Authentication: 100,
		Authorization:  100,
		TLS:            100,
		CORS:           100,
		RateLimit:      100,
		PromptGuard:    100,
		ToolScope:      100,
	}

	// Gateway routing
	if policy.RequireAgentGateway {
		if !view.RoutedThroughGateway {
			bd.GatewayRouting = 0
		} else if len(view.RelatedBackends) == 0 {
			bd.GatewayRouting = 30
		}
	}

	// Authentication
	if policy.RequireJWTAuth {
		if !view.HasJWT && !view.HasAuth {
			bd.Authentication = 0
		} else if view.HasJWT && view.JWTMode == "Optional" {
			bd.Authentication = 50
		} else if view.HasAuth && !view.HasJWT {
			bd.Authentication = 70
		}
	}

	// Authorization
	if policy.RequireRBAC {
		if !view.HasRBAC {
			bd.Authorization = 0
		}
	}

	// TLS
	if policy.RequireTLS {
		if !view.HasTLS {
			bd.TLS = 0
		}
	}

	// CORS
	if policy.RequireCORS {
		if !view.HasCORS {
			bd.CORS = 0
		}
	}

	// Rate Limit
	// Score 100 only if configured, otherwise 0 (feature not deployed)
	// If not required by policy, it still counts toward weighted score but at 0
	if !view.HasRateLimit {
		bd.RateLimit = 0
	}

	// Prompt Guard
	// Score 100 only if configured, otherwise 0 (feature not deployed)
	// If not required by policy, it still counts toward weighted score but at 0
	if !view.HasPromptGuard {
		bd.PromptGuard = 0
	}

	// Tool Scope - score based on effective tool count (after policy restrictions)
	// An MCP server with 0 tools is not properly configured
	if view.ToolCount == 0 {
		bd.ToolScope = 0
		// Add a finding for 0-tools
		view.Findings = append(view.Findings, Finding{
			ID:          fmt.Sprintf("TOOLS-000-%s", view.Name),
			Severity:    SeverityHigh,
			Category:    "Tool Governance",
			Title:       fmt.Sprintf("MCP server %s has no tools", view.Name),
			Description: fmt.Sprintf("The MCP server '%s' in namespace '%s' has 0 tools discovered. Tools should be attached to the MCP server for proper governance.", view.Name, view.Namespace),
			Impact:      "Without tools, the MCP server cannot serve AI agents, and tool-level governance cannot be applied.",
			Remediation: "Ensure the MCP server exposes tools and that tool discovery is working correctly. Verify the MCP server spec.tools or spec.toolsets configuration.",
			ResourceRef: view.ID,
			Namespace:   view.Namespace,
		})
	} else if policy.MaxToolsCritical > 0 && view.EffectiveToolCount > policy.MaxToolsCritical {
		bd.ToolScope = 0
	} else if policy.MaxToolsWarning > 0 && view.EffectiveToolCount > policy.MaxToolsWarning {
		bd.ToolScope = 50
	}

	view.ScoreBreakdown = bd

	// Weighted average using policy weights
	w := policy.Weights
	totalWeight := 0
	weightedScore := 0

	type entry struct {
		score  int
		weight int
		req    bool
	}
	entries := []entry{
		{bd.GatewayRouting, w.AgentGatewayIntegration, policy.RequireAgentGateway},
		{bd.Authentication, w.Authentication, policy.RequireJWTAuth},
		{bd.Authorization, w.Authorization, policy.RequireRBAC},
		{bd.TLS, w.TLSEncryption, policy.RequireTLS},
		{bd.CORS, w.CORSPolicy, policy.RequireCORS},
		{bd.RateLimit, w.RateLimit, policy.RequireRateLimit},
		{bd.PromptGuard, w.PromptGuard, policy.RequirePromptGuard},
		{bd.ToolScope, w.ToolScope, policy.MaxToolsWarning > 0 || policy.MaxToolsCritical > 0 || view.ToolCount == 0},
	}

	for _, e := range entries {
		if e.req {
			totalWeight += e.weight
			weightedScore += e.score * e.weight
		}
	}

	if totalWeight > 0 {
		view.Score = weightedScore / totalWeight
	} else {
		view.Score = 100
	}

	// Grade and status
	switch {
	case view.Score >= 90:
		view.Grade = "A"
		view.Status = "compliant"
	case view.Score >= 70:
		view.Grade = "B"
		view.Status = "warning"
	case view.Score >= 50:
		view.Grade = "C"
		view.Status = "failing"
	case view.Score >= 30:
		view.Grade = "D"
		view.Status = "failing"
	default:
		view.Grade = "F"
		view.Status = "critical"
	}

	// Override status based on findings severity
	for _, f := range view.Findings {
		if f.Severity == SeverityCritical {
			view.Status = "critical"
			break
		}
	}

	// Build score explanations
	view.ScoreExplanations = buildScoreExplanations(view, policy)
}

// ComputeSuppressedFindingIDs returns the set of finding IDs that were suppressed
// during MCP-server-centric correlation. This lets callers filter the raw findings
// list so that the Findings tab / Resource Inventory stay consistent with the
// MCP Server views.
func ComputeSuppressedFindingIDs(views []MCPServerView, allFindings []Finding) map[string]bool {
	suppressed := map[string]bool{}
	for _, view := range views {
		for _, f := range allFindings {
			if view.HasToolRestriction && strings.HasPrefix(f.ID, "TOOLS-001-") && strings.Contains(f.ID, view.Name) {
				suppressed[f.ID] = true
			}
			if view.RoutedThroughGateway && strings.HasPrefix(f.ID, "EXP-001-") && strings.Contains(f.ID, view.Name) {
				suppressed[f.ID] = true
			}
			if view.HasJWT && strings.HasPrefix(f.ID, "AUTH-100-") && strings.Contains(f.ID, view.Name) {
				suppressed[f.ID] = true
			}
			if view.HasRBAC && strings.HasPrefix(f.ID, "RBAC-100-") && strings.Contains(f.ID, view.Name) {
				suppressed[f.ID] = true
			}
		}
	}
	return suppressed
}

// FilterFindings removes suppressed findings from a raw findings slice.
func FilterFindings(findings []Finding, suppressed map[string]bool) []Finding {
	if len(suppressed) == 0 {
		return findings
	}
	var filtered []Finding
	for _, f := range findings {
		if !suppressed[f.ID] {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// BuildMCPServerSummary creates a cluster-level summary from MCP server views.
func BuildMCPServerSummary(views []MCPServerView) MCPServerSummary {
	summary := MCPServerSummary{
		TotalMCPServers: len(views),
	}
	totalScore := 0
	for _, v := range views {
		totalScore += v.Score
		if v.RoutedThroughGateway {
			summary.RoutedServers++
		} else {
			summary.UnroutedServers++
		}
		if v.Score >= 70 {
			summary.SecuredServers++
		} else {
			summary.AtRiskServers++
		}
		if v.Score < 30 {
			summary.CriticalServers++
		}
		summary.TotalTools += v.ToolCount
		summary.ExposedTools += v.EffectiveToolCount
	}
	if len(views) > 0 {
		summary.AverageScore = totalScore / len(views)
	}
	return summary
}

// --- Helper functions ---

// buildScoreExplanations produces a per-category explanation of how the score was calculated.
func buildScoreExplanations(view *MCPServerView, policy Policy) []ScoreExplanation {
	bd := view.ScoreBreakdown
	var explanations []ScoreExplanation

	// Helper to find source resources
	policyNames := func() []string {
		var names []string
		for _, p := range view.RelatedPolicies {
			names = append(names, fmt.Sprintf("%s/%s", p.Kind, p.Name))
		}
		return names
	}
	backendNames := func() []string {
		var names []string
		for _, b := range view.RelatedBackends {
			names = append(names, fmt.Sprintf("%s/%s", b.Kind, b.Name))
		}
		return names
	}
	routeNames := func() []string {
		var names []string
		for _, r := range view.RelatedRoutes {
			names = append(names, fmt.Sprintf("%s/%s", r.Kind, r.Name))
		}
		return names
	}
	gatewayNames := func() []string {
		var names []string
		for _, g := range view.RelatedGateways {
			names = append(names, fmt.Sprintf("%s/%s", g.Kind, g.Name))
		}
		return names
	}

	statusFor := func(score int) string {
		if score >= 100 {
			return "pass"
		}
		if score > 0 {
			return "partial"
		}
		return "fail"
	}

	// 1. Gateway Routing
	{
		exp := ScoreExplanation{
			Category: "Gateway Routing",
			Score:    bd.GatewayRouting,
			MaxScore: 100,
		}
		if !policy.RequireAgentGateway {
			exp.Status = "not-required"
			exp.Reasons = []string{"Gateway routing is not required by the governance policy."}
		} else {
			exp.Status = statusFor(bd.GatewayRouting)
			if view.RoutedThroughGateway {
				exp.Reasons = append(exp.Reasons, "MCP server is routed through agentgateway.")
				exp.Sources = append(exp.Sources, gatewayNames()...)
				exp.Sources = append(exp.Sources, routeNames()...)
				if len(view.RelatedBackends) > 0 {
					exp.Reasons = append(exp.Reasons, "AgentgatewayBackend provides proxy configuration.")
					exp.Sources = append(exp.Sources, backendNames()...)
				} else {
					exp.Suggestions = append(exp.Suggestions, "Create an AgentgatewayBackend for full proxy control.")
				}
			} else {
				exp.Reasons = append(exp.Reasons, "MCP server is NOT routed through agentgateway.")
				exp.Suggestions = append(exp.Suggestions, "Create a Gateway (agentgateway class), AgentgatewayBackend, and HTTPRoute to proxy traffic through agentgateway.")
			}
		}
		explanations = append(explanations, exp)
	}

	// 2. Authentication
	{
		exp := ScoreExplanation{
			Category: "Authentication",
			Score:    bd.Authentication,
			MaxScore: 100,
		}
		if !policy.RequireJWTAuth {
			exp.Status = "not-required"
			exp.Reasons = []string{"JWT authentication is not required by the governance policy."}
		} else {
			exp.Status = statusFor(bd.Authentication)
			if view.HasJWT {
				mode := view.JWTMode
				if mode == "" {
					mode = "Strict"
				}
				exp.Reasons = append(exp.Reasons, fmt.Sprintf("JWT authentication is enabled in %s mode.", mode))
				exp.Sources = policyNames()
				if mode == "Optional" {
					exp.Suggestions = append(exp.Suggestions, "Switch JWT mode from Optional to Strict for full enforcement.")
				}
			} else if view.HasAuth {
				exp.Reasons = append(exp.Reasons, "Backend-level authentication is configured but not JWT.")
				exp.Sources = backendNames()
				exp.Suggestions = append(exp.Suggestions, "Add JWT authentication via an AgentgatewayPolicy with traffic.jwtAuthentication for stronger auth.")
			} else {
				exp.Reasons = append(exp.Reasons, "No authentication is configured.")
				exp.Suggestions = append(exp.Suggestions, "Create an AgentgatewayPolicy with traffic.jwtAuthentication targeting your Gateway or HTTPRoute.")
			}
		}
		explanations = append(explanations, exp)
	}

	// 3. Authorization
	{
		exp := ScoreExplanation{
			Category: "Authorization",
			Score:    bd.Authorization,
			MaxScore: 100,
		}
		if !policy.RequireRBAC {
			exp.Status = "not-required"
			exp.Reasons = []string{"RBAC authorization is not required by the governance policy."}
		} else {
			exp.Status = statusFor(bd.Authorization)
			if view.HasRBAC {
				exp.Reasons = append(exp.Reasons, "RBAC/CEL-based authorization is enabled.")
				exp.Sources = policyNames()
				if view.HasToolRestriction {
					exp.Reasons = append(exp.Reasons, fmt.Sprintf("Tool access is restricted to %d of %d tools via CEL policy.", view.EffectiveToolCount, view.ToolCount))
				}
			} else {
				exp.Reasons = append(exp.Reasons, "No RBAC authorization is configured.")
				exp.Suggestions = append(exp.Suggestions, "Add an AgentgatewayPolicy with traffic.authorization using CEL matchExpressions for tool-level access control.")
			}
		}
		explanations = append(explanations, exp)
	}

	// 4. TLS
	{
		exp := ScoreExplanation{
			Category: "TLS Encryption",
			Score:    bd.TLS,
			MaxScore: 100,
		}
		if !policy.RequireTLS {
			exp.Status = "not-required"
			exp.Reasons = []string{"TLS encryption is not required by the governance policy."}
		} else {
			exp.Status = statusFor(bd.TLS)
			if view.HasTLS {
				exp.Reasons = append(exp.Reasons, "TLS is enabled on the backend connection.")
				exp.Sources = backendNames()
			} else {
				exp.Reasons = append(exp.Reasons, "No TLS encryption is configured.")
				exp.Suggestions = append(exp.Suggestions, "Add spec.policies.tls with an SNI to the AgentgatewayBackend for encrypted backend connections.")
			}
		}
		explanations = append(explanations, exp)
	}

	// 5. CORS
	{
		exp := ScoreExplanation{
			Category: "CORS Policy",
			Score:    bd.CORS,
			MaxScore: 100,
		}
		if !policy.RequireCORS {
			exp.Status = "not-required"
			exp.Reasons = []string{"CORS policy is not required by the governance policy."}
		} else {
			exp.Status = statusFor(bd.CORS)
			if view.HasCORS {
				// Determine source of CORS
				fromPolicy := false
				fromRoute := false
				for _, p := range view.RelatedPolicies {
					if hasCORS, ok := p.Details["hasCORS"].(bool); ok && hasCORS {
						fromPolicy = true
					}
				}
				for _, r := range view.RelatedRoutes {
					if hasCORS, ok := r.Details["hasCORSFilter"].(bool); ok && hasCORS {
						fromRoute = true
					}
				}
				if fromPolicy && fromRoute {
					exp.Reasons = append(exp.Reasons, "CORS is configured at both the AgentgatewayPolicy and HTTPRoute levels.")
				} else if fromPolicy {
					exp.Reasons = append(exp.Reasons, "CORS is configured via the AgentgatewayPolicy (traffic.cors).")
					exp.Sources = policyNames()
				} else if fromRoute {
					exp.Reasons = append(exp.Reasons, "CORS is configured via the HTTPRoute CORS filter.")
					exp.Sources = routeNames()
				} else {
					exp.Reasons = append(exp.Reasons, "CORS is enabled.")
				}
			} else {
				exp.Reasons = append(exp.Reasons, "No CORS policy is configured.")
				exp.Suggestions = append(exp.Suggestions, "Add traffic.cors to an AgentgatewayPolicy or add a CORS filter to the HTTPRoute.")
			}
		}
		explanations = append(explanations, exp)
	}

	// 6. Rate Limit
	{
		exp := ScoreExplanation{
			Category: "Rate Limiting",
			Score:    bd.RateLimit,
			MaxScore: 100,
		}
		if !policy.RequireRateLimit {
			exp.Status = "not-required"
			exp.Reasons = []string{"Rate limiting is not required by the governance policy."}
		} else {
			exp.Status = statusFor(bd.RateLimit)
			if view.HasRateLimit {
				exp.Reasons = append(exp.Reasons, "Rate limiting is enabled via AgentgatewayPolicy.")
				exp.Sources = policyNames()
			} else {
				exp.Reasons = append(exp.Reasons, "No rate limiting is configured.")
				exp.Suggestions = append(exp.Suggestions, "Add traffic.rateLimit.local to an AgentgatewayPolicy to enforce request rate limits.")
			}
		}
		explanations = append(explanations, exp)
	}

	// 7. Prompt Guard
	{
		exp := ScoreExplanation{
			Category: "Prompt Guard",
			Score:    bd.PromptGuard,
			MaxScore: 100,
		}
		if !policy.RequirePromptGuard {
			exp.Status = "not-required"
			exp.Reasons = []string{"Prompt guard is not required by the governance policy."}
		} else {
			exp.Status = statusFor(bd.PromptGuard)
			if view.HasPromptGuard {
				exp.Reasons = append(exp.Reasons, "Prompt guard is enabled with request/response inspection.")
				exp.Sources = policyNames()
			} else {
				exp.Reasons = append(exp.Reasons, "No prompt guard is configured.")
				exp.Suggestions = append(exp.Suggestions, "Add backend.ai.promptGuard to an AgentgatewayPolicy with regex reject/mask patterns for injection protection.")
			}
		}
		explanations = append(explanations, exp)
	}

	// 8. Tool Scope
	{
		exp := ScoreExplanation{
			Category: "Tool Scope",
			Score:    bd.ToolScope,
			MaxScore: 100,
		}
		hasToolPolicy := policy.MaxToolsWarning > 0 || policy.MaxToolsCritical > 0
		if !hasToolPolicy {
			exp.Status = "not-required"
			exp.Reasons = []string{"Tool scope limits are not configured in the governance policy."}
		} else if view.ToolCount == 0 {
			exp.Status = "fail"
			exp.Reasons = []string{"No tools discovered for this MCP server."}
			exp.Suggestions = []string{"Ensure the MCP server exposes tools and that tool discovery is working correctly."}
		} else {
			exp.Status = statusFor(bd.ToolScope)
			if view.HasToolRestriction {
				exp.Reasons = append(exp.Reasons, fmt.Sprintf("Tool access restricted to %d tools (out of %d discovered) via CEL authorization policy.", view.EffectiveToolCount, view.ToolCount))
				exp.Sources = policyNames()
			}
			if bd.ToolScope >= 100 {
				exp.Reasons = append(exp.Reasons, fmt.Sprintf("Effective tool count (%d) is within governance limits (warning: %d, critical: %d).", view.EffectiveToolCount, policy.MaxToolsWarning, policy.MaxToolsCritical))
			} else if bd.ToolScope >= 50 {
				exp.Reasons = append(exp.Reasons, fmt.Sprintf("Effective tool count (%d) exceeds warning threshold (%d).", view.EffectiveToolCount, policy.MaxToolsWarning))
				exp.Suggestions = append(exp.Suggestions, "Reduce the number of exposed tools by adding stricter CEL authorization rules.")
			} else {
				exp.Reasons = append(exp.Reasons, fmt.Sprintf("Effective tool count (%d) exceeds critical threshold (%d).", view.EffectiveToolCount, policy.MaxToolsCritical))
				exp.Suggestions = append(exp.Suggestions, "Urgently restrict tool exposure via CEL authorization policies.")
			}
		}
		explanations = append(explanations, exp)
	}

	return explanations
}

func matchesMCPTarget(view *MCPServerView, target MCPTargetInfo) bool {
	if target.Name == view.Name {
		return true
	}
	expectedHost := fmt.Sprintf("%s.%s.svc.cluster.local", view.Name, view.Namespace)
	if target.Host == expectedHost || target.Host == view.Name {
		return true
	}
	// For RemoteMCPServers: check if the target host appears in the server's URL
	// e.g. URL "http://kagent-tools.kagent:8084/mcp" and host "kagent-tools.kagent.svc.cluster.local"
	if view.URL != "" && target.Host != "" {
		// Extract the short hostname (before .svc.cluster.local)
		shortHost := strings.Split(target.Host, ".svc.cluster.local")[0]
		if shortHost != "" && strings.Contains(view.URL, shortHost) {
			return true
		}
	}
	return false
}

func isCoveredByExistingView(views []MCPServerView, target MCPTargetInfo) bool {
	for _, v := range views {
		if v.Name == target.Name {
			return true
		}
		expectedHost := fmt.Sprintf("%s.%s.svc.cluster.local", v.Name, v.Namespace)
		if target.Host == expectedHost {
			return true
		}
	}
	return false
}

func isCoveredByExistingViewForService(views []MCPServerView, svc ServiceResource) bool {
	for _, v := range views {
		if v.Name == svc.Name && v.Namespace == svc.Namespace {
			return true
		}
	}
	return false
}

func mcpBackendStatus(b AgentgatewayBackendResource) string {
	if b.HasTLS {
		return "healthy"
	}
	return "warning"
}

func mcpRouteStatus(r HTTPRouteResource) string {
	// Route status is "healthy" — individual CORS is checked at the view level
	// (may come from the route filter or from the AgentgatewayPolicy)
	return "healthy"
}

func mcpGatewayStatus(gw GatewayResource) string {
	if gw.Programmed && gw.GatewayClassName == "agentgateway" {
		return "healthy"
	}
	if !gw.Programmed {
		return "critical"
	}
	return "warning"
}

func mcpPolicyStatus(p AgentgatewayPolicyResource) string {
	if p.HasJWT && p.HasRBAC && p.HasCORS && p.HasRateLimit {
		return "healthy"
	}
	if p.HasJWT {
		return "warning"
	}
	return "critical"
}

func mcpAgentStatus(a KagentAgentResource) string {
	if a.Ready {
		return "healthy"
	}
	return "warning"
}

