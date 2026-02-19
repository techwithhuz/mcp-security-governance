'use client';

import { useState, useRef, useEffect } from 'react';
import {
  ShieldCheck, ShieldAlert, ShieldX, Clock, ChevronDown, ChevronUp,
  BadgeCheck, Building2, User, Server, Layers, Info, X, Bot, Wrench, Search,
} from 'lucide-react';
import type { VerifiedScore, VerifiedInventory } from '@/lib/types';

interface VerifiedCatalogProps {
  inventory: VerifiedInventory;
}

const statusConfig: Record<string, { label: string; color: string; bg: string; border: string; icon: typeof ShieldCheck }> = {
  Verified:   { label: 'Verified',   color: '#22c55e', bg: 'bg-green-500/10',  border: 'border-green-500/30',  icon: ShieldCheck },
  Unverified: { label: 'Unverified', color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30', icon: ShieldAlert },
  Rejected:   { label: 'Rejected',   color: '#ef4444', bg: 'bg-red-500/10',    border: 'border-red-500/30',    icon: ShieldX },
  Pending:    { label: 'Pending',    color: '#6b7280', bg: 'bg-gray-500/10',   border: 'border-gray-500/30',   icon: Clock },
};

const getScoreColor = (score: number) => {
  if (score >= 90) return '#22c55e';
  if (score >= 70) return '#eab308';
  if (score >= 50) return '#f97316';
  return '#ef4444';
};

const getGradeColor = (grade: string) => {
  if (grade === 'A' || grade === 'A+') return '#22c55e';
  if (grade === 'B' || grade === 'B+') return '#84cc16';
  if (grade === 'C' || grade === 'C+') return '#eab308';
  if (grade === 'D') return '#f97316';
  return '#ef4444';
};

type FilterStatus = 'All' | string;

export default function VerifiedCatalog({ inventory }: VerifiedCatalogProps) {
  const [expandedItem, setExpandedItem] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('All');
  const [searchQuery, setSearchQuery] = useState('');

  const items = inventory.items || [];
  const filtered = items.filter(item => {
    if (filterStatus !== 'All' && item.status !== filterStatus) return false;
    if (!searchQuery.trim()) return true;
    const query = searchQuery.toLowerCase();
    return (
      item.catalogName?.toLowerCase().includes(query) ||
      item.namespace?.toLowerCase().includes(query) ||
      (item.verifiedOrg && item.verifiedOrg.toLowerCase().includes(query)) ||
      (item.verifiedPublisher && item.verifiedPublisher.toLowerCase().includes(query))
    );
  });

  // Sort: Rejected first, then Unverified, Pending, Verified
  const statusOrder: Record<string, number> = { Rejected: 0, Unverified: 1, Pending: 2, Verified: 3 };
  const sorted = [...filtered].sort((a, b) => (statusOrder[a.status] ?? 4) - (statusOrder[b.status] ?? 4));

  return (
    <div className="bg-gov-surface rounded-2xl border border-gov-border">
      {/* Header */}
      <div className="p-5 border-b border-gov-border">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-bold">Verified Catalog Inventory</h2>
            <p className="text-xs text-gov-text-3 mt-1">
              MCPServerCatalog entries scored for trust, security, and compliance
            </p>
          </div>
          <div className="flex items-center gap-3">
            {/* Average Score */}
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-gov-bg border border-gov-border">
              <span className="text-xs text-gov-text-3">Avg Score</span>
              <span
                className="text-sm font-black tabular-nums"
                style={{ color: getScoreColor(inventory.averageScore) }}
              >
                {inventory.averageScore}
              </span>
            </div>
          </div>
        </div>

        {/* Summary Cards */}
        <div className="grid grid-cols-5 gap-3 mb-4">
          <div className="bg-gov-bg rounded-xl border border-gov-border p-3 text-center">
            <p className="text-2xl font-black tabular-nums">{inventory.totalScored}</p>
            <p className="text-[10px] text-gov-text-3 uppercase tracking-wider font-semibold mt-1">Total</p>
          </div>
          <div className="bg-gov-bg rounded-xl border border-green-500/20 p-3 text-center">
            <p className="text-2xl font-black tabular-nums text-green-400">{inventory.totalVerified}</p>
            <p className="text-[10px] text-green-400/70 uppercase tracking-wider font-semibold mt-1">Verified</p>
          </div>
          <div className="bg-gov-bg rounded-xl border border-yellow-500/20 p-3 text-center">
            <p className="text-2xl font-black tabular-nums text-yellow-400">{inventory.totalUnverified}</p>
            <p className="text-[10px] text-yellow-400/70 uppercase tracking-wider font-semibold mt-1">Unverified</p>
          </div>
          <div className="bg-gov-bg rounded-xl border border-red-500/20 p-3 text-center">
            <p className="text-2xl font-black tabular-nums text-red-400">{inventory.totalRejected}</p>
            <p className="text-[10px] text-red-400/70 uppercase tracking-wider font-semibold mt-1">Rejected</p>
          </div>
          <div className="bg-gov-bg rounded-xl border border-gray-500/20 p-3 text-center">
            <p className="text-2xl font-black tabular-nums text-gray-400">{inventory.totalPending}</p>
            <p className="text-[10px] text-gray-400/70 uppercase tracking-wider font-semibold mt-1">Pending</p>
          </div>
        </div>

        {/* Filter */}
        <div className="space-y-3">
          {/* Search Bar */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gov-text-3" />
            <input
              type="text"
              placeholder="Search by name, namespace, organization, or publisher..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-10 py-2.5 bg-gov-bg border border-gov-border rounded-lg text-sm text-gov-text placeholder-gov-text-3 focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10 transition-all"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gov-text-3 hover:text-gov-text transition-colors"
                title="Clear search"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>

          {/* Filter buttons */}
          <div className="flex gap-2 flex-wrap">
          {['All', 'Verified', 'Unverified', 'Rejected', 'Pending'].map(status => (
            <button
              key={status}
              onClick={() => setFilterStatus(status)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                filterStatus === status
                  ? 'bg-gov-accent/15 text-blue-400 border border-blue-500/30'
                  : 'text-gov-text-3 hover:text-gov-text-2 bg-gov-bg border border-gov-border hover:border-gov-border-light'
              }`}
            >
              {status}
              {status !== 'All' && (
                <span className="ml-1 tabular-nums">
                  {items.filter(i => i.status === status).length}
                </span>
              )}
              {status === 'All' && (
                <span className="ml-1 tabular-nums">{items.length}</span>
              )}
            </button>
          ))}
          </div>
        </div>
      </div>

      {/* Catalog Entries */}
      <div className="divide-y divide-gov-border">
        {sorted.length === 0 && (
          <div className="p-8 text-center text-gov-text-3 text-sm">
            No catalog entries found matching the filter.
          </div>
        )}
        {sorted.map(item => {
          const cfg = statusConfig[item.status] || statusConfig.Pending;
          const StatusIcon = cfg.icon;
          const scoreColor = getScoreColor(item.score);
          const isExpanded = expandedItem === item.resourceRef;

          return (
            <div key={item.resourceRef} className="hover:bg-gov-bg/50 transition-colors">
              {/* Row */}
              <div
                className="flex items-center gap-4 p-4 cursor-pointer"
                onClick={() => setExpandedItem(isExpanded ? null : item.resourceRef ?? null)}
              >
                {/* Score circle */}
                <div
                  className="w-11 h-11 rounded-full flex items-center justify-center border-2 font-black text-sm tabular-nums flex-shrink-0"
                  style={{ borderColor: scoreColor, color: scoreColor, backgroundColor: `${scoreColor}10` }}
                >
                  {item.score}
                </div>

                {/* Name + namespace */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-gov-text truncate">{item.catalogName}</span>
                    <span
                      className="text-xs font-black px-1.5 py-0.5 rounded"
                      style={{ color: getGradeColor(item.grade), backgroundColor: `${getGradeColor(item.grade)}15` }}
                    >
                      {item.grade}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-xs font-mono text-gov-text-3 px-1.5 py-0.5 bg-gov-surface rounded">
                      {item.namespace}
                    </span>
                    {item.verifiedOrg && (
                      <span className="flex items-center gap-1 text-xs text-gov-text-3">
                        <Building2 size={10} />
                        {item.verifiedOrg}
                      </span>
                    )}
                    {item.verifiedPublisher && (
                      <span className="flex items-center gap-1 text-xs text-gov-text-3">
                        <User size={10} />
                        {item.verifiedPublisher}
                      </span>
                    )}
                  </div>
                </div>

                {/* Status badge */}
                <div className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold ${cfg.bg} ${cfg.border} border`}
                  style={{ color: cfg.color }}
                >
                  <StatusIcon size={12} />
                  {cfg.label}
                </div>

                {/* Agent count */}
                {item.usedByAgents && item.usedByAgents.length > 0 && (
                  <div className="flex items-center gap-1 text-xs text-gov-text-3">
                    <Bot size={12} />
                    <span className="tabular-nums">{item.usedByAgents.length}</span>
                  </div>
                )}

                {/* Expand chevron */}
                <div className="text-gov-text-3">
                  {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                </div>
              </div>

              {/* Expanded detail */}
              {isExpanded && (
                <div className="px-4 pb-4 ml-[60px]">
                  <div className="bg-gov-bg rounded-xl border border-gov-border p-4 space-y-4">
                    {/* Reason */}
                    <div>
                      <p className="text-xs text-gov-text-3 uppercase tracking-wider font-semibold mb-1">Reason</p>
                      <p className="text-sm text-gov-text-2">{item.reason}</p>
                    </div>

                    {/* Category Scores */}
                    <div>
                      <p className="text-xs text-gov-text-3 uppercase tracking-wider font-semibold mb-2">Category Scores</p>
                      <div className="grid grid-cols-3 gap-3">
                        <ScoreBar label="Security" score={item.securityScore} weight="50%" item={item} />
                        <ScoreBar label="Trust" score={item.trustScore} weight="30%" item={item} />
                        <ScoreBar label="Compliance" score={item.complianceScore} weight="20%" item={item} />
                      </div>
                      <p className="text-[10px] text-gov-text-3 mt-2 italic">
                        💡 Each category is <strong>normalized to 0-100</strong> for fair comparison. Click the ℹ️ icon to see raw check points (score/maxScore).
                      </p>
                    </div>

                    {/* Publisher Trust Scores */}
                    <div>
                      <p className="text-xs text-gov-text-3 uppercase tracking-wider font-semibold mb-2">Publisher Trust</p>
                      <div className="grid grid-cols-2 gap-3">
                        <OrgPublisherTile
                          icon={Building2}
                          label="Organization"
                          score={item.orgScore}
                          name={item.verifiedOrg}
                          infoTitle="Organization Score"
                          infoDescription="Derived from the environment and cluster labels on the MCPServerCatalog resource (agentregistry.dev/environment, agentregistry.dev/cluster). Full score when both labels are present."
                        />
                        <OrgPublisherTile
                          icon={User}
                          label="Publisher"
                          score={item.publisherScore}
                          name={item.verifiedPublisher}
                          infoTitle="Publisher Score"
                          infoDescription="Derived from the source tracking labels (agentregistry.dev/source-kind, agentregistry.dev/source-name) and management type on the MCPServerCatalog. Full score when source is tracked and management type is set."
                        />
                      </div>
                    </div>

                    {/* Linked Agents */}
                    {item.usedByAgents && item.usedByAgents.length > 0 && (
                      <div>
                        <p className="text-xs text-gov-text-3 uppercase tracking-wider font-semibold mb-2">Linked Agents</p>
                        <div className="flex flex-wrap gap-2">
                          {item.usedByAgents.map(agent => (
                            <span
                              key={`${agent.namespace}/${agent.name}`}
                              className="flex items-center gap-1 text-xs px-2 py-1 rounded-lg bg-gov-surface border border-gov-border text-gov-text-2 font-mono"
                            >
                              <Bot size={10} className="text-purple-400" />
                              {agent.namespace}/{agent.name}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Tools */}
                    {item.toolNames && item.toolNames.length > 0 && (
                      <div>
                        <p className="text-xs text-gov-text-3 uppercase tracking-wider font-semibold mb-2">
                          Available Tools <span className="text-gov-text-3 font-normal">({item.toolNames.length})</span>
                        </p>
                        <div className="flex flex-wrap gap-1.5">
                          {item.toolNames.slice(0, 30).map(tool => (
                            <span
                              key={tool}
                              className="text-[10px] px-1.5 py-0.5 rounded bg-gov-surface border border-gov-border text-gov-text-2 font-mono"
                            >
                              <Wrench size={8} className="inline mr-0.5 text-blue-400" />
                              {tool}
                            </span>
                          ))}
                          {item.toolNames.length > 30 && (
                            <span className="text-[10px] px-1.5 py-0.5 rounded bg-gov-surface border border-gov-border text-gov-text-3">
                              +{item.toolNames.length - 30} more
                            </span>
                          )}
                        </div>
                      </div>
                    )}

                    {/* Scored at */}
                    <div className="flex items-center gap-1.5 text-xs text-gov-text-3">
                      <Clock size={12} />
                      Scored at {new Date(item.scoredAt || '').toLocaleString()}
                    </div>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

/** Small helper: horizontal score bar for a category with info popup. */
function ScoreBar({ label, score, weight, item }: { label: string; score: number; weight: string; item: VerifiedScore }) {
  const color = getScoreColor(score);
  const [showPopup, setShowPopup] = useState(false);
  const popupRef = useRef<HTMLDivElement>(null);

  // Close popup when clicking outside
  useEffect(() => {
    if (!showPopup) return;
    const handler = (e: MouseEvent) => {
      if (popupRef.current && !popupRef.current.contains(e.target as Node)) {
        setShowPopup(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [showPopup]);

  return (
    <div className="bg-gov-surface rounded-lg p-2.5 border border-gov-border relative">
      <div className="flex items-center justify-between mb-1.5">
        <div className="flex items-center gap-1">
          <span className="text-xs text-gov-text-2 font-medium">{label}</span>
          <button
            onClick={(e) => { e.stopPropagation(); setShowPopup(!showPopup); }}
            className="text-gov-text-3 hover:text-blue-400 transition-colors"
            title={`How ${label} score is calculated`}
          >
            <Info size={11} />
          </button>
        </div>
        <span className="text-[10px] text-gov-text-3">{weight}</span>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex-1 h-1.5 rounded-full bg-gov-bg overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-500"
            style={{ width: `${score}%`, backgroundColor: color }}
          />
        </div>
        <span className="text-xs font-black tabular-nums" style={{ color }}>
          {score}
        </span>
      </div>

      {/* Popup */}
      {showPopup && (
        <div
          ref={popupRef}
          className="absolute z-50 left-0 top-full mt-2 w-80 bg-gov-bg rounded-xl border border-gov-border shadow-2xl shadow-black/40 p-4"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between mb-3">
            <h4 className="text-sm font-bold text-gov-text">{label} Score Breakdown</h4>
            <button onClick={() => setShowPopup(false)} className="text-gov-text-3 hover:text-gov-text transition-colors">
              <X size={14} />
            </button>
          </div>

          {label === 'Security' && <SecurityPopup item={item} />}
          {label === 'Trust' && <TrustPopup item={item} />}
          {label === 'Compliance' && <CompliancePopup item={item} />}

          {/* Formula */}
          <div className="mt-3 pt-3 border-t border-gov-border">
            <p className="text-[10px] text-gov-text-3 font-mono">
              Composite = (Security×50% + Trust×30% + Compliance×20%)
            </p>
            <p className="text-[10px] text-gov-text-3 font-mono mt-0.5">
              = ({item.securityScore}×0.5 + {item.trustScore}×0.3 + {item.complianceScore}×0.2) = <span className="font-bold text-gov-text-2">{item.score}</span>
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

/** Security score popup: shows matched MCP servers and their governance scores. */
function SecurityPopup({ item }: { item: VerifiedScore }) {
  const securityChecks = item.checks?.filter(c => c.category === 'transport' || c.category === 'deployment') || [];
  const secMax = securityChecks.reduce((sum, c) => sum + c.maxScore, 0);
  const secEarned = securityChecks.reduce((sum, c) => sum + c.score, 0);
  
  return (
    <div className="space-y-2">
      <p className="text-xs text-gov-text-2">
        Transport security and deployment readiness checks.
      </p>
      <div className="bg-gov-surface rounded-lg p-2.5 border border-gov-border space-y-1.5">
        {securityChecks.length > 0 ? (
          securityChecks.map(check => {
            const statusIcon = check.passed ? '✓' : '✗';
            const statusColor = check.passed ? 'text-green-400' : 'text-red-400';
            
            return (
              <div key={check.id} className="flex items-start justify-between text-[11px] pb-1.5 border-b border-gov-border/30 last:border-0 last:pb-0">
                <div className="flex items-start gap-1.5 flex-1">
                  <span className={`font-bold ${statusColor} mt-0.5`}>{statusIcon}</span>
                  <div className="flex-1 min-w-0">
                    <div className="font-mono text-gov-text-2 text-[10px]">{check.id}</div>
                    <div className="text-gov-text-2 truncate">{check.name}</div>
                    {check.detail && <div className="text-gov-text-3 text-[10px] mt-0.5">{check.detail}</div>}
                  </div>
                </div>
                <div className={`font-mono font-semibold ${statusColor} whitespace-nowrap ml-2`}>
                  {check.score}/{check.maxScore}
                </div>
              </div>
            );
          })
        ) : (
          <p className="text-[11px] text-gov-text-3 italic">No checks available</p>
        )}
        <div className="mt-2 pt-2 border-t border-gov-border/50">
          <p className="text-[11px] text-gov-text-3">
            Raw Total: <span className="font-mono">{secEarned}/{secMax}</span> points
          </p>
          <p className="text-[11px] text-gov-text-3 font-bold">
            Normalized Score (0-100): <span className="text-gov-text-2">{item.securityScore}/100</span>
          </p>
        </div>
      </div>
    </div>
  );
}

/** Trust score popup: shows source tracking + environment traceability. */
function TrustPopup({ item }: { item: VerifiedScore }) {
  const publisherChecks = item.checks?.filter(c => c.category === 'publisher') || [];
  const pubMax = publisherChecks.reduce((sum, c) => sum + c.maxScore, 0);
  const pubEarned = publisherChecks.reduce((sum, c) => sum + c.score, 0);
  
  return (
    <div className="space-y-2">
      <p className="text-xs text-gov-text-2">
        Source tracking and publisher verification from MCPServerCatalog labels.
      </p>
      <div className="bg-gov-surface rounded-lg p-2.5 border border-gov-border space-y-1.5">
        {publisherChecks.length > 0 ? (
          publisherChecks.map(check => {
            const statusIcon = check.passed ? '✓' : '✗';
            const statusColor = check.passed ? 'text-green-400' : 'text-red-400';
            
            return (
              <div key={check.id} className="flex items-start justify-between text-[11px] pb-1.5 border-b border-gov-border/30 last:border-0 last:pb-0">
                <div className="flex items-start gap-1.5 flex-1">
                  <span className={`font-bold ${statusColor} mt-0.5`}>{statusIcon}</span>
                  <div className="flex-1 min-w-0">
                    <div className="font-mono text-gov-text-2 text-[10px]">{check.id}</div>
                    <div className="text-gov-text-2 truncate">{check.name}</div>
                    {check.detail && <div className="text-gov-text-3 text-[10px] mt-0.5">{check.detail}</div>}
                  </div>
                </div>
                <div className={`font-mono font-semibold ${statusColor} whitespace-nowrap ml-2`}>
                  {check.score}/{check.maxScore}
                </div>
              </div>
            );
          })
        ) : (
          <p className="text-[11px] text-gov-text-3 italic">No checks available</p>
        )}
        <div className="mt-2 pt-2 border-t border-gov-border/50">
          <p className="text-[11px] text-gov-text-3">
            Raw Total: <span className="font-mono">{pubEarned}/{pubMax}</span> points
          </p>
          <p className="text-[11px] text-gov-text-3 font-bold">
            Normalized Score (0-100): <span className="text-gov-text-2">{item.trustScore}/100</span>
          </p>
        </div>
      </div>
    </div>
  );
}

/** Compliance score popup: shows metadata completeness checks. */
function CompliancePopup({ item }: { item: VerifiedScore }) {
  const complianceChecks = item.checks?.filter(c => c.category === 'toolScope' || c.category === 'usage') || [];
  const compMax = complianceChecks.reduce((sum, c) => sum + c.maxScore, 0);
  const compEarned = complianceChecks.reduce((sum, c) => sum + c.score, 0);
  
  return (
    <div className="space-y-2">
      <p className="text-xs text-gov-text-2">
        Tool scope and agent usage validation.
      </p>
      <div className="bg-gov-surface rounded-lg p-2.5 border border-gov-border space-y-1.5">
        {complianceChecks.length > 0 ? (
          complianceChecks.map(check => {
            const statusIcon = check.passed ? '✓' : '✗';
            const statusColor = check.passed ? 'text-green-400' : 'text-red-400';
            
            return (
              <div key={check.id} className="flex items-start justify-between text-[11px] pb-1.5 border-b border-gov-border/30 last:border-0 last:pb-0">
                <div className="flex items-start gap-1.5 flex-1">
                  <span className={`font-bold ${statusColor} mt-0.5`}>{statusIcon}</span>
                  <div className="flex-1 min-w-0">
                    <div className="font-mono text-gov-text-2 text-[10px]">{check.id}</div>
                    <div className="text-gov-text-2 truncate">{check.name}</div>
                    {check.detail && <div className="text-gov-text-3 text-[10px] mt-0.5">{check.detail}</div>}
                  </div>
                </div>
                <div className={`font-mono font-semibold ${statusColor} whitespace-nowrap ml-2`}>
                  {check.score}/{check.maxScore}
                </div>
              </div>
            );
          })
        ) : (
          <p className="text-[11px] text-gov-text-3 italic">No checks available</p>
        )}
        <div className="mt-2 pt-2 border-t border-gov-border/50">
          <p className="text-[11px] text-gov-text-3">
            Raw Total: <span className="font-mono">{compEarned}/{compMax}</span> points
          </p>
          <p className="text-[11px] text-gov-text-3 font-bold">
            Normalized Score (0-100): <span className="text-gov-text-2">{item.complianceScore}/100</span>
          </p>
        </div>
      </div>
    </div>
  );
}

/** Renders a single row in a scoring popup. */
function PopupRow({ label, value, status, detail }: { label: string; value: string; status: 'pass' | 'fail' | 'partial' | 'infer' | 'unknown'; detail?: string }) {
  const statusColors: Record<string, string> = {
    pass: 'text-green-400',
    fail: 'text-red-400',
    partial: 'text-yellow-400',
    infer: 'text-blue-400',
    unknown: 'text-gov-text-3',
  };
  const statusIcons: Record<string, string> = {
    pass: '✓',
    fail: '✗',
    partial: '~',
    infer: '?',
    unknown: '·',
  };

  return (
    <div className="flex items-center justify-between text-[11px]">
      <div className="flex items-center gap-1.5">
        <span className={`font-bold ${statusColors[status]}`}>{statusIcons[status]}</span>
        <span className="text-gov-text-2">{label}</span>
      </div>
      <div className="flex items-center gap-2">
        {detail && <span className="text-gov-text-3 text-[10px]">{detail}</span>}
        <span className={`font-mono font-semibold ${statusColors[status]}`}>{value}</span>
      </div>
    </div>
  );
}

/** Organization / Publisher tile with info icon popup. */
function OrgPublisherTile({ icon: Icon, label, score, name, infoTitle, infoDescription }: {
  icon: typeof Building2;
  label: string;
  score: number;
  name?: string;
  infoTitle: string;
  infoDescription: string;
}) {
  const [showInfo, setShowInfo] = useState(false);
  const popupRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!showInfo) return;
    const handler = (e: MouseEvent) => {
      if (popupRef.current && !popupRef.current.contains(e.target as Node)) {
        setShowInfo(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [showInfo]);

  return (
    <div className="relative flex items-center justify-between bg-gov-surface rounded-lg p-2.5 border border-gov-border">
      <div className="flex items-center gap-2">
        <Icon size={14} className="text-gov-text-3" />
        <span className="text-xs text-gov-text-2">{label}</span>
        <button
          onClick={(e) => { e.stopPropagation(); setShowInfo(!showInfo); }}
          className="text-gov-text-3 hover:text-blue-400 transition-colors"
          title={`About ${label} score`}
        >
          <Info size={11} />
        </button>
      </div>
      <div className="flex items-center gap-2">
        {name && <span className="text-[10px] text-gov-text-3 font-mono truncate max-w-[120px]">{name}</span>}
        <span className="text-sm font-black tabular-nums" style={{ color: getScoreColor(score) }}>
          {score}
        </span>
      </div>

      {showInfo && (
        <div
          ref={popupRef}
          className="absolute z-50 left-0 top-full mt-2 w-80 bg-gov-bg rounded-xl border border-gov-border shadow-2xl shadow-black/40 p-4"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between mb-2">
            <h4 className="text-sm font-bold text-gov-text">{infoTitle}</h4>
            <button onClick={() => setShowInfo(false)} className="text-gov-text-3 hover:text-gov-text transition-colors">
              <X size={14} />
            </button>
          </div>
          <p className="text-xs text-gov-text-2 leading-relaxed">{infoDescription}</p>
          <div className="mt-2 pt-2 border-t border-gov-border">
            <p className="text-[10px] text-gov-text-3">
              Current score: <span className="font-bold text-gov-text-2">{score}/100</span>
            </p>
          </div>
        </div>
      )}
    </div>
  );
}