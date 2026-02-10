export interface Finding {
  id: string;
  severity: 'Critical' | 'High' | 'Medium' | 'Low';
  category: string;
  title: string;
  description: string;
  resource?: string;
  resourceRef?: string;
  namespace: string;
  impact: string;
  remediation: string;
}

export interface ScoreBreakdown {
  agentGatewayScore?: number;
  authenticationScore?: number;
  authorizationScore?: number;
  corsScore?: number;
  tlsScore?: number;
  promptGuardScore?: number;
  rateLimitScore?: number;
  toolScopeScore?: number;
}

export interface ScoreCategory {
  category: string;
  score: number;
  weight: number;
  weighted: number;
  status: 'passing' | 'warning' | 'failing' | 'critical';
}

export interface ScoreResponse {
  score: number;
  grade: string;
  phase: string;
  timestamp: string;
  categories: ScoreCategory[];
  explanation: string;
}

export interface ResourceDetail {
  resourceRef: string;
  kind: string;
  name: string;
  namespace: string;
  status: 'compliant' | 'critical' | 'failing' | 'warning' | 'info';
  score: number;
  findings: Finding[];
  critical: number;
  high: number;
  medium: number;
  low: number;
}

export interface ResourceDetailResponse {
  resources: ResourceDetail[];
  total: number;
}

export interface ResourceSummary {
  gatewaysFound: number;
  agentgatewayBackends: number;
  agentgatewayPolicies: number;
  httpRoutes: number;
  kagentAgents: number;
  kagentMCPServers: number;
  kagentRemoteMCPServers: number;
  compliantResources: number;
  nonCompliantResources: number;
  totalMCPEndpoints: number;
  exposedMCPEndpoints: number;
}

export interface NamespaceScore {
  namespace: string;
  score: number;
  findings: number;
}

export interface TrendPoint {
  timestamp: string;
  score: number;
  findings: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
}

export interface GovernanceData {
  score: number;
  grade: string;
  phase: string;
  findings: Finding[];
  resources: ResourceSummary;
  namespaces: NamespaceScore[];
  breakdown: ScoreBreakdown;
  trends: TrendPoint[];
}
