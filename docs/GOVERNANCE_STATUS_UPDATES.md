# Governance Controller Status Updates — Implementation Guide

## Overview

The governance controller now automatically updates the `.status.publisher` field of catalog resources (MCPServerCatalog, AgentCatalog, SkillCatalog, ModelCatalog) with verified governance scores. This feature enables real-time trust verification display in the Agent Registry UI and other integrations.

## Feature Highlights

✅ **Real-time Scoring** — Scores catalog resources on add/update events  
✅ **Automatic Patching** — Updates `.status.publisher` field with governance metrics  
✅ **Letter Grades** — Converts numeric scores (0–100) to letter grades (A–F)  
✅ **Publisher Verification** — Tracks publisher and organization verification status  
✅ **UI Integration Ready** — Status fields designed for color-coded badge display  
✅ **RBAC Secured** — Requires explicit status patching permissions  

## Implementation Details

### New Components

#### 1. **Status Patcher** (`controller/pkg/inventory/patcher.go`)

Responsible for patching catalog resource status fields with governance scores.

**Key Functions:**
- `NewStatusPatcher(client)` — Creates a new patcher instance
- `PatchCatalogStatus(ctx, resource)` — Patches a single resource's status
- `PatchMultipleCatalogs(ctx, resources)` — Batches patches with concurrency control

**Features:**
- Uses Kubernetes Merge Patch strategy to avoid conflicts
- Applies patches to the status subresource (safe for spec fields)
- Includes error logging and best-effort retry semantics
- Parallel patching with configurable concurrency (5 concurrent)

#### 2. **Inventory Watcher Updates** (`controller/pkg/inventory/watcher.go`)

Enhanced to call the patcher when resources are scored.

**Changes:**
- Added `StatusPatcher` field to `Watcher` struct
- Added `PatchStatusOnUpdate` boolean configuration option
- Modified `onAdd()` and `onUpdate()` event handlers to patch status

**Configuration:**
```go
watcher, err := inventory.NewWatcher(inventory.WatcherConfig{
    DynamicClient: client,
    Policy: policy,
    Namespace: "",
    PatchStatusOnUpdate: true,  // Enable status patching
    OnChange: callback,
})
```

#### 3. **Main Controller Updates** (`controller/cmd/api/main.go`)

Enabled status patching in the inventory watcher initialization.

**Changes:**
- Set `PatchStatusOnUpdate: true` when creating watcher
- Updated log message to indicate status patching is enabled

### Data Structure

**PublisherVerification** struct fields:

```go
type PublisherVerification struct {
    VerifiedPublisher bool
    VerifiedOrganization bool
    Score int              // 0–100
    Grade string           // A/B/C/D/F
    GradedAt metav1.Time
}
```

### Status Patch Format

The controller uses JSON Merge Patch strategy to update:

```json
{
  "status": {
    "publisher": {
      "verifiedPublisher": true,
      "verifiedOrganization": true,
      "score": 78,
      "grade": "B",
      "gradedAt": "2026-02-20T04:51:45Z"
    }
  }
}
```

## Deployment

### Prerequisites

1. **RBAC Permissions** — Controller service account must have:
   - `get`, `list`, `watch` on catalog resources (spec)
   - `get`, `patch`, `update` on catalog/status subresource

2. **Kubernetes Version** — 1.20+

### Helm Installation

The controller is deployed via Helm with status patching enabled by default:

```bash
helm install mcp-governance ./charts/mcp-governance \
  -n default \
  --set controller.statusPatching.enabled=true
```

### RBAC Configuration

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mcp-governance-controller
rules:
# ... existing rules ...
- apiGroups: ["agentregistry.dev"]
  resources:
    - mcpservercatalogs/status
    - agentcatalogs/status
    - skillcatalogs/status
    - modelcatalogs/status
  verbs: ["get", "patch", "update"]
- apiGroups: ["agentregistry.dev"]
  resources:
    - mcpservercatalogs
    - agentcatalogs
    - skillcatalogs
    - modelcatalogs
  verbs: ["get", "list", "watch"]
```

## Testing

### Local Testing

1. **Build the image:**
   ```bash
   cd controller
   podman build --platform linux/arm64 -t governance-controller:latest .
   ```

2. **Load into Kind cluster:**
   ```bash
   podman save governance-controller:latest | kind load image-archive --name mcp-governance
   ```

3. **Verify status updates:**
   ```bash
   kubectl get mcpservercatalog -n agentregistry -o jsonpath='{.items[*].status.publisher}'
   ```

### Expected Behavior

**Console Logs:**
```
[patcher] Successfully patched agentregistry/kagent-kagent-grafana-mcp status: score=78, grade=B, verifiedPublisher=true, verifiedOrg=true
[patcher] Successfully patched agentregistry/kagent-kagent-tool-server status: score=86, grade=B, verifiedPublisher=true, verifiedOrg=true
```

**Resource Status:**
```bash
$ kubectl get mcpservercatalog kagent-kagent-grafana-mcp -n agentregistry -o yaml
apiVersion: agentregistry.dev/v1alpha1
kind: MCPServerCatalog
metadata:
  name: kagent-kagent-grafana-mcp
  namespace: agentregistry
spec:
  # ... spec fields ...
status:
  publisher:
    score: 78
    grade: "B"
    verifiedPublisher: true
    verifiedOrganization: true
    gradedAt: "2026-02-20T04:51:45Z"
  published: true
  # ... other status fields ...
```

## Architecture Flow

```
┌─────────────────────────────────────────────┐
│  Kubernetes API — MCPServerCatalog Created  │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  Inventory Watcher   │ (onAdd/onUpdate)
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │   Score Catalog      │ (ScoreCatalog)
        │   0–100 numeric      │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  Status Patcher      │ (PatchCatalogStatus)
        │  Merge Patch to      │
        │  .status.publisher   │
        └──────────┬───────────┘
                   │
                   ▼
      ┌────────────────────────────┐
      │  Kubernetes API Update     │
      │  .status.publisher patched │
      └────────────────────────────┘
                   │
                   ▼
      ┌────────────────────────────┐
      │  Agent Registry UI         │
      │  Display Color Badge       │
      └────────────────────────────┘
```

## Error Handling

### Common Issues

1. **RBAC Forbidden Error**
   ```
   forbidden: User "system:serviceaccount:default:mcp-governance-controller" 
   cannot patch resource "mcpservercatalogs/status"
   ```
   **Solution:** Add status patching rules to ClusterRole

2. **Patch Timeout**
   ```
   failed to patch status: context deadline exceeded
   ```
   **Solution:** Patcher uses 5-second timeout; check API server health

3. **Connection Refused**
   ```
   failed to patch status: connection refused
   ```
   **Solution:** Verify Kubernetes API server is reachable from pod

### Logging

Enable detailed logging by checking pod logs:

```bash
kubectl logs -n default deployment/mcp-governance-controller -f | grep patcher
```

## Performance Considerations

- **Concurrency:** Patches up to 5 resources in parallel
- **Timeout:** 5-second timeout per patch operation
- **Retry Logic:** Best-effort (no built-in retries; relies on watcher re-events)
- **Memory:** Minimal overhead; patches are small JSON objects

## Future Enhancements

1. **Webhook Validation** — Add CRD webhook to validate score range (0–100)
2. **Event Recording** — Record Kubernetes events on status changes
3. **Historical Tracking** — Store score history in annotations
4. **Export Metrics** — Expose scores as Prometheus metrics
5. **Multi-Catalog Support** — Extend to support more catalog types

## Monitoring

### Metrics to Watch

- Number of successful patches
- Number of failed patches
- Average patch latency
- Error rate by error type

### Commands

```bash
# Count successful patches in logs
kubectl logs -n default deployment/mcp-governance-controller | grep "Successfully patched" | wc -l

# Show failed patch attempts
kubectl logs -n default deployment/mcp-governance-controller | grep "WARNING.*patch"

# Watch status updates in real-time
kubectl get mcpservercatalog -n agentregistry -w -o jsonpath='{.items[*].status.publisher.score}'
```

## References

- [Kubernetes Patch Strategy](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/#how-apply-calculates-differences)
- [Agent Registry CRD Documentation](https://github.com/den-vasyliev/agentregistry-inventory/blob/enterprise-controller/docs/governance-controller.md)
- [MCP Governance Controller Source](./controller/pkg/inventory/)
