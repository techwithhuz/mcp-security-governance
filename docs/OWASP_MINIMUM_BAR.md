# OWASP MCP Security — Minimum Bar Checklist Alignment

> **Source:** *A Practical Guide for Secure MCP Server Development v1.0*, OWASP GenAI Security Project, February 2026 — Page 14  
> **Project:** [`mcp-security-governance`](https://github.com/techwithhuz/mcp-security-governance)  
> **Assessment Date:** March 2026

---

## Legend

| Symbol | Meaning |
|:---:|---|
| ✅ | **Implemented** — Runtime check exists, finding generated, scored |
| ⚠️ | **Partial** — Some coverage exists but key gaps remain |
| 🔧 | **Scaffolded** — Types/fields defined in code but no runtime logic |
| ❌ | **Not Covered** — No implementation exists |
| 🚫 | **Out of Scope** — Application-layer concern; not detectable from K8s runtime |

---

## Checklist Alignment Table

| # | OWASP Minimum Bar Requirement | Status | How We Check It | Gap / Notes |
|:---:|---|:---:|---|---|
| **— Pillar 1: Strong Identity, Auth & Policy Enforcement —** |||||
| 1.1 | All remote MCP servers use **OAuth 2.1 / OIDC** | ✅ | `checkAuthentication()` raises `AUTH-002` (Critical) when no JWT policy exists | Fully covered |
| 1.2 | Tokens are **short-lived** and **scoped** | ❌ | No token TTL or scope field inspected | `AgentgatewayPolicyResource` has no `tokenTTL`/`scopes` field |
| 1.3 | Tokens **validated on every call** (iss, aud, exp, sig) | ⚠️ | `AUTH-001` checks JWT mode (Strict/Optional/Permissive) | Mode only — individual claims (`exp`, `aud`) not inspected |
| 1.4 | **No token passthrough** to downstream APIs | ✅ | `checkExposure()` raises `EXP-001` when RemoteMCPServer bypasses agentgateway | Architecture enforced by agentgateway proxy layer |
| 1.5 | **Policy enforcement is centralized** | ✅ | `checkAgentGatewayCompliance()` raises `AGW-001`/`AGW-100` when servers bypass agentgateway | Core design principle — agentgateway is the single enforcement point |
| **— Pillar 2: Strict Isolation & Lifecycle Control —** |||||
| 2.1 | Users, sessions, and execution contexts are **fully isolated** | 🚫 | Not detectable from K8s runtime | Application-layer concern — requires static code analysis |
| 2.2 | **No shared state** for user data | 🚫 | Not detectable from K8s runtime | Application-layer concern |
| 2.3 | Sessions have **deterministic cleanup** | ❌ | No session lifecycle check exists | No session timeout/cleanup inspection |
| 2.4 | Sessions have **enforced resource quotas** | ⚠️ | `checkRateLimit()` raises `RL-001` when no rate limiting policy exists | Global rate limit only — not per-session or per-identity keyed |
| **— Pillar 3: Trusted, Controlled Tooling —** |||||
| 3.1 | Tools are **cryptographically signed** | ❌ | No signature/hash field in `MCPServerCatalog` CRD | VerifiedCatalog scorer has no `SHA256` or `Signature` field |
| 3.2 | Tools are **version-pinned** | ⚠️ | `checkVersioning()` in `inventory/scorer.go` checks for version label | Label-based only — not cryptographically pinned |
| 3.3 | Tools are **formally approved** (SAST, SCA, manual review) | 🚫 | Pipeline/process concern | Cannot inspect CI/CD approval gates from K8s runtime |
| 3.4 | Tool descriptions **validated against runtime behavior** | ❌ | No behavioral validation exists | Requires runtime tool call interception |
| 3.5 | Only **minimal, necessary tool fields** exposed to the model | ⚠️ | `checkToolCount()` raises `TOOLS-001` when tool count exceeds thresholds | Quantity checked — field-level exposure not inspected |
| **— Pillar 4: Schema-Driven Validation Everywhere —** |||||
| 4.1 | All MCP messages, tool inputs/outputs are **schema-validated** | 🚫 | Not detectable from K8s runtime | Application-layer JSON schema enforcement |
| 4.2 | Inputs/outputs are **sanitized** and **treated as untrusted** | 🚫 | Not detectable from K8s runtime | Application-layer sanitization logic |
| 4.3 | Inputs/outputs are **size-limited** | ❌ | No size limit field in `AgentgatewayPolicyResource` | Policy CRD needs `maxRequestSize` / `maxResponseSize` |
| 4.4 | **Structured (JSON) tool invocation** is required | ⚠️ | `MCPTargetInfo.Protocol` inspected (StreamableHTTP / SSE) | Transport type checked — invocation format (JSON vs free-text) not validated |
| **— Pillar 5: Hardened Deployment & Continuous Oversight —** |||||
| 5.1 | Server runs **containerized** | 🔧 | `WorkloadResource.Kind` field planned (Deployment/StatefulSet) | `discoverWorkloads()` not implemented; no finding generated |
| 5.2 | Server runs as **non-root** | 🔧 | `WorkloadResource.RunAsNonRoot` field planned | `discoverWorkloads()` not implemented; no finding generated |
| 5.3 | Server is **network-restricted** (NetworkPolicy) | 🔧 | `NetworkPolicyResource` struct planned | `discoverNetworkPolicies()` not implemented; no finding generated |
| 5.4 | **Secrets stored in vaults** — not in env vars or code | 🔧 | `WorkloadResource.HasPlaintextEnvSecrets` field planned | `discoverWorkloads()` not implemented; no vault annotation check |
| 5.5 | Secrets **never exposed to the LLM** | ❌ | No middleware/annotation check exists | No Vault agent injection or External Secret operator check |
| 5.6 | **CI/CD security gates** are mandatory | 🚫 | Not detectable from K8s runtime | Pipeline config (OPA, Trivy, Gosec) lives outside the cluster |
| 5.7 | **Audit logs** are mandatory — every tool call, auth event, config change | ❌ | No `auditor.go` or audit event emitter exists | Required by OWASP Minimum Bar; completely absent |
| 5.8 | **Continuous monitoring** is mandatory (SIEM, real-time alerts) | ⚠️ | Controller runs a 30s Kubernetes reconcile loop | Scan loop is continuous; no SIEM export, webhook, or alerting pipeline |

---

## Summary by Pillar

| Pillar | Total Checks | ✅ Done | ⚠️ Partial | 🔧 Scaffolded | ❌ Missing | 🚫 Out of Scope |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| **1. Identity, Auth & Policy** | 5 | 3 | 1 | 0 | 1 | 1 |
| **2. Isolation & Lifecycle** | 4 | 0 | 1 | 0 | 1 | 2 |
| **3. Trusted Tooling** | 5 | 0 | 2 | 0 | 1 | 2 |
| **4. Schema Validation** | 4 | 0 | 1 | 0 | 1 | 2 |
| **5. Hardened Deployment** | 8 | 0 | 1 | 4 | 2 | 1 |
| **Total** | **26** | **3 (12%)** | **6 (23%)** | **4 (15%)** | **6 (23%)** | **7 (27%)** |

---

## What Must Be Built to Pass the Minimum Bar

Listed in priority order:

| Priority | Item | Pillar | Effort |
|:---:|---|:---:|:---:|
| 🔴 1 | Implement `discoverWorkloads()` in `discovery.go` — read `securityContext` from Deployments/StatefulSets | 5 | Medium |
| 🔴 2 | Implement `checkHardenedDeployment()` in `evaluator.go` — non-root, read-only FS, no priv escalation, cap drop | 5 | Medium |
| 🔴 3 | Implement `discoverNetworkPolicies()` in `discovery.go` — check NetworkPolicy exists in MCP namespaces | 5 | Small |
| 🔴 4 | Build `auditor.go` — emit audit records for tool invocations, auth events, and config changes | 5 | Large |
| 🟠 5 | Add plaintext secret env var check — flag `value:` env vars with names matching `TOKEN/KEY/SECRET/PASSWORD` | 5 | Small |
| 🟠 6 | Add vault annotation check — inspect `vault.hashicorp.com/agent-inject` on pods | 5 | Small |
| 🟠 7 | Add `tokenTTL` / `scopes` fields to `AgentgatewayPolicyResource` and check in `checkAuthentication()` | 1 | Medium |
| 🟡 8 | Add SIEM/webhook export — emit findings to Prometheus, OpenTelemetry, or HTTP webhook sink | 5 | Large |
| 🟡 9 | Add `SHA256` / `Signature` field to `MCPServerCatalog` CRD and verify in `inventory/scorer.go` | 3 | Large |
| 🟡 10 | Add `maxRequestSize` / `maxResponseSize` to policy and check in `checkRateLimit()` or new `checkInputLimits()` | 4 | Medium |

---

## Minimum Bar Pass/Fail Status

```
Pillar 1 — Identity, Auth & Policy:   PARTIAL PASS  (3/5 fully met)
Pillar 2 — Isolation & Lifecycle:     FAIL          (0/4 fully met; 2 are out of scope)
Pillar 3 — Trusted Tooling:           FAIL          (0/5 fully met; 2 are out of scope)
Pillar 4 — Schema Validation:         FAIL          (0/4 fully met; 2 are out of scope)
Pillar 5 — Hardened Deployment:       FAIL          (0/8 fully met; 4 scaffolded only)

Overall Minimum Bar:  ❌ FAIL  —  3 of 26 checks fully implemented (12%)
```

> **Note on "Out of Scope" items:** The 7 application-layer checks (🚫) cannot be enforced by a K8s governance controller — they must be addressed in the MCP server application code, CI/CD pipeline, or developer process. Removing these from scope, the adjusted pass rate is **3 of 19 in-scope checks (16%)**.

---

*Generated from codebase analysis against OWASP MCP Security Guide v1.0 (Feb 2026), Page 14 — "MCP Security Minimum Bar (Review Checklist)".*
