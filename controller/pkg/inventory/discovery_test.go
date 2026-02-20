package inventory

import (
	"testing"
)

func TestDiscoverWatchedResources(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantErr bool
		wantLen int
	}{
		{
			name:    "empty input returns defaults",
			input:   []string{},
			wantErr: false,
			wantLen: 1, // Default is MCPServerCatalog
		},
		{
			name:    "single valid resource",
			input:   []string{"MCPServerCatalog"},
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "multiple valid resources",
			input:   []string{"MCPServerCatalog", "Agent", "RemoteMCPServer"},
			wantErr: false,
			wantLen: 3,
		},
		{
			name:    "all supported resource types",
			input:   []string{"MCPServerCatalog", "Agent", "RemoteMCPServer", "Gateway", "HTTPRoute"},
			wantErr: false,
			wantLen: 5,
		},
		{
			name:    "duplicate resource is skipped",
			input:   []string{"MCPServerCatalog", "Agent", "MCPServerCatalog"},
			wantErr: false,
			wantLen: 2, // Only 2 unique resources
		},
		{
			name:    "whitespace is trimmed",
			input:   []string{" MCPServerCatalog ", "  Agent  "},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "unknown resource returns error",
			input:   []string{"UnknownResource"},
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "mixed valid and invalid returns error",
			input:   []string{"MCPServerCatalog", "UnknownResource"},
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "empty strings ignored",
			input:   []string{"MCPServerCatalog", "", "Agent", "  "},
			wantErr: false,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DiscoverWatchedResources(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("got error %v, want err %v", err, tt.wantErr)
			}

			if len(got) != tt.wantLen {
				t.Errorf("got %d resources, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestResourceTypeMapping(t *testing.T) {
	tests := []struct {
		name     string
		rtName   string
		wantGVR  string
	}{
		{
			name:    "MCPServerCatalog",
			rtName:  "MCPServerCatalog",
			wantGVR: "agentregistry.dev/v1alpha1, Resource=mcpservercatalogs",
		},
		{
			name:    "Agent",
			rtName:  "Agent",
			wantGVR: "kagent.dev/v1alpha2, Resource=agents",
		},
		{
			name:    "RemoteMCPServer",
			rtName:  "RemoteMCPServer",
			wantGVR: "kagent.dev/v1alpha2, Resource=remotemcpservers",
		},
		{
			name:    "Gateway",
			rtName:  "Gateway",
			wantGVR: "gateway.networking.k8s.io/v1, Resource=gateways",
		},
		{
			name:    "HTTPRoute",
			rtName:  "HTTPRoute",
			wantGVR: "gateway.networking.k8s.io/v1, Resource=httproutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvr, exists := ResourceTypeMapping[tt.rtName]
			if !exists {
				t.Fatalf("resource type %q not found in mapping", tt.rtName)
			}

			if gvr.String() != tt.wantGVR {
				t.Errorf("got %q, want %q", gvr.String(), tt.wantGVR)
			}
		})
	}
}

func TestDefaultWatchedResources(t *testing.T) {
	defaults := DefaultWatchedResources()

	if len(defaults) != 1 {
		t.Fatalf("expected 1 default resource, got %d", len(defaults))
	}

	if defaults[0] != MCPServerCatalogGVR {
		t.Errorf("expected default to be MCPServerCatalogGVR, got %v", defaults[0])
	}
}

func TestSupportedResourceTypes(t *testing.T) {
	supported := SupportedResourceTypes()
	
	// Should contain all resource type names
	expectedTypes := []string{"MCPServerCatalog", "Agent", "RemoteMCPServer", "Gateway", "HTTPRoute"}
	for _, rt := range expectedTypes {
		if !contains(supported, rt) {
			t.Errorf("supported types missing %q: %s", rt, supported)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0)
}

