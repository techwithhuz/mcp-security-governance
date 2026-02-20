# GovernanceEvaluation CRD - Integrated Scoring Documentation

## Overview

The `GovernanceEvaluation` CRD has been extended to store both **Verified Catalog Scores** and **MCP Server Scores** directly in its status. This allows users to query governance evaluation results via kubectl and see all scoring information in one place.

## What Was Added

### 1. Verified Catalog Scores (`verifiedCatalogScores`)

Stores the governance score for each `MCPServerCatalog` resource discovered in the cluster.

**Array of VerifiedCatalogScore objects:**

```yaml
verifiedCatalogScores:
  - catalogName: "kagent/my-mcp-server"
    namespace: "kagent"
    resourceVersion: "12345"
    status: "Verified"                          # "Verified", "Unverified", "Rejected", "Pending"
    compositeScore: 72                          # Final weighted score (0-100)
    securityScore: 75                           # Transport + Deployment checks (0-100)
    trustScore: 68                              # Publisher Verification (0-100)
    complianceScore: 70                         # Tool Scope + Usage (0-100)
    checks:
      - id: "PUB-001"
        name: "Publisher Verified"
        points: 10                              # Earned points
        maxPoints: 10                           # Max possible
      - id: "SEC-001"
        name: "Transport Type"
        points: 8
        maxPoints: 10
      # ... more checks
    lastScored: "2026-02-19T10:30:00Z"
```

### 2. MCP Server Scores (`mcpServerScores`)

Stores the governance score for each MCP server discovered in the cluster (KagentMCPServer, RemoteMCPServer, etc.).

**Array of MCPServerScore objects:**

```yaml
mcpServerScores:
  - name: "kagent-tool-server"
    namespace: "kagent"
    source: "KagentMCPServer"                   # Source type
    status: "compliant"                         # "compliant", "warning", "failing", "critical"
    score: 85                                   # Governance score (0-100)
    scoreBreakdown:
      gatewayRouting: 25                        # Points for gateway integration
      authentication: 20                        # Points for JWT/auth
      authorization: 15                         # Points for RBAC
      tls: 10                                   # Points for TLS
      cors: 5                                   # Points for CORS
      rateLimit: 5                              # Points for rate limiting
      promptGuard: 3                            # Points for prompt guard
      toolScope: 2                              # Points for tool restriction
    toolCount: 15                               # Total tools exposed
    effectiveToolCount: 10                      # Tools after policy enforcement
    relatedResources:
      gateways: 1
      backends: 1
      policies: 1
      routes: 1
    criticalFindings: 0                         # Number of critical issues
    lastEvaluated: "2026-02-19T10:30:00Z"
```

## How to Query GovernanceEvaluation

### Get the cluster-wide evaluation with all scores:

```bash
kubectl get governanceevaluation -A -o json | jq '.items[0].status'
```

### Get just Verified Catalog scores:

```bash
kubectl get governanceevaluation -A -o json | jq '.items[0].status.verifiedCatalogScores'
```

### Get just MCP Server scores:

```bash
kubectl get governanceevaluation -A -o json | jq '.items[0].status.mcpServerScores'
```

### View as table (with custom columns):

```bash
kubectl get governanceevaluation -o custom-columns=\
NAME:.metadata.name,\
PHASE:.status.phase,\
OVERALL-SCORE:.status.score,\
VERIFIED-CATALOGS:.status.verifiedCatalogScores[*].catalogName,\
MCP-SERVERS:.status.mcpServerScores[*].name
```

### Filter by specific catalog or server:

```bash
# Get score for a specific catalog
kubectl get governanceevaluation -A -o json | \
  jq '.items[0].status.verifiedCatalogScores[] | select(.catalogName=="kagent/my-mcp-server")'

# Get score for a specific MCP server
kubectl get governanceevaluation -A -o json | \
  jq '.items[0].status.mcpServerScores[] | select(.name=="kagent-tool-server")'
```

## Example GovernanceEvaluation Resource

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: GovernanceEvaluation
metadata:
  name: cluster-evaluation
  namespace: mcp-governance
spec:
  policyRef: default-policy
  evaluationScope: cluster
status:
  score: 78                           # Overall cluster governance score
  phase: Complete
  lastEvaluationTime: "2026-02-19T10:30:00Z"
  
  # Legacy: Overall breakdown (for backward compatibility)
  scoreBreakdown:
    agentGatewayScore: 25
    authenticationScore: 20
    authorizationScore: 15
    tlsScore: 10
    corsScore: 5
    rateLimitScore: 3
    promptGuardScore: 2
    toolScopeScore: 1
  
  # NEW: Verified Catalog scores
  verifiedCatalogScores:
    - catalogName: "kagent/my-mcp-server"
      namespace: "kagent"
      status: "Verified"
      compositeScore: 72
      securityScore: 75
      trustScore: 68
      complianceScore: 70
      checks:
        - id: "PUB-001"
          name: "Publisher Verified"
          points: 10
          maxPoints: 10
        - id: "PUB-002"
          name: "Publisher Response Rate"
          points: 5
          maxPoints: 10
        - id: "PUB-003"
          name: "Publisher Adoption"
          points: 7
          maxPoints: 10
        - id: "SEC-001"
          name: "Transport Type"
          points: 8
          maxPoints: 10
        - id: "SEC-002"
          name: "Deployment Type"
          points: 9
          maxPoints: 10
        - id: "TOOL-001"
          name: "Tool Count"
          points: 8
          maxPoints: 15
        - id: "USE-001"
          name: "Adoption Rate"
          points: 9
          maxPoints: 10
      lastScored: "2026-02-19T10:30:00Z"
  
  # NEW: MCP Server scores
  mcpServerScores:
    - name: "kagent-tool-server"
      namespace: "kagent"
      source: "KagentMCPServer"
      status: "compliant"
      score: 85
      scoreBreakdown:
        gatewayRouting: 25
        authentication: 20
        authorization: 15
        tls: 10
        cors: 5
        rateLimit: 5
        promptGuard: 3
        toolScope: 2
      toolCount: 15
      effectiveToolCount: 10
      relatedResources:
        gateways: 1
        backends: 1
        policies: 1
        routes: 1
      criticalFindings: 0
      lastEvaluated: "2026-02-19T10:30:00Z"
    
    - name: "another-mcp-server"
      namespace: "default"
      source: "RemoteMCPServer"
      status: "warning"
      score: 55
      scoreBreakdown:
        gatewayRouting: 0      # Not routed through gateway
        authentication: 0      # No authentication
        authorization: 10
        tls: 8
        cors: 0
        rateLimit: 3
        promptGuard: 0
        toolScope: 5
      toolCount: 20
      effectiveToolCount: 5
      relatedResources:
        gateways: 0
        backends: 0
        policies: 1
        routes: 0
      criticalFindings: 2
      lastEvaluated: "2026-02-19T10:30:00Z"
  
  findings:
    - id: "F-001"
      severity: "Critical"
      category: "AgentGateway"
      title: "MCP Server Not Behind Gateway"
      description: "another-mcp-server is not routed through AgentGateway"
      resourceRef: "RemoteMCPServer/another-mcp-server"
      namespace: "default"
      timestamp: "2026-02-19T10:30:00Z"
  
  resourceSummary:
    gatewaysFound: 1
    agentgatewayBackends: 2
    agentgatewayPolicies: 2
    httpRoutes: 2
    kagentAgents: 3
    kagentMCPServers: 1
    kagentRemoteMCPServers: 1
    compliantResources: 1
    nonCompliantResources: 1
    totalMCPEndpoints: 2
    exposedMCPEndpoints: 1
  
  findingsCount: 2
```

## Integration with Dashboard

### What This Enables

1. **Direct Kubernetes Access**: Users can query scores via kubectl without needing the dashboard
2. **Integration with CI/CD**: Tools and scripts can read scores from the CRD
3. **Audit Trail**: All historical evaluations are in the cluster as resources
4. **Monitoring**: Prometheus and other monitoring tools can scrape these values

### Dashboard Usage (Current)

The dashboard still pulls data from the controller API and displays:

- **Verified Catalog Tab**: Shows `verifiedCatalogScores` with scoring breakdown
- **MCP Servers Tab**: Shows `mcpServerScores` with governance status

When you view a catalog or server in the dashboard, it now corresponds to entries in the GovernanceEvaluation status.

## Field Descriptions

### VerifiedCatalogScore Fields

| Field | Type | Description |
|-------|------|-------------|
| `catalogName` | string | Name of the MCPServerCatalog resource |
| `namespace` | string | Namespace where the catalog is deployed |
| `resourceVersion` | string | Kubernetes resource version |
| `status` | enum | "Verified", "Unverified", "Rejected", or "Pending" |
| `compositeScore` | int | Final weighted score (0-100) |
| `securityScore` | int | Security category (transport + deployment, 0-100) |
| `trustScore` | int | Trust category (publisher verification, 0-100) |
| `complianceScore` | int | Compliance category (tool scope + usage, 0-100) |
| `checks` | array | Individual check details with points earned |
| `lastScored` | timestamp | When this was last evaluated |

### MCPServerScore Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Name of the MCP server |
| `namespace` | string | Kubernetes namespace |
| `source` | string | Type: "KagentMCPServer", "RemoteMCPServer", etc. |
| `status` | enum | "compliant", "warning", "failing", or "critical" |
| `score` | int | Governance score (0-100) |
| `scoreBreakdown` | object | Points for each governance control |
| `toolCount` | int | Total tools exposed |
| `effectiveToolCount` | int | Tools after policy enforcement |
| `relatedResources` | object | Count of gateways, backends, policies, routes |
| `criticalFindings` | int | Number of critical/failing findings |
| `lastEvaluated` | timestamp | When this was last evaluated |

## Common Queries

### Find all non-compliant MCP servers:

```bash
kubectl get governanceevaluation -A -o json | \
  jq '.items[].status.mcpServerScores[] | select(.status != "compliant")'
```

### Find low-scoring catalogs (< 50):

```bash
kubectl get governanceevaluation -A -o json | \
  jq '.items[].status.verifiedCatalogScores[] | select(.compositeScore < 50)'
```

### Get average MCP server score:

```bash
kubectl get governanceevaluation -A -o json | \
  jq '.items[].status.mcpServerScores | map(.score) | add/length'
```

### List all servers with critical findings:

```bash
kubectl get governanceevaluation -A -o json | \
  jq '.items[].status.mcpServerScores[] | select(.criticalFindings > 0)'
```

### Export all scores to CSV:

```bash
kubectl get governanceevaluation -A -o json | jq -r \
  '.items[].status.mcpServerScores[] | 
   [.name, .namespace, .source, .status, .score, .toolCount, .criticalFindings] | 
   @csv'
```

## Implementation Notes

### When Scores Are Updated

- **Verified Catalog Scores**: Updated whenever `MCPServerCatalog` resources are evaluated
- **MCP Server Scores**: Updated whenever `KagentMCPServer`, `RemoteMCPServer`, or other server resources are evaluated
- **Frequency**: Depends on policy reconciliation interval (typically every 5-10 minutes)

### Score Calculation

**Verified Catalog Composite Score:**
```
compositeScore = (securityScore × 0.5) + (trustScore × 0.3) + (complianceScore × 0.2)
```
Weighted to emphasize security and trust over compliance.

**MCP Server Governance Score:**
```
score = sum of all scoreBreakdown values
      = gatewayRouting + authentication + authorization + tls + cors + rateLimit + promptGuard + toolScope
```

### Backward Compatibility

The original `scoreBreakdown` and `score` fields in the status remain for cluster-level evaluation. The new fields are additions and don't replace existing functionality.

## Real-World Example: Monitoring Script

```bash
#!/bin/bash
# Monitor MCP server compliance over time

while true; do
  echo "=== MCP Server Governance Status ==="
  echo "Timestamp: $(date)"
  echo ""
  
  echo "Compliant servers:"
  kubectl get governanceevaluation -A -o json | \
    jq '.items[0].status.mcpServerScores[] | select(.status=="compliant") | .name'
  
  echo ""
  echo "Servers needing attention:"
  kubectl get governanceevaluation -A -o json | \
    jq '.items[0].status.mcpServerScores[] | select(.status!="compliant") | "\(.name): \(.status) (score: \(.score))"'
  
  echo ""
  echo "Average governance score:"
  kubectl get governanceevaluation -A -o json | \
    jq '.items[0].status.mcpServerScores | map(.score) | add/length'
  
  sleep 300  # Check every 5 minutes
done
```

## Related Files

- **CRD Definition**: `/charts/mcp-governance/crds/governanceevaluations.yaml`
- **Go Types**: `/controller/pkg/apis/governance/v1alpha1/types.go`
- **Dashboard Display**: `/dashboard/src/components/MCPServerList.tsx` and `VerifiedCatalog.tsx`
- **Controller Evaluator**: `/controller/pkg/evaluator/evaluator.go`
