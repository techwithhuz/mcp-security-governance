# Skill Governance CRD Updates - Visual Summary

## Quick Reference: What Changed in MCPGovernancePolicies & GovernanceEvaluations CRs

---

## 1. MCPGovernancePolicy (governance.mcp.io/v1alpha1)

### BEFORE (Existing Structure)
```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-mcp-policy
spec:
  # MCP Gateway & Security Controls
  requireAgentGateway: true
  requireCORS: true
  requireJWTAuth: true
  requireRBAC: true
  requirePromptGuard: false
  requireTLS: true
  requireRateLimit: false
  requireHardenedDeployment: true
  
  # Scoring & Penalties
  scoringWeights:
    agentGatewayIntegration: 25
    authentication: 20
    authorization: 15
    corsPolicy: 10
    tlsEncryption: 10
    promptGuard: 10
    rateLimit: 5
    toolScope: 5
    hardenedDeployment: 15
  
  severityPenalties:
    critical: 40
    high: 25
    medium: 15
    low: 5
  
  # Verified Catalog Scoring (existing feature)
  verifiedCatalogScoring:
    securityWeight: 50
    trustWeight: 30
    complianceWeight: 20
    verifiedThreshold: 70
    unverifiedThreshold: 50
    checkMaxScores:
      PUB-001: 20
      SEC-001: 15
  
  # Monitoring Scope
  targetNamespaces: []
  excludeNamespaces:
    - kube-system
  
  # Control Plane Config
  scanInterval: "5m"
  enableAuditLogging: true
  clusterName: "prod-cluster"
  
  # AI Agent (existing feature)
  aiAgent:
    enabled: true
    provider: "gemini"
    model: "gemini-2.5-flash"
```

### AFTER (With Skill Governance)
```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-mcp-policy
spec:
  # ... all existing fields above ...
  
  # ✨ NEW: Skill Governance Configuration (lines 1240-1290 in discovery.go)
  skillGovernance:
    # Enable skill governance checks (SKL-001 through SKL-SEC-006)
    enabled: true
    
    # Scan GitHub repository for security patterns
    scanRepoContent: false  # Set to true for deep scanning
    
    # GitHub token for private repositories (optional)
    githubToken: ""
    
    # Cache scan results for 60 minutes
    scanCacheTTLMinutes: 60
    
    # Mark overall score as FAILING if critical patterns found
    failOnPromptInjection: true      # Force fail on SKL-SEC-001
    failOnPrivilegeEscalation: true  # Force fail on SKL-SEC-002
    
    # Allow data exfiltration to these trusted domains
    allowedExternalDomains:
      - "github.com"
      - "api.trusted-service.io"
    
    # Categories that MUST have safety guardrails
    requireSafetyGuardrails:
      - database      # Needs: "confirmation required", "dry run", etc.
      - infra         # Needs: "change management", "dry run", etc.
      - admin         # Needs: "admin access required", "audit logged", etc.
```

### Go Type: Policy.SkillGovernance

```go
// In controller/pkg/evaluator/evaluator.go
type Policy struct {
    // ... existing fields ...
    SkillGovernance SkillGovernancePolicy  // ✨ NEW
}

type SkillGovernancePolicy struct {
    Enabled                   bool     // Activate checks
    ScanRepoContent           bool     // Fetch from GitHub
    GitHubToken               string   // Auth token
    ScanCacheTTLMinutes       int      // Cache duration
    FailOnPromptInjection     bool     // Force fail on SKL-SEC-001
    FailOnPrivilegeEscalation bool     // Force fail on SKL-SEC-002
    AllowedExternalDomains    []string // Trusted domains
    RequireSafetyGuardrails   []string // Guardrail categories
    PatternMountPath          string   // ConfigMap mount path
}
```

### Discovery Parsing

```go
// In discovery.go lines 1240-1290
if sgMap, ok := spec["skillGovernance"].(map[string]interface{}); ok {
    sg := evaluator.SkillGovernancePolicy{
        PatternMountPath: "/etc/mcp-governance/skill-patterns",
    }
    // Extract all fields from YAML spec...
    policy.SkillGovernance = sg
} else {
    // Default: enable metadata checks only (no repo scanning)
    policy.SkillGovernance = evaluator.SkillGovernancePolicy{
        Enabled:                   true,
        ScanRepoContent:           false,
        FailOnPromptInjection:     true,
        FailOnPrivilegeEscalation: true,
        ScanCacheTTLMinutes:       60,
        RequireSafetyGuardrails:   []string{"database", "infra", "admin"},
        PatternMountPath:          "/etc/mcp-governance/skill-patterns",
    }
}
```

---

## 2. GovernanceEvaluation Status (governance.mcp.io/v1alpha1)

### BEFORE (Existing Status Fields)
```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: GovernanceEvaluation
metadata:
  name: enterprise-evaluation
spec:
  policyRef: enterprise-mcp-policy
  evaluationScope: cluster
status:
  # Overall score
  score: 85
  phase: "Compliant"  # Compliant, PartiallyCompliant, NonCompliant, Critical
  findingsCount: 5
  lastEvaluationTime: "2026-03-31T10:15:30Z"
  
  # General findings
  findings:
    - id: "GATE-001"
      severity: "High"
      category: "AgentGateway"
      title: "MCP server not behind agentgateway"
      description: "..."
  
  # Resource summary
  resourceSummary:
    gatewaysFound: 2
    agentgatewayBackends: 3
    agentgatewayPolicies: 2
    httpRoutes: 5
    kagentAgents: 3
    kagentMCPServers: 4
    kagentRemoteMCPServers: 2
    services: 10
    namespaces: 5
    compliantResources: 8
    nonCompliantResources: 2
    totalMCPEndpoints: 10
    exposedMCPEndpoints: 3
  
  # Score breakdown by category
  scoreBreakdown:
    agentGatewayScore: 80
    authenticationScore: 75
    authorizationScore: 70
    corsScore: 85
    tlsScore: 90
    promptGuardScore: 60
    rateLimitScore: 70
    toolScopeScore: 85
    hardenedDeploymentScore: 88
  
  # Scores per namespace
  namespaceScores:
    - namespace: default
      score: 85
      findings: 2
    - namespace: agents
      score: 90
      findings: 1
  
  # Verified Catalog scores (existing feature)
  verifiedCatalogScores:
    - catalogName: "anthropic-servers-1-0-0"
      namespace: "default"
      status: "Verified"
      compositeScore: 92
      securityScore: 88
      trustScore: 95
      complianceScore: 90
  
  # MCP Server scores (existing feature)
  mcpServerScores:
    - name: "claude-mcp-server"
      namespace: "default"
      source: "KagentMCPServer"
      status: "compliant"
      score: 88
      toolCount: 15
      effectiveToolCount: 12
      criticalFindings: 0
```

### AFTER (With Skill Governance Scores)
```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: GovernanceEvaluation
metadata:
  name: enterprise-evaluation
spec:
  policyRef: enterprise-mcp-policy
  evaluationScope: cluster
status:
  # ... all existing fields above ...
  
  # ✨ NEW: Skill Catalog governance scores (lines 1519-1550 in discovery.go)
  skillCatalogScores:
    - name: anthoropic-skills-1-0-0              # SkillCatalog CR name
      namespace: default                         # SkillCatalog namespace
      version: "1.0.0"                          # SkillCatalog.spec.version
      category: "data-processing"               # SkillCatalog.spec.category
      repoURL: "https://github.com/anthropic/skills"
      score: 95                                 # 0-100 governance score
      status: "pass"                            # pass | warning | fail
      scannedFiles: 15                          # Files scanned from repo
      findings:
        - checkID: "SKL-001"                    # Metadata checks
          severity: "Low"
          category: "Metadata"
          title: "Missing version field"
          remediation: "Add spec.version to SkillCatalog"
        - checkID: "SKL-003"                    # HTTP security check
          severity: "Critical"
          category: "Security"
          title: "Repository uses HTTP instead of HTTPS"
          remediation: "Update repository URL to use HTTPS"
        - checkID: "SKL-SEC-001"                # Content scan findings
          severity: "Critical"
          category: "Security"
          title: "Prompt injection pattern detected"
          remediation: "Review and remove prompt injection patterns"
          filePath: "main.py"
          line: 42
          matchedPattern: "ignore previous instructions"
    
    - name: custom-skill-catalog                # Another skill
      namespace: agents
      version: "2.1.0"
      category: "admin"
      repoURL: "https://github.com/org/admin-skills"
      score: 78                                 # Warning threshold
      status: "warning"
      scannedFiles: 23
      findings:
        - checkID: "SKL-SEC-006"                # Safety guardrails
          severity: "Medium"
          category: "Safety"
          title: "Admin category missing safety guardrails"
          remediation: "Add comments like 'confirmation required', 'audit logged'"
```

### Go Type: EvaluationResult.SkillCatalogScores

```go
// In controller/pkg/evaluator/evaluator.go
type EvaluationResult struct {
    // ... existing fields ...
    SkillCatalogScores []SkillCatalogScore  // ✨ NEW
}

type SkillCatalogScore struct {
    Name         string                `json:"name"`
    Namespace    string                `json:"namespace"`
    Version      string                `json:"version"`
    Category     string                `json:"category"`
    RepoURL      string                `json:"repoURL,omitempty"`
    Score        int                   `json:"score"`           // 0-100
    Status       string                `json:"status"`          // "pass", "warning", "fail"
    Findings     []SkillCatalogFinding `json:"findings,omitempty"`
    ScannedFiles int                   `json:"scannedFiles"`
}

type SkillCatalogFinding struct {
    CheckID        string `json:"checkID"`        // SKL-001, SKL-SEC-001, etc.
    Severity       string `json:"severity"`       // Critical, High, Medium, Low
    Category       string `json:"category"`
    Title          string `json:"title"`
    Remediation    string `json:"remediation"`
    FilePath       string `json:"filePath,omitempty"`
    Line           int    `json:"line,omitempty"`
    MatchedPattern string `json:"matchedPattern,omitempty"`
}
```

### Status Update Flow

```go
// In discovery.go - UpdateEvaluationStatus() function

// (1) Build findings array
for _, scs := range result.SkillCatalogScores {
    sfindings := make([]interface{}, 0)
    for _, sf := range scs.Findings {
        sfindings = append(sfindings, map[string]interface{}{
            "checkID":     sf.CheckID,
            "severity":    sf.Severity,
            "category":    sf.Category,
            "title":       sf.Title,
            "remediation": sf.Remediation,
        })
    }
    skillCatalogScores = append(skillCatalogScores, map[string]interface{}{
        "name":         scs.Name,
        "namespace":    scs.Namespace,
        "version":     scs.Version,
        "category":    scs.Category,
        "repoURL":     scs.RepoURL,
        "score":       int64(scs.Score),
        "status":      scs.Status,
        "scannedFiles": int64(scs.ScannedFiles),
        "findings":    sfindings,
    })
}

// (2) Write to GovernanceEvaluation.status
status := map[string]interface{}{
    // ... existing fields ...
    "skillCatalogScores": skillCatalogScores,
    // ... other fields ...
}
```

---

## 3. Discovery Process (What Happens During Evaluation)

### Complete Evaluation Pipeline with Skill Governance

```
┌─────────────────────────────────────────────────────────────────┐
│ (1) Controller Reads MCPGovernancePolicy                        │
│     ├─ Lines 1240-1290 in discovery.go                         │
│     └─ Parses skillGovernance section into SkillGovernancePolicy│
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ (2) DiscoverClusterState() - Lines 100-101                      │
│     └─ state.SkillCatalogs = d.discoverSkillCatalogs(ctx)       │
│        └─ Lists all agentregistry.dev/v1alpha1/skillcatalogs   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ (3) Evaluate() Function - Calls checkSkillCatalogs()            │
│     └─ evaluator_skills.go                                      │
│        ├─ checkSkillMetadata(skill)     [SKL-001..SKL-008]     │
│        ├─ scanSkillRepo(skill)          [SKL-SEC-001..006]     │
│        └─ scoreSkillCatalog(findings)   [0-100 score + status] │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ (4) EvaluationResult Built                                      │
│     ├─ result.SkillCatalogScores = [score1, score2, ...]       │
│     ├─ result.Score = overall cluster score                    │
│     └─ result.Findings = all findings                          │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ (5) UpdateEvaluationStatus() - Lines 1519-1550                  │
│     └─ Writes skillCatalogScores to GovernanceEvaluation.status │
│        for each GovernanceEvaluation CR that references the     │
│        MCPGovernancePolicy                                      │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ (6) Dashboard Displays Results                                  │
│     ├─ /api/governance/skill-catalogs endpoint (controller)     │
│     ├─ /api/governance/skill-catalogs proxy (dashboard)         │
│     └─ SkillCatalog.tsx component (frontend)                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Pattern ConfigMap Integration

### How mcp-governance-skill-patterns ConfigMap Works

```
ConfigMap: mcp-governance-skill-patterns
├─ Key: "prompt-injection" → 20+ pattern strings
├─ Key: "privilege-escalation" → 20+ pattern strings
├─ Key: "data-exfiltration" → 15+ pattern strings
├─ Key: "credential-harvesting" → 15+ pattern strings
├─ Key: "scope-creep" → Category-specific rules
└─ Key: "safety-guardrails" → Category requirements

↓ [Kubernetes]

Pod Volume Mount
├─ Name: "skill-patterns"
├─ ConfigMap: "mcp-governance-skill-patterns"
└─ Mount Path: "/etc/mcp-governance/skill-patterns"

↓ [Controller Pod Filesystem]

/etc/mcp-governance/skill-patterns/
├─ prompt-injection          (plaintext file with patterns)
├─ privilege-escalation
├─ data-exfiltration
├─ credential-harvesting
├─ scope-creep
└─ safety-guardrails

↓ [PatternLoader Hot-Reload]

PatternLoader.Load() called
├─ Check TTL cache (30s)
├─ If expired, re-read files from disk
├─ Parse patterns into PatternSet
├─ Return patterns for scanning

↓ [Pattern Matching During Scan]

scanSkillRepo(skill, policy, patternLoader)
├─ Load latest patterns from ConfigMap
├─ Fetch skill repo content (GitHub API)
├─ For each pattern category:
│  ├─ Match against file contents
│  ├─ Generate SkillCatalogFinding per match
│  └─ Deduct points from score
└─ Return final SkillCatalogScore
```

### Update Pattern in Real-Time (No Pod Restart!)

```bash
# (1) Edit ConfigMap
kubectl edit configmap mcp-governance-skill-patterns -n mcp-governance

# (2) Add new prompt injection patterns to "prompt-injection" key
# ...
# jailbreak
# developer mode
# break free
# ...

# (3) Save and exit

# (4) Next time controller runs Evaluate():
#     - PatternLoader.Load() detects TTL expired
#     - Re-reads from /etc/mcp-governance/skill-patterns
#     - Applies new patterns to scanSkillRepo()
#
# No pod restart needed! ✨ Hot-reload working!
```

---

## 5. File Changes Summary

### CRD Definitions (YAML)
| File | Lines | Change |
|------|-------|--------|
| `deploy/crds/governance-crds.yaml` | 1-240 | MCPGovernancePolicy CRD (no schema changes needed) |
| `deploy/crds/governance-crds.yaml` | 241-489 | GovernanceEvaluation CRD **+ skillCatalogScores field** |
| `deploy/k8s/deployment.yaml` | 40-60 | **+ skill-patterns ConfigMap volumeMount** |
| `deploy/k8s/skill-patterns-configmap.yaml` | NEW | **New ConfigMap with 6 pattern keys** |

### Go Implementation
| File | Lines | Change |
|------|-------|--------|
| `controller/pkg/evaluator/evaluator.go` | 438 | `SkillGovernance SkillGovernancePolicy` field added to Policy |
| `controller/pkg/evaluator/evaluator.go` | 441-475 | **New SkillGovernancePolicy struct definition** |
| `controller/pkg/discovery/discovery.go` | 100-101 | **DiscoverClusterState() calls discoverSkillCatalogs()** |
| `controller/pkg/discovery/discovery.go` | 1240-1290 | **Parse skillGovernance config from MCPGovernancePolicy** |
| `controller/pkg/discovery/discovery.go` | 1519-1550 | **Write skillCatalogScores to GovernanceEvaluation status** |
| `controller/pkg/evaluator/evaluator_skills.go` | NEW | **All skill governance check functions (SKL-001..SKL-SEC-006)** |
| `controller/pkg/skillscanner/scanner.go` | NEW | **PatternLoader + pattern matching implementation** |
| `controller/pkg/skillscanner/patterns.go` | NEW | **Pattern definitions + ParsePatternSet()** |

### Dashboard (TypeScript/React)
| File | Change |
|------|--------|
| `dashboard/src/lib/types.ts` | **+ SkillCatalogScore, SkillCatalogFinding, SkillCatalogsResponse interfaces** |
| `dashboard/src/components/SkillCatalog.tsx` | **New component for displaying skill scores** |
| `dashboard/src/app/page.tsx` | **+ SkillCatalogs tab, fetch call, navigation** |
| `dashboard/src/lib/api.ts` | **Updated to use relative paths for API proxy** |
| `dashboard/src/app/api/governance/[...path]/route.ts` | **New catch-all proxy route for skill API calls** |

---

## 6. Configuration Checklist

### Prerequisites
- [ ] Controller pod has permission to list `skillcatalogs.agentregistry.dev`
  - RBAC ClusterRole updated: `deploy/k8s/deployment.yaml` ✅
- [ ] ConfigMap `mcp-governance-skill-patterns` exists in `mcp-governance` namespace
  - Apply: `kubectl apply -f deploy/k8s/skill-patterns-configmap.yaml` ✅
- [ ] Controller pod has volumeMount for ConfigMap
  - Deployment updated: `deploy/k8s/deployment.yaml` ✅

### Activation
- [ ] Create MCPGovernancePolicy with `skillGovernance` section (optional section)
  - If omitted: defaults to metadata-only checks
  - Example: `deploy/samples/governance-policy.yaml`
- [ ] Create SkillCatalog CRs in cluster
  - These are discovered from `agentregistry.dev/v1alpha1`
- [ ] Run Evaluate() or wait for next scan cycle (default: 5m)
- [ ] Check GovernanceEvaluation status for `skillCatalogScores`

### Verification
```bash
# (1) Verify ConfigMap mounted
kubectl exec -n mcp-governance deployment/mcp-governance-controller -- \
  ls -la /etc/mcp-governance/skill-patterns

# (2) Check skill scores in evaluation
kubectl get governanceevaluations -o yaml | grep skillCatalogScores

# (3) View dashboard
open http://localhost:3000
# Navigate to "Skill Catalogs" tab → Should show discovered skills with scores
```

---

## 7. Key Takeaways

### What's NEW for Skill Governance:
1. ✨ **MCPGovernancePolicy.spec.skillGovernance** — Policy configuration
2. ✨ **GovernanceEvaluation.status.skillCatalogScores[]** — Evaluation results
3. ✨ **mcp-governance-skill-patterns ConfigMap** — Pattern definitions
4. ✨ **evaluator_skills.go** — 8 metadata + 6 content checks (SKL-001..SKL-SEC-006)
5. ✨ **skillscanner package** — Pattern loader & scanner implementation
6. ✨ **Dashboard SkillCatalog tab** — Visual display of scores

### What's UNCHANGED:
- ✅ MCPServerCatalog (Verified Catalog scoring)
- ✅ AgentgatewayBackend, Backend, Policy CRs
- ✅ KagentAgent, KagentMCPServer, RemoteMCPServer CRs
- ✅ Existing governance checks (GATE-001, AUTH-001, etc.)
- ✅ Score computation model (same 0-100 scale)

### Configuration Is OPTIONAL:
- If `skillGovernance` section is omitted from MCPGovernancePolicy:
  - **Defaults to metadata-only checks** (SKL-001..SKL-008)
  - No repo content scanning
  - No pattern matching
  - Still contributes to overall cluster score
- If included:
  - Can enable `scanRepoContent: true` for deep scanning
  - Can set `failOnPromptInjection: true` to force fail on critical patterns
  - Can customize allowed domains and guardrail requirements

