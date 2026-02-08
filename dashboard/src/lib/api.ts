const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

async function fetchAPI<T>(endpoint: string): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    cache: 'no-store',
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

export async function getScore(): Promise<{ score: number; grade: string; phase: string; categories: any[]; explanation: string }> {
  return fetchAPI('/api/governance/score');
}

export async function getFindings(): Promise<{ findings: any[]; total: number; bySeverity: Record<string, number> }> {
  return fetchAPI('/api/governance/findings');
}

export async function getResources(): Promise<any> {
  return fetchAPI('/api/governance/resources');
}

export async function getNamespaces(): Promise<{ namespaces: any[] }> {
  return fetchAPI('/api/governance/namespaces');
}

export async function getBreakdown(): Promise<any> {
  return fetchAPI('/api/governance/breakdown');
}

export async function getTrends(): Promise<{ trends: any[] }> {
  return fetchAPI('/api/governance/trends');
}

export async function getFullEvaluation(): Promise<any> {
  return fetchAPI('/api/governance/evaluation');
}

export async function getResourceDetail(): Promise<{ resources: any[]; total: number }> {
  return fetchAPI('/api/governance/resources/detail');
}
