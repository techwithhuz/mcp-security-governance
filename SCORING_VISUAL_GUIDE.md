# Verified Catalog Scoring Visual Guide

## The Three-Tier System

```
┌─────────────────────────────────────────────────────────────────┐
│                     TIER 1: INDIVIDUAL CHECKS                   │
│                    (10 checks, raw points vary)                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  PUB-001 (10 pts)   SEC-001 (15 pts)   DEP-001 (5 pts)         │
│  PUB-002 (10 pts)   SEC-002 (10 pts)   DEP-002 (10 pts)        │
│  PUB-003 (10 pts)   TOOL-001 (15 pts)  DEP-003 (5 pts)         │
│                     USE-001 (10 pts)                           │
│                                                                 │
│  Total: 100 points max                                         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                TIER 2: CATEGORY AGGREGATION                     │
│          (3 categories, raw totals vary, then normalize)        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECURITY = (PUB-001 + PUB-002 + PUB-003)                      │
│           = Earned/30 × 100 = ?/100 normalized                 │
│           (Publisher verification)                             │
│                                                                 │
│  SECURITY = (SEC-001 + SEC-002 + DEP-001 + DEP-002 + DEP-003)  │
│           = Earned/45 × 100 = ?/100 normalized                 │
│           (Transport + Deployment security)                    │
│                                                                 │
│  COMPLIANCE = (TOOL-001 + USE-001)                             │
│             = Earned/25 × 100 = ?/100 normalized               │
│             (Tool scope + Agent usage)                         │
│                                                                 │
│  → Each normalized to 0-100 for fair comparison                │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                TIER 3: WEIGHTED COMPOSITE                       │
│              (Final 0-100 score with configurable weights)      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  COMPOSITE = (Security × 50%)                                  │
│            + (Trust × 30%)                                     │
│            + (Compliance × 20%)                                │
│                                                                 │
│  Example:  (66 × 0.5) + (90 × 0.3) + (80 × 0.2)               │
│          = 33 + 27 + 16                                        │
│          = 76/100 ✓ "Verified" Grade A                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Real Example: Breakdown

### Scenario: Real MCPServerCatalog Evaluation

```
╔═══════════════════════════════════════════════════════════════╗
║         TIER 1: ACTUAL CHECK RESULTS                          ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║  PUBLISHER VERIFICATION                                      ║
║  ✓ PUB-001: Source Kind Tracked          10/10 pts           ║
║  ⚠ PUB-002: Environment Labelled          5/10 pts (partial)║
║  ✗ PUB-003: Management Type Set           0/10 pts           ║
║                                  Subtotal: 15/30 pts         ║
║                                                               ║
║  TRANSPORT SECURITY                                          ║
║  ✓ SEC-001: Transport Type               15/15 pts           ║
║  ✓ SEC-002: Remote Endpoint TLS          10/10 pts           ║
║                                  Subtotal: 25/25 pts         ║
║                                                               ║
║  DEPLOYMENT HEALTH                                           ║
║  ✓ DEP-001: Published                     5/5 pts            ║
║  ✓ DEP-002: Deployment Ready             10/10 pts           ║
║  ✓ DEP-003: Versioned                     5/5 pts            ║
║                                  Subtotal: 20/20 pts         ║
║                                                               ║
║  COMPLIANCE                                                  ║
║  ⚠ TOOL-001: Tool Scope                   7/15 pts (11 tools)║
║  ✓ USE-001: Agent Usage                  10/10 pts           ║
║                                  Subtotal: 17/25 pts         ║
║                                                               ║
║  ─────────────────────────────────────────────────────────  ║
║  TOTAL RAW POINTS:                       77/100 pts          ║
╚═══════════════════════════════════════════════════════════════╝
                           ↓
╔═══════════════════════════════════════════════════════════════╗
║         TIER 2: CATEGORY NORMALIZATION (0-100)               ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║  TRUST = PUB × 100 / 30                                      ║
║        = 15 × 100 / 30 = 50/100                              ║
║        → Status: ⚠ NEEDS WORK (missing source tracking)     ║
║                                                               ║
║  SECURITY = (TRANSPORT + DEPLOYMENT) × 100 / 45              ║
║           = (25 + 20) × 100 / 45 = 45 × 100 / 45            ║
║           = 100/100                                          ║
║           → Status: ✓ EXCELLENT (transport + deployment OK) ║
║                                                               ║
║  COMPLIANCE = (TOOL + USE) × 100 / 25                        ║
║             = 17 × 100 / 25 = 68/100                         ║
║             → Status: ⚠ FAIR (too many tools exposed)       ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
                           ↓
╔═══════════════════════════════════════════════════════════════╗
║      TIER 3: FINAL WEIGHTED COMPOSITE SCORE                  ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║  Security (100/100) × weight 50% = 100 × 0.5 = 50 points   ║
║  Trust (50/100)     × weight 30% =  50 × 0.3 = 15 points   ║
║  Compliance (68/100) × weight 20% =  68 × 0.2 = 13.6 pts   ║
║                                  ─────────────────────────  ║
║  COMPOSITE SCORE = 50 + 15 + 13.6 = 78.6 ≈ 79/100           ║
║                                                               ║
║  Final Status: ✓ VERIFIED (≥70)                             ║
║  Grade: B (80-89 range)                                     ║
║  Reason: Overall good, but publisher verification incomplete ║
║                                                               ║
║  Findings Generated:                                        ║
║  🟡 Medium: Environment label missing                       ║
║  🟡 Medium: Tool count exceeds warning (11 > 10)            ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
```

---

## Why Normalize? (Important Concept)

Without normalization, the weighting would be broken:

```
❌ WRONG (without normalization):
  Raw points: Security=45, Trust=30, Compliance=25
  If weighted equally: (45 + 30 + 25) / 3 = 33 🤦
  → Security dominates because it has more raw points!

✅ CORRECT (with normalization):
  Normalized: Security=100, Trust=90, Compliance=75
  Weighted: (100×50% + 90×30% + 75×20%) = 89
  → Fair comparison, each category weighted as configured
```

---

## How to Read the UI Now

### In the Expanded Detail View:

```
Category Scores
┌─────────────┬─────────────┬─────────────┐
│  Security   │   Trust     │  Compliance │
│     66/100  │    90/100   │    80/100   │
│   ℹ️ Click  │  ℹ️ Click   │  ℹ️ Click   │
│   50% weight│  30% weight │  20% weight │
└─────────────┴─────────────┴─────────────┘

💡 Each category is normalized to 0-100 for fair comparison. 
Click the ℹ️ icon to see raw check points (score/maxScore).
```

### When You Click "Security" ℹ️:

```
Security Score Breakdown

Transport Security Checks:
✓ SEC-001: Transport Type        15/15
✓ SEC-002: Remote Endpoint TLS   10/10

Deployment Health Checks:
✓ DEP-001: Published              5/5
✓ DEP-002: Deployment Ready      10/10
✓ DEP-003: Versioned              5/5

Raw Total: 45/45 points
Normalized Score (0-100): 100/100
```

---

## The Math Explained (Simple)

Imagine you're grading:
- **Student A** (100 questions, gets 75 correct) = 75%
- **Student B** (50 questions, gets 42 correct) = 84%

Without normalization, you can't compare them fairly because Student A did more questions.

**Same concept here:**
- **Security** has 45 max points
- **Trust** has 30 max points
- **Compliance** has 25 max points

Normalizing to 0-100 lets us say "they're all equally important" (or weight them differently) fairly.

---

## Configurable Thresholds (Advanced)

You can adjust the scoring in the MCPGovernancePolicy CRD:

```yaml
spec:
  verifiedCatalogScoring:
    # Category weights (must sum to 100)
    securityWeight: 50     # Transport + Deployment importance
    trustWeight: 30        # Publisher verification importance
    complianceWeight: 20   # Tool scope + usage importance
    
    # Status thresholds
    verifiedThreshold: 70   # Score >= 70 → "Verified"
    unverifiedThreshold: 50 # Score >= 50 → "Unverified"
                            # Score < 50 → "Rejected"
    
    # Per-check max score overrides (optional)
    checkMaxScores:
      "SEC-001": 20    # Override default 15 → 20
      "TOOL-001": 20   # Override default 15 → 20
```

When you override a check's max score, the normalization adjusts automatically! ✅
