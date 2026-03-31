'use client';

import { useState } from 'react';
import {
  BookOpen, ChevronDown, ChevronUp, ShieldAlert, ShieldX, ShieldCheck,
  ExternalLink, Search, X, FileCode, AlertTriangle, CheckCircle2,
} from 'lucide-react';
import type { SkillCatalogScore, SkillCatalogFinding, SkillCatalogsResponse } from '@/lib/types';

interface SkillCatalogProps {
  data: SkillCatalogsResponse | null;
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

function CatalogCard({ catalog }: { catalog: SkillCatalogScore }) {
  const [expanded, setExpanded] = useState(false);
  const scoreColor = getScoreColor(catalog.score);
  const status = statusConfig[catalog.status] || statusConfig.warning;
  const StatusIcon = status.icon;
  const findings = catalog.findings || [];
  const criticalCount = findings.filter(f => f.severity === 'Critical').length;
  const highCount = findings.filter(f => f.severity === 'High').length;
  const mediumCount = findings.filter(f => f.severity === 'Medium').length;
  const lowCount = findings.filter(f => f.severity === 'Low').length;

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
          {catalog.score}
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

        {/* Status badge + findings count + expand */}
        <div className="flex items-center gap-2 shrink-0">
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

      {/* Expanded detail */}
      {expanded && (
        <div className="border-t border-gov-border px-4 pb-4 pt-3 space-y-3">
          {/* Stats row */}
          <div className="flex items-center gap-4 text-xs text-gov-text-3">
            {catalog.scannedFiles > 0 && (
              <span className="flex items-center gap-1">
                <FileCode className="w-3.5 h-3.5" />
                {catalog.scannedFiles} file{catalog.scannedFiles !== 1 ? 's' : ''} scanned
              </span>
            )}
            <span className="flex items-center gap-1">
              <AlertTriangle className="w-3.5 h-3.5" />
              {findings.length} finding{findings.length !== 1 ? 's' : ''}
            </span>
          </div>

          {findings.length === 0 ? (
            <div className="flex items-center gap-2 text-sm text-green-400 py-2">
              <CheckCircle2 className="w-4 h-4" />
              No issues found — this skill catalog passes all checks.
            </div>
          ) : (
            <div>
              <p className="text-xs font-semibold text-gov-text-3 uppercase tracking-wider mb-2">Findings</p>
              {findings.map((f, i) => (
                <FindingRow key={`${f.checkID}-${i}`} finding={f} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function SkillCatalog({ data }: SkillCatalogProps) {
  const [filterStatus, setFilterStatus] = useState<'all' | 'pass' | 'warning' | 'fail'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const catalogs = data?.catalogs || [];
  const summary = data?.summary || { total: 0, passCount: 0, warningCount: 0, failCount: 0, averageScore: 0 };

  const filtered = catalogs.filter(c => {
    if (filterStatus !== 'all' && c.status !== filterStatus) return false;
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
        {sorted.map(catalog => (
          <CatalogCard key={`${catalog.namespace}/${catalog.name}`} catalog={catalog} />
        ))}
      </div>
    </div>
  );
}
