'use client';

import { useEffect, useState } from 'react';

interface ScoreGaugeProps {
  score: number;
  grade: string;
  phase: string;
}

export default function ScoreGauge({ score, grade, phase }: ScoreGaugeProps) {
  const [animatedScore, setAnimatedScore] = useState(0);

  useEffect(() => {
    const duration = 1500;
    const steps = 60;
    const increment = score / steps;
    let current = 0;
    const timer = setInterval(() => {
      current += increment;
      if (current >= score) {
        current = score;
        clearInterval(timer);
      }
      setAnimatedScore(Math.round(current));
    }, duration / steps);
    return () => clearInterval(timer);
  }, [score]);

  const radius = 90;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (animatedScore / 100) * circumference;

  const getScoreColor = (s: number) => {
    if (s >= 90) return '#22c55e';
    if (s >= 70) return '#eab308';
    if (s >= 50) return '#f97316';
    return '#ef4444';
  };

  const getGradeLabel = (g: string) => {
    switch (g) {
      case 'A': return 'Excellent';
      case 'B': return 'Good';
      case 'C': return 'Fair';
      case 'D': return 'Poor';
      case 'F': return 'Critical';
      default: return 'Unknown';
    }
  };

  const color = getScoreColor(animatedScore);

  return (
    <div className="flex flex-col items-center justify-center p-8 bg-gov-surface rounded-2xl border border-gov-border card-hover glow-accent">
      <h3 className="text-xs font-semibold uppercase tracking-wider text-gov-text-3 mb-3">Cluster Governance Score</h3>
      <div className="relative">
        <svg width="220" height="220" viewBox="0 0 220 220" className="transform -rotate-90">
          {/* Background circle */}
          <circle
            cx="110"
            cy="110"
            r={radius}
            fill="none"
            stroke="#1e293b"
            strokeWidth="12"
          />
          {/* Animated score arc */}
          <circle
            cx="110"
            cy="110"
            r={radius}
            fill="none"
            stroke={color}
            strokeWidth="12"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            className="score-ring"
            style={{ filter: `drop-shadow(0 0 8px ${color}40)` }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-6xl font-black tabular-nums" style={{ color }}>
            {animatedScore}
          </span>
          <span className="text-sm text-gov-text-2 font-medium mt-1">out of 100</span>
        </div>
      </div>
      <div className="mt-4 text-center">
        <div className="flex items-center justify-center gap-2">
          <span
            className="text-2xl font-bold px-3 py-1 rounded-lg"
            style={{ backgroundColor: `${color}15`, color }}
          >
            Grade {grade}
          </span>
        </div>
        <p className="text-gov-text-2 text-sm mt-2">{getGradeLabel(grade)}</p>
        <div className="mt-3 flex items-center gap-2">
          <span className="relative flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75"
              style={{ backgroundColor: color }} />
            <span className="relative inline-flex rounded-full h-2.5 w-2.5"
              style={{ backgroundColor: color }} />
          </span>
          <span className="text-xs text-gov-text-3 uppercase tracking-wider font-semibold">{phase}</span>
        </div>
      </div>
    </div>
  );
}
