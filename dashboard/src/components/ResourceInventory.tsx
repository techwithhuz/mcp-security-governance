'use client';

import { useState } from 'react';
import {
  Shield, Server, Route, Bot, Plug, Globe, ChevronDown, ChevronUp,
  CheckCircle2, AlertTriangle, Box, Layers
} from 'lucide-react';
import { ResourceDetail } from '@/lib/types';

interface ResourceInventoryProps {
  resources: ResourceDetail[];
}

const kindConfig: Record<string, { icon: typeof Shield; color: string }> = {
  Gateway: { icon: Shield, color: '#3b82f6' },
  AgentgatewayBackend: { icon: Server, color: '#6366f1' },
  AgentgatewayPolicy: { icon: Shield, color: '#8b5cf6' },
  HTTPRoute: { icon: Route, color: '#a855f7' },
  Agent: { icon: Bot, color: '#ec4899' },
  MCPServer: { icon: Plug, color: '#f43f5e' },
  RemoteMCPServer: { icon: Globe, color: '#f97316' },
  Cluster: { icon: Layers, color: '#64748b' },
};

const statusConfig: Record<string, { label: string; color: string; bg: string; border: string; glow: string }> = {
  compliant: { label: 'Compliant', color: '#22c55e', bg: 'bg-green-500/10', border: 'border-green-500/30', glow: 'glow-low' },
  warning: { label: 'Warning', color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30', glow: 'glow-medium' },
  failing: { label: 'Failing', color: '#f97316', bg: 'bg-orange-500/10', border: 'border-orange-500/30', glow: 'glow-high' },
  critical: { label: 'Critical', color: '#ef4444', bg: 'bg-red-500/10', border: 'border-red-500/30', glow: 'glow-critical' },
  info: { label: 'Info', color: '#3b82f6', bg: 'bg-blue-500/10', border: 'border-blue-500/30', glow: 'glow-accent' },
};

const severityConfig: Record<string, { color: string; bg: string }> = {
  Critical: { color: '#ef4444', bg: 'bg-red-500/10' },
  High: { color: '#f97316', bg: 'bg-orange-500/10' },
  Medium: { color: '#eab308', bg: 'bg-yellow-500/10' },
  Low: { color: '#22c55e', bg: 'bg-green-500/10' },
};

const getScoreColor = (score: number) => {
  if (score >= 90) return '#22c55e';
  if (score >= 70) return '#eab308';
  if (score >= 50) return '#f97316';
  return '#ef4444';
};

type FilterKind = 'All' | string;
type FilterStatus = 'All' | string;

export default function ResourceInventory({ resources }: ResourceInventoryProps) {
  const [expandedRef, setExpandedRef] = useState<string | null>(null);
  const [filterKind, setFilterKind] = useState<FilterKind>('All');
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('All');

  // Get unique kinds
  const kinds = ['All', ...Array.from(new Set(resources.map(r => r.kind)))];
  const statuses = ['All', 'critical', 'failing', 'warning', 'compliant'];

  const filtered = resources.filter(r => {
    if (filterKind !== 'All' && r.kind !== filterKind) return false;
    if (filterStatus !== 'All' && r.status !== filterStatus) return false;
    return true;
  });

  // Sort: critical first, then failing, then warning, then compliant
  const statusOrder: Record<string, number> = { critical: 0, failing: 1, warning: 2, info: 3, compliant: 4 };
  const sorted = [...filtered].sort((a, b) => (statusOrder[a.status] ?? 5) - (statusOrder[b.status] ?? 5));

  // Summary counts
  const compliantCount = resources.filter(r => r.status === 'compliant').length;
  const failingCount = resources.filter(r => r.status === 'failing' || r.status === 'critical').length;
  const warningCount = resources.filter(r => r.status === 'warning').length;

  return (
    <div className="bg-gov-surface rounded-2xl border border-gov-border">
      {/* Header */}
      <div className="p-5 border-b border-gov-border">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-bold">Resource Inventory</h2>
            <p className="text-xs text-gov-text-3 mt-0.5">Per-resource security posture — click any resource to see its findings</p>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-green-500/10 border border-green-500/20">
              <CheckCircle2 size={14} className="text-green-400" />
              <span className="text-xs font-semibold text-green-400">{compliantCount} OK</span>
            </div>
            <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-orange-500/10 border border-orange-500/20">
              <AlertTriangle size={14} className="text-orange-400" />
              <span className="text-xs font-semibold text-orange-400">{warningCount + failingCount} Issues</span>
            </div>
          </div>
        </div>

        {/* Filters */}
        <div className="flex gap-4">
          {/* Kind filter */}
          <div className="flex gap-2 flex-wrap">
            <span className="text-xs text-gov-text-3 font-semibold uppercase tracking-wider self-center mr-1">Type:</span>
            {kinds.map(kind => {
              const kc = kindConfig[kind];
              return (
                <button
                  key={kind}
                  onClick={() => setFilterKind(kind)}
                  className={`px-3 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 ${
                    filterKind === kind
                      ? 'bg-gov-accent/20 text-blue-400 border border-blue-500/30'
                      : 'text-gov-text-3 hover:text-gov-text-2 border border-transparent hover:border-gov-border'
                  }`}
                >
                  {kc && <kc.icon size={12} style={{ color: filterKind === kind ? '#60a5fa' : kc.color }} />}
                  {kind === 'All' ? 'All Types' : kind}
                </button>
              );
            })}
          </div>
        </div>

        {/* Status filter */}
        <div className="flex gap-2 mt-3">
          <span className="text-xs text-gov-text-3 font-semibold uppercase tracking-wider self-center mr-1">Status:</span>
          {statuses.map(status => {
            const sc = statusConfig[status];
            return (
              <button
                key={status}
                onClick={() => setFilterStatus(status)}
                className={`px-3 py-1 rounded-lg text-xs font-medium transition-all ${
                  filterStatus === status
                    ? 'bg-gov-accent/20 text-blue-400 border border-blue-500/30'
                    : 'text-gov-text-3 hover:text-gov-text-2 border border-transparent hover:border-gov-border'
                }`}
              >
                {status === 'All' ? 'All' : sc?.label || status}
              </button>
            );
          })}
        </div>
      </div>

      {/* Resource cards */}
      <div className="divide-y divide-gov-border max-h-[700px] overflow-y-auto">
        {sorted.map(resource => {
          const kc = kindConfig[resource.kind] || { icon: Box, color: '#64748b' };
          const sc = statusConfig[resource.status] || statusConfig.info;
          const Icon = kc.icon;
          const isExpanded = expandedRef === resource.resourceRef;
          const scoreColor = getScoreColor(resource.score);

          return (
            <div key={resource.resourceRef} className="transition-colors hover:bg-gov-surface-2/50">
              {/* Resource summary row */}
              <button
                onClick={() => setExpandedRef(isExpanded ? null : resource.resourceRef)}
                className="w-full p-4 text-left"
              >
                <div className="flex items-center gap-4">
                  {/* Kind icon */}
                  <div className="p-2.5 rounded-xl" style={{ backgroundColor: `${kc.color}15` }}>
                    <Icon size={20} style={{ color: kc.color }} />
                  </div>

                  {/* Resource info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-semibold text-sm text-gov-text">{resource.name}</span>
                      <span className="px-2 py-0.5 rounded text-xs bg-gov-bg text-gov-text-3 font-mono">
                        {resource.kind}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-gov-text-3">
                      <span className="font-mono">{resource.namespace || 'cluster-wide'}</span>
                      {resource.findings.length > 0 && (
                        <span className="flex items-center gap-1">
                          <AlertTriangle size={10} />
                          {resource.findings.length} finding{resource.findings.length !== 1 ? 's' : ''}
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Severity counts */}
                  <div className="flex items-center gap-2">
                    {resource.critical > 0 && (
                      <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-bold bg-red-500/15 text-red-400">
                        {resource.critical}C
                      </span>
                    )}
                    {resource.high > 0 && (
                      <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-bold bg-orange-500/15 text-orange-400">
                        {resource.high}H
                      </span>
                    )}
                    {resource.medium > 0 && (
                      <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-bold bg-yellow-500/15 text-yellow-400">
                        {resource.medium}M
                      </span>
                    )}
                    {resource.low > 0 && (
                      <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-bold bg-green-500/15 text-green-400">
                        {resource.low}L
                      </span>
                    )}
                  </div>

                  {/* Score circle */}
                  <div className="flex flex-col items-center">
                    <div
                      className="w-12 h-12 rounded-full flex items-center justify-center border-2 font-black text-lg tabular-nums"
                      style={{
                        borderColor: scoreColor,
                        color: scoreColor,
                        backgroundColor: `${scoreColor}10`,
                      }}
                    >
                      {resource.score}
                    </div>
                  </div>

                  {/* Status badge */}
                  <div
                    className="px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider"
                    style={{ backgroundColor: `${sc.color}15`, color: sc.color }}
                  >
                    {sc.label}
                  </div>

                  {/* Expand toggle */}
                  <div className="text-gov-text-3">
                    {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                  </div>
                </div>
              </button>

              {/* Expanded findings */}
              {isExpanded && (
                <div className="px-4 pb-4">
                  {resource.findings.length === 0 ? (
                    <div className="ml-14 bg-green-500/5 border border-green-500/20 rounded-xl p-4 flex items-center gap-3">
                      <CheckCircle2 size={20} className="text-green-400" />
                      <div>
                        <p className="text-sm font-semibold text-green-400">All checks passed</p>
                        <p className="text-xs text-gov-text-3 mt-0.5">This resource meets all governance requirements.</p>
                      </div>
                    </div>
                  ) : (
                    <div className="ml-14 space-y-2">
                      {resource.findings.map((finding) => {
                        const sevConfig = severityConfig[finding.severity] || severityConfig.Medium;
                        return (
                          <div
                            key={finding.id}
                            className="bg-gov-bg rounded-xl border border-gov-border p-4"
                          >
                            <div className="flex items-start gap-3">
                              {/* Severity dot */}
                              <div className="mt-1">
                                <div
                                  className="w-3 h-3 rounded-full"
                                  style={{ backgroundColor: sevConfig.color }}
                                />
                              </div>

                              <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2 mb-1">
                                  <span
                                    className="px-2 py-0.5 rounded-full text-xs font-bold uppercase"
                                    style={{ backgroundColor: `${sevConfig.color}15`, color: sevConfig.color }}
                                  >
                                    {finding.severity}
                                  </span>
                                  <span className="px-2 py-0.5 rounded text-xs bg-gov-surface text-gov-text-3">
                                    {finding.category}
                                  </span>
                                  <span className="font-mono text-xs text-gov-text-3">{finding.id}</span>
                                </div>
                                <h4 className="font-semibold text-sm text-gov-text mb-1">{finding.title}</h4>
                                <p className="text-xs text-gov-text-3 mb-2">{finding.description}</p>
                                <div className="flex gap-6 text-xs">
                                  <div>
                                    <span className="text-gov-text-3 font-semibold uppercase tracking-wider">Impact: </span>
                                    <span className="text-gov-text-2">{finding.impact}</span>
                                  </div>
                                </div>
                                <div className="mt-2 bg-green-500/5 border border-green-500/10 rounded-lg p-2.5">
                                  <span className="text-xs text-green-400 font-semibold uppercase tracking-wider">Fix: </span>
                                  <span className="text-xs text-green-400/80 font-mono">{finding.remediation}</span>
                                </div>
                              </div>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}

        {sorted.length === 0 && (
          <div className="p-12 text-center">
            <CheckCircle2 className="mx-auto text-green-400 mb-3" size={32} />
            <p className="text-gov-text-2 font-medium">No resources match your filters</p>
            <p className="text-gov-text-3 text-sm mt-1">Try adjusting your filter criteria</p>
          </div>
        )}
      </div>
    </div>
  );
}
