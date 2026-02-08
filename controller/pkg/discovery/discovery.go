package discovery

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/techwithhuz/mcp-security-governance/controller/pkg/evaluator"
)

// K8sDiscoverer discovers resources from a real Kubernetes cluster
type K8sDiscoverer struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewK8sDiscoverer creates a new discoverer with in-cluster or kubeconfig auth
func NewK8sDiscoverer() (*K8sDiscoverer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig for local dev
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &K8sDiscoverer{
		clientset:     clientset,
		dynamicClient: dynClient,
	}, nil
}

// DiscoverClusterState discovers all relevant resources from the cluster
func (d *K8sDiscoverer) DiscoverClusterState(ctx context.Context) *evaluator.ClusterState {
	state := &evaluator.ClusterState{}

	// Discover namespaces
	state.Namespaces = d.discoverNamespaces(ctx)

	// Discover Gateway API resources
	state.Gateways = d.discoverGateways(ctx)
	state.HTTPRoutes = d.discoverHTTPRoutes(ctx)

	// Discover agentgateway CRDs
	state.AgentgatewayBackends = d.discoverAgentgatewayBackends(ctx)
	state.AgentgatewayPolicies = d.discoverAgentgatewayPolicies(ctx)

	// Discover kagent CRDs
	state.KagentAgents = d.discoverKagentAgents(ctx)
	state.KagentMCPServers = d.discoverKagentMCPServers(ctx)
	state.KagentRemoteMCPServers = d.discoverKagentRemoteMCPServers(ctx)

	// Discover Services with MCP labels/appProtocol
	state.Services = d.discoverServices(ctx)

	log.Printf("[discovery] Found: %d gateways, %d backends, %d policies, %d routes, %d agents, %d mcpservers, %d remote-mcpservers, %d services, %d namespaces",
		len(state.Gateways), len(state.AgentgatewayBackends), len(state.AgentgatewayPolicies),
		len(state.HTTPRoutes), len(state.KagentAgents), len(state.KagentMCPServers),
		len(state.KagentRemoteMCPServers), len(state.Services), len(state.Namespaces))

	return state
}

// discoverNamespaces lists all namespaces
func (d *K8sDiscoverer) discoverNamespaces(ctx context.Context) []string {
	nsList, err := d.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] Failed to list namespaces: %v", err)
		return []string{"default"}
	}
	var namespaces []string
	for _, ns := range nsList.Items {
		// Skip kube-system and other internal namespaces for governance
		name := ns.Name
		if name == "kube-system" || name == "kube-public" || name == "kube-node-lease" || name == "local-path-storage" {
			continue
		}
		namespaces = append(namespaces, name)
	}
	return namespaces
}

// discoverGateways discovers Gateway API Gateway resources
func (d *K8sDiscoverer) discoverGateways(ctx context.Context) []evaluator.GatewayResource {
	gvr := schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "gateways",
	}
	return d.discoverGatewayResources(ctx, gvr)
}

func (d *K8sDiscoverer) discoverGatewayResources(ctx context.Context, gvr schema.GroupVersionResource) []evaluator.GatewayResource {
	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] Gateway API not available: %v", err)
		return nil
	}

	var gateways []evaluator.GatewayResource
	for _, item := range list.Items {
		gw := evaluator.GatewayResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		spec, _ := getNestedMap(item.Object, "spec")
		if spec != nil {
			gw.GatewayClassName, _ = getNestedString(spec, "gatewayClassName")

			listeners, _ := getNestedSlice(spec, "listeners")
			for _, l := range listeners {
				lm, ok := l.(map[string]interface{})
				if !ok {
					continue
				}
				li := evaluator.ListenerInfo{}
				li.Name, _ = getNestedString(lm, "name")
				li.Protocol, _ = getNestedString(lm, "protocol")
				port, _ := getNestedInt(lm, "port")
				li.Port = int(port)
				gw.Listeners = append(gw.Listeners, li)
			}
		}

		// Check status for programmed condition
		status, _ := getNestedMap(item.Object, "status")
		if status != nil {
			conditions, _ := getNestedSlice(status, "conditions")
			for _, c := range conditions {
				cm, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				t, _ := getNestedString(cm, "type")
				s, _ := getNestedString(cm, "status")
				if t == "Programmed" && s == "True" {
					gw.Programmed = true
				}
			}
		}

		gateways = append(gateways, gw)
	}
	return gateways
}

// discoverHTTPRoutes discovers HTTPRoute resources
func (d *K8sDiscoverer) discoverHTTPRoutes(ctx context.Context) []evaluator.HTTPRouteResource {
	gvr := schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "httproutes",
	}

	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] HTTPRoute CRD not available: %v", err)
		return nil
	}

	var routes []evaluator.HTTPRouteResource
	for _, item := range list.Items {
		hr := evaluator.HTTPRouteResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		spec, _ := getNestedMap(item.Object, "spec")
		if spec != nil {
			// Get parent gateway
			parentRefs, _ := getNestedSlice(spec, "parentRefs")
			for _, pr := range parentRefs {
				pm, ok := pr.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := getNestedString(pm, "name")
				hr.ParentGateway = name
			}

			// Get backend refs
			rules, _ := getNestedSlice(spec, "rules")
			for _, r := range rules {
				rm, ok := r.(map[string]interface{})
				if !ok {
					continue
				}
				backendRefs, _ := getNestedSlice(rm, "backendRefs")
				for _, br := range backendRefs {
					bm, ok := br.(map[string]interface{})
					if !ok {
						continue
					}
					name, _ := getNestedString(bm, "name")
					hr.BackendRefs = append(hr.BackendRefs, name)
				}

				// Check for CORS filter
				filters, _ := getNestedSlice(rm, "filters")
				for _, f := range filters {
					fm, ok := f.(map[string]interface{})
					if !ok {
						continue
					}
					fType, _ := getNestedString(fm, "type")
					if fType == "ExtensionRef" {
						hr.HasCORSFilter = true
					}
				}
			}
		}

		routes = append(routes, hr)
	}
	return routes
}

// discoverAgentgatewayBackends discovers AgentgatewayBackend CRs
func (d *K8sDiscoverer) discoverAgentgatewayBackends(ctx context.Context) []evaluator.AgentgatewayBackendResource {
	gvr := schema.GroupVersionResource{
		Group:    "agentgateway.dev",
		Version:  "v1alpha1",
		Resource: "agentgatewaybackends",
	}

	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] AgentgatewayBackend CRD not available: %v", err)
		return nil
	}

	var backends []evaluator.AgentgatewayBackendResource
	for _, item := range list.Items {
		b := evaluator.AgentgatewayBackendResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		spec, _ := getNestedMap(item.Object, "spec")
		if spec != nil {
			b.BackendType, _ = getNestedString(spec, "type")

			// Check MCP targets
			mcp, _ := getNestedMap(spec, "mcp")
			if mcp != nil {
				targets, _ := getNestedSlice(mcp, "targets")
				for _, t := range targets {
					tm, ok := t.(map[string]interface{})
					if !ok {
						continue
					}
					target := evaluator.MCPTargetInfo{}
					target.Name, _ = getNestedString(tm, "name")
					target.Host, _ = getNestedString(tm, "host")
					port, _ := getNestedInt(tm, "port")
					target.Port = int(port)

					// Check for auth config
					auth, _ := getNestedMap(tm, "authentication")
					if auth != nil {
						target.HasAuth = true
					}
					// Check for RBAC
					authz, _ := getNestedMap(tm, "authorization")
					if authz != nil {
						target.HasRBAC = true
					}

					b.MCPTargets = append(b.MCPTargets, target)
				}
			}

			// Check TLS
			policies, _ := getNestedMap(spec, "policies")
			if policies != nil {
				tls, _ := getNestedMap(policies, "tls")
				if tls != nil {
					b.HasTLS = true
				}
			}
		}

		backends = append(backends, b)
	}
	return backends
}

// discoverAgentgatewayPolicies discovers AgentgatewayPolicy CRs
func (d *K8sDiscoverer) discoverAgentgatewayPolicies(ctx context.Context) []evaluator.AgentgatewayPolicyResource {
	gvr := schema.GroupVersionResource{
		Group:    "agentgateway.dev",
		Version:  "v1alpha1",
		Resource: "agentgatewaypolicies",
	}

	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] AgentgatewayPolicy CRD not available: %v", err)
		return nil
	}

	var policies []evaluator.AgentgatewayPolicyResource
	for _, item := range list.Items {
		p := evaluator.AgentgatewayPolicyResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		spec, _ := getNestedMap(item.Object, "spec")
		if spec != nil {
			// Target refs
			targetRef, _ := getNestedMap(spec, "targetRef")
			if targetRef != nil {
				tr := evaluator.PolicyTargetRef{}
				tr.Group, _ = getNestedString(targetRef, "group")
				tr.Kind, _ = getNestedString(targetRef, "kind")
				tr.Name, _ = getNestedString(targetRef, "name")
				p.TargetRefs = append(p.TargetRefs, tr)
			}

			// Check default policies
			defaults, _ := getNestedMap(spec, "default")
			if defaults != nil {
				// JWT
				jwt, _ := getNestedMap(defaults, "jwt")
				if jwt != nil {
					p.HasJWT = true
					p.JWTMode = "Strict" // default
					mode, _ := getNestedString(jwt, "mode")
					if mode != "" {
						p.JWTMode = mode
					}
				}

				// CORS
				cors, _ := getNestedMap(defaults, "cors")
				if cors != nil {
					p.HasCORS = true
				}

				// CSRF
				csrf, _ := getNestedMap(defaults, "csrf")
				if csrf != nil {
					p.HasCSRF = true
				}

				// Rate limit
				rateLimit, _ := getNestedMap(defaults, "rateLimit")
				if rateLimit != nil {
					p.HasRateLimit = true
				}

				// RBAC
				rbac, _ := getNestedMap(defaults, "rbac")
				if rbac != nil {
					p.HasRBAC = true
				}

				// Prompt guard
				pg, _ := getNestedMap(defaults, "promptGuard")
				if pg != nil {
					p.HasPromptGuard = true
				}
			}
		}

		policies = append(policies, p)
	}
	return policies
}

// discoverKagentAgents discovers kagent Agent CRs (v1alpha2)
func (d *K8sDiscoverer) discoverKagentAgents(ctx context.Context) []evaluator.KagentAgentResource {
	gvr := schema.GroupVersionResource{
		Group:    "kagent.dev",
		Version:  "v1alpha2",
		Resource: "agents",
	}

	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] Kagent Agent CRD not available: %v", err)
		return nil
	}

	var agents []evaluator.KagentAgentResource
	for _, item := range list.Items {
		a := evaluator.KagentAgentResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		spec, _ := getNestedMap(item.Object, "spec")
		if spec != nil {
			// Tools can be under spec.tools or spec.declarative.tools
			var tools []interface{}
			if t, ok := getNestedSlice(spec, "tools"); ok {
				tools = t
			} else if decl, ok := getNestedMap(spec, "declarative"); ok {
				tools, _ = getNestedSlice(decl, "tools")
			}

			for _, t := range tools {
				tm, ok := t.(map[string]interface{})
				if !ok {
					continue
				}
				toolRef := evaluator.KagentToolRef{}
				toolRef.Type, _ = getNestedString(tm, "type")

				mcpServer, _ := getNestedMap(tm, "mcpServer")
				if mcpServer != nil {
					// Real kagent uses flat apiGroup/kind/name in mcpServer
					toolRef.Kind, _ = getNestedString(mcpServer, "kind")
					toolRef.Name, _ = getNestedString(mcpServer, "name")

					// Also check nested ref (some versions use ref subobject)
					if toolRef.Kind == "" {
						ref, _ := getNestedMap(mcpServer, "ref")
						if ref != nil {
							toolRef.Kind, _ = getNestedString(ref, "kind")
							toolRef.Name, _ = getNestedString(ref, "name")
						}
					}

					toolNamesList, _ := getNestedSlice(mcpServer, "toolNames")
					for _, tn := range toolNamesList {
						if s, ok := tn.(string); ok {
							toolRef.ToolNames = append(toolRef.ToolNames, s)
						}
					}
				}

				a.Tools = append(a.Tools, toolRef)
			}
		}

		// Check status for ready condition
		status, _ := getNestedMap(item.Object, "status")
		if status != nil {
			conditions, _ := getNestedSlice(status, "conditions")
			for _, c := range conditions {
				cm, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				t, _ := getNestedString(cm, "type")
				s, _ := getNestedString(cm, "status")
				if t == "Ready" && s == "True" {
					a.Ready = true
				}
			}
		}

		agents = append(agents, a)
	}
	return agents
}

// discoverKagentMCPServers discovers kagent MCPServer CRs (v1alpha1)
func (d *K8sDiscoverer) discoverKagentMCPServers(ctx context.Context) []evaluator.KagentMCPServerResource {
	gvr := schema.GroupVersionResource{
		Group:    "kagent.dev",
		Version:  "v1alpha1",
		Resource: "mcpservers",
	}

	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] Kagent MCPServer CRD not available: %v", err)
		return nil
	}

	var servers []evaluator.KagentMCPServerResource
	for _, item := range list.Items {
		s := evaluator.KagentMCPServerResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		spec, _ := getNestedMap(item.Object, "spec")
		if spec != nil {
			// Check transport type
			if _, exists := spec["stdioTransport"]; exists {
				s.Transport = "stdio"
			}
			if sseTransport, exists := spec["sseTransport"]; exists {
				s.Transport = "sse"
				if st, ok := sseTransport.(map[string]interface{}); ok {
					port, _ := getNestedInt(st, "port")
					s.Port = int(port)
				}
			}
			if _, exists := spec["streamableHttpTransport"]; exists {
				s.Transport = "streamablehttp"
			}
		}

		servers = append(servers, s)
	}
	return servers
}

// discoverKagentRemoteMCPServers discovers kagent RemoteMCPServer CRs (v1alpha2)
func (d *K8sDiscoverer) discoverKagentRemoteMCPServers(ctx context.Context) []evaluator.KagentRemoteMCPServerResource {
	gvr := schema.GroupVersionResource{
		Group:    "kagent.dev",
		Version:  "v1alpha2",
		Resource: "remotemcpservers",
	}

	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] Kagent RemoteMCPServer CRD not available: %v", err)
		return nil
	}

	var servers []evaluator.KagentRemoteMCPServerResource
	for _, item := range list.Items {
		s := evaluator.KagentRemoteMCPServerResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		spec, _ := getNestedMap(item.Object, "spec")
		if spec != nil {
			s.URL, _ = getNestedString(spec, "url")
		}

		// Count discovered tools from status
		status, _ := getNestedMap(item.Object, "status")
		if status != nil {
			discoveredTools, ok := getNestedSlice(status, "discoveredTools")
			if ok {
				s.ToolCount = len(discoveredTools)
			}
		}

		servers = append(servers, s)
	}
	return servers
}

// discoverServices discovers Services that may be MCP endpoints
func (d *K8sDiscoverer) discoverServices(ctx context.Context) []evaluator.ServiceResource {
	svcList, err := d.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] Failed to list services: %v", err)
		return nil
	}

	var services []evaluator.ServiceResource
	for _, svc := range svcList.Items {
		// Skip kube-system services
		if svc.Namespace == "kube-system" || svc.Namespace == "kube-public" || svc.Namespace == "local-path-storage" {
			continue
		}

		sr := evaluator.ServiceResource{
			Name:      svc.Name,
			Namespace: svc.Namespace,
		}

		// Check for MCP-related labels or appProtocol
		isMCP := false
		for _, port := range svc.Spec.Ports {
			sr.Ports = append(sr.Ports, int(port.Port))
			if port.AppProtocol != nil {
				sr.AppProtocol = *port.AppProtocol
				if strings.Contains(*port.AppProtocol, "mcp") || strings.Contains(*port.AppProtocol, "kgateway.dev/mcp") {
					isMCP = true
				}
			}
		}

		// Also check labels for MCP indicators
		for k, v := range svc.Labels {
			if strings.Contains(k, "mcp") || strings.Contains(v, "mcp") {
				isMCP = true
			}
		}

		sr.IsMCP = isMCP
		services = append(services, sr)
	}
	return services
}

// DiscoverGovernancePolicy discovers MCPGovernancePolicy resources
func (d *K8sDiscoverer) DiscoverGovernancePolicy(ctx context.Context) *evaluator.Policy {
	gvr := schema.GroupVersionResource{
		Group:    "governance.mcp.io",
		Version:  "v1alpha1",
		Resource: "mcpgovernancepolicies",
	}

	list, err := d.dynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("[discovery] Failed to list MCPGovernancePolicies: %v. Using default policy.", err)
		return nil
	}

	if len(list.Items) == 0 {
		log.Printf("[discovery] No MCPGovernancePolicy found. Using default policy.")
		return nil
	}

	// Use the first policy found (typically there should only be one cluster-wide policy)
	policyObj := list.Items[0]
	spec, found, err := unstructured.NestedMap(policyObj.Object, "spec")
	if !found || err != nil {
		log.Printf("[discovery] Failed to parse MCPGovernancePolicy spec: %v. Using default policy.", err)
		return nil
	}

	policy := &evaluator.Policy{}

	// Parse boolean requirements
	if val, ok := spec["requireAgentGateway"].(bool); ok {
		policy.RequireAgentGateway = val
	}
	if val, ok := spec["requireCORS"].(bool); ok {
		policy.RequireCORS = val
	}
	if val, ok := spec["requireJWTAuth"].(bool); ok {
		policy.RequireJWTAuth = val
	}
	if val, ok := spec["requireRBAC"].(bool); ok {
		policy.RequireRBAC = val
	}
	if val, ok := spec["requirePromptGuard"].(bool); ok {
		policy.RequirePromptGuard = val
	}
	if val, ok := spec["requireTLS"].(bool); ok {
		policy.RequireTLS = val
	}
	if val, ok := spec["requireRateLimit"].(bool); ok {
		policy.RequireRateLimit = val
	}

	// Parse tool count thresholds
	if val, ok := spec["maxToolsWarning"].(int64); ok {
		policy.MaxToolsWarning = int(val)
	}
	if val, ok := spec["maxToolsCritical"].(int64); ok {
		policy.MaxToolsCritical = int(val)
	}

	// Parse scoring weights
	if weightsMap, ok := spec["scoringWeights"].(map[string]interface{}); ok {
		weights := evaluator.ScoringWeights{}
		if val, ok := weightsMap["agentGatewayIntegration"].(int64); ok {
			weights.AgentGatewayIntegration = int(val)
		}
		if val, ok := weightsMap["authentication"].(int64); ok {
			weights.Authentication = int(val)
		}
		if val, ok := weightsMap["authorization"].(int64); ok {
			weights.Authorization = int(val)
		}
		if val, ok := weightsMap["corsPolicy"].(int64); ok {
			weights.CORSPolicy = int(val)
		}
		if val, ok := weightsMap["tlsEncryption"].(int64); ok {
			weights.TLSEncryption = int(val)
		}
		if val, ok := weightsMap["promptGuard"].(int64); ok {
			weights.PromptGuard = int(val)
		}
		if val, ok := weightsMap["rateLimit"].(int64); ok {
			weights.RateLimit = int(val)
		}
		if val, ok := weightsMap["toolScope"].(int64); ok {
			weights.ToolScope = int(val)
		}
		policy.Weights = weights
	}

	// Parse severity penalties
	if penaltiesMap, ok := spec["severityPenalties"].(map[string]interface{}); ok {
		penalties := evaluator.DefaultSeverityPenalties()
		if val, ok := penaltiesMap["critical"].(int64); ok {
			penalties.Critical = int(val)
		}
		if val, ok := penaltiesMap["high"].(int64); ok {
			penalties.High = int(val)
		}
		if val, ok := penaltiesMap["medium"].(int64); ok {
			penalties.Medium = int(val)
		}
		if val, ok := penaltiesMap["low"].(int64); ok {
			penalties.Low = int(val)
		}
		policy.SeverityPenalties = penalties
	} else {
		policy.SeverityPenalties = evaluator.DefaultSeverityPenalties()
	}

	log.Printf("[discovery] Loaded MCPGovernancePolicy: %s/%s", policyObj.GetNamespace(), policyObj.GetName())
	return policy
}

// ---- helper functions for unstructured data access ----

func getNestedMap(obj map[string]interface{}, keys ...string) (map[string]interface{}, bool) {
	current := obj
	for _, key := range keys {
		val, ok := current[key]
		if !ok {
			return nil, false
		}
		m, ok := val.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = m
	}
	return current, true
}

func getNestedString(obj map[string]interface{}, key string) (string, bool) {
	val, ok := obj[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

func getNestedInt(obj map[string]interface{}, key string) (int64, bool) {
	val, ok := obj[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}

func getNestedSlice(obj map[string]interface{}, key string) ([]interface{}, bool) {
	val, ok := obj[key]
	if !ok {
		return nil, false
	}
	s, ok := val.([]interface{})
	return s, ok
}

// Silence unused import warning
var _ = unstructured.Unstructured{}
