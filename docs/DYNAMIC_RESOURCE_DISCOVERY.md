# Dynamic Resource Type Discovery & Watching

## Overview

This feature enables administrators to dynamically configure which Kubernetes resource types the governance controller should watch and monitor. Instead of hardcoding resource types, they can be specified in the `MCPGovernancePolicy` CRD, allowing for flexible policy-driven resource discovery.

## Motivation

**Current State:**
- Resource types to watch are hardcoded in `controller/pkg/watcher/watcher.go`
- Adding/removing resources requires code changes and redeployment
- Not flexible for different governance scenarios

**Proposed State:**
- Resource types defined in `MCPGovernancePolicy.spec.watchedResourceTypes`
- Dynamic discovery at policy load time
- Easy add/remove without code changes
- Per-policy resource monitoring

## Design

### 1. CRD Extension (`types.go`)

Add a new field to `MCPGovernancePolicySpec`:

```go
type MCPGovernancePolicySpec struct {
    // ...existing fields...
    
    // WatchedResourceTypes defines which Kubernetes resources to monitor
    // If empty, defaults to standard MCP governance resource types
    WatchedResourceTypes []WatchedResourceType `json:"watchedResourceTypes,omitempty"`
}

// WatchedResourceType specifies a Kubernetes resource to watch
type WatchedResourceType struct {
    // Name is a human-friendly label (e.g., "MCPServer", "Agent", "Gateway")
    Name string `json:"name"`
    
    // APIGroup is the API group (e.g., "kagent.dev", "gateway.networking.k8s.io")
    APIGroup string `json:"apiGroup"`
    
    // Version is the API version (e.g., "v1", "v1alpha1", "v1alpha2")
    Version string `json:"version"`
    
    // Resource is the resource name (plural, lowercase) (e.g., "mcpservers", "agents", "gateways")
    Resource string `json:"resource"`
    
    // Enabled controls whether this resource type is actively watched
    // Allows disabling without removing the config
    Enabled *bool `json:"enabled,omitempty"`
}
```

### 2. CRD Example

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-policy
spec:
  requireAgentGateway: true
  requireCORS: true
  requireJWTAuth: true
  requireRBAC: true
  
  # Dynamic resource type configuration
  watchedResourceTypes:
    # Gateway API resources
    - name: "Gateway"
      apiGroup: "gateway.networking.k8s.io"
      version: "v1"
      resource: "gateways"
      enabled: true
    
    - name: "HTTPRoute"
      apiGroup: "gateway.networking.k8s.io"
      version: "v1"
      resource: "httproutes"
      enabled: true
    
    # AgentGateway resources
    - name: "AgentgatewayBackend"
      apiGroup: "agentgateway.dev"
      version: "v1alpha1"
      resource: "agentgatewaybackends"
      enabled: true
    
    - name: "AgentgatewayPolicy"
      apiGroup: "agentgateway.dev"
      version: "v1alpha1"
      resource: "agentgatewaypolicies"
      enabled: true
    
    # Kagent resources
    - name: "MCPServer"
      apiGroup: "kagent.dev"
      version: "v1alpha2"
      resource: "mcpservers"
      enabled: true
    
    - name: "RemoteMCPServer"
      apiGroup: "kagent.dev"
      version: "v1alpha2"
      resource: "remotemcpservers"
      enabled: true
    
    - name: "Agent"
      apiGroup: "kagent.dev"
      version: "v1alpha2"
      resource: "agents"
      enabled: true
    
    # MCPServerCatalog for inventory scoring
    - name: "MCPServerCatalog"
      apiGroup: "agentregistry.dev"
      version: "v1alpha1"
      resource: "mcpservercatalogs"
      enabled: true
    
    # Optional: Custom resources could be added here
    - name: "CustomSecurityPolicy"
      apiGroup: "security.example.com"
      version: "v1"
      resource: "customsecuritypolicies"
      enabled: false  # Disabled for now
```

### 3. Discovery Logic Implementation

**File:** `controller/pkg/watcher/discovery.go` (new file)

```go
package watcher

import (
	"fmt"
	"log"
	"sort"

	v1alpha1 "github.com/techwithhuz/mcp-security-governance/controller/pkg/apis/governance/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DiscoverWatchedResources builds the list of resources to watch from the policy
func DiscoverWatchedResources(
	policyResourceTypes []v1alpha1.WatchedResourceType,
) ([]WatchedResource, error) {
	
	// If policy specifies resource types, use those
	if len(policyResourceTypes) > 0 {
		return buildResourcesFromPolicy(policyResourceTypes)
	}
	
	// Otherwise, use built-in defaults
	return getDefaultWatchedResources(), nil
}

// buildResourcesFromPolicy converts CRD resource type specs to WatchedResource structs
func buildResourcesFromPolicy(
	policyResourceTypes []v1alpha1.WatchedResourceType,
) ([]WatchedResource, error) {
	var resources []WatchedResource
	seenResources := make(map[string]bool)
	
	for _, rt := range policyResourceTypes {
		// Validate required fields
		if err := validateResourceType(rt); err != nil {
			return nil, fmt.Errorf("invalid resource type %q: %w", rt.Name, err)
		}
		
		// Skip if disabled
		if rt.Enabled != nil && !*rt.Enabled {
			log.Printf("[discovery] Skipping disabled resource type: %s", rt.Name)
			continue
		}
		
		// Skip duplicates
		key := fmt.Sprintf("%s/%s/%s", rt.APIGroup, rt.Version, rt.Resource)
		if seenResources[key] {
			log.Printf("[discovery] WARNING: Duplicate resource type %s, skipping", key)
			continue
		}
		seenResources[key] = true
		
		// Convert to WatchedResource
		wr := WatchedResource{
			Label: rt.Name,
			GVR: schema.GroupVersionResource{
				Group:    rt.APIGroup,
				Version:  rt.Version,
				Resource: rt.Resource,
			},
		}
		resources = append(resources, wr)
	}
	
	if len(resources) == 0 {
		return nil, fmt.Errorf("no enabled resource types found in policy")
	}
	
	// Sort for deterministic ordering
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Label < resources[j].Label
	})
	
	log.Printf("[discovery] Discovered %d resource types from policy", len(resources))
	return resources, nil
}

// validateResourceType checks that all required fields are present
func validateResourceType(rt v1alpha1.WatchedResourceType) error {
	if rt.Name == "" {
		return fmt.Errorf("name is required")
	}
	if rt.APIGroup == "" {
		return fmt.Errorf("apiGroup is required")
	}
	if rt.Version == "" {
		return fmt.Errorf("version is required")
	}
	if rt.Resource == "" {
		return fmt.Errorf("resource is required")
	}
	return nil
}

// getDefaultWatchedResources returns the built-in defaults (fallback)
func getDefaultWatchedResources() []WatchedResource {
	return []WatchedResource{
		{
			Label: "Gateway",
			GVR: schema.GroupVersionResource{
				Group:    "gateway.networking.k8s.io",
				Version:  "v1",
				Resource: "gateways",
			},
		},
		// ... all other defaults ...
	}
}

// ResourceTypeStats provides summary information about discovered resources
type ResourceTypeStats struct {
	Total     int
	Enabled   int
	Disabled  int
	Resources []ResourceTypeInfo
}

// ResourceTypeInfo provides details about a single resource type
type ResourceTypeInfo struct {
	Name     string
	APIGroup string
	Version  string
	Resource string
	Enabled  bool
}

// GetResourceTypeStats returns statistics about discovered resource types
func GetResourceTypeStats(
	policyResourceTypes []v1alpha1.WatchedResourceType,
) ResourceTypeStats {
	stats := ResourceTypeStats{
		Total:     len(policyResourceTypes),
		Resources: make([]ResourceTypeInfo, 0, len(policyResourceTypes)),
	}
	
	for _, rt := range policyResourceTypes {
		enabled := rt.Enabled == nil || *rt.Enabled
		if enabled {
			stats.Enabled++
		} else {
			stats.Disabled++
		}
		
		stats.Resources = append(stats.Resources, ResourceTypeInfo{
			Name:     rt.Name,
			APIGroup: rt.APIGroup,
			Version:  rt.Version,
			Resource: rt.Resource,
			Enabled:  enabled,
		})
	}
	
	return stats
}
```

### 4. Watcher Integration

**File:** `controller/pkg/watcher/watcher.go` (modifications)

Update the `New()` function to accept watched resource types:

```go
// Config holds the configuration for the ResourceWatcher
type Config struct {
	// ...existing fields...
	
	// WatchedResourceTypes defines resources to watch from policy
	// If empty, uses DefaultWatchedResources()
	WatchedResourceTypes []v1alpha1.WatchedResourceType `json:"watchedResourceTypes,omitempty"`
}

// New creates and returns a new ResourceWatcher with dynamic resource discovery
func New(cfg Config) (*ResourceWatcher, error) {
	if cfg.DynamicClient == nil {
		return nil, fmt.Errorf("DynamicClient is required")
	}
	if cfg.Reconcile == nil {
		return nil, fmt.Errorf("Reconcile callback is required")
	}
	
	// ... existing validation ...
	
	// Discover resources from policy or use defaults
	var watchedGVRs []WatchedResource
	var err error
	
	if len(cfg.WatchedResourceTypes) > 0 {
		log.Printf("[watcher] Discovering resource types from policy")
		watchedGVRs, err = DiscoverWatchedResources(cfg.WatchedResourceTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to discover resources from policy: %w", err)
		}
	} else {
		log.Printf("[watcher] Using default resource types")
		watchedGVRs = DefaultWatchedResources()
	}
	
	if cfg.WatchedResources != nil {
		watchedGVRs = cfg.WatchedResources
	}
	
	return &ResourceWatcher{
		dynClient:    cfg.DynamicClient,
		reconcile:    cfg.Reconcile,
		debounce:     cfg.Debounce,
		resyncPeriod: cfg.ResyncPeriod,
		watchedGVRs:  watchedGVRs,
		stopCh:       make(chan struct{}),
	}, nil
}
```

### 5. Main Controller Integration

**File:** `controller/cmd/api/main.go` (modifications)

Update the resource watcher initialization to pass policy resource types:

```go
// Start resource watcher with policy-defined resource types
if discoverer != nil {
    // Load the policy first
    policy = loadPolicy()
    
    w, err := watcher.New(watcher.Config{
        DynamicClient: discoverer.DynamicClient(),
        Reconcile: func(reason string) {
            doPeriodicScan()
        },
        Debounce:     3 * time.Second,
        ResyncPeriod: scanInterval,
        
        // Pass watched resource types from policy
        WatchedResourceTypes: policy.WatchedResourceTypes,
    })
    if err != nil {
        log.Printf("[governance] WARNING: Failed to create resource watcher: %v", err)
        scanMode = "poll"
        startPollingLoop()
    } else {
        resourceWatcher = w
        scanMode = "watch"
        log.Printf("[governance] Watch mode enabled with policy-defined resource types")
        go resourceWatcher.Start(context.Background())
    }
}
```

## Usage Examples

### Example 1: Basic Policy with Resource Type Discovery

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: standard-policy
spec:
  requireAgentGateway: true
  watchedResourceTypes:
    - name: "MCPServer"
      apiGroup: "kagent.dev"
      version: "v1alpha2"
      resource: "mcpservers"
    - name: "Agent"
      apiGroup: "kagent.dev"
      version: "v1alpha2"
      resource: "agents"
```

### Example 2: Minimal Policy (Uses Defaults)

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: default-policy
spec:
  requireAgentGateway: true
  # watchedResourceTypes omitted → uses built-in defaults
```

### Example 3: Extended Monitoring with Custom Resources

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: extended-policy
spec:
  requireAgentGateway: true
  watchedResourceTypes:
    # Standard resources
    - name: "MCPServer"
      apiGroup: "kagent.dev"
      version: "v1alpha2"
      resource: "mcpservers"
      enabled: true
    
    # Custom security resource
    - name: "SecurityPolicy"
      apiGroup: "security.io"
      version: "v1"
      resource: "securitypolicies"
      enabled: true
    
    # Temporarily disabled
    - name: "Audit"
      apiGroup: "audit.io"
      version: "v1"
      resource: "audits"
      enabled: false
```

## Implementation Steps

### Phase 1: CRD Update
1. Add `WatchedResourceType` struct to `types.go`
2. Add `WatchedResourceTypes` field to `MCPGovernancePolicySpec`
3. Update CRD YAML file in `charts/mcp-security-governance/crds/`

### Phase 2: Discovery Logic
1. Create `controller/pkg/watcher/discovery.go`
2. Implement `DiscoverWatchedResources()` function
3. Add validation and error handling
4. Add unit tests

### Phase 3: Watcher Integration
1. Update `Config` struct in `watcher.go`
2. Modify `New()` function to accept policy resource types
3. Add logging for discovered resources
4. Update `DefaultWatchedResources()` to be fallback

### Phase 4: Controller Integration
1. Pass policy resource types to watcher in `main.go`
2. Update initialization logging
3. Add error handling for invalid policy configs

### Phase 5: Testing & Documentation
1. Create unit tests for discovery logic
2. Create integration tests for policy-driven watching
3. Update README with examples
4. Create migration guide for existing users

## Benefits

✅ **Flexibility** — Add/remove resources without code changes  
✅ **Policy-driven** — Governance decisions control monitoring scope  
✅ **Backward Compatible** — Falls back to defaults if omitted  
✅ **Validation** — Validates all required fields  
✅ **Extensible** — Easy to add custom resources  
✅ **Observable** — Clear logging of discovered resources  

## Backward Compatibility

- If `watchedResourceTypes` is omitted → uses `DefaultWatchedResources()`
- Existing deployments continue to work unchanged
- No breaking changes to CRD API

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Empty resource list | Error: "no enabled resource types found" |
| Missing required field | Error: "invalid resource type" |
| Invalid apiGroup format | Error: "apiGroup is required" |
| Duplicate resources | Logged as warning, second one skipped |
| Disabled resource | Silently skipped |

## Future Enhancements

1. **Resource Filtering** — Filter by namespace or label selectors
2. **Regex Patterns** — Support wildcard resource matching
3. **Dynamic Reloading** — Reload watched resources when policy changes
4. **Per-Resource Policies** — Different scoring for different resource types
5. **Resource Aliasing** — Map custom CRDs to standard governance checks

## Migration Path

**For Existing Users:**

Option 1: Do nothing (keeps working with defaults)

Option 2: Migrate to policy-defined resources:
```bash
# 1. Add watchedResourceTypes to your MCPGovernancePolicy
kubectl patch mcpgovernancepolicy enterprise-policy -p '{"spec":{"watchedResourceTypes":[...]}}'

# 2. Restart controller
kubectl rollout restart deployment/mcp-governance-controller
```

Option 3: Create new policy with custom resource types

## Testing Strategy

### Unit Tests
- Test `validateResourceType()` with valid/invalid inputs
- Test `buildResourcesFromPolicy()` with various configs
- Test duplicate detection
- Test sorting determinism

### Integration Tests
- Deploy policy with resource types
- Verify watcher starts with correct resources
- Verify changes to policy update watched resources
- Test fallback to defaults

### E2E Tests
- Create custom resource
- Add to watchedResourceTypes
- Verify controller picks up changes
- Verify scoring reflects new resource

## Documentation

Create new documentation:
- `docs/DYNAMIC_RESOURCE_DISCOVERY.md` — Feature overview and examples
- Update `README.md` — Add section on configurable resource types
- Create example policies showing different configurations

---

## Implementation Checklist

- [ ] Add `WatchedResourceType` struct to types.go
- [ ] Add `WatchedResourceTypes` field to MCPGovernancePolicySpec
- [ ] Create `discovery.go` with discovery logic
- [ ] Update `watcher.go` Config struct
- [ ] Update `New()` function to use discovery
- [ ] Update `main.go` to pass resource types to watcher
- [ ] Add unit tests for discovery
- [ ] Add integration tests
- [ ] Update CRD YAML
- [ ] Update README with examples
- [ ] Create migration guide
- [ ] Create example policies
- [ ] Create feature documentation

---

**This design enables flexible, policy-driven resource discovery while maintaining backward compatibility and providing clear error handling and validation.**
