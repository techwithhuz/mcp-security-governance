package inventory

import (
	"fmt"
	"log"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceTypeMapping defines the mapping from logical resource type names
// to Kubernetes GroupVersionResources. This allows policy to use simple names
// like "MCPServerCatalog" without needing to specify apiGroup, version, resource.
var ResourceTypeMapping = map[string]schema.GroupVersionResource{
	"MCPServerCatalog": {
		Group:    "agentregistry.dev",
		Version:  "v1alpha1",
		Resource: "mcpservercatalogs",
	},
	"Agent": {
		Group:    "kagent.dev",
		Version:  "v1alpha2",
		Resource: "agents",
	},
	"RemoteMCPServer": {
		Group:    "kagent.dev",
		Version:  "v1alpha2",
		Resource: "remotemcpservers",
	},
	"Gateway": {
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "gateways",
	},
	"HTTPRoute": {
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "httproutes",
	},
}

// DiscoverWatchedResources converts resource type names from the policy
// into Kubernetes GroupVersionResource objects that the watcher can use.
// Supported resource types: "MCPServerCatalog", "Agent", "RemoteMCPServer", "Gateway", "HTTPRoute"
func DiscoverWatchedResources(resourceTypes []string) ([]schema.GroupVersionResource, error) {
	if len(resourceTypes) == 0 {
		log.Printf("[discovery] No resource types configured, using defaults")
		return DefaultWatchedResources(), nil
	}

	var gvrs []schema.GroupVersionResource
	seen := make(map[string]bool) // Track to avoid duplicates

	for _, rtName := range resourceTypes {
		rtName = strings.TrimSpace(rtName) // Trim whitespace
		if rtName == "" {
			continue
		}

		if seen[rtName] {
			log.Printf("[discovery] Skipping duplicate resource type: %s", rtName)
			continue
		}

		gvr, exists := ResourceTypeMapping[rtName]
		if !exists {
			return nil, fmt.Errorf("unknown resource type %q (supported: %s)",
				rtName, SupportedResourceTypes())
		}

		gvrs = append(gvrs, gvr)
		seen[rtName] = true
		log.Printf("[discovery] Discovered resource: %s (%s)", rtName, gvr.String())
	}

	if len(gvrs) == 0 {
		log.Printf("[discovery] No valid resource types found, using defaults")
		return DefaultWatchedResources(), nil
	}

	log.Printf("[discovery] Discovered %d resource types from policy", len(gvrs))
	return gvrs, nil
}

// DefaultWatchedResources returns the default set of resources to watch if none
// are specified in the policy.
func DefaultWatchedResources() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		MCPServerCatalogGVR, // agentregistry.dev/v1alpha1/mcpservercatalogs
	}
}

// SupportedResourceTypes returns a comma-separated list of supported resource type names.
func SupportedResourceTypes() string {
	var names []string
	for name := range ResourceTypeMapping {
		names = append(names, fmt.Sprintf("%q", name))
	}
	return strings.Join(names, ", ")
}
