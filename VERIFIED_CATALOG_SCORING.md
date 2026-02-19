# Verified Catalog Scoring System

## Overview

The Verified Catalog scoring system evaluates MCPServerCatalog resources from the Agent Registry inventory across **3 governance categories** using **10 individual checks**.

## Scoring Architecture

### Individual Checks (10 Total)

Each check has:
- **ID**: Unique identifier (e.g., `SEC-001`)
- **Name**: Human-readable description
- **MaxScore**: Maximum points for the check (varies by check)
- **Score**: Points earned (0 if failed, up to MaxScore if passed)
- **Category**: Publisher, Transport, Deployment, ToolScope, or Usage

### Category Maximum Scores

| Category | Checks | Total Points |
|----------|--------|--------------|
| **Publisher (Trust)** | PUB-001, PUB-002, PUB-003 | 30 |
| **Transport (Security)** | SEC-001, SEC-002 | 25 |
| **Deployment (Security)** | DEP-001, DEP-002, DEP-003 | 20 |
| **ToolScope (Compliance)** | TOOL-001 | 15 |
| **Usage (Compliance)** | USE-001 | 10 |
| **TOTAL** | 10 checks | **100 points** |

## Three Tier Scoring System

### Tier 1: Individual Check Points (Raw)
Each check awards 0 to maxScore points:
```
PUB-001: 10 points (source tracking)
PUB-002: 10 points (environment labels)
PUB-003: 10 points (management type)
SEC-001: 15 points (transport type)
SEC-002: 10 points (remote endpoint TLS)
DEP-001:  5 points (published)
DEP-002: 10 points (deployment ready)
DEP-003:  5 points (versioning)
TOOL-001: 15 points (tool scope)
USE-001: 10 points (agent usage)
```

### Tier 2: Category Scores (Normalized to 0-100)

Raw check points are summed per category, then **normalized to 0-100** for fair comparison:

```
Security = (Transport Points + Deployment Points) / Max × 100
         = (SEC-001 + SEC-002 + DEP-001 + DEP-002 + DEP-003) / 45 × 100
         = (0-45) points → (0-100) normalized

Trust = Publisher Points / Max × 100
      = (PUB-001 + PUB-002 + PUB-003) / 30 × 100
      = (0-30) points → (0-100) normalized

Compliance = (ToolScope + Usage) / Max × 100
           = (TOOL-001 + USE-001) / 25 × 100
           = (0-25) points → (0-100) normalized
```

### Tier 3: Composite Score (Weighted 0-100)

The three normalized category scores are combined using **configurable weights**:

```
Composite = (Security × 50% + Trust × 30% + Compliance × 20%)
          = (66 × 0.5 + 90 × 0.3 + 80 × 0.2)
          = 33 + 27 + 16
          = 76/100
```

**Default Weights:**
- Security: 50% (transport + deployment security)
- Trust: 30% (publisher verification)
- Compliance: 20% (tool scope + agent usage)

## Example Calculation

### Scenario: MCPServerCatalog with All Checks Passing

**Raw Check Points:**
```
PUB-001: 10/10 ✓
PUB-002: 10/10 ✓
PUB-003: 10/10 ✓
SEC-001: 15/15 ✓
SEC-002: 10/10 ✓
DEP-001:  5/5  ✓
DEP-002: 10/10 ✓
DEP-003:  5/5  ✓
TOOL-001: 15/15 ✓
USE-001: 10/10 ✓
─────────────────
TOTAL: 100/100 points
```

**Category Scores (Normalized):**
```
Security = (15 + 10 + 5 + 10 + 5) / 45 × 100 = 45/45 → 100%
Trust = (10 + 10 + 10) / 30 × 100 = 30/30 → 100%
Compliance = (15 + 10) / 25 × 100 = 25/25 → 100%
```

**Composite Score:**
```
= 100×0.5 + 100×0.3 + 100×0.2
= 50 + 30 + 20
= 100/100 ✓ "Verified" Grade: A
```

---

### Scenario: MCPServerCatalog with Partial Checks

**Raw Check Points:**
```
PUB-001: 10/10 ✓ (source tracked)
PUB-002:  5/10 ⚠ (environment label only, missing cluster)
PUB-003:  0/10 ✗ (no management type)
SEC-001: 15/15 ✓ (HTTPS configured)
SEC-002: 10/10 ✓ (remote TLS present)
DEP-001:  5/5  ✓ (published)
DEP-002:  0/10 ✗ (deployment not ready)
DEP-003:  0/5  ✗ (using "latest" version)
TOOL-001:  7/15 ⚠ (11 tools: between warning=10 and critical=20, gets 50% score)
USE-001: 10/10 ✓ (used by 1 agent)
─────────────────
TOTAL: 62/100 points
```

**Category Scores (Normalized):**
```
Security = (15 + 10 + 5 + 0 + 0) / 45 × 100 = 30/45 → 67%
Trust = (10 + 5 + 0) / 30 × 100 = 15/30 → 50%
Compliance = (7 + 10) / 25 × 100 = 17/25 → 68%
```

**Composite Score:**
```
= 67×0.5 + 50×0.3 + 68×0.2
= 33.5 + 15 + 13.6
= 62/100 ⚠ "Unverified" Grade: C
```

**Findings Generated:**
- 🟡 PUB-002: Missing cluster label
- 🔴 PUB-003: No management type set
- 🔴 DEP-002: Deployment not ready (Critical)
- 🟡 DEP-003: Version not semantic (use v1.0.0 not "latest")
- 🟡 TOOL-001: Tool count exceeds warning threshold

---

## UI Display Explanation

### Category Score Bars

When you see:
```
Security: 66/100
```

This means:
- ✅ **Raw Score**: Security checks earned 30 out of 45 possible points
- 📊 **Normalized**: 30/45 × 100 = **66%** (displayed as 66/100 for consistency)

**Why Normalize?**
- Security has 45 max points
- Trust has 30 max points  
- Compliance has 25 max points
- Without normalization, they can't be fairly weighted and compared

### Check Details

Click the **ℹ️ icon** on a category bar to see:

```
Transport Security Checks:
✓ SEC-001: Transport Type        15/15 pts
✓ SEC-002: Remote Endpoint TLS   10/10 pts

Deployment Health Checks:
✓ DEP-001: Published              5/5 pts
✗ DEP-002: Deployment Ready       0/10 pts  (Critical finding)
✗ DEP-003: Versioned              0/5 pts   (Warning finding)

Raw Total: 30/45 points
Normalized Score (0-100): 66/100
```

---

## Status Determination

Based on composite score and configurable thresholds:

| Score | Status | Grade | Meaning |
|-------|--------|-------|---------|
| ≥ 70 | **Verified** | A-B | Meets governance requirements |
| 50-69 | **Unverified** | C-D | Partial compliance, needs attention |
| < 50 | **Rejected** | F | Does not meet minimum requirements |

**Configurable Thresholds** (via MCPGovernancePolicy CRD):
```yaml
spec:
  verifiedCatalogScoring:
    verifiedThreshold: 70      # score >= 70 → "Verified"
    unverifiedThreshold: 50    # 50 ≤ score < 70 → "Unverified"
                               # score < 50 → "Rejected"
```

---

## Summary

Your question about the scoring discrepancy is now resolved:

✅ **Individual checks** have variable max scores (10, 15, 5, etc.)
✅ **Category totals** combine those raw points (e.g., 45 for Security)
✅ **Category scores** normalize to 0-100 for fair comparison
✅ **Composite score** weights the three categories and produces the final 0-100 result

**The bottom line: 66/100 is the normalized, weighted, final governance score.**
