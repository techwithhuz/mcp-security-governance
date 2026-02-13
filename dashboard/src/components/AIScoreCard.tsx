'use client';

import { useEffect, useState } from 'react';
import { Brain, Sparkles, AlertTriangle, Lightbulb, ArrowRight, ChevronDown, ChevronUp, TrendingUp, TrendingDown, Minus, RefreshCw, Pause, Play } from 'lucide-react';

interface AIRisk {
  category: string;
  severity: string;
  description: string;
  impact: string;
}

interface AIScoreData {
  aiScore?: {
    score: number;
    grade: string;
    reasoning: string;
    risks: AIRisk[];
    suggestions: string[];
    timestamp: string;
  };
  available: boolean;
  enabled: boolean;
  message?: string;
  comparison?: {
    aiScore: number;
    aiGrade: string;
    algorithmicScore: number;
    algorithmicGrade: string;
    scoreDifference: number;
  };
  scanConfig?: {
    scanInterval: string;
    scanPaused: boolean;
  };
}

export default function AIScoreCard() {
  const [data, setData] = useState<AIScoreData | null>(null);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [toggling, setToggling] = useState(false);

  const fetchAIScore = async () => {
    try {
      const resp = await fetch('/api/governance/ai-score');
      const json = await resp.json();
      setData(json);
    } catch {
      // silently fail — AI score is optional
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAIScore();
    const interval = setInterval(fetchAIScore, 30000);
    return () => clearInterval(interval);
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await fetch('/api/governance/ai-score/refresh', { method: 'POST' });
      // Wait a moment then re-fetch to show updated state
      setTimeout(async () => {
        await fetchAIScore();
        setRefreshing(false);
      }, 3000);
    } catch {
      setRefreshing(false);
    }
  };

  const handleToggle = async () => {
    setToggling(true);
    try {
      await fetch('/api/governance/ai-score/toggle', { method: 'POST' });
      await fetchAIScore();
    } catch {
      // silently fail
    } finally {
      setToggling(false);
    }
  };

  if (loading) {
    return (
      <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 animate-pulse">
        <div className="flex items-center gap-2 mb-3">
          <div className="w-5 h-5 bg-gov-bg rounded" />
          <div className="h-4 w-24 bg-gov-bg rounded" />
        </div>
        <div className="h-8 w-16 bg-gov-bg rounded mb-2" />
        <div className="h-3 w-full bg-gov-bg rounded" />
      </div>
    );
  }

  if (!data || !data.enabled) return null;

  // Show error / loading state when enabled but not yet available
  if (!data.available || !data.aiScore || !data.comparison) {
    const scanPaused = data.scanConfig?.scanPaused ?? false;
    return (
      <div className="bg-gov-surface rounded-2xl border border-purple-500/20 overflow-hidden">
        <div className="bg-gradient-to-r from-purple-500/10 via-blue-500/10 to-cyan-500/10 px-5 py-3 border-b border-purple-500/10">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-purple-500/15">
              <Brain className="w-4 h-4 text-purple-400" />
            </div>
            <span className="text-sm font-bold text-purple-300">AI Governance Analysis</span>
            <Sparkles className="w-3.5 h-3.5 text-purple-400/60" />
            <div className="ml-auto flex items-center gap-2">
              <button
                onClick={handleRefresh}
                disabled={refreshing}
                className="p-1.5 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/20 text-purple-400 transition-all disabled:opacity-50"
                title="Run AI analysis now"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
              </button>
              <button
                onClick={handleToggle}
                disabled={toggling}
                className={`p-1.5 rounded-lg border transition-all disabled:opacity-50 ${
                  scanPaused
                    ? 'bg-yellow-500/10 hover:bg-yellow-500/20 border-yellow-500/20 text-yellow-400'
                    : 'bg-green-500/10 hover:bg-green-500/20 border-green-500/20 text-green-400'
                }`}
                title={scanPaused ? 'Resume periodic scanning' : 'Pause periodic scanning'}
              >
                {scanPaused ? <Play className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>
        </div>
        <div className="p-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-yellow-500/10 border border-yellow-500/20 flex items-center justify-center">
              <AlertTriangle className="w-5 h-5 text-yellow-400" />
            </div>
            <div>
              <p className="text-sm font-semibold text-gov-text">Waiting for AI Analysis</p>
              <p className="text-xs text-gov-text-3 mt-0.5">
                {data.message || 'The AI agent is initializing or waiting for API quota to reset. Results will appear automatically.'}
              </p>
            </div>
          </div>
          {data.scanConfig && (
            <div className="flex items-center gap-3 mt-3 text-[10px] text-gov-text-3">
              <span>Scan interval: <span className="font-mono text-gov-text-2">{data.scanConfig.scanInterval}</span></span>
              <span>•</span>
              <span className={scanPaused ? 'text-yellow-400' : 'text-green-400'}>
                {scanPaused ? '⏸ Paused' : '▶ Active'}
              </span>
            </div>
          )}
        </div>
      </div>
    );
  }

  const ai = data.aiScore;
  const cmp = data.comparison;

  const getScoreColor = (s: number) => {
    if (s >= 90) return '#22c55e';
    if (s >= 70) return '#eab308';
    if (s >= 50) return '#f97316';
    return '#ef4444';
  };

  const getSeverityColor = (sev: string) => {
    switch (sev) {
      case 'Critical': return 'text-red-400 bg-red-500/10 border-red-500/20';
      case 'High': return 'text-orange-400 bg-orange-500/10 border-orange-500/20';
      case 'Medium': return 'text-yellow-400 bg-yellow-500/10 border-yellow-500/20';
      case 'Low': return 'text-green-400 bg-green-500/10 border-green-500/20';
      default: return 'text-gov-text-3 bg-gov-bg border-gov-border';
    }
  };

  const scoreColor = getScoreColor(ai.score);
  const diffAbs = Math.abs(cmp.scoreDifference);
  const DiffIcon = cmp.scoreDifference > 0 ? TrendingUp : cmp.scoreDifference < 0 ? TrendingDown : Minus;
  const scanPaused = data.scanConfig?.scanPaused ?? false;

  return (
    <div className="bg-gov-surface rounded-2xl border border-purple-500/20 overflow-hidden">
      {/* Header strip */}
      <div className="bg-gradient-to-r from-purple-500/10 via-blue-500/10 to-cyan-500/10 px-5 py-3 border-b border-purple-500/10">
        <div className="flex items-center gap-2">
          <div className="p-1.5 rounded-lg bg-purple-500/15">
            <Brain className="w-4 h-4 text-purple-400" />
          </div>
          <span className="text-sm font-bold text-purple-300">AI Governance Analysis</span>
          <Sparkles className="w-3.5 h-3.5 text-purple-400/60" />
          <div className="ml-auto flex items-center gap-2">
            {data.scanConfig && (
              <span className="text-[10px] text-gov-text-3 font-mono mr-1">
                {data.scanConfig.scanInterval} {scanPaused ? '⏸' : '▶'}
              </span>
            )}
            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="p-1.5 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/20 text-purple-400 transition-all disabled:opacity-50"
              title="Run AI analysis now"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={handleToggle}
              disabled={toggling}
              className={`p-1.5 rounded-lg border transition-all disabled:opacity-50 ${
                scanPaused
                  ? 'bg-yellow-500/10 hover:bg-yellow-500/20 border-yellow-500/20 text-yellow-400'
                  : 'bg-green-500/10 hover:bg-green-500/20 border-green-500/20 text-green-400'
              }`}
              title={scanPaused ? 'Resume periodic scanning' : 'Pause periodic scanning'}
            >
              {scanPaused ? <Play className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />}
            </button>
          </div>
        </div>
      </div>

      <div className="p-5">
        {/* Score + Comparison Row */}
        <div className="flex items-start gap-6 mb-4">
          {/* AI Score */}
          <div className="flex-shrink-0">
            <div
              className="w-20 h-20 rounded-2xl flex flex-col items-center justify-center border-2"
              style={{
                borderColor: scoreColor,
                backgroundColor: `${scoreColor}08`,
              }}
            >
              <span className="text-3xl font-black tabular-nums" style={{ color: scoreColor }}>
                {ai.score}
              </span>
              <span className="text-[10px] font-bold tracking-wider" style={{ color: scoreColor }}>
                {ai.grade}
              </span>
            </div>
            <p className="text-[10px] text-gov-text-3 text-center mt-1.5 font-medium">AI Score</p>
          </div>

          {/* Comparison */}
          <div className="flex-1 space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-gov-bg border border-gov-border">
                <span className="text-xs text-gov-text-3">Algorithmic</span>
                <span className="text-sm font-black tabular-nums" style={{ color: getScoreColor(cmp.algorithmicScore) }}>
                  {cmp.algorithmicScore}
                </span>
              </div>
              <ArrowRight className="w-4 h-4 text-gov-text-3" />
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-purple-500/5 border border-purple-500/20">
                <span className="text-xs text-purple-300">AI</span>
                <span className="text-sm font-black tabular-nums" style={{ color: scoreColor }}>
                  {ai.score}
                </span>
              </div>
              {diffAbs > 0 && (
                <div className="flex items-center gap-1 text-xs text-gov-text-3">
                  <DiffIcon className="w-3.5 h-3.5" />
                  <span className="tabular-nums">{diffAbs}pt diff</span>
                </div>
              )}
            </div>

            {/* AI Reasoning */}
            <p className="text-xs text-gov-text-2 leading-relaxed">
              {ai.reasoning}
            </p>
          </div>
        </div>

        {/* Expand / Collapse */}
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full flex items-center justify-center gap-2 py-2 rounded-xl bg-gov-bg border border-gov-border hover:border-gov-border-light text-xs text-gov-text-3 hover:text-gov-text-2 transition-all"
        >
          {expanded ? (
            <>Hide Details <ChevronUp className="w-3.5 h-3.5" /></>
          ) : (
            <>View Risks & Suggestions <ChevronDown className="w-3.5 h-3.5" /></>
          )}
        </button>

        {expanded && (
          <div className="mt-4 space-y-4 animate-in fade-in slide-in-from-top-2 duration-200">
            {/* Risks */}
            {ai.risks && ai.risks.length > 0 && (
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <AlertTriangle className="w-4 h-4 text-red-400" />
                  <h4 className="text-sm font-bold text-gov-text">Identified Risks</h4>
                </div>
                <div className="space-y-2">
                  {ai.risks.map((risk, i) => (
                    <div
                      key={i}
                      className={`rounded-xl border p-3 ${getSeverityColor(risk.severity)}`}
                    >
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-xs font-bold uppercase tracking-wider">{risk.severity}</span>
                        <span className="text-xs font-semibold opacity-80">— {risk.category}</span>
                      </div>
                      <p className="text-xs opacity-90 leading-relaxed">{risk.description}</p>
                      <p className="text-[11px] opacity-60 mt-1">
                        <span className="font-semibold">Impact:</span> {risk.impact}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Suggestions */}
            {ai.suggestions && ai.suggestions.length > 0 && (
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <Lightbulb className="w-4 h-4 text-yellow-400" />
                  <h4 className="text-sm font-bold text-gov-text">AI Suggestions</h4>
                </div>
                <div className="space-y-2">
                  {ai.suggestions.map((s, i) => (
                    <div
                      key={i}
                      className="flex gap-3 rounded-xl border border-gov-border bg-gov-bg p-3"
                    >
                      <span className="flex-shrink-0 w-5 h-5 rounded-full bg-blue-500/15 text-blue-400 flex items-center justify-center text-[10px] font-black">
                        {i + 1}
                      </span>
                      <p className="text-xs text-gov-text-2 leading-relaxed">{s}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Timestamp */}
            <p className="text-[10px] text-gov-text-3 text-right">
              Analyzed at {new Date(ai.timestamp).toLocaleString()}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
