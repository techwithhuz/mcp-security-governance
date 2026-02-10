import {
  ScoreResponse,
  Finding,
  ResourceSummary,
  ResourceDetailResponse,
  NamespaceScore,
  ScoreBreakdown,
  TrendPoint,
  GovernanceData,
} from './types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

async function fetchAPI<T>(endpoint: string): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    cache: 'no-store',
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

export async function getScore(): Promise<ScoreResponse> {
  return fetchAPI('/api/governance/score');
}

export async function getFindings(): Promise<{ findings: Finding[]; total: number; bySeverity: Record<string, number> }> {
  return fetchAPI('/api/governance/findings');
}

export async function getResources(): Promise<ResourceSummary> {
  return fetchAPI('/api/governance/resources');
}

export async function getNamespaces(): Promise<{ namespaces: NamespaceScore[] }> {
  return fetchAPI('/api/governance/namespaces');
}

export async function getBreakdown(): Promise<ScoreBreakdown> {
  return fetchAPI('/api/governance/breakdown');
}

export async function getTrends(): Promise<{ trends: TrendPoint[] }> {
  return fetchAPI('/api/governance/trends');
}

export async function getFullEvaluation(): Promise<GovernanceData> {
  return fetchAPI('/api/governance/evaluation');
}

export async function getResourceDetail(): Promise<ResourceDetailResponse> {
  return fetchAPI('/api/governance/resources/detail');
}
