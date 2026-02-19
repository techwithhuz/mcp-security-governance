package inventory

import (
	"fmt"
	"strings"
	"time"
)

// ScoreCatalog evaluates a single MCPServerCatalog resource and returns its VerifiedScore.
// The scoring model checks 5 governance categories:
//   1. Publisher Verification — source tracking, environment labels, management type
//   2. Transport Security    — HTTPS/TLS for remote endpoints, transport type
//   3. Deployment Health     — published, deployment ready, versioned
//   4. Tool Scope            — number of tools exposed, blast radius
//   5. Usage & Integration   — used by agents, source reference present
func ScoreCatalog(res *VerifiedResource, policy ScoringPolicy) VerifiedScore {
	var checks []VerifiedCheck
	var findings []VerifiedFinding

	// ======================================================================
	// Category 1: Publisher Verification (max 30 points)
	// ======================================================================
	checks = append(checks, checkPublisherSource(res, policy))
	checks = append(checks, checkEnvironmentLabels(res, policy))
	checks = append(checks, checkManagementType(res, policy))

	// ======================================================================
	// Category 2: Transport Security (max 25 points)
	// ======================================================================
	checks = append(checks, checkTransportSecurity(res, policy))
	checks = append(checks, checkRemoteEndpointTLS(res, policy))

	// ======================================================================
	// Category 3: Deployment Health (max 20 points)
	// ======================================================================
	checks = append(checks, checkPublished(res, policy))
	checks = append(checks, checkDeploymentReady(res, policy))
	checks = append(checks, checkVersioning(res, policy))

	// ======================================================================
	// Category 4: Tool Scope (max 15 points)
	// ======================================================================
	checks = append(checks, checkToolCount(res, policy))

	// ======================================================================
	// Category 5: Usage & Integration (max 10 points)
	// ======================================================================
	checks = append(checks, checkAgentUsage(res, policy))

	// Tally score
	totalScore := 0
	maxScore := 0
	passed := 0
	for _, c := range checks {
		totalScore += c.Score
		maxScore += c.MaxScore
		if c.Passed {
			passed++
		}
	}

	// Normalize to 0–100
	normalizedScore := 0
	if maxScore > 0 {
		normalizedScore = (totalScore * 100) / maxScore
	}
	if normalizedScore > 100 {
		normalizedScore = 100
	}

	// Generate findings from failed checks
	for _, c := range checks {
		if !c.Passed {
			sev := severityForCheck(c)
			findings = append(findings, VerifiedFinding{
				Severity:    sev,
				Category:    c.Category,
				Title:       c.Name + " — " + c.Detail,
				Description: c.Description,
				Remediation: remediationForCheck(c),
			})
		}
	}

	// Compute composite category scores (each normalized to 0–100) and raw max scores
	securityScore, trustScore, complianceScore, orgScore, publisherScore := computeCategoryScores(checks)
	// Note: Each category score is normalized to 0-100 for consistent comparison

	// Derive org/publisher names from resource labels
	verifiedOrg := res.Environment   // best-effort: use environment label as org
	verifiedPublisher := res.SourceName
	if res.SourceKind != "" {
		verifiedPublisher = res.SourceKind + "/" + res.SourceName
	}

	// Build human-readable reason
	reason := buildReason(normalizedScore, securityScore, trustScore, complianceScore, len(findings), policy)

	return VerifiedScore{
		Score:         normalizedScore,
		Grade:         GradeFromScore(normalizedScore),
		Status:        StatusFromScoreWithThresholds(normalizedScore, policyVerifiedThreshold(policy), policyUnverifiedThreshold(policy)),
		Checks:        checks,
		Findings:      findings,
		ChecksPassed:  passed,
		ChecksTotal:   len(checks),
		LastEvaluated: time.Now(),

		SecurityScore:   securityScore,
		TrustScore:      trustScore,
		ComplianceScore: complianceScore,
		OrgScore:        orgScore,
		PublisherScore:  publisherScore,
		VerifiedOrg:     verifiedOrg,
		VerifiedPublisher: verifiedPublisher,
		Reason:          reason,
	}
}

// computeCategoryScores normalises each governance category to 0–100.
func computeCategoryScores(checks []VerifiedCheck) (security, trust, compliance, org, publisher int) {
	type cat struct{ earned, max int }
	cats := map[string]*cat{
		"transport":  {},
		"deployment": {},
		"publisher":  {},
		"toolScope":  {},
		"usage":      {},
	}
	for _, c := range checks {
		bucket, ok := cats[c.Category]
		if !ok {
			bucket = &cat{}
			cats[c.Category] = bucket
		}
		bucket.earned += c.Score
		bucket.max += c.MaxScore
	}

	norm := func(c *cat) int {
		if c == nil || c.max == 0 {
			return 0
		}
		v := (c.earned * 100) / c.max
		if v > 100 {
			return 100
		}
		return v
	}

	// Security = transport + deployment
	secEarned := cats["transport"].earned + cats["deployment"].earned
	secMax := cats["transport"].max + cats["deployment"].max
	if secMax > 0 {
		security = (secEarned * 100) / secMax
	}

	// Trust = publisher category
	trust = norm(cats["publisher"])

	// Compliance = toolScope + usage (metadata & integration completeness)
	compEarned := cats["toolScope"].earned + cats["usage"].earned
	compMax := cats["toolScope"].max + cats["usage"].max
	if compMax > 0 {
		compliance = (compEarned * 100) / compMax
	}

	// Org score = PUB-002 (environment labels) normalised to 0-100
	// Publisher score = PUB-001 + PUB-003 normalised to 0-100
	orgEarned, orgMax := 0, 0
	pubEarned, pubMax := 0, 0
	for _, c := range checks {
		if c.Category != "publisher" {
			continue
		}
		switch c.ID {
		case "PUB-002":
			orgEarned += c.Score
			orgMax += c.MaxScore
		default: // PUB-001, PUB-003
			pubEarned += c.Score
			pubMax += c.MaxScore
		}
	}
	if orgMax > 0 {
		org = (orgEarned * 100) / orgMax
	}
	if pubMax > 0 {
		publisher = (pubEarned * 100) / pubMax
	}
	return
}

// buildReason produces a human-readable verification summary.
func buildReason(overall, security, trust, compliance, findingCount int, policy ScoringPolicy) string {
	status := StatusFromScoreWithThresholds(overall, policyVerifiedThreshold(policy), policyUnverifiedThreshold(policy))
	var parts []string
	parts = append(parts, fmt.Sprintf("Overall score %d/100 (%s)", overall, status))
	if security < 50 {
		parts = append(parts, fmt.Sprintf("Security needs attention (%d%%)", security))
	}
	if trust < 50 {
		parts = append(parts, fmt.Sprintf("Trust verification incomplete (%d%%)", trust))
	}
	if compliance < 50 {
		parts = append(parts, fmt.Sprintf("Compliance gaps detected (%d%%)", compliance))
	}
	if findingCount > 0 {
		parts = append(parts, fmt.Sprintf("%d finding(s) require review", findingCount))
	}
	if findingCount == 0 && overall >= 70 {
		parts = append(parts, "All governance checks passed")
	}
	return strings.Join(parts, ". ") + "."
}

// ScoringPolicy configures scoring thresholds and category weights for Verified Catalog scoring.
// Users can customise these values via the MCPGovernancePolicy CRD's verifiedCatalogScoring section.
type ScoringPolicy struct {
	MaxToolsWarning  int // tools count above this = Medium finding (default: 10)
	MaxToolsCritical int // tools count above this = Critical finding (default: 20)

	// Category weights (must sum to 100) — controls how the final weighted score is computed.
	SecurityWeight   int // weight for Security category (transport + deployment), default: 50
	TrustWeight      int // weight for Trust category (publisher verification), default: 30
	ComplianceWeight int // weight for Compliance category (toolScope + usage), default: 20

	// Status thresholds — score boundaries for Verified / Unverified / Rejected.
	VerifiedThreshold   int // score >= this → "Verified"  (default: 70)
	UnverifiedThreshold int // score >= this → "Unverified" (default: 50); below → "Rejected"

	// Per-check max scores (0 = use built-in default).
	CheckMaxScores map[string]int // e.g. {"PUB-001": 10, "SEC-001": 15, ...}
}

// DefaultScoringPolicy returns sensible defaults.
func DefaultScoringPolicy() ScoringPolicy {
	return ScoringPolicy{
		MaxToolsWarning:     10,
		MaxToolsCritical:    20,
		SecurityWeight:      50,
		TrustWeight:         30,
		ComplianceWeight:    20,
		VerifiedThreshold:   70,
		UnverifiedThreshold: 50,
	}
}

// policyVerifiedThreshold returns the verified threshold or the default.
func policyVerifiedThreshold(p ScoringPolicy) int {
	if p.VerifiedThreshold > 0 {
		return p.VerifiedThreshold
	}
	return 70
}

// policyUnverifiedThreshold returns the unverified threshold or the default.
func policyUnverifiedThreshold(p ScoringPolicy) int {
	if p.UnverifiedThreshold > 0 {
		return p.UnverifiedThreshold
	}
	return 50
}

// checkMaxScore returns the per-check max score from policy overrides or falls back to the built-in default.
func checkMaxScore(policy ScoringPolicy, checkID string, builtinDefault int) int {
	if policy.CheckMaxScores != nil {
		if v, ok := policy.CheckMaxScores[checkID]; ok && v > 0 {
			return v
		}
	}
	return builtinDefault
}

// ---------- Individual Checks ----------

func checkPublisherSource(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "PUB-001", 10)
	c := VerifiedCheck{
		ID:       "PUB-001",
		Name:     "Source Kind Tracked",
		Category: "publisher",
		MaxScore: maxSc,
		Description: "MCP server catalog has a known source kind (e.g. MCPServer, RemoteMCPServer) from discovery",
	}
	if res.SourceKind != "" && res.SourceName != "" {
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Source: " + res.SourceKind + "/" + res.SourceNamespace + "/" + res.SourceName
	} else {
		c.Detail = "No source tracking labels found — origin unknown"
	}
	return c
}

func checkEnvironmentLabels(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "PUB-002", 10)
	c := VerifiedCheck{
		ID:       "PUB-002",
		Name:     "Environment Labelled",
		Category: "publisher",
		MaxScore: maxSc,
		Description: "Catalog entry has environment and cluster labels for traceability",
	}
	hasEnv := res.Environment != ""
	hasCluster := res.Cluster != ""
	if hasEnv && hasCluster {
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Environment: " + res.Environment + ", Cluster: " + res.Cluster
	} else if hasEnv || hasCluster {
		c.Passed = true
		c.Score = maxSc / 2
		c.Detail = "Partial labels — missing environment or cluster"
	} else {
		c.Detail = "No environment/cluster labels — cannot trace origin"
	}
	return c
}

func checkManagementType(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "PUB-003", 10)
	c := VerifiedCheck{
		ID:       "PUB-003",
		Name:     "Management Type Set",
		Category: "publisher",
		MaxScore: maxSc,
		Description: "Catalog entry has a management type (external=auto-discovered, managed=registry-managed)",
	}
	if res.ManagementType != "" {
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Management type: " + res.ManagementType
	} else {
		c.Detail = "No management type — lifecycle ownership unclear"
	}
	return c
}

func checkTransportSecurity(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "SEC-001", 15)
	c := VerifiedCheck{
		ID:       "SEC-001",
		Name:     "Transport Type",
		Category: "transport",
		MaxScore: maxSc,
		Description: "MCP server uses a recognized secure transport type",
	}
	t := strings.ToLower(res.Transport)
	switch {
	case t == "streamable-http" || t == "http":
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Transport: " + res.Transport + " (HTTP-based, can be secured with TLS)"
	case t == "stdio":
		c.Passed = true
		c.Score = (maxSc * 2) / 3
		c.Detail = "Transport: stdio (local process, no network exposure)"
	case t == "sse":
		c.Passed = true
		c.Score = (maxSc * 4) / 5
		c.Detail = "Transport: SSE (server-sent events, HTTP-based)"
	case t == "":
		// Check remotes/packages for transport info
		if res.RemoteURL != "" {
			c.Passed = true
			c.Score = (maxSc * 4) / 5
			c.Detail = "Remote endpoint configured (transport inferred from URL)"
		} else if res.PackageImage != "" {
			c.Passed = true
			c.Score = (maxSc * 2) / 3
			c.Detail = "Package image configured (likely stdio transport)"
		} else {
			c.Score = maxSc / 3
			c.Passed = true
			c.Detail = "No explicit transport — may use default"
		}
	default:
		c.Score = maxSc / 3
		c.Passed = true
		c.Detail = "Unknown transport type: " + res.Transport
	}
	return c
}

func checkRemoteEndpointTLS(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "SEC-002", 10)
	c := VerifiedCheck{
		ID:       "SEC-002",
		Name:     "Remote Endpoint TLS",
		Category: "transport",
		MaxScore: maxSc,
		Description: "Remote MCP server endpoints use HTTPS/TLS",
	}
	if res.RemoteURL == "" {
		// Not a remote endpoint — N/A, give full score
		c.Passed = true
		c.Score = maxSc
		c.Detail = "No remote endpoint — local/stdio transport (TLS not applicable)"
		return c
	}
	if strings.HasPrefix(res.RemoteURL, "https://") {
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Remote URL uses HTTPS: " + res.RemoteURL
	} else if strings.HasPrefix(res.RemoteURL, "http://") {
		c.Score = 0
		c.Detail = "Remote URL uses unencrypted HTTP: " + res.RemoteURL
	} else {
		c.Score = maxSc / 2
		c.Passed = true
		c.Detail = "Remote URL scheme unclear: " + res.RemoteURL
	}
	return c
}

func checkPublished(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "DEP-001", 5)
	c := VerifiedCheck{
		ID:       "DEP-001",
		Name:     "Published",
		Category: "deployment",
		MaxScore: maxSc,
		Description: "Catalog entry is published and visible in the registry",
	}
	if res.Published {
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Published and active"
	} else {
		c.Detail = "Not published — not visible in registry"
	}
	return c
}

func checkDeploymentReady(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "DEP-002", 10)
	c := VerifiedCheck{
		ID:       "DEP-002",
		Name:     "Deployment Ready",
		Category: "deployment",
		MaxScore: maxSc,
		Description: "The backing MCP server deployment is healthy and ready",
	}
	if res.DeploymentReady {
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Deployment is ready"
	} else {
		c.Detail = "Deployment is not ready or health unknown"
	}
	return c
}

func checkVersioning(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "DEP-003", 5)
	c := VerifiedCheck{
		ID:       "DEP-003",
		Name:     "Versioned",
		Category: "deployment",
		MaxScore: maxSc,
		Description: "Catalog entry has a meaningful version (not just 'latest')",
	}
	v := strings.TrimSpace(res.Version)
	switch {
	case v == "" || v == "latest" || v == "unknown":
		c.Detail = "Version is '" + v + "' — no semantic versioning"
	case strings.HasPrefix(v, "v") || strings.Contains(v, "."):
		c.Passed = true
		c.Score = maxSc
		c.Detail = "Version: " + v
	default:
		c.Passed = true
		c.Score = (maxSc * 3) / 5
		c.Detail = "Version tag: " + v + " (not semantic)"
	}
	return c
}

func checkToolCount(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "TOOL-001", 15)
	c := VerifiedCheck{
		ID:       "TOOL-001",
		Name:     "Tool Scope",
		Category: "toolScope",
		MaxScore: maxSc,
		Description: "MCP server exposes a reasonable number of tools (blast radius control)",
	}
	count := res.ToolCount
	if count == 0 && len(res.ToolNames) > 0 {
		count = len(res.ToolNames)
	}

	maxWarn := policy.MaxToolsWarning
	if maxWarn <= 0 {
		maxWarn = 10
	}
	maxCrit := policy.MaxToolsCritical
	if maxCrit <= 0 {
		maxCrit = 20
	}

	switch {
	case count == 0:
		// No tool info — give partial credit
		c.Passed = true
		c.Score = (maxSc * 2) / 3
		c.Detail = "No tool count info — cannot assess blast radius"
	case count <= maxWarn:
		c.Passed = true
		c.Score = maxSc
		c.Detail = fmt.Sprintf("%d tools exposed (within limit of %d)", count, maxWarn)
	case count <= maxCrit:
		c.Passed = true
		c.Score = maxSc / 2
		c.Detail = fmt.Sprintf("%d tools exposed — exceeds warning threshold of %d", count, maxWarn)
	default:
		c.Score = 0
		c.Detail = fmt.Sprintf("%d tools exposed — exceeds critical threshold of %d", count, maxCrit)
	}
	return c
}

func checkAgentUsage(res *VerifiedResource, policy ScoringPolicy) VerifiedCheck {
	maxSc := checkMaxScore(policy, "USE-001", 10)
	c := VerifiedCheck{
		ID:       "USE-001",
		Name:     "Agent Usage",
		Category: "usage",
		MaxScore: maxSc,
		Description: "MCP server is referenced by at least one agent (actively used)",
	}
	n := len(res.UsedByAgents)
	if n > 0 {
		c.Passed = true
		c.Score = maxSc
		c.Detail = fmt.Sprintf("Used by %d agent(s)", n)
	} else {
		c.Score = maxSc / 2
		c.Passed = true
		c.Detail = "Not referenced by any agent — may be orphaned"
	}
	return c
}

// ---------- Helpers ----------

func severityForCheck(c VerifiedCheck) string {
	switch c.Category {
	case "transport":
		if c.Score == 0 {
			return "High"
		}
		return "Medium"
	case "publisher":
		return "Medium"
	case "deployment":
		if c.ID == "DEP-002" {
			return "High"
		}
		return "Low"
	case "toolScope":
		if c.Score == 0 {
			return "Critical"
		}
		return "Medium"
	case "usage":
		return "Low"
	default:
		return "Medium"
	}
}

func remediationForCheck(c VerifiedCheck) string {
	switch c.ID {
	case "PUB-001":
		return "Ensure MCP server is discovered via Agent Registry DiscoveryConfig with proper source labels."
	case "PUB-002":
		return "Add environment and cluster labels to the DiscoveryConfig environment spec."
	case "PUB-003":
		return "The management type is set automatically by the inventory controller. Verify the discovery pipeline."
	case "SEC-001":
		return "Configure the MCP server with a recognized transport type (streamable-http, stdio, sse)."
	case "SEC-002":
		return "Switch remote MCP server endpoint to HTTPS. Update the RemoteMCPServer URL to use TLS."
	case "DEP-001":
		return "Publish the catalog entry by setting status.published to true."
	case "DEP-002":
		return "Ensure the backing deployment is running and healthy. Check pod status and readiness probes."
	case "DEP-003":
		return "Use semantic versioning (e.g. v1.0.0) instead of 'latest' for better governance tracking."
	case "TOOL-001":
		return "Reduce the number of tools exposed by the MCP server. Split into multiple focused servers."
	case "USE-001":
		return "No action required — informational. The MCP server has no agent consumers."
	default:
		return "Review the governance check and address the finding."
	}
}
