# Quick Reference Card - Skill Governance Configuration

## 30-Second Overview

**Q: Is MCPGovernancePolicies updated?**
✅ YES - New optional `skillGovernance` section in spec

**Q: Is GovernanceEvaluations updated?**
✅ YES - New `skillCatalogScores[]` field in status

**Q: Where does scoring use the ConfigMap?**
✅ ConfigMap patterns → used by pattern scanner → generates findings → deduct points from score

---

## Configuration Examples

### Minimal MCPGovernancePolicy (Metadata Checks Only)
```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: default-policy
spec:
  # Existing fields...
  requireAgentGateway: true
  
  # Skill governance: metadata checks only (no repo scanning)
  skillGovernance:
    enabled: true
```

### Full MCPGovernancePolicy (All Checks + Repo Scanning)
```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-policy
spec:
  # Existing fields...
  requireAgentGateway: true
  
  # Skill governance: with repository scanning
  skillGovernance:
    enabled: true
    scanRepoContent: true                    # Enable GitHub scanning
    githubToken: ""                          # Set via environment
    scanCacheTTLMinutes: 60
    failOnPromptInjection: true              # Immediate fail
    failOnPrivilegeEscalation: true          # Immediate fail
    allowedExternalDomains:                  # Skip exfiltration check
      - github.com
      - api.internal.example.com
    requireSafetyGuardrails:                 # Must have guardrails
      - database
      - infra
      - admin
```

---

## Scoring Breakdown

### Starting Point
**Base Score: 100 points**

### Deductions

| Source | Check Type | Deduction | Enabled By |
|--------|-----------|-----------|------------|
| Missing version | Metadata (SKL-001) | -10 | Always |
| Missing labels | Metadata (SKL-004) | -5 | Always |
| HTTP repo URL | Metadata (SKL-003) | -25 | Always |
| Prompt injection pattern | Content (SKL-SEC-001) | -40 or FAIL | `scanRepoContent: true` |
| Privilege escalation pattern | Content (SKL-SEC-002) | -40 or FAIL | `scanRepoContent: true` |
| Data exfiltration pattern | Content (SKL-SEC-003) | -30 | `scanRepoContent: true` |
| Credential harvesting pattern | Content (SKL-SEC-004) | -30 | `scanRepoContent: true` |
| Missing safety guardrails | Content (SKL-SEC-006) | -15 | `scanRepoContent: true` |

### Final Status
```
Score >= 80     → ✅ PASS
50 <= Score < 80 → ⚠️ WARNING
Score < 50      → ❌ FAIL
```

---

## ConfigMap Pattern Files

**Location:** `/etc/mcp-governance/skill-patterns/`

| File | Check ID | Used By | Content |
|------|----------|---------|---------|
| `prompt-injection` | SKL-SEC-001 | `scanSkillRepo()` | Jailbreak keywords, "ignore instructions", etc. |
| `privilege-escalation` | SKL-SEC-002 | `scanSkillRepo()` | Sudo, chmod, root escalation keywords |
| `data-exfiltration` | SKL-SEC-003 | `scanSkillRepo()` | External domains, webhook URLs |
| `credential-harvesting` | SKL-SEC-004 | `scanSkillRepo()` | Credential theft patterns |
| `scope-creep` | SKL-SEC-005 | `scanSkillRepo()` | Category-specific forbidden keywords |
| `safety-guardrails` | SKL-SEC-006 | `scanSkillRepo()` | Required safety phrases per category |

---

## Key Files Reference

| File | Purpose | Lines |
|------|---------|-------|
| `deploy/crds/governance-crds.yaml` | CRD definitions | 1-240 (MCPGovernancePolicy), 241-489 (GovernanceEvaluation) |
| `deploy/k8s/skill-patterns-configmap.yaml` | Pattern definitions | 6 keys with pattern strings |
| `deploy/k8s/deployment.yaml` | Pod volumeMount config | Lines with `skill-patterns` volume |
| `controller/pkg/evaluator/evaluator.go` | Policy types | Line 438: `SkillGovernance` field |
| `controller/pkg/discovery/discovery.go` | Policy parsing | Lines 1240-1290 |
| `controller/pkg/discovery/discovery.go` | Status writer | Lines 1519-1550 |
| `controller/pkg/evaluator/evaluator_skills.go` | Scoring logic | All checks (SKL-001..SKL-SEC-006) |
| `controller/pkg/skillscanner/scanner.go` | Pattern loader | Hot-reload with TTL |

---

## Deployment Commands

```bash
# Apply CRD
kubectl apply -f deploy/crds/governance-crds.yaml

# Create ConfigMap
kubectl apply -f deploy/k8s/skill-patterns-configmap.yaml

# Update deployment with volumeMount
kubectl apply -f deploy/k8s/deployment.yaml

# Verify ConfigMap mounted
kubectl exec -n mcp-governance deployment/mcp-governance-controller -- \
  ls /etc/mcp-governance/skill-patterns/

# View evaluation results
kubectl get governanceevaluations -o yaml | grep -A 20 skillCatalogScores
```

---

## Scoring Flow Diagram

```
MCPGovernancePolicy (skillGovernance config)
          ↓
    DiscoverClusterState()
          ↓
    discoverSkillCatalogs()  ← Lists all SkillCatalog CRs
          ↓
    checkSkillCatalogs()
    ├─ checkSkillMetadata()  [SKL-001..008] → 0-100 base score
    └─ scanSkillRepo()  [SKL-SEC-001..006]
       ├─ PatternLoader.Load() ← Reads from ConfigMap!
       ├─ GitHub API fetch
       └─ Pattern match → Findings → Deductions
          ↓
    scoreSkillCatalog(findings) → 0-100 final score + "pass"/"warning"/"fail"
          ↓
    UpdateEvaluationStatus() → Write to GovernanceEvaluation.status.skillCatalogScores[]
```

---

## Configuration Decision Tree

```
Do you want to scan SkillCatalog resources?
│
├─ NO
│  └─ Skip skillGovernance section entirely
│     └─ Only existing MCP governance checks run
│
├─ YES (Metadata Only)
│  └─ skillGovernance:
│       enabled: true
│       (scanRepoContent defaults to false)
│     └─ Runs SKL-001..SKL-008 metadata checks
│     └─ Example: missing version, HTTP repo URL, etc.
│
└─ YES (Metadata + Content)
   └─ skillGovernance:
        enabled: true
        scanRepoContent: true
      └─ Runs both metadata + content checks
      └─ Requires ConfigMap mcp-governance-skill-patterns
      └─ Needs GitHub token for private repos
      └─ Can force fail on critical patterns:
         ├─ failOnPromptInjection: true
         └─ failOnPrivilegeEscalation: true
```

---

## Hot-Reload in 3 Steps

1. **Edit ConfigMap**
   ```bash
   kubectl edit configmap mcp-governance-skill-patterns -n mcp-governance
   ```

2. **Save (Kubernetes auto-updates pod filesystem)**

3. **Next evaluation (within 30 seconds)**
   ```
   PatternLoader TTL expires → Re-reads patterns from disk → 
   New patterns applied to scanSkillRepo() → Score updated
   ```

✨ **No pod restart needed!**

---

## Backward Compatibility

| Component | Before | After | Breaking? |
|-----------|--------|-------|-----------|
| MCPGovernancePolicy | No skillGovernance | Optional skillGovernance | ❌ No |
| GovernanceEvaluation | No skillCatalogScores | Optional skillCatalogScores | ❌ No |
| Existing policies | Still work | Skip skill checks by default | ❌ No |
| API endpoints | No `/skill-catalogs` | `/api/governance/skill-catalogs` added | ❌ No |

✅ **Fully backward compatible - safe to deploy**

---

## Troubleshooting

### ConfigMap Not Mounted
```bash
# Check if ConfigMap exists
kubectl get configmap mcp-governance-skill-patterns -n mcp-governance

# If missing, create it
kubectl apply -f deploy/k8s/skill-patterns-configmap.yaml

# Verify in pod
kubectl exec -n mcp-governance deployment/mcp-governance-controller -- \
  ls -la /etc/mcp-governance/skill-patterns/
```

### Patterns Not Applied to Scores
```bash
# Verify scanRepoContent is true in policy
kubectl get mcpgovernancepolicy enterprise-mcp-policy -o yaml | grep scanRepoContent

# If false, update policy to enable repo scanning
kubectl edit mcpgovernancepolicy enterprise-mcp-policy
# Set: scanRepoContent: true
```

### Score Not Updated After ConfigMap Edit
```bash
# Wait 30 seconds (TTL cache)
# Then trigger re-evaluation
kubectl delete pod deployment/mcp-governance-controller
# Or wait for scheduled scan (default: 5m)
```

---

## Performance Notes

- **Pattern Loading TTL:** 30 seconds (cached, no filesystem I/O)
- **Scan Result Caching:** 60 minutes (prevents GitHub API rate limiting)
- **Evaluation Interval:** 5 minutes (default, configurable)
- **Skill Scan Time:** 500ms - 5 seconds per skill (depends on repo size)

---

## Status Output Example

```yaml
status:
  skillCatalogScores:
  - name: anthropic-skills-1-0-0
    namespace: default
    version: "1.0.0"
    category: "data-processing"
    score: 95
    status: pass
    findings:
    - checkID: SKL-004
      severity: Low
      title: Missing resource-uid label
```

---

## Document Map

1. **`SKILL_GOVERNANCE_CRD_CONFIGURATION.md`** - Full reference
2. **`SKILL_GOVERNANCE_VISUAL_SUMMARY.md`** - Before/after comparison
3. **`SKILL_GOVERNANCE_SCORING_ARCHITECTURE.md`** - Deep dive
4. **`SKILL_GOVERNANCE_CONFIG_SUMMARY.md`** - Executive summary
5. **`SKILL_GOVERNANCE_QUICK_REFERENCE.md`** - This file!

