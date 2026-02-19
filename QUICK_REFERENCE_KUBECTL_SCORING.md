# Quick Reference: Check Scoring via kubectl

## What Changed?

The `GovernanceEvaluation` CRD now includes all your Verified Catalog and MCP Server scores. You can query them directly without opening the dashboard!

## Quick Start

### 1. View All Scores at Once

```bash
kubectl get governanceevaluation -A
```

Shows: Overall score, phase, scope, findings count

### 2. View Verified Catalog Scores (JSON)

```bash
kubectl get governanceevaluation -n mcp-governance cluster-evaluation -o json | jq '.status.verifiedCatalogScores'
```

**Output:**
```json
[
  {
    "catalogName": "kagent/my-mcp-server",
    "namespace": "kagent",
    "status": "Verified",
    "compositeScore": 72,
    "securityScore": 75,
    "trustScore": 68,
    "complianceScore": 70,
    "checks": [
      {"id": "PUB-001", "name": "Publisher Verified", "points": 10, "maxPoints": 10},
      {"id": "SEC-001", "name": "Transport Type", "points": 8, "maxPoints": 10}
    ]
  }
]
```

### 3. View MCP Server Scores (JSON)

```bash
kubectl get governanceevaluation -n mcp-governance cluster-evaluation -o json | jq '.status.mcpServerScores'
```

**Output:**
```json
[
  {
    "name": "kagent-tool-server",
    "namespace": "kagent",
    "source": "KagentMCPServer",
    "status": "compliant",
    "score": 85,
    "toolCount": 15,
    "effectiveToolCount": 10,
    "criticalFindings": 0
  }
]
```

## Useful Commands

| Task | Command |
|------|---------|
| Get catalog score | `kubectl get governanceevaluation -o json \| jq '.items[0].status.verifiedCatalogScores[0]'` |
| Get server score | `kubectl get governanceevaluation -o json \| jq '.items[0].status.mcpServerScores[0]'` |
| List all catalogs | `kubectl get governanceevaluation -o json \| jq '.items[0].status.verifiedCatalogScores[].catalogName'` |
| List all servers | `kubectl get governanceevaluation -o json \| jq '.items[0].status.mcpServerScores[].name'` |
| Find non-compliant servers | `kubectl get governanceevaluation -o json \| jq '.items[0].status.mcpServerScores[] \| select(.status!="compliant")'` |
| Get average MCP score | `kubectl get governanceevaluation -o json \| jq '.items[0].status.mcpServerScores \| map(.score) \| add/length'` |

## Field Reference

### Verified Catalog Score

| Field | Meaning | Range |
|-------|---------|-------|
| `compositeScore` | Overall verification score | 0-100 |
| `securityScore` | Transport & deployment quality | 0-100 |
| `trustScore` | Publisher verification | 0-100 |
| `complianceScore` | Tool scope & usage | 0-100 |
| `status` | Verification status | Verified, Unverified, Rejected, Pending |

### MCP Server Score

| Field | Meaning | Range |
|-------|---------|-------|
| `score` | Governance score | 0-100 |
| `status` | Compliance status | compliant, warning, failing, critical |
| `toolCount` | Total tools exposed | 0+ |
| `effectiveToolCount` | Tools after policies | 0+ |
| `criticalFindings` | Critical issues | 0+ |

## Where to Find It

**Resource Name:** `GovernanceEvaluation`  
**API Group:** `governance.mcp.io`  
**Kind:** `GovernanceEvaluation`

**Default location:**
```bash
kubectl get governanceevaluation -n mcp-governance cluster-evaluation
```

## Real Examples

### Example 1: Check if a specific catalog is verified

```bash
kubectl get governanceevaluation -o json | \
  jq '.items[].status.verifiedCatalogScores[] | 
      select(.catalogName=="kagent/my-mcp-server") | 
      {catalogName, status, compositeScore}'
```

**Output:**
```json
{
  "catalogName": "kagent/my-mcp-server",
  "status": "Verified",
  "compositeScore": 72
}
```

### Example 2: List all servers that are failing governance

```bash
kubectl get governanceevaluation -o json | \
  jq '.items[].status.mcpServerScores[] | 
      select(.status=="failing") | 
      {name, score, criticalFindings}'
```

### Example 3: Find servers with the lowest scores

```bash
kubectl get governanceevaluation -o json | \
  jq '.items[].status.mcpServerScores | 
      sort_by(.score) | 
      .[0:3] | 
      .[] | {name, score, status}'
```

## Automatic Updates

- Scores update automatically every 5-10 minutes
- Check `.status.lastEvaluated` to see when data was last updated
- All timestamps are in UTC (ISO 8601 format)

## Combining with Other Tools

### Prometheus Monitoring

Export scores for monitoring:
```bash
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores[] | 
      "mcp_server_score{name=\"\(.name)\",namespace=\"\(.namespace)\",status=\"\(.status)\"} \(.score)"'
```

### CI/CD Integration

Use in pipelines to fail if governance drops:
```bash
#!/bin/bash
SCORE=$(kubectl get governanceevaluation -o json | jq '.items[0].status.score')
if [ "$SCORE" -lt 70 ]; then
  echo "Governance score too low: $SCORE"
  exit 1
fi
```

### Dashboarding

Compare scores across time:
```bash
# Get score history (requires storing periodically)
for i in {1..10}; do
  echo "Sample $i: $(date) - Score: $(kubectl get governanceevaluation -o json | jq '.items[0].status.score')"
  sleep 300
done
```

## Troubleshooting

### GovernanceEvaluation not found?

```bash
# Check if it exists
kubectl get governanceevaluation -A

# If empty, check the policy:
kubectl get mcpgovernancepolicy -A
```

### Scores not updating?

```bash
# Check last evaluation time
kubectl get governanceevaluation -o json | jq '.items[0].status.lastEvaluationTime'

# Check for errors in controller logs
kubectl logs -n mcp-governance -l app=mcp-governance-controller -f
```

### Can't find a specific server/catalog?

```bash
# List all servers with their exact names
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores[].name'

# Use the exact name to query
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores[] | select(.name=="exact-name-here")'
```

## Learn More

- Full documentation: `GOVERNANCE_EVALUATION_CRD_SCORING.md`
- CRD definition: `charts/mcp-governance/crds/governanceevaluations.yaml`
- View scores in dashboard: Visit the MCP Servers and Verified Catalog tabs
