# ⚠️ Malicious Skills Example

> **WARNING — FOR TESTING PURPOSES ONLY**
> This skill catalog contains **intentional security vulnerabilities** to demonstrate the MCP-G On-Demand Skills Scanner capabilities.
> **Never deploy these skills in a real environment.**

---

## What This Demonstrates

This example shows how MCP-G detects dangerous patterns inside Skill Markdown files.
Apply the `skill-catalog.yaml` CR to your cluster, then run the On-Demand Scanner in the MCP-G dashboard against this folder's `skills/` path to see multiple checks fail.

---

## Triggered Security Checks

| Check ID | Severity | Pattern | Skill File |
|----------|----------|---------|------------|
| SKL-SEC-001 | Critical | Prompt Injection | `skills/automation/SKILL.md` |
| SKL-SEC-002 | High | Privilege Escalation | `skills/automation/SKILL.md` |
| SKL-SEC-007 | High | SSRF / Metadata Endpoint | `skills/automation/SKILL.md` |
| SKL-SEC-008 | High | Code Injection (`eval`) | `skills/automation/SKILL.md` |
| SKL-SEC-003 | Critical | Data Exfiltration | `skills/data-pipeline/SKILL.md` |
| SKL-SEC-004 | Critical | Hardcoded Secret | `skills/data-pipeline/SKILL.md` |
| SKL-SEC-005 | High | Dangerous Shell Command | `skills/data-pipeline/SKILL.md` |
| SKL-SEC-009 | High | Path Traversal | `skills/data-pipeline/SKILL.md` |
| SKL-SEC-012 | Medium | Template Injection | `skills/data-pipeline/SKILL.md` |

**Expected score: 0 / 100 — Status: FAIL**

---

## How to Use

### 1. Apply the SkillCatalog CR
```bash
kubectl apply -f skill-catalog.yaml
```

### 2. Open MCP-G Dashboard → Skills Catalog tab
The catalog `malicious-skills-example` should appear.

### 3. Run the On-Demand Scanner
Click **"Scan Skills"** on the catalog card.
Point the scanner at the local path: `examples/malicious-skills/skills`

### 4. Review Results
The scanner will flag 9+ security issues and calculate a score of **0**.
Each finding shows the check ID, severity, matched pattern, and line number.

---

## Directory Structure

```
malicious-skills/
├── README.md                  ← this file
├── skill-catalog.yaml         ← SkillCatalog CR manifest
└── skills/
    ├── automation/
    │   └── SKILL.md           ← triggers SKL-SEC-001, -002, -007, -008
    └── data-pipeline/
        └── SKILL.md           ← triggers SKL-SEC-003, -004, -005, -009, -012
```

---

## Metadata Check Results

In addition to security checks, the controller also evaluates metadata.
This example catalog is missing several metadata fields intentionally to also trigger metadata failures:

| Check ID | Issue |
|----------|-------|
| SKL-003 | No `spec.version` semantic tag — version is `0.1.0` (acceptable but minimal) |
| SKL-006 | `spec.category` is too generic (`automation`) |
| SKL-007 | Missing `spec.repository.source` validation (points to non-existent repo) |
| SKL-008 | GitHub URL does not point to a known org |

> Note: exact metadata results depend on the live controller evaluation.
