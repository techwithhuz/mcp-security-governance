package inventory

import "time"

// VerifiedResource represents an MCP server catalog entry from the Agent Registry
// inventory that has been scored by the governance controller.
type VerifiedResource struct {
	// Identity — from MCPServerCatalog metadata
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	CatalogName     string `json:"catalogName"`     // spec.name (e.g. "kagent/my-mcp-server")
	Title           string `json:"title"`            // spec.title
	Description     string `json:"description"`      // spec.description
	Version         string `json:"version"`          // spec.version

	// Source tracking
	SourceKind      string `json:"sourceKind"`       // label: agentregistry.dev/source-kind
	SourceName      string `json:"sourceName"`       // label: agentregistry.dev/source-name
	SourceNamespace string `json:"sourceNamespace"`  // label: agentregistry.dev/source-namespace
	Environment     string `json:"environment"`      // label: agentregistry.dev/environment
	Cluster         string `json:"cluster"`          // label: agentregistry.dev/cluster

	// Deployment state (from status)
	Published       bool   `json:"published"`
	DeploymentReady bool   `json:"deploymentReady"`
	ManagementType  string `json:"managementType"`   // "external" or "managed"

	// Transport & Packages
	Transport       string   `json:"transport,omitempty"`
	PackageImage    string   `json:"packageImage,omitempty"`
	RemoteURL       string   `json:"remoteURL,omitempty"`
	ToolNames       []string `json:"toolNames,omitempty"`
	ToolCount       int      `json:"toolCount"`

	// Used-by tracking
	UsedByAgents    []AgentUsage `json:"usedByAgents,omitempty"`

	// Verified Score — computed by governance controller
	VerifiedScore   VerifiedScore `json:"verifiedScore"`

	// Lifecycle
	LastScored      time.Time `json:"lastScored"`
	ResourceVersion string    `json:"resourceVersion"` // to detect changes
}

// AgentUsage tracks which agent uses this MCP server catalog.
type AgentUsage struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	ToolNames []string `json:"toolNames,omitempty"`
}

// VerifiedScore is the governance verification result for a single MCPServerCatalog.
type VerifiedScore struct {
	Score           int              `json:"score"`           // 0–100
	Grade           string           `json:"grade"`           // A/B/C/D/F
	Status          string           `json:"status"`          // "Verified", "Unverified", "Rejected", "Pending"
	Checks          []VerifiedCheck  `json:"checks"`          // individual governance checks
	Findings        []VerifiedFinding `json:"findings"`        // issues found
	ChecksPassed    int              `json:"checksPassed"`
	ChecksTotal     int              `json:"checksTotal"`
	LastEvaluated   time.Time        `json:"lastEvaluated"`

	// Composite category scores (0–100 each)
	SecurityScore   int    `json:"securityScore"`   // from transport + deployment checks
	TrustScore      int    `json:"trustScore"`      // from publisher checks + org/publisher verification
	ComplianceScore int    `json:"complianceScore"` // from metadata completeness (version, toolScope, usage)

	// Publisher trust sub-scores (0–100 each)
	OrgScore        int    `json:"orgScore"`        // organization verification score
	PublisherScore  int    `json:"publisherScore"`  // publisher verification score
	VerifiedOrg     string `json:"verifiedOrg,omitempty"`     // org name if present
	VerifiedPublisher string `json:"verifiedPublisher,omitempty"` // publisher name if present

	// Reason — human-readable summary of the verification result
	Reason          string `json:"reason"`
}

// VerifiedCheck is a single governance verification check result.
type VerifiedCheck struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`    // "publisher", "transport", "deployment", "toolScope", "usage"
	Passed      bool   `json:"passed"`
	Score       int    `json:"score"`       // points awarded (0 if failed)
	MaxScore    int    `json:"maxScore"`    // max points for this check
	Description string `json:"description"` // what was checked
	Detail      string `json:"detail"`      // why it passed/failed
}

// VerifiedFinding is a governance issue found during verification.
type VerifiedFinding struct {
	Severity    string `json:"severity"`    // Critical, High, Medium, Low
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// VerifiedSummary is the cluster-level summary of all verified resources.
type VerifiedSummary struct {
	TotalCatalogs     int       `json:"totalCatalogs"`
	TotalScored       int       `json:"totalScored"`
	VerifiedCount     int       `json:"verifiedCount"`     // score >= 70 → "Verified"
	UnverifiedCount   int       `json:"unverifiedCount"`   // 50 <= score < 70 → "Unverified"
	RejectedCount     int       `json:"rejectedCount"`     // score < 50 → "Rejected"
	PendingCount      int       `json:"pendingCount"`      // not yet scored → "Pending"
	WarningCount      int       `json:"warningCount"`      // kept for backward compat
	CriticalCount     int       `json:"criticalCount"`     // score < 30
	AverageScore      int       `json:"averageScore"`
	TotalTools        int       `json:"totalTools"`
	TotalAgentUsages  int       `json:"totalAgentUsages"`
	LastReconcile     time.Time `json:"lastReconcile"`
}

// GradeFromScore returns the letter grade for a numeric score.
func GradeFromScore(score int) string {
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

// StatusFromScore returns the verification status for a numeric score using default thresholds.
func StatusFromScore(score int) string {
	return StatusFromScoreWithThresholds(score, 70, 50)
}

// StatusFromScoreWithThresholds returns the verification status using custom thresholds.
func StatusFromScoreWithThresholds(score, verifiedThreshold, unverifiedThreshold int) string {
	switch {
	case score >= verifiedThreshold:
		return "Verified"
	case score >= unverifiedThreshold:
		return "Unverified"
	default:
		return "Rejected"
	}
}
