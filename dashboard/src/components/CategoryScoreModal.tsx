'use client';

import { useEffect, useRef } from 'react';
import {
  X,
  Info,
  AlertTriangle,
  AlertCircle,
  CheckCircle2,
  XCircle,
  ArrowDown,
  Minus,
  Wrench,
  ChevronDown,
  ChevronRight,
  ShieldAlert,
  ShieldCheck,
  TrendingDown,
  Server,
} from 'lucide-react';
import { useState } from 'react';

interface Finding {
  id: string;
  severity: string;
  category: string;
  title: string;
  description: string;
  resource?: string;
  resourceRef?: string;
  namespace: string;
  impact: string;
  remediation: string;
}

interface ServerContribution {
  name: string;
  score: number;
  grade: string;
}

interface ScoreCategory {
  category: string;
  score: number;
  weight: number;
  weighted: number;
  status: string;
  infraAbsent?: boolean;
  servers?: ServerContribution[];
}

interface SeverityPenalties {
  Critical: number;
  High: number;
  Medium: number;
  Low: number;
}

interface CategoryScoreModalProps {
  category: ScoreCategory;
  findings: Finding[];
  severityPenalties: SeverityPenalties;
  onClose: () => void;
}

const severityConfig: Record<string, { icon: typeof XCircle; color: string; bg: string; border: string; label: string }> = {
  Critical: { icon: XCircle, color: '#ef4444', bg: 'bg-red-500/10', border: 'border-red-500/30', label: 'Critical' },
  High: { icon: AlertCircle, color: '#f97316', bg: 'bg-orange-500/10', border: 'border-orange-500/30', label: 'High' },
  Medium: { icon: AlertTriangle, color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30', label: 'Medium' },
  Low: { icon: Info, color: '#22c55e', bg: 'bg-green-500/10', border: 'border-green-500/30', label: 'Low' },
};

const statusConfig: Record<string, { color: string; label: string }> = {
  passing: { color: '#22c55e', label: 'Passing' },
  warning: { color: '#eab308', label: 'Warning' },
  failing: { color: '#f97316', label: 'Failing' },
  critical: { color: '#ef4444', label: 'Critical' },
};

function scoreColor(score: number): string {
  if (score >= 90) return '#22c55e';
  if (score >= 70) return '#eab308';
  if (score >= 50) return '#f97316';
  return '#ef4444';
}

export default function CategoryScoreModal({ category, findings, severityPenalties, onClose }: CategoryScoreModalProps) {
  const overlayRef = useRef<HTMLDivElement>(null);
  const [expandedFinding, setExpandedFinding] = useState<string | null>(null);

  // Close on Escape
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, [onClose]);

  // Close on overlay click
  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === overlayRef.current) onClose();
  };

  const servers = category.servers || [];
  const hasServers = servers.length > 0;

  // Count findings by severity
  const severityCounts = { Critical: 0, High: 0, Medium: 0, Low: 0 };
  findings.forEach(f => {
    if (f.severity in severityCounts) {
      severityCounts[f.severity as keyof typeof severityCounts]++;
    }
  });

  const config = statusConfig[category.status] || statusConfig.critical;

  return (
    <div
      ref={overlayRef}
      onClick={handleOverlayClick}
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
    >
      <div className="bg-gov-surface border border-gov-border rounded-2xl shadow-2xl max-w-2xl w-full max-h-[85vh] flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-200">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gov-border shrink-0">
          <div className="flex items-center gap-3">
            <div
              className="w-12 h-12 rounded-xl flex items-center justify-center border-2 font-black text-xl tabular-nums"
              style={{
                borderColor: config.color,
                color: config.color,
                backgroundColor: `${config.color}10`,
              }}
            >
              {category.score}
            </div>
            <div>
              <h2 className="text-lg font-bold text-gov-text">{category.category}</h2>
              <div className="flex items-center gap-2 mt-0.5">
                <span
                  className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold uppercase"
                  style={{ backgroundColor: `${config.color}15`, color: config.color }}
                >
                  {config.label}
                </span>
                <span className="text-xs text-gov-text-3">Weight: {category.weight}%</span>
                <span className="text-xs text-gov-text-3">•</span>
                <span className="text-xs text-gov-text-3">Contribution: {category.weighted.toFixed(1)}pts</span>
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-lg hover:bg-gov-bg transition-colors text-gov-text-3 hover:text-gov-text"
          >
            <X size={20} />
          </button>
        </div>

        {/* Scrollable content */}
        <div className="overflow-y-auto p-6 space-y-6">
          {/* Score Calculation Breakdown — MCP Server Average */}
          <div className="bg-gov-bg rounded-xl border border-gov-border p-5">
            <div className="flex items-center gap-2 mb-4">
              <Server size={16} className="text-blue-400" />
              <h3 className="text-sm font-bold uppercase tracking-wider text-gov-text-2">Score Calculation</h3>
            </div>

            {hasServers ? (
              <div className="space-y-3">
                {/* Explanation */}
                <p className="text-xs text-gov-text-3 leading-relaxed">
                  The cluster-level score for <span className="font-semibold text-gov-text-2">{category.category}</span> is the average score across all {servers.length} MCP server{servers.length !== 1 ? 's' : ''}.
                </p>

                {/* Per-server scores */}
                {servers.map((srv) => {
                  const sColor = scoreColor(srv.score);
                  return (
                    <div key={srv.name} className="flex items-center justify-between px-3 py-2.5 rounded-lg bg-gov-surface border border-gov-border/50">
                      <div className="flex items-center gap-2">
                        <Server size={14} className="text-gov-text-3" />
                        <span className="text-sm font-medium text-gov-text">{srv.name}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-bold tabular-nums" style={{ color: sColor }}>
                          {srv.score}
                        </span>
                        <span className="text-xs text-gov-text-3">/100</span>
                        <span
                          className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-bold"
                          style={{ backgroundColor: `${sColor}15`, color: sColor }}
                        >
                          {srv.grade}
                        </span>
                      </div>
                    </div>
                  );
                })}

                {/* Divider */}
                <div className="border-t border-gov-border" />

                {/* Average calculation */}
                <div className="flex items-center justify-between px-3 py-3 rounded-lg bg-gov-surface border border-gov-border">
                  <span className="text-sm font-bold text-gov-text">Cluster Average</span>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gov-text-3">
                      ({servers.map(s => s.score).join(' + ')}) ÷ {servers.length} =
                    </span>
                    <span
                      className="text-lg font-black tabular-nums"
                      style={{ color: config.color }}
                    >
                      {category.score}
                    </span>
                    <span className="text-xs text-gov-text-3">/100</span>
                  </div>
                </div>

                {/* Weighted contribution */}
                <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-blue-500/5 border border-blue-500/20">
                  <span className="text-sm text-gov-text-2">
                    Weighted Contribution to Final Score
                  </span>
                  <div className="flex items-center gap-1">
                    <span className="text-xs text-gov-text-3">
                      {category.score} × {category.weight}% =
                    </span>
                    <span className="text-sm font-bold text-blue-400">
                      {category.weighted.toFixed(1)}pts
                    </span>
                  </div>
                </div>
              </div>
            ) : (
              /* Fallback if no server data available */
              <div className="flex items-center gap-3 p-4 rounded-xl bg-gov-surface border border-gov-border">
                <Info size={24} className="text-gov-text-3" />
                <div>
                  <p className="text-sm font-semibold text-gov-text-2">No MCP Servers Found</p>
                  <p className="text-xs text-gov-text-3 mt-0.5">No MCP servers discovered for scoring.</p>
                </div>
              </div>
            )}
          </div>

          {/* Findings List */}
          {findings.length > 0 && (
            <div>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <AlertTriangle size={16} className="text-amber-400" />
                  <h3 className="text-sm font-bold uppercase tracking-wider text-gov-text-2">
                    Related Findings ({findings.length})
                  </h3>
                </div>
                <div className="flex gap-2">
                  {Object.entries(severityCounts)
                    .filter(([, count]) => count > 0)
                    .map(([severity, count]) => {
                      const sevConfig = severityConfig[severity];
                      return (
                        <span
                          key={severity}
                          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold"
                          style={{ backgroundColor: `${sevConfig.color}15`, color: sevConfig.color }}
                        >
                          {count} {severity}
                        </span>
                      );
                    })}
                </div>
              </div>

              <div className="space-y-2">
                {findings.map((finding) => {
                  const sevConfig = severityConfig[finding.severity] || severityConfig.Medium;
                  const SevIcon = sevConfig.icon;
                  const isExpanded = expandedFinding === finding.id;

                  return (
                    <div
                      key={finding.id}
                      className={`rounded-xl border transition-all ${sevConfig.border} ${sevConfig.bg}`}
                    >
                      {/* Finding header - clickable */}
                      <button
                        onClick={() => setExpandedFinding(isExpanded ? null : finding.id)}
                        className="w-full flex items-center gap-3 px-4 py-3 text-left"
                      >
                        <SevIcon size={16} style={{ color: sevConfig.color }} className="shrink-0" />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-mono text-gov-text-3">{finding.id}</span>
                            <span className="text-sm font-semibold text-gov-text truncate">{finding.title}</span>
                          </div>
                          {finding.resourceRef && (
                            <span className="text-xs text-gov-text-3 font-mono">{finding.resourceRef}</span>
                          )}
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                          <span
                            className="text-xs font-semibold px-1.5 py-0.5 rounded"
                            style={{ backgroundColor: `${sevConfig.color}15`, color: sevConfig.color }}
                          >
                            {finding.severity}
                          </span>
                          {isExpanded ? <ChevronDown size={14} className="text-gov-text-3" /> : <ChevronRight size={14} className="text-gov-text-3" />}
                        </div>
                      </button>

                      {/* Expanded details */}
                      {isExpanded && (
                        <div className="px-4 pb-4 space-y-3 border-t border-gov-border/50 pt-3">
                          <div>
                            <p className="text-xs font-semibold text-gov-text-3 uppercase tracking-wider mb-1">Description</p>
                            <p className="text-sm text-gov-text-2 leading-relaxed">{finding.description}</p>
                          </div>

                          {finding.impact && (
                            <div>
                              <p className="text-xs font-semibold text-gov-text-3 uppercase tracking-wider mb-1">Impact</p>
                              <p className="text-sm text-gov-text-2 leading-relaxed">{finding.impact}</p>
                            </div>
                          )}

                          {finding.remediation && (
                            <div className="bg-gov-bg rounded-lg border border-gov-border p-3">
                              <div className="flex items-center gap-1.5 mb-1.5">
                                <Wrench size={12} className="text-blue-400" />
                                <p className="text-xs font-semibold text-blue-400 uppercase tracking-wider">How to Fix</p>
                              </div>
                              <p className="text-sm text-gov-text-2 leading-relaxed font-mono">{finding.remediation}</p>
                            </div>
                          )}

                          {finding.namespace && (
                            <div className="flex items-center gap-2 text-xs text-gov-text-3">
                              <span>Namespace:</span>
                              <span className="font-mono px-1.5 py-0.5 bg-gov-surface rounded">{finding.namespace}</span>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-gov-border shrink-0 bg-gov-bg/50">
          <div className="flex items-center justify-between">
            <p className="text-xs text-gov-text-3">
              {findings.length > 0 ? 'Click on each finding to see details and remediation steps' : 'Click on individual MCP servers in the MCP Servers tab for detailed score explanations'}
            </p>
            <button
              onClick={onClose}
              className="px-4 py-2 bg-gov-accent hover:bg-blue-600 text-white rounded-lg text-sm font-medium transition-all"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
