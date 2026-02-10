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

interface ScoreCategory {
  category: string;
  score: number;
  weight: number;
  weighted: number;
  status: string;
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

  // Count findings by severity
  const severityCounts = { Critical: 0, High: 0, Medium: 0, Low: 0 };
  findings.forEach(f => {
    if (f.severity in severityCounts) {
      severityCounts[f.severity as keyof typeof severityCounts]++;
    }
  });

  // Calculate the penalty breakdown
  const penaltyBreakdown = Object.entries(severityCounts)
    .filter(([, count]) => count > 0)
    .map(([severity, count]) => ({
      severity,
      count,
      penaltyEach: severityPenalties[severity as keyof SeverityPenalties],
      totalPenalty: count * severityPenalties[severity as keyof SeverityPenalties],
    }));

  const totalPenalty = penaltyBreakdown.reduce((sum, p) => sum + p.totalPenalty, 0);
  const hasInfraAbsence = category.score === 0 && findings.length > 0;

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
          {/* Score Calculation Breakdown */}
          <div className="bg-gov-bg rounded-xl border border-gov-border p-5">
            <div className="flex items-center gap-2 mb-4">
              <TrendingDown size={16} className="text-blue-400" />
              <h3 className="text-sm font-bold uppercase tracking-wider text-gov-text-2">Score Calculation</h3>
            </div>

            {findings.length === 0 ? (
              <div className="flex items-center gap-3 p-4 rounded-xl bg-green-500/10 border border-green-500/30">
                <ShieldCheck size={24} className="text-green-400" />
                <div>
                  <p className="text-sm font-semibold text-green-400">Fully Compliant</p>
                  <p className="text-xs text-gov-text-3 mt-0.5">No findings detected. This category starts at 100 and has no deductions.</p>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                {/* Start score */}
                <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-gov-surface">
                  <span className="text-sm text-gov-text-2">Starting Score</span>
                  <span className="text-sm font-bold text-green-400">100</span>
                </div>

                {/* Penalty breakdown */}
                {hasInfraAbsence ? (
                  <div className="flex items-center gap-3 p-4 rounded-xl bg-red-500/10 border border-red-500/30">
                    <ShieldAlert size={24} className="text-red-400" />
                    <div>
                      <p className="text-sm font-semibold text-red-400">Infrastructure Absent → Score = 0</p>
                      <p className="text-xs text-gov-text-3 mt-0.5">
                        Required infrastructure is completely missing. When core infrastructure is absent,
                        the category score is set directly to 0 (not calculated via penalties).
                      </p>
                    </div>
                  </div>
                ) : (
                  <>
                    {penaltyBreakdown.map(pb => {
                      const sevConfig = severityConfig[pb.severity];
                      const SevIcon = sevConfig.icon;
                      return (
                        <div key={pb.severity} className="flex items-center justify-between px-3 py-2 rounded-lg bg-gov-surface">
                          <div className="flex items-center gap-2">
                            <SevIcon size={14} style={{ color: sevConfig.color }} />
                            <span className="text-sm text-gov-text-2">
                              {pb.count} × {pb.severity} finding{pb.count > 1 ? 's' : ''}
                            </span>
                            <span className="text-xs text-gov-text-3">
                              (−{pb.penaltyEach}pts each)
                            </span>
                          </div>
                          <span className="text-sm font-bold text-red-400">
                            −{pb.totalPenalty}
                          </span>
                        </div>
                      );
                    })}
                  </>
                )}

                {/* Divider */}
                <div className="border-t border-gov-border" />

                {/* Final score */}
                <div className="flex items-center justify-between px-3 py-3 rounded-lg bg-gov-surface border border-gov-border">
                  <span className="text-sm font-bold text-gov-text">Final Category Score</span>
                  <div className="flex items-center gap-2">
                    {!hasInfraAbsence && (
                      <span className="text-xs text-gov-text-3">
                        100 − {totalPenalty} = 
                      </span>
                    )}
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
            )}
          </div>

          {/* Findings List */}
          {findings.length > 0 && (
            <div>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <AlertTriangle size={16} className="text-amber-400" />
                  <h3 className="text-sm font-bold uppercase tracking-wider text-gov-text-2">
                    Findings ({findings.length})
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
                  const penalty = severityPenalties[finding.severity as keyof SeverityPenalties] || 0;

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
                          <span className="text-xs font-bold text-red-400">−{penalty}pts</span>
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
              Click on each finding to see details and remediation steps
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
