package evaluator

import (
	"testing"
)

// TestCheckHardenedDeployment_NoWorkloads tests HDN-000 finding
func TestCheckHardenedDeployment_NoWorkloads(t *testing.T) {
	state := &ClusterState{
		Workloads:       []WorkloadResource{},
		NetworkPolicies: []NetworkPolicyResource{},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	if len(findings) == 0 {
		t.Error("Expected HDN-000 finding for no workloads, got none")
		return
	}

	hdn000 := findings[0]
	if hdn000.ID != "HDN-000" {
		t.Errorf("Expected HDN-000, got %s", hdn000.ID)
	}
	if hdn000.Severity != SeverityHigh {
		t.Errorf("Expected High severity, got %s", hdn000.Severity)
	}
	if hdn000.Category != CategoryHardening {
		t.Errorf("Expected Hardening category, got %s", hdn000.Category)
	}
}

// TestCheckHardenedDeployment_ContainerRunAsRoot tests HDN-001 finding
func TestCheckHardenedDeployment_ContainerRunAsRoot(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "vulnerable-app",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   false, // Running as root
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           false,
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:           "test-policy",
				Namespace:      "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	// Should have HDN-001 finding
	found := false
	for _, f := range findings {
		if f.ID == "HDN-001-vulnerable-app" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("HDN-001 should be Critical, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-001 finding for container running as root, not found")
	}
}

// TestCheckHardenedDeployment_WriteableRootFS tests HDN-002 finding
func TestCheckHardenedDeployment_WriteableRootFS(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-writeable-fs",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: false, // Writeable FS
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           false,
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-002-app-writeable-fs" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("HDN-002 should be High, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-002 finding for writeable root filesystem, not found")
	}
}

// TestCheckHardenedDeployment_PrivilegeEscalation tests HDN-003 finding
func TestCheckHardenedDeployment_PrivilegeEscalation(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-privesc",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: false, // Privilege escalation allowed
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           false,
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-003-app-privesc" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("HDN-003 should be High, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-003 finding for privilege escalation, not found")
	}
}

// TestCheckHardenedDeployment_CapabilitiesNotDropped tests HDN-004 finding
func TestCheckHardenedDeployment_CapabilitiesNotDropped(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-caps",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: false, // Capabilities not dropped
				SeccompProfileSet:      true,
				HasLatestTag:           false,
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-004-app-caps" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("HDN-004 should be Medium, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-004 finding for capabilities not dropped, not found")
	}
}

// TestCheckHardenedDeployment_NoSeccompProfile tests HDN-005 finding
func TestCheckHardenedDeployment_NoSeccompProfile(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-noseccomp",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      false, // No seccomp profile
				HasLatestTag:           false,
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-005-app-noseccomp" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("HDN-005 should be Medium, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-005 finding for no seccomp profile, not found")
	}
}

// TestCheckHardenedDeployment_LatestTag tests HDN-006 finding
func TestCheckHardenedDeployment_LatestTag(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-latest",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           true, // Uses :latest tag
				ImageNames:             []string{"nginx:latest"},
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-006-app-latest" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("HDN-006 should be Medium, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-006 finding for :latest tag, not found")
	}
}

// TestCheckHardenedDeployment_NoNetworkPolicy tests HDN-007 finding
func TestCheckHardenedDeployment_NoNetworkPolicy(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-nonetpol",
				Namespace:              "unprotected-ns",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           false,
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			// No NetworkPolicy for "unprotected-ns"
			{
				Name:            "policy-other-ns",
				Namespace:       "other-ns",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-007-unprotected-ns" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("HDN-007 should be Critical, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-007 finding for no NetworkPolicy, not found")
	}
}

// TestCheckHardenedDeployment_PlaintextSecrets tests HDN-008 finding
func TestCheckHardenedDeployment_PlaintextSecrets(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-secrets",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           false,
				HasPlaintextEnvSecrets: true, // Plaintext secrets in env
				PlaintextEnvVarNames:   []string{"DATABASE_PASSWORD", "API_KEY"},
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-008-app-secrets" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("HDN-008 should be High, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-008 finding for plaintext secrets, not found")
	}
}

// TestCheckHardenedDeployment_NoVaultESO tests HDN-009 finding
func TestCheckHardenedDeployment_NoVaultESO(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-novault",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           false,
				HasPlaintextEnvSecrets: false,
				HasVaultInjection:      false, // No Vault
				HasESOAnnotation:       false, // No ESO
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-009-app-novault" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("HDN-009 should be Medium, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-009 finding for no Vault/ESO, not found")
	}
}

// TestCheckHardenedDeployment_NoImageSignature tests HDN-010 finding
func TestCheckHardenedDeployment_NoImageSignature(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "app-nosig",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll: true,
				SeccompProfileSet:      true,
				HasLatestTag:           false,
				HasPlaintextEnvSecrets: false,
				HasVaultInjection:      true,
				HasESOAnnotation:       false,
				HasImageSignature:      false, // No image signature
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "test-policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	found := false
	for _, f := range findings {
		if f.ID == "HDN-010-app-nosig" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("HDN-010 should be Medium, got %s", f.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected HDN-010 finding for no image signature, not found")
	}
}

// TestCheckHardenedDeployment_FullyHardened tests no findings for a fully hardened deployment
func TestCheckHardenedDeployment_FullyHardened(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                      "hardened-app",
				Namespace:                 "default",
				Kind:                      "Deployment",
				AllContainersNonRoot:      true,
				AllContainersReadOnlyRootFS: true,
				AllContainersNoPrivEscalation: true,
				AllContainersCapDropAll:   true,
				SeccompProfileSet:         true,
				HasLatestTag:              false,
				HasPlaintextEnvSecrets:    false,
				HasVaultInjection:         true,
				HasESOAnnotation:          false,
				HasImageSignature:         true,
				ImageNames:                []string{"nginx:1.27-alpine"},
			},
		},
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "allow-hardened",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: true,
	}

	findings := checkHardenedDeployment(state, policy)

	// Should have no hardening findings (only HDN-000 is expected if workloads is empty, but it's not)
	hdnFindings := 0
	for _, f := range findings {
		if f.Category == CategoryHardening {
			hdnFindings++
		}
	}

	if hdnFindings > 0 {
		t.Errorf("Expected no hardening findings for fully hardened deployment, got %d", hdnFindings)
	}
}

// TestCheckHardenedDeployment_Disabled tests that no findings are returned when disabled
func TestCheckHardenedDeployment_Disabled(t *testing.T) {
	state := &ClusterState{
		Workloads: []WorkloadResource{
			{
				Name:                   "vulnerable-app",
				Namespace:              "default",
				Kind:                   "Deployment",
				AllContainersNonRoot:   false, // Vulnerable but checks disabled
			},
		},
	}
	policy := Policy{
		RequireHardenedDeployment: false, // Disabled
	}

	findings := checkHardenedDeployment(state, policy)

	if len(findings) > 0 {
		t.Errorf("Expected no findings when hardening is disabled, got %d", len(findings))
	}
}

// BenchmarkCheckHardenedDeployment benchmarks the hardening check performance
func BenchmarkCheckHardenedDeployment(b *testing.B) {
	state := &ClusterState{
		Workloads: make([]WorkloadResource, 100),
		NetworkPolicies: []NetworkPolicyResource{
			{
				Name:            "policy",
				Namespace:       "default",
				HasIngressRules: true,
				HasEgressRules: true,
			},
		},
	}

	// Create 100 workloads with mixed hardening postures
	for i := 0; i < 100; i++ {
		state.Workloads[i] = WorkloadResource{
			Name:                      "app-" + string(rune('0'+i%10)),
			Namespace:                 "default",
			Kind:                      "Deployment",
			AllContainersNonRoot:      i%2 == 0,
			AllContainersReadOnlyRootFS: i%3 == 0,
			AllContainersNoPrivEscalation: i%4 == 0,
			AllContainersCapDropAll:   i%5 == 0,
			SeccompProfileSet:         i%6 == 0,
			HasLatestTag:              i%7 == 0,
			HasPlaintextEnvSecrets:    i%8 == 0,
			HasVaultInjection:         i%9 == 0,
			HasImageSignature:         i%10 == 0,
		}
	}

	policy := Policy{
		RequireHardenedDeployment: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checkHardenedDeployment(state, policy)
	}
}
