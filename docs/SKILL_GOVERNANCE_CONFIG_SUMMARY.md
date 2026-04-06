# Skill Governance Configuration - Executive Summary

## Your Questions Answered

### Question 1: Is there any configuration update in MCPGovernancePolicies CR?

**YES ✅ — New `skillGovernance` Section Added**

The MCPGovernancePolicy CRD now supports an optional `skillGovernance` configuration block:

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-mcp-policy
spec:
  # ... existing fields (requireAgentGateway, requireTLS, etc.) ...
  
  # ✨ NEW: Skill Governance Configuration
  skillGovernance:
    enabled: true
    scanRepoContent: false          # Set to true for GitHub repo scanning
    githubToken: ""                 # For private repos
    scanCacheTTLMinutes: 60         # Cache duration
    failOnPromptInjection: true     # Force fail on SKL-SEC-001
    failOnPrivilegeEscalation: true # Force fail on SKL-SEC-002
    allowedExternalDomains:         # Skip SKL-SEC-003 for these
      - "github.com"
      - "api.example.com"
    requireSafetyGuardrails:        # Categories needing guardrails
      - database
      - infra
      - admin
```

**Key Files:**
- **CRD:** `deploy/crds/governance-crds.yaml` (no schema changes needed; section is parsed dynamically)
- **Types:** `controller/pkg/evaluator/evaluator.go` (line 438: `SkillGovernance SkillGovernancePolicy`)
- **Parsing:** `controller/pkg/discovery/discovery.go` (lines 1240-1290: `DiscoverGovernancePolicy()`)
- **Defaults:** If section omitted, metadata-only checks run (no repo scanning)

**Backward Compatible:** ✅ The field is optional; existing policies work as-is.

---

### Question 2: Is there any configuration update in GovernanceEvaluations CR?

**YES ✅ — New `skillCatalogScores` Field Added**

The GovernanceEvaluation CRD now includes skill governance scores in the status:

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
  
  # ... existing fields (findings, resourceSummary, scoreBreakdown, etc.) ...
  
  # ✨ NEW: Skill Catalog Governance Scores
  skillCatalogScores:
    - name: anthoropic-skills-1-0-0
      namespace: default
      version: "1.0.0"
      category: "data-processing"
      repoURL: "https://github.com/anthropic/skills"
      score: 95                     # 0-100 governance score
      status: "pass"                # pass | warning | fail
      scannedFiles: 15              # Files scanned from repo
      findings:
        - checkID: "SKL-001"        # Metadata checks
          severity: "Low"
          category: "Metadata"
          title: "Missing version field"
          remediation: "Add spec.version"
        - checkID: "SKL-SEC-001"    # Content scan findings
          severity: "Critical"
          category: "Security"
          title: "Prompt injection pattern detected"
          remediation: "Review and remove patterns"
          filePath: "main.py"
          line: 42
          matchedPattern: "ignore previous instructions"
```

**Key Files:**
- **CRD:** `deploy/crds/governance-crds.yaml` (lines 380+: `skillCatalogScores` schema)
- **Types:** `controller/pkg/evaluator/evaluator.go` (lines ~200-230: `SkillCatalogScore` struct)
- **Writer:** `controller/pkg/discovery/discovery.go` (lines 1519-1550: `UpdateEvaluationStatus()`)

**Backward Compatible:** ✅ New field is optional; existing evaluations display without it.

---

### Question 3: Where is the scoring related to the mcp-governance-skill-patterns ConfigMap?

**The ConfigMap contains the patterns used for security scanning (SKL-SEC-001 through SKL-SEC-006)**

### Complete Scoring Architecture

```
MCPGovernancePolicy
(skillGovernance config)
          ↓
Evaluate() → checkSkillCatalogs()
          ↓
┌─────────────────────────────────────────┐
│ SCORING PHASE 1: Metadata Checks       │
│ (SKL-001 to SKL-008)                   │
│                                         │
│ - Check version field (SKL-001)        │
│ - Check repository URL (SKL-003)       │
│ - Check labels (SKL-004)                │
│ - Check description (SKL-006)          │
│ - Check environment (SKL-007)          │
│ etc.                                    │
│                                         │
│ Result: 0-100 base score               │
│ Deductions: 5-25 points per finding    │
└─────────────────────────────────────────┘
          ↓
┌─────────────────────────────────────────┐
│ SCORING PHASE 2: Content Scanning      │
│ (SKL-SEC-001 to SKL-SEC-006)           │
│ ← USES mcp-governance-skill-patterns   │
│                                         │
│ scanSkillRepo() calls patternLoader    │
│ patternLoader reads ConfigMap files:   │
│                                         │
│ /etc/mcp-governance/skill-patterns/    │
│ ├─ prompt-injection ──→ SKL-SEC-001   │
│ ├─ privilege-escalation ─→ SKL-SEC-002│
│ ├─ data-exfiltration ──→ SKL-SEC-003  │
│ ├─ credential-harvesting ─→ SKL-SEC-004
│ ├─ scope-creep ──→ SKL-SEC-005        │
│ └─ safety-guardrails ─→ SKL-SEC-006   │
│                                         │
│ For each GitHub file:                  │
│   Match patterns → Generate findings   │
│   Findings determine score deductions  │
│                                         │
│ Result: Final 0-100 score              │
│ Deductions: 15-40 points per finding   │
└─────────────────────────────────────────┘
          ↓
┌─────────────────────────────────────────┐
│ FINAL STATUS ASSIGNED                  │
│                                         │
│ Score >= 80  → status: "pass" ✅       │
│ 50 <= Score < 80 → status: "warning" ⚠️│
│ Score < 50   → status: "fail" ❌       │
└─────────────────────────────────────────┘
          ↓
Updates GovernanceEvaluation.status.skillCatalogScores[]
```

### How the ConfigMap Works

**File:** `deploy/k8s/skill-patterns-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-governance-skill-patterns
  namespace: mcp-governance
data:
  # Each key is a category of security patterns
  prompt-injection: |
    ignore previous instructions
    bypass safety
    jailbreak
    # ... 20+ patterns ...
  
  privilege-escalation: |
    sudo
    run as root
    chmod 777
    # ... 18+ patterns ...
  
  data-exfiltration: |
    exfiltrate
    webhook.site
    send data to external
    # ... 15+ patterns ...
  
  credential-harvesting: |
    steal credentials
    harvest passwords
    # ... 14+ patterns ...
  
  scope-creep: |
    data: deploy, provision infrastructure
    analytics: delete database
    # Category-specific scope rules ...
  
  safety-guardrails: |
    database: confirmation required, dry run
    infra: change management, dry run
    admin: admin access required, audit logged
    # Category requirements ...
```

### Kubernetes Integration

**File:** `deploy/k8s/deployment.yaml`

```yaml
spec:
  containers:
  - name: controller
    volumeMounts:
    - name: skill-patterns
      mountPath: /etc/mcp-governance/skill-patterns
      readOnly: true
  volumes:
  - name: skill-patterns
    configMap:
      name: mcp-governance-skill-patterns
      optional: true  # Non-fatal if missing
```

**What Happens:**
1. Kubernetes mounts ConfigMap data as files in the pod
2. File structure:
   ```
   /etc/mcp-governance/skill-patterns/
   ├─ prompt-injection          (file with patterns)
   ├─ privilege-escalation
   ├─ data-exfiltration
   ├─ credential-harvesting
   ├─ scope-creep
   └─ safety-guardrails
   ```

### Pattern Loading During Evaluation

**File:** `controller/pkg/skillscanner/scanner.go`

```go
type PatternLoader struct {
    mountPath string                          // "/etc/mcp-governance/skill-patterns"
    ttl time.Duration                         // 30 seconds default
    cache *PatternSet                         // Cached patterns
    cacheTime time.Time                       // Last load time
}

func (pl *PatternLoader) Load() (*PatternSet, error) {
    // Check cache TTL
    if time.Since(pl.cacheTime) < pl.ttl {
        return pl.cache, nil  // Return cached
    }
    
    // Re-read from filesystem (HOT-RELOAD! ✨)
    patterns := ParsePatternSet(readConfigMapFiles(pl.mountPath))
    pl.cache = patterns
    pl.cacheTime = time.Now()
    return patterns, nil
}
```

**Key Feature: HOT-RELOAD**
- Update ConfigMap → Patterns reloaded within 30 seconds
- No pod restart needed!
- Graceful fallback if ConfigMap deleted

### Pattern Matching & Scoring

**File:** `controller/pkg/skillscanner/scanner.go`

```go
func ScanContent(filePath, content, patterns, category) {
    findings := []SkillCatalogFinding{}
    
    for _, pattern := range patterns[category] {  // ← From ConfigMap
        if strings.Contains(content, pattern) {
            finding := SkillCatalogFinding{
                CheckID: "SKL-SEC-001",  // Category determines check ID
                Severity: "Critical",
                Title: fmt.Sprintf("Pattern detected: %s", pattern),
                Remediation: "Review and remove pattern",
                FilePath: filePath,
                Line: lineNumber,
                MatchedPattern: pattern,
            }
            findings = append(findings, finding)
        }
    }
    return findings
}
```

### Score Computation

**File:** `controller/pkg/evaluator/evaluator_skills.go`

```go
func scoreSkillCatalog(skill, metaFindings, contentFindings, policy) SkillCatalogScore {
    score := 100  // Start at 100
    
    // Deduct for metadata findings (SKL-001..008)
    for _, f := range metaFindings {
        score -= f.SeverityPenalty()  // 5-25 points
    }
    
    // Deduct for content findings (SKL-SEC-001..006)
    for _, f := range contentFindings {
        // Critical patterns can force immediate fail
        if f.CheckID == "SKL-SEC-001" && policy.FailOnPromptInjection {
            return SkillCatalogScore{Score: 0, Status: "fail"}  // FAIL!
        }
        if f.CheckID == "SKL-SEC-002" && policy.FailOnPrivilegeEscalation {
            return SkillCatalogScore{Score: 0, Status: "fail"}  // FAIL!
        }
        
        // Otherwise deduct points
        score -= f.SeverityPenalty()  // 15-40 points
    }
    
    // Ensure score is in valid range
    if score < 0 { score = 0 }
    if score > 100 { score = 100 }
    
    // Assign status
    status := "pass"      // >= 80
    if score < 80 { status = "warning" }   // 50-79
    if score < 50 { status = "fail" }      // < 50
    
    return SkillCatalogScore{
        Score: score,
        Status: status,
        Findings: contentFindings,
    }
}
```

---

## Check ID Reference Table

### Metadata Checks (No ConfigMap Needed)

| ID | Severity | Deduction | What It Checks |
|----|----------|-----------|---|
| SKL-001 | Medium | -10 | version field is empty |
| SKL-002 | Low | -5 | repository.source is unknown |
| SKL-003 | Critical | -25 | repository.url uses HTTP (not HTTPS) |
| SKL-004 | Low | -5 | labels don't include resource-uid |
| SKL-005 | Low | -5 | category field is empty |
| SKL-006 | Low | -5 | description is < 20 characters |
| SKL-007 | High | -20 | environment=production with no version |
| SKL-008 | Medium | -10 | repository URL is personal GitHub account |

### Content Checks (ConfigMap Patterns)

| ID | Severity | Deduction | ConfigMap Key | What It Scans |
|----|----------|-----------|---|---|
| SKL-SEC-001 | Critical | -40 or FAIL | `prompt-injection` | GitHub files for jailbreak patterns |
| SKL-SEC-002 | Critical | -40 or FAIL | `privilege-escalation` | GitHub files for root/sudo patterns |
| SKL-SEC-003 | High | -30 | `data-exfiltration` | GitHub files for exfiltration patterns |
| SKL-SEC-004 | High | -30 | `credential-harvesting` | GitHub files for credential theft patterns |
| SKL-SEC-005 | Medium | -15 | `scope-creep` | Skill category vs. repository functionality mismatch |
| SKL-SEC-006 | Medium | -15 | `safety-guardrails` | Safety guardrail phrases in skill files |

---

## Summary: What Changed

### MCPGovernancePolicies
| Field | Type | Location | Purpose |
|-------|------|----------|---------|
| `skillGovernance` | object | spec | NEW - Configure skill governance behavior |
| `skillGovernance.enabled` | bool | spec | Enable/disable skill checks |
| `skillGovernance.scanRepoContent` | bool | spec | Enable/disable GitHub repository scanning |
| `skillGovernance.failOnPromptInjection` | bool | spec | Force fail if prompt injection found |
| `skillGovernance.failOnPrivilegeEscalation` | bool | spec | Force fail if privilege escalation found |
| `skillGovernance.allowedExternalDomains` | []string | spec | Skip exfiltration check for trusted domains |
| `skillGovernance.requireSafetyGuardrails` | []string | spec | Categories requiring safety phrases |

### GovernanceEvaluations
| Field | Type | Location | Purpose |
|-------|------|----------|---------|
| `skillCatalogScores` | []object | status | NEW - Array of skill scores |
| `skillCatalogScores[].name` | string | status | Skill catalog CR name |
| `skillCatalogScores[].score` | int | status | Governance score (0-100) |
| `skillCatalogScores[].status` | string | status | pass/warning/fail |
| `skillCatalogScores[].findings` | []object | status | Array of governance findings |

### New Kubernetes Resources
| Resource | File | Purpose |
|----------|------|---------|
| ConfigMap `mcp-governance-skill-patterns` | `deploy/k8s/skill-patterns-configmap.yaml` | NEW - Pattern definitions for content scanning |

---

## Deployment Checklist

- [ ] Apply CRD: `kubectl apply -f deploy/crds/governance-crds.yaml`
- [ ] Create ConfigMap: `kubectl apply -f deploy/k8s/skill-patterns-configmap.yaml`
- [ ] Create MCPGovernancePolicy with skillGovernance section
- [ ] Update deployment with volumeMount: `kubectl apply -f deploy/k8s/deployment.yaml`
- [ ] Verify ConfigMap mounted: `kubectl exec <pod> -- ls /etc/mcp-governance/skill-patterns`
- [ ] Create test SkillCatalog CR
- [ ] Wait for evaluation (default: 5 minutes)
- [ ] Check GovernanceEvaluation status: `kubectl get governanceevaluations -o yaml | grep skillCatalogScores`

---

## Documentation Files Created

I've created three comprehensive documentation files for you:

1. **`SKILL_GOVERNANCE_CRD_CONFIGURATION.md`**
   - Complete CRD structure reference
   - Go type definitions
   - Discovery & parsing logic
   - Scoring thresholds & check reference

2. **`SKILL_GOVERNANCE_VISUAL_SUMMARY.md`**
   - Before/after YAML comparisons
   - Visual evaluation pipeline
   - ConfigMap integration details
   - File changes index

3. **`SKILL_GOVERNANCE_SCORING_ARCHITECTURE.md`**
   - Complete data flow diagram
   - Score computation breakdown
   - Hot-reload mechanism
   - Performance & caching strategy

**All files are in the project root directory.**

