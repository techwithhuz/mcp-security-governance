'use client';

import { useState } from 'react';
import {
  Scan, Github, Loader, CheckCircle2, XCircle, AlertCircle, Eye, EyeOff,
  Info, Shield, AlertTriangle, Folder, FolderOpen, ChevronDown,
  ChevronUp, FileText, ShieldCheck, ShieldAlert, GitBranch, KeyRound,
} from 'lucide-react';

// ─── Types ────────────────────────────────────────────────────────────────────

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

interface SecurityCheck {
  id: string;
  name: string;
  passed: boolean;
  description: string;
  findingCount: number;
}

interface FolderScanResult {
  folderPath: string;
  filesScanned: number;
  findings: ScanFinding[];
  securityChecks: SecurityCheck[];
  score: number;
  status: 'pass' | 'warning' | 'fail';
}

interface ScanResult {
  status: 'success' | 'error';
  repoUrl: string;
  scanPath: string;
  totalFilesScanned: number;
  totalFindings: number;
  folderResults: FolderScanResult[];
  error?: string;
}

// ─── Constants ────────────────────────────────────────────────────────────────

const CHECK_META: Record<string, { label: string; description: string }> = {
  // ── Compromised-skill checks (E004, E006 family) ─────────────────────────
  'SKL-SEC-001': { label: 'Prompt Injection (E004)',          description: 'No patterns that attempt to override AI system prompts or safety guidelines' },
  'SKL-SEC-002': { label: 'Malicious Code / Priv-Esc',        description: 'No privilege escalation, shell injection, or obfuscated execution patterns' },
  'SKL-SEC-003': { label: 'Data Exfiltration',                description: 'No external data transmission or exfiltration patterns' },
  'SKL-SEC-004': { label: 'Insecure Credential Handling',     description: 'No instructions requiring verbatim credential output or credential harvesting' },
  'SKL-SEC-005': { label: 'Scope Creep',                      description: 'Skill stays within its declared capability scope' },
  // ── Guardrail check ──────────────────────────────────────────────────────
  'SKL-SEC-006': { label: 'Missing Safety Guardrails',        description: 'Explicit safety guardrail phrases (do not, never, must not…) are present' },
  // ── Extended security checks ──────────────────────────────────────────────
  'SKL-SEC-007': { label: 'Suspicious Download URL',          description: 'No suspicious executables, URL shorteners, or untrusted download links' },
  'SKL-SEC-008': { label: 'Hardcoded Secrets',                description: 'No API keys, tokens, private keys, or secrets embedded in skill content' },
  'SKL-SEC-009': { label: 'Financial Execution',              description: 'No direct financial transaction or crypto trading capabilities without consent gates' },
  'SKL-SEC-010': { label: 'Untrusted Content Exposure',       description: 'Skill does not expose agent to arbitrary user-supplied URLs or untrusted content' },
  'SKL-SEC-011': { label: 'External Runtime Dependency',      description: 'Skill does not fetch instructions or code from external URLs at runtime' },
  'SKL-SEC-012': { label: 'System Service Modification',      description: 'No system service, cron, or startup script modification patterns' },
  'SKL-SEC-013': { label: 'Skill Metadata (Frontmatter)',     description: 'SKILL.md has valid YAML frontmatter with name and description fields' },
  'SKL-SEC-014': { label: 'Missing SKILL.md',                 description: 'Folder contains a SKILL.md file documenting purpose and security properties' },
};

const SEVERITY_COLOR: Record<string, string> = {
  Critical: 'text-red-400',
  High:     'text-orange-400',
  Medium:   'text-yellow-400',
  Low:      'text-blue-400',
};

const SEVERITY_BG: Record<string, string> = {
  Critical: 'bg-red-500/15 border-red-500/30',
  High:     'bg-orange-500/15 border-orange-500/30',
  Medium:   'bg-yellow-500/15 border-yellow-500/30',
  Low:      'bg-blue-500/15 border-blue-500/30',
};

// ─── Component ────────────────────────────────────────────────────────────────

export default function RepoScanner() {
  const [repoUrl, setRepoUrl]       = useState('');
  const [folderPath, setFolderPath] = useState('');
  const [isPrivate, setIsPrivate]   = useState(false);
  const [runtimeToken, setRuntimeToken] = useState('');
  const [showRuntimeToken, setShowRuntimeToken] = useState(false);

  const [scanning, setScanning]     = useState(false);
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [error, setError]           = useState<string | null>(null);
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());

  // ── helpers ───────────────────────────────────────────────────────────────

  const toggleFolder = (path: string) => setExpandedFolders(prev => {
    const next = new Set(prev);
    next.has(path) ? next.delete(path) : next.add(path);
    return next;
  });

  const handleScan = async () => {
    if (!repoUrl.trim()) { setError('Please enter a repository URL'); return; }
    if (isPrivate && !runtimeToken.trim()) { setError('Please enter a Personal Access Token for private repositories'); return; }

    setError(null);
    setScanning(true);
    setScanResult(null);
    setExpandedFolders(new Set());

    try {
      const scanRequest: Record<string, unknown> = {
        repoUrl:    repoUrl.trim(),
        folderPath: folderPath.trim() || '',
        isPrivate,
      };
      if (isPrivate && runtimeToken.trim()) {
        scanRequest.credentialToken = runtimeToken.trim();
      }

      const response = await fetch('/api/scan/repo', {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify(scanRequest),
      });

      let data: ScanResult;
      const ct = response.headers.get('content-type') || '';
      if (ct.includes('application/json')) {
        data = await response.json();
      } else {
        const text = await response.text();
        try { data = JSON.parse(text); }
        catch { setError(`Unexpected server response: ${text.slice(0, 150)}`); return; }
      }

      if (!response.ok) { setError((data as any).error || 'Scan failed'); return; }

      setScanResult(data);
      if (data.folderResults?.length) setExpandedFolders(new Set([data.folderResults[0].folderPath]));
    } catch (err: any) {
      setError(err.message || 'Failed to scan repository');
    } finally {
      setScanning(false);
    }
  };

  return (
    <div className="space-y-4">

      {/* ── Input card ──────────────────────────────────────────────────────── */}
      <div className="bg-gov-surface rounded-2xl border border-gov-border">
        {/* Header */}
        <div className="p-5 border-b border-gov-border">
          <div className="flex items-center gap-2 mb-1">
            <Scan className="w-5 h-5 text-cyan-400" />
            <h2 className="text-lg font-bold">Skills On-Demand Scanner</h2>
          </div>
          <p className="text-xs text-gov-text-3">
            Point at any GitHub or GitLab repository folder and run all 6 SKL-SEC security checks across every{' '}
            <code className="font-mono text-cyan-400">skills.md</code> file, including nested sub-folders.
          </p>
        </div>

        <div className="p-5 space-y-4">
          {/* Two-col inputs */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-semibold text-gov-text mb-1.5">
                Repository URL <span className="text-red-400">*</span>
              </label>
              <div className="relative">
                <Github className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gov-text-3 pointer-events-none" />
                <input type="text" placeholder="https://github.com/owner/repo" value={repoUrl} onChange={e => setRepoUrl(e.target.value)}
                  className="w-full pl-9 pr-3 py-2 bg-gov-bg border border-gov-border rounded-lg text-sm text-gov-text placeholder-gov-text-3 focus:outline-none focus:border-cyan-500/50 focus:ring-2 focus:ring-cyan-500/10 transition-all" />
              </div>
              <p className="text-xs text-gov-text-3 mt-1">GitHub or GitLab HTTPS URL</p>
            </div>
            <div>
              <label className="block text-sm font-semibold text-gov-text mb-1.5">
                Skills Folder Path
                <span className="ml-1.5 text-[10px] font-normal text-gov-text-3">(leave blank = repo root)</span>
              </label>
              <div className="relative">
                <Folder className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gov-text-3 pointer-events-none" />
                <input type="text" placeholder="e.g.  skills  or  agents/skills" value={folderPath} onChange={e => setFolderPath(e.target.value)}
                  className="w-full pl-9 pr-3 py-2 bg-gov-bg border border-gov-border rounded-lg text-sm text-gov-text placeholder-gov-text-3 focus:outline-none focus:border-cyan-500/50 focus:ring-2 focus:ring-cyan-500/10 transition-all" />
              </div>
              <p className="text-xs text-gov-text-3 mt-1">Sub-folders are scanned recursively</p>
            </div>
          </div>

          {/* Private toggle */}
          <div className="flex items-center gap-3 p-3 bg-gov-bg rounded-lg border border-gov-border/50">
            <input type="checkbox" id="isPrivate" checked={isPrivate} onChange={e => { setIsPrivate(e.target.checked); if (!e.target.checked) setRuntimeToken(''); }} className="w-4 h-4 rounded accent-cyan-500" />
            <label htmlFor="isPrivate" className="flex-1 text-sm cursor-pointer">
              <div className="font-semibold text-gov-text">Private Repository</div>
              <div className="text-xs text-gov-text-3">Requires a Personal Access Token (used only for this scan)</div>
            </label>
          </div>

          {/* Runtime token input — only shown for private repos, never saved */}
          {isPrivate && (
            <div>
              <label className="block text-sm font-semibold text-gov-text mb-1.5">
                Personal Access Token <span className="text-red-400">*</span>
              </label>
              <div className="flex gap-2">
                <div className="relative flex-1">
                  <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gov-text-3 pointer-events-none" />
                  <input
                    type={showRuntimeToken ? 'text' : 'password'}
                    placeholder="ghp_... or glpat-..."
                    value={runtimeToken}
                    onChange={e => setRuntimeToken(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 bg-gov-bg border border-gov-border rounded-lg text-sm text-gov-text placeholder-gov-text-3 focus:outline-none focus:border-cyan-500/50 focus:ring-2 focus:ring-cyan-500/10 transition-all"
                  />
                </div>
                <button onClick={() => setShowRuntimeToken(!showRuntimeToken)} className="px-3 py-2 bg-gov-bg border border-gov-border rounded-lg hover:bg-gov-bg/80 transition-all">
                  {showRuntimeToken ? <EyeOff className="w-4 h-4 text-gov-text-3" /> : <Eye className="w-4 h-4 text-gov-text-3" />}
                </button>
              </div>
              <p className="text-xs text-gov-text-3 mt-1 flex items-center gap-1">
                <Shield className="w-3 h-3 text-green-400 shrink-0" />
                Token is used only for this scan and is never saved or persisted.
              </p>
            </div>
          )}

          {/* Error */}
          {error && (
            <div className="p-3 bg-red-500/10 border border-red-500/30 rounded-lg flex items-start gap-2 text-sm text-red-400">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <div>{error}</div>
            </div>
          )}

          {/* Scan button */}
          <button onClick={handleScan} disabled={scanning}
            className="w-full px-4 py-3 bg-cyan-500 hover:bg-cyan-600 disabled:bg-cyan-500/40 text-white font-semibold rounded-lg transition-all flex items-center justify-center gap-2 text-sm">
            {scanning ? <><Loader className="w-4 h-4 animate-spin" /> Scanning repository…</> : <><Scan className="w-4 h-4" /> Scan Repository</>}
          </button>
        </div>
      </div>

      {/* ── Results ─────────────────────────────────────────────────────────── */}
      {scanResult && (
        <div className="space-y-4">

          {/* Summary bar */}
          <div className={`rounded-2xl border p-5 ${scanResult.status === 'error' ? 'bg-red-500/5 border-red-500/30' : 'bg-gov-surface border-gov-border'}`}>
            {scanResult.status === 'error' ? (
              <div className="flex items-start gap-3 text-red-400">
                <XCircle className="w-5 h-5 shrink-0 mt-0.5" />
                <div>
                  <div className="font-semibold">Scan failed</div>
                  <div className="text-sm mt-1 text-red-400/80">{scanResult.error}</div>
                </div>
              </div>
            ) : (
              <>
                <div className="flex items-center justify-between mb-4 flex-wrap gap-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl bg-cyan-500/15 flex items-center justify-center shrink-0">
                      <GitBranch className="w-5 h-5 text-cyan-400" />
                    </div>
                    <div>
                      <div className="font-bold text-gov-text">Scan Complete</div>
                      <div className="text-xs text-gov-text-3 font-mono mt-0.5 truncate max-w-[420px]">
                        {scanResult.repoUrl}{scanResult.scanPath ? ` / ${scanResult.scanPath}` : ''}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-6 text-center">
                    {[
                      { label: 'Folders',  value: scanResult.folderResults?.length ?? 0, color: 'text-gov-text' },
                      { label: 'Files',    value: scanResult.totalFilesScanned,           color: 'text-gov-text' },
                      { label: 'Findings', value: scanResult.totalFindings,               color: scanResult.totalFindings > 0 ? 'text-red-400' : 'text-green-400' },
                      {
                        label: 'Passed',
                        value: `${(scanResult.folderResults ?? []).filter(f => f.status === 'pass').length}/${scanResult.folderResults?.length ?? 0}`,
                        color: (scanResult.folderResults ?? []).every(f => f.status === 'pass') ? 'text-green-400'
                             : (scanResult.folderResults ?? []).some(f => f.status === 'fail')  ? 'text-red-400'
                             : 'text-yellow-400',
                      },
                    ].map(stat => (
                      <div key={stat.label}>
                        <div className={`text-2xl font-black ${stat.color}`}>{stat.value}</div>
                        <div className="text-[10px] text-gov-text-3 uppercase font-semibold">{stat.label}</div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Global SKL-SEC check legend */}
                <div className="grid grid-cols-3 md:grid-cols-6 gap-2">
                  {Object.entries(CHECK_META).map(([id, meta]) => {
                    const anyFail = (scanResult.folderResults ?? []).some(fr => fr.securityChecks.find(c => c.id === id && !c.passed));
                    return (
                      <div key={id} className={`rounded-lg border px-2 py-2.5 text-center ${anyFail ? 'border-red-500/30 bg-red-500/5' : 'border-green-500/20 bg-green-500/5'}`}>
                        <div className="flex justify-center mb-1">
                          {anyFail ? <ShieldAlert className="w-4 h-4 text-red-400" /> : <ShieldCheck className="w-4 h-4 text-green-400" />}
                        </div>
                        <div className={`text-[10px] font-bold font-mono ${anyFail ? 'text-red-400' : 'text-green-400'}`}>{id}</div>
                        <div className="text-[9px] text-gov-text-3 mt-0.5 leading-tight">{meta.label}</div>
                      </div>
                    );
                  })}
                </div>
              </>
            )}
          </div>

          {/* Per-folder results */}
          {(scanResult.folderResults ?? []).length > 0 && (
            <div className="space-y-3">
              <h3 className="text-sm font-bold text-gov-text flex items-center gap-2">
                <FolderOpen className="w-4 h-4 text-cyan-400" />
                Folder Results ({scanResult.folderResults!.length})
              </h3>

              {scanResult.folderResults!.map(fr => {
                const isExp = expandedFolders.has(fr.folderPath);
                const passedChecks = fr.securityChecks.filter(c => c.passed).length;
                const borderCls = fr.status === 'pass' ? 'border-green-500/30 bg-green-500/5'
                                : fr.status === 'warning' ? 'border-yellow-500/30 bg-yellow-500/5'
                                : 'border-red-500/30 bg-red-500/5';
                const scoreColor = fr.score >= 70 ? '#22c55e' : fr.score >= 40 ? '#eab308' : '#ef4444';

                return (
                  <div key={fr.folderPath} className={`rounded-xl border overflow-hidden ${borderCls}`}>
                    {/* Row header */}
                    <button onClick={() => toggleFolder(fr.folderPath)} className="w-full px-4 py-3 flex items-center gap-3 hover:bg-gov-bg/40 transition-colors text-left">
                      <div className="shrink-0">
                        {fr.status === 'pass' ? <ShieldCheck className="w-5 h-5 text-green-400" />
                          : fr.status === 'warning' ? <ShieldAlert className="w-5 h-5 text-yellow-400" />
                          : <ShieldAlert className="w-5 h-5 text-red-400" />}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <code className="text-sm font-bold text-gov-text font-mono truncate">{fr.folderPath || '/ (root)'}</code>
                          <span className={`text-[10px] font-bold uppercase px-1.5 py-0.5 rounded-full ${
                            fr.status === 'pass' ? 'bg-green-500/15 text-green-400'
                            : fr.status === 'warning' ? 'bg-yellow-500/15 text-yellow-400'
                            : 'bg-red-500/15 text-red-400'}`}>{fr.status}</span>
                        </div>
                        <div className="flex items-center gap-4 text-xs text-gov-text-3 mt-0.5 flex-wrap">
                          <span><FileText className="inline w-3 h-3 mr-1" />{fr.filesScanned} file{fr.filesScanned !== 1 ? 's' : ''}</span>
                          <span className={fr.findings.length > 0 ? 'text-red-400' : 'text-green-400'}>
                            <AlertTriangle className="inline w-3 h-3 mr-1" />{fr.findings.length} finding{fr.findings.length !== 1 ? 's' : ''}
                          </span>
                          <span className={passedChecks === fr.securityChecks.length ? 'text-green-400' : 'text-yellow-400'}>
                            <ShieldCheck className="inline w-3 h-3 mr-1" />{passedChecks}/{fr.securityChecks.length} checks passed
                          </span>
                        </div>
                      </div>
                      <div className="flex items-center gap-3 shrink-0">
                        <div className="w-10 h-10 rounded-full flex items-center justify-center border-2 font-black text-sm tabular-nums"
                          style={{ borderColor: scoreColor, color: scoreColor, backgroundColor: `${scoreColor}15` }}>
                          {fr.score}
                        </div>
                        {isExp ? <ChevronUp className="w-4 h-4 text-gov-text-3" /> : <ChevronDown className="w-4 h-4 text-gov-text-3" />}
                      </div>
                    </button>

                    {/* Detail panel */}
                    {isExp && (
                      <div className="border-t border-inherit px-4 pb-4 pt-3 space-y-5">

                        {/* Security checks grid */}
                        <div>
                          <div className="text-xs font-bold text-gov-text-2 uppercase tracking-wider mb-2">Security Checks (SKL-SEC-001 → SKL-SEC-014)</div>
                          <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                            {fr.securityChecks.map(check => {
                              const meta = CHECK_META[check.id];
                              return (
                                <div key={check.id} className={`rounded-lg border p-3 flex items-start gap-2.5 ${check.passed ? 'border-green-500/20 bg-green-500/5' : 'border-red-500/30 bg-red-500/5'}`}>
                                  <div className="shrink-0 mt-0.5">
                                    {check.passed ? <CheckCircle2 className="w-4 h-4 text-green-400" /> : <XCircle className="w-4 h-4 text-red-400" />}
                                  </div>
                                  <div className="min-w-0">
                                    <div className="flex items-center gap-1.5 flex-wrap">
                                      <span className="text-xs font-bold font-mono text-gov-text-2">{check.id}</span>
                                      {!check.passed && check.findingCount > 0 && (
                                        <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/20 text-red-400 font-bold">{check.findingCount}</span>
                                      )}
                                    </div>
                                    <div className={`text-xs font-semibold mt-0.5 ${check.passed ? 'text-green-400' : 'text-red-400'}`}>
                                      {meta?.label ?? check.name}
                                    </div>
                                    <div className="text-[10px] text-gov-text-3 mt-0.5 leading-tight">{meta?.description ?? check.description}</div>
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        </div>

                        {/* Findings list */}
                        {fr.findings.length > 0 ? (
                          <div>
                            <div className="text-xs font-bold text-gov-text-2 uppercase tracking-wider mb-2">Findings ({fr.findings.length})</div>
                            <div className="space-y-2">
                              {fr.findings.map((f, i) => (
                                <div key={i} className={`rounded-lg border p-3 ${SEVERITY_BG[f.severity] ?? 'bg-gov-bg border-gov-border'}`}>
                                  <div className="flex items-start justify-between gap-2 flex-wrap">
                                    <div className="flex items-center gap-2 flex-wrap">
                                      <span className="text-[10px] font-bold font-mono text-gov-text-3">{f.checkID}</span>
                                      <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full border ${SEVERITY_BG[f.severity]} ${SEVERITY_COLOR[f.severity]}`}>{f.severity}</span>
                                      <span className="text-xs text-gov-text-3">{f.category}</span>
                                    </div>
                                    {f.filePath && <code className="text-[10px] text-gov-text-3 font-mono">{f.filePath}{f.line ? `:${f.line}` : ''}</code>}
                                  </div>
                                  <div className="text-sm font-semibold text-gov-text mt-1">{f.title}</div>
                                  {f.matchedPattern && (
                                    <div className="mt-1 text-xs text-gov-text-3">
                                      Matched: <code className="font-mono text-yellow-400">&quot;{f.matchedPattern}&quot;</code>
                                    </div>
                                  )}
                                  <div className="mt-2 text-xs text-gov-text-3 flex items-start gap-1.5">
                                    <Info className="w-3 h-3 shrink-0 mt-0.5 text-cyan-400" />
                                    <span>{f.remediation}</span>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        ) : (
                          <div className="flex items-center gap-2 text-sm text-green-400 bg-green-500/5 border border-green-500/20 rounded-lg p-3">
                            <CheckCircle2 className="w-4 h-4 shrink-0" />
                            No security issues found in this folder.
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {/* Empty state */}
          {scanResult.status === 'success' && (scanResult.folderResults ?? []).length === 0 && (
            <div className="bg-gov-surface rounded-2xl border border-gov-border p-8 text-center">
              <FileText className="w-10 h-10 text-gov-text-3 mx-auto mb-3 opacity-40" />
              <div className="font-semibold text-gov-text mb-1">No skill files found</div>
              <div className="text-sm text-gov-text-3">
                No <code className="font-mono text-cyan-400">skills.md</code> files were found in{' '}
                <code className="font-mono text-gov-text">{scanResult.scanPath || 'the repository root'}</code>.
                Try specifying a different folder path.
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
