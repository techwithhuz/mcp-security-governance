# Quick Reference: Resource Types Configuration

## Simplest Possible Usage

### Default (No Config Needed)

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: my-policy
spec:
  requireAgentGateway: true
  # That's it! Uses MCPServerCatalog by default
```

### Add More Resource Types

```yaml
spec:
  resourceTypes:
    - "MCPServerCatalog"
    - "Agent"
    - "RemoteMCPServer"
```

---

## Supported Resource Types

```
"MCPServerCatalog"  → agentregistry.dev/v1alpha1/mcpservercatalogs
"Agent"             → kagent.dev/v1alpha2/agents
"RemoteMCPServer"   → kagent.dev/v1alpha2/remotemcpservers
"Gateway"           → gateway.networking.k8s.io/v1/gateways
"HTTPRoute"         → gateway.networking.k8s.io/v1/httproutes
```

---

## Common Scenarios

### Scenario 1: Watch Everything

```yaml
resourceTypes:
  - "MCPServerCatalog"
  - "Agent"
  - "RemoteMCPServer"
  - "Gateway"
  - "HTTPRoute"
```

### Scenario 2: Watch Only Agents

```yaml
resourceTypes:
  - "Agent"
```

### Scenario 3: Watch Gateways + Routes

```yaml
resourceTypes:
  - "Gateway"
  - "HTTPRoute"
```

---

## What Happens Under the Hood

1. Policy applied with `resourceTypes: ["MCPServerCatalog", "Agent"]`
2. Controller reads policy
3. Discovery converts to:
   ```
   - agentregistry.dev/v1alpha1, Resource=mcpservercatalogs
   - kagent.dev/v1alpha2, Resource=agents
   ```
4. Informers watch both resources
5. Resources are scored and patched in real-time

---

## Expected Logs

```
[discovery] Discovered resource: MCPServerCatalog (agentregistry.dev/v1alpha1, Resource=mcpservercatalogs)
[discovery] Discovered resource: Agent (kagent.dev/v1alpha2, Resource=agents)
[discovery] Discovered 2 resource types from policy

[inventory] Starting resource watcher for 2 resource types
[inventory] Added watcher for agentregistry.dev/v1alpha1, Resource=mcpservercatalogs
[inventory] Added watcher for kagent.dev/v1alpha2, Resource=agents
[inventory] Catalog resource cache synced — watching for changes across 2 resource types
```

---

## Errors & Solutions

| Error | Cause | Fix |
|-------|-------|-----|
| `unknown resource type "MyType"` | Invalid type name | Use supported names |
| `No resource types configured` | Empty list | Omit field for defaults |
| `Skipping duplicate` | Same type listed twice | Remove duplicate |

---

## Adding New Resource Types

To support a new resource type (e.g., `"Skill"`):

**File:** `controller/pkg/inventory/discovery.go`

```go
var ResourceTypeMapping = map[string]schema.GroupVersionResource{
    // ... existing types ...
    "Skill": {
        Group:    "kagent.dev",
        Version:  "v1alpha2",
        Resource: "skills",
    },
}
```

Then you can use `"Skill"` in policies immediately!

---

## Testing Your Config

```bash
# Apply policy
kubectl apply -f my-policy.yaml

# Check logs for discovery messages
kubectl logs -f deployment/mcp-governance-controller | grep discovery

# Verify resources are watched
kubectl logs -f deployment/mcp-governance-controller | grep "Added watcher"

# Verify resources are scored
kubectl logs -f deployment/mcp-governance-controller | grep "ADDED\|UPDATED"
```

---

## Summary

- ✅ Just use resource type names (strings)
- ✅ Controller maps to actual Kubernetes GVRs
- ✅ Add to `resourceTypes` array in spec
- ✅ No changes needed for backward compatibility
- ✅ 5 built-in types ready to use

**That's it! Simple and clean.** 🚀
