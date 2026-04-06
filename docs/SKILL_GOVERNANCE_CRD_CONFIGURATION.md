# Skill Governance CRD Configuration Guide

## Overview

This document explains the **configuration updates** made to support Skill Governance (Agent Skill Catalogs) in the MCP Security Governance system, including:

1. **MCPGovernancePolicy CRD** — Policy definitions with new `skillGovernance` section
2. **GovernanceEvaluation CRD** — Evaluation results with new `skillCatalogScores` field
3. **Scoring Architecture** — How skill-patterns ConfigMap drives governance checks

---

## 1. MCPGovernancePolicy Configuration Updates

### File Location
- **CRD Definition:** `deploy/crds/governance-crds.yaml` (lines 1-240)
- **Types Definition:** `controller/pkg/apis/governance/v1alpha1/types.go` (lines 440-475)
- **Sample Policy:** `deploy/samples/governance-policy.yaml`

### New `skillGovernance` Section in MCPGovernancePolicySpec

The MCPGovernancePolicy now includes a new optional `skillGovernance` block:

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-mcp-policy
spec:
  # ... existing fields (requireAgentGateway, requireCORS, etc.) ...
  
  # NEW: Skill Governance Configuration
  skillGovernance:
    # Enable skill governance checks (default: true)
    enabled: true
    
    # Scan GitHub repository content for security patterns
    # If false: only metadata checks are performed (SKL-001..SKL-008)
    # If true: also performs content scanning (SKL-SEC-001..SKL-SEC-006)
    scanRepoContent: false
    
    # GitHub Personal Access Token (optional, for private repos)
    githubToken: ""  # Can be read from environment: $GITHUB_TOKEN
    
    # How long to cache scan results (minutes)
    scanCacheTTLMinutes: 60
    
    # Mark overall skill score as FAILING if prompt injection is detected (SKL-SEC-001)
    failOnPromptInjection: true
    
    # Mark overall skill score as FAILING if privilege escalation is detected (SKL-SEC-002)
    failOnPrivilegeEscalation: true
    
    # Allowed external domains (suppress SKL-SEC-003 for these)
    allowedExternalDomains:
      - "api.example.com"
      - "trusted-service.io"
    
    # Categories that MUST include safety guardrails (SKL-SEC-006)
    # Defaults to: ["database", "infra", "admin"]
    requireSafetyGuardrails:
      - database
      - infra
      - admin
```

### Go Type Definition

```go
// File: controller/pkg/evaluator/evaluator.go (lines 441-475)

type SkillGovernancePolicy struct {
    // Enabled activates skill governance checks (SKL-001 through SKL-SEC-006)
    Enabled bool
    
    // ScanRepoContent fetches and pattern-scans GitHub repository content
    ScanRepoContent bool
    
    // GitHubToken for private repositories
    GitHubToken string
    
    // ScanCacheTTLMinutes caches scan results
    ScanCacheTTLMinutes int
    
    // FailOnPromptInjection marks skill as failing on SKL-SEC-001
    FailOnPromptInjection bool
    
    // FailOnPrivilegeEscalation marks skill as failing on SKL-SEC-002
    FailOnPrivilegeEscalation bool
    
    // AllowedExternalDomains suppresses SKL-SEC-003 for these domains
    AllowedExternalDomains []string
    
    // RequireSafetyGuardrails lists categories needing guardrails (SKL-SEC-006)
    RequireSafetyGuardrails []string
    
    // PatternMountPath filesystem path to skill-patterns ConfigMap
    // Default: /etc/mcp-governance/skill-patterns
    PatternMountPath string
}
```

### Discovery & Parsing

**File:** `controller/pkg/discovery/discovery.go` (lines 1240-1290)

The `DiscoverGovernancePolicy()` function parses the `skillGovernance` block:

```go
// Parse skillGovernance configuration
if sgMap, ok := spec["skillGovernance"].(map[string]interface{}); ok {
    sg := evaluator.SkillGovernancePolicy{
        PatternMountPath: "/etc/mcp-governance/skill-patterns",
    }
    if val, ok := sgMap["enabled"].(bool); ok {
        sg.Enabled = val
    }
    if val, ok := sgMap["scanRepoContent"].(bool); ok {
        sg.ScanRepoContent = val
    }
    if val, ok := sgMap["githubToken"].(string); ok {
        sg.GitHubToken = val
    }
    // ... etc ...
    policy.SkillGovernance = sg
} else {
    // Default: enable metadata checks only
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

## 2. GovernanceEvaluation Status Updates

### New `skillCatalogScores` Field

**File:** `deploy/crds/governance-crds.yaml` (lines 380+)

The GovernanceEvaluation CRD now includes `skillCatalogScores` in the status:

```yaml
status:
  # ... existing fields (score, phase, findings, etc.) ...
  
  # NEW: Skill Catalog governance scores
  skillCatalogScores:
    type: array
    description: "Per-skill-catalog governance scores"
    items:
      type: object
      properties:
        name:
          type: string
          description: "Name of the SkillCatalog CR"
        namespace:
          type: string
          description: "Namespace of the SkillCatalog CR"
        version:
          type: string
          description: "Version from SkillCatalog.spec.version"
        category:
          type: string
          description: "Category from SkillCatalog.spec.category"
        repoURL:
          type: string
          description: "Repository URL for skill implementation"
        score:
          type: integer
          minimum: 0
          maximum: 100
          description: "Final governance score (0-100)"
        status:
          type: string
          enum: ["pass", "warning", "fail"]
          description: "Status based on score thresholds"
        scannedFiles:
          type: integer
          description: "Number of files scanned from repository"
        findings:
          type: array
          items:
            type: object
            properties:
              checkID:
                type: string
                description: "Check ID (e.g., SKL-001, SKL-SEC-001)"
              severity:
                type: string
                enum: ["Critical", "High", "Medium", "Low"]
              category:
                type: string
              title:
                type: string
              remediation:
                type: string
```

### Go Type Definition

**File:** `controller/pkg/evaluator/evaluator.go` (lines ~200-230)

```go
type SkillCatalogScore struct {
    Name         string                `json:"name"`
    Namespace    string                `json:"namespace"`
    Version      string                `json:"version"`
    Category     string                `json:"category"`
    RepoURL      string                `json:"repoURL,omitempty"`
    Status       string                `json:"status"` // "pass", "warning", "fail"
    Score        int                   `json:"score"`  // 0-100
    Findings     []SkillCatalogFinding `json:"findings,omitempty"`
    ScannedFiles int                   `json:"scannedFiles"`
}

type SkillCatalogFinding struct {
    CheckID        string `json:"checkID"`    // e.g., "SKL-001", "SKL-SEC-001"
    Severity       string `json:"severity"`   // "Critical", "High", "Medium", "Low"
    Category       string `json:"category"`   // Metadata, Content, etc.
    Title          string `json:"title"`
    Remediation    string `json:"remediation"`
    FilePath       string `json:"filePath,omitempty"`
    Line           int    `json:"line,omitempty"`
    MatchedPattern string `json:"matchedPattern,omitempty"`
}
```

### Status Update Flow

**File:** `controller/pkg/discovery/discovery.go` (lines 1519-1550)

The `UpdateEvaluationStatus()` function writes skill governance scores:

```go
// Build skillCatalogScores
skillCatalogScores := make([]interface{}, 0)
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
        "version":      scs.Version,
        "category":     scs.Category,
        "repoURL":      scs.RepoURL,
        "score":        int64(scs.Score),
        "status":       scs.Status,
        "scannedFiles": int64(scs.ScannedFiles),
        "findings":     sfindings,
    })
}

// Set in status
status := map[string]interface{}{
    // ... existing fields ...
    "skillCatalogScores": skillCatalogScores,
    // ... other fields ...
}
```

---

## 3. Skill Patterns ConfigMap & Scoring

### ConfigMap Structure

**File:** `deploy/k8s/skill-patterns-configmap.yaml`

The `mcp-governance-skill-patterns` ConfigMap contains pattern definitions for security scanning:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-governance-skill-patterns
  namespace: mcp-governance
data:
  # SKL-SEC-001: Prompt Injection Patterns
  prompt-injection: |
    ignore previous instructions
    disregard your instructions
    bypass safety
    jailbreak
    # ... more patterns ...

  # SKL-SEC-002: Privilege Escalation
  privilege-escalation: |
    sudo
    run as root
    escalate privileges
    chmod 777
    # ... more patterns ...

  # SKL-SEC-003: Data Exfiltration
  data-exfiltration: |
    exfiltrate
    send data to external
    webhook.site
    # ... more patterns ...

  # SKL-SEC-004: Credential Harvesting
  credential-harvesting: |
    steal credentials
    harvest passwords
    # ... more patterns ...

  # SKL-SEC-005: Scope Creep
  scope-creep: |
    data: deploy, provision infrastructure
    analytics: delete database, drop table
    # ... category rules ...

  # SKL-SEC-006: Safety Guardrails
  safety-guardrails: |
    database: confirmation required, dry run
    infra: confirmation required, change management
    # ... category requirements ...
```

### Kubernetes Deployment Integration

**File:** `deploy/k8s/deployment.yaml`

The ConfigMap is mounted to the controller pod:

```yaml
spec:
  containers:
  - name: controller
    volumeMounts:
    # ... existing mounts ...
    - name: skill-patterns
      mountPath: /etc/mcp-governance/skill-patterns
      readOnly: true
  volumes:
  # ... existing volumes ...
  - name: skill-patterns
    configMap:
      name: mcp-governance-skill-patterns
      optional: true  # Non-fatal if ConfigMap doesn't exist
```

### Pattern Loading & Hot-Reload

**File:** `controller/pkg/skillscanner/scanner.go`

The `PatternLoader` reads patterns from the mounted ConfigMap:

```go
type PatternLoader struct {
    mountPath string
    ttl       time.Duration
    cache     *PatternSet
    cacheTime time.Time
}

func (pl *PatternLoader) Load() (*PatternSet, error) {
    // Check TTL cache (default 30 seconds)
    if time.Since(pl.cacheTime) < pl.ttl {
        return pl.cache, nil
    }
    
    // Re-read from filesystem
    patterns := ParsePatternSet(readConfigMapFiles(pl.mountPath))
    pl.cache = patterns
    pl.cacheTime = time.Now()
    return patterns, nil
}
```

**Benefits:**
- **Hot-reload:** Update ConfigMap → patterns are reloaded on next scan (no pod restart)
- **Fallback:** If ConfigMap is missing or inaccessible, uses built-in defaults
- **Caching:** TTL-based caching reduces filesystem reads

---

## 4. Governance Evaluation Flow

### Complete Skill Governance Evaluation Pipeline

```
MCPGovernancePolicy (skillGovernance config)
    ↓
DiscoverClusterState()
    └─→ discoverSkillCatalogs() [discovers all SkillCatalog CRs from agentregistry.dev]
    ↓
EvaluationResult = Evaluate(state, policy)
    └─→ checkSkillCatalogs(state, policy, patternLoader)
        ├─→ checkSkillMetadata(skill) [SKL-001 through SKL-008]
        ├─→ scanSkillRepo(skill, policy, patterns) [SKL-SEC-001 through SKL-SEC-006]
        └─→ scoreSkillCatalog(findings) [compute score 0-100, assign status]
    ↓
UpdateEvaluationStatus(policyName, result)
    └─→ writes skillCatalogScores to all GovernanceEvaluation CRs
```

### Scoring Logic

**File:** `controller/pkg/evaluator/evaluator_skills.go`

1. **Metadata Checks (SKL-001..SKL-008):** 0-100 base score
   - Deductions: 10-25 points per finding
   - Final: score = 100 - (sum of deductions)

2. **Content Scanning (SKL-SEC-001..SKL-SEC-006):** Pattern matching
   - If critical pattern found (e.g., prompt injection) → can force score to 0
   - If high severity → 20-30 point deduction

3. **Final Status:**
   - `pass`: score >= 80
   - `warning`: score 50-79
   - `fail`: score < 50

---

## 5. Example: Complete Configuration

### Sample MCPGovernancePolicy with Skill Governance

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-mcp-policy
  namespace: mcp-governance
spec:
  # ... existing MCP governance config ...
  requireAgentGateway: true
  requireCORS: true
  requireTLS: true
  
  # NEW: Skill Governance Configuration
  skillGovernance:
    enabled: true
    scanRepoContent: true
    githubToken: "" # Set via environment variable
    scanCacheTTLMinutes: 60
    failOnPromptInjection: true
    failOnPrivilegeEscalation: true
    allowedExternalDomains:
      - "github.com"
      - "api.example.com"
    requireSafetyGuardrails:
      - database
      - infra
      - admin
  
  # Existing policy fields
  targetNamespaces: []
  excludeNamespaces:
    - kube-system
```

### Sample GovernanceEvaluation Result

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: GovernanceEvaluation
metadata:
  name: enterprise-evaluation
spec:
  policyRef: enterprise-mcp-policy
  evaluationScope: cluster
status:
  score: 85
  phase: "Compliant"
  lastEvaluationTime: "2026-03-31T10:15:30Z"
  
  # Skill Catalog Scores
  skillCatalogScores:
    - name: anthoropic-skills-1-0-0
      namespace: default
      version: "1.0.0"
      category: "data-processing"
      repoURL: "https://github.com/anthropic/skills-repo"
      score: 95
      status: "pass"
      scannedFiles: 15
      findings:
        - checkID: "SKL-004"
          severity: "Low"
          category: "Metadata"
          title: "Missing resource-uid label"
          remediation: "Add resource-uid label to SkillCatalog spec"
```

---

## 6. Configuration Reference

### skillGovernance Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | true | Activate skill governance checks |
| `scanRepoContent` | bool | false | Fetch & scan GitHub repository |
| `githubToken` | string | "" | Personal access token for private repos |
| `scanCacheTTLMinutes` | int | 60 | Cache duration for scan results |
| `failOnPromptInjection` | bool | true | Fail overall score on SKL-SEC-001 |
| `failOnPrivilegeEscalation` | bool | true | Fail overall score on SKL-SEC-002 |
| `allowedExternalDomains` | []string | [] | Domains allowed for data exfiltration (SKL-SEC-003) |
| `requireSafetyGuardrails` | []string | ["database", "infra", "admin"] | Categories requiring guardrails (SKL-SEC-006) |

### Scoring Thresholds

| Threshold | Score Range | Status |
|-----------|------------|--------|
| **Pass** | >= 80 | ✅ Compliant |
| **Warning** | 50-79 | ⚠️ Needs Review |
| **Fail** | < 50 | ❌ Non-Compliant |

### Check ID Reference

#### Metadata Checks (SKL-001..SKL-008)

| ID | Severity | Check | Points |
|----|----------|-------|--------|
| SKL-001 | Medium | Missing version field | 10 |
| SKL-002 | Low | Unknown repository source | 5 |
| SKL-003 | Critical | HTTP URL (not HTTPS) | 25 |
| SKL-004 | Low | Missing resource-uid label | 5 |
| SKL-005 | Low | Missing category field | 5 |
| SKL-006 | Low | Description < 20 characters | 5 |
| SKL-007 | High | Production env without version | 20 |
| SKL-008 | Medium | Personal GitHub account | 10 |

#### Content Checks (SKL-SEC-001..SKL-SEC-006)

| ID | Severity | Pattern Category | Scope |
|----|----------|------------------|-------|
| SKL-SEC-001 | Critical | Prompt Injection | Repository content |
| SKL-SEC-002 | Critical | Privilege Escalation | Repository content |
| SKL-SEC-003 | High | Data Exfiltration | Repository content |
| SKL-SEC-004 | High | Credential Harvesting | Repository content |
| SKL-SEC-005 | Medium | Scope Creep | Category mismatch |
| SKL-SEC-006 | Medium | Safety Guardrails | Category mismatch |

---

## 7. File Index

### Configuration Files
- **Policy CRD:** `deploy/crds/governance-crds.yaml`
- **Policy Sample:** `deploy/samples/governance-policy.yaml`
- **Skill Patterns ConfigMap:** `deploy/k8s/skill-patterns-configmap.yaml`
- **Deployment with volume mounts:** `deploy/k8s/deployment.yaml`

### Implementation Files
- **Policy Types:** `controller/pkg/apis/governance/v1alpha1/types.go`
- **Evaluator Core:** `controller/pkg/evaluator/evaluator.go` (SkillGovernancePolicy struct)
- **Skill Checks:** `controller/pkg/evaluator/evaluator_skills.go`
- **Pattern Scanner:** `controller/pkg/skillscanner/scanner.go`
- **Pattern Definitions:** `controller/pkg/skillscanner/patterns.go`
- **Discovery & Status:** `controller/pkg/discovery/discovery.go`

### Dashboard Files
- **API Types:** `dashboard/src/lib/types.ts` (SkillCatalogScore, etc.)
- **API Client:** `dashboard/src/lib/api.ts`
- **Component:** `dashboard/src/components/SkillCatalog.tsx`
- **Page:** `dashboard/src/app/page.tsx`
- **Proxy Route:** `dashboard/src/app/api/governance/[...path]/route.ts` (NEW)

---

## 8. Deployment Checklist

- [ ] Apply CRD: `kubectl apply -f deploy/crds/governance-crds.yaml`
- [ ] Create ConfigMap: `kubectl apply -f deploy/k8s/skill-patterns-configmap.yaml`
- [ ] Create MCPGovernancePolicy: `kubectl apply -f deploy/samples/governance-policy.yaml`
- [ ] Deploy controller with volumeMount: `kubectl apply -f deploy/k8s/deployment.yaml`
- [ ] Verify ConfigMap is mounted: `kubectl exec <controller-pod> -- ls /etc/mcp-governance/skill-patterns`
- [ ] Create test SkillCatalog CR
- [ ] Verify scores in GovernanceEvaluation status: `kubectl get governanceevaluations -o yaml`
- [ ] Access dashboard at `http://localhost:3000` and navigate to **Skill Catalogs** tab

---

## 9. No Changes Required

The following CRs **do not need updates** for skill governance:
- ✅ **MCPServerCatalog** — Verified Catalog scoring (separate feature)
- ✅ **AgentgatewayBackend** — MCP gateway configuration (unchanged)
- ✅ **KagentAgent** — Agent configurations (unchanged)
- ✅ **KagentMCPServer / RemoteMCPServer** — MCP server definitions (unchanged)

Skill governance is **additive** — it evaluates new SkillCatalog CRs without affecting existing resources.

