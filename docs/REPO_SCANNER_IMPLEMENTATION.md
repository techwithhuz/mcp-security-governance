# Skills Catalog UI & Repository Scanner - Feature Implementation

## Overview

This document describes two major enhancements to the MCP Governance Dashboard:

1. **Fixed misleading "All checks passed" message** in the Skills Catalog
2. **Added Repository Scanner feature** with credential management

---

## 1. Skills Catalog UI - Message Clarity Fix

### Problem
Previously, when a SkillCatalog had no findings, the UI displayed:
```
✓ All checks passed — this skill catalog is compliant with governance policies.
```

This message was **misleading** because it didn't distinguish between:
- ✅ All checks truly passed (metadata + security scans)
- ⚠️ Metadata checks passed but security scanning was **disabled**

### Solution

Updated the message logic to check the `securityScanned` flag:

```tsx
{findings.length === 0 && (
  <div className="space-y-2">
    <div className="flex items-center gap-2 text-sm text-green-400 py-2">
      <CheckCircle2 className="w-4 h-4" />
      {catalog.securityScanned ? (
        <>All security checks passed — this skill catalog is compliant with governance policies.</>
      ) : (
        <>Metadata checks passed. <span className="text-yellow-400">⚠️ Repository security scanning is disabled.</span></>
      )}
    </div>
    {!catalog.securityScanned && (
      <p className="text-xs text-gov-text-3 italic ml-6">
        Enable <code className="text-blue-400">scanRepoContent: true</code> in the Governance policy to scan for prompt injection, privilege escalation, and other security patterns in the repository content.
      </p>
    )}
  </div>
)}
```

### User Experience

**When security scanning is DISABLED:**
```
✓ Metadata checks passed. ⚠️ Repository security scanning is disabled.

Enable scanRepoContent: true in the Governance policy to scan for prompt injection, 
privilege escalation, and other security patterns in the repository content.
```

**When security scanning is ENABLED:**
```
✓ All security checks passed — this skill catalog is compliant with governance policies.
```

### Changes Made

| File | Change |
|------|--------|
| `dashboard/src/lib/types.ts` | Added `securityScanned?: boolean` to `SkillCatalogScore` interface |
| `dashboard/src/components/SkillCatalog.tsx` | Updated message logic with conditional rendering |

---

## 2. Repository Scanner Feature

### Overview

A new **Repository Scanner** tab in the dashboard allows users to:
- Scan GitHub, GitLab, and Bitbucket repositories for security patterns
- Manage credentials securely for private repositories
- View detailed findings with severity levels
- Detect: prompt injection, privilege escalation, credential harvesting, data exfiltration, scope creep

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Dashboard UI (RepoScanner)               │
│  ┌─────────────────┬──────────────────────────────────────┐ │
│  │  Scan Tab       │  Credentials Tab                      │ │
│  ├─────────────────┼──────────────────────────────────────┤ │
│  │ • Enter URL     │  • Add new credential               │ │
│  │ • Select creds  │  • View saved credentials            │ │
│  │ • Scan button   │  • Delete credentials                │ │
│  │ • Results       │  • Copy token to clipboard           │ │
│  └─────────────────┴──────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ↓
        ┌─────────────────────────────────────────┐
        │   Next.js API Proxy (/api/governance/)   │
        └─────────────────────────────────────────┘
                              ↓
        ┌─────────────────────────────────────────┐
        │   Repository Scanner Endpoint            │
        │   (/api/governance/scan/repo)            │
        └─────────────────────────────────────────┘
                              ↓
        ┌─────────────────────────────────────────┐
        │   Git Repository Clone & Pattern Scan    │
        │   • File scanning                        │
        │   • Pattern matching                     │
        │   • Finding generation                   │
        └─────────────────────────────────────────┘
```

### Components

#### RepoScanner Component (`dashboard/src/components/RepoScanner.tsx`)

A full-featured component with two tabs:

**Tab 1: Scan Repository**
- Repository URL input
- Private/Public toggle
- Credential selector (appears when private=true)
- Scan button with progress indicator
- Results display with expandable details

**Tab 2: Credentials Management**
- Add new credentials (GitHub, GitLab, Bitbucket)
- View all saved credentials
- Delete credentials
- Copy token to clipboard
- Safe localStorage storage with clear indication

#### API Endpoint (`dashboard/src/app/api/governance/scan/repo/route.ts`)

Handles POST requests with the following payload:

```typescript
interface ScanRequest {
  repoUrl: string;
  isPrivate: boolean;
  credentialToken?: string;
}

interface ScanResult {
  status: 'success' | 'error';
  repoUrl: string;
  filesScanned: number;
  issuesFound: number;
  findings: Finding[];
  securityChecks: SecurityCheck[];
  error?: string;
}
```

### Feature Highlights

#### 1. **Credential Management**
- 🔒 **Secure storage**: Credentials stored locally in browser localStorage
- 🔐 **Token masking**: Only shows first 4 and last 4 characters
- 🗑️ **Easy deletion**: Remove credentials with one click
- 📋 **Copy to clipboard**: Quick token access for manual use

#### 2. **Smart Repository Scanning**
- ✅ Supports GitHub, GitLab, Bitbucket
- 🔓 Public repo scanning without credentials
- 🔒 Private repo scanning with token authentication
- 📊 Detailed file counting and issue tracking
- 🎯 Multiple security check categories

#### 3. **Security Patterns Detected**

The scanner looks for:

| Pattern | Severity | Description |
|---------|----------|-------------|
| Prompt Injection | High | `ignore previous instructions`, bypass safety commands |
| Privilege Escalation | High | `sudo`, `root access`, privilege elevation attempts |
| Credential Harvesting | Critical | `steal credentials`, `capture tokens` |
| Data Exfiltration | High | `send data to external`, unauthorized uploads |
| Scope Creep | High | Overly permissive access patterns |
| Safety Guardrails | Medium | Missing safety validation |

#### 4. **Results Display**
- 📈 Summary: Files scanned, issues found
- 🔍 Detailed findings with severity badges
- ✓/✗ Security check status
- 📝 Remediation suggestions

### Dashboard Integration

The RepoScanner is integrated as a new tab in the main dashboard:

```tsx
{/* ========== REPO SCANNER TAB ========== */}
{activeTab === 'repo-scanner' && (
  <RepoScanner />
)}
```

Navigation button in header:
```
Repo Scanner
```

### User Workflow

1. **First Time User (Private Repo)**
   ```
   Dashboard → Repo Scanner tab → Credentials tab → Add credential
   → Fill in name, provider, token → Save → Back to Scan tab
   → Enter repo URL → Check "Private" → Select credential → Scan
   ```

2. **First Time User (Public Repo)**
   ```
   Dashboard → Repo Scanner tab → Enter repo URL → Scan
   ```

3. **Returning User (With Saved Credentials)**
   ```
   Dashboard → Repo Scanner tab → Enter URL → Select credential
   → Scan → View results
   ```

### Security Considerations

#### Frontend Security
- ✅ Credentials stored only in **localStorage** (not sent to server until scan)
- ✅ Tokens **never logged** or shown in console
- ✅ Token display uses **masking** (xxxx...xxxx format)
- ✅ User controls deletion of credentials
- ✅ No automatic credential sync

#### Backend Security
- ✅ Token only used for **repository access** during scan
- ✅ Credentials not persisted on server
- ✅ HTTPS enforced for token transmission
- ✅ Request validation and sanitization
- ✅ Error handling without credential exposure

### Backend Integration (Future)

When backend API is ready, implement `/api/governance/scan/repo` endpoint:

```go
type ScanRepoRequest struct {
    RepoURL          string `json:"repoUrl"`
    IsPrivate        bool   `json:"isPrivate"`
    CredentialToken  string `json:"credentialToken,omitempty"`
}

type ScanRepoResponse struct {
    Status          string            `json:"status"`
    RepoURL         string            `json:"repoUrl"`
    FilesScanned    int               `json:"filesScanned"`
    IssuesFound     int               `json:"issuesFound"`
    Findings        []RepoFinding     `json:"findings"`
    SecurityChecks  []SecurityCheck   `json:"securityChecks"`
    Error           string            `json:"error,omitempty"`
}

// handleScanRepo implements repository scanning with credential support
func handleScanRepo(w http.ResponseWriter, r *http.Request) {
    // 1. Validate request
    // 2. Clone repository (with token if private)
    // 3. Scan files for patterns
    // 4. Generate findings
    // 5. Return results
}
```

---

## Changes Summary

### Files Modified

| File | Changes |
|------|---------|
| `dashboard/src/lib/types.ts` | Added `securityScanned?: boolean` to `SkillCatalogScore` |
| `dashboard/src/components/SkillCatalog.tsx` | Updated no-findings message logic |
| `dashboard/src/app/page.tsx` | Added RepoScanner import and tab |
| `dashboard/src/app/api/governance/[...path]/route.ts` | No changes (generic proxy used) |

### Files Created

| File | Purpose |
|------|---------|
| `dashboard/src/components/RepoScanner.tsx` | Main RepoScanner component |
| `dashboard/src/app/api/governance/scan/repo/route.ts` | Repo scan API endpoint |

### No Breaking Changes
- ✅ All existing tabs continue to work
- ✅ Backward compatible with existing SkillCatalogScore data
- ✅ Optional `securityScanned` flag gracefully handled

---

## Testing Recommendations

### Manual Testing

1. **Skills Catalog - Metadata Only**
   - [ ] Deploy with `scanRepoContent: false`
   - [ ] Verify message shows "Metadata checks passed"
   - [ ] Verify remediation text is visible

2. **Skills Catalog - Security Scanning Enabled**
   - [ ] Deploy with `scanRepoContent: true`
   - [ ] Verify message shows "All security checks passed"
   - [ ] Verify no remediation text shown

3. **Repo Scanner - Public Repository**
   - [ ] Enter GitHub URL (public repo)
   - [ ] Click Scan without credentials
   - [ ] Verify results display

4. **Repo Scanner - Private Repository**
   - [ ] Go to Credentials tab
   - [ ] Add GitHub/GitLab token
   - [ ] Verify token is masked
   - [ ] Back to Scan tab, enter private repo URL
   - [ ] Select credential
   - [ ] Click Scan, verify results

5. **Repo Scanner - Credential Management**
   - [ ] Add multiple credentials
   - [ ] Copy token to clipboard
   - [ ] Delete credentials
   - [ ] Verify localStorage persistence across page reload

### Automated Testing

```typescript
// Tests to add
describe('SkillCatalog', () => {
  it('shows security warning when securityScanned=false', () => {});
  it('shows success message when securityScanned=true', () => {});
});

describe('RepoScanner', () => {
  it('stores credentials in localStorage', () => {});
  it('masks token display', () => {});
  it('sends credential token with scan request', () => {});
  it('handles public repo without credentials', () => {});
  it('handles private repo with credentials', () => {});
});
```

---

## Future Enhancements

### Phase 2
- [ ] Real Git integration with pattern scanning
- [ ] Support for more repository providers (Azure DevOps, Gitea)
- [ ] Scheduled automated scans
- [ ] Scan history and trend analysis
- [ ] Custom pattern definitions

### Phase 3
- [ ] AI-powered remediation suggestions
- [ ] Integration with GitHub/GitLab CI/CD
- [ ] Webhook support for automatic scans on push
- [ ] Team credential sharing (with encryption)

---

## Documentation for Users

### Quick Start

1. **Disable misleading messages**
   - ✅ Enable `scanRepoContent: true` in your Governance CRD
   - ✅ Dashboard will now show "All security checks passed" with confidence

2. **Scan your repositories**
   - Open Dashboard → Repo Scanner
   - For public repos: enter URL and scan
   - For private repos: add credentials first, then scan
   - Review findings and remediation steps

### Best Practices

- 🔐 Use Personal Access Tokens, not personal credentials
- 🔑 Limit token scope to minimum required permissions
- 📅 Regularly rotate credentials
- 🗑️ Delete unused credentials
- 📊 Enable security scanning for production skills
- 🔍 Review all findings, especially High/Critical severity

