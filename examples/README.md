# MCP-G Examples

This directory contains example skill catalogs for testing and demonstrating the **MCP-G On-Demand Skills Scanner**.

---

## Examples

| Folder | Purpose | Expected Score | Expected Status |
|--------|---------|----------------|-----------------|
| [`malicious-skills/`](malicious-skills/) | Intentional security violations — triggers 9 checks across 2 skill files | 0 / 100 | 🔴 FAIL |
| [`safe-skills/`](safe-skills/) | Fully compliant reference implementation — all 13 security + 8 metadata checks pass | 100 / 100 | 🟢 PASS |

---

## How to Use

### Step 1 — Apply the SkillCatalog CRs

```bash
# Apply both example catalogs to your cluster
kubectl apply -f examples/malicious-skills/skill-catalog.yaml
kubectl apply -f examples/safe-skills/skill-catalog.yaml
```

Verify they appear:

```bash
kubectl get skillcatalogs -n default
# NAME                       STATUS    SCORE   AGE
# malicious-skills-example   fail      25      10s
# safe-skills-example        pass      100     10s
```

### Step 2 — Open the MCP-G Dashboard

Navigate to the **Skills Catalog** tab.
Both catalogs will appear in the catalog grid.

### Step 3 — Run the On-Demand Scanner

Click **"Scan Skills"** on either catalog card.
When prompted, enter the path to the `skills/` folder inside the example:

- Malicious: `examples/malicious-skills/skills`
- Safe: `examples/safe-skills/skills`

### Step 4 — Compare Results

| | malicious-skills-example | safe-skills-example |
|-|--------------------------|---------------------|
| SKL-SEC-001 Prompt Injection | ❌ FAIL | ✅ Pass |
| SKL-SEC-002 Privilege Escalation | ❌ FAIL | ✅ Pass |
| SKL-SEC-003 Data Exfiltration | ❌ FAIL | ✅ Pass |
| SKL-SEC-004 Hardcoded Secrets | ❌ FAIL | ✅ Pass |
| SKL-SEC-005 Dangerous Commands | ❌ FAIL | ✅ Pass |
| SKL-SEC-006 Safety Guardrails | ❌ FAIL | ✅ Pass |
| SKL-SEC-007 SSRF | ❌ FAIL | ✅ Pass |
| SKL-SEC-008 Code Injection | ❌ FAIL | ✅ Pass |
| SKL-SEC-009 Path Traversal | ❌ FAIL | ✅ Pass |
| SKL-SEC-012 Template Injection | ❌ FAIL | ✅ Pass |
| **Score** | **0 / 100** | **100 / 100** |
| **Status** | **🔴 FAIL** | **🟢 PASS** |

---

## Security Checks Reference

All 13 security checks evaluated by the MCP-G On-Demand Scanner:

| Check ID | Severity | Description |
|----------|----------|-------------|
| SKL-SEC-001 | Critical | Prompt injection patterns |
| SKL-SEC-002 | High | Privilege escalation commands |
| SKL-SEC-003 | Critical | Data exfiltration patterns |
| SKL-SEC-004 | Critical | Hardcoded credentials / secrets |
| SKL-SEC-005 | High | Dangerous shell commands (`rm -rf`, `mkfs`, etc.) |
| SKL-SEC-006 | High | Missing safety guardrails |
| SKL-SEC-007 | High | SSRF / cloud metadata endpoint access |
| SKL-SEC-008 | High | Arbitrary code execution (`eval`, `exec`) |
| SKL-SEC-009 | High | Path traversal patterns |
| SKL-SEC-010 | Medium | Sensitive data in skill outputs |
| SKL-SEC-011 | Medium | Open redirect patterns |
| SKL-SEC-012 | Medium | Template injection (`{{ }}`, `{%  %}`) |
| SKL-SEC-013 | Low | Missing input validation documentation |

---

## Scoring Formula

```
score = max(0, 100 - Σ penalties)

Penalties:  Critical = 40 pts
            High     = 25 pts
            Medium   = 15 pts
            Low      =  5 pts

Status:  score ≥ 80  → pass
         score 50–79 → warning
         score < 50  → fail
```

---

## Directory Structure

```
examples/
├── README.md                                    ← this file
├── malicious-skills/
│   ├── README.md                                ← check-by-check breakdown
│   ├── skill-catalog.yaml                       ← SkillCatalog CR
│   └── skills/
│       ├── automation/
│       │   └── SKILL.md                         ← SKL-SEC-001, -002, -007, -008
│       └── data-pipeline/
│           └── SKILL.md                         ← SKL-SEC-003, -004, -005, -009, -012
└── safe-skills/
    ├── README.md                                ← best practices guide
    ├── skill-catalog.yaml                       ← SkillCatalog CR
    └── skills/
        ├── automation/
        │   └── SKILL.md                         ← clean, passes all checks
        └── database/
            └── SKILL.md                         ← clean, explicit safety guardrails
```
