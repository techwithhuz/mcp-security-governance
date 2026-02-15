'use client';

import { useState } from 'react';
import {
  Server, Shield, ShieldAlert, ShieldCheck, ShieldX,
  ChevronRight, Route, Network, Plug
} from 'lucide-react';
import type { MCPServerView, MCPServerSummary } from '@/lib/types';

interface MCPServerListProps {
  servers: MCPServerView[];
  summary: MCPServerSummary;
  onSelectServer: (server: MCPServerView) => void;
}

const statusConfig: Record<string, { color: string; bg: string; border: string; icon: typeof Shield; label: string }> = {
  compliant: { color: '#22c55e', bg: 'bg-green-500/10', border: 'border-green-500/30', icon: ShieldCheck, label: 'Compliant' },
  warning: { color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30', icon: Shield, label: 'Warning' },
  failing: { color: '#f97316', bg: 'bg-orange-500/10', border: 'border-orange-500/30', icon: ShieldAlert, label: 'Failing' },
  critical: { color: '#ef4444', bg: 'bg-red-500/10', border: 'border-red-500/30', icon: ShieldX, label: 'Critical' },
};

const sourceLabels: Record<string, { label: string; icon: typeof Plug }> = {
  KagentMCPServer: { label: 'Kagent MCPServer', icon: Plug },
  KagentRemoteMCPServer: { label: 'Kagent Remote', icon: Network },
  AgentgatewayBackendTarget: { label: 'AGW Backend', icon: Route },
  Service: { label: 'K8s Service', icon: Server },
};

export default function MCPServerList({ servers, summary, onSelectServer }: MCPServerListProps) {
  const [filter, setFilter] = useState<'all' | 'critical' | 'warning' | 'compliant'>('all');

  const filtered = servers.filter(s => {
    if (filter === 'all') return true;
    if (filter === 'critical') return s.status === 'critical' || s.status === 'failing';
    if (filter === 'warning') return s.status === 'warning';
    if (filter === 'compliant') return s.status === 'compliant';
    return true;
  });

  // Sort: critical first, then failing, then warning, then compliant
  const statusOrder: Record<string, number> = { critical: 0, failing: 1, warning: 2, compliant: 3 };
  const sorted = [...filtered].sort((a, b) => (statusOrder[a.status] ?? 4) - (statusOrder[b.status] ?? 4));

  const criticalCount = servers.filter(s => s.status === 'critical' || s.status === 'failing').length;
  const warningCount = servers.filter(s => s.status === 'warning').length;
  const compliantCount = servers.filter(s => s.status === 'compliant').length;

  const avgColor = summary.averageScore >= 70 ? '#22c55e' : summary.averageScore >= 50 ? '#eab308' : '#ef4444';

  return (
    <div className="space-y-6">
      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
          <div className="flex items-center gap-2 text-gov-text-3 mb-2">
            <Server className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wider">MCP Servers</span>
          </div>
          <div className="text-4xl font-black">{summary.totalMCPServers}</div>
          <div className="text-xs text-gov-text-3 mt-1">discovered in cluster</div>
        </div>

        <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
          <div className="flex items-center gap-2 text-gov-text-3 mb-2">
            <Route className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wider">Routed via Gateway</span>
          </div>
          <div className="text-4xl font-black">
            <span className="text-green-400">{summary.routedServers}</span>
            <span className="text-lg text-gov-text-3 font-normal">/{summary.totalMCPServers}</span>
          </div>
          <div className="text-xs mt-1">
            {summary.unroutedServers > 0 ? (
              <span className="text-red-400">{summary.unroutedServers} exposed directly</span>
            ) : (
              <span className="text-green-400">All routed through gateway</span>
            )}
          </div>
        </div>

        <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
          <div className="flex items-center gap-2 text-gov-text-3 mb-2">
            <ShieldAlert className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wider">At Risk</span>
          </div>
          <div className="text-4xl font-black">
            <span className="text-red-400">{summary.atRiskServers}</span>
            <span className="text-lg text-gov-text-3 font-normal">/{summary.totalMCPServers}</span>
          </div>
          <div className="text-xs text-gov-text-3 mt-1">
            {summary.criticalServers > 0 && <span className="text-red-400">{summary.criticalServers} critical</span>}
            {summary.criticalServers === 0 && <span className="text-green-400">No critical servers</span>}
          </div>
        </div>

        <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
          <div className="flex items-center gap-2 text-gov-text-3 mb-2">
            <Shield className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wider">Average Score</span>
          </div>
          <div className="text-4xl font-black" style={{ color: avgColor }}>
            {summary.averageScore}
            <span className="text-lg text-gov-text-3 font-normal">/100</span>
          </div>
          <div className="text-xs text-gov-text-3 mt-1">{summary.totalTools} total tools</div>
        </div>
      </div>

      {/* Filter tabs */}
      <div className="flex gap-2">
        {([
          { id: 'all' as const, label: `All (${servers.length})` },
          { id: 'critical' as const, label: `Critical (${criticalCount})` },
          { id: 'warning' as const, label: `Warning (${warningCount})` },
          { id: 'compliant' as const, label: `Compliant (${compliantCount})` },
        ]).map(f => (
          <button
            key={f.id}
            onClick={() => setFilter(f.id)}
            className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
              filter === f.id
                ? 'bg-gov-accent/15 text-blue-400 border border-blue-500/30'
                : 'bg-gov-surface text-gov-text-3 hover:text-gov-text-2 hover:bg-gov-surface/80 border border-gov-border'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Server list */}
      <div className="space-y-3">
        {sorted.map(server => {
          const config = statusConfig[server.status] || statusConfig.critical;
          const StatusIcon = config.icon;
          const sourceInfo = sourceLabels[server.source] || { label: server.source, icon: Server };
          const criticalFindings = server.findings.filter(f => f.severity === 'Critical').length;
          const highFindings = server.findings.filter(f => f.severity === 'High').length;
          const mediumFindings = server.findings.filter(f => f.severity === 'Medium').length;

          return (
            <button
              key={server.id}
              onClick={() => onSelectServer(server)}
              className={`w-full text-left bg-gov-surface rounded-2xl border ${config.border} hover:border-gov-border-light transition-all card-hover p-5`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className="p-2.5 rounded-xl" style={{ backgroundColor: `${config.color}15` }}>
                    <StatusIcon size={22} style={{ color: config.color }} />
                  </div>
                  <div>
                    <div className="text-base font-bold text-gov-text flex items-center gap-2">
                      {server.name}
                      <span className="text-xs font-mono px-1.5 py-0.5 bg-gov-bg rounded text-gov-text-3">{sourceInfo.label}</span>
                    </div>
                    <div className="text-xs text-gov-text-3 mt-0.5 flex items-center gap-2">
                      <span>{server.namespace}</span>
                      {server.transport && (
                        <>
                          <span className="text-gov-border-light">•</span>
                          <span>{server.transport}</span>
                        </>
                      )}
                      {server.toolCount > 0 && (
                        <>
                          <span className="text-gov-border-light">•</span>
                          <span>{server.effectiveToolCount}/{server.toolCount} tools</span>
                        </>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-5">
                  {/* Security badges */}
                  <div className="hidden md:flex gap-1.5">
                    {server.routedThroughGateway ? (
                      <Badge label="Routed" color="#22c55e" />
                    ) : (
                      <Badge label="Exposed" color="#ef4444" />
                    )}
                    {server.hasJWT && <Badge label="JWT" color="#3b82f6" />}
                    {server.hasTLS && <Badge label="TLS" color="#8b5cf6" />}
                    {server.hasRBAC && <Badge label="RBAC" color="#6366f1" />}
                    {server.hasCORS && <Badge label="CORS" color="#a855f7" />}
                    {server.hasRateLimit && <Badge label="RL" color="#ec4899" />}
                    {server.hasPromptGuard && <Badge label="PG" color="#f97316" />}
                  </div>

                  {/* Score */}
                  <div className="text-right min-w-[60px]">
                    <div
                      className="text-2xl font-black tabular-nums"
                      style={{ color: config.color }}
                    >
                      {server.score}
                    </div>
                    <div className="text-[10px] text-gov-text-3 font-semibold">Grade {server.grade}</div>
                  </div>

                  <ChevronRight size={18} className="text-gov-text-3" />
                </div>
              </div>

              {/* Findings summary */}
              {server.findings.length > 0 && (
                <div className="mt-3 pt-3 border-t border-gov-border flex gap-3 text-xs">
                  {criticalFindings > 0 && (
                    <span className="text-red-400 font-semibold">{criticalFindings} Critical</span>
                  )}
                  {highFindings > 0 && (
                    <span className="text-orange-400 font-semibold">{highFindings} High</span>
                  )}
                  {mediumFindings > 0 && (
                    <span className="text-yellow-400 font-semibold">{mediumFindings} Medium</span>
                  )}
                  <span className="text-gov-text-3 ml-auto">{server.findings.length} total findings</span>
                </div>
              )}
            </button>
          );
        })}

        {sorted.length === 0 && (
          <div className="text-center py-12 text-gov-text-3">
            <Server size={32} className="mx-auto mb-3 opacity-50" />
            <p className="text-sm">No MCP servers match the current filter.</p>
          </div>
        )}
      </div>
    </div>
  );
}

function Badge({ label, color }: { label: string; color: string }) {
  return (
    <span
      className="px-2 py-0.5 rounded-md text-[10px] font-bold uppercase tracking-wider"
      style={{ backgroundColor: `${color}15`, color, border: `1px solid ${color}30` }}
    >
      {label}
    </span>
  );
}
