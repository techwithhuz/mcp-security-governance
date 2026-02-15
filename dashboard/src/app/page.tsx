'use client';

import { useEffect, useState, useCallback, useRef } from 'react';
import Image from 'next/image';
import { Shield, RefreshCw, Activity, Clock, AlertTriangle, Wifi, WifiOff, Server, ChevronRight, Plug, Scan, Wrench } from 'lucide-react';
import ScoreGauge from '@/components/ScoreGauge';
import ResourceCards from '@/components/ResourceCards';
import FindingsTable from '@/components/FindingsTable';
import BreakdownChart from '@/components/BreakdownChart';
import TrendChart from '@/components/TrendChart';
import ScoreExplainer from '@/components/ScoreExplainer';
import ResourceInventory from '@/components/ResourceInventory';
import AIScoreCard from '@/components/AIScoreCard';
import MCPServerList from '@/components/MCPServerList';
import MCPServerDetail from '@/components/MCPServerDetail';
import type { MCPServerView, MCPServerSummary, MCPServersResponse } from '@/lib/types';

interface DashboardData {
  score: { score: number; grade: string; phase: string; categories: any[]; explanation: string; severityPenalties?: { Critical: number; High: number; Medium: number; Low: number } };
  findings: { findings: any[]; total: number; bySeverity: Record<string, number> };
  resources: any;
  breakdown: any;
  trends: { trends: any[] };
  resourceDetail: { resources: any[]; total: number };
  mcpServers: MCPServersResponse;
}

export default function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [connected, setConnected] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [activeTab, setActiveTab] = useState<'mcp-servers' | 'overview' | 'resources' | 'findings'>('overview');
  const [version, setVersion] = useState('');
  const [selectedMCPServer, setSelectedMCPServer] = useState<MCPServerView | null>(null);
  const [previousTab, setPreviousTab] = useState<'mcp-servers' | 'overview' | 'resources' | 'findings' | null>(null);
  const [lastScanTime, setLastScanTime] = useState<string>('');
  const [scanInterval, setScanInterval] = useState<string>('');
  const [scanning, setScanning] = useState(false);

  // Use a ref for selectedMCPServer inside fetchData so it doesn't cause re-fetch cascades
  const selectedMCPServerRef = useRef<MCPServerView | null>(null);
  useEffect(() => { selectedMCPServerRef.current = selectedMCPServer; }, [selectedMCPServer]);

  const fetchData = useCallback(async () => {
    try {
      setRefreshing(true);
      const [score, findings, resources, breakdown, trends, resourceDetail, mcpServers, health] = await Promise.all([
        fetch('/api/governance/score').then(r => r.json()),
        fetch('/api/governance/findings').then(r => r.json()),
        fetch('/api/governance/resources').then(r => r.json()),
        fetch('/api/governance/breakdown').then(r => r.json()),
        fetch('/api/governance/trends').then(r => r.json()),
        fetch('/api/governance/resources/detail').then(r => r.json()),
        fetch('/api/governance/mcp-servers').then(r => r.json()).catch(() => ({ servers: [], summary: {} })),
        fetch('/api/health').then(r => r.json()).catch(() => null),
      ]);

      if (health?.version) setVersion(health.version);
      if (health?.lastScanTime) setLastScanTime(health.lastScanTime);
      if (health?.scanInterval) setScanInterval(health.scanInterval);
      setData({ score, findings, resources, breakdown, trends, resourceDetail, mcpServers });
      setConnected(true);
      setError(null);

      // Update selected MCP server data if one is selected
      const currentSelected = selectedMCPServerRef.current;
      if (currentSelected && mcpServers?.servers) {
        const updated = mcpServers.servers.find((s: MCPServerView) => s.id === currentSelected.id);
        if (updated) setSelectedMCPServer(updated);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to connect to governance API');
      setConnected(false);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  const triggerScan = useCallback(async () => {
    try {
      setScanning(true);
      await fetch('/api/governance/scan/refresh', { method: 'POST' });
      await fetchData();
    } catch (err) {
      console.error('Scan failed:', err);
    } finally {
      setScanning(false);
    }
  }, [fetchData]);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000); // refresh UI data every 30s
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
          <h2 className="text-xl font-bold gradient-text mb-2">MCP Governance <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-gradient-to-r from-purple-500 to-pink-500 text-white uppercase tracking-wider align-middle">AI-Powered</span></h2>
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
  const mcpServers: MCPServerView[] = data.mcpServers?.servers || [];
  const mcpSummary: MCPServerSummary = data.mcpServers?.summary || {
    totalMCPServers: 0, routedServers: 0, unroutedServers: 0,
    securedServers: 0, atRiskServers: 0, criticalServers: 0,
    totalTools: 0, exposedTools: 0, averageScore: 0,
  };

  // Compute total findings across all MCP servers (each server counts its own findings)
  const mcpAggregatedFindings: any[] = [];
  mcpServers.forEach(s => {
    (s.findings || []).forEach((f: any) => {
      mcpAggregatedFindings.push({
        ...f,
        id: `${f.id}-${s.name}`,
        resourceRef: f.resourceRef || s.name,
      });
    });
  });
  const mcpTotalFindings = mcpAggregatedFindings.length;
  const mcpFindingsBySeverity: Record<string, number> = {};
  mcpAggregatedFindings.forEach((f: any) => {
    mcpFindingsBySeverity[f.severity] = (mcpFindingsBySeverity[f.severity] || 0) + 1;
  });

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="sticky top-0 z-50 bg-gov-bg/80 backdrop-blur-xl border-b border-gov-border">
        <div className="max-w-[1600px] mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4 cursor-pointer" onClick={() => { setActiveTab('overview'); setSelectedMCPServer(null); }}>
              <Image src="/logo.svg" alt="MCP Governance" width={44} height={44} className="drop-shadow-lg" />
              <div>
                <h1 className="text-xl font-bold flex items-center gap-2">
                  <span className="gradient-text">MCP Governance</span>
                  <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-gradient-to-r from-purple-500 to-pink-500 text-white uppercase tracking-wider">AI-Powered</span>
                  <span className="text-gov-text-3 font-normal ml-1 text-sm">Dashboard</span>
                </h1>
                <p className="text-xs text-gov-text-3 mt-0.5">
                  AI-Powered Kubernetes-native MCP Security Posture Management
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
              {lastScanTime && (
                <div className="flex items-center gap-1.5 text-xs text-gov-text-3" title={`Scan interval: ${scanInterval || 'N/A'}`}>
                  <Clock className="w-3.5 h-3.5" />
                  <span>Scanned {new Date(lastScanTime).toLocaleTimeString()}</span>
                  {scanInterval && <span className="text-gov-text-3/50">({scanInterval})</span>}
                </div>
              )}

              {/* Scan Now button */}
              <button
                onClick={triggerScan}
                disabled={scanning}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-gov-accent/10 border border-blue-500/30 hover:bg-gov-accent/20 transition-all disabled:opacity-50 text-xs font-medium text-blue-400"
                title="Trigger an on-demand governance scan"
              >
                <Scan className={`w-3.5 h-3.5 ${scanning ? 'animate-spin' : ''}`} />
                {scanning ? 'Scanning...' : 'Scan Now'}
              </button>

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
              { id: 'mcp-servers' as const, label: 'MCP Servers', icon: Plug },
              { id: 'resources' as const, label: 'Resource Inventory', icon: Server },
              { id: 'findings' as const, label: 'All Findings', icon: AlertTriangle },
            ].map(tab => (
              <button
                key={tab.id}
                onClick={() => { setActiveTab(tab.id); setSelectedMCPServer(null); }}
                className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                  activeTab === tab.id
                    ? 'bg-gov-accent/15 text-blue-400 border border-blue-500/30'
                    : 'text-gov-text-3 hover:text-gov-text-2 hover:bg-gov-surface'
                }`}
              >
                <tab.icon size={14} />
                {tab.label}
                {tab.id === 'mcp-servers' && data.mcpServers?.summary?.totalMCPServers > 0 && (
                  <span className="ml-1 px-1.5 py-0.5 rounded-full text-xs bg-blue-500/15 text-blue-400 font-bold tabular-nums">
                    {data.mcpServers.summary.totalMCPServers}
                  </span>
                )}
                {tab.id === 'resources' && (
                  <span className="ml-1 px-1.5 py-0.5 rounded-full text-xs bg-gov-bg font-bold tabular-nums">
                    {data.resourceDetail.total || 0}
                  </span>
                )}
                {tab.id === 'findings' && (
                  <span className="ml-1 px-1.5 py-0.5 rounded-full text-xs bg-red-500/15 text-red-400 font-bold tabular-nums">
                    {mcpTotalFindings}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="max-w-[1600px] mx-auto px-6 py-6 space-y-6">

        {/* ========== MCP SERVERS TAB ========== */}
        {activeTab === 'mcp-servers' && (
          selectedMCPServer ? (
            <MCPServerDetail
              server={selectedMCPServer}
              onBack={() => {
                setSelectedMCPServer(null);
                if (previousTab && previousTab !== 'mcp-servers') {
                  setActiveTab(previousTab);
                  setPreviousTab(null);
                }
              }}
            />
          ) : (
            <MCPServerList
              servers={data.mcpServers?.servers || []}
              summary={data.mcpServers?.summary || {
                totalMCPServers: 0,
                routedViaGateway: 0,
                atRiskCount: 0,
                averageScore: 0,
                byStatus: { critical: 0, warning: 0, compliant: 0 },
              }}
              onSelectServer={(s) => { setPreviousTab(null); setSelectedMCPServer(s); }}
            />
          )
        )}

        {/* ========== OVERVIEW TAB ========== */}
        {activeTab === 'overview' && (
          <>
            {/* Top row: Score Gauge + MCP Server Summary + Breakdown */}
            <div className="grid grid-cols-12 gap-6">
              {/* Score gauge */}
              <div className="col-span-3">
                <ScoreGauge
                  score={data.score.score}
                  grade={data.score.grade}
                  phase={data.score.phase}
                />
              </div>

              {/* MCP-centric quick stats */}
              <div className="col-span-5">
                <div className="grid grid-cols-2 gap-4 h-full">
                  {/* MCP Servers */}
                  <div
                    className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col cursor-pointer"
                    onClick={() => setActiveTab('mcp-servers')}
                  >
                    <div className="flex items-center gap-2 text-gov-text-3 mb-1">
                      <Plug className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">MCP Servers</span>
                    </div>
                    <div className="flex-1 flex items-center justify-center">
                      <span className="text-6xl font-black">{mcpSummary.totalMCPServers}</span>
                    </div>
                    <div className="flex gap-3 justify-center mt-1">
                      <span className="text-xs text-green-400">{mcpSummary.routedServers} routed</span>
                      <span className="text-xs text-red-400">{mcpSummary.unroutedServers} unrouted</span>
                    </div>
                  </div>

                  {/* MCP Servers At Risk */}
                  <div
                    className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col cursor-pointer"
                    onClick={() => setActiveTab('mcp-servers')}
                  >
                    <div className="flex items-center gap-2 text-gov-text-3 mb-1">
                      <AlertTriangle className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">MCP Servers At Risk</span>
                    </div>
                    <div className="flex-1 flex items-center justify-center">
                      <span className="text-6xl font-black text-red-400">{mcpSummary.atRiskServers}</span>
                      <span className="text-2xl text-gov-text-3 font-normal ml-1">/{mcpSummary.totalMCPServers}</span>
                    </div>
                    <div className="text-xs text-center mt-1">
                      {mcpSummary.criticalServers > 0 ? (
                        <span className="text-red-400">{mcpSummary.criticalServers} critical</span>
                      ) : (
                        <span className="text-green-400">No critical servers</span>
                      )}
                    </div>
                  </div>

                  {/* Total Findings */}
                  <div
                    className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col cursor-pointer"
                    onClick={() => setActiveTab('findings')}
                  >
                    <div className="flex items-center gap-2 text-gov-text-3 mb-1">
                      <AlertTriangle className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">Total Findings</span>
                    </div>
                    <div className="flex-1 flex items-center justify-center">
                      <span className="text-6xl font-black">{mcpTotalFindings}</span>
                    </div>
                    <div className="flex gap-2 justify-center mt-1">
                      {Object.entries(mcpFindingsBySeverity).map(([sev, count]) => {
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

                  {/* Total Tools */}
                  <div
                    className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover flex flex-col cursor-pointer"
                    onClick={() => setActiveTab('mcp-servers')}
                  >
                    <div className="flex items-center gap-2 text-gov-text-3 mb-1">
                      <Wrench className="w-4 h-4" />
                      <span className="text-xs font-semibold uppercase tracking-wider">Tools Exposed</span>
                    </div>
                    <div className="flex-1 flex items-center justify-center">
                      <span className="text-6xl font-black">{mcpSummary.exposedTools}</span>
                      <span className="text-2xl text-gov-text-3 font-normal ml-1">/{mcpSummary.totalTools}</span>
                    </div>
                    <div className="text-xs text-center mt-1 text-gov-text-3">
                      Across {mcpSummary.totalMCPServers} MCP server{mcpSummary.totalMCPServers !== 1 ? 's' : ''}
                    </div>
                  </div>
                </div>
              </div>

              {/* Score breakdown */}
              <div className="col-span-4">
                <BreakdownChart breakdown={data.breakdown} />
              </div>
            </div>

            {/* MCP Servers Quick View */}
            {mcpServers.length > 0 && (
              <div className="bg-gov-surface rounded-2xl border border-gov-border p-5">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-2">
                    <Plug size={18} className="text-blue-400" />
                    <h2 className="text-lg font-bold">MCP Servers</h2>
                    <span className="px-2 py-0.5 rounded-full text-xs font-bold bg-blue-500/15 text-blue-400">
                      {mcpServers.length}
                    </span>
                  </div>
                  <button
                    onClick={() => setActiveTab('mcp-servers')}
                    className="flex items-center gap-1 text-xs font-semibold text-blue-400 hover:text-blue-300 transition-colors"
                  >
                    View All <ChevronRight size={14} />
                  </button>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                  {mcpServers.slice(0, 6).map((s: MCPServerView) => {
                    const scoreColor = s.score >= 70 ? '#22c55e' : s.score >= 50 ? '#eab308' : s.score >= 30 ? '#f97316' : '#ef4444';
                    return (
                      <div
                        key={s.id}
                        className="bg-gov-bg rounded-xl border border-gov-border p-4 hover:border-gov-border-light transition-all cursor-pointer"
                        onClick={() => { setPreviousTab('overview'); setActiveTab('mcp-servers'); setSelectedMCPServer(s); }}
                      >
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-sm font-semibold text-gov-text truncate max-w-[150px]">{s.name}</span>
                          <div
                            className="w-9 h-9 rounded-full flex items-center justify-center border-2 font-black text-sm tabular-nums"
                            style={{ borderColor: scoreColor, color: scoreColor, backgroundColor: `${scoreColor}10` }}
                          >
                            {s.score}
                          </div>
                        </div>
                        <div className="flex items-center gap-2 text-xs text-gov-text-3 mb-2">
                          <span className="font-mono px-1.5 py-0.5 bg-gov-surface rounded">{s.source === 'KagentMCPServer' ? 'MCPServer' : 'RemoteMCPServer'}</span>
                          <span>{s.namespace}</span>
                        </div>
                        <div className="flex gap-1.5 flex-wrap">
                          {s.routedThroughGateway ? (
                            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-green-500/15 text-green-400 font-semibold">Routed</span>
                          ) : (
                            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/15 text-red-400 font-semibold">Exposed</span>
                          )}
                          {s.hasJWT && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-blue-500/15 text-blue-400 font-semibold">JWT</span>}
                          {s.hasTLS && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-purple-500/15 text-purple-400 font-semibold">TLS</span>}
                          {s.findings.length > 0 && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/15 text-red-400 font-semibold">{s.findings.length} findings</span>}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Score Explanation — explains WHY score is 41 / Grade D */}
            <ScoreExplainer
              score={data.score.score}
              grade={data.score.grade}
              categories={data.score.categories || []}
              explanation={data.score.explanation || ''}
              findings={data.findings.findings || []}
              severityPenalties={data.score.severityPenalties}
            />

            {/* AI Governance Analysis */}
            <AIScoreCard />

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
          <FindingsTable findings={mcpAggregatedFindings} />
        )}

        {/* Footer */}
        <footer className="border-t border-gov-border pt-4 pb-8">
          <div className="flex items-center justify-between text-xs text-gov-text-3">
            <div className="flex items-center gap-2">
              <Shield className="w-3.5 h-3.5" />
              <span>MCP Security Governance Controller {version ? `${version}` : ''}</span>
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
