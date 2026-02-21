# MCP Governance: AI-Powered Kubernetes-Native Security for Model Context Protocol Infrastructure

## Introduction

The Model Context Protocol (MCP) is transforming how AI agents interact with tools and data. By exposing MCP servers across your Kubernetes clusters, you unlock powerful AI capabilities — but without proper governance, you're exposed to critical security risks.

Imagine this scenario: An AI agent with access to an unprotected MCP server that exposes 50+ tools. A prompt injection attack tricks the agent into executing an unauthorized database dump tool, exfiltrating sensitive customer data. No authentication layer stops it. No rate limiting prevents the runaway loop. No audit trail captures what happened.

**MCP Governance (MCP-G)** solves this problem by providing enterprise-grade security and compliance for MCP infrastructure in Kubernetes. It's an AI-powered governance platform that discovers, evaluates, and secures all MCP-related resources in your cluster.

In this deep-dive, we'll explore how MCP-G works, why it matters, and how to deploy it to your Kubernetes cluster.

---

## The Problem: Unsecured MCP Servers in Kubernetes

### Why MCP Governance Matters

As AI agents become mainstream, organizations are deploying MCP servers across Kubernetes clusters to expose tools, databases, and APIs. However, this creates unprecedented security challenges:

**1. Zero Visibility**
- How many MCP servers are running in your cluster?
- Which agents have access to which tools?
- Are there unprotected servers exposed directly to the internet?
- Without discovery and inventory, you can't govern what you can't see.

**2. Unconstrained Access**
- MCP servers often expose 50+ tools without restrictions
- A compromised agent or prompt injection attack can invoke any tool
- No principle of least privilege — agents get "all or nothing"
- Without authorization controls, blast radius grows exponentially

**3. Prompt Injection Vulnerability**
- AI agents are susceptible to prompt injection attacks that trick them into executing unauthorized actions
- Malicious inputs can manipulate tool calls, triggering sensitive operations
- Sensitive data in tool responses (SSNs, credit cards, API keys) can be leaked to the AI model

**4. No Audit Trail**
- Who called which tools? When? From which agent?
- Without logging and monitoring, compliance and forensics are impossible
- Incident response becomes guesswork

**5. Configuration Drift**
- Security policies are defined in policy documents, not code
- Clusters drift from intended security posture over time
- No enforcement mechanism ensures policies are continuously applied

---

## The Solution: MCP Governance (MCP-G)

MCP Governance is a Kubernetes-native platform that:

1. **Discovers** all MCP-related resources in your cluster automatically
2. **Correlates** resources into MCP-server-centric views
3. **Evaluates** security posture across 8 governance categories
4. **Scores** cluster compliance on a 0–100 scale
5. **Analyzes** risks with AI agents (optional: Google Gemini or Ollama)
6. **Surfaces** findings in a real-time enterprise dashboard

---

## Architecture: How MCP-G Works

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐          ┌─────────────────────────────┐ │
│  │  🖥️ Dashboard    │          │  ⚙️ Governance Controller   │ │
│  │  Next.js :3000   │◄────────►│                             │ │
│  │  (every 15s)     │          │  • Go API Server :8090      │ │
│  └──────────────────┘          │  • Scoring Engine           │ │
│                                │  • AI Agent (Gemini/Ollama) │ │
│                                │  • Resource Discovery       │ │
│                                └──────────────────────────────┘ │
│                                         ▲                        │
│                                         │                        │
│                            ┌────────────┴──────────────┐         │
│                            │ list / watch (30s)        │         │
│                            ▼                            ▼         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │          📦 Discovered Resources                         │   │
│  │                                                          │   │
│  │  🛡️ AgentGateway    🤖 Kagent      🌐 Gateway API       │   │
│  │  • Backend          • Agent        • Gateway            │   │
│  │  • Policy           • MCPServer    • HTTPRoute          │   │
│  │  • Parameters       • RemoteMCPServer                  │   │
│  │                                                          │   │
│  │  📋 Governance Resources                               │   │
│  │  • MCPGovernancePolicy  • GovernanceEvaluation        │   │
│  │                                                          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌──────────────────┐                                          │
│  │  🌐 LLM Provider │                                          │
│  │  • Google Gemini │  (Optional AI analysis)                 │
│  │  • Ollama (local)│                                          │
│  └──────────────────┘                                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Discovery (Every 30 seconds)**
   - Controller uses Kubernetes API to list all relevant resources
   - Correlates AgentGateway, Kagent, Gateway API, and governance resources
   - Builds an MCP-server-centric inventory

2. **Policy Reading**
   - Controller reads `MCPGovernancePolicy` CRD
   - Determines scoring thresholds, AI configuration, and enforcement rules

3. **Evaluation**
   - Evaluator scores each MCP server across 8 governance categories
   - Calculates per-server score and cluster-level aggregate
   - Generates per-resource findings and recommendations

4. **AI Analysis (Optional)**
   - If enabled, sends cluster state to LLM (Gemini or Ollama)
   - AI provides deeper risk analysis and remediation suggestions
   - Results cached to avoid redundant API calls

5. **API & Status Update**
   - Results exposed via REST API
   - `GovernanceEvaluation` CRD updated with scoring details
   - Dashboard polls API and refreshes visualizations

---

## Scoring System: 8 Governance Categories

MCP-G evaluates security across 8 distinct categories, each with specific pass/fail criteria:

### 1. **Gateway Routing** (20 points)
**The Question:** Is the MCP server routed through AgentGateway?

- **Pass:** Server exposed through governed gateway → +20 points
- **Fail:** Direct exposure without gateway layer → 0 points

**Why It Matters:** Without a gateway, there's no central enforcement point. Any tool call reaches the server unfiltered.

### 2. **Authentication** (20 points)
**The Question:** Is JWT or mutual-TLS authentication enforced?

- **Pass:** JWT Strict mode enforced on the route → +20 points
- **Partial:** JWT present but permissive mode → +10 points
- **Fail:** No authentication required → 0 points

**Why It Matters:** Unauthenticated MCP servers let any agent or script invoke tools without identity verification.

### 3. **Authorization** (15 points)
**The Question:** Are tool calls filtered by an allow-list policy?

- **Pass:** Tool allow-list enforced via AgentgatewayPolicy → +15 points
- **Fail:** All tools exposed with no filter → 0 points

**Why It Matters:** Even authenticated agents should only access tools they need (principle of least privilege).

### 4. **TLS Encryption** (15 points)
**The Question:** Is TLS configured on backend connections?

- **Pass:** TLS with SNI verification enabled → +15 points
- **Fail:** Plaintext backend connection → 0 points

**Why It Matters:** Without TLS, tool payloads can be intercepted in transit inside the cluster.

### 5. **CORS Policy** (10 points)
**The Question:** Is CORS configured to restrict cross-origin access?

- **Pass:** CORS policy restricts allowed origins → +10 points
- **Partial:** CORS configured via policy → +8 points
- **Fail:** No CORS policy, all origins allowed → 0 points

**Why It Matters:** Without CORS, browser-based clients can be tricked into making unauthorized tool calls from malicious origins.

### 6. **Rate Limiting** (10 points)
**The Question:** Is rate limiting applied to prevent abuse?

- **Pass:** Rate limit policy applied → +10 points
- **Fail:** No rate limit, unlimited calls allowed → 0 points

**Why It Matters:** Agents in a runaway loop can make thousands of calls in seconds, causing disruption or data exfiltration.

### 7. **Prompt Guard** (10 points)
**The Question:** Is AI prompt injection protection enabled?

- **Pass:** Prompt guard with injection detection + data masking → +10 points
- **Partial:** Partial guard (request or response only) → +5 points
- **Fail:** No prompt protection → 0 points

**Why It Matters:** Prompt injection attacks trick AI agents into unauthorized actions. Sensitive data in responses must be masked.

### 8. **Tool Scope** (10 points)
**The Question:** How tightly is the exposed tool surface restricted?

- **Pass:** ≤25% of tools exposed (maximum restriction) → +10 points
- **Graduated:** Sliding scale based on exposure percentage
- **Fail:** 100% of tools exposed (no restriction) → 0 points

**Why It Matters:** Exposing all 50+ tools when an agent only needs 5 vastly increases the blast radius of compromise.

### Overall Scoring

- **Maximum Score:** 100 points
- **Per-Server Score:** Weighted average of category scores
- **Cluster Score:** Weighted average of all per-server scores
- **Grade Scale:**
  - **A (90–100):** Enterprise-grade security
  - **B (80–89):** Good security posture
  - **C (70–79):** Adequate controls, needs improvement
  - **D (60–69):** Significant gaps
  - **F (<60):** Critical security issues

---

## Verified Catalog Scoring

Beyond MCP server evaluation, MCP-G includes **Verified Catalog Scoring** — a framework for evaluating MCP server catalog entries from your Agent Registry.

### What It Evaluates

Each MCPServerCatalog is scored across 5 categories:

| Category | Max Points | What's Checked |
|----------|------------|----------------|
| **Publisher Verification** | 25 | Organization identity, signing certificates, credibility |
| **Transport Security** | 20 | TLS support, secure URLs, protocol compliance |
| **Deployment Health** | 20 | Published status, readiness state, management type |
| **Tool Scope** | 18 | Tool count compliance, exposure restrictions |
| **Usage & Integration** | 17 | Agent usage patterns, integration diversity |

### Scoring Breakdown

Each catalog entry receives:

- **Composite Score (0–100)** — Overall verification score
- **Grade (A–F)** — Letter grade based on thresholds
- **Status** — Verified, Unverified, Rejected, or Pending
- **Category Scores** — Individual scores for each category
- **Check Details** — Per-check pass/fail status with points

### Verification Checks

| Check | Category | What It Verifies |
|-------|----------|------------------|
| **PUB-001** | Publisher | Organization verification present |
| **PUB-002** | Publisher | Publisher identity verified |
| **SEC-001** | Transport | Remote URL uses HTTPS |
| **SEC-002** | Transport | Supported transport protocol |
| **DEP-001** | Deployment | Catalog published |
| **DEP-002** | Deployment | Deployment ready |
| **TOOL-001** | Tool Scope | Tool count within limits |
| **TOOL-002** | Tool Scope | Tool list available |
| **USAGE-001** | Usage | Agent integration present |
| **USAGE-002** | Usage | Multiple agent usage |

---

## Key Features

### 🔍 Comprehensive Discovery
- Automatically discovers all MCP-related resources (Kagent, AgentGateway, Gateway API)
- Monitors 9 resource types across multiple Kubernetes API groups
- Real-time inventory updates every 30 seconds

### 📊 MCP-Server-Centric View
- Correlates resources into per-server security profiles
- Shows which gateway routes serve each MCP server
- Identifies authorization policies and tool restrictions
- Displays agent consumers and tool usage patterns

### 🎯 8-Category Scoring System
- Evaluates Gateway routing, Authentication, Authorization, TLS, CORS, Rate limiting, Prompt guard, and Tool scope
- Weighted scoring model
- Per-server and cluster-level aggregation
- A–F grade scale with explanations

### 🧠 AI-Powered Risk Analysis
- Optional integration with Google Gemini or local Ollama
- Deeper risk assessment beyond algorithmic scoring
- Actionable remediation suggestions
- Reasoning and explanation for findings

### 📋 Verified Catalog Scoring
- Evaluates MCP server catalog entries
- 5-category assessment (publisher, transport, deployment, tools, usage)
- Automated compliance checking against governance policies
- Status tracking (Verified, Unverified, Rejected, Pending)

### 🖥️ Real-Time Dashboard
- Next.js-based interactive UI
- 6 main tabs: Overview, MCP Servers, Verified Catalog, Resource Inventory, Findings, About
- Per-server drill-down with detailed scoring
- Tool exposure metrics and risk indicators
- Auto-refresh every 15 seconds

### ✍️ Policy-as-Code
- `MCPGovernancePolicy` Kubernetes CRD for declarative governance
- CEL-based expressions for custom scoring logic
- Threshold configuration for pass/fail criteria
- Easy policy updates and versioning

### 📈 Comprehensive Findings
- Critical, High, Medium, Low severity classifications
- Per-check pass/fail status
- Remediation guidance
- Compliance tracking

### 🔌 REST API
- `/api/governance/overview` — Cluster-level score and summary
- `/api/governance/mcp-servers` — Per-server security details
- `/api/governance/findings` — Complete findings list
- `/api/governance/inventory/verified` — Catalog verification scores
- Full JSON response format for integration

---

## Prerequisites

Before deploying MCP-G, ensure you have:

### Required

- **Kubernetes Cluster** (1.24+)
  - Kind, EKS, GKE, AKS, or any CNCF-certified Kubernetes
  - Cluster admin access for CRD installation
  - Minimum: 2 CPU cores, 2GB RAM

- **kubectl** (1.24+)
  - Configured to access your cluster

- **Helm 3** (3.13+)
  - For deploying MCP-G via Helm chart

### Optional but Recommended

- **AgentGateway** (latest)
  - For MCP server routing and policy enforcement
  - [Installation Guide](https://agentgateway.dev)

- **Kagent** (latest)
  - For agent and MCPServer resources
  - [Installation Guide](https://kagent.dev)

- **Agent Registry** (latest)
  - For MCPServerCatalog resources
  - [Installation Guide](https://github.com/den-vasyliev/agentregistry-inventory)

- **LLM Provider** (for AI analysis)
  - **Google Cloud Account** with Gemini API enabled, OR
  - **Ollama** running locally or in cluster

- **DNS** (optional)
  - For accessing dashboard via domain name
  - Gateway API route with HTTP/HTTPS support

---

## Installation Guide

### Step 1: Add the MCP-G Helm Repository

```bash
helm repo add mcp-governance https://charts.techwithhuz.dev
helm repo update
```

### Step 2: Create Namespace

```bash
kubectl create namespace mcp-governance
```

### Step 3: Create AI Configuration Secret (Optional)

If using Google Gemini:

```bash
kubectl create secret generic gemini-config \
  --from-literal=api-key=YOUR_GEMINI_API_KEY \
  -n mcp-governance
```

If using Ollama:

```bash
kubectl create secret generic ollama-config \
  --from-literal=endpoint=http://ollama:11434 \
  -n mcp-governance
```

### Step 4: Deploy with Helm

**Quick Start (Development):**

```bash
helm install mcp-governance mcp-governance/mcp-governance \
  -n mcp-governance \
  --set environment=dev
```

**Production Deployment:**

```bash
helm install mcp-governance mcp-governance/mcp-governance \
  -n mcp-governance \
  --set environment=prod \
  --set controller.replicas=3 \
  --set dashboard.replicas=2 \
  --set ai.provider=gemini \
  --set ai.gemini.secretName=gemini-config
```

### Step 5: Verify Deployment

```bash
# Check if pods are running
kubectl get pods -n mcp-governance

# Expected output:
# NAME                                          READY   STATUS
# mcp-governance-controller-xxxx                1/1     Running
# mcp-governance-dashboard-xxxx                 1/1     Running

# Check logs
kubectl logs -n mcp-governance deployment/mcp-governance-controller -f

# Access dashboard
kubectl port-forward -n mcp-governance svc/mcp-governance-dashboard 3000:3000

# Then open http://localhost:3000
```

### Step 6: Create a Governance Policy

```bash
cat <<EOF | kubectl apply -f -
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-policy
  namespace: default
spec:
  enabled: true
  
  # Scoring weights
  weights:
    gatewayRouting: 20
    authentication: 20
    authorization: 15
    tlsEncryption: 15
    cors: 10
    rateLimit: 10
    promptGuard: 10
    toolScope: 10
  
  # Pass/fail thresholds
  thresholds:
    minScoreForPassing: 70
    criticalGatewayRequired: true
    tlsRequired: true
  
  # AI configuration (optional)
  aiScoring:
    enabled: true
    provider: gemini  # or "ollama"
    cacheResults: true
  
  # Verified catalog scoring
  verifiedCatalogScoring:
    enabled: true
    weights:
      security: 50
      trust: 30
      compliance: 20
EOF
```

### Step 7: Verify Governance Policy

```bash
# List policies
kubectl get mcpgovernancepolicies

# View results
kubectl describe mcpgovernancepolicies enterprise-policy

# Watch evaluation results
kubectl get governanceevaluations -w
```

---

## Dashboard Overview

Once deployed, access the dashboard at `http://localhost:3000` (or your configured domain).

### 📊 Overview Tab
- **Cluster Score:** Weighted average of all MCP server scores
- **Grade:** A–F letter grade
- **MCP Servers:** Count of discovered servers
- **Average Server Score:** Aggregate security posture
- **Critical Findings:** Count of critical-severity issues
- **Score Trend:** 7-day trend visualization

### 🔧 MCP Servers Tab
- Table view of all discovered MCP servers
- Per-server score with color-coded status
- Details: namespace, gateway routes, authentication method
- Expandable drill-down: Full scoring breakdown, tools exposed, connected agents
- Filter and sort by score, namespace, status

### ✓ Verified Catalog Tab
- List of MCPServerCatalog resources
- Score, grade, and verification status
- Security assessment per governance category
- Check results (pass/fail with points)
- Expandable details: Findings, tools, agent consumers

### 📦 Resource Inventory Tab
- Flat list of all discovered resources
- AgentGateway, Kagent, Gateway API, Governance resources
- Quick status indicators
- Resource consumption tracking

### 🚨 All Findings Tab
- Complete list of governance findings
- Severity filtering (Critical, High, Medium, Low)
- Per-server and per-resource organization
- Remediation guidance

### ℹ️ About MCP-G Tab
- Architecture diagram
- Prerequisites and installation links
- Features and scoring system explanation
- Kubernetes resources examples

---

## Configuration: MCPGovernancePolicy CRD

### Example: Basic Policy

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: basic-policy
spec:
  enabled: true
  weights:
    gatewayRouting: 20
    authentication: 20
    authorization: 15
    tlsEncryption: 15
    cors: 10
    rateLimit: 10
    promptGuard: 10
    toolScope: 10
```

### Example: Strict Policy (Production)

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: strict-policy
spec:
  enabled: true
  
  weights:
    gatewayRouting: 20
    authentication: 20
    authorization: 15
    tlsEncryption: 15
    cors: 10
    rateLimit: 10
    promptGuard: 10
    toolScope: 10
  
  thresholds:
    minScoreForPassing: 85
    criticalGatewayRequired: true
    tlsRequired: true
    authenticationRequired: true
    rateLimit:
      enabled: true
      requestsPerSecond: 10
```

### Example: With AI Scoring

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: ai-policy
spec:
  enabled: true
  
  aiScoring:
    enabled: true
    provider: gemini
    secretName: gemini-config
    cacheResults: true
    cacheTTL: 3600
    model: "gemini-pro"
    
  verifiedCatalogScoring:
    enabled: true
    weights:
      security: 50
      trust: 30
      compliance: 20
    thresholds:
      minScoreForVerified: 80
```

---

## API Reference

### Overview Endpoint

**Request:**
```bash
curl http://localhost:8090/api/governance/overview
```

**Response:**
```json
{
  "clusterScore": 82.5,
  "grade": "B",
  "totalMCPServers": 12,
  "averageServerScore": 78.3,
  "criticalFindings": 2,
  "highFindings": 8,
  "scoreTrend": [75, 76, 78, 80, 81, 82, 82.5]
}
```

### MCP Servers Endpoint

**Request:**
```bash
curl http://localhost:8090/api/governance/mcp-servers
```

**Response:**
```json
{
  "servers": [
    {
      "name": "grafana-mcp",
      "namespace": "default",
      "score": 85,
      "grade": "B",
      "scores": {
        "gatewayRouting": 20,
        "authentication": 20,
        "authorization": 15,
        "tlsEncryption": 12,
        "cors": 8,
        "rateLimit": 5,
        "promptGuard": 3,
        "toolScope": 2
      },
      "gatewayRoutes": ["grafana-route"],
      "toolsExposed": 12,
      "agentConsumers": ["agent-1", "agent-2"],
      "lastEvaluated": "2025-02-21T10:30:00Z"
    }
  ]
}
```

### Findings Endpoint

**Request:**
```bash
curl http://localhost:8090/api/governance/findings
```

**Response:**
```json
{
  "findings": [
    {
      "id": "finding-001",
      "server": "grafana-mcp",
      "severity": "High",
      "category": "Authorization",
      "title": "Tool Allow-List Not Configured",
      "description": "No authorization policy restricts tool access",
      "remediation": "Attach AgentgatewayPolicy with tool allow-list",
      "pointsLost": 15
    }
  ]
}
```

### Verified Catalog Endpoint

**Request:**
```bash
curl http://localhost:8090/api/governance/inventory/verified
```

**Response:**
```json
{
  "resources": [
    {
      "name": "grafana-mcp",
      "namespace": "default",
      "verifiedScore": {
        "score": 85,
        "grade": "B",
        "status": "Verified",
        "securityScore": 90,
        "trustScore": 75,
        "complianceScore": 85
      }
    }
  ]
}
```

---

## Use Cases

### 1. Enterprise Security Compliance
Ensure all MCP servers meet compliance standards (SOC 2, ISO 27001, etc.) with automated scoring and audit trails.

### 2. Multi-Tenant Isolation
Validate that each tenant's MCP servers are properly isolated with gateway routing and RBAC policies.

### 3. AI Agent Risk Assessment
Before deploying a new AI agent to production, evaluate which MCP servers it can access and their security posture.

### 4. Incident Response
When a security incident occurs, quickly identify which agents accessed which tools via the findings tab and audit logs.

### 5. Continuous Governance
Maintain security posture as new MCP servers are deployed — automatic discovery and evaluation prevent configuration drift.

### 6. Vendor Risk Management
Evaluate third-party MCP server catalogs before integrating them — Verified Catalog scoring ensures only trustworthy sources are used.

---

## Troubleshooting

### Controller Not Discovering Resources

**Symptom:** Dashboard shows 0 MCP servers found

**Solution:**
1. Check controller logs: `kubectl logs deployment/mcp-governance-controller -f`
2. Verify Kubernetes API connectivity: `kubectl auth can-i list mcpservers --as=system:serviceaccount:mcp-governance:controller`
3. Ensure Kagent, AgentGateway, and Governance CRDs are installed

### Dashboard Not Updating

**Symptom:** Dashboard shows stale data

**Solution:**
1. Check dashboard logs: `kubectl logs deployment/mcp-governance-dashboard -f`
2. Verify API endpoint is reachable: `curl http://localhost:8090/api/governance/overview`
3. Check browser console for network errors
4. Refresh browser and check API polling (Network tab)

### AI Scoring Errors

**Symptom:** "Failed to score with AI agent" errors

**Solution:**
1. Verify secret is created: `kubectl get secret gemini-config -n mcp-governance`
2. Test API key: `curl -H "Authorization: Bearer $API_KEY" https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent`
3. Check controller logs for LLM errors
4. Disable AI scoring temporarily: Set `aiScoring.enabled: false` in policy

---

## Best Practices

### 1. Start with Discovery
Deploy MCP-G first to discover your current MCP infrastructure. Don't enforce policies immediately.

### 2. Establish Baseline
Review the initial findings and scoring. Understand why scores are low before enforcing stricter policies.

### 3. Gradual Hardening
Incrementally increase policy strictness:
- Week 1: Discovery and baseline assessment
- Week 2: Enable soft enforcement (monitoring only)
- Week 3: Enable gateway routing requirement
- Week 4: Enable authentication and authorization
- Week 5: Enable TLS and rate limiting

### 4. Regular Policy Reviews
Review MCPGovernancePolicy quarterly to ensure it aligns with evolving security posture and business needs.

### 5. Use AI for Risk Analysis
Enable AI scoring to get deeper insights beyond algorithmic scoring — AI provides context and reasoning.

### 6. Monitor Trends
Watch the score trend over time. Declining scores indicate new security gaps; rising scores indicate improving posture.

### 7. Automate Remediation
Integrate findings with your incident management system to automate remediation workflows.

### 8. Document Exceptions
When a server can't meet a requirement, document the exception and timeline for remediation.

---

## Roadmap

### Current (v0.17.0)
- ✅ 9 resource type discovery
- ✅ 8-category scoring system
- ✅ AI-powered risk analysis (Gemini + Ollama)
- ✅ Verified catalog scoring
- ✅ REST API endpoints
- ✅ Real-time dashboard with 6 tabs
- ✅ MCPGovernancePolicy CRD

### Upcoming (v1.0.0)
- 🔄 Soft policy enforcement (warnings → errors)
- 🔄 Automated remediation suggestions with one-click apply
- 🔄 Compliance reports (SOC 2, ISO 27001, PCI-DSS)
- 🔄 Audit logging and forensics
- 🔄 Multi-cluster governance federation
- 🔄 Custom scoring extensions (plugins)
- 🔄 Webhook-based event notifications

---

## Conclusion

MCP Governance brings enterprise-grade security to Model Context Protocol infrastructure in Kubernetes. By combining automated discovery, multi-category scoring, and optional AI-powered risk analysis, MCP-G gives you visibility and control over your MCP ecosystem.

Whether you're running a few MCP servers or managing a multi-tenant infrastructure, MCP-G helps you:
- Discover and inventory all MCP resources
- Evaluate security posture across 8 categories
- Identify and remediate vulnerabilities
- Maintain continuous compliance
- Scale governance with your organization

**Get started today:**

1. Clone the repository: `git clone https://github.com/techwithhuz/mcp-security-governance`
2. Deploy to your cluster: `helm install mcp-governance ./charts/mcp-governance`
3. Access the dashboard and review your MCP security posture

For questions, issues, or contributions, visit [GitHub](https://github.com/techwithhuz/mcp-security-governance).

---

## Additional Resources

- **GitHub Repository:** [techwithhuz/mcp-security-governance](https://github.com/techwithhuz/mcp-security-governance)
- **Helm Chart:** `charts/mcp-governance`
- **CRD Documentation:** `charts/mcp-governance/crds/`
- **API Documentation:** [API Reference](#api-reference)
- **AgentGateway:** [agentgateway.dev](https://agentgateway.dev)
- **Kagent:** [kagent.dev](https://kagent.dev)
- **Model Context Protocol:** [modelcontextprotocol.io](https://modelcontextprotocol.io)

---

**Author:** Tech With Huz  
**Last Updated:** February 2025  
**License:** MIT
