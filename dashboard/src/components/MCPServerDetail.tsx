'use client';

import { useState } from 'react';
import {
  ArrowLeft, Shield, Server, Route, FileKey, Blocks, Bot, Network,
  Lock, Zap, ShieldCheck, ShieldX, ShieldAlert, ExternalLink, X, Info, CheckCircle2, AlertTriangle, XCircle, MinusCircle
} from 'lucide-react';
import type { MCPServerView, MCPServerScoreBreakdown, Finding, RelatedResource, ScoreExplanation } from '@/lib/types';

interface MCPServerDetailProps {
  server: MCPServerView;
  onBack: () => void;
}

const categoryLabels: { key: keyof MCPServerScoreBreakdown; label: string; icon: typeof Shield }[] = [
  { key: 'gatewayRouting', label: 'Gateway Routing', icon: Route },
  { key: 'authentication', label: 'Authentication', icon: FileKey },
  { key: 'authorization', label: 'Authorization', icon: Lock },
  { key: 'tls', label: 'TLS Encryption', icon: Shield },
  { key: 'cors', label: 'CORS Policy', icon: Blocks },
  { key: 'rateLimit', label: 'Rate Limiting', icon: Zap },
  { key: 'promptGuard', label: 'Prompt Guard', icon: ShieldCheck },
  { key: 'toolScope', label: 'Tool Scope', icon: Server },
];

const severityColors: Record<string, string> = {
  Critical: '#ef4444',
  High: '#f97316',
  Medium: '#eab308',
  Low: '#22c55e',
};

const getScoreColor = (score: number) => {
  if (score >= 90) return '#22c55e';
  if (score >= 70) return '#eab308';
  if (score >= 50) return '#f97316';
  return '#ef4444';
};

export default function MCPServerDetail({ server, onBack }: MCPServerDetailProps) {
  const bd = server.scoreBreakdown;
  const scoreColor = getScoreColor(server.score);
  const [selectedResource, setSelectedResource] = useState<RelatedResource | null>(null);
  const [selectedControl, setSelectedControl] = useState<ScoreExplanation | null>(null);

  const totalRelated =
    server.relatedGateways.length +
    server.relatedBackends.length +
    server.relatedRoutes.length +
    server.relatedPolicies.length +
    server.relatedAgents.length +
    server.relatedServices.length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-gov-surface rounded-2xl border border-gov-border p-6">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <button
              onClick={onBack}
              className="p-2 rounded-xl bg-gov-bg hover:bg-gov-border/30 transition mt-1"
              title="Back to MCP Servers"
            >
              <ArrowLeft size={20} className="text-gov-text-2" />
            </button>
            <div>
              <h2 className="text-2xl font-black text-gov-text flex items-center gap-3">
                <Server size={24} style={{ color: scoreColor }} />
                {server.name}
              </h2>
              <div className="text-sm text-gov-text-3 mt-1 flex flex-wrap items-center gap-2">
                <span className="font-mono px-2 py-0.5 bg-gov-bg rounded text-xs">{server.namespace}</span>
                <span className="text-gov-border-light">•</span>
                <span>{server.source}</span>
                {server.transport && (
                  <>
                    <span className="text-gov-border-light">•</span>
                    <span>{server.transport}</span>
                  </>
                )}
                {server.url && (
                  <>
                    <span className="text-gov-border-light">•</span>
                    <span className="font-mono text-xs text-gov-text-3">{server.url}</span>
                  </>
                )}
                {(server.port ?? 0) > 0 && (
                  <>
                    <span className="text-gov-border-light">•</span>
                    <span>Port {server.port}</span>
                  </>
                )}
              </div>

              {/* Security badges */}
              <div className="flex flex-wrap gap-1.5 mt-3">
                {server.routedThroughGateway ? (
                  <SecurityBadge label="Routed via Gateway" color="#22c55e" enabled />
                ) : (
                  <SecurityBadge label="Not Routed — Exposed" color="#ef4444" enabled />
                )}
                <SecurityBadge label="JWT Auth" color="#3b82f6" enabled={server.hasJWT} detail={server.jwtMode} />
                <SecurityBadge label="TLS" color="#8b5cf6" enabled={server.hasTLS} />
                <SecurityBadge label="RBAC" color="#6366f1" enabled={server.hasRBAC} />
                <SecurityBadge label="CORS" color="#a855f7" enabled={server.hasCORS} />
                <SecurityBadge label="Rate Limit" color="#ec4899" enabled={server.hasRateLimit} />
                <SecurityBadge label="Prompt Guard" color="#f97316" enabled={server.hasPromptGuard} />
              </div>
            </div>
          </div>

          {/* Score gauge */}
          <div className="text-center ml-6 flex-shrink-0">
            <div
              className="w-24 h-24 rounded-2xl border-2 flex flex-col items-center justify-center"
              style={{ borderColor: scoreColor, backgroundColor: `${scoreColor}08` }}
            >
              <span className="text-3xl font-black tabular-nums" style={{ color: scoreColor }}>
                {server.score}
              </span>
              <span className="text-xs text-gov-text-3 font-semibold">Grade {server.grade}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Security Controls Grid */}
      <div>
        <h3 className="text-sm font-bold text-gov-text-2 mb-3 px-1 uppercase tracking-wider">Security Controls</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {categoryLabels.map(cat => {
            const score = bd[cat.key];
            const color = getScoreColor(score);
            const Icon = cat.icon;
            const explanation = server.scoreExplanations?.find(e => e.category === cat.label);
            return (
              <button
                key={cat.key}
                className="text-left bg-gov-surface rounded-xl border border-gov-border p-4 card-hover group cursor-pointer hover:border-gov-border-light transition-all"
                onClick={() => explanation && setSelectedControl(explanation)}
                title={`Click to see how ${cat.label} score is calculated`}
              >
                <div className="flex items-center gap-2 mb-2">
                  <div className="p-1.5 rounded-lg" style={{ backgroundColor: `${color}15` }}>
                    <Icon size={14} style={{ color }} />
                  </div>
                  <span className="text-xs text-gov-text-3 font-medium flex-1">{cat.label}</span>
                  <Info size={12} className="text-gov-text-3 opacity-0 group-hover:opacity-100 transition-opacity" />
                </div>
                <div className="text-2xl font-black tabular-nums" style={{ color }}>
                  {score}
                </div>
                <div className="mt-2 h-1.5 bg-gov-bg rounded-full overflow-hidden">
                  <div
                    className="h-full rounded-full transition-all duration-700"
                    style={{
                      width: `${score}%`,
                      backgroundColor: color,
                      boxShadow: `0 0 6px ${color}40`,
                    }}
                  />
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Related Resources */}
      <div>
        <h3 className="text-sm font-bold text-gov-text-2 mb-3 px-1 uppercase tracking-wider">
          Related Resources <span className="text-gov-text-3 font-normal normal-case">({totalRelated})</span>
        </h3>

        {totalRelated === 0 && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-4">
            <div className="flex items-start gap-3">
              <ShieldX size={20} className="text-red-400 mt-0.5" />
              <div>
                <div className="text-sm font-semibold text-red-400">No Related Resources Found</div>
                <div className="text-xs text-red-400/70 mt-1">
                  This MCP server has no associated gateway, backend, or policy resources. It is completely ungoverned.
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="space-y-3">
          <ResourceGroup label="Gateways" icon={Route} resources={server.relatedGateways} onSelect={setSelectedResource} />
          <ResourceGroup label="Backends" icon={Server} resources={server.relatedBackends} onSelect={setSelectedResource} />
          <ResourceGroup label="HTTP Routes" icon={Network} resources={server.relatedRoutes} onSelect={setSelectedResource} />
          <ResourceGroup label="Policies" icon={Shield} resources={server.relatedPolicies} onSelect={setSelectedResource} />
          <ResourceGroup label="Agents" icon={Bot} resources={server.relatedAgents} onSelect={setSelectedResource} />
          <ResourceGroup label="Services" icon={Blocks} resources={server.relatedServices} onSelect={setSelectedResource} />
        </div>
      </div>

      {/* Resource Detail Modal */}
      {selectedResource && (
        <ResourceDetailModal resource={selectedResource} onClose={() => setSelectedResource(null)} />
      )}

      {/* Score Explanation Modal */}
      {selectedControl && (
        <ScoreExplanationModal explanation={selectedControl} onClose={() => setSelectedControl(null)} />
      )}

      {/* Tools */}
      {(server.effectiveToolCount > 0 || server.toolNames.length > 0) && (
        <div>
          <h3 className="text-sm font-bold text-gov-text-2 mb-3 px-1 uppercase tracking-wider">
            {server.hasToolRestriction ? 'Allowed Tools' : 'Exposed Tools'}{' '}
            <span className="text-gov-text-3 font-normal normal-case">
              ({server.effectiveToolCount})
              {server.hasToolRestriction && (
                <span className="ml-1 text-green-400">
                  — restricted from {server.toolCount} discovered
                </span>
              )}
            </span>
          </h3>
          <div className="bg-gov-surface rounded-xl border border-gov-border p-4">
            {server.hasToolRestriction && server.effectiveToolNames && server.effectiveToolNames.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {server.effectiveToolNames.map((tool, i) => (
                  <span key={i} className="px-2.5 py-1 bg-green-500/10 rounded-lg text-xs font-mono text-green-400 border border-green-500/20">
                    {tool}
                  </span>
                ))}
              </div>
            ) : (
              <div className="flex flex-wrap gap-2">
                {server.toolNames.map((tool, i) => (
                  <span key={i} className="px-2.5 py-1 bg-gov-bg rounded-lg text-xs font-mono text-gov-text-2 border border-gov-border">
                    {tool}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Findings */}
      {server.findings.length > 0 && (
        <div>
          <h3 className="text-sm font-bold text-gov-text-2 mb-3 px-1 uppercase tracking-wider">
            Findings <span className="text-gov-text-3 font-normal normal-case">({server.findings.length})</span>
          </h3>
          <div className="space-y-2">
            {server.findings
              .sort((a, b) => {
                const order: Record<string, number> = { Critical: 0, High: 1, Medium: 2, Low: 3 };
                return (order[a.severity] ?? 4) - (order[b.severity] ?? 4);
              })
              .map((f, i) => (
                <FindingCard key={`${f.id}-${i}`} finding={f} />
              ))}
          </div>
        </div>
      )}
    </div>
  );
}

function SecurityBadge({ label, color, enabled, detail }: { label: string; color: string; enabled: boolean; detail?: string }) {
  if (!enabled) {
    return (
      <span className="px-2 py-1 rounded-lg text-xs font-medium bg-gov-bg text-gov-text-3 border border-gov-border opacity-60 line-through">
        {label}
      </span>
    );
  }
  return (
    <span
      className="px-2 py-1 rounded-lg text-xs font-semibold"
      style={{ backgroundColor: `${color}15`, color, border: `1px solid ${color}30` }}
    >
      {label}
      {detail && <span className="opacity-70 ml-1">({detail})</span>}
    </span>
  );
}

function ResourceGroup({ label, icon: Icon, resources, onSelect }: {
  label: string;
  icon: typeof Server;
  resources: RelatedResource[];
  onSelect: (r: RelatedResource) => void;
}) {
  if (resources.length === 0) return null;

  const statusColors: Record<string, string> = {
    healthy: '#22c55e',
    warning: '#eab308',
    critical: '#ef4444',
    missing: '#6b7280',
  };

  return (
    <div>
      <div className="flex items-center gap-2 mb-2 px-1">
        <Icon size={14} className="text-gov-text-3" />
        <span className="text-xs font-bold text-gov-text-2 uppercase tracking-wider">{label}</span>
        <span className="text-xs text-gov-text-3">({resources.length})</span>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {resources.map((r, i) => {
          const sColor = statusColors[r.status] || '#6b7280';
          return (
            <button
              key={`${r.kind}-${r.name}-${i}`}
              onClick={() => onSelect(r)}
              className="text-left bg-gov-surface rounded-xl border border-gov-border p-4 hover:border-gov-border-light transition-all card-hover group"
            >
              <div className="flex items-start justify-between mb-2">
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-semibold text-gov-text truncate group-hover:text-blue-400 transition-colors">
                    {r.name}
                  </div>
                  <div className="text-xs text-gov-text-3 mt-0.5 flex items-center gap-1.5">
                    <span className="font-mono">{r.namespace}</span>
                    <span className="text-gov-border-light">•</span>
                    <span>{r.kind}</span>
                  </div>
                </div>
                <span
                  className="px-2 py-0.5 rounded-lg text-[10px] font-bold uppercase tracking-wider flex-shrink-0 ml-2"
                  style={{
                    backgroundColor: `${sColor}15`,
                    color: sColor,
                  }}
                >
                  {r.status}
                </span>
              </div>
              {/* Quick badges */}
              {r.details && (
                <div className="flex gap-1 flex-wrap mt-2">
                  {r.details.hasTLS === true && <DetailBadge label="TLS" color="#8b5cf6" />}
                  {r.details.hasAuth === true && <DetailBadge label="Auth" color="#3b82f6" />}
                  {r.details.hasJWT === true && <DetailBadge label="JWT" color="#3b82f6" />}
                  {r.details.hasRBAC === true && <DetailBadge label="RBAC" color="#6366f1" />}
                  {r.details.hasCORS === true && <DetailBadge label="CORS" color="#a855f7" />}
                  {r.details.hasRateLimit === true && <DetailBadge label="Rate Limit" color="#ec4899" />}
                  {r.details.hasPromptGuard === true && <DetailBadge label="Prompt Guard" color="#f97316" />}
                  {r.details.programmed === true && <DetailBadge label="Programmed" color="#22c55e" />}
                  {r.details.programmed === false && <DetailBadge label="Not Programmed" color="#ef4444" />}
                  {r.details.ready === true && <DetailBadge label="Ready" color="#22c55e" />}
                  {r.details.ready === false && <DetailBadge label="Not Ready" color="#ef4444" />}
                  {!!r.details.backendType && <DetailBadge label={`${String(r.details.backendType)}`} color="#64748b" />}
                </div>
              )}
              <div className="mt-2 flex items-center gap-1 text-[10px] text-gov-text-3 opacity-0 group-hover:opacity-100 transition-opacity">
                <ExternalLink size={10} />
                <span>Click to view details</span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function DetailBadge({ label, color }: { label: string; color: string }) {
  return (
    <span
      className="px-1.5 py-0.5 rounded text-[10px] font-medium"
      style={{ backgroundColor: `${color}10`, color, border: `1px solid ${color}20` }}
    >
      {label}
    </span>
  );
}

function ResourceDetailModal({ resource, onClose }: { resource: RelatedResource; onClose: () => void }) {
  const statusColors: Record<string, string> = {
    healthy: '#22c55e',
    warning: '#eab308',
    critical: '#ef4444',
    missing: '#6b7280',
  };
  const sColor = statusColors[resource.status] || '#6b7280';

  const detailLabels: Record<string, string> = {
    hasTLS: 'TLS Encryption',
    hasAuth: 'Authentication',
    hasJWT: 'JWT Authentication',
    jwtMode: 'JWT Mode',
    hasRBAC: 'RBAC Authorization',
    hasCORS: 'CORS Policy',
    hasCORSFilter: 'CORS Filter (Route)',
    hasCORSFromPolicy: 'CORS (via Policy)',
    hasRateLimit: 'Rate Limiting',
    hasPromptGuard: 'Prompt Guard',
    programmed: 'Programmed',
    ready: 'Ready',
    backendType: 'Backend Type',
    gatewayClassName: 'Gateway Class',
    host: 'Host',
    port: 'Port',
    protocol: 'Protocol',
    url: 'URL',
    transport: 'Transport',
    toolCount: 'Tool Count',
    allowedTools: 'Allowed Tools',
    listeners: 'Listeners',
    parentRefs: 'Parent Refs',
    rules: 'Rules',
    usesAGWBackend: 'Uses AGW Backend',
    parentGateway: 'Parent Gateway',
    targetName: 'Target Name',
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="bg-gov-surface rounded-2xl border border-gov-border shadow-2xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col"
        onClick={e => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-start justify-between p-5 border-b border-gov-border">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs font-mono px-2 py-0.5 rounded bg-gov-bg text-gov-text-3">{resource.kind}</span>
              <span
                className="px-2 py-0.5 rounded-lg text-[10px] font-bold uppercase tracking-wider"
                style={{ backgroundColor: `${sColor}15`, color: sColor }}
              >
                {resource.status}
              </span>
            </div>
            <h3 className="text-lg font-bold text-gov-text break-all">{resource.name}</h3>
            <div className="text-xs text-gov-text-3 mt-0.5 font-mono">{resource.namespace}</div>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-xl hover:bg-gov-bg transition-colors flex-shrink-0 ml-3"
          >
            <X size={18} className="text-gov-text-3" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-5 overflow-y-auto flex-1 space-y-4">
          {resource.details && Object.keys(resource.details).length > 0 ? (
            <>
              {/* Boolean properties as visual cards */}
              <div className="grid grid-cols-2 gap-2">
                {Object.entries(resource.details)
                  .filter(([, v]) => typeof v === 'boolean')
                  .map(([key, value]) => {
                    let enabled = value as boolean;
                    let label = detailLabels[key] || key;

                    // Special handling: if hasCORSFilter is false but hasCORSFromPolicy is true,
                    // show CORS as enabled via policy instead of a misleading red "Disabled" card
                    if (key === 'hasCORSFilter' && !enabled && resource.details?.hasCORSFromPolicy === true) {
                      return null; // Skip — covered by hasCORSFromPolicy
                    }

                    const color = enabled ? '#22c55e' : '#ef4444';
                    return (
                      <div
                        key={key}
                        className="rounded-xl border p-3 flex items-center gap-3"
                        style={{ borderColor: `${color}30`, backgroundColor: `${color}06` }}
                      >
                        <div
                          className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                          style={{ backgroundColor: color, boxShadow: `0 0 6px ${color}40` }}
                        />
                        <div>
                          <div className="text-xs font-semibold text-gov-text">{label}</div>
                          <div className="text-[10px] font-medium" style={{ color }}>
                            {enabled ? 'Enabled' : 'Disabled'}
                          </div>
                        </div>
                      </div>
                    );
                  }).filter(Boolean)}
              </div>

              {/* Non-boolean properties as key-value rows */}
              {Object.entries(resource.details)
                .filter(([, v]) => typeof v !== 'boolean' && v !== null && v !== undefined && v !== '')
                .length > 0 && (
                <div className="bg-gov-bg rounded-xl border border-gov-border overflow-hidden">
                  <div className="divide-y divide-gov-border">
                    {Object.entries(resource.details)
                      .filter(([, v]) => typeof v !== 'boolean' && v !== null && v !== undefined && v !== '')
                      .map(([key, value]) => (
                        <div key={key} className="px-4 py-3 flex items-start justify-between gap-4">
                          <span className="text-xs font-semibold text-gov-text-3 uppercase tracking-wider whitespace-nowrap">
                            {detailLabels[key] || key}
                          </span>
                          <span className="text-sm text-gov-text font-mono text-right break-all">
                            {Array.isArray(value)
                              ? value.join(', ')
                              : typeof value === 'object'
                                ? JSON.stringify(value, null, 2)
                                : String(value)}
                          </span>
                        </div>
                      ))}
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="text-center py-8">
              <Server size={28} className="mx-auto mb-3 text-gov-text-3 opacity-40" />
              <p className="text-sm text-gov-text-3">No additional details available for this resource.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function ScoreExplanationModal({ explanation, onClose }: { explanation: ScoreExplanation; onClose: () => void }) {
  const score = explanation.score;
  const color = getScoreColor(score);

  const statusConfig: Record<string, { icon: typeof CheckCircle2; label: string; color: string }> = {
    pass: { icon: CheckCircle2, label: 'Passing', color: '#22c55e' },
    partial: { icon: AlertTriangle, label: 'Partial', color: '#eab308' },
    fail: { icon: XCircle, label: 'Failing', color: '#ef4444' },
    'not-required': { icon: MinusCircle, label: 'Not Required', color: '#64748b' },
  };

  const cfg = statusConfig[explanation.status] || statusConfig['fail'];
  const StatusIcon = cfg.icon;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="bg-gov-surface rounded-2xl border border-gov-border shadow-2xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-start justify-between p-5 border-b border-gov-border">
          <div className="flex items-center gap-3">
            <div
              className="w-14 h-14 rounded-xl border-2 flex flex-col items-center justify-center"
              style={{ borderColor: color, backgroundColor: `${color}08` }}
            >
              <span className="text-xl font-black tabular-nums" style={{ color }}>{score}</span>
              <span className="text-[9px] text-gov-text-3">/ {explanation.maxScore}</span>
            </div>
            <div>
              <h3 className="text-lg font-bold text-gov-text">{explanation.category}</h3>
              <div className="flex items-center gap-1.5 mt-0.5">
                <StatusIcon size={12} style={{ color: cfg.color }} />
                <span className="text-xs font-semibold" style={{ color: cfg.color }}>{cfg.label}</span>
              </div>
            </div>
          </div>
          <button onClick={onClose} className="p-2 rounded-xl hover:bg-gov-bg transition-colors flex-shrink-0 ml-3">
            <X size={18} className="text-gov-text-3" />
          </button>
        </div>

        {/* Body */}
        <div className="p-5 overflow-y-auto flex-1 space-y-4">
          {/* Score bar */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <span className="text-xs text-gov-text-3 font-medium">Score</span>
              <span className="text-xs font-bold" style={{ color }}>{score} / {explanation.maxScore}</span>
            </div>
            <div className="h-2.5 bg-gov-bg rounded-full overflow-hidden">
              <div
                className="h-full rounded-full transition-all duration-700"
                style={{
                  width: `${(score / explanation.maxScore) * 100}%`,
                  backgroundColor: color,
                  boxShadow: `0 0 8px ${color}40`,
                }}
              />
            </div>
          </div>

          {/* Reasons */}
          {explanation.reasons && explanation.reasons.length > 0 && (
            <div>
              <div className="text-xs font-bold text-gov-text-2 uppercase tracking-wider mb-2">How This Score Is Calculated</div>
              <div className="space-y-2">
                {explanation.reasons.map((reason, i) => (
                  <div key={i} className="flex items-start gap-2.5 bg-gov-bg rounded-xl p-3 border border-gov-border">
                    <CheckCircle2 size={14} className="text-blue-400 mt-0.5 flex-shrink-0" />
                    <span className="text-sm text-gov-text leading-relaxed">{reason}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Suggestions */}
          {explanation.suggestions && explanation.suggestions.length > 0 && (
            <div>
              <div className="text-xs font-bold text-gov-text-2 uppercase tracking-wider mb-2">Suggestions to Improve</div>
              <div className="space-y-2">
                {explanation.suggestions.map((suggestion, i) => (
                  <div key={i} className="flex items-start gap-2.5 bg-yellow-500/5 rounded-xl p-3 border border-yellow-500/20">
                    <AlertTriangle size={14} className="text-yellow-400 mt-0.5 flex-shrink-0" />
                    <span className="text-sm text-gov-text leading-relaxed">{suggestion}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Sources */}
          {explanation.sources && explanation.sources.length > 0 && (
            <div>
              <div className="text-xs font-bold text-gov-text-2 uppercase tracking-wider mb-2">Contributing Resources</div>
              <div className="flex flex-wrap gap-2">
                {explanation.sources.map((source, i) => (
                  <span key={i} className="px-2.5 py-1 bg-gov-bg rounded-lg text-xs font-mono text-gov-text-2 border border-gov-border">
                    {source}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function FindingCard({ finding }: { finding: Finding }) {
  const color = severityColors[finding.severity] || '#6b7280';
  return (
    <div className="bg-gov-surface rounded-xl border border-gov-border p-4">
      <div className="flex items-center gap-2 mb-2">
        <span
          className="px-2 py-0.5 rounded-md text-[10px] font-black uppercase tracking-wider"
          style={{ backgroundColor: `${color}15`, color }}
        >
          {finding.severity}
        </span>
        <span className="text-xs font-mono text-gov-text-3">{finding.id}</span>
        <span className="text-gov-border-light text-xs">•</span>
        <span className="text-xs text-gov-text-3">{finding.category}</span>
      </div>
      <div className="text-sm font-semibold text-gov-text">{finding.title}</div>
      <div className="text-xs text-gov-text-3 mt-1.5 leading-relaxed">{finding.description}</div>
      {finding.impact && (
        <div className="text-xs text-red-400/80 mt-2">
          <span className="font-semibold">Impact:</span> {finding.impact}
        </div>
      )}
      {finding.remediation && (
        <div className="text-xs text-blue-400/80 mt-1">
          <span className="font-semibold">💡 Fix:</span> {finding.remediation}
        </div>
      )}
    </div>
  );
}
