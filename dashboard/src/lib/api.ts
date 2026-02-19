import {
  ScoreResponse,
  Finding,
  ResourceSummary,
  ResourceDetailResponse,
  NamespaceScore,
  ScoreBreakdown,
  TrendPoint,
  GovernanceData,
  MCPServersResponse,
  MCPServerView,
  MCPServerSummary,
  VerifiedCatalogResponse,
  VerifiedSummary,
  VerifiedResource,
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

export async function getMCPServers(): Promise<MCPServersResponse> {
  return fetchAPI('/api/governance/mcp-servers');
}

export async function getMCPServerSummary(): Promise<MCPServerSummary> {
  return fetchAPI('/api/governance/mcp-servers/summary');
}

export async function getMCPServerDetail(id: string): Promise<MCPServerView> {
  return fetchAPI(`/api/governance/mcp-servers/detail?id=${encodeURIComponent(id)}`);
}

// ---------- Verified Catalog (Inventory) ----------

export async function getVerifiedCatalog(): Promise<VerifiedCatalogResponse> {
  return fetchAPI('/api/governance/inventory/verified');
}

export async function getVerifiedSummary(): Promise<VerifiedSummary> {
  return fetchAPI('/api/governance/inventory/summary');
}

export async function getVerifiedDetail(namespace: string, name: string): Promise<VerifiedResource> {
  return fetchAPI(`/api/governance/inventory/detail?namespace=${encodeURIComponent(namespace)}&name=${encodeURIComponent(name)}`);
}
