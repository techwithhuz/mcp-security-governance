# Skill Governance Scoring Architecture - Quick Reference

## Where is Skill Scoring Related to mcp-governance-skill-patterns ConfigMap?

### Complete Data Flow

```
┌────────────────────────────────────────────────────────────────┐
│ Step 1: Policy Configuration                                  │
│ ─────────────────────────────────────────────────────────────  │
│ MCPGovernancePolicy.spec.skillGovernance:                      │
│   enabled: true                                                │
│   scanRepoContent: true          ← Enables pattern scanning    │
│   scanCacheTTLMinutes: 60                                      │
│   PatternMountPath: "/etc/mcp-governance/skill-patterns"       │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 2: Kubernetes Mounts ConfigMap to Pod                    │
│ ─────────────────────────────────────────────────────────────  │
│ File: deploy/k8s/deployment.yaml                              │
│                                                                │
│ volumes:                                                       │
│   - name: skill-patterns                                      │
│     configMap:                                                │
│       name: mcp-governance-skill-patterns                     │
│       optional: true                                          │
│                                                                │
│ volumeMounts:                                                 │
│   - name: skill-patterns                                      │
│     mountPath: /etc/mcp-governance/skill-patterns             │
│     readOnly: true                                            │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 3: ConfigMap Data Structure                              │
│ ─────────────────────────────────────────────────────────────  │
│ File: deploy/k8s/skill-patterns-configmap.yaml                │
│                                                                │
│ data:                                                          │
│   prompt-injection:           (Pattern strings for SKL-SEC-001)│
│     ignore previous instructions                              │
│     disregard your instructions                               │
│     bypass safety                                             │
│     ...                                                       │
│                                                                │
│   privilege-escalation:       (Pattern strings for SKL-SEC-002)│
│     sudo                                                      │
│     run as root                                               │
│     escalate privileges                                       │
│     ...                                                       │
│                                                                │
│   data-exfiltration:          (Pattern strings for SKL-SEC-003)│
│     exfiltrate                                                │
│     send data to external                                     │
│     webhook.site                                              │
│     ...                                                       │
│                                                                │
│   credential-harvesting:      (Pattern strings for SKL-SEC-004)│
│     steal credentials                                         │
│     harvest passwords                                         │
│     ...                                                       │
│                                                                │
│   scope-creep:                (Category rules for SKL-SEC-005) │
│     data: deploy, provision infrastructure                    │
│     analytics: delete database, drop table                    │
│     ...                                                       │
│                                                                │
│   safety-guardrails:          (Guardrail phrases for SKL-SEC-006)
│     database: confirmation required, dry run                  │
│     infra: change management, dry run                         │
│     admin: admin access required, audit logged                │
│     ...                                                       │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 4: PatternLoader Reads from Filesystem                   │
│ ─────────────────────────────────────────────────────────────  │
│ File: controller/pkg/skillscanner/scanner.go                  │
│                                                                │
│ type PatternLoader struct {                                   │
│   mountPath string    // "/etc/mcp-governance/skill-patterns"  │
│   ttl time.Duration   // 30s default                          │
│   cache *PatternSet                                           │
│   cacheTime time.Time                                         │
│ }                                                              │
│                                                                │
│ func (pl *PatternLoader) Load() (*PatternSet, error) {        │
│   // Check TTL cache                                          │
│   if time.Since(pl.cacheTime) < pl.ttl {                      │
│     return pl.cache, nil  // Return cached patterns           │
│   }                                                            │
│                                                                │
│   // Re-read from /etc/mcp-governance/skill-patterns/*         │
│   patterns := ParsePatternSet(readFiles(pl.mountPath))        │
│   pl.cache = patterns                                         │
│   pl.cacheTime = time.Now()                                   │
│   return patterns, nil  // HOT-RELOAD! ✨                    │
│ }                                                              │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 5: Pattern Scanning During Evaluation                    │
│ ─────────────────────────────────────────────────────────────  │
│ File: controller/pkg/evaluator/evaluator_skills.go            │
│                                                                │
│ func checkSkillCatalogs(state, policy, patternLoader) {       │
│   for _, skill := range state.SkillCatalogs {                 │
│     // (A) Metadata checks (SKL-001..SKL-008)                 │
│     metaFindings := checkSkillMetadata(skill)                 │
│                                                                │
│     // (B) Content scanning (SKL-SEC-001..SEC-006)            │
│     contentFindings := scanSkillRepo(                         │
│       skill, policy,                                          │
│       patternLoader.Load()  ← LOADS PATTERNS HERE!            │
│     )                                                          │
│                                                                │
│     // (C) Compute score (0-100)                             │
│     score := scoreSkillCatalog(                               │
│       skill, metaFindings, contentFindings, policy            │
│     )                                                          │
│                                                                │
│     result.SkillCatalogScores = append(..., score)            │
│   }                                                            │
│   return result                                               │
│ }                                                              │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 6: Pattern Matching Generates Findings                   │
│ ─────────────────────────────────────────────────────────────  │
│ File: controller/pkg/skillscanner/scanner.go                  │
│                                                                │
│ func ScanContent(filePath, content, patterns, category) {     │
│   for _, pattern := range patterns[category] {               │
│     if strings.Contains(content, pattern) {                   │
│       finding := SkillCatalogFinding{                         │
│         CheckID: "SKL-SEC-001",  // e.g., Prompt Injection   │
│         Severity: "Critical",                                 │
│         Category: "Security",                                 │
│         Title: "Prompt injection pattern detected",           │
│         Remediation: "Remove pattern and review usage",       │
│         FilePath: filePath,                                   │
│         Line: lineNumber,                                     │
│         MatchedPattern: pattern,                              │
│       }                                                       │
│       findings = append(findings, finding)                    │
│     }                                                          │
│   }                                                            │
│   return findings                                             │
│ }                                                              │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 7: Score Computed from Findings                          │
│ ─────────────────────────────────────────────────────────────  │
│ File: controller/pkg/evaluator/evaluator_skills.go            │
│                                                                │
│ Starting Score: 100                                           │
│                                                                │
│ For each Finding:                                             │
│   - If SKL-SEC-001 (Prompt Injection, Critical):              │
│     → If failOnPromptInjection=true: score = 0 (FAIL!)        │
│     → Else: score -= 40 points                                │
│                                                                │
│   - If SKL-SEC-002 (Privilege Escalation, Critical):          │
│     → If failOnPrivilegeEscalation=true: score = 0 (FAIL!)    │
│     → Else: score -= 40 points                                │
│                                                                │
│   - If SKL-001..SKL-008 (Metadata, varies):                   │
│     → score -= severity_penalty (5-25 points)                 │
│                                                                │
│   - If in allowedExternalDomains:                             │
│     → Suppress SKL-SEC-003 deduction                          │
│                                                                │
│   - If missing guardrails in required category:               │
│     → score -= 15 points (SKL-SEC-006)                        │
│                                                                │
│ Final Status:                                                 │
│   if score >= 80: status = "pass" ✅                          │
│   if 50 <= score < 80: status = "warning" ⚠️                  │
│   if score < 50: status = "fail" ❌                           │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 8: Results Written to GovernanceEvaluation Status        │
│ ─────────────────────────────────────────────────────────────  │
│ File: controller/pkg/discovery/discovery.go (UpdateEvaluationStatus)
│                                                                │
│ GovernanceEvaluation.status.skillCatalogScores: [              │
│   {                                                           │
│     name: "anthoropic-skills-1-0-0",                          │
│     namespace: "default",                                     │
│     version: "1.0.0",                                         │
│     category: "data-processing",                              │
│     repoURL: "https://github.com/anthropic/skills",           │
│     score: 95,                                                │
│     status: "pass",                                           │
│     scannedFiles: 15,                                         │
│     findings: [                                               │
│       {                                                       │
│         checkID: "SKL-004",                                   │
│         severity: "Low",                                      │
│         title: "Missing resource-uid label",                  │
│         remediation: "Add resource-uid label"                 │
│       }                                                       │
│     ]                                                         │
│   }                                                           │
│ ]                                                             │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│ Step 9: Dashboard Displays Results                            │
│ ─────────────────────────────────────────────────────────────  │
│ File: dashboard/src/components/SkillCatalog.tsx               │
│                                                                │
│ ┌─ Skill Catalogs Tab (Score 95, Status PASS) ──────┐       │
│ │                                                    │       │
│ │ Summary:  1 Total  |  1 Pass  |  0 Warning  |     │       │
│ │                                                    │       │
│ │ ┌──────────────────────────────────────────────┐  │       │
│ │ │ ✅ anthoropic-skills-1-0-0      Score: 95   │  │       │
│ │ │ Version: 1.0.0                              │  │       │
│ │ │ Category: data-processing                   │  │       │
│ │ │ Findings: 1 (1L)                            │  │       │
│ │ │                                              │  │       │
│ │ │ ▼ FINDINGS                                   │  │       │
│ │ │   🔵 SKL-004 (Low) - Metadata               │  │       │
│ │ │   Missing resource-uid label                │  │       │
│ │ │   → Add resource-uid label                  │  │       │
│ │ └──────────────────────────────────────────────┘  │       │
│ │                                                    │       │
│ └────────────────────────────────────────────────────┘       │
└────────────────────────────────────────────────────────────────┘
```

---

## Scoring Breakdown by Check ID

### Metadata Checks (SKL-001 to SKL-008)

| Check ID | Severity | Category | Deduction | Field | Pattern |
|----------|----------|----------|-----------|-------|---------|
| SKL-001 | Medium | Metadata | -10 | version | Empty/missing |
| SKL-002 | Low | Metadata | -5 | repository.source | Unknown |
| SKL-003 | Critical | Security | -25 | repository.url | Starts with `http://` |
| SKL-004 | Low | Metadata | -5 | labels | No `resource-uid` |
| SKL-005 | Low | Metadata | -5 | category | Empty/missing |
| SKL-006 | Low | Metadata | -5 | description | Length < 20 chars |
| SKL-007 | High | Metadata | -20 | environment | `production` + no version |
| SKL-008 | Medium | Metadata | -10 | repository.url | Personal GitHub account |

### Content Checks (SKL-SEC-001 to SKL-SEC-006)

| Check ID | Severity | Category | Triggers | ConfigMap Key | Deduction |
|----------|----------|----------|----------|---------------|-----------|
| SKL-SEC-001 | Critical | Security | Prompt injection patterns | `prompt-injection` | -40 or FAIL |
| SKL-SEC-002 | Critical | Security | Privilege escalation patterns | `privilege-escalation` | -40 or FAIL |
| SKL-SEC-003 | High | Security | Data exfiltration patterns | `data-exfiltration` | -30 |
| SKL-SEC-004 | High | Security | Credential harvesting patterns | `credential-harvesting` | -30 |
| SKL-SEC-005 | Medium | Security | Category scope creep | `scope-creep` | -15 |
| SKL-SEC-006 | Medium | Safety | Missing guardrails | `safety-guardrails` | -15 |

### Pattern Sources in ConfigMap

```
mcp-governance-skill-patterns ConfigMap
│
├─ prompt-injection (20+ patterns)
│  ├─ "ignore previous instructions"
│  ├─ "disregard your instructions"
│  ├─ "bypass safety"
│  ├─ "jailbreak"
│  └─ ... triggers SKL-SEC-001
│
├─ privilege-escalation (18+ patterns)
│  ├─ "sudo"
│  ├─ "run as root"
│  ├─ "chmod 777"
│  └─ ... triggers SKL-SEC-002
│
├─ data-exfiltration (15+ patterns)
│  ├─ "exfiltrate"
│  ├─ "webhook.site"
│  ├─ "send data to external"
│  └─ ... triggers SKL-SEC-003
│
├─ credential-harvesting (14+ patterns)
│  ├─ "steal credentials"
│  ├─ "harvest passwords"
│  └─ ... triggers SKL-SEC-004
│
├─ scope-creep (category rules)
│  ├─ "data: deploy, provision infrastructure"
│  ├─ "analytics: delete database"
│  └─ ... triggers SKL-SEC-005
│
└─ safety-guardrails (category requirements)
   ├─ "database: confirmation required, dry run"
   ├─ "infra: change management, dry run"
   ├─ "admin: admin access required, audit logged"
   └─ ... triggers SKL-SEC-006
```

---

## Policy Configuration Impact on Scoring

### Configuration Option: `failOnPromptInjection`

```
If DISABLED (false):
  ┌─────────────────────────────────────────┐
  │ SKL-SEC-001 Pattern Found               │
  │ → Deduct 40 points from score           │
  │ → Example: 100 - 40 = 60 (warning)      │
  └─────────────────────────────────────────┘

If ENABLED (true):  ← DEFAULT
  ┌─────────────────────────────────────────┐
  │ SKL-SEC-001 Pattern Found               │
  │ → Score = 0 IMMEDIATELY (FAIL!)         │
  │ → No matter what other checks passed    │
  │ → Status: "fail" ❌                     │
  └─────────────────────────────────────────┘
```

### Configuration Option: `scanRepoContent`

```
If DISABLED (false):  ← DEFAULT
  ┌─────────────────────────────────────────┐
  │ Only Metadata Checks Run (SKL-001..008) │
  │ Content Checks SKIPPED (SEC-001..006)   │
  │ No GitHub API calls                     │
  │ Score based only on metadata (max 75)   │
  └─────────────────────────────────────────┘

If ENABLED (true):
  ┌─────────────────────────────────────────┐
  │ Both Metadata + Content Checks Run      │
  │ Fetch all files from GitHub repo        │
  │ Pattern-match each file (cached 60m)    │
  │ Score based on all findings (0-100)     │
  │ Requires: githubToken for private repos │
  └─────────────────────────────────────────┘
```

### Configuration Option: `allowedExternalDomains`

```
If SKL-SEC-003 (Data Exfiltration) Pattern Found:

  For domain in allowedExternalDomains:
    if exfiltrationURL matches domain:
      → SUPPRESS finding (no deduction)
      
  For domain NOT in allowedExternalDomains:
    → Score -= 30 points

Example:
  allowedExternalDomains: ["github.com", "api.example.com"]
  
  If skill has: "send data to github.com"
    → ALLOWED (no deduction)
  
  If skill has: "send data to unknown.com"
    → BLOCKED (-30 points, SKL-SEC-003 finding)
```

---

## Hot-Reload in Action

### Scenario: Add New Prompt Injection Pattern

**Current State:**
- Skill has text: "ignore all previous instructions"
- Pattern exists in ConfigMap → detected → SKL-SEC-001 triggered

**Admin Updates ConfigMap:**
```bash
kubectl edit configmap mcp-governance-skill-patterns -n mcp-governance

# Add new pattern:
prompt-injection: |
  ignore previous instructions
  disregard your instructions
  ... existing patterns ...
  new_jailbreak_attempt  ← NEW PATTERN ADDED
```

**What Happens Next:**

```
Time 0:00 - TTL Cache Valid
  └─ PatternLoader.Load() returns cached patterns
     (new_jailbreak_attempt not yet loaded)

Time 0:30+ - TTL Cache Expired
  └─ PatternLoader.Load() re-reads from /etc/mcp-governance/skill-patterns
     (mounted ConfigMap updated automatically by Kubernetes)
     └─ Loads new_jailbreak_attempt pattern! ✨

Time 0:35 - Next Evaluation Runs
  └─ checkSkillCatalogs() calls patternLoader.Load()
  └─ Matches against NEW patterns
  └─ If skill has "new_jailbreak_attempt" text:
     └─ NEW finding generated automatically
     └─ Score recalculated with new finding
     └─ No pod restart needed! 🎉
```

**Benefits:**
- ✅ Update security patterns without redeploying controller
- ✅ Changes take effect within 30 seconds (TTL)
- ✅ Backward compatible (optional ConfigMap)
- ✅ Graceful fallback if ConfigMap deleted

---

## Integration Points

### How Each Component Uses Patterns:

| Component | File | Reads Patterns | Purpose |
|-----------|------|-----------------|---------|
| **PatternLoader** | `skillscanner/scanner.go` | `/etc/mcp-governance/skill-patterns/*` | Load & cache patterns |
| **ScanContent()** | `skillscanner/scanner.go` | From PatternLoader | Match against file content |
| **scanSkillRepo()** | `evaluator_skills.go` | Via ScanContent() | Scan GitHub repo files |
| **checkSkillCatalogs()** | `evaluator_skills.go` | Via scanSkillRepo() | Generate findings & score |
| **Evaluate()** | `evaluator.go` | Via checkSkillCatalogs() | Compute overall score |
| **UpdateEvaluationStatus()** | `discovery.go` | Via Evaluate() result | Write to GovernanceEvaluation CR |
| **Dashboard API** | `cmd/api/main.go` | Via GET /api/governance/skill-catalogs | Display in UI |

---

## Error Handling & Fallbacks

### If ConfigMap Is Missing

```go
// In PatternLoader.Load()
if _, err := os.Stat(pl.mountPath); err != nil {
  // ConfigMap not mounted or doesn't exist
  log.Warnf("Pattern mount path not found: %v", err)
  
  // Fallback to built-in patterns
  pl.cache = GetDefaultPatternSet()
  return pl.cache, nil  // Continue without error
}
```

**Result:**
- ✅ Controller doesn't crash
- ✅ Uses sensible default patterns
- ✅ Admin can still deploy without ConfigMap
- ✅ Patterns can be injected later via ConfigMap

### If Skill Repo Is Unreachable

```go
// In scanSkillRepo()
files, err := FetchSkillFiles(repoURL, token)
if err != nil {
  // GitHub API call failed (rate limit, network, etc.)
  log.Warnf("Failed to scan %s: %v", repoURL, err)
  
  return []SkillCatalogFinding{
    {
      CheckID: "SKL-SEC-SCAN-ERROR",
      Severity: "Medium",
      Title: "Repository scan failed",
      Remediation: "Check GitHub token, rate limits, or repository access"
    }
  }, err  // Score reduced, but evaluation continues
}
```

**Result:**
- ✅ Metadata checks still run (SKL-001..008)
- ✅ Content scan marked as failed
- ✅ Finding recorded for admin review
- ✅ Evaluation completes without crashing

---

## Performance Notes

### Caching Strategy

```
Pattern Loading TTL:
  ├─ First call: Load from filesystem
  ├─ Next 30 seconds: Return cached PatternSet
  ├─ After 30 seconds: Re-read and update cache
  └─ Worst case: ~500ms per reload (file I/O)

Scan Result Caching:
  ├─ Repository scans cached for 60 minutes (configurable)
  ├─ Cache key: repo URL + commit hash
  ├─ Prevents GitHub API rate limit exhaustion
  └─ Trade-off: Latest patterns applied within 30s, results stale up to 60m

Evaluation Cycle:
  ├─ Full cluster scan: 2-5 seconds (depends on resource count)
  ├─ Skill catalog scanning: 500ms - 5 seconds (depends on repo size + network)
  ├─ Default interval: 5 minutes
  └─ Can be customized via policy.scanInterval
```

---

## Testing Checklist

- [ ] ConfigMap mounted to controller pod
- [ ] PatternLoader successfully loads patterns
- [ ] Patterns have patterns (files non-empty)
- [ ] Create test SkillCatalog CR
- [ ] Run Evaluate() manually or wait for scan
- [ ] Verify findings in GovernanceEvaluation status
- [ ] Update ConfigMap with new pattern
- [ ] Wait 30+ seconds for cache TTL
- [ ] Re-run Evaluate() and verify new pattern detected
- [ ] Check dashboard displays scores correctly

