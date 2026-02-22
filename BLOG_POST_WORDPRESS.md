# WordPress Blog Post — Ready to Copy & Paste
# Format: WordPress Block Editor (Gutenberg) — Plain Text / Paragraph blocks
# SEO: Optimized for "MCP Governance", "Kubernetes AI Security", "Model Context Protocol Security"
# ─────────────────────────────────────────────────────────────────────────────

## SEO SETTINGS (Paste into your SEO plugin — Yoast / RankMath / AIOSEO)

**SEO Title:**
MCP Governance: AI-Powered Kubernetes Security for Model Context Protocol | Tech With Huz

**Meta Description (155 chars):**
Discover MCP Governance (MCP-G), the open-source Kubernetes-native platform that discovers, scores, and secures all your Model Context Protocol servers with AI-powered risk analysis.

**Focus Keyphrase:** MCP Governance Kubernetes Security

**Secondary Keywords:**
- Model Context Protocol security
- Kubernetes AI security
- MCP server governance
- AI agent security Kubernetes
- MCP-G open source

**Slug:** mcp-governance-kubernetes-ai-security

**Categories:** Kubernetes, AI Security, DevOps, Cloud Native

**Tags:** Kubernetes, MCP, Model Context Protocol, AI Agents, Security, DevSecOps, Helm, Go, Next.js, Open Source

---

## POST CONTENT — COPY EVERYTHING BELOW THIS LINE INTO WORDPRESS

---

# MCP Governance: AI-Powered Kubernetes-Native Security for Model Context Protocol Infrastructure

---

**Imagine this scenario.** An AI agent has access to an MCP server exposing 50+ tools. A prompt injection attack tricks it into dumping your customer database. No authentication stops it. No rate limiting prevents the runaway loop. No audit trail captures what happened.

That's the reality of unprotected Model Context Protocol (MCP) infrastructure in Kubernetes today — and it's exactly why I built **MCP Governance (MCP-G)**.

**MCP-G** monitors and secures resources from **AgentGateway**, **Kagent**, and **Agent Registry** — the core components that expose and manage MCP servers in your cluster. It automatically discovers these resources, evaluates their security posture, and provides actionable insights to harden your MCP infrastructure.

In this post, I'll walk you through the full architecture, scoring system, and deployment guide for MCP-G: an open-source, AI-powered Kubernetes-native governance platform for MCP infrastructure.

---

## Table of Contents

1. The Problem: Why MCP Governance Matters
2. The Solution: What Is MCP-G?
3. Architecture: How It Works
4. The 8-Category Scoring System
5. Verified Catalog Scoring
6. Key Features
7. Prerequisites
8. Step-by-Step Installation Guide
9. Dashboard Walkthrough
10. MCPGovernancePolicy CRD Reference
11. REST API Reference
12. Use Cases
13. Best Practices
14. Roadmap
15. Conclusion

---

## 1. The Problem: Why MCP Governance Matters

The Model Context Protocol is transforming how AI agents interact with tools, databases, and APIs. As organizations deploy MCP servers across Kubernetes clusters, they're unlocking powerful AI capabilities — but also creating serious security blind spots.

Here are the five biggest risks organizations face today:

**Zero Visibility**
How many MCP servers are running in your cluster? Which agents have access to which tools? Are there unprotected servers exposed to the internet? Without discovery and inventory, you cannot govern what you cannot see.

**Unconstrained Access**
MCP servers often expose 50+ tools with no restrictions. A compromised agent or prompt injection attack can invoke any tool. Without authorization controls, the blast radius of a compromise grows exponentially.

**Prompt Injection Vulnerability**
AI agents are susceptible to attacks that trick them into executing unauthorized actions. Sensitive data in tool responses — SSNs, credit card numbers, API keys — can be leaked directly into the model's context.

**No Audit Trail**
Who called which tools? When? From which agent? Without logging, compliance and forensics are impossible. Incident response becomes guesswork.

**Configuration Drift**
Security policies are often defined in documents, not code. Clusters drift from their intended security posture over time, and without enforcement, policies are just suggestions.

---

## 2. The Solution: What Is MCP-G?

**MCP Governance (MCP-G)** is a Kubernetes-native platform that:

✅ **Discovers** all MCP-related resources in your cluster automatically
✅ **Correlates** resources into MCP-server-centric security views
✅ **Evaluates** security posture across 8 governance categories
✅ **Scores** cluster compliance on a 0–100 scale
✅ **Analyzes** risks with AI agents (Google Gemini or local Ollama)
✅ **Surfaces** findings in a real-time enterprise dashboard

It's built in Go for the controller, Next.js for the dashboard, deployed via Helm, and fully open source. **Note:** MCP-G monitors resources from AgentGateway, Kagent, and Agent Registry, so these platforms should be deployed in your cluster for MCP-G to discover and evaluate your MCP infrastructure.

🔗 **GitHub:** https://github.com/techwithhuz/mcp-g

---

## 3. Architecture: How MCP-G Works

MCP-G follows a clean controller-dashboard architecture inside your Kubernetes cluster.

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐          ┌────────────────────────────┐  │
│  │  🖥️ Dashboard    │          │  ⚙️ Governance Controller  │  │
│  │  Next.js :3000   │◄────────►│                            │  │
│  │  (every 15s)     │          │  • Go API Server :8090     │  │
│  └──────────────────┘          │  • Scoring Engine          │  │
│                                │  • AI Agent (Gemini/Ollama)│  │
│                                │  • Resource Discovery      │  │
│                                └────────────────────────────┘  │
│                                         ▲                       │
│                            ┌────────────┴──────────────┐        │
│                            │  list / watch (30s)       │        │
│                            ▼                           ▼        │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  📦 AgentGateway · Kagent · Gateway API · Governance    │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow (5 Steps)

**Step 1 — Discovery (Every 30 seconds)**
The controller uses the Kubernetes API to list all relevant resources: AgentGateway, Kagent, Gateway API, and governance CRDs. It then correlates these into an MCP-server-centric inventory.

**Step 2 — Policy Reading**
The controller reads your `MCPGovernancePolicy` CRD to determine scoring weights, AI configuration, and enforcement rules.

**Step 3 — Evaluation**
The scoring engine evaluates each MCP server across 8 governance categories, calculates per-server scores, and generates findings with remediation guidance.

**Step 4 — AI Analysis (Optional)**
If enabled, the cluster state is sent to an LLM (Google Gemini or local Ollama) for deeper risk analysis and remediation suggestions. Results are cached to avoid redundant API calls.

**Step 5 — API & Status Update**
Results are exposed via REST API, the `GovernanceEvaluation` CRD is updated, and the dashboard auto-refreshes every 15 seconds.

---

## 4. The 8-Category Scoring System

MCP-G evaluates every MCP server across 8 security categories, producing a composite score out of 100.

### Category 1: Gateway Routing — 20 Points

**The Question:** Is the MCP server routed through AgentGateway?

- ✅ Pass: Server exposed through a governed gateway → +20 points
- ❌ Fail: Direct exposure without a gateway layer → 0 points

Without a gateway, there is no central enforcement point. Any tool call reaches the server unfiltered.

---

### Category 2: Authentication — 20 Points

**The Question:** Is JWT or mutual-TLS authentication enforced?

- ✅ Pass: JWT Strict mode enforced on the route → +20 points
- ⚠️ Partial: JWT present but in permissive mode → +10 points
- ❌ Fail: No authentication required → 0 points

Unauthenticated MCP servers let any agent or script invoke tools without identity verification.

---

### Category 3: Authorization — 15 Points

**The Question:** Are tool calls filtered by an allow-list policy?

- ✅ Pass: Tool allow-list enforced via AgentgatewayPolicy → +15 points
- ❌ Fail: All tools exposed with no filter → 0 points

Even authenticated agents should only access the tools they need — principle of least privilege.

---

### Category 4: TLS Encryption — 15 Points

**The Question:** Is TLS configured on backend connections?

- ✅ Pass: TLS with SNI verification enabled → +15 points
- ❌ Fail: Plaintext backend connection → 0 points

Without TLS, tool payloads can be intercepted in transit inside the cluster.

---

### Category 5: CORS Policy — 10 Points

**The Question:** Is CORS configured to restrict cross-origin access?

- ✅ Pass: CORS policy restricts allowed origins → +10 points
- ⚠️ Partial: CORS configured via policy → +8 points
- ❌ Fail: No CORS policy, all origins allowed → 0 points

Without CORS restrictions, browser-based clients can be tricked into making unauthorized tool calls from malicious origins.

---

### Category 6: Rate Limiting — 10 Points

**The Question:** Is rate limiting applied to prevent abuse?

- ✅ Pass: Rate limit policy applied → +10 points
- ❌ Fail: No rate limit, unlimited calls allowed → 0 points

Agents in a runaway loop can make thousands of calls per second, causing disruption or mass data exfiltration.

---

### Category 7: Prompt Guard — 10 Points

**The Question:** Is AI prompt injection protection enabled?

- ✅ Pass: Prompt guard with injection detection + data masking → +10 points
- ⚠️ Partial: Request or response guard only → +5 points
- ❌ Fail: No prompt protection → 0 points

Prompt injection attacks trick AI agents into unauthorized actions. Sensitive data in responses must be masked before it enters the model context.

---

### Category 8: Tool Scope — 10 Points

**The Question:** How tightly is the exposed tool surface restricted?

- ✅ Pass: ≤25% of tools exposed → +10 points
- 📉 Graduated: Sliding scale based on exposure percentage
- ❌ Fail: 100% of tools exposed → 0 points

Exposing all 50+ tools when an agent only needs 5 vastly increases the blast radius of any compromise.

---

### Grade Scale

| Score | Grade | Meaning |
|-------|-------|---------|
| 90–100 | A | Enterprise-grade security |
| 80–89 | B | Good security posture |
| 70–79 | C | Adequate controls, needs improvement |
| 60–69 | D | Significant gaps |
| < 60 | F | Critical security issues |

---

## 5. Verified Catalog Scoring

Beyond MCP server evaluation, MCP-G includes **Verified Catalog Scoring** — a framework for evaluating MCPServerCatalog entries from your Agent Registry.

Each catalog entry is scored across 5 categories:

| Category | Max Points | What's Checked |
|----------|------------|----------------|
| Publisher Verification | 25 | Organization identity, signing certificates |
| Transport Security | 20 | TLS support, secure URLs |
| Deployment Health | 20 | Published status, readiness state |
| Tool Scope | 18 | Tool count compliance, restrictions |
| Usage & Integration | 17 | Agent usage patterns, diversity |

Each entry receives a composite score (0–100), a letter grade (A–F), and a verification status: **Verified**, **Unverified**, **Rejected**, or **Pending**.

---

## 6. Key Features

🔍 **Comprehensive Discovery** — Automatically discovers all MCP-related resources across 9 resource types, with real-time updates every 30 seconds.

📊 **MCP-Server-Centric View** — Correlates resources into per-server security profiles showing gateway routes, authorization policies, agent consumers, and tool usage.

🎯 **8-Category Weighted Scoring** — Evaluates all critical governance dimensions with per-server and cluster-level aggregation.

🧠 **AI-Powered Risk Analysis** — Optional integration with Google Gemini or local Ollama for deeper risk assessment and actionable remediation.

📋 **Verified Catalog Scoring** — 5-category assessment for MCPServerCatalog entries with automated compliance checking.

🖥️ **Real-Time Dashboard** — Next.js-based UI with 6 tabs (Overview, MCP Servers, Verified Catalog, Resource Inventory, Findings, About), auto-refreshing every 15 seconds.

✍️ **Policy-as-Code** — Declarative `MCPGovernancePolicy` Kubernetes CRD with threshold configuration and easy versioning.

📈 **Comprehensive Findings** — Critical / High / Medium / Low severity findings with remediation guidance and compliance tracking.

🔌 **REST API** — Full JSON API for integration with existing security tooling and dashboards.

---

## 7. Prerequisites

### Required

- **Kubernetes 1.24+** — Kind, EKS, GKE, AKS, or any CNCF-certified cluster
- **kubectl 1.24+** — Configured with cluster admin access
- **Helm 3.13+** — For Helm-based deployment

### Optional but Recommended

- **AgentGateway** — For MCP server routing and policy enforcement
- **Kagent** — For Agent and MCPServer resources
- **Agent Registry** — For MCPServerCatalog resources
- **Google Gemini API** or **Ollama** — For AI-powered risk analysis

---

## 8. Step-by-Step Installation Guide

### Step 1: Add the Helm Repository

```bash
helm repo add mcp-governance https://charts.techwithhuz.dev
helm repo update
```

### Step 2: Create the Namespace

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

### Step 5: Verify the Deployment

```bash
# Check pods
kubectl get pods -n mcp-governance

# Check controller logs
kubectl logs -n mcp-governance deployment/mcp-governance-controller -f

# Access the dashboard
kubectl port-forward -n mcp-governance svc/mcp-governance-dashboard 3000:3000
# Then open http://localhost:3000
```

### Step 6: Create a Governance Policy

```yaml
apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-policy
  namespace: default
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
    minScoreForPassing: 70
    criticalGatewayRequired: true
    tlsRequired: true
  aiScoring:
    enabled: true
    provider: gemini
    cacheResults: true
```

Apply it:

```bash
kubectl apply -f enterprise-policy.yaml
```

### Step 7: Verify Policy and Evaluations

```bash
# List policies
kubectl get mcpgovernancepolicies

# Watch evaluation results
kubectl get governanceevaluations -w
```

---

## 9. Dashboard Walkthrough

Once deployed, open `http://localhost:3000` to access the dashboard.

**Overview Tab** — Cluster score, grade, MCP server count, critical findings, and a 7-day score trend.

**MCP Servers Tab** — Table of all discovered servers with per-server score, namespace, gateway routes, and authentication method. Click any server to expand the full scoring breakdown, tool exposure details, and connected agents.

**Verified Catalog Tab** — MCPServerCatalog resources with score, grade, verification status, and per-category check results.

**Resource Inventory Tab** — Flat list of all discovered Kubernetes resources (AgentGateway, Kagent, Gateway API, Governance CRDs).

**All Findings Tab** — Complete findings list with severity filter (Critical / High / Medium / Low) and remediation guidance.

**About MCP-G Tab** — Architecture diagrams, prerequisites, installation links, and scoring system explanation.

---

## 10. MCPGovernancePolicy CRD Reference

### Strict Production Policy

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

### Policy with AI Scoring

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

## 11. REST API Reference

All endpoints are served by the controller on port 8090.

### GET /api/governance/overview

Returns the cluster-level score summary.

```bash
curl http://localhost:8090/api/governance/overview
```

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

### GET /api/governance/mcp-servers

Returns per-server security details.

```bash
curl http://localhost:8090/api/governance/mcp-servers
```

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
      "toolsExposed": 12,
      "agentConsumers": ["agent-1", "agent-2"],
      "lastEvaluated": "2026-02-21T10:30:00Z"
    }
  ]
}
```

### GET /api/governance/findings

Returns all governance findings.

```bash
curl http://localhost:8090/api/governance/findings
```

### GET /api/governance/inventory/verified

Returns Verified Catalog scoring results.

```bash
curl http://localhost:8090/api/governance/inventory/verified
```

---

## 12. Use Cases

**Enterprise Security Compliance** — Ensure all MCP servers meet compliance standards (SOC 2, ISO 27001) with automated scoring and audit-ready findings.

**Multi-Tenant Isolation** — Validate that each tenant's MCP servers are properly isolated with gateway routing and RBAC policies.

**AI Agent Risk Assessment** — Before deploying a new AI agent to production, evaluate which MCP servers it can access and what their current security posture is.

**Incident Response** — When a security incident occurs, quickly identify which agents accessed which tools via the findings tab and evaluation history.

**Continuous Governance** — As new MCP servers are deployed, automatic discovery and evaluation prevent configuration drift from your security baseline.

**Vendor Risk Management** — Evaluate third-party MCP server catalogs before integration — Verified Catalog scoring ensures only trustworthy sources are used.

---

## 13. Best Practices

**Start with Discovery, Not Enforcement**
Deploy MCP-G first to understand your current MCP infrastructure. Review the baseline findings before enforcing any policies.

**Gradual Hardening (5-Week Plan)**
- Week 1: Discovery and baseline assessment
- Week 2: Enable soft enforcement (monitoring only)
- Week 3: Enable gateway routing requirement
- Week 4: Enable authentication and authorization
- Week 5: Enable TLS and rate limiting

**Use AI for Risk Analysis**
Enable AI scoring to get reasoning and context beyond algorithmic scoring. It surfaces risks that rule-based systems miss.

**Monitor Score Trends**
A declining trend indicates new security gaps being introduced. A rising trend confirms your hardening work is paying off.

**Review Policies Quarterly**
Evolving infrastructure and threat landscape mean policies need regular review to stay relevant.

**Document Exceptions**
When a server can't meet a requirement temporarily, document the exception and timeline for remediation — don't just ignore the finding.

---

## 14. Roadmap

### Current (v0.18.0)
✅ 9 resource type discovery
✅ 8-category scoring system
✅ AI-powered risk analysis (Gemini + Ollama)
✅ Verified catalog scoring
✅ REST API with full JSON responses
✅ Real-time Next.js dashboard with 6 tabs
✅ MCPGovernancePolicy CRD

### Upcoming (v1.0.0)
🔄 Soft policy enforcement (warnings → hard block)
🔄 Automated remediation with one-click apply
🔄 Compliance reports (SOC 2, ISO 27001, PCI-DSS)
🔄 Audit logging and forensics
🔄 Multi-cluster governance federation
🔄 Custom scoring extensions (plugin system)
🔄 Webhook-based event notifications

---

## 15. Conclusion

As AI agents become production workloads, the MCP servers they rely on become critical attack surfaces. Without governance, your organization is one prompt injection attack away from a serious incident.

**MCP Governance (MCP-G)** gives you the visibility, scoring, and control you need to run MCP infrastructure securely at scale — without slowing down your AI agent development.

Whether you're running 3 MCP servers or managing a multi-tenant cluster, MCP-G helps you:

✅ Discover and inventory all MCP resources automatically
✅ Evaluate security posture across 8 governance categories
✅ Identify and prioritize vulnerabilities by severity
✅ Maintain continuous compliance as your infrastructure evolves
✅ Leverage AI to surface risks that rule-based scoring misses

---

### Get Started Today

**1. Clone the repository:**

```bash
git clone https://github.com/techwithhuz/mcp-g
```

**2. Deploy to your cluster:**

```bash
helm install mcp-governance ./charts/mcp-governance
```

**3. Open the dashboard and review your MCP security posture.**

---

🔗 **GitHub:** https://github.com/techwithhuz/mcp-g
📖 **Documentation:** https://github.com/techwithhuz/mcp-g/blob/main/README.md
🛡️ **AgentGateway:** https://agentgateway.dev
🤖 **Kagent:** https://kagent.dev
🌐 **Model Context Protocol:** https://modelcontextprotocol.io

---

*Have questions or want to contribute? Open an issue or PR on GitHub — contributions are very welcome!*

*— Tech With Huz | February 2026*

---

## WORDPRESS PUBLISHING CHECKLIST

Before hitting Publish, verify:

- [ ] Paste SEO Title into Yoast / RankMath SEO Title field
- [ ] Paste Meta Description into Yoast / RankMath Meta Description field
- [ ] Set Focus Keyphrase: **MCP Governance Kubernetes Security**
- [ ] Set URL slug to: **mcp-governance-kubernetes-ai-security**
- [ ] Add Featured Image (architecture diagram or dashboard screenshot)
- [ ] Set Categories: Kubernetes, AI Security, DevOps, Cloud Native
- [ ] Add Tags: Kubernetes, MCP, Model Context Protocol, AI Agents, Security, DevSecOps, Helm, Go, Next.js, Open Source
- [ ] Set Author: Tech With Huz
- [ ] Enable social preview (Open Graph image)
- [ ] Schedule or Publish

### INTERNAL LINKING SUGGESTIONS (add these manually)
- Link "Model Context Protocol" to your introductory MCP post (if you have one)
- Link "AgentGateway" to your AgentGateway tutorial post
- Link "Kagent" to your Kagent getting started post
- Link "Helm" to your Helm fundamentals post
