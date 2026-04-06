# Skill Governance Documentation Index

## Complete Answer to Your Questions

### Question 1: Is there any configuration update in MCPGovernancePolicies CRD?

**✅ YES — New `skillGovernance` Section**

**Answer:** The MCPGovernancePolicy CRD now supports an optional `skillGovernance` configuration block at the spec level. This allows administrators to control skill governance behavior including:
- Enable/disable skill governance checks
- Toggle GitHub repository content scanning
- Force fail on critical security patterns
- Customize allowed external domains
- Require safety guardrails for specific skill categories

**Files Updated:**
- `controller/pkg/evaluator/evaluator.go` — Lines 438, 441-475
- `controller/pkg/discovery/discovery.go` — Lines 1240-1290 (parsing logic)
- `deploy/samples/governance-policy.yaml` — Example configuration

**CRD:** No schema changes needed in `deploy/crds/governance-crds.yaml` — the field is parsed dynamically from the spec object.

---

### Question 2: Is there any configuration update in GovernanceEvaluations CRD?

**✅ YES — New `skillCatalogScores[]` Field in Status**

**Answer:** The GovernanceEvaluation CRD now includes a new `skillCatalogScores` array in the status section. This array contains:
- Per-skill governance scores (0-100)
- Pass/warning/fail status for each skill
- Detailed findings with check IDs and remediation
- Number of files scanned from repository

**Files Updated:**
- `deploy/crds/governance-crds.yaml` — Lines 380+
- `controller/pkg/evaluator/evaluator.go` — Lines ~200-230 (type definitions)
- `controller/pkg/discovery/discovery.go` — Lines 1519-1550 (status writer)

---

### Question 3: Where is the scoring related to the mcp-governance-skill-patterns ConfigMap?

**✅ ConfigMap Contains Patterns Used for Security Scanning**

**Answer:** The `mcp-governance-skill-patterns` ConfigMap contains pattern definitions (6 files) that are:
1. Mounted as `/etc/mcp-governance/skill-patterns/` in the controller pod
2. Hot-loaded by `PatternLoader` (30-second TTL cache)
3. Applied by `scanSkillRepo()` during content scanning (SKL-SEC-001 through SKL-SEC-006)
4. Matched against GitHub repository files to generate findings
5. Findings are converted to score deductions (15-40 points each)
6. Final score written to `GovernanceEvaluation.status.skillCatalogScores[]`

**Files Updated:**
- `deploy/k8s/skill-patterns-configmap.yaml` — Pattern definitions
- `deploy/k8s/deployment.yaml` — volumeMount configuration
- `controller/pkg/skillscanner/scanner.go` — Pattern loader & matching
- `controller/pkg/skillscanner/patterns.go` — Pattern set definitions
- `controller/pkg/evaluator/evaluator_skills.go` — Scoring logic

---

## Documentation Files (All in Root Directory)

### 1. **SKILL_GOVERNANCE_QUICK_REFERENCE.md** ⭐ START HERE
**File Size:** 8.8K  
**Best For:** Quick lookup, deployment commands, decision tree  
**Contains:**
- 30-second overview
- Configuration examples (minimal vs. full)
- Scoring breakdown table
- Hot-reload instructions
- Troubleshooting guide

**Read this if you:** Want quick answers and command examples

---

### 2. **SKILL_GOVERNANCE_CONFIG_SUMMARY.md** ⭐ RECOMMENDED
**File Size:** 15K  
**Best For:** High-level understanding, executive summary  
**Contains:**
- Detailed answers to all 3 questions
- Configuration examples with explanations
- Check ID reference table
- Summary of what changed
- Deployment checklist

**Read this if you:** Want complete answers with examples

---

### 3. **SKILL_GOVERNANCE_CRD_CONFIGURATION.md** 📚 COMPREHENSIVE
**File Size:** 19K  
**Best For:** CRD structure, type definitions, detailed reference  
**Contains:**
- MCPGovernancePolicies policy structure (before/after)
- GovernanceEvaluation status structure (before/after)
- Go type definitions for all new structures
- Discovery & parsing flow
- Pattern ConfigMap integration
- Deployment checklist
- File index

**Read this if you:** Need complete CRD reference

---

### 4. **SKILL_GOVERNANCE_VISUAL_SUMMARY.md** 🎨 VISUAL
**File Size:** 22K  
**Best For:** Visual learners, before/after comparison  
**Contains:**
- Side-by-side YAML before/after
- Visual evaluation pipeline
- ConfigMap integration diagram
- Complete file changes summary
- Performance notes
- Key takeaways

**Read this if you:** Prefer visual comparisons

---

### 5. **SKILL_GOVERNANCE_SCORING_ARCHITECTURE.md** 🔬 DEEP DIVE
**File Size:** 29K  
**Best For:** Implementation details, scoring algorithm, architecture  
**Contains:**
- Complete data flow diagram (9 steps)
- Scoring breakdown by check ID
- Policy configuration impact on scoring
- Hot-reload mechanism in detail
- Error handling & fallbacks
- Performance & caching strategy
- Testing checklist

**Read this if you:** Want to understand the implementation deeply

---

## Quick Navigation

**If you want to know...**

| Question | File | Heading |
|----------|------|---------|
| What changed in MCPGovernancePolicy? | #2 | "Is there any configuration update in MCPGovernancePolicies CR?" |
| What changed in GovernanceEvaluation? | #2 | "Is there any configuration update in GovernanceEvaluations CR?" |
| How does the ConfigMap relate to scoring? | #2 | "Where is the scoring related to the mcp-governance-skill-patterns ConfigMap?" |
| Example MCPGovernancePolicy YAML? | #1 or #2 | "Configuration Examples" or "Example: Complete Configuration" |
| How do I deploy this? | #1 | "Deployment Commands" |
| What's the scoring algorithm? | #5 | "Scoring Breakdown by Check ID" |
| How does hot-reload work? | #5 | "Hot-Reload in Action" |
| What are all the check IDs? | #2 or #5 | "Check ID Reference" |
| Before/after YAML comparison? | #4 | "MCPGovernancePolicy (BEFORE/AFTER)" |
| File structure overview? | #3 | "File Index" |

---

## Key Concepts Summary

### 1. MCPGovernancePolicies.spec.skillGovernance
```yaml
skillGovernance:
  enabled: true                           # Activate skill checks
  scanRepoContent: true                   # GitHub repo scanning
  failOnPromptInjection: true             # Force fail if critical pattern found
  allowedExternalDomains: [...]           # Skip checks for trusted domains
  requireSafetyGuardrails: [...]          # Categories needing guardrails
```

### 2. GovernanceEvaluations.status.skillCatalogScores
```yaml
skillCatalogScores:
  - name: skill-name
    score: 95                             # 0-100 governance score
    status: "pass"                        # pass | warning | fail
    findings: [...]                       # Detected security issues
```

### 3. ConfigMap Pattern Flow
```
ConfigMap (6 files) → Mount in Pod → PatternLoader → scanSkillRepo() → 
Pattern Matching → Findings → Score Deduction → Final Score → Status
```

---

## Scoring at a Glance

### Starting Point
**Base Score: 100 points**

### Metadata Checks (Always Run)
- Missing fields: -5 to -10 points
- HTTP URLs: -25 points (CRITICAL)

### Content Checks (If scanRepoContent=true)
- Prompt injection (from ConfigMap): -40 or FAIL
- Privilege escalation (from ConfigMap): -40 or FAIL
- Other patterns (from ConfigMap): -15 to -30 points

### Final Status
- **Score >= 80:** ✅ PASS
- **50 <= Score < 80:** ⚠️ WARNING
- **Score < 50:** ❌ FAIL

---

## Implementation Files Changed

### Backend (Go)
| File | Component | Changes |
|------|-----------|---------|
| `evaluator.go` | Types | Added `SkillGovernancePolicy` struct |
| `discovery.go` | Parsing | Added `DiscoverGovernancePolicy()` logic for skillGovernance |
| `discovery.go` | Status Writer | Added skill scores to `UpdateEvaluationStatus()` |
| `evaluator_skills.go` | **NEW** | All skill governance checks (SKL-001..SKL-SEC-006) |
| `skillscanner/scanner.go` | **NEW** | Pattern loader & hot-reload mechanism |
| `skillscanner/patterns.go` | **NEW** | Pattern set definitions |

### Kubernetes (YAML)
| File | Changes |
|------|---------|
| `deploy/k8s/deployment.yaml` | Added volumeMount for skill-patterns ConfigMap |
| `deploy/k8s/skill-patterns-configmap.yaml` | **NEW** — 6 pattern keys |
| `deploy/crds/governance-crds.yaml` | Added skillCatalogScores schema to GovernanceEvaluation |

### Dashboard (TypeScript)
| File | Changes |
|------|---------|
| `dashboard/src/lib/types.ts` | Added SkillCatalogScore types |
| `dashboard/src/components/SkillCatalog.tsx` | **NEW** — Display skill scores |
| `dashboard/src/app/page.tsx` | Added SkillCatalogs tab |
| `dashboard/src/app/api/governance/[...path]/route.ts` | **NEW** — API proxy |

---

## Backward Compatibility

✅ **Fully Backward Compatible**

- MCPGovernancePolicy: skillGovernance section is optional
- GovernanceEvaluation: skillCatalogScores field is optional
- Existing policies: Continue to work without changes
- No breaking changes to existing APIs

---

## Check ID Reference

### Metadata Checks (SKL-001 to SKL-008)
Checked automatically on all SkillCatalog CRs. No configuration needed.

Example findings:
- SKL-001: Missing version field (-10 points)
- SKL-003: HTTP repository URL (-25 points)
- SKL-004: Missing resource-uid label (-5 points)

### Content Checks (SKL-SEC-001 to SKL-SEC-006)
Only run if `skillGovernance.scanRepoContent = true`. Patterns from ConfigMap.

Example findings:
- SKL-SEC-001: Prompt injection pattern found (-40 points or FAIL)
- SKL-SEC-002: Privilege escalation pattern found (-40 points or FAIL)
- SKL-SEC-006: Missing safety guardrails (-15 points)

---

## Deployment Steps

```bash
# 1. Apply CRD
kubectl apply -f deploy/crds/governance-crds.yaml

# 2. Create ConfigMap
kubectl apply -f deploy/k8s/skill-patterns-configmap.yaml

# 3. Update deployment with volumeMount
kubectl apply -f deploy/k8s/deployment.yaml

# 4. Create policy with skillGovernance (optional)
kubectl apply -f deploy/samples/governance-policy.yaml

# 5. Verify ConfigMap mounted
kubectl exec -n mcp-governance deployment/mcp-governance-controller -- \
  ls /etc/mcp-governance/skill-patterns/

# 6. Check results
kubectl get governanceevaluations -o yaml | grep -A 20 skillCatalogScores
```

---

## Next Steps

1. **Read:** Start with `SKILL_GOVERNANCE_QUICK_REFERENCE.md` (this file)
2. **Understand:** Review `SKILL_GOVERNANCE_CONFIG_SUMMARY.md` 
3. **Implement:** Use `SKILL_GOVERNANCE_CRD_CONFIGURATION.md` as reference
4. **Deploy:** Follow deployment steps above
5. **Troubleshoot:** Use `SKILL_GOVERNANCE_QUICK_REFERENCE.md` troubleshooting section

---

## Questions?

Refer to the appropriate documentation file:
- **Quick questions?** → QUICK_REFERENCE.md
- **How does it work?** → CONFIG_SUMMARY.md or VISUAL_SUMMARY.md
- **Deep dive?** → SCORING_ARCHITECTURE.md
- **API reference?** → CRD_CONFIGURATION.md

---

## Document Sizes & Reading Time

| File | Size | Time |
|------|------|------|
| QUICK_REFERENCE.md | 8.8K | 5-10 min |
| CONFIG_SUMMARY.md | 15K | 10-15 min |
| CRD_CONFIGURATION.md | 19K | 15-20 min |
| VISUAL_SUMMARY.md | 22K | 15-20 min |
| SCORING_ARCHITECTURE.md | 29K | 20-30 min |
| **TOTAL** | **~95K** | **~90 min** |

Pick the ones that match your needs!

