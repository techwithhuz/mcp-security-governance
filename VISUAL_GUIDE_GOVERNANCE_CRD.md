# Visual Guide: Governance Scoring in CRD

## Feature at a Glance

```
BEFORE (Dashboard Only)
┌──────────────────────────┐
│  MCP Governance          │
│  Dashboard               │
│                          │
│  • Verified Catalog Tab  │
│  • MCP Servers Tab       │
│                          │
│  (No CLI/API access)     │
└──────────────────────────┘

AFTER (Dashboard + CRD)
┌──────────────────────────┐
│  MCP Governance          │
│  Dashboard               │
└────────────┬─────────────┘
             │
   ┌─────────┴────────────────┐
   │                          │
   ▼                          ▼
┌──────────────────┐  ┌──────────────────┐
│  kubectl access  │  │  Script/API      │
│                  │  │  Automation      │
│  GovernanceEval  │  │                  │
│  CRD Status      │  │  CI/CD           │
│                  │  │  Monitoring      │
└──────────────────┘  └──────────────────┘
```

## Data Structure

```
GovernanceEvaluation.status
│
├── score: 78                    ◄── Overall cluster score
├── phase: "Complete"
├── lastEvaluationTime: "..."
│
├── scoreBreakdown:              ◄── Cluster-level breakdown
│   ├── agentGatewayScore
│   ├── authenticationScore
│   └── ...
│
├── verifiedCatalogScores: [     ◄── NEW: Catalog scores
│   │
│   ├── [0]:
│   │   ├── catalogName: "kagent/my-server"
│   │   ├── namespace: "kagent"
│   │   ├── status: "Verified"
│   │   ├── compositeScore: 72
│   │   ├── securityScore: 75
│   │   ├── trustScore: 68
│   │   ├── complianceScore: 70
│   │   ├── checks: [
│   │   │   ├── {id: "PUB-001", points: 10, maxPoints: 10}
│   │   │   ├── {id: "SEC-001", points: 8, maxPoints: 10}
│   │   │   └── ...
│   │   ]
│   │   └── lastScored: "..."
│   │
│   └── [1]: { ... next catalog ... }
│
└── mcpServerScores: [           ◄── NEW: Server scores
    │
    ├── [0]:
    │   ├── name: "kagent-tool-server"
    │   ├── namespace: "kagent"
    │   ├── source: "KagentMCPServer"
    │   ├── status: "compliant"
    │   ├── score: 85
    │   ├── scoreBreakdown: {
    │   │   ├── gatewayRouting: 25
    │   │   ├── authentication: 20
    │   │   ├── authorization: 15
    │   │   ├── tls: 10
    │   │   ├── cors: 5
    │   │   ├── rateLimit: 5
    │   │   ├── promptGuard: 3
    │   │   └── toolScope: 2
    │   }
    │   ├── toolCount: 15
    │   ├── effectiveToolCount: 10
    │   ├── relatedResources: {
    │   │   ├── gateways: 1
    │   │   ├── backends: 1
    │   │   ├── policies: 1
    │   │   └── routes: 1
    │   }
    │   ├── criticalFindings: 0
    │   └── lastEvaluated: "..."
    │
    └── [1]: { ... next server ... }
```

## Use Case Flows

### Use Case 1: Check Catalog Verification Status

```
┌─────────────────────────────────────┐
│  User runs:                         │
│  kubectl get governanceevaluation.. │
│  | jq '..[].verifiedCatalogScores'  │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  View all catalog scores            │
│  • Catalog name                     │
│  • Verification status              │
│  • Composite score (0-100)          │
│  • Category breakdown               │
│  • Individual check points          │
└─────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Decision:                          │
│  • Is it verified? → Yes/No         │
│  • What score? → 72/100             │
│  • What needs improvement? → Checks │
└─────────────────────────────────────┘
```

### Use Case 2: Monitor Server Compliance

```
┌─────────────────────────────────────┐
│  Automated monitoring script         │
│  • Runs every 5 minutes              │
│  • Fetches mcpServerScores          │
│  • Checks for status != "compliant"  │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  For each server, check:            │
│  • Status (compliant/warning/etc)   │
│  • Score (0-100)                    │
│  • Critical findings                │
│  • Tool count vs effective count    │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Actions:                           │
│  • Alert if score drops             │
│  • Report findings                  │
│  • Suggest remediation              │
└─────────────────────────────────────┘
```

### Use Case 3: CI/CD Integration

```
┌──────────────────────────┐
│  Deploy MCP Server       │
│  (Helm/kubectl)          │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│  Controller evaluates:   │
│  • Governance config     │
│  • Security settings     │
│  • Tool scope            │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│  Updates GovernanceEval  │
│  CRD status with scores  │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│  CI/CD script reads CRD: │
│  SCORE=$(kubectl get..)  │
│  if [ $SCORE < 70 ];     │
│    FAIL_DEPLOYMENT       │
│  fi                      │
└────────────┬─────────────┘
             │
             ▼
        [SUCCESS]
        [FAILURE]
```

## Query Patterns

### Pattern 1: Simple Read

```bash
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores[0]'

Result: Single server score object
```

### Pattern 2: Filter by Status

```bash
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores[] | \
      select(.status != "compliant")'

Result: All non-compliant servers
```

### Pattern 3: Extract Specific Fields

```bash
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores[] | \
      {name, score, status}'

Result: Simplified view of each server
```

### Pattern 4: Aggregate Stats

```bash
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores | \
      {count: length, avgScore: (map(.score) | add/length)}'

Result: Cluster-wide statistics
```

## Status Values

### Verified Catalog Status

```
┌──────────────┬─────────────────────────────┐
│ Status       │ Meaning                     │
├──────────────┼─────────────────────────────┤
│ Verified     │ Score ≥ 70 (meets standard)│
│ Unverified   │ Score ≥ 50, < 70           │
│ Rejected     │ Score < 50                 │
│ Pending      │ Still evaluating           │
└──────────────┴─────────────────────────────┘
```

### MCP Server Status

```
┌──────────────┬──────────────────────────┐
│ Status       │ Meaning                  │
├──────────────┼──────────────────────────┤
│ compliant    │ All governance met       │
│ warning      │ Some issues detected     │
│ failing      │ Multiple issues          │
│ critical     │ Severe governance gaps   │
└──────────────┴──────────────────────────┘
```

## Score Interpretation

### Verified Catalog Scores

```
         100 ┌─────────────────────────────┐
             │ IDEAL - All checks passed   │
          80 │ ┌─────────────────────────┐ │
             │ │ Verified Zone           │ │
          70 │ │ (meets requirements)    │ │
             │ └─────────────┬───────────┘ │
          50 │ ┌─────────────▼───────────┐ │
             │ │ Unverified Zone         │ │
             │ │ (improvements needed)   │ │
           0 │ │ Rejected Zone           │ │
             │ │ (does not meet minimum) │ │
             └─────────────────────────────┘
```

### MCP Server Scores

```
         100 ┌────────────────────────────┐
             │ PERFECT                    │
             │ All controls enabled       │
          80 │ ├────────────────────────┤ │ ← Compliant
             │ │ Gateway ✓ Auth ✓      │ │
          60 │ │ TLS ✓ CORS ✓          │ │
             │ ├────────────────────────┤ │
          40 │ │ Missing some controls  │ │ ← Warning
             │ │ Gateway ✓ Auth ✗      │ │
          20 │ ├────────────────────────┤ │ ← Failing
             │ │ Major gaps             │ │
           0 │ │ No governance          │ │ ← Critical
             └────────────────────────────┘
```

## Integration Architecture

```
┌─────────────────────────────────────────────────────┐
│         Kubernetes Cluster                          │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ MCP Governance Controller                    │  │
│  │                                              │  │
│  │ 1. Discovery Agent (watches resources)      │  │
│  │    └─ MCPServerCatalog                      │  │
│  │    └─ KagentMCPServer                       │  │
│  │    └─ RemoteMCPServer                       │  │
│  │                                              │  │
│  │ 2. Evaluator (calculates scores)            │  │
│  │    └─ Security checks                       │  │
│  │    └─ Trust evaluation                      │  │
│  │    └─ Compliance assessment                 │  │
│  │                                              │  │
│  │ 3. Transformer (converts to CRD format)     │  │
│  │    └─ Internal types → CRD types            │  │
│  │                                              │  │
│  │ 4. Status Updater (persists results)        │  │
│  │    └─ GovernanceEvaluation.status ← update  │  │
│  └──────────┬───────────────────────────────────┘  │
│             │                                       │
│             ▼                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │ GovernanceEvaluation CRD (etcd)              │  │
│  │                                              │  │
│  │ status:                                      │  │
│  │   verifiedCatalogScores: [...]              │  │
│  │   mcpServerScores: [...]                    │  │
│  │   lastEvaluationTime: "..."                 │  │
│  └──────────┬───────────────────────────────────┘  │
│             │                                       │
│  ┌──────────┴──────────────────────────────────┐   │
│  │                                              │   │
│  ▼                          ▼                  ▼    │
│ Dashboard              kubectl                API   │
│ (UI View)            (CLI Query)         (Scripts)  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## Data Flow Example

```
┌──────────────────────────────┐
│ MCPServerCatalog discovered  │
│ name: "kagent/my-server"     │
│ namespace: "kagent"          │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│ Evaluator calculates scores: │
│ PUB-001: 10/10 (verified)   │
│ PUB-002: 5/10 (low adoption)│
│ PUB-003: 7/10 (good rate)   │
│ SEC-001: 8/10 (https)       │
│ SEC-002: 9/10 (managed)     │
│ TOOL-001: 8/15 (5 tools)    │
│ USE-001: 9/10 (used widely) │
└──────────────┬───────────────┘
               │ Aggregates:
               │ Security: 75 (17/25)
               │ Trust: 68 (22/30)
               │ Compliance: 70 (8/15, but weight=20)
               │
               ▼
┌──────────────────────────────┐
│ Transformer creates CRD      │
│ VerifiedCatalogScore:        │
│ ├─ catalogName: "..."        │
│ ├─ compositeScore: 72        │
│ ├─ securityScore: 75         │
│ ├─ trustScore: 68            │
│ ├─ complianceScore: 70       │
│ └─ checks: [...]             │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│ Controller updates status:   │
│ GovernanceEvaluation         │
│ .status                      │
│ .verifiedCatalogScores[0]    │
│ = new VerifiedCatalogScore   │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│ User queries:                │
│ kubectl get ge -o json |     │
│ jq '.items[0].status...'     │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│ User sees:                   │
│ {                            │
│   "catalogName": "...",      │
│   "status": "Verified",      │
│   "compositeScore": 72,      │
│   "checks": [...]            │
│ }                            │
└──────────────────────────────┘
```

## Comparison: Dashboard vs CRD

```
┌─────────────────┬──────────────────┬──────────────────┐
│ Feature         │ Dashboard        │ CRD (kubectl)    │
├─────────────────┼──────────────────┼──────────────────┤
│ View UI         │ ✓ Detailed       │ ✗ N/A            │
│ Query data      │ ✗ (via API only) │ ✓ Any jq filter  │
│ Scripting       │ ✗ Complex        │ ✓ Easy           │
│ CI/CD           │ ✗ Not suitable   │ ✓ Perfect        │
│ Monitoring      │ ✗ One-off views  │ ✓ Continuous     │
│ Historical data │ ✗ Limited        │ ✓ In etcd        │
│ Permissions     │ Web auth         │ RBAC (standard)  │
│ Dependencies    │ Browser, server  │ kubectl only     │
└─────────────────┴──────────────────┴──────────────────┘
```

## Getting Started

```
Step 1: Check CRD is installed
├─ kubectl get crd governanceevaluations.governance.mcp.io
└─ Should return: governanceevaluations.governance.mcp.io

Step 2: Find GovernanceEvaluation resource
├─ kubectl get governanceevaluation -A
└─ Usually in: mcp-governance namespace

Step 3: Query scores
├─ kubectl get governanceevaluation -o json
├─ Pipe to: jq '.items[0].status.verifiedCatalogScores'
└─ Or: jq '.items[0].status.mcpServerScores'

Step 4: Use in automation
├─ Parse JSON in scripts
├─ Extract specific fields
└─ Make decisions based on scores
```

## Performance Reference

```
┌─────────────────────────────────────┐
│ Query Performance (approximate)      │
├─────────────────────────────────────┤
│ Get all scores:           <50ms      │
│ Filter by status:         <100ms     │
│ Calculate aggregates:     <200ms     │
│ Export to CSV (100 items):<500ms     │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│ Storage Impact (approximate)         │
├─────────────────────────────────────┤
│ Per catalog score:        ~1KB       │
│ Per server score:         ~500B      │
│ 100 resources total:      ~150KB     │
│ etcd size impact:         Minimal    │
└─────────────────────────────────────┘
```

---

**For more details, see:**
- User Guide: `QUICK_REFERENCE_KUBECTL_SCORING.md`
- Full Docs: `GOVERNANCE_EVALUATION_CRD_SCORING.md`
- Implementation: `CONTROLLER_INTEGRATION_GUIDE.md`
