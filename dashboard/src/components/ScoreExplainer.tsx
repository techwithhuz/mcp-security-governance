'use client';

import { useState } from 'react';
import { Info, CheckCircle2, AlertTriangle, AlertCircle, XCircle } from 'lucide-react';
import CategoryScoreModal from './CategoryScoreModal';

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

interface ScoreExplainerProps {
  score: number;
  grade: string;
  categories: ScoreCategory[];
  explanation: string;
  findings?: Finding[];
  severityPenalties?: { Critical: number; High: number; Medium: number; Low: number };
}

// Map display category names to finding category values
const categoryToFindingCategory: Record<string, string[]> = {
  'AgentGateway Compliance': ['AgentGateway'],
  'Authentication': ['Authentication'],
  'Authorization': ['Authorization'],
  'CORS': ['CORS'],
  'TLS': ['TLS'],
  'Prompt Guard': ['PromptGuard'],
  'Rate Limit': ['RateLimit'],
  'Tool Scope': ['ToolScope'],
};

const statusConfig: Record<string, { icon: typeof CheckCircle2; color: string; bg: string; border: string }> = {
  passing: { icon: CheckCircle2, color: '#22c55e', bg: 'bg-green-500/10', border: 'border-green-500/30' },
  warning: { icon: AlertTriangle, color: '#eab308', bg: 'bg-yellow-500/10', border: 'border-yellow-500/30' },
  failing: { icon: AlertCircle, color: '#f97316', bg: 'bg-orange-500/10', border: 'border-orange-500/30' },
  critical: { icon: XCircle, color: '#ef4444', bg: 'bg-red-500/10', border: 'border-red-500/30' },
};

export default function ScoreExplainer({ score, grade, categories, explanation, findings = [], severityPenalties }: ScoreExplainerProps) {
  const totalWeighted = categories.reduce((sum, c) => sum + c.weighted, 0);
  const [selectedCategory, setSelectedCategory] = useState<ScoreCategory | null>(null);

  // Default penalties if not provided (matches Go defaults)
  const penalties = severityPenalties || { Critical: 40, High: 25, Medium: 15, Low: 5 };

  // Get findings for a specific category
  const getFindingsForCategory = (cat: ScoreCategory): Finding[] => {
    const findingCategories = categoryToFindingCategory[cat.category] || [];
    return findings.filter(f => findingCategories.includes(f.category));
  };

  return (
    <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
      {/* Header */}
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 rounded-lg bg-blue-500/10">
          <Info size={16} className="text-blue-400" />
        </div>
        <h2 className="text-lg font-bold">How is the Score Calculated?</h2>
      </div>

      {/* Explanation text */}
      <p className="text-sm text-gov-text-3 mb-5 leading-relaxed">{explanation}</p>

      {/* Category breakdown table */}
      <div className="space-y-2">
        {/* Table header */}
        <div className="grid grid-cols-12 gap-2 px-3 py-2 text-xs font-semibold uppercase tracking-wider text-gov-text-3">
          <div className="col-span-4">Category</div>
          <div className="col-span-2 text-center">Score</div>
          <div className="col-span-2 text-center">Weight</div>
          <div className="col-span-2 text-center">Contribution</div>
          <div className="col-span-2 text-center">Status</div>
        </div>

        {/* Category rows */}
        {categories.map((cat) => {
          const config = statusConfig[cat.status] || statusConfig.critical;
          const Icon = config.icon;
          const catFindings = getFindingsForCategory(cat);

          return (
            <div
              key={cat.category}
              onClick={() => setSelectedCategory(cat)}
              className={`grid grid-cols-12 gap-2 items-center px-3 py-3 rounded-xl border transition-all hover:scale-[1.01] cursor-pointer ${config.bg} ${config.border}`}
              title={`Click to see how this score is calculated (${catFindings.length} finding${catFindings.length !== 1 ? 's' : ''})`}
            >
              {/* Category name */}
              <div className="col-span-4 flex items-center gap-2">
                <Icon size={14} style={{ color: config.color }} />
                <span className="text-sm font-medium text-gov-text">{cat.category}</span>
              </div>

              {/* Raw score */}
              <div className="col-span-2 text-center">
                <span className="text-sm font-bold tabular-nums" style={{ color: config.color }}>
                  {cat.score}
                </span>
                <span className="text-xs text-gov-text-3">/100</span>
              </div>

              {/* Weight */}
              <div className="col-span-2 text-center">
                <span className="text-sm font-semibold text-gov-text-2">{cat.weight}%</span>
              </div>

              {/* Weighted contribution */}
              <div className="col-span-2 text-center">
                <span className="text-sm font-bold tabular-nums" style={{ color: config.color }}>
                  {cat.weighted.toFixed(1)}
                </span>
                <span className="text-xs text-gov-text-3">pts</span>
              </div>

              {/* Status badge */}
              <div className="col-span-2 text-center">
                <span
                  className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold uppercase"
                  style={{ backgroundColor: `${config.color}15`, color: config.color }}
                >
                  {cat.status}
                </span>
              </div>
            </div>
          );
        })}

        {/* Total row */}
        <div className="grid grid-cols-12 gap-2 items-center px-3 py-3 rounded-xl bg-gov-bg border border-gov-border mt-2">
          <div className="col-span-4 flex items-center gap-2">
            <span className="text-sm font-bold text-gov-text">Final Score</span>
          </div>
          <div className="col-span-2 text-center">
            <span className="text-lg font-black tabular-nums text-gov-accent">{score}</span>
            <span className="text-xs text-gov-text-3">/100</span>
          </div>
          <div className="col-span-2 text-center">
            <span className="text-sm font-semibold text-gov-text-2">100%</span>
          </div>
          <div className="col-span-2 text-center">
            <span className="text-lg font-black tabular-nums text-gov-accent">
              {totalWeighted.toFixed(1)}
            </span>
          </div>
          <div className="col-span-2 text-center">
            <span className="text-lg font-black" style={{
              color: score >= 90 ? '#22c55e' : score >= 70 ? '#eab308' : score >= 50 ? '#f97316' : '#ef4444'
            }}>
              {grade}
            </span>
          </div>
        </div>
      </div>

      {/* Visual bar showing contribution */}
      <div className="mt-5 pt-4 border-t border-gov-border">
        <div className="text-xs font-semibold text-gov-text-3 uppercase tracking-wider mb-2">
          Score Composition
        </div>
        <div className="flex h-4 rounded-full overflow-hidden bg-gov-bg">
          {categories.map((cat, i) => {
            const config = statusConfig[cat.status] || statusConfig.critical;
            const widthPercent = (cat.weighted / 100) * 100;
            return (
              <div
                key={cat.category}
                className="h-full relative group"
                style={{
                  width: `${Math.max(widthPercent, 0.5)}%`,
                  backgroundColor: config.color,
                  opacity: 0.8,
                }}
                title={`${cat.category}: ${cat.weighted.toFixed(1)}pts`}
              >
                {/* Tooltip on hover */}
                <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block z-10">
                  <div className="bg-gov-bg border border-gov-border rounded-lg px-2 py-1 text-xs whitespace-nowrap shadow-xl">
                    <span className="font-semibold">{cat.category}</span>: {cat.weighted.toFixed(1)}pts
                  </div>
                </div>
              </div>
            );
          })}
          {/* Remaining gap to 100 */}
          <div
            className="h-full bg-gov-border/30"
            style={{ width: `${Math.max(100 - (totalWeighted / 100) * 100, 0)}%` }}
          />
        </div>
        <div className="flex justify-between mt-1 text-xs text-gov-text-3">
          <span>0</span>
          <span className="font-semibold text-gov-text-2">{totalWeighted.toFixed(1)} / 100</span>
          <span>100</span>
        </div>
      </div>

      {/* Click hint */}
      <div className="mt-4 flex items-center gap-2 text-xs text-gov-text-3">
        <Info size={12} />
        <span>Click on any category row above to see detailed score calculation and findings</span>
      </div>

      {/* Category Score Modal */}
      {selectedCategory && (
        <CategoryScoreModal
          category={selectedCategory}
          findings={getFindingsForCategory(selectedCategory)}
          severityPenalties={penalties}
          onClose={() => setSelectedCategory(null)}
        />
      )}
    </div>
  );
}
