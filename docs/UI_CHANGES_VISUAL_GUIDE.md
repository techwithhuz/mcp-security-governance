# UI Changes - Visual Guide

## 1. Skills Catalog - Fixed Message Logic

### Scenario A: Security Scanning DISABLED ⚠️

**Location:** Skills Catalog Tab → Expand any catalog card (if no findings)

**Before:**
```
┌─────────────────────────────────────────────────────────┐
│ ✓ All checks passed — this skill catalog is             │
│   compliant with governance policies.                   │
└─────────────────────────────────────────────────────────┘
```

**After (NEW - More Informative):**
```
┌─────────────────────────────────────────────────────────┐
│ ✓ Metadata checks passed. ⚠️ Repository security        │
│   scanning is disabled.                                 │
│                                                         │
│ Enable scanRepoContent: true in the Governance policy  │
│ to scan for prompt injection, privilege escalation, and │
│ other security patterns in the repository content.      │
└─────────────────────────────────────────────────────────┘
```

### Scenario B: Security Scanning ENABLED ✅

**Location:** Skills Catalog Tab → Expand any catalog card (if no findings)

**Message:**
```
┌─────────────────────────────────────────────────────────┐
│ ✓ All security checks passed — this skill catalog is   │
│   compliant with governance policies.                   │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Dashboard Navigation - New Tab

### Before:
```
┌──────────────────────────────────────────────────────────────────┐
│ Overview | MCP Servers | Verified Catalog | Skills Catalog |    │
│ Resources | All Findings | About MCP-G                           │
└──────────────────────────────────────────────────────────────────┘
```

### After (NEW):
```
┌──────────────────────────────────────────────────────────────────┐
│ Overview | MCP Servers | Verified Catalog | Skills Catalog |    │
│ 🔍 Repo Scanner | Resources | All Findings | About MCP-G        │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3. Repository Scanner - Complete Interface

### Tab 1: Scan Repository

```
╔═══════════════════════════════════════════════════════════════╗
║  🔍 Repository Scanner                                        ║
║  ───────────────────────────────────────────────────────────  ║
║  Scan GitHub, GitLab, or Bitbucket repositories for          ║
║  security patterns, prompt injection risks, and compliance   ║
║  issues.                                                      ║
║                                                               ║
║  📋 Scan Repository | 🔑 Credentials (3)                      ║
║                                                               ║
║  ┌───────────────────────────────────────────────────────┐  ║
║  │ Repository URL                                        │  ║
║  │ https://github.com/owner/repo                         │  ║
║  └───────────────────────────────────────────────────────┘  ║
║                                                               ║
║  ┌─────────────────────────────────────────────────────┐    ║
║  │ ☑ Private Repository                                │    ║
║  │ Check if the repository requires authentication    │    ║
║  └─────────────────────────────────────────────────────┘    ║
║                                                               ║
║  ┌───────────────────────────────────────────────────────┐  ║
║  │ Select Credential                          *         │  ║
║  │ [-- Select a credential --]                         │  ║
║  │ My GitHub Token (GITHUB)                           │  ║
║  │ Work GitLab Account (GITLAB)                        │  ║
║  └───────────────────────────────────────────────────────┘  ║
║                                                               ║
║  [🔍 Scan Repository]                                        ║
║                                                               ║
║  ┌────────────────────────────────────────────────────┐     ║
║  │ ✓ Scan Complete                         ▼           │    ║
║  ├────────────────────────────────────────────────────┤     ║
║  │ Files Scanned: 42                                 │     ║
║  │ Issues Found: 2                                   │     ║
║  │                                                    │     ║
║  │ Findings:                                          │     ║
║  │ • Potential prompt injection pattern [High]       │     ║
║  │ • Hardcoded credentials detected [Critical]       │     ║
║  └────────────────────────────────────────────────────┘     ║
╚═══════════════════════════════════════════════════════════════╝
```

### Tab 2: Credentials Management

```
╔═══════════════════════════════════════════════════════════════╗
║  🔍 Repository Scanner                                        ║
║  ───────────────────────────────────────────────────────────  ║
║  Scan GitHub, GitLab, or Bitbucket repositories...           ║
║                                                               ║
║  📋 Scan Repository | 🔑 Credentials (2)                      ║
║                                                               ║
║  ℹ️  Safe credential storage                                  ║
║  Credentials are stored locally in your browser's            ║
║  localStorage. They are only sent to the server during       ║
║  repository scans and never logged or stored on backend.     ║
║                                                               ║
║  [+ Add New Credential]                                       ║
║                                                               ║
║  ┌─────────────────────────────────────────────────────┐    ║
║  │ Add Credential                                       │    ║
║  │ ──────────────────────────────────────────────────  │    ║
║  │ Provider: [GitHub  ▼]                              │    ║
║  │ Credential Name: [My GitHub Token           ]      │    ║
║  │ Token: [●●●●●●●●●●●●●●] [eye-icon]              │    ║
║  │ Create a Personal Access Token with repo read      │    ║
║  │ permissions.                                        │    ║
║  │                                                      │    ║
║  │ [Save Credential]  [Cancel]                         │    ║
║  └─────────────────────────────────────────────────────┘    ║
║                                                               ║
║  Saved Credentials                                            ║
║  ────────────────────                                         ║
║                                                               ║
║  ┌─────────────────────────────────────────────────────┐    ║
║  │ My GitHub Token                                     │ [✕] │
║  │ GITHUB • ghp_...Ew3y2 [📋]                          │    ║
║  └─────────────────────────────────────────────────────┘    ║
║                                                               ║
║  ┌─────────────────────────────────────────────────────┐    ║
║  │ Work GitLab Account                                 │ [✕] │
║  │ GITLAB • glpat_...5kB9 [📋]                         │    ║
║  └─────────────────────────────────────────────────────┘    ║
╚═══════════════════════════════════════════════════════════════╝
```

---

## 4. Scan Results Display

### Success Case:
```
┌──────────────────────────────────────────────────────────┐
│ ✓ Scan Complete                            ▲             │
├──────────────────────────────────────────────────────────┤
│ Files Scanned: 42                                         │
│ Issues Found: 2                                           │
│                                                            │
│ Findings:                                                  │
│ • Potential prompt injection pattern detected [High]      │
│   Found patterns that could be used for prompt           │
│   injection attacks                                       │
│                                                            │
│ • Hardcoded credentials detected [Critical]               │
│   Repository contains hardcoded API keys or tokens       │
│                                                            │
│ Security Checks:                                          │
│ ✓ SKL-SEC-001: Prompt Injection Detection (PASSED)       │
│ ✗ SKL-SEC-002: Privilege Escalation Check (FAILED)       │
│ ✓ SKL-SEC-003: Data Exfiltration Check (PASSED)          │
│ ✗ SKL-SEC-004: Credential Harvesting Check (FAILED)      │
│ ✓ SKL-SEC-005: Scope Creep Validation (PASSED)           │
│ ✓ SKL-SEC-006: Safety Guardrails (PASSED)                │
└──────────────────────────────────────────────────────────┘
```

### Error Case:
```
┌──────────────────────────────────────────────────────────┐
│ ✗ Scan Failed                               ▲             │
├──────────────────────────────────────────────────────────┤
│ Error: Failed to authenticate with private repository   │
│        Verify your credentials and try again.            │
└──────────────────────────────────────────────────────────┘
```

---

## 5. Credential Add Form - Expanded View

```
┌─────────────────────────────────────────────────────────┐
│ Add Credential                                           │
│ ───────────────────────────────────────────────────────  │
│                                                          │
│ Provider *                                               │
│ ┌──────────────┐                                         │
│ │ GitHub    ▼  │                                         │
│ │ GitLab       │                                         │
│ │ Bitbucket    │                                         │
│ └──────────────┘                                         │
│                                                          │
│ Credential Name *                                        │
│ ┌────────────────────────────────────────────────────┐  │
│ │ e.g., My GitHub Token                              │  │
│ └────────────────────────────────────────────────────┘  │
│                                                          │
│ Token / Personal Access Token *                          │
│ ┌────────────────────────────────────────────┐ [eye]    │
│ │ ghp_... or glpat-... or Bitbucket token    │          │
│ └────────────────────────────────────────────┘          │
│ Create a Personal Access Token with repo read           │
│ permissions.                                             │
│                                                          │
│ [Save Credential]        [Cancel]                       │
└─────────────────────────────────────────────────────────┘
```

---

## 6. Before/After: Skills Catalog with Multiple Scenarios

### Scenario 1: No Findings + Scanning Disabled ⚠️
```
┌────────────────────────────────────────────────────────┐
│ my-skill v1.0.0 [DATA] [namespace] · github.com/...   │
├────────────────────────────────────────────────────────┤
│ Score: 85/100                                Pass       │
│                                                         │
│ ✓ Metadata checks passed. ⚠️ Repository security       │
│   scanning is disabled.                                │
│                                                         │
│ Enable scanRepoContent: true in the Governance policy │
│ to scan for prompt injection, privilege escalation,   │
│ and other security patterns in the repository content.│
└────────────────────────────────────────────────────────┘
```

### Scenario 2: No Findings + Scanning Enabled ✅
```
┌────────────────────────────────────────────────────────┐
│ my-skill v1.0.0 [DATA] [namespace] · github.com/...   │
├────────────────────────────────────────────────────────┤
│ Score: 95/100                                Pass       │
│                                                         │
│ ✓ All security checks passed — this skill catalog is  │
│   compliant with governance policies.                  │
└────────────────────────────────────────────────────────┘
```

### Scenario 3: With Findings
```
┌────────────────────────────────────────────────────────┐
│ risky-skill v1.0.0 [ADMIN] [namespace] · ...          │
├────────────────────────────────────────────────────────┤
│ Score: 45/100                         Warning  1C 2H 1M │
│                                                         │
│ Issues Found                                            │
│ ⚠️  [SKL-SEC-001] Potential prompt injection detected   │
│    Pattern: ignore previous instructions               │
│                                                         │
│ ⚠️  [SKL-002] Repository uses HTTP instead of HTTPS    │
│    Remediation: Change spec.repository.url to HTTPS    │
│                                                         │
│ • [SKL-003] No version specified...                    │
└────────────────────────────────────────────────────────┘
```

---

## Color Scheme

| Component | Color |
|-----------|-------|
| Success/Pass | Green #22c55e |
| Warning/Caution | Yellow #eab308 |
| Critical/Error | Red #ef4444 |
| Info/Help Text | Blue #3b82f6 |
| Prompt Injection | Purple/Blue |
| Privilege Escalation | Orange |
| Credentials/Security | Cyan #06b6d4 |

---

## Accessibility Features

✅ Proper semantic HTML with `<button>`, `<input>`, `<select>`
✅ ARIA labels for icons
✅ Color contrast meets WCAG standards
✅ Keyboard navigation support
✅ Tab order logical and intuitive
✅ Error messages clearly associated with fields
✅ Loading states with spinner animation

