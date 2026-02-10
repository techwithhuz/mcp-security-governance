'use client';

import { useEffect, useState, useCallback } from 'react';
import Image from 'next/image';
import { Shield, RefreshCw, Activity, Clock, AlertTriangle, CheckCircle2, Wifi, WifiOff, Server, ChevronRight } from 'lucide-react';
import ScoreGauge from '@/components/ScoreGauge';
import ResourceCards from '@/components/ResourceCards';
import FindingsTable from '@/components/FindingsTable';
import BreakdownChart from '@/components/BreakdownChart';
import TrendChart from '@/components/TrendChart';
import ScoreExplainer from '@/components/ScoreExplainer';
import ResourceInventory from '@/components/ResourceInventory';

interface DashboardData {
  score: { score: number; grade: string; phase: string; categories: any[]; explanation: string };
  findings: { findings: any[]; total: number; bySeverity: Record<string, number> };
  resources: any;
  breakdown: any;
  trends: { trends: any[] };
  resourceDetail: { resources: any[]; total: number };
}

export default function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [connected, setConnected] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'resources' | 'findings'>('overview');

  const fetchData = useCallback(async () => {
    try {
      setRefreshing(true);
      const [score, findings, resources, breakdown, trends, resourceDetail] = await Promise.all([
        fetch('/api/governance/score').then(r => r.json()),
        fetch('/api/governance/findings').then(r => r.json()),
        fetch('/api/governance/resources').then(r => r.json()),
        fetch('/api/governance/breakdown').then(r => r.json()),
        fetch('/api/governance/trends').then(r => r.json()),
        fetch('/api/governance/resources/detail').then(r => r.json()),
      ]);

      setData({ score, findings, resources, breakdown, trends, resourceDetail });
      setLastUpdated(new Date());
      setConnected(true);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to connect to governance API');
      setConnected(false);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 15000);
    return () => clearInterval(interval);
  }, [fetchData]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="relative mb-6">
            <Image src="/logo.svg" alt="MCP Governance" width={64} height={64} className="mx-auto animate-pulse drop-shadow-lg" />
            <div className="absolute -inset-4 bg-blue-500/10 rounded-full animate-ping" />
          </div>
          <h2 className="text-xl font-bold gradient-text mb-2">MCP Governance</h2>
          <p className="text-gov-text-3 text-sm">Connecting to governance controller...</p>
          <div className="mt-4 flex items-center justify-center gap-2">
            <div className="w-2 h-2 bg-blue-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
            <div className="w-2 h-2 bg-blue-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
            <div className="w-2 h-2 bg-blue-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
          </div>
        </div>
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center max-w-md">
          <WifiOff className="w-16 h-16 text-red-400 mx-auto mb-4" />
          <h2 className="text-xl font-bold text-gov-text mb-2">Connection Failed</h2>
          <p className="text-gov-text-3 text-sm mb-4">{error}</p>
          <p className="text-gov-text-3 text-xs mb-6">
            Make sure the governance controller is running and reachable.
          </p>
          <button
            onClick={fetchData}
            className="px-6 py-2.5 bg-gov-accent hover:bg-blue-600 text-white rounded-xl font-medium transition-all text-sm"
          >
            Retry Connection
          </button>
        </div>
      </div>
    );
  }

  if (!data) return null;

  const failingResources = (data.resourceDetail.resources || []).filter((r: any) => r.status === 'critical' || r.status === 'failing');
  const compliantResources = (data.resourceDetail.resources || []).filter((r: any) => r.status === 'compliant');

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="sticky top-0 z-50 bg-gov-bg/80 backdrop-blur-xl border-b border-gov-border">
        <div className="max-w-[1600px] mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Image src="/logo.svg" alt="MCP Governance" width={44} height={44} className="drop-shadow-lg" />
              <div>
                <h1 className="text-xl font-bold">
                  <span className="gradient-text">MCP Governance</span>
                  <span className="text-gov-text-3 font-normal ml-2 text-sm">Dashboard</span>
                </h1>
                <p className="text-xs text-gov-text-3 mt-0.5">
                  Kubernetes-native MCP Security Posture Management
                </p>
              </div>
            </div>
            <div className="flex items-center gap-4">
              {/* Connection status */}
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-gov-surface border border-gov-border">
                {connected ? (
                  <>
                    <Wifi className="w-3.5 h-3.5 text-green-400" />
                    <span className="text-xs text-green-400 font-medium">Connected</span>
                  </>
                ) : (
                  <>
                    <WifiOff className="w-3.5 h-3.5 text-red-400" />
                    <span className="text-xs text-red-400 font-medium">Disconnected</span>
                  </>
                )}
              </div>

              {/* Last updated */}
              {lastUpdated && (
                <div className="flex items-center gap-1.5 text-xs text-gov-text-3">
                  <Clock className="w-3.5 h-3.5" />
                  <span>{lastUpdated.toLocaleTimeString()}</span>
                </div>
              )}

              {/* Refresh button */}
              <button
                onClick={fetchData}
                disabled={refreshing}
                className="p-2 rounded-xl bg-gov-surface border border-gov-border hover:border-gov-border-light transition-all disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 text-gov-text-2 ${refreshing ? 'animate-spin' : ''}`} />
              </button>
            </div>
          </div>

          {/* Tab navigation */}
          <div className="flex gap-1 mt-4">
            {[
              { id: 'overview' as const, label: 'Overview', icon: Activity },
              { id: 'resources' as const, label: 'Resource Inventory', icon: Server },
              { id: 'findings' as const, label: 'All Findings', icon: AlertTriangle },
            ].map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                  activeTab === tab.id
                    ? 'bg-gov-accent/15 text-blue-400 border border-blue-500/30'
                    : 'text-gov-text-3 hover:text-gov-text-2 hover:bg-gov-surface'
                }`}
              >
                <tab.icon size={14} />
                {tab.label}
                {tab.id === 'resources' && (
                  <span className="ml-1 px-1.5 py-0.5 rounded-full text-xs bg-gov-bg font-bold tabular-nums">
                    {data.resourceDetail.total || 0}
                  </span>
                )}
                {tab.id === 'findings' && (
                  <span className="ml-1 px-1.5 py-0.5 rounded-full text-xs bg-red-500/15 text-red-400 font-bold tabular-nums">
                    {data.findings.total}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="max-w-[1600px] mx-auto px-6 py-6 space-y-6">

        {/* ========== OVERVIEW TAB ========== */}
        {activeTab === 'overview' && (
          <>
            {/* Top row: Score Gauge + Quick Stats */}
            <div className="grid grid-cols-12 gap-6">
              {/* Score gauge */}
              <div className="col-span-3">
                <ScoreGauge
                  score={data.score.score}
                  grade={data.score.grade}
                  phase={data.score.phase}
                />
              </div>

              {/* Quick stats cards */}
              <div className="col-span-5">
                <div className="grid grid-cols-2 gap-4 h-full">
                  {/* Total Findings */}
                  <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col justify-between">
                    <div className="flex items-center gap-2 text-gov-text-3 mb-2">
                      <AlertTriangle className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">Total Findings</span>
                    </div>
                    <div className="text-4xl font-black">{data.findings.total}</div>
                    <div className="flex gap-2 mt-2">
                      {Object.entries(data.findings.bySeverity || {}).map(([sev, count]) => {
                        const colors: Record<string, string> = {
                          Critical: 'text-red-400',
                          High: 'text-orange-400',
                          Medium: 'text-yellow-400',
                          Low: 'text-green-400',
                        };
                        return (
                          <span key={sev} className={`text-xs ${colors[sev] || 'text-gov-text-3'}`}>
                            {count as number} {sev.charAt(0)}
                          </span>
                        );
                      })}
                    </div>
                  </div>

                  {/* Resources at Risk */}
                  <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col justify-between">
                    <div className="flex items-center gap-2 text-gov-text-3 mb-2">
                      <Server className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">Resources</span>
                    </div>
                    <div className="text-4xl font-black">
                      <span className="text-red-400">{failingResources.length}</span>
                      <span className="text-lg text-gov-text-3 font-normal">/{data.resourceDetail.total || 0}</span>
                    </div>
                    <div className="text-xs text-gov-text-3 mt-1">
                      {failingResources.length > 0 ? (
                        <span className="text-red-400">{failingResources.length} need attention</span>
                      ) : (
                        <span className="text-green-400">All resources compliant</span>
                      )}
                    </div>
                  </div>

                  {/* MCP Endpoints */}
                  <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col justify-between">
                    <div className="flex items-center gap-2 text-gov-text-3 mb-2">
                      <Activity className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">MCP Endpoints</span>
                    </div>
                    <div className="text-4xl font-black">{data.resources.totalMCPEndpoints || 0}</div>
                    <div className="text-xs text-gov-text-3 mt-1">discovered endpoints</div>
                  </div>

                  {/* Compliant Resources */}
                  <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col justify-between">
                    <div className="flex items-center gap-2 text-gov-text-3 mb-2">
                      <CheckCircle2 className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">Compliant</span>
                    </div>
                    <div className="text-4xl font-black text-green-400">
                      {compliantResources.length}
                      <span className="text-lg text-gov-text-3 font-normal">/{data.resourceDetail.total || 0}</span>
                    </div>
                    <div className="text-xs text-gov-text-3 mt-1">resources passing all checks</div>
                  </div>
                </div>
              </div>

              {/* Score breakdown */}
              <div className="col-span-4">
                <BreakdownChart breakdown={data.breakdown} />
              </div>
            </div>

            {/* Score Explanation — explains WHY score is 41 / Grade D */}
            <ScoreExplainer
              score={data.score.score}
              grade={data.score.grade}
              categories={data.score.categories || []}
              explanation={data.score.explanation || ''}
            />

            {/* Failing Resources Quick View */}
            {failingResources.length > 0 && (
              <div className="bg-gov-surface rounded-2xl border border-red-500/20 p-5">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-2">
                    <AlertTriangle size={18} className="text-red-400" />
                    <h2 className="text-lg font-bold">Resources Needing Attention</h2>
                    <span className="px-2 py-0.5 rounded-full text-xs font-bold bg-red-500/15 text-red-400">
                      {failingResources.length}
                    </span>
                  </div>
                  <button
                    onClick={() => setActiveTab('resources')}
                    className="flex items-center gap-1 text-xs font-semibold text-blue-400 hover:text-blue-300 transition-colors"
                  >
                    View All Resources <ChevronRight size={14} />
                  </button>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                  {failingResources.slice(0, 6).map((r: any) => {
                    const scoreColor = r.score >= 70 ? '#22c55e' : r.score >= 50 ? '#eab308' : r.score >= 30 ? '#f97316' : '#ef4444';
                    return (
                      <div
                        key={r.resourceRef}
                        className="bg-gov-bg rounded-xl border border-gov-border p-4 hover:border-gov-border-light transition-all cursor-pointer"
                        onClick={() => setActiveTab('resources')}
                      >
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-semibold text-gov-text truncate max-w-[150px]">{r.name}</span>
                          </div>
                          <div
                            className="w-9 h-9 rounded-full flex items-center justify-center border-2 font-black text-sm tabular-nums"
                            style={{ borderColor: scoreColor, color: scoreColor, backgroundColor: `${scoreColor}10` }}
                          >
                            {r.score}
                          </div>
                        </div>
                        <div className="flex items-center gap-2 text-xs text-gov-text-3 mb-2">
                          <span className="font-mono px-1.5 py-0.5 bg-gov-surface rounded">{r.kind}</span>
                          <span>{r.namespace}</span>
                        </div>
                        <div className="flex gap-1.5">
                          {r.critical > 0 && <span className="text-xs font-bold text-red-400">{r.critical}C</span>}
                          {r.high > 0 && <span className="text-xs font-bold text-orange-400">{r.high}H</span>}
                          {r.medium > 0 && <span className="text-xs font-bold text-yellow-400">{r.medium}M</span>}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Resource Counts + Trends */}
            <div className="grid grid-cols-12 gap-6">
              <div className="col-span-5 space-y-6">
                <ResourceCards resources={data.resources} />
              </div>
              <div className="col-span-7">
                <TrendChart trends={data.trends.trends || []} />
              </div>
            </div>
          </>
        )}

        {/* ========== RESOURCES TAB ========== */}
        {activeTab === 'resources' && (
          <ResourceInventory resources={data.resourceDetail.resources || []} />
        )}

        {/* ========== FINDINGS TAB ========== */}
        {activeTab === 'findings' && (
          <FindingsTable findings={data.findings.findings || []} />
        )}

        {/* Footer */}
        <footer className="border-t border-gov-border pt-4 pb-8">
          <div className="flex items-center justify-between text-xs text-gov-text-3">
            <div className="flex items-center gap-2">
              <Shield className="w-3.5 h-3.5" />
              <span>MCP Security Governance Controller v0.1.0</span>
            </div>
            <div className="flex items-center gap-4">
              <span>AgentGateway + Kagent</span>
              <span>•</span>
              <span>Kubernetes-native</span>
              <span>•</span>
              <span className="font-mono">governance.mcp.io/v1alpha1</span>
            </div>
          </div>
        </footer>
      </main>
    </div>
  );
}
