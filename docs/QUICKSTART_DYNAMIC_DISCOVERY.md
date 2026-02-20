# Quick Start: Implementing Dynamic Resource Discovery

## Step-by-Step Implementation

### Step 1: Update the CRD Types (5 minutes)

**File:** `controller/pkg/apis/governance/v1alpha1/types.go`

Add the new struct before `MCPGovernancePolicySpec`:

```go
// WatchedResourceType specifies a Kubernetes resource to watch
type WatchedResourceType struct {
    // Name is a human-friendly label
    Name string `json:"name"`
    // APIGroup is the API group (e.g., "kagent.dev")
    APIGroup string `json:"apiGroup"`
    // Version is the API version (e.g., "v1alpha2")
    Version string `json:"version"`
    // Resource is the plural resource name (e.g., "mcpservers")
    Resource string `json:"resource"`
    // Enabled controls whether this resource is actively watched
    Enabled *bool `json:"enabled,omitempty"`
}
```

Then update `MCPGovernancePolicySpec` to add:

```go
type MCPGovernancePolicySpec struct {
    // ...existing fields...
    
    // WatchedResourceTypes defines which Kubernetes resources to monitor
    WatchedResourceTypes []WatchedResourceType `json:"watchedResourceTypes,omitempty"`
}
```

### Step 2: Create Discovery Logic (10 minutes)

**File:** `controller/pkg/watcher/discovery.go` (NEW FILE)

Copy the full implementation from the design document above. Key functions:
- `DiscoverWatchedResources()` — Main entry point
- `buildResourcesFromPolicy()` — Converts policy to WatchedResource list
- `validateResourceType()` — Validates required fields

### Step 3: Update Watcher Config (5 minutes)

**File:** `controller/pkg/watcher/watcher.go`

Add to `Config` struct:

```go
type Config struct {
    // ...existing fields...
    
    // WatchedResourceTypes defines resources to watch from policy
    WatchedResourceTypes []v1alpha1.WatchedResourceType `json:"watchedResourceTypes,omitempty"`
}
```

Update the `New()` function initialization section:

```go
// Discover resources from policy or use defaults
var watchedGVRs []WatchedResource

if len(cfg.WatchedResourceTypes) > 0 {
    log.Printf("[watcher] Discovering resource types from policy")
    var err error
    watchedGVRs, err = DiscoverWatchedResources(cfg.WatchedResourceTypes)
    if err != nil {
        return nil, fmt.Errorf("failed to discover resources: %w", err)
    }
} else {
    log.Printf("[watcher] Using default resource types")
    watchedGVRs = DefaultWatchedResources()
}

if cfg.WatchedResources != nil {
    watchedGVRs = cfg.WatchedResources
}

// Continue with existing code...
return &ResourceWatcher{
    dynClient:   cfg.DynamicClient,
    // ...rest of initialization...
    watchedGVRs: watchedGVRs,
}
```

### Step 4: Update Main Controller (5 minutes)

**File:** `controller/cmd/api/main.go`

Update the resource watcher initialization:

```go
// Around line 115-130, update:
if discoverer != nil {
    w, err := watcher.New(watcher.Config{
        DynamicClient: discoverer.DynamicClient(),
        Reconcile: func(reason string) {
            doPeriodicScan()
        },
        Debounce:     3 * time.Second,
        ResyncPeriod: scanInterval,
        
        // Add this line (pass policy resource types):
        WatchedResourceTypes: policy.WatchedResourceTypes,
    })
    if err != nil {
        log.Printf("[governance] WARNING: Failed to create resource watcher: %v", err)
        scanMode = "poll"
        startPollingLoop()
    } else {
        resourceWatcher = w
        scanMode = "watch"
        log.Printf("[governance] Watch mode enabled with dynamic resource discovery")
        go resourceWatcher.Start(context.Background())
    }
}
```

### Step 5: Add Import Statement (1 minute)

**File:** `controller/pkg/watcher/watcher.go`

Add to imports at top of file:

```go
import (
    // ...existing imports...
    v1alpha1 "github.com/techwithhuz/mcp-security-governance/controller/pkg/apis/governance/v1alpha1"
)
```

### Step 6: Build and Test (5 minutes)

```bash
cd controller
go mod tidy
go build -o /tmp/test-controller ./cmd/api
```

### Step 7: Test with Policy

Create a test policy YAML:

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: dynamic-test
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
    - name: "Gateway"
      apiGroup: "gateway.networking.k8s.io"
      version: "v1"
      resource: "gateways"
```

Apply and watch logs:
```bash
kubectl apply -f test-policy.yaml
kubectl logs -n default deployment/mcp-governance-controller -f | grep discovery
```

You should see:
```
[discovery] Discovered 3 resource types from policy
[watcher] Discovering resource types from policy
[watcher] Watching MCPServer (mcpservers)
[watcher] Watching Agent (agents)
[watcher] Watching Gateway (gateways)
```

## Key Points

✅ **Backward Compatible** — If `watchedResourceTypes` is omitted, uses defaults  
✅ **Validation** — All required fields checked with clear error messages  
✅ **Logging** — Detailed logs for debugging resource discovery  
✅ **Extensible** — Easy to add custom resources  

## Testing Checklist

- [ ] Build succeeds with no errors
- [ ] Existing policies (without watchedResourceTypes) still work
- [ ] New policy with resourceTypes is recognized
- [ ] Controller logs show "Discovering resource types from policy"
- [ ] All specified resources are being watched
- [ ] Changes to policy are reflected in controller behavior

## Troubleshooting

**Error: "no enabled resource types found"**
→ Check that you have at least one resource with `enabled: true` or no `enabled` field

**Error: "invalid resource type"**
→ Verify all required fields are present: `name`, `apiGroup`, `version`, `resource`

**Log shows "Using default resource types"**
→ You haven't set `watchedResourceTypes` in the policy (this is OK, defaults are used)

**New resources not being watched**
→ Restart controller after updating policy
→ Check logs for any validation errors

## Next: Write Unit Tests

Create `controller/pkg/watcher/discovery_test.go`:

```go
package watcher

import (
    "testing"
    v1alpha1 "github.com/techwithhuz/mcp-security-governance/controller/pkg/apis/governance/v1alpha1"
)

func TestDiscoverWatchedResources(t *testing.T) {
    resourceTypes := []v1alpha1.WatchedResourceType{
        {
            Name:     "TestResource",
            APIGroup: "example.com",
            Version:  "v1",
            Resource: "testresources",
        },
    }
    
    resources, err := DiscoverWatchedResources(resourceTypes)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if len(resources) != 1 {
        t.Fatalf("expected 1 resource, got %d", len(resources))
    }
    
    if resources[0].Label != "TestResource" {
        t.Errorf("expected label 'TestResource', got %q", resources[0].Label)
    }
}

func TestValidateResourceType(t *testing.T) {
    tests := []struct {
        name    string
        rt      v1alpha1.WatchedResourceType
        wantErr bool
    }{
        {
            name: "valid",
            rt: v1alpha1.WatchedResourceType{
                Name:     "Test",
                APIGroup: "example.com",
                Version:  "v1",
                Resource: "tests",
            },
            wantErr: false,
        },
        {
            name: "missing name",
            rt: v1alpha1.WatchedResourceType{
                APIGroup: "example.com",
                Version:  "v1",
                Resource: "tests",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateResourceType(tt.rt)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, want err %v", err, tt.wantErr)
            }
        })
    }
}
```

## Timeline

- **Phase 1 (CRD):** 5 minutes
- **Phase 2 (Discovery):** 10 minutes  
- **Phase 3 (Watcher):** 5 minutes
- **Phase 4 (Main):** 5 minutes
- **Phase 5 (Import):** 1 minute
- **Phase 6-7 (Build/Test):** 10 minutes
- **Total: ~40 minutes**

This implementation is completely optional and can be done incrementally without breaking existing functionality!
