# ✅ Safe Skills Example

> **Reference Implementation**
> This skill catalog demonstrates a **fully MCP-G compliant** skill set.
> All 13 security checks and all 8 metadata checks pass with a score of **100 / 100**.

---

## What This Demonstrates

This example shows how a properly designed skill catalog looks when evaluated by the MCP-G On-Demand Skills Scanner.
Apply the `skill-catalog.yaml` CR to your cluster, then run the On-Demand Scanner from the MCP-G dashboard against `examples/safe-skills/skills`.

---

## Security Check Results (All Pass ✅)

| Check ID | Severity | Description | Status |
|----------|----------|-------------|--------|
| SKL-SEC-001 | Critical | No prompt injection patterns | ✅ Pass |
| SKL-SEC-002 | High | No privilege escalation commands | ✅ Pass |
| SKL-SEC-003 | Critical | No data exfiltration patterns | ✅ Pass |
| SKL-SEC-004 | Critical | No hardcoded secrets or credentials | ✅ Pass |
| SKL-SEC-005 | High | No dangerous shell commands | ✅ Pass |
| SKL-SEC-006 | High | Safety guardrails present | ✅ Pass |
| SKL-SEC-007 | High | No SSRF / metadata endpoint access | ✅ Pass |
| SKL-SEC-008 | High | No arbitrary code execution (`eval`) | ✅ Pass |
| SKL-SEC-009 | High | No path traversal patterns | ✅ Pass |
| SKL-SEC-010 | Medium | No sensitive data in outputs | ✅ Pass |
| SKL-SEC-011 | Medium | No open redirect patterns | ✅ Pass |
| SKL-SEC-012 | Medium | No template injection patterns | ✅ Pass |
| SKL-SEC-013 | Low | Input validation documented | ✅ Pass |

**Expected score: 100 / 100 — Status: PASS**

---

## Metadata Check Results (All Pass ✅)

| Check ID | Description | Status |
|----------|-------------|--------|
| SKL-001 | Name present and valid | ✅ Pass |
| SKL-002 | Description present | ✅ Pass |
| SKL-003 | Version follows semantic versioning | ✅ Pass |
| SKL-004 | Category defined | ✅ Pass |
| SKL-005 | Repository URL present | ✅ Pass |
| SKL-006 | Specific category (not generic) | ✅ Pass |
| SKL-007 | Repository source defined | ✅ Pass |
| SKL-008 | Website URL points to known org | ✅ Pass |

---

## How to Use

### 1. Apply the SkillCatalog CR
```bash
kubectl apply -f skill-catalog.yaml
```

### 2. Open MCP-G Dashboard → Skills Catalog tab
The catalog `safe-skills-example` should appear with a **Pass** status badge.

### 3. Run the On-Demand Scanner
Click **"Scan Skills"** on the catalog card.
Point the scanner at: `examples/safe-skills/skills`

### 4. Review Results
All checks pass with a score of **100**.
No findings will be reported.

---

## Directory Structure

```
safe-skills/
├── README.md                  ← this file
├── skill-catalog.yaml         ← SkillCatalog CR manifest
└── skills/
    ├── automation/
    │   └── SKILL.md           ← clean automation skill, all checks pass
    └── database/
        └── SKILL.md           ← clean DB skill with explicit safety guardrails
```

---

## Best Practices Demonstrated

- ✅ No `eval`, `exec`, or dynamic code execution
- ✅ No hardcoded credentials — uses environment variable references instead
- ✅ No `sudo` or privilege escalation commands
- ✅ All inputs are validated and sanitized
- ✅ Path inputs are constrained to an allowlist
- ✅ Safety guardrails require explicit human approval for destructive operations
- ✅ No direct external HTTP calls — uses a designated output bus
- ✅ Jinja2 templating is not used; output uses structured JSON only
- ✅ Semantic versioning (`1.2.0`)
- ✅ Repository URL points to a known org (`techwithhuz`)
