'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  BookOpen, ChevronDown, ChevronUp, ShieldAlert, ShieldX, ShieldCheck,
  ExternalLink, Search, X, FileCode, AlertTriangle, CheckCircle2,
  ScanLine, KeyRound, Eye, EyeOff, Loader2, RefreshCw, Lock,
} from 'lucide-react';
import type { SkillCatalogScore, SkillCatalogFinding, SkillCatalogsResponse } from '@/lib/types';

// ---------- Types for inline scan state ----------

interface ScanFinding {
  checkID: string;
  severity: 'Critical' | 'High' | 'Medium' | 'Low';
  category: string;
  title: string;
  remediation: string;
  filePath?: string;
  line?: number;
  matchedPattern?: string;
}

interface FolderSecurityCheck {
  id: string;
  name: string;
  passed: boolean;
  description: string;
  remediation: string;
  severity: string;
}

interface FolderScanResult {
  folderPath: string;
  filesScanned: number;
  score: number;
  status: string;
  securityChecks: FolderSecurityCheck[];
  findings: ScanFinding[];
}

interface ScanResult {
  securityScanned: boolean;
  scannedFiles: number;
  scannedFolder?: string;  // folder path that was scanned (empty = full repo)
  findings: ScanFinding[];
  folderResults: FolderScanResult[];  // per-folder breakdown
  error?: string;
  authRequired?: boolean; // server got 401/403 → show token prompt
  noSkillFiles?: boolean;  // scan succeeded but repo has no SKILL.md files
}

// localStorage key prefix for saved tokens, keyed by hostname (e.g. github.com)
const TOKEN_KEY_PREFIX = 'skillcatalog_token_';

function getStoredToken(repoURL: string): string {
  try {
    const host = new URL(repoURL).hostname;
    return localStorage.getItem(TOKEN_KEY_PREFIX + host) || '';
  } catch { return ''; }
}

function saveToken(repoURL: string, token: string) {
  try {
    const host = new URL(repoURL).hostname;
    if (token) localStorage.setItem(TOKEN_KEY_PREFIX + host, token);
    else localStorage.removeItem(TOKEN_KEY_PREFIX + host);
  } catch { /* ignore */ }
}

/**
 * Extract the folder path to scan from the catalog's websiteUrl.
 *
 * websiteUrl in a SkillCatalog CR often points directly to the skills folder, e.g.:
 *   https://github.com/org/repo/tree/main/skills
 *   https://github.com/org/repo/blob/main/skills/SKILL.md
 *
 * We strip the known GitHub/GitLab "tree/<branch>/" or "blob/<branch>/" prefix
 * and return only the folder path relative to the repo root (e.g. "skills").
 * Returns an empty string if no path can be extracted, which causes the scanner
 * to fall back to scanning the full repository.
 */
function extractFolderPathFromWebsiteUrl(websiteUrl: string | undefined): string {
  if (!websiteUrl) return '';
  try {
    const url = new URL(websiteUrl);
    // pathname looks like:  /owner/repo/tree/main/skills/subfolder
    //                   or  /owner/repo/blob/main/skills/SKILL.md
    const parts = url.pathname.split('/').filter(Boolean);
    // parts[0]=owner, parts[1]=repo, parts[2]=tree|blob, parts[3]=branch, parts[4..]=path
    const markerIdx = parts.findIndex(p => p === 'tree' || p === 'blob');
    if (markerIdx >= 0 && parts.length > markerIdx + 2) {
      // skip owner/repo/tree|blob/<branch>, take the rest as the folder path
      const pathParts = parts.slice(markerIdx + 2); // skip past the branch name too
      // If the last segment looks like a file (has an extension), drop it to get the folder
      const last = pathParts[pathParts.length - 1];
      if (last && last.includes('.')) pathParts.pop();
      return pathParts.join('/');
    }
  } catch { /* ignore invalid URLs */ }
  return '';
}

interface SkillCatalogProps {
  data: SkillCatalogsResponse | null;
  isActive?: boolean;
}

const severityConfig: Record<string, { color: string; bg: string; border: string }> = {
  Critical: { color: '#ef4444', bg: 'bg-red-500/10',    border: 'border-red-500/30' },
  High:     { color: '#f97316', bg: 'bg-orange-500/10', border: 'border-orange-500/30' },
  Medium:   { color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30' },
  Low:      { color: '#3b82f6', bg: 'bg-blue-500/10',   border: 'border-blue-500/30' },
};

const statusConfig: Record<string, { label: string; color: string; bg: string; border: string; icon: typeof ShieldCheck }> = {
  pass:    { label: 'Pass',    color: '#22c55e', bg: 'bg-green-500/10',  border: 'border-green-500/30',  icon: ShieldCheck },
  warning: { label: 'Warning', color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30', icon: ShieldAlert },
  fail:    { label: 'Fail',    color: '#ef4444', bg: 'bg-red-500/10',    border: 'border-red-500/30',    icon: ShieldX },
};

const getScoreColor = (score: number) => {
  if (score >= 90) return '#22c55e';
  if (score >= 70) return '#eab308';
  if (score >= 50) return '#f97316';
  return '#ef4444';
};

// All possible skill governance checks
const allSkillChecks: Record<string, { category: string; title: string; description: string; type: 'metadata' | 'security' }> = {
  'SKL-001': { category: 'Metadata', title: 'Version Specified', description: 'spec.version is set', type: 'metadata' },
  'SKL-002': { category: 'Metadata', title: 'Repository Source Known', description: 'spec.repository.source is github/gitlab/bitbucket', type: 'metadata' },
  'SKL-003': { category: 'Metadata', title: 'HTTPS Repository URL', description: 'spec.repository.url uses HTTPS (not HTTP)', type: 'metadata' },
  'SKL-004': { category: 'Metadata', title: 'Resource UID Label', description: 'agentregistry.dev/resource-uid label present', type: 'metadata' },
  'SKL-005': { category: 'Metadata', title: 'Category Specified', description: 'spec.category is set', type: 'metadata' },
  'SKL-006': { category: 'Metadata', title: 'Description Provided', description: 'spec.description ≥ 20 characters', type: 'metadata' },
  'SKL-007': { category: 'Metadata', title: 'Production Versioning', description: 'Production skills have version pin', type: 'metadata' },
  'SKL-008': { category: 'Metadata', title: 'Organization Repository', description: 'Not a personal GitHub account', type: 'metadata' },
  'SKL-SEC-001': { category: 'Security', title: 'No Prompt Injection', description: 'No prompt injection patterns detected', type: 'security' },
  'SKL-SEC-002': { category: 'Security', title: 'No Privilege Escalation', description: 'No privilege escalation patterns detected', type: 'security' },
  'SKL-SEC-003': { category: 'Security', title: 'No Data Exfiltration', description: 'No external data transfer patterns detected', type: 'security' },
  'SKL-SEC-004': { category: 'Security', title: 'No Credential Harvesting', description: 'No credential harvesting patterns detected', type: 'security' },
  'SKL-SEC-005': { category: 'Security', title: 'Scope Compliance', description: 'No scope-creep keywords for category', type: 'security' },
  'SKL-SEC-006': { category: 'Security', title: 'Safety Guardrails', description: 'Contains safety guardrail phrases where required', type: 'security' },
  'SKL-SEC-007': { category: 'Security', title: 'No Suspicious Downloads',     description: 'No suspicious binary download patterns detected', type: 'security' },
  'SKL-SEC-008': { category: 'Security', title: 'No Hardcoded Secrets',         description: 'No hardcoded credential or secret keywords detected', type: 'security' },
  'SKL-SEC-009': { category: 'Security', title: 'No Financial Execution',       description: 'No autonomous financial operation patterns detected', type: 'security' },
  'SKL-SEC-010': { category: 'Security', title: 'No Untrusted Content',         description: 'No arbitrary URL fetch / content ingestion patterns', type: 'security' },
  'SKL-SEC-011': { category: 'Security', title: 'No External Runtime Dep',      description: 'No unverifiable external runtime dependency patterns', type: 'security' },
  'SKL-SEC-012': { category: 'Security', title: 'No Service Modification',      description: 'No system service installation or persistence patterns', type: 'security' },
  'SKL-SEC-013': { category: 'Security', title: 'Skill Metadata / Frontmatter', description: 'SKILL.md has valid frontmatter with name & description', type: 'security' },
  'SKL-SEC-014': { category: 'Security', title: 'SKILL.md Present',             description: 'Repository contains a SKILL.md file', type: 'security' },
};

function FindingRow({ finding }: { finding: SkillCatalogFinding }) {
  const [open, setOpen] = useState(false);
  const sev = severityConfig[finding.severity] || severityConfig.Low;
  return (
    <div className={`rounded-lg border ${sev.border} ${sev.bg} mb-2 last:mb-0 overflow-hidden`}>
      <button
        className="w-full flex items-start gap-3 p-3 text-left hover:brightness-105 transition-all"
        onClick={() => setOpen(o => !o)}
      >
        <span
          className="text-[10px] font-black px-1.5 py-0.5 rounded shrink-0 mt-0.5"
          style={{ color: sev.color, backgroundColor: `${sev.color}20` }}
        >
          {finding.severity.toUpperCase()}
        </span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs font-mono text-gov-text-3">{finding.checkID}</span>
            <span className="text-xs font-semibold text-gov-text">{finding.title}</span>
          </div>
          {finding.filePath && (
            <div className="flex items-center gap-1 mt-0.5 text-[10px] text-gov-text-3">
              <FileCode className="w-3 h-3" />
              <span className="font-mono truncate">{finding.filePath}{finding.line ? `:${finding.line}` : ''}</span>
            </div>
          )}
        </div>
        {open ? <ChevronUp className="w-4 h-4 text-gov-text-3 shrink-0 mt-0.5" /> : <ChevronDown className="w-4 h-4 text-gov-text-3 shrink-0 mt-0.5" />}
      </button>
      {open && (
        <div className="px-3 pb-3 space-y-2 border-t border-white/5">
          {finding.matchedPattern && (
            <div className="mt-2">
              <p className="text-[10px] text-gov-text-3 font-semibold uppercase tracking-wider mb-1">Matched Pattern</p>
              <code className="text-xs font-mono text-orange-300 bg-gov-bg px-2 py-1 rounded block break-all">
                {finding.matchedPattern}
              </code>
            </div>
          )}
          <div>
            <p className="text-[10px] text-gov-text-3 font-semibold uppercase tracking-wider mb-1">Remediation</p>
            <p className="text-xs text-gov-text-2">{finding.remediation}</p>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Per-folder security checks accordion ─────────────────────────────────────

function FolderSecurityGrid({ folder, defaultOpen = false }: { folder: FolderScanResult; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  const folderScore = folder.score ?? 100;
  const failedChecks = folder.securityChecks.filter(c => !c.passed);
  const allPassed = failedChecks.length === 0;
  const scoreColor = getScoreColor(folderScore);

  return (
    <div className={`rounded-lg border ${allPassed ? 'border-green-500/20' : 'border-red-500/20'} overflow-hidden`}>
      {/* Folder header — click to expand */}
      <button
        type="button"
        className={`w-full flex items-center gap-2 px-3 py-2 text-left transition-colors ${allPassed ? 'bg-green-500/5 hover:bg-green-500/10' : 'bg-red-500/5 hover:bg-red-500/10'}`}
        onClick={() => setOpen(o => !o)}
      >
        <FileCode className="w-3.5 h-3.5 text-gov-text-3 shrink-0" />
        <span className="text-[11px] font-mono font-semibold text-gov-text flex-1 truncate">{folder.folderPath || '(root)'}</span>
        <span className="text-[10px] text-gov-text-3 shrink-0">{folder.filesScanned} file{folder.filesScanned !== 1 ? 's' : ''}</span>
        {/* Score pill */}
        <span
          className="text-[10px] font-black px-1.5 py-0.5 rounded shrink-0 tabular-nums"
          style={{ color: scoreColor, backgroundColor: `${scoreColor}20` }}
        >
          {folderScore}
        </span>
        {/* Pass/fail summary */}
        {allPassed ? (
          <span className="flex items-center gap-1 text-[10px] text-green-400 font-semibold shrink-0">
            <CheckCircle2 className="w-3 h-3" /> All passed
          </span>
        ) : (
          <span className="flex items-center gap-1 text-[10px] text-red-400 font-semibold shrink-0">
            <ShieldX className="w-3 h-3" /> {failedChecks.length} failed
          </span>
        )}
        {open ? <ChevronUp className="w-3.5 h-3.5 text-gov-text-3 shrink-0" /> : <ChevronDown className="w-3.5 h-3.5 text-gov-text-3 shrink-0" />}
      </button>

      {/* Checks grid */}
      {open && (
        <div className="p-2 border-t border-gov-border grid grid-cols-2 gap-1.5">
          {Object.entries(allSkillChecks).filter(([, info]) => info.type === 'security').map(([checkID, checkInfo]) => {
            const check = folder.securityChecks.find(c => c.id === checkID);
            const isFailing = check ? !check.passed : false;
            const color = isFailing ? '#ef4444' : '#22c55e';
            return (
              <div key={checkID} className={`rounded border p-1.5 ${isFailing ? 'bg-red-500/5 border-red-500/20' : 'bg-green-500/5 border-green-500/20'}`}>
                <div className="flex items-start gap-1.5">
                  <div
                    className="w-3.5 h-3.5 rounded flex items-center justify-center flex-shrink-0 mt-0.5 text-white text-[7px] font-bold"
                    style={{ backgroundColor: color }}
                  >
                    {isFailing ? '✕' : '✓'}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-1 flex-wrap">
                      <span className="text-[9px] font-mono font-bold" style={{ color }}>{checkID}</span>
                      <span className="text-[9px] font-semibold text-gov-text truncate">{checkInfo.title}</span>
                    </div>
                    {isFailing && check && (
                      <p className="text-[9px] text-red-400/80 mt-0.5 leading-tight">
                        {check.remediation || check.description}
                      </p>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Per-folder findings (file-level detail) */}
      {open && folder.findings.length > 0 && (
        <div className="px-2 pb-2 border-t border-gov-border space-y-1 pt-2">
          <p className="text-[9px] text-gov-text-3 uppercase tracking-wider font-bold mb-1">File-level findings</p>
          {folder.findings.map((f, i) => {
            const sev = severityConfig[f.severity] || severityConfig.Low;
            return (
              <div key={i} className={`flex items-start gap-1.5 rounded px-2 py-1 ${sev.bg} border ${sev.border}`}>
                <span className="text-[8px] font-black px-1 py-0.5 rounded shrink-0" style={{ color: sev.color, backgroundColor: `${sev.color}20` }}>
                  {f.severity.toUpperCase()}
                </span>
                <div className="min-w-0">
                  <div className="flex items-center gap-1">
                    <span className="text-[9px] font-mono text-gov-text-3">{f.checkID}</span>
                    <span className="text-[9px] text-gov-text">{f.title}</span>
                  </div>
                  {f.filePath && (
                    <span className="text-[9px] font-mono text-gov-text-3 truncate block">
                      {f.filePath}{f.line ? `:${f.line}` : ''}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function CatalogCard({
  catalog,
  scanResult,
  onScanResult,
  autoScanFiredKeys,
  isActive = true,
}: {
  catalog: SkillCatalogScore;
  scanResult: ScanResult | null;
  onScanResult: (r: ScanResult | null) => void;
  autoScanFiredKeys: React.MutableRefObject<Set<string>>;
  isActive?: boolean;
}) {
  const [expanded, setExpanded] = useState(false);

  // Per-card UI state (doesn't need to survive tab switches)
  const [scanning, setScanning] = useState(false);
  const [showScanPanel, setShowScanPanel] = useState(false);
  const [token, setToken] = useState('');
  const [showToken, setShowToken] = useState(false);
  const [saveTokenPref, setSaveTokenPref] = useState(false);

  // Stable key for this catalog
  const catalogKey = `${catalog.namespace}/${catalog.name}`;

  // On mount: pre-fill token from localStorage if we have a repoURL
  useEffect(() => {
    if (catalog.repoURL) {
      const stored = getStoredToken(catalog.repoURL);
      if (stored) { setToken(stored); setSaveTokenPref(true); }
    }
  }, [catalog.repoURL]);

  const runScan = useCallback(async () => {
    if (!catalog.repoURL) return;
    setScanning(true);
    onScanResult(null);
    try {
      if (saveTokenPref && token) saveToken(catalog.repoURL, token);
      else if (!saveTokenPref && catalog.repoURL) saveToken(catalog.repoURL, '');

      // Derive the folder path to scan:
      // 1. Use the path extracted from websiteUrl (e.g. "skills" from the GitHub tree URL)
      // 2. Fall back to scanning the full repo if websiteUrl is absent or has no path
      const folderPath = extractFolderPathFromWebsiteUrl(catalog.websiteUrl);

      console.log('[SkillCatalog] Starting scan for:', catalog.repoURL,
        folderPath ? `(folder: ${folderPath})` : '(full repo — no websiteUrl path)');

      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 60000); // 60s timeout

      const res = await fetch('/api/scan/repo', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repoUrl: catalog.repoURL,
          folderPath,                          // ← skills folder from websiteUrl (or '' for full repo)
          catalogName: catalog.name,
          namespace: catalog.namespace,
          credentialToken: token || undefined,
        }),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      console.log('[SkillCatalog] Scan response status:', res.status);

      if (!res.ok) {
        const errorText = await res.text();
        console.error('[SkillCatalog] Scan request failed:', res.status, errorText);
        onScanResult({
          securityScanned: false,
          scannedFiles: 0,
          findings: [],
          folderResults: [],
          error: `Server error: ${res.status} ${res.statusText}`,
        });
        setShowScanPanel(false);
        return;
      }

      const json = await res.json();
      console.log('[SkillCatalog] Scan result:', json);

      if (json.authRequired) {
        onScanResult({ securityScanned: false, scannedFiles: 0, findings: [], folderResults: [], authRequired: true });
        setShowScanPanel(true);
        return;
      }

      // API response shape: { status, folderResults: [{ folderPath, securityChecks, filesScanned, score, status, findings }] }
      const rawFolderResults: Array<{
        folderPath: string;
        filesScanned: number;
        score: number;
        status: string;
        securityChecks: Array<{ id: string; name: string; passed: boolean; description: string; remediation: string; severity?: string }>;
        findings: Array<{ checkID: string; severity: string; category: string; title: string; remediation: string; filePath?: string; line?: number; matchedPattern?: string }>;
      }> = json.folderResults ?? [];

      const totalFiles = rawFolderResults.reduce((sum, f) => sum + (f.filesScanned ?? 0), 0);

      // Build typed per-folder results
      const folderResults: FolderScanResult[] = rawFolderResults.map(fr => ({
        folderPath: fr.folderPath ?? '',
        filesScanned: fr.filesScanned ?? 0,
        score: fr.score ?? 100,
        status: fr.status ?? 'pass',
        securityChecks: (fr.securityChecks ?? []).map(c => ({
          id: c.id,
          name: c.name,
          passed: c.passed,
          description: c.description,
          remediation: c.remediation ?? c.description,
          severity: c.severity ?? 'High',
        })),
        findings: (fr.findings ?? []).map(f => ({
          checkID: f.checkID,
          severity: f.severity as 'Critical' | 'High' | 'Medium' | 'Low',
          category: f.category ?? 'Security',
          title: f.title,
          remediation: f.remediation,
          filePath: f.filePath,
          line: f.line,
          matchedPattern: f.matchedPattern,
        })),
      }));

      // Aggregate: a check fails if it failed in ANY folder (for the summary findings list)
      const failedMap = new Map<string, { id: string; name: string; description: string; remediation: string; severity: string }>();
      for (const folder of folderResults) {
        for (const c of folder.securityChecks) {
          if (!c.passed && !failedMap.has(c.id)) {
            failedMap.set(c.id, c);
          }
        }
      }
      const findings: ScanFinding[] = Array.from(failedMap.values()).map(c => ({
        checkID: c.id,
        severity: c.severity as 'Critical' | 'High' | 'Medium' | 'Low',
        category: 'Security',
        title: c.name,
        remediation: c.remediation,
      }));

      // Only mark as securityScanned if we actually found and scanned skill files.
      const didScanFiles = totalFiles > 0 || folderResults.length > 0;

      console.log('[SkillCatalog] runScan result for', catalog.name, ':', {
        didScanFiles, totalFiles, foldersCount: folderResults.length,
        findingIDs: findings.map(f => f.checkID),
        securityScanned: json.status === 'success' && didScanFiles,
      });

      onScanResult({
        securityScanned: json.status === 'success' && didScanFiles,
        scannedFiles: totalFiles,
        scannedFolder: json.scanPath || '',
        findings,
        folderResults,
        error: didScanFiles ? json.error : undefined,
        noSkillFiles: !didScanFiles && json.status === 'success',
      });
      setShowScanPanel(false);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      console.error('[SkillCatalog] Scan error:', errorMsg);
      onScanResult({
        securityScanned: false,
        scannedFiles: 0,
        findings: [],
        folderResults: [],
        error: `Network error: ${errorMsg}`,
      });
      setShowScanPanel(false);
    } finally {
      setScanning(false);
    }
  }, [catalog.repoURL, catalog.websiteUrl, catalog.name, catalog.namespace, token, saveTokenPref, onScanResult]);

  // Auto-scan once per catalog (survives tab switches because autoScanFiredKeys lives in parent)
  useEffect(() => {
    if (!isActive) return;                          // don't fire while tab is hidden
    if (!catalog.repoURL) return;
    // Only skip auto-scan if the controller actually scanned repo content (has SKL-SEC-* findings)
    if (catalog.securityScanned && (catalog.findings || []).some(f => f.checkID.startsWith('SKL-SEC-'))) return;
    if (scanResult !== null) return;                // already have results
    if (autoScanFiredKeys.current.has(catalogKey)) return; // already fired for this catalog
    autoScanFiredKeys.current.add(catalogKey);
    const timer = setTimeout(() => runScan(), 500);
    return () => clearTimeout(timer);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isActive]); // re-check whenever tab becomes active

  // Effective values: prefer live scan result over CR data
  // For SECURITY findings: use live scan result when available (fresh browser scan)
  // For METADATA findings: always use catalog.findings (only the controller knows metadata state)
  const metadataFindings: SkillCatalogFinding[] = catalog.findings?.filter(f => !f.checkID.startsWith('SKL-SEC-')) || [];
  const securityFindings: SkillCatalogFinding[] = scanResult
    ? scanResult.findings as SkillCatalogFinding[]
    : catalog.findings?.filter(f => f.checkID.startsWith('SKL-SEC-')) || [];
  const effectiveFindings: SkillCatalogFinding[] = [...metadataFindings, ...securityFindings];
  const effectiveScannedFiles = scanResult?.scannedFiles ?? catalog.scannedFiles ?? 0;

  // Security checks panel is shown when:
  //  (a) a live browser scan was just run (scanResult present with securityScanned=true), OR
  //  (b) the controller has actual SKL-SEC-* findings (real content scan)
  const hasSecurityFindings = securityFindings.length > 0;
  const effectiveSecurityScanned = scanResult?.securityScanned
    ?? (catalog.securityScanned && hasSecurityFindings)
    ?? false;

  // Compute effective score:
  // When a live scan has run, recalculate score from ALL findings (metadata + security)
  // using the same penalty formula as the controller: Critical=-40, High=-25, Medium=-15, Low=-5
  const effectiveScore = (() => {
    if (!scanResult?.securityScanned) return catalog.score;
    const penaltyMap: Record<string, number> = { Critical: 40, High: 25, Medium: 15, Low: 5 };
    const penalty = effectiveFindings.reduce((sum, f) => sum + (penaltyMap[f.severity] ?? 5), 0);
    return Math.max(0, 100 - penalty);
  })();

  const scoreColor = getScoreColor(effectiveScore);
  // Recompute status from effective score when scan has run
  const effectiveStatus = scanResult?.securityScanned
    ? (effectiveScore >= 80 ? 'pass' : effectiveScore >= 50 ? 'warning' : 'fail')
    : catalog.status;
  const status = statusConfig[effectiveStatus] || statusConfig.warning;
  const StatusIcon = status.icon;
  const findings = effectiveFindings;
  const criticalCount = findings.filter(f => f.severity === 'Critical').length;
  const highCount = findings.filter(f => f.severity === 'High').length;
  const mediumCount = findings.filter(f => f.severity === 'Medium').length;
  const lowCount = findings.filter(f => f.severity === 'Low').length;

  const needsToken = !catalog.repoURL
    ? false
    : scanResult?.authRequired || showScanPanel;

  return (
    <div className={`bg-gov-bg rounded-xl border transition-all ${expanded ? 'border-gov-border-light' : 'border-gov-border hover:border-gov-border-light'}`}>
      {/* Header row */}
      <div
        className="flex items-center gap-4 p-4 cursor-pointer"
        onClick={() => setExpanded(e => !e)}
      >
        {/* Score circle */}
        <div
          className="w-12 h-12 rounded-full flex items-center justify-center border-2 font-black text-sm tabular-nums shrink-0"
          style={{ borderColor: scoreColor, color: scoreColor, backgroundColor: `${scoreColor}10` }}
        >
          {effectiveScore}
        </div>

        {/* Name + meta */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-semibold text-gov-text truncate">{catalog.name}</span>
            {catalog.version && (
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-gov-surface text-gov-text-3">
                v{catalog.version}
              </span>
            )}
            {catalog.category && (
              <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded-full bg-purple-500/10 text-purple-400 border border-purple-500/20">
                {catalog.category}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2 mt-1 text-xs text-gov-text-3">
            <span className="font-mono">{catalog.namespace}</span>
            {catalog.repoURL && (
              <>
                <span>·</span>
                <a
                  href={catalog.repoURL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-blue-400 hover:text-blue-300 transition-colors truncate max-w-[200px]"
                  onClick={e => e.stopPropagation()}
                >
                  <ExternalLink className="w-3 h-3 shrink-0" />
                  <span className="truncate">{catalog.repoURL.replace(/^https?:\/\//, '')}</span>
                </a>
              </>
            )}
          </div>
        </div>

        {/* Status badge + scan indicator + findings count + expand */}
        <div className="flex items-center gap-2 shrink-0">
          {/* Scanned indicator */}
          {effectiveSecurityScanned && !scanning && (() => {
            const folder = scanResult?.scannedFolder ?? extractFolderPathFromWebsiteUrl(catalog.websiteUrl);
            return (
              <span className="flex items-center gap-1 text-[10px] text-green-400 px-1.5 py-0.5 rounded bg-green-500/10 border border-green-500/20 font-semibold" title={folder ? `Scanned folder: ${folder}` : 'Full repository scanned'}>
                <ScanLine className="w-3 h-3" />
                Scanned
                {folder
                  ? <span className="font-mono text-green-300 opacity-80">/{folder}</span>
                  : <span className="opacity-60">(full repo)</span>
                }
              </span>
            );
          })()}
          {scanning && (
            <span className="flex items-center gap-1 text-[10px] text-blue-400 px-1.5 py-0.5 rounded bg-blue-500/10 border border-blue-500/20 font-semibold">
              <Loader2 className="w-3 h-3 animate-spin" /> Scanning…
            </span>
          )}
          {/* Status */}
          <div
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg border text-xs font-semibold ${status.bg} ${status.border}`}
            style={{ color: status.color }}
          >
            <StatusIcon className="w-3.5 h-3.5" />
            {status.label}
          </div>
          {/* Severity badges */}
          {criticalCount > 0 && (
            <span className="text-xs font-bold text-red-400 px-1.5 py-0.5 rounded bg-red-500/10">{criticalCount}C</span>
          )}
          {highCount > 0 && (
            <span className="text-xs font-bold text-orange-400 px-1.5 py-0.5 rounded bg-orange-500/10">{highCount}H</span>
          )}
          {mediumCount > 0 && (
            <span className="text-xs font-bold text-yellow-400 px-1.5 py-0.5 rounded bg-yellow-500/10">{mediumCount}M</span>
          )}
          {lowCount > 0 && (
            <span className="text-xs font-bold text-blue-400 px-1.5 py-0.5 rounded bg-blue-500/10">{lowCount}L</span>
          )}
          {expanded ? (
            <ChevronUp className="w-4 h-4 text-gov-text-3 ml-1" />
          ) : (
            <ChevronDown className="w-4 h-4 text-gov-text-3 ml-1" />
          )}
        </div>
      </div>

      {/* ── Inline Scan Panel ── */}
      {expanded && catalog.repoURL && !effectiveSecurityScanned && !scanResult?.noSkillFiles && (
        <div className="border-t border-gov-border px-4 py-3">
          {/* Scan prompt / token form */}
          {(showScanPanel || scanResult?.authRequired) ? (
            <div className="rounded-xl border border-blue-500/20 bg-blue-500/5 p-3 space-y-3">
              <div className="flex items-center gap-2">
                <Lock className="w-4 h-4 text-blue-400 shrink-0" />
                <div>
                  <p className="text-xs font-semibold text-blue-400">
                    {scanResult?.authRequired ? 'Authentication Required' : 'Private Repository — Enter Token'}
                  </p>
                  <p className="text-[10px] text-gov-text-3 mt-0.5">
                    {scanResult?.authRequired
                      ? 'The repository returned 401/403. Provide a Personal Access Token (PAT) with read access.'
                      : 'Provide a PAT with read access to clone this private repository.'}
                  </p>
                </div>
              </div>
              <div className="relative">
                <KeyRound className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gov-text-3" />
                <input
                  type={showToken ? 'text' : 'password'}
                  placeholder="ghp_xxxxxxxxxxxx  /  glpat-xxxxx  /  Bitbucket app password"
                  value={token}
                  onChange={e => setToken(e.target.value)}
                  className="w-full pl-8 pr-10 py-2 bg-gov-bg border border-gov-border rounded-lg text-xs text-gov-text placeholder-gov-text-3 focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10 transition-all font-mono"
                  onClick={e => e.stopPropagation()}
                />
                <button
                  type="button"
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gov-text-3 hover:text-gov-text transition-colors"
                  onClick={e => { e.stopPropagation(); setShowToken(v => !v); }}
                >
                  {showToken ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                </button>
              </div>
              <div className="flex items-center justify-between gap-3">
                <label
                  className="flex items-center gap-1.5 text-[10px] text-gov-text-3 cursor-pointer select-none"
                  onClick={e => e.stopPropagation()}
                >
                  <input
                    type="checkbox"
                    checked={saveTokenPref}
                    onChange={e => setSaveTokenPref(e.target.checked)}
                    className="w-3 h-3 rounded accent-blue-500"
                  />
                  Remember token for this host (stored locally in browser)
                </label>
                <div className="flex gap-2">
                  <button
                    type="button"
                    className="px-3 py-1.5 text-xs rounded-lg border border-gov-border text-gov-text-3 hover:text-gov-text hover:border-gov-border-light transition-all"
                    onClick={e => { e.stopPropagation(); setShowScanPanel(false); onScanResult(null); }}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={scanning}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-semibold transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                    onClick={e => { e.stopPropagation(); runScan(); }}
                  >
                    {scanning ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <ScanLine className="w-3.5 h-3.5" />}
                    {scanning ? 'Scanning…' : 'Scan Repository'}
                  </button>
                </div>
              </div>
            </div>
          ) : (
            /* Default: compact scan bar */
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 text-xs text-gov-text-3 flex-1 min-w-0">
                <AlertTriangle className="w-3.5 h-3.5 text-yellow-400 shrink-0" />
                <span>Security scan not yet run for this catalog.</span>
              </div>
              {scanResult?.error && (
                <span className="text-[10px] text-red-400 truncate max-w-[200px]" title={scanResult.error}>
                  ✕ {scanResult.error}
                </span>
              )}
              <div className="flex gap-2 shrink-0">
                {/* Quick scan (public) */}
                <button
                  type="button"
                  disabled={scanning}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg bg-purple-600/20 hover:bg-purple-600/30 text-purple-300 border border-purple-500/30 font-semibold transition-all disabled:opacity-50"
                  onClick={e => { e.stopPropagation(); runScan(); }}
                  title="Scan as public repository"
                >
                  {scanning ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <ScanLine className="w-3.5 h-3.5" />}
                  {scanning ? 'Scanning…' : 'Quick Scan'}
                </button>
                {/* Private / with token */}
                <button
                  type="button"
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg bg-gov-surface hover:bg-gov-bg text-gov-text-2 border border-gov-border hover:border-gov-border-light font-semibold transition-all"
                  onClick={e => { e.stopPropagation(); setShowScanPanel(true); }}
                  title="Scan with credentials (private repo)"
                >
                  <KeyRound className="w-3.5 h-3.5" />
                  Private / Token
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* No SKILL.md found banner */}
      {expanded && catalog.repoURL && scanResult?.noSkillFiles && !scanning && (
        <div className="border-t border-gov-border px-4 py-3 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 text-xs text-gov-text-3 flex-1 min-w-0">
            <AlertTriangle className="w-3.5 h-3.5 text-yellow-400 shrink-0" />
            <span>No <code className="text-blue-400">SKILL.md</code> file found in the repository — security checks require a skill manifest.</span>
          </div>
          <button
            type="button"
            disabled={scanning}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg text-gov-text-3 hover:text-gov-text border border-gov-border hover:border-gov-border-light transition-all disabled:opacity-50 shrink-0"
            onClick={e => { e.stopPropagation(); runScan(); }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${scanning ? 'animate-spin' : ''}`} />
            Re-scan
          </button>
        </div>
      )}

      {/* Re-scan button when already scanned */}
      {expanded && catalog.repoURL && (effectiveSecurityScanned || (scanResult !== null && !scanResult.noSkillFiles && !scanResult.authRequired)) && (
        <div className="border-t border-gov-border px-4 py-2 flex justify-end">
          <button
            type="button"
            disabled={scanning}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg text-gov-text-3 hover:text-gov-text border border-gov-border hover:border-gov-border-light transition-all disabled:opacity-50"
            onClick={e => { e.stopPropagation(); runScan(); }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${scanning ? 'animate-spin' : ''}`} />
            Re-scan
          </button>
        </div>
      )}

      {/* Expanded detail */}
      {expanded && (
        <div className="border-t border-gov-border px-4 pb-4 pt-3 space-y-4">
          <div className="bg-gov-bg rounded-xl border border-gov-border p-4 space-y-4">
            {/* Stats row */}
            <div className="flex items-center gap-4 text-xs text-gov-text-3 flex-wrap">
              {effectiveScannedFiles > 0 && (
                <span className="flex items-center gap-1">
                  <FileCode className="w-3.5 h-3.5" />
                  {effectiveScannedFiles} file{effectiveScannedFiles !== 1 ? 's' : ''} scanned
                </span>
              )}
              {/* Show which folder was scanned */}
              {(() => {
                const folder = scanResult?.scannedFolder ?? extractFolderPathFromWebsiteUrl(catalog.websiteUrl);
                const hasAnyInfo = scanResult !== null || !!catalog.websiteUrl;
                if (!hasAnyInfo) return null;
                return (
                  <span className="flex items-center gap-1" title="Folder scanned (derived from websiteUrl)">
                    <ScanLine className="w-3.5 h-3.5" />
                    {folder
                      ? <>Folder: <code className="font-mono text-blue-400 px-1 py-0.5 rounded bg-blue-500/10">{folder}</code></>
                      : <span className="italic">Full repository</span>
                    }
                  </span>
                );
              })()}
              <span className="flex items-center gap-1">
                <AlertTriangle className="w-3.5 h-3.5" />
                {findings.length} finding{findings.length !== 1 ? 's' : ''}
              </span>
            </div>

            {/* Check Breakdown */}
            <div>
              <p className="text-xs text-gov-text-3 uppercase tracking-wider font-semibold mb-3">All Checks</p>
              <div className="space-y-2">
                {/* Metadata Checks */}
                <div>
                  <p className="text-[10px] text-gov-text-3 font-bold uppercase tracking-wider mb-2 opacity-70">Metadata Checks (SKL-001 to SKL-008)</p>
                  <div className="grid grid-cols-2 gap-2">
                    {Object.entries(allSkillChecks).filter(([_, info]) => info.type === 'metadata').map(([checkID, checkInfo]) => {
                      // Controller findings have checkID format like "SKL-006-resourcename-filepath", extract the base
                      // Metadata checkIDs: "SKL-001" → 2 parts (SKL + 001)
                      // Security checkIDs: "SKL-SEC-001" → 3 parts (SKL + SEC + 001)
                      const hasFinding = findings.some(f => {
                        const parts = f.checkID.split('-');
                        const baseCheckID = parts.length >= 3 && parts[1] === 'SEC'
                          ? parts.slice(0, 3).join('-')   // SKL-SEC-003
                          : parts.slice(0, 2).join('-');  // SKL-006
                        return baseCheckID === checkID || f.checkID === checkID;
                      });
                      const isFailing = hasFinding;
                      const color = isFailing ? '#ef4444' : '#22c55e';
                      return (
                        <div key={checkID} className={`rounded-lg border p-2 ${isFailing ? 'bg-red-500/5 border-red-500/20' : 'bg-green-500/5 border-green-500/20'}`}>
                          <div className="flex items-start gap-2">
                            <div
                              className="w-4 h-4 rounded flex items-center justify-center flex-shrink-0 mt-0.5 text-white text-[8px] font-bold"
                              style={{ backgroundColor: color }}
                            >
                              {isFailing ? '✕' : '✓'}
                            </div>
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-1.5">
                                <span className="text-[10px] font-mono font-bold" style={{ color }}>{checkID}</span>
                                <span className="text-[10px] font-semibold text-gov-text truncate">{checkInfo.title}</span>
                              </div>
                              <p className="text-[9px] text-gov-text-3 mt-0.5">{checkInfo.description}</p>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>

                {/* Security Checks — per-folder accordion when scan ran, flat list from controller otherwise */}
                {effectiveSecurityScanned ? (
                  <div className="mt-3">
                    <p className="text-[10px] text-gov-text-3 font-bold uppercase tracking-wider mb-2 opacity-70">
                      Security Checks (SKL-SEC-001 → SKL-SEC-014)
                      {scanResult?.folderResults && scanResult.folderResults.length > 1 && (
                        <span className="ml-2 text-blue-400 normal-case font-normal">
                          — {scanResult.folderResults.length} folders
                        </span>
                      )}
                    </p>

                    {/* Per-folder breakdown from live scan */}
                    {scanResult?.folderResults && scanResult.folderResults.length > 0 ? (
                      <div className="space-y-2">
                        {scanResult.folderResults.map((folder, i) => (
                          <FolderSecurityGrid
                            key={folder.folderPath || i}
                            folder={folder}
                            defaultOpen={scanResult.folderResults.length === 1 || !folder.securityChecks.every(c => c.passed)}
                          />
                        ))}
                      </div>
                    ) : (
                      /* Fallback: flat grid for controller-sourced security findings (no per-folder data) */
                      <div className="grid grid-cols-2 gap-2">
                        {Object.entries(allSkillChecks).filter(([, info]) => info.type === 'security').map(([checkID, checkInfo]) => {
                          const hasFinding = securityFindings.some(f => {
                            const parts = f.checkID.split('-');
                            const base = parts.length >= 3 && parts[1] === 'SEC'
                              ? parts.slice(0, 3).join('-')
                              : parts.slice(0, 2).join('-');
                            return base === checkID || f.checkID === checkID;
                          });
                          const color = hasFinding ? '#ef4444' : '#22c55e';
                          return (
                            <div key={checkID} className={`rounded-lg border p-2 ${hasFinding ? 'bg-red-500/5 border-red-500/20' : 'bg-green-500/5 border-green-500/20'}`}>
                              <div className="flex items-start gap-2">
                                <div className="w-4 h-4 rounded flex items-center justify-center flex-shrink-0 mt-0.5 text-white text-[8px] font-bold" style={{ backgroundColor: color }}>
                                  {hasFinding ? '✕' : '✓'}
                                </div>
                                <div className="flex-1 min-w-0">
                                  <div className="flex items-center gap-1.5">
                                    <span className="text-[10px] font-mono font-bold" style={{ color }}>{checkID}</span>
                                    <span className="text-[10px] font-semibold text-gov-text truncate">{checkInfo.title}</span>
                                  </div>
                                  <p className="text-[9px] text-gov-text-3 mt-0.5">{checkInfo.description}</p>
                                </div>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="mt-3 flex items-start gap-2 rounded-lg border border-yellow-500/20 bg-yellow-500/5 px-3 py-2.5">
                    <AlertTriangle className="w-3.5 h-3.5 text-yellow-400 shrink-0 mt-0.5" />
                    <div>
                      {scanning && (
                        <>
                          <p className="text-[10px] font-bold text-yellow-400 uppercase tracking-wider flex items-center gap-1.5">
                            <Loader2 className="w-3 h-3 animate-spin" />
                            Scanning Repository
                          </p>
                          <p className="text-[10px] text-gov-text-3 mt-0.5">
                            Running security checks (SKL-SEC-001 through SKL-SEC-014)
                            {extractFolderPathFromWebsiteUrl(catalog.websiteUrl)
                              ? <> in folder <code className="text-blue-400">{extractFolderPathFromWebsiteUrl(catalog.websiteUrl)}</code>…</>
                              : <> across the full repository…</>
                            }
                          </p>
                        </>
                      )}
                      {!scanning && scanResult?.authRequired && (
                        <>
                          <p className="text-[10px] font-bold text-orange-400 uppercase tracking-wider">⚠️ Private Repository</p>
                          <p className="text-[10px] text-gov-text-3 mt-0.5">
                            This repository is private. Please provide a personal access token to run security scans.
                          </p>
                        </>
                      )}
                      {!scanning && scanResult?.noSkillFiles && (
                        <>
                          <p className="text-[10px] font-bold text-yellow-400 uppercase tracking-wider">No SKILL.md Found</p>
                          <p className="text-[10px] text-gov-text-3 mt-0.5">
                            Repository scanned but no <code className="text-blue-400">SKILL.md</code> file was found. Security checks require a skill manifest.
                          </p>
                        </>
                      )}
                      {!scanning && !scanResult?.authRequired && !scanResult?.noSkillFiles && (
                        <>
                          <p className="text-[10px] font-bold text-yellow-400 uppercase tracking-wider">Security Checks Not Run</p>
                          <p className="text-[10px] text-gov-text-3 mt-0.5">
                            {catalog.repoURL
                              ? <>Use the scan buttons above to run SKL-SEC-001 through SKL-SEC-014
                                  {extractFolderPathFromWebsiteUrl(catalog.websiteUrl)
                                    ? <> on folder <code className="text-blue-400">{extractFolderPathFromWebsiteUrl(catalog.websiteUrl)}</code>.</>
                                    : <> on this repository.</>
                                  }</>
                              : <>No repository URL configured.</>
                            }
                          </p>
                        </>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Summary findings — only for metadata findings; per-folder details are in the accordion above */}
            {findings.filter(f => !f.checkID.startsWith('SKL-SEC-')).length > 0 && (
              <div>
                <p className="text-xs font-semibold text-gov-text-3 uppercase tracking-wider mb-2">Metadata Issues Found</p>
                {findings.filter(f => !f.checkID.startsWith('SKL-SEC-')).map((f, i) => (
                  <FindingRow key={`${f.checkID}-${i}`} finding={f} />
                ))}
              </div>
            )}

            {/* No issues message */}
            {findings.length === 0 && (
              <div className="flex items-center gap-2 text-sm py-2">
                <CheckCircle2 className="w-4 h-4 text-green-400 shrink-0" />
                {effectiveSecurityScanned ? (
                  <span className="text-green-400">All checks passed — this skill catalog is fully compliant with governance policies.</span>
                ) : (
                  <span className="text-green-400">All metadata checks passed.</span>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default function SkillCatalog({ data, isActive = true }: SkillCatalogProps) {
  const [filterStatus, setFilterStatus] = useState<'all' | 'pass' | 'warning' | 'fail'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  // Lift scan state OUT of CatalogCard so it survives tab switches.
  // Key = `${namespace}/${name}`, value = the ScanResult for that catalog.
  // Initialise from sessionStorage so results survive page refreshes within the same browser session.
  const [scanResults, setScanResults] = useState<Map<string, ScanResult>>(() => {
    try {
      const raw = sessionStorage.getItem('skillcatalog_scan_results');
      if (raw) {
        const obj = JSON.parse(raw) as Record<string, ScanResult>;
        return new Map(Object.entries(obj));
      }
    } catch { /* ignore parse errors */ }
    return new Map();
  });

  const setScanResult = useCallback((key: string, result: ScanResult | null) => {
    setScanResults(prev => {
      const next = new Map(prev);
      if (result === null) next.delete(key);
      else next.set(key, result);
      // Persist to sessionStorage (survives refresh, cleared when tab closes)
      try {
        const obj: Record<string, ScanResult> = {};
        next.forEach((v, k) => { obj[k] = v; });
        sessionStorage.setItem('skillcatalog_scan_results', JSON.stringify(obj));
      } catch { /* ignore quota errors */ }
      return next;
    });
  }, []);

  // Track which catalogs have already had auto-scan fired (survives tab switches)
  // Pre-populate from sessionStorage keys so we don't re-scan on page refresh when results are already cached.
  const autoScanFiredKeys = useRef<Set<string>>(new Set(
    (() => {
      try {
        const raw = sessionStorage.getItem('skillcatalog_scan_results');
        if (raw) return Object.keys(JSON.parse(raw) as Record<string, ScanResult>);
      } catch { /* ignore */ }
      return [];
    })()
  ));

  const catalogs = data?.catalogs || [];
  const rawSummary = data?.summary || { total: 0, passCount: 0, warningCount: 0, failCount: 0, averageScore: 0 };

  // Recompute summary counts using effective scores (same penalty formula as CatalogCard)
  const penaltyMapSummary: Record<string, number> = { Critical: 40, High: 25, Medium: 15, Low: 5 };
  const effectiveSummary = (() => {
    if (scanResults.size === 0) return rawSummary;
    let passCount = 0, warningCount = 0, failCount = 0, scoreSum = 0;
    for (const c of catalogs) {
      const key = `${c.namespace}/${c.name}`;
      const sr = scanResults.get(key) ?? null;
      let effScore: number;
      if (sr?.securityScanned) {
        const metaFindings = (c.findings || []).filter(f => !f.checkID.startsWith('SKL-SEC-'));
        const allFindings = [...metaFindings, ...(sr.findings as typeof c.findings)];
        const penalty = allFindings.reduce((s, f) => s + (penaltyMapSummary[f.severity] ?? 5), 0);
        effScore = Math.max(0, 100 - penalty);
      } else {
        effScore = c.score;
      }
      const effStatus = effScore >= 80 ? 'pass' : effScore >= 50 ? 'warning' : 'fail';
      if (effStatus === 'pass') passCount++;
      else if (effStatus === 'warning') warningCount++;
      else failCount++;
      scoreSum += effScore;
    }
    const total = catalogs.length;
    return {
      total,
      passCount,
      warningCount,
      failCount,
      averageScore: total > 0 ? Math.round(scoreSum / total) : 0,
    };
  })();
  const summary = effectiveSummary;

  const filtered = catalogs.filter(c => {
    if (filterStatus !== 'all') {
      // Use effective status if we have a scan result for this catalog
      const key = `${c.namespace}/${c.name}`;
      const sr = scanResults.get(key) ?? null;
      let effStatus: string;
      if (sr?.securityScanned) {
        const metaF = (c.findings || []).filter(f => !f.checkID.startsWith('SKL-SEC-'));
        const allF = [...metaF, ...(sr.findings as typeof c.findings)];
        const penalty = allF.reduce((s, f) => s + (penaltyMapSummary[f.severity] ?? 5), 0);
        const score = Math.max(0, 100 - penalty);
        effStatus = score >= 80 ? 'pass' : score >= 50 ? 'warning' : 'fail';
      } else {
        effStatus = c.status;
      }
      if (effStatus !== filterStatus) return false;
    }
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return (
      c.name.toLowerCase().includes(q) ||
      c.namespace.toLowerCase().includes(q) ||
      (c.category && c.category.toLowerCase().includes(q)) ||
      (c.repoURL && c.repoURL.toLowerCase().includes(q))
    );
  });

  // Sort: fail first, then warning, then pass
  const statusOrder: Record<string, number> = { fail: 0, warning: 1, pass: 2 };
  const sorted = [...filtered].sort((a, b) => (statusOrder[a.status] ?? 3) - (statusOrder[b.status] ?? 3));

  const avgColor = getScoreColor(summary.averageScore);

  return (
    <div className="bg-gov-surface rounded-2xl border border-gov-border">
      {/* Header */}
      <div className="p-5 border-b border-gov-border">
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="flex items-center gap-2">
              <BookOpen className="w-5 h-5 text-purple-400" />
              <h2 className="text-lg font-bold">Skill Catalog Governance</h2>
            </div>
            <p className="text-xs text-gov-text-3 mt-1">
              SkillCatalog resources scanned for security, metadata, and content policy compliance
            </p>
          </div>
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-gov-bg border border-gov-border">
            <span className="text-xs text-gov-text-3">Avg Score</span>
            <span className="text-sm font-black tabular-nums" style={{ color: avgColor }}>
              {summary.averageScore}
            </span>
          </div>
        </div>

        {/* Summary cards */}
        <div className="grid grid-cols-4 gap-3 mb-4">
          <div className="bg-gov-bg rounded-xl border border-gov-border p-3 text-center">
            <p className="text-2xl font-black tabular-nums">{summary.total}</p>
            <p className="text-[10px] text-gov-text-3 uppercase tracking-wider font-semibold mt-1">Total</p>
          </div>
          <div className="bg-gov-bg rounded-xl border border-green-500/20 p-3 text-center">
            <p className="text-2xl font-black tabular-nums text-green-400">{summary.passCount}</p>
            <p className="text-[10px] text-green-400/70 uppercase tracking-wider font-semibold mt-1">Pass</p>
          </div>
          <div className="bg-gov-bg rounded-xl border border-yellow-500/20 p-3 text-center">
            <p className="text-2xl font-black tabular-nums text-yellow-400">{summary.warningCount}</p>
            <p className="text-[10px] text-yellow-400/70 uppercase tracking-wider font-semibold mt-1">Warning</p>
          </div>
          <div className="bg-gov-bg rounded-xl border border-red-500/20 p-3 text-center">
            <p className="text-2xl font-black tabular-nums text-red-400">{summary.failCount}</p>
            <p className="text-[10px] text-red-400/70 uppercase tracking-wider font-semibold mt-1">Fail</p>
          </div>
        </div>

        {/* Search + filters */}
        <div className="space-y-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gov-text-3" />
            <input
              type="text"
              placeholder="Search by name, namespace, category, or repo URL..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-10 py-2.5 bg-gov-bg border border-gov-border rounded-lg text-sm text-gov-text placeholder-gov-text-3 focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10 transition-all"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gov-text-3 hover:text-gov-text transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>
          <div className="flex gap-2">
            {(['all', 'pass', 'warning', 'fail'] as const).map(s => (
              <button
                key={s}
                onClick={() => setFilterStatus(s)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all capitalize ${
                  filterStatus === s
                    ? 'bg-gov-accent/15 text-blue-400 border border-blue-500/30'
                    : 'text-gov-text-3 hover:text-gov-text-2 bg-gov-bg border border-gov-border hover:border-gov-border-light'
                }`}
              >
                {s === 'all' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
                <span className="ml-1 tabular-nums">
                  {s === 'all'
                    ? catalogs.length
                    : s === 'pass'
                    ? summary.passCount
                    : s === 'warning'
                    ? summary.warningCount
                    : summary.failCount}
                </span>
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Catalog list */}
      <div className="p-4 space-y-3">
        {sorted.length === 0 && catalogs.length === 0 && (
          <div className="text-center py-12 text-gov-text-3">
            <BookOpen className="w-10 h-10 mx-auto mb-3 opacity-30" />
            <p className="text-sm font-medium">No SkillCatalog resources found</p>
            <p className="text-xs mt-1">
              Deploy SkillCatalog resources (agentregistry.dev/v1alpha1) to see governance scores here.
            </p>
          </div>
        )}
        {sorted.length === 0 && catalogs.length > 0 && (
          <div className="text-center py-8 text-gov-text-3 text-sm">
            No skill catalogs match the current filter.
          </div>
        )}
        {sorted.map(catalog => {
          const key = `${catalog.namespace}/${catalog.name}`;
          return (
            <CatalogCard
              key={key}
              catalog={catalog}
              scanResult={scanResults.get(key) ?? null}
              onScanResult={(r) => setScanResult(key, r)}
              autoScanFiredKeys={autoScanFiredKeys}
              isActive={isActive}
            />
          );
        })}
      </div>
    </div>
  );
}
