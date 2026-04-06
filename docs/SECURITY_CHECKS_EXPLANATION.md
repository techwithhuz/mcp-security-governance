# SkillCatalog Security Checks (SKL-SEC-001 to SKL-SEC-006) - Detailed Analysis

## Executive Summary

✅ **Security scanning is now ENABLED** (`scanRepoContent: true` in MCPGovernancePolicy)

Both skills have been re-scanned with security patterns. Here's what we discovered:

- **anthoropic-skills-1-0-0**: Score 95/100 - 1 finding (SKL-006 - description too short)
- **xlsx-1-0-0**: Score 100/100 - 0 findings (legitimate Anthropic repository content)

The reason there are no SKL-SEC-006 (Safety Guardrails) findings is because **neither skill is in a category that requires safety guardrails**.

---

## How SKL-SEC-006 (Safety Guardrails) Check Works

### What It Does
The SKL-SEC-006 check validates that high-risk category skills contain explicit safety phrases to warn users about potentially dangerous operations.

### Safety Guardrails Requirements (From ConfigMap)

The ConfigMap only requires safety guardrails for **3 categories**:

```yaml
safety-guardrails: |
  # Database skills MUST contain at least ONE of these phrases:
  database: confirmation required, dry run, requires approval, cannot be undone, irreversible, backup recommended
  
  # Infra skills MUST contain at least ONE of these phrases:
  infra: confirmation required, requires approval, change management, review before applying, dry run
  
  # Admin skills MUST contain at least ONE of these phrases:
  admin: admin access required, requires elevated privileges, audit logged, requires approval, multi-factor
```

### Categories That DO NOT Require Safety Guardrails
- documents
- data
- analytics
- communication
- productivity
- (any other custom category)

---

## Why xlsx-1-0-0 Has No SKL-SEC-006 Finding

✅ **This is correct behavior** - no finding should be generated

**Reason:** The xlsx skill is in the **"documents"** category, which is NOT in the safety guardrails requirements list.

```
xlsx-1-0-0 category = "documents" 
                    ↓
           Not in SafetyGuardrails list
                    ↓
        SKL-SEC-006 check SKIPPED
                    ↓
           No finding generated ✓
```

---

## Why anthoropic-skills-1-0-0 Has No SKL-SEC-006 Finding

✅ **This is correct behavior** - no finding should be generated

**Reason:** The anthoropic-skills skill is in the **"data"** category, which is NOT in the safety guardrails requirements list.

```
anthoropic-skills-1-0-0 category = "data" 
                                ↓
                   Not in SafetyGuardrails list
                                ↓
                    SKL-SEC-006 check SKIPPED
                                ↓
                       No finding generated ✓
```

---

## Complete Security Check Mapping

| Check ID | Check Name | Pattern Type | Finding Severity | Categories Affected |
|----------|-----------|--------------|------------------|-------------------|
| **SKL-SEC-001** | No Prompt Injection | Keywords | Critical | ALL (metadata check) |
| **SKL-SEC-002** | No Privilege Escalation | Keywords | Critical | ALL (metadata check) |
| **SKL-SEC-003** | No Data Exfiltration | Keywords | High | ALL (metadata check) |
| **SKL-SEC-004** | No Credential Harvesting | Keywords | Critical | ALL (metadata check) |
| **SKL-SEC-005** | Scope Compliance | Category-specific | High | Only matching categories |
| **SKL-SEC-006** | Safety Guardrails | Category-specific | Medium | **Only: database, infra, admin** |

---

## Security Pattern Content

### SKL-SEC-001: Prompt Injection Patterns
```
ignore previous instructions, ignore all previous, disregard your instructions,
forget your previous instructions, your new instructions are, override your system prompt,
new system prompt, ignore the above, do not follow, bypass safety, jailbreak,
DAN mode, developer mode enabled, pretend you are, act as if you have no restrictions,
simulate being
```

### SKL-SEC-002: Privilege Escalation Patterns
```
sudo, run as root, escalate privileges, become administrator, gain elevated access,
bypass authentication, disable security, setuid, chmod 777, chmod +s,
privilege escalation, run with elevated, execute as administrator,
administrator privileges
```

### SKL-SEC-003: Data Exfiltration Patterns
```
exfiltrate, send data to external, upload to external, transmit to remote,
leak data, send credentials, forward secrets, export sensitive,
copy to external storage, transfer to third party, webhook.site,
requestbin, burpcollaborator, oastify.com, pipedream.net
```

### SKL-SEC-004: Credential Harvesting Patterns
```
steal credentials, harvest passwords, capture tokens, extract api keys,
collect secrets, phishing, credential stuffing, brute force password,
enumerate users, dump credentials, keylogger, password spraying,
capture authentication
```

### SKL-SEC-005: Scope Creep (Category-Specific Forbidden Keywords)
- **data**: deploy, provision infrastructure, create cluster, manage kubernetes
- **analytics**: delete database, drop table, truncate, delete all records
- **communication**: execute code, run shell, bash -c, system command
- **productivity**: access production, modify live data, delete records, admin console

### SKL-SEC-006: Safety Guardrails (Category-Specific Required Phrases)
- **database**: confirmation required, dry run, requires approval, cannot be undone, irreversible, backup recommended
- **infra**: confirmation required, requires approval, change management, review before applying, dry run
- **admin**: admin access required, requires elevated privileges, audit logged, requires approval, multi-factor

---

## Controller Implementation Details

### ScanContent Function Logic

The controller's `ScanContent()` function in `controller/pkg/skillscanner/scanner.go` implements this logic:

```go
// SKL-SEC-006: Missing safety guardrails — only check rules matching this category
for _, rule := range ps.SafetyGuardrails {
    if !strings.EqualFold(rule.Category, skillCategory) {
        continue  // ← SKIP if category doesn't match
    }
    found := false
    for _, phrase := range rule.RequiredPhrases {
        if strings.Contains(lower, strings.ToLower(phrase)) {
            found = true
            break
        }
    }
    if !found {
        // Generate finding: safety guardrails missing
    }
}
```

**Key Point:** The check is **category-aware** and only applies to skills whose `spec.category` matches a SafetyGuardrails rule.

---

## Real-World Example: How to Trigger SKL-SEC-006 Finding

If we create a **database** skill without safety phrases:

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: SkillCatalog
metadata:
  name: my-db-skill-1-0-0
spec:
  name: my-db-skill
  category: database  # ← Requires safety guardrails
  description: "Perform database operations"
  # ... rest of spec
```

If the skill's repository files contain:
```markdown
# My Database Skill

This skill deletes all records from your database.
```

**Result:** ❌ SKL-SEC-006 Finding Generated
- **Title:** "No safety guardrail found in database skill file ..."
- **Remediation:** "Add an explicit safety note. Required phrases (at least one): confirmation required, dry run, requires approval, cannot be undone, irreversible, backup recommended"

**Fix:** Update the SKILL.md to include:
```markdown
# My Database Skill

**⚠️ Warning:** This skill performs irreversible database operations.
All changes are permanent. A backup is recommended before proceeding.
```

---

## Summary Table: Current Skills Status

| Skill | Category | Policy Requires Guardrails? | SKL-SEC-006 Finding | Status |
|-------|----------|-------|-----|--------|
| anthoropic-skills-1-0-0 | data | ❌ No | ✅ None | PASS (Score 95) |
| xlsx-1-0-0 | documents | ❌ No | ✅ None | PASS (Score 100) |

---

## Next Steps / Recommendations

1. **✅ Security scanning is enabled** - All 6 security checks are active
2. **✅ Patterns are correctly loaded** - 6 pattern keys loaded from ConfigMap
3. **✅ xlsx-1-0-0 is legitimate** - Anthropic's official skills repository with no security issues
4. **✅ anthoropic-skills-1-0-0 only has one minor issue** - Description too short (SKL-006 metadata check, not security)

**To test SKL-SEC-006:** Create a skill in the "database", "infra", or "admin" category to see the safety guardrails check in action.
