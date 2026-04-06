# Ready to Use: What You Can Do Now

## Current Status ✅

**Everything is deployed and ready!** You can now:

1. ✅ Access the dashboard with search features
2. ✅ Query the CRD from kubectl
3. ✅ Prepare for the next integration phase

---

## 1. Access the Dashboard

### Start Port Forward
```bash
kubectl port-forward svc/mcp-governance-dashboard 3000:3000
```

### Open Dashboard
```
http://localhost:3000
```

### Dashboard Features Available Now
- **MCP Servers Tab**: Search bar to filter servers by name, namespace, or source
- **Verified Catalog Tab**: Search bar to filter catalogs by name, namespace, org, or publisher
- Scoring displays
- Governance checks
- Status indicators

---

## 2. Query Governance via kubectl

### Check All Governance Status
```bash
kubectl get governanceevaluation
```

**Output:**
```
NAME   SCORE   PHASE     SCOPE
...    78      Complete  cluster
```

### View Full Status (JSON)
```bash
kubectl get governanceevaluation -o json | jq '.items[0].status'
```

### Check Available Scoring Arrays
```bash
kubectl get governanceevaluation -o json | jq '.items[0].status | keys'
```

**Output includes:**
```json
[
  "findingsCount",
  "lastEvaluationTime",
  "mcpServerScores",
  "namespaceScores",
  "phase",
  "resourceSummary",
  "score",
  "scoreBreakdown",
  "verifiedCatalogScores"  ← NEW
]
```

### Verify CRD Fields Exist
```bash
kubectl get crd governanceevaluations.governance.mcp.io -o json | \
  jq '.spec.versions[0].schema.openAPIV3Schema.properties.status.properties | keys'
```

**Output:**
```json
[
  "findings",
  "findingsCount",
  "lastEvaluationTime",
  "mcpServerScores",        ← NEW
  "namespaceScores",
  "phase",
  "resourceSummary",
  "score",
  "scoreBreakdown",
  "verifiedCatalogScores"   ← NEW
]
```

---

## 3. Test Dashboard Search Features

### MCP Servers Tab
1. Open dashboard: `http://localhost:3000`
2. Go to **MCP Servers** tab
3. In search bar, type:
   - Server name (e.g., "kagent")
   - Namespace (e.g., "default")
   - Source type (e.g., "Remote")
4. Results filter in real-time ✅

### Verified Catalog Tab
1. Click **Verified Catalog** tab
2. In search bar, type:
   - Catalog name (e.g., "kagent/server")
   - Namespace
   - Organization name
   - Publisher name
3. Results filter in real-time ✅

### Search Features
- ✅ Real-time filtering as you type
- ✅ Clear button (X) to reset search
- ✅ Works alongside status filters
- ✅ Case-insensitive matching
- ✅ Partial string matching

---

## 4. Prepare for Phase 3: Integration

### What's Needed
The CRD fields exist but are empty until the controller is updated to populate them. To fill these fields:

1. **Create transformer.go** - Functions to convert internal types to CRD types
2. **Update evaluator.go** - Call transformers to collect scores
3. **Update controller loop** - Write collected scores to CRD status
4. **Add RBAC** - Grant status update permissions

### Timeline for Phase 3
- Development: 2-4 hours
- Testing: 1-2 hours
- Integration: 1-2 hours
- **Total: 4-8 hours**

### See Detailed Instructions
👉 Read: `CONTROLLER_INTEGRATION_GUIDE.md`

---

## 5. Quick Commands Reference

### Dashboard
```bash
# Port forward
kubectl port-forward svc/mcp-governance-dashboard 3000:3000 &

# View logs
kubectl logs -f deployment/mcp-governance-dashboard

# Check status
kubectl get deployment mcp-governance-dashboard
```

### Controller
```bash
# View logs
kubectl logs -f deployment/mcp-governance-controller

# Check status
kubectl get deployment mcp-governance-controller

# Port forward to API
kubectl port-forward svc/mcp-governance-controller 8090:8090 &

# Test API health
curl http://localhost:8090/api/health
```

### CRD Operations
```bash
# List all GovernanceEvaluations
kubectl get governanceevaluations -A

# Describe the CRD
kubectl describe crd governanceevaluations.governance.mcp.io

# Watch for changes
kubectl get governanceevaluation -w

# Export to file
kubectl get governanceevaluation -o yaml > evaluation.yaml

# Apply changes
kubectl apply -f evaluation.yaml
```

---

## 6. Next: Sample Governance Policy

Once the controller is updated to populate scores, you can create a governance policy:

```bash
# Apply sample policy
kubectl apply -f deploy/samples/governance-policy.yaml

# Or create a custom policy
cat > my-policy.yaml << 'EOF'
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: default
  namespace: mcp-governance
spec:
  requireAgentGateway: true
  requireCORS: true
  requireJWTAuth: true
  requireRBAC: true
  requirePromptGuard: true
  requireTLS: true
  requireRateLimit: true
  verifiedCatalogScoring:
    securityWeight: 50
    trustWeight: 30
    complianceWeight: 20
    verifiedThreshold: 70
    unverifiedThreshold: 50
EOF

kubectl apply -f my-policy.yaml
```

---

## 7. Documentation You Can Read Now

### Quick Start (5 minutes)
📄 `QUICK_REFERENCE_KUBECTL_SCORING.md`
- Quick commands
- Common queries
- Usage examples

### Full Reference (30 minutes)
📄 `GOVERNANCE_EVALUATION_CRD_SCORING.md`
- Complete field descriptions
- Real examples
- Integration patterns

### Visual Guide (10 minutes)
📄 `VISUAL_GUIDE_GOVERNANCE_CRD.md`
- Diagrams
- Data flow
- Architecture

### For Implementation (2 hours)
📄 `CONTROLLER_INTEGRATION_GUIDE.md`
- Step-by-step code changes
- Testing approach
- Integration tips

---

## 8. Architecture Deployed

```
┌────────────────────────────────────────────────────┐
│         Kubernetes Cluster (Kind)                  │
│                                                    │
│  ┌──────────────────────────────────────────────┐ │
│  │  Controller (Port 8090)                      │ │
│  │  ✓ Running                                   │ │
│  │  ✓ Ready for integration                     │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  ┌──────────────────────────────────────────────┐ │
│  │  Dashboard (Port 3000)                       │ │
│  │  ✓ Search enabled                           │ │
│  │  ✓ MCP Servers tab with search              │ │
│  │  ✓ Verified Catalog tab with search         │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  ┌──────────────────────────────────────────────┐ │
│  │  CRD (etcd Storage)                          │ │
│  │  ✓ verifiedCatalogScores field ready        │ │
│  │  ✓ mcpServerScores field ready              │ │
│  │  ⏳ Awaiting data from controller integration│ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
└────────────────────────────────────────────────────┘
```

---

## 9. What Happens After Integration

Once the controller is updated (Phase 3):

```bash
# These will show actual data:
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.verifiedCatalogScores'

# Example output will be:
[
  {
    "catalogName": "kagent/my-server",
    "status": "Verified",
    "compositeScore": 72,
    "securityScore": 75,
    "trustScore": 68,
    "complianceScore": 70,
    "checks": [
      {"id": "PUB-001", "points": 10, "maxPoints": 10},
      ...
    ],
    "lastScored": "2026-02-19T10:30:00Z"
  }
]
```

And for MCP servers:
```bash
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores'

# Example output will be:
[
  {
    "name": "kagent-tool-server",
    "namespace": "kagent",
    "status": "compliant",
    "score": 85,
    "scoreBreakdown": {
      "gatewayRouting": 25,
      "authentication": 20,
      ...
    },
    "toolCount": 15,
    "effectiveToolCount": 10,
    "criticalFindings": 0
  }
]
```

---

## 10. Helpful Links

### Run Commands
```bash
# Start dashboard port forward
kubectl port-forward svc/mcp-governance-dashboard 3000:3000

# Check all pods
kubectl get pods -A

# View controller logs in real-time
kubectl logs -f deployment/mcp-governance-controller

# View dashboard logs
kubectl logs -f deployment/mcp-governance-dashboard
```

### Files to Review
- `/charts/mcp-governance/crds/governanceevaluations.yaml` - CRD definition
- `/controller/pkg/apis/governance/v1alpha1/types.go` - Go types
- `/dashboard/src/components/MCPServerList.tsx` - Search implementation
- `/dashboard/src/components/VerifiedCatalog.tsx` - Search implementation

---

## Summary

✅ **What's Ready**
- Dashboard with search features
- CRD with scoring fields
- Controller running
- Documentation complete

⏳ **What's Next**
- Integrate transformer functions
- Populate CRD fields with scores
- Test in running cluster
- Deploy to production

**Estimated time to complete integration: 4-8 hours**

---

## Quick Start Checklist

- [ ] Start port forward: `kubectl port-forward svc/mcp-governance-dashboard 3000:3000`
- [ ] Open dashboard: http://localhost:3000
- [ ] Test MCP Servers search
- [ ] Test Verified Catalog search
- [ ] Query CRD: `kubectl get governanceevaluation -o json | jq '.items[0].status'`
- [ ] Read integration guide: `CONTROLLER_INTEGRATION_GUIDE.md`
- [ ] Plan Phase 3 implementation

---

**You're all set! Ready for the next phase?** 🚀
