'use client';

import { NamespaceScore } from '@/lib/types';

interface NamespaceHeatmapProps {
  namespaces: NamespaceScore[];
}

const getScoreColor = (score: number) => {
  if (score >= 90) return { bg: 'rgba(34, 197, 94, 0.2)', border: 'rgba(34, 197, 94, 0.4)', text: '#22c55e' };
  if (score >= 70) return { bg: 'rgba(234, 179, 8, 0.2)', border: 'rgba(234, 179, 8, 0.4)', text: '#eab308' };
  if (score >= 50) return { bg: 'rgba(249, 115, 22, 0.2)', border: 'rgba(249, 115, 22, 0.4)', text: '#f97316' };
  return { bg: 'rgba(239, 68, 68, 0.2)', border: 'rgba(239, 68, 68, 0.4)', text: '#ef4444' };
};

export default function NamespaceHeatmap({ namespaces }: NamespaceHeatmapProps) {
  const sorted = [...namespaces].sort((a, b) => a.score - b.score);

  return (
    <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
      <h2 className="text-lg font-bold mb-4">Namespace Compliance</h2>
      <div className="space-y-3">
        {sorted.map(ns => {
          const colors = getScoreColor(ns.score);
          return (
            <div
              key={ns.namespace}
              className="rounded-xl p-4 border transition-all hover:scale-[1.01]"
              style={{ backgroundColor: colors.bg, borderColor: colors.border }}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-mono text-sm font-semibold text-gov-text">{ns.namespace}</span>
                <span className="text-xl font-black tabular-nums" style={{ color: colors.text }}>
                  {ns.score}
                </span>
              </div>
              <div className="flex gap-3 text-xs">
                <span className="flex items-center gap-1">
                  <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: colors.text }} />
                  <span style={{ color: colors.text }}>{ns.findings} finding{ns.findings !== 1 ? 's' : ''}</span>
                </span>
              </div>
              {/* Mini bar */}
              <div className="w-full h-1.5 bg-black/20 rounded-full mt-2 overflow-hidden">
                <div
                  className="h-full rounded-full"
                  style={{ width: `${ns.score}%`, backgroundColor: colors.text }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
