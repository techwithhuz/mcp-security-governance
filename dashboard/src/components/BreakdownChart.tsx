'use client';

import { ScoreBreakdown } from '@/lib/types';

interface BreakdownChartProps {
  breakdown: ScoreBreakdown;
}

const allCategories = [
  { key: 'agentGatewayScore', label: 'AgentGateway', shortLabel: 'AGW' },
  { key: 'authenticationScore', label: 'Authentication', shortLabel: 'Auth' },
  { key: 'authorizationScore', label: 'Authorization', shortLabel: 'Authz' },
  { key: 'corsScore', label: 'CORS', shortLabel: 'CORS' },
  { key: 'tlsScore', label: 'TLS', shortLabel: 'TLS' },
  { key: 'promptGuardScore', label: 'Prompt Guard', shortLabel: 'PG' },
  { key: 'rateLimitScore', label: 'Rate Limit', shortLabel: 'RL' },
  { key: 'toolScopeScore', label: 'Tool Scope', shortLabel: 'Tools' },
];

const getBarColor = (score: number) => {
  if (score >= 90) return '#22c55e';
  if (score >= 70) return '#eab308';
  if (score >= 50) return '#f97316';
  return '#ef4444';
};

export default function BreakdownChart({ breakdown }: BreakdownChartProps) {
  // Only show categories that are present in the breakdown data (i.e. required by policy)
  const categories = allCategories.filter(cat => (breakdown as any)[cat.key] !== undefined);

  return (
    <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
      <h2 className="text-lg font-bold mb-5">Score Breakdown</h2>
      <div className="space-y-4">
        {categories.map(cat => {
          const score = (breakdown as any)[cat.key] || 0;
          const color = getBarColor(score);

          return (
            <div key={cat.key}>
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-sm font-medium text-gov-text-2">{cat.label}</span>
                <span className="text-sm font-bold tabular-nums" style={{ color }}>
                  {score}
                </span>
              </div>
              <div className="w-full h-2.5 bg-gov-bg rounded-full overflow-hidden">
                <div
                  className="h-full rounded-full transition-all duration-1000 ease-out"
                  style={{
                    width: `${score}%`,
                    backgroundColor: color,
                    boxShadow: `0 0 8px ${color}40`,
                  }}
                />
              </div>
            </div>
          );
        })}
      </div>

      {/* Radar-like visual */}
      <div className="mt-6 pt-5 border-t border-gov-border">
        <div className="flex justify-center">
          <svg viewBox="0 0 200 200" width="180" height="180">
            {/* Background circles */}
            {[25, 50, 75, 100].map(r => (
              <circle
                key={r}
                cx="100"
                cy="100"
                r={r * 0.8}
                fill="none"
                stroke="#1e293b"
                strokeWidth="0.5"
              />
            ))}
            {/* Axis lines */}
            {categories.map((_, i) => {
              const angle = (Math.PI * 2 * i) / categories.length - Math.PI / 2;
              const x2 = 100 + Math.cos(angle) * 80;
              const y2 = 100 + Math.sin(angle) * 80;
              return (
                <line key={i} x1="100" y1="100" x2={x2} y2={y2} stroke="#1e293b" strokeWidth="0.5" />
              );
            })}
            {/* Data polygon */}
            <polygon
              points={categories
                .map((cat, i) => {
                  const score = (breakdown as any)[cat.key] || 0;
                  const angle = (Math.PI * 2 * i) / categories.length - Math.PI / 2;
                  const r = (score / 100) * 80;
                  return `${100 + Math.cos(angle) * r},${100 + Math.sin(angle) * r}`;
                })
                .join(' ')}
              fill="rgba(59, 130, 246, 0.15)"
              stroke="#3b82f6"
              strokeWidth="2"
            />
            {/* Data points */}
            {categories.map((cat, i) => {
              const score = (breakdown as any)[cat.key] || 0;
              const angle = (Math.PI * 2 * i) / categories.length - Math.PI / 2;
              const r = (score / 100) * 80;
              const x = 100 + Math.cos(angle) * r;
              const y = 100 + Math.sin(angle) * r;
              return (
                <circle key={i} cx={x} cy={y} r="3" fill="#3b82f6" stroke="#0a0e1a" strokeWidth="1.5" />
              );
            })}
            {/* Labels */}
            {categories.map((cat, i) => {
              const angle = (Math.PI * 2 * i) / categories.length - Math.PI / 2;
              const x = 100 + Math.cos(angle) * 95;
              const y = 100 + Math.sin(angle) * 95;
              return (
                <text
                  key={i}
                  x={x}
                  y={y}
                  textAnchor="middle"
                  dominantBaseline="middle"
                  fill="#64748b"
                  fontSize="8"
                  fontWeight="500"
                >
                  {cat.shortLabel}
                </text>
              );
            })}
          </svg>
        </div>
      </div>
    </div>
  );
}
