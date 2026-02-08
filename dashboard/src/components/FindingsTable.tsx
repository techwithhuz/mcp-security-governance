'use client';

import { AlertTriangle, AlertCircle, Info, CheckCircle, ChevronDown, ChevronUp } from 'lucide-react';
import { useState } from 'react';
import { Finding } from '@/lib/types';

interface FindingsTableProps {
  findings: Finding[];
}

const severityConfig = {
  Critical: { icon: AlertCircle, color: '#ef4444', bg: 'bg-red-500/10', border: 'border-red-500/30', text: 'text-red-400' },
  High: { icon: AlertTriangle, color: '#f97316', bg: 'bg-orange-500/10', border: 'border-orange-500/30', text: 'text-orange-400' },
  Medium: { icon: Info, color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30', text: 'text-yellow-400' },
  Low: { icon: CheckCircle, color: '#22c55e', bg: 'bg-green-500/10', border: 'border-green-500/30', text: 'text-green-400' },
};

export default function FindingsTable({ findings }: FindingsTableProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [filterSeverity, setFilterSeverity] = useState<string>('All');
  const [filterCategory, setFilterCategory] = useState<string>('All');

  const categories = ['All', ...Array.from(new Set(findings.map(f => f.category)))];
  const severities = ['All', 'Critical', 'High', 'Medium', 'Low'];

  const filtered = findings.filter(f => {
    if (filterSeverity !== 'All' && f.severity !== filterSeverity) return false;
    if (filterCategory !== 'All' && f.category !== filterCategory) return false;
    return true;
  });

  const severityCounts = {
    Critical: findings.filter(f => f.severity === 'Critical').length,
    High: findings.filter(f => f.severity === 'High').length,
    Medium: findings.filter(f => f.severity === 'Medium').length,
    Low: findings.filter(f => f.severity === 'Low').length,
  };

  return (
    <div className="bg-gov-surface rounded-2xl border border-gov-border">
      {/* Header with severity pills */}
      <div className="p-5 border-b border-gov-border">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold">Security Findings</h2>
          <span className="text-sm text-gov-text-3">{filtered.length} of {findings.length} findings</span>
        </div>

        {/* Severity summary pills */}
        <div className="flex gap-2 mb-4">
          {Object.entries(severityCounts).map(([sev, count]) => {
            const config = severityConfig[sev as keyof typeof severityConfig];
            return (
              <button
                key={sev}
                onClick={() => setFilterSeverity(filterSeverity === sev ? 'All' : sev)}
                className={`severity-badge border transition-all ${
                  filterSeverity === sev ? `${config.bg} ${config.border} ${config.text}` : 'border-gov-border text-gov-text-3 hover:border-gov-border-light'
                }`}
              >
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: config.color }} />
                {count} {sev}
              </button>
            );
          })}
        </div>

        {/* Category filter */}
        <div className="flex gap-2 flex-wrap">
          {categories.map(cat => (
            <button
              key={cat}
              onClick={() => setFilterCategory(cat)}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition-all ${
                filterCategory === cat
                  ? 'bg-gov-accent/20 text-blue-400 border border-blue-500/30'
                  : 'text-gov-text-3 hover:text-gov-text-2 border border-transparent hover:border-gov-border'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>
      </div>

      {/* Findings list */}
      <div className="divide-y divide-gov-border max-h-[600px] overflow-y-auto">
        {filtered.map(finding => {
          const config = severityConfig[finding.severity];
          const Icon = config.icon;
          const isExpanded = expandedId === finding.id;

          return (
            <div key={finding.id} className="transition-colors hover:bg-gov-surface-2/50">
              <button
                onClick={() => setExpandedId(isExpanded ? null : finding.id)}
                className="w-full p-4 text-left"
              >
                <div className="flex items-start gap-3">
                  <div className={`p-1.5 rounded-lg ${config.bg} mt-0.5`}>
                    <Icon size={16} style={{ color: config.color }} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-mono text-xs text-gov-text-3">{finding.id}</span>
                      <span className={`severity-badge ${config.bg} border ${config.border} ${config.text}`}>
                        {finding.severity}
                      </span>
                      <span className="px-2 py-0.5 rounded text-xs bg-gov-bg text-gov-text-3">
                        {finding.category}
                      </span>
                    </div>
                    <h3 className="font-semibold text-sm text-gov-text">{finding.title}</h3>
                    <p className="text-xs text-gov-text-3 mt-0.5 truncate">{finding.resourceRef || finding.resource || 'N/A'}</p>
                  </div>
                  <div className="text-gov-text-3">
                    {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                  </div>
                </div>
              </button>

              {isExpanded && (
                <div className="px-4 pb-4 ml-10 space-y-3">
                  <div className="bg-gov-bg rounded-xl p-4 space-y-3 text-sm">
                    <div>
                      <span className="text-gov-text-3 text-xs uppercase tracking-wider font-semibold">Description</span>
                      <p className="text-gov-text-2 mt-1">{finding.description}</p>
                    </div>
                    <div>
                      <span className="text-gov-text-3 text-xs uppercase tracking-wider font-semibold">Impact</span>
                      <p className="text-gov-text-2 mt-1">{finding.impact}</p>
                    </div>
                    <div>
                      <span className="text-gov-text-3 text-xs uppercase tracking-wider font-semibold">Remediation</span>
                      <p className="text-green-400/80 mt-1 font-mono text-xs leading-relaxed">{finding.remediation}</p>
                    </div>
                    <div className="flex gap-4 pt-2 border-t border-gov-border">
                      <div>
                        <span className="text-gov-text-3 text-xs">Namespace</span>
                        <p className="text-gov-text font-mono text-xs">{finding.namespace}</p>
                      </div>
                      <div>
                        <span className="text-gov-text-3 text-xs">Resource</span>
                        <p className="text-gov-text font-mono text-xs">{finding.resourceRef || finding.resource || 'N/A'}</p>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          );
        })}

        {filtered.length === 0 && (
          <div className="p-12 text-center">
            <CheckCircle className="mx-auto text-green-400 mb-3" size={32} />
            <p className="text-gov-text-2 font-medium">No findings match your filters</p>
            <p className="text-gov-text-3 text-sm mt-1">Try adjusting your filter criteria</p>
          </div>
        )}
      </div>
    </div>
  );
}
