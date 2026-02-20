# Integration Guide: Populating CRD Status with Scoring

## Overview

This guide explains how to integrate the scoring data collection with the controller's evaluator to automatically populate the `verifiedCatalogScores` and `mcpServerScores` arrays in the GovernanceEvaluation CRD status.

## Integration Points

### 1. Controller Evaluator (`pkg/evaluator/evaluator.go`)

The main evaluator function needs to:
1. Collect all MCPServerView objects (evaluated servers)
2. Collect all VerifiedScore objects (evaluated catalogs)
3. Transform them into the CRD-compatible format
4. Update the GovernanceEvaluation status

**Pseudo-code:**

```go
func (e *Evaluator) EvaluateCluster(ctx context.Context) error {
    // ... existing evaluation logic ...
    
    // Collect MCP Server evaluations
    mcpServerScores := []v1alpha1.MCPServerScore{}
    for _, server := range allMCPServers {
        score := v1alpha1.MCPServerScore{
            Name:              server.Name,
            Namespace:         server.Namespace,
            Source:            server.Source,
            Status:            server.Status,
            Score:             server.Score,
            ScoreBreakdown: v1alpha1.MCPServerScoreBreakdown{
                GatewayRouting: server.ScoreBreakdown.GatewayRouting,
                Authentication: server.ScoreBreakdown.Authentication,
                // ... other fields ...
            },
            ToolCount:         server.ToolCount,
            EffectiveToolCount: server.EffectiveToolCount,
            RelatedResources: v1alpha1.RelatedResourceSummary{
                Gateways: len(server.RelatedGateways),
                Backends: len(server.RelatedBackends),
                Policies: len(server.RelatedPolicies),
                Routes:   len(server.RelatedRoutes),
            },
            CriticalFindings: countCriticalFindings(server.Findings),
            LastEvaluated:    now(),
        }
        mcpServerScores = append(mcpServerScores, score)
    }
    
    // Collect Verified Catalog evaluations
    verifiedCatalogScores := []v1alpha1.VerifiedCatalogScore{}
    for _, catalog := range allVerifiedCatalogs {
        checks := []v1alpha1.CatalogScoringCheck{}
        for _, check := range catalog.ScoringChecks {
            checks = append(checks, v1alpha1.CatalogScoringCheck{
                ID:        check.ID,
                Name:      check.Name,
                Points:    check.Points,
                MaxPoints: check.MaxPoints,
            })
        }
        
        score := v1alpha1.VerifiedCatalogScore{
            CatalogName:     catalog.CatalogName,
            Namespace:       catalog.Namespace,
            ResourceVersion: catalog.ResourceVersion,
            Status:          catalog.Status,
            CompositeScore:  catalog.CompositeScore,
            SecurityScore:   catalog.SecurityScore,
            TrustScore:      catalog.TrustScore,
            ComplianceScore: catalog.ComplianceScore,
            Checks:          checks,
            LastScored:      now(),
        }
        verifiedCatalogScores = append(verifiedCatalogScores, score)
    }
    
    // Update GovernanceEvaluation status
    ge := &v1alpha1.GovernanceEvaluation{}
    if err := e.client.Get(ctx, types.NamespacedName{...}, ge); err != nil {
        return err
    }
    
    ge.Status.MCPServerScores = mcpServerScores
    ge.Status.VerifiedCatalogScores = verifiedCatalogScores
    ge.Status.LastEvaluationTime = now()
    
    return e.client.Status().Update(ctx, ge)
}
```

### 2. Watcher Integration (`pkg/watcher/watcher.go` or `pkg/inventory/watcher.go`)

The watcher that triggers evaluations should:
1. Watch for changes to MCPServerCatalog, KagentMCPServer, RemoteMCPServer
2. Trigger full evaluation when resources change
3. Controller automatically updates the status arrays

**Existing pattern likely already in place:**

```go
func (w *Watcher) watchMCPServers() error {
    // ... watch MCPServerCatalog resources ...
    
    // When change detected:
    w.evaluator.EvaluateCluster(ctx)  // This will update status arrays
}
```

### 3. Data Transformation Layer

Create a transformer to convert from internal types to CRD types:

**File location suggestion:** `pkg/evaluator/transformer.go`

```go
package evaluator

import "k8s.io/apimachinery/pkg/apis/meta/v1"

// TransformMCPServerViewToCRD converts MCPServerView to CRD type
func TransformMCPServerViewToCRD(srv *MCPServerView) *v1alpha1.MCPServerScore {
    return &v1alpha1.MCPServerScore{
        Name:              srv.Name,
        Namespace:         srv.Namespace,
        Source:            srv.Source,
        Status:            srv.Status,
        Score:             srv.Score,
        ScoreBreakdown: v1alpha1.MCPServerScoreBreakdown{
            GatewayRouting: srv.ScoreBreakdown.GatewayRouting,
            Authentication: srv.ScoreBreakdown.Authentication,
            Authorization:  srv.ScoreBreakdown.Authorization,
            TLS:            srv.ScoreBreakdown.TLS,
            CORS:           srv.ScoreBreakdown.CORS,
            RateLimit:      srv.ScoreBreakdown.RateLimit,
            PromptGuard:    srv.ScoreBreakdown.PromptGuard,
            ToolScope:      srv.ScoreBreakdown.ToolScope,
        },
        ToolCount:         srv.ToolCount,
        EffectiveToolCount: srv.EffectiveToolCount,
        RelatedResources: v1alpha1.RelatedResourceSummary{
            Gateways: len(srv.RelatedGateways),
            Backends: len(srv.RelatedBackends),
            Policies: len(srv.RelatedPolicies),
            Routes:   len(srv.RelatedRoutes),
        },
        CriticalFindings: countCriticalFindings(srv.Findings),
        LastEvaluated:    ptrToTime(time.Now()),
    }
}

// TransformVerifiedScoreToCRD converts VerifiedScore to CRD type
func TransformVerifiedScoreToCRD(vs *VerifiedScore) *v1alpha1.VerifiedCatalogScore {
    checks := make([]v1alpha1.CatalogScoringCheck, len(vs.Checks))
    for i, check := range vs.Checks {
        checks[i] = v1alpha1.CatalogScoringCheck{
            ID:        check.ID,
            Name:      check.Name,
            Points:    check.Points,
            MaxPoints: check.MaxPoints,
        }
    }
    
    return &v1alpha1.VerifiedCatalogScore{
        CatalogName:     vs.CatalogName,
        Namespace:       vs.Namespace,
        ResourceVersion: vs.ResourceVersion,
        Status:          vs.Status,
        CompositeScore:  vs.CompositeScore,
        SecurityScore:   vs.SecurityScore,
        TrustScore:      vs.TrustScore,
        ComplianceScore: vs.ComplianceScore,
        Checks:          checks,
        LastScored:      ptrToTime(time.Now()),
    }
}

func countCriticalFindings(findings []Finding) int {
    count := 0
    for _, f := range findings {
        if f.Severity == "Critical" || f.Status == "failing" {
            count++
        }
    }
    return count
}

func ptrToTime(t time.Time) *v1.Time {
    mt := v1.NewTime(t)
    return &mt
}
```

## Step-by-Step Integration

### Step 1: Update Evaluator Main Function

**File:** `pkg/evaluator/evaluator.go`

```go
// In the Evaluate or EvaluateCluster function, add:

// Transform collected data to CRD types
mcpServerScores := make([]v1alpha1.MCPServerScore, len(evaluatedServers))
for i, srv := range evaluatedServers {
    mcpServerScores[i] = *TransformMCPServerViewToCRD(srv)
}

verifiedCatalogScores := make([]v1alpha1.VerifiedCatalogScore, len(evaluatedCatalogs))
for i, catalog := range evaluatedCatalogs {
    verifiedCatalogScores[i] = *TransformVerifiedScoreToCRD(catalog)
}

// Update status
evaluation.Status.MCPServerScores = mcpServerScores
evaluation.Status.VerifiedCatalogScores = verifiedCatalogScores
```

### Step 2: Create Transformer Functions

**File:** `pkg/evaluator/transformer.go` (new file)

Copy the transformer functions shown above.

### Step 3: Update Status Periodically

**File:** `pkg/watcher/watcher.go` or main controller loop

```go
// After evaluation completes:
func (c *Controller) reconcile(ctx context.Context) error {
    // Run evaluation
    results := c.evaluator.EvaluateCluster(ctx)
    
    // Update GovernanceEvaluation status
    ge := &v1alpha1.GovernanceEvaluation{}
    if err := c.client.Get(ctx, client.ObjectKeyFromObject(ge), ge); err != nil {
        return err
    }
    
    ge.Status.Score = results.OverallScore
    ge.Status.MCPServerScores = results.ServerScores
    ge.Status.VerifiedCatalogScores = results.CatalogScores
    ge.Status.LastEvaluationTime = ptr(v1.Now())
    ge.Status.Phase = "Complete"
    
    return c.client.Status().Update(ctx, ge)
}
```

### Step 4: Handle Empty Arrays

Ensure arrays are never nil in responses:

```go
// In evaluator
if mcpServerScores == nil {
    mcpServerScores = []v1alpha1.MCPServerScore{}
}
if verifiedCatalogScores == nil {
    verifiedCatalogScores = []v1alpha1.VerifiedCatalogScore{}
}

evaluation.Status.MCPServerScores = mcpServerScores
evaluation.Status.VerifiedCatalogScores = verifiedCatalogScores
```

## Testing the Integration

### Unit Tests

```go
// pkg/evaluator/transformer_test.go

func TestTransformMCPServerViewToCRD(t *testing.T) {
    srv := &MCPServerView{
        Name:      "test-server",
        Namespace: "default",
        Source:    "KagentMCPServer",
        Status:    "compliant",
        Score:     85,
        ToolCount: 10,
    }
    
    result := TransformMCPServerViewToCRD(srv)
    
    assert.Equal(t, "test-server", result.Name)
    assert.Equal(t, 85, result.Score)
    assert.Equal(t, "compliant", result.Status)
}
```

### Integration Tests

```bash
# After deployment, query to verify:
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores | length'

# Should return > 0 if servers were evaluated
```

## Monitoring Integration

### Prometheus Scraping

If using metrics exporter:

```go
// Export status fields as metrics
func (c *Controller) exportMetrics(ge *v1alpha1.GovernanceEvaluation) {
    for _, score := range ge.Status.MCPServerScores {
        mcpServerScore.WithLabelValues(
            score.Name,
            score.Namespace,
            score.Source,
            score.Status,
        ).Set(float64(score.Score))
    }
    
    for _, score := range ge.Status.VerifiedCatalogScores {
        verifiedCatalogScore.WithLabelValues(
            score.CatalogName,
            score.Namespace,
            score.Status,
        ).Set(float64(score.CompositeScore))
    }
}
```

### Alert Rules

```yaml
# alerting-rules.yaml
groups:
  - name: mcp-governance
    rules:
      - alert: MCPServerLowGovernanceScore
        expr: mcp_server_score < 50
        for: 10m
        annotations:
          summary: "MCP Server {{ $labels.name }} has low governance score"
      
      - alert: VerifiedCatalogNotVerified
        expr: verified_catalog_score{status="Rejected"} > 0
        for: 5m
        annotations:
          summary: "Catalog {{ $labels.catalogName }} was rejected"
```

## Data Consistency

### Ensure Consistency Between Dashboard and CRD

The dashboard should read from the same evaluator output:

```go
// In API handler
func (h *Handler) GetMCPServers(w http.ResponseWriter, r *http.Request) {
    // Get from evaluator
    servers := h.evaluator.GetEvaluatedServers()
    
    // Also populate CRD (if not done elsewhere)
    h.updateGovernanceEvaluation(servers)
    
    // Return to API client
    json.NewEncoder(w).Encode(servers)
}
```

## Backward Compatibility

The implementation maintains existing functionality:

- Original `score` and `scoreBreakdown` fields still exist
- New arrays are additions
- No breaking changes to status schema
- Existing controllers continue to work

## Debugging

### Check if Status is Updating

```bash
# Watch for changes
kubectl get governanceevaluation -w

# Check specific field
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.mcpServerScores[0]'

# Check timestamp
kubectl get governanceevaluation -o json | \
  jq '.items[0].status.lastEvaluationTime'
```

### Common Issues

**Issue**: Arrays are empty even after evaluation

**Solution**: Ensure transformer functions are called in evaluator

**Issue**: Status not updating

**Solution**: Verify controller has permission to update status subresource

```yaml
# RBAC requirement
rules:
  - apiGroups: ["governance.mcp.io"]
    resources: ["governanceevaluations/status"]
    verbs: ["update", "patch"]
```

**Issue**: Old data persisting

**Solution**: Arrays are replaced entirely on update (not merged)

## Performance Considerations

- **Array size**: With 100+ servers, arrays will be fairly large
- **etcd impact**: Moderate increase in resource usage
- **Query performance**: jq queries may be slow on very large arrays
- **Pagination**: Consider implementing pagination for large clusters

## Rollout Plan

1. Deploy updated CRD
2. Deploy updated types.go
3. Implement transformer functions
4. Integrate into evaluator
5. Add status update in controller
6. Deploy updated controller
7. Test with `kubectl get governanceevaluation`
8. Monitor metrics
9. Announce to users

## Related Documentation

- CRD definition: `charts/mcp-governance/crds/governanceevaluations.yaml`
- Type definitions: `controller/pkg/apis/governance/v1alpha1/types.go`
- Query reference: `QUICK_REFERENCE_KUBECTL_SCORING.md`
- Full documentation: `GOVERNANCE_EVALUATION_CRD_SCORING.md`
