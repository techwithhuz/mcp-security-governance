# OWASP MCP Security Compliance Alignment

> **Source:** [A Practical Guide for Secure MCP Server Development v1.0](https://genai.owasp.org) — OWASP GenAI Security Project, February 2026  
> **Project:** [`mcp-security-governance`](https://github.com/techwithhuz/mcp-security-governance) — Kubernetes-native MCP governance controller  
> **Assessment Date:** March 2026

---

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | **Implemented** — Runtime check exists, findings generated, scored |
| ⚠️ | **Partial** — Some coverage, key gaps remain |
| 🔧 | **Scaffolded** — Data structures/types defined but logic not implemented |
| ❌ | **Not Covered** — No implementation; gap exists |
| 🚫 | **Out of Scope** — Application-layer concern, not detectable from K8s runtime |

---

## Executive Score Card

| OWASP Pillar | Score | Status |
|---|:---:|---|
| **1. Strong Identity, Auth & Policy Enforcement** | **80%** | ✅ Strong |
| **2. Strict Isolation & Lifecycle Control** | **15%** | ❌ Mostly app-layer |
| **3. Trusted, Controlled Tooling** | **35%** | ⚠️ Partial |
| **4. Schema-Driven Validation Everywhere** | **20%** | ❌ Mostly app-layer |
| **5. Hardened Deployment & Continuous Oversight** | **10%** | 🔧 Scaffolded only |
| | | |
| **🔴 Overall Alignment** | **~32%** | Significant gaps remain |

---

## Pillar 1 — Strong Identity, Auth & Policy Enforcement

*OWASP source: Pages 10, 14 — Chapter 5 "Authentication & Authorization" + Minimum Bar §1*

| # | OWASP Requirement | Implementation | Status | Code Reference |
|---|---|---|---|---|
| 1.1 | All remote MCP servers enforce **OAuth 2.1 / OIDC** | `checkAuthentication()` flags missing or weak-mode JWT auth | ✅ | `AUTH-001`, `AUTH-002` findings |
| 1.2 | Tokens validated on **every request** (iss, aud, exp, signature) | JWT presence and mode (Strict/Optional/Permissive) checked | ⚠️ | `AUTH-001` — mode only; no `exp`/`aud` field inspection |
| 1.3 | Tokens are **short-lived and scoped** (minutes, narrow scopes) | Not checked — no token TTL or scope inspection | ❌ | No `tokenTTL` or `scope` field in `AgentgatewayPolicyResource` |
| 1.4 | **No token passthrough** to downstream APIs | Architecture enforced — agentgateway is the proxy; `EXP-001` flags direct access | ✅ | `checkExposure()` → `EXP-001` |
| 1.5 | Use **token delegation** (RFC 8693 / On-Behalf-Of) | Not checked | ❌ | Not discoverable from K8s resources |
| 1.6 | **Centralized policy enforcement** (auth, authz, audit, tool filtering) | agentgateway Gateway + `checkAgentGatewayCompliance()` enforces centralized routing | ✅ | `AGW-001` → `AGW-200` findings |
| 1.7 | Sessions treated as **state, not identity** — re-check authz before sensitive actions | Not checked — application-layer concern | 🚫 | Cannot inspect from K8s runtime |

**Pillar 1 Score: 80%** — Auth infrastructure is well-covered; token lifetime/scoping and delegation flows are gaps.

---

## Pillar 2 — Strict Isolation & Lifecycle Control

*OWASP source: Pages 6, 14 — Chapter 1 "Secure MCP Architecture" + Minimum Bar §2*

| # | OWASP Requirement | Implementation | Status | Code Reference |
|---|---|---|---|---|
| 2.1 | **Users and sessions fully isolated** — separate execution contexts, memory, temp storage | Not checked — application-layer concern | 🚫 | No session isolation discoverable from K8s |
| 2.2 | **No shared state** for user data (no global variables, class-level singletons) | Not checked — application-layer concern | 🚫 | Requires code-level static analysis |
| 2.3 | Separate objects **per session** or session-keyed state store (e.g. Redis namespaced) | Not checked | 🚫 | Not inspectable from K8s |
| 2.4 | **Deterministic session cleanup** — flush file handles, temp storage, tokens on disconnect | Not checked | ❌ | No session lifecycle check exists |
| 2.5 | **Per-session resource quotas** (memory, CPU, filesystem, API call limits) | Rate limiting checked globally via `checkRateLimit()` | ⚠️ | `RL-001`, `RL-002` — global rate limit, not per-session |
| 2.6 | **Compute isolation** — containers, micro-VMs, no shared host process memory | Pod security context inspection planned but not implemented | 🔧 | `WorkloadResource` struct planned; no `discoverWorkloads()` exists |

**Pillar 2 Score: 15%** — Almost entirely application-layer. The only partial credit is rate limiting (global, not per-session).

---

## Pillar 3 — Trusted, Controlled Tooling

*OWASP source: Pages 7, 14 — Chapter 2 "Safe Tool Design" + Minimum Bar §3*

| # | OWASP Requirement | Implementation | Status | Code Reference |
|---|---|---|---|---|
| 3.1 | Tools have **cryptographically signed manifests** (description, schema, version, permissions) | VerifiedCatalog scorer checks publisher source; no SHA256/signature field | ⚠️ | `inventory/scorer.go` → `checkPublisherSource()` |
| 3.2 | Signature and **hash verified at load time** | Not implemented — no cryptographic verification | ❌ | No `SHA256`, `Signature` field in `MCPServerCatalog` CRD |
| 3.3 | Formal **approval workflow** for adding/updating tools (SAST, SCA, manual review) | Not checked — pipeline/process concern | 🚫 | Cannot inspect from K8s runtime |
| 3.4 | Tool descriptions **validated against runtime behavior** | Not implemented | ❌ | No behavioral validation exists |
| 3.5 | **Tool pinning** — version-pinned, flag tools that changed post-approval | VerifiedCatalog scorer checks versioning label | ⚠️ | `checkVersioning()` — label-based, not cryptographic |
| 3.6 | Only **minimal tool fields** exposed to model; sensitive fields hidden | Tool count threshold checked; field-level inspection not done | ⚠️ | `checkToolCount()` → `TOOLS-001` — quantity only |
| 3.7 | Flag tools performing **actions not in their description** | Not implemented | ❌ | Requires runtime behavioral analysis |

**Pillar 3 Score: 35%** — VerifiedCatalog scorer provides partial coverage on versioning and publisher trust; cryptographic integrity is missing.

---

## Pillar 4 — Schema-Driven Validation Everywhere

*OWASP source: Pages 8–9, 14 — Chapters 3 & 4 + Minimum Bar §4*

| # | OWASP Requirement | Implementation | Status | Code Reference |
|---|---|---|---|---|
| 4.1 | All MCP messages, tool inputs/outputs are **JSON schema-validated** | Not checked — application-layer concern | 🚫 | Cannot inspect schema enforcement from K8s |
| 4.2 | Reject any request that **doesn't match expected schema** | Not checked | 🚫 | Application-layer |
| 4.3 | Inputs/outputs **sanitized and encoded** (strip XSS, SQLi, RCE sequences) | Not checked | 🚫 | Application-layer |
| 4.4 | **Size limits** enforced on all tool outputs and model inputs | Not checked | ❌ | No size limit field in `AgentgatewayPolicyResource` |
| 4.5 | **Structured JSON tool invocation** (not free-form text commands) | Transport protocol checked (StreamableHTTP/SSE) but not invocation format | ⚠️ | `MCPTargetInfo.Protocol` — transport only |
| 4.6 | **Prompt injection controls** — content filtering, LLM-as-a-Judge, HITL for high-risk actions | `checkPromptGuard()` checks for guard presence/regex rules | ⚠️ | `PG-001`, `PG-002` — guard presence, not HITL or LLM-Judge |
| 4.7 | **One task, one session** context compartmentalization | Not checked — application-layer | 🚫 | Cannot inspect from K8s |

**Pillar 4 Score: 20%** — Prompt guard check provides partial coverage; true schema and input validation is application-layer and outside K8s governance scope.

---

## Pillar 5 — Hardened Deployment & Continuous Oversight

*OWASP source: Pages 11–13, 14 — Chapters 6, 7, 8 + Minimum Bar §5*

| # | OWASP Requirement | Implementation | Status | Code Reference |
|---|---|---|---|---|
| 5.1 | MCP server runs in **minimal, hardened container** | `WorkloadResource` struct planned with all security context fields | 🔧 | Types not yet implemented; no discovery or check logic |
| 5.2 | Container runs as **non-root user** | `RunAsNonRoot` field planned in `WorkloadResource` | 🔧 | Field not implemented; no finding generated |
| 5.3 | **Unnecessary Linux capabilities dropped** (`capabilities.drop: [ALL]`) | `AllContainersCapDropAll` field planned | 🔧 | Field not implemented; no discovery |
| 5.4 | **Network segmentation** via K8s `NetworkPolicy` | `NetworkPolicyResource` struct planned | 🔧 | Type not implemented; no discovery |
| 5.5 | **Secrets stored in vaults** — not in env vars, logs, or code | `HasPlaintextEnvSecrets` field planned in `WorkloadResource` | 🔧 | Field not implemented; no discovery or check logic |
| 5.6 | **LLM never has access to secrets** — transparent middleware only | Not checked | ❌ | No annotation or vault check exists |
| 5.7 | **Supply chain controls** — version-pinned deps, signed images, AIBOM | `HasLatestTag` field planned in `WorkloadResource` | 🔧 | Field not implemented; no discovery logic |
| 5.8 | **Signed container images** (cosign / Sigstore) | Not implemented | ❌ | No cosign annotation check |
| 5.9 | **CI/CD security gates** (OPA policy-as-code, fail on vulnerability) | Not checkable from K8s runtime | 🚫 | Pipeline config lives outside the cluster |
| 5.10 | **Audit logs and trails** — log every tool invocation, auth event, config change | Not implemented | ❌ | No `auditor.go` or audit event emitter exists |
| 5.11 | **Immutable, secure audit log storage** for forensics | Not implemented | ❌ | No audit sink or storage configured |
| 5.12 | **Continuous monitoring** — SIEM integration, real-time alerts | Controller runs 30s reconcile loop implicitly | ⚠️ | Scan loop runs continuously; no SIEM export or alerting |
| 5.13 | **Seccomp / AppArmor** runtime protection profiles | `SeccompProfileSet` field planned in `WorkloadResource` | 🔧 | Field not implemented; no discovery |
| 5.14 | **Safe error handling** — no stack traces, tokens, paths in responses | Not checked — application-layer | 🚫 | Cannot inspect error response content from K8s |
| 5.15 | **Cryptographic integrity** for tools, deps, and registry manifests | VerifiedCatalog scorer partially covers version pinning | ⚠️ | `checkVersioning()` — label-based only, no crypto |
| 5.16 | **Non-Human Identity (NHI) governance** — unique credentials per agent, tightly scoped | Not implemented | ❌ | No NHI audit or service account inspection |

**Pillar 5 Score: 10%** — Data structures planned but no runtime discovery or check logic exists. Audit logging, SIEM, NHI governance, and CI/CD gate inspection are absent.

---

## Full OWASP Chapter Reference

### Chapter 1 — Secure MCP Architecture (Page 6)

| Requirement | Status | Notes |
|---|:---:|---|
| Prefer STDIO/Unix sockets for local servers; bind HTTP to 127.0.0.1 only | 🚫 | Application deployment decision |
| Enforce TLS 1.2+ for all remote connections | ✅ | `checkTLS()` → `TLS-001`, `TLS-002` |
| Validate JSON-RPC messages against MCP schema | 🚫 | Application-layer |
| Use allowlists / mTLS for static client relationships | ❌ | No mTLS check exists |
| Use OAuth 2.1 / OIDC for dynamic client identity | ✅ | `checkAuthentication()` → `AUTH-001`, `AUTH-002` |
| Validate Origin header for local HTTP | 🚫 | Application-layer |
| Isolate users/sessions — no shared state | 🚫 | Application-layer |
| Deterministic session cleanup (flush tokens, temp files on disconnect) | ❌ | No session lifecycle check |
| Per-session resource quotas (CPU, memory, API calls) | ⚠️ | Global rate limit only (`RL-001`) |

### Chapter 2 — Safe Tool Design (Page 7)

| Requirement | Status | Notes |
|---|:---:|---|
| Cryptographic tool manifests (signed, hashed at load time) | ❌ | VerifiedCatalog lacks signature field |
| Formal approval workflow (SAST, SCA, manual review) | 🚫 | Pipeline/process concern |
| Validate tool description vs. runtime behavior | ❌ | No behavioral analysis |
| Tool pinning — flag post-approval changes | ⚠️ | `checkVersioning()` label-based only |
| Expose only minimal tool fields to the model | ⚠️ | `checkToolCount()` counts tools; no field inspection |

### Chapter 3 — Data Validation & Resource Management (Page 8)

| Requirement | Status | Notes |
|---|:---:|---|
| Rate limits and quotas on tool invocations per session | ⚠️ | `checkRateLimit()` → `RL-001`; global, not per-session |
| Timeouts and isolated memory/compute budgets | 🚫 | Application-layer |
| JSON Schema validation for every tool input/output | 🚫 | Application-layer |
| Sanitize and encode all inputs (strip XSS, SQLi, RCE) | 🚫 | Application-layer |
| Size limits on all tool outputs | ❌ | Not configurable in policy |

### Chapter 4 — Prompt Injection Controls (Page 9)

| Requirement | Status | Notes |
|---|:---:|---|
| Structured JSON tool invocation (not free-form text) | ⚠️ | Transport type checked; invocation format not |
| Human-in-the-Loop (HITL) for high-risk actions | ❌ | No HITL check or elicitation detection |
| LLM-as-a-Judge approval for high-risk calls | ❌ | Not implemented |
| One task, one session — context compartmentalization | 🚫 | Application-layer |

### Chapter 5 — Authentication & Authorization (Page 10)

| Requirement | Status | Notes |
|---|:---:|---|
| OAuth 2.1 / OIDC mandatory for all remote servers | ✅ | `AUTH-002` Critical if missing |
| Validate `iss`, `aud`, `exp`, signature every request | ⚠️ | Mode checked; individual claims not inspected |
| Token delegation (RFC 8693) | ❌ | Not implemented |
| No token passthrough — On-Behalf-Of flows enforced | ✅ | Architecture enforced via agentgateway proxy |
| Short-lived scoped tokens (minutes, narrow scopes) | ❌ | No TTL/scope inspection |
| Centralized policy enforcement layer | ✅ | agentgateway is the enforcement point; `AGW-001` → `AGW-200` |
| RBAC / tool-level access control (CEL expressions) | ✅ | `checkAuthorization()` → `RBAC-001`, `RBAC-100` |
| CORS + CSRF protection | ✅ | `checkCORS()` → `CORS-001`, `CORS-002`, `CORS-003` |

### Chapter 6 — Secure Deployment & Updates (Page 11)

| Requirement | Status | Notes |
|---|:---:|---|
| Secrets in vaults — not in env vars, logs, or code | 🔧 | `HasPlaintextEnvSecrets` planned; no discovery |
| LLM never accesses secrets (transparent middleware) | ❌ | No check exists |
| Containerize: run as non-root, drop capabilities | 🔧 | `WorkloadResource` fields planned; no discovery |
| Network segmentation via `NetworkPolicy` | 🔧 | `NetworkPolicyResource` planned; no discovery |
| Version-pin dependencies, signed images, AIBOM | 🔧 | `HasLatestTag` planned; no discovery |
| CI/CD security gates (OPA, policy-as-code) | 🚫 | Outside K8s runtime scope |
| Safe error handling (no stack traces in responses) | 🚫 | Application-layer |

### Chapter 7 — Governance (Page 12)

| Requirement | Status | Notes |
|---|:---:|---|
| Cryptographic signing and version pinning for all tools/deps/registries | ⚠️ | `checkVersioning()` label-based; no crypto signing |
| Peer review policy for new tools/major changes | 🚫 | Process concern |
| Audit logs: every tool invocation, auth event, config change | ❌ | No `auditor.go` exists |
| Redact/hash sensitive data before logging | ❌ | No log pipeline |
| Secure, immutable audit log storage for forensics | ❌ | Not implemented |
| NHI governance — unique creds, scoped permissions, continuous audit | ❌ | No service account or NHI inspection |

### Chapter 8 — Tools & Continuous Validation (Page 13)

| Requirement | Status | Notes |
|---|:---:|---|
| SAST with custom MCP rules in CI/CD pipeline | 🚫 | Pipeline concern |
| SCA (dependency scanning) — break builds on vulnerabilities | 🚫 | Pipeline concern |
| Runtime seccomp / AppArmor profiles | 🔧 | `SeccompProfileSet` planned; no discovery |
| Continuous monitoring — feed audit logs to SIEM | ❌ | No SIEM export |
| Real-time alerts for suspicious patterns (spikes, high-freq calls) | ❌ | No alerting pipeline |
| OpenSSF Scorecard for supply chain posture | 🚫 | Project-level concern |
| Monitor OSV for new CVEs in dependencies | 🚫 | Pipeline concern |

---

## Gap Priority Matrix

Sorted by: **OWASP severity × implementation effort**

| Priority | Gap | OWASP Pillar | Effort | Impact |
|:---:|---|---|:---:|---|
| 🔴 P1 | Implement `checkHardenedDeployment()` + `discoverWorkloads()` + `discoverNetworkPolicies()` | Pillar 5 | Medium | Enables non-root, NetworkPolicy, plaintext-secret, `:latest` image checks |
| 🔴 P1 | Audit logging — `auditor.go` for tool invocations, auth events, config changes | Pillar 5 / Ch 7 | High | Required by OWASP Minimum Bar §5 |
| 🔴 P1 | Wire `HardenedDeploymentScore` into scoring pipeline + `MCPServerScoreBreakdown` | Pillar 5 | Low | Completes scoring for hardening scaffold |
| 🟠 P2 | Token TTL / scope inspection — add `TokenTTL` / `Scopes` to `AgentgatewayPolicyResource` | Pillar 1 | Medium | Closes short-lived token gap |
| 🟠 P2 | Vault annotation check — inspect `vault.hashicorp.com/agent-inject` or External Secret annotations | Pillar 5 / Ch 6 | Low | Detects vault-backed vs. raw K8s Secret usage |
| 🟠 P2 | Image signature check — inspect cosign/Sigstore annotations on discovered pods | Pillar 5 / Ch 6 | Medium | Supply chain integrity |
| 🟡 P3 | Per-session rate limiting — verify `RateLimit` is keyed by identity/session-id | Pillar 2 | Medium | Closes per-session quota gap |
| 🟡 P3 | Cryptographic tool manifest — add `SHA256` / `Signature` field to `MCPServerCatalog` CRD | Pillar 3 | High | Closes tool signing gap |
| 🟡 P3 | SIEM / alerting export — emit findings to webhook, Prometheus, or SIEM sink | Pillar 5 / Ch 8 | High | Continuous monitoring requirement |
| 🔵 P4 | NHI governance — inspect `ServiceAccount` RBAC bindings for MCP workloads | Pillar 5 / Ch 7 | Medium | Non-human identity audit |
| 🔵 P4 | mTLS check for static client relationships | Ch 1 | Medium | Closes static trust gap |

---

## What This Project Does Well ✅

1. **Centralized enforcement architecture** — agentgateway as the MCP control plane exactly matches OWASP's "Centralize Policy Enforcement" requirement (Ch 5)
2. **JWT/OIDC authentication detection** — catches missing, Optional/Permissive, and absent JWT policies (`AUTH-001`, `AUTH-002`)
3. **RBAC / tool-level access control** — CEL-based `matchExpressions` checked per MCP target (`RBAC-001`, `RBAC-100`)
4. **TLS enforcement** — backend TLS checked per `AgentgatewayBackend` (`TLS-001`, `TLS-002`)
5. **CORS + CSRF protection** — both detected in policies and HTTPRoutes (`CORS-001`, `CORS-002`, `CORS-003`)
6. **Tool scope limits** — configurable warning/critical thresholds for tool count (`TOOLS-001`)
7. **Exposure detection** — `RemoteMCPServer` URLs validated to route through agentgateway (`EXP-001`)
8. **Per-server scoring** — MCP-server-centric views with individual `ScoreBreakdown` across 8 categories
9. **VerifiedCatalog scorer** — publisher source, transport security, versioning, deployment health checks
10. **Continuous scanning** — Kubernetes reconcile loop provides ongoing policy evaluation

---

## What Needs to Be Built ❌

1. **`checkHardenedDeployment()`** — function called in `Evaluate()` but does not exist (compilation error)
2. **`discoverWorkloads()`** in `discovery.go` — list Deployments/StatefulSets, extract pod `securityContext`
3. **`discoverNetworkPolicies()`** in `discovery.go` — list `networking.k8s.io/v1/NetworkPolicy` resources
4. **Audit logging pipeline** — `auditor.go` with tool invocation, auth event, and config change records
5. **Token TTL/scope validation** — extend `AgentgatewayPolicyResource` to capture token lifetime config
6. **Vault secret detection** — check `vault.hashicorp.com/agent-inject` or External Secret operator annotations
7. **Image signature verification** — check cosign/Sigstore annotations on discovered pods
8. **SIEM export** — webhook/Prometheus/OpenTelemetry sink for findings and audit events
9. **NHI service account audit** — inspect `ServiceAccount` RBAC bindings for MCP workloads
10. **Wire `HardenedDeploymentScore`** into `calculateScores()`, `calculateOverallScore()`, and `MCPServerScoreBreakdown`

---

*This document is generated from codebase analysis against OWASP MCP Security Guide v1.0 (Feb 2026).*  
*Re-run assessment after implementing items in the Gap Priority Matrix above.*
