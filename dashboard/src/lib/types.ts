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
  infraAbsent?: boolean;
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

// ---------- MCP Server-Centric Types ----------

export interface MCPServerScoreBreakdown {
  gatewayRouting: number;
  authentication: number;
  authorization: number;
  tls: number;
  cors: number;
  rateLimit: number;
  promptGuard: number;
  toolScope: number;
}

export interface ScoreExplanation {
  category: string;
  score: number;
  maxScore: number;
  status: 'pass' | 'partial' | 'fail' | 'not-required';
  reasons: string[];
  suggestions: string[];
  sources: string[];
}

export interface RelatedResource {
  kind: string;
  name: string;
  namespace: string;
  status: 'healthy' | 'warning' | 'critical' | 'missing';
  details?: Record<string, unknown>;
}

export interface MCPServerView {
  id: string;
  name: string;
  namespace: string;
  source: 'KagentMCPServer' | 'KagentRemoteMCPServer' | 'AgentgatewayBackendTarget' | 'Service';
  transport?: string;
  url?: string;
  port?: number;
  toolCount: number;
  toolNames: string[];
  effectiveToolCount: number;
  effectiveToolNames?: string[];
  hasToolRestriction: boolean;
  toolsByRoute?: Record<string, string[]>; // Route name -> allowed tools
  toolsByPolicy?: Record<string, Record<string, string[]>>; // Route name -> Policy name -> allowed tools
  pathTools?: Record<string, string[]>; // Path label (/ro, /rw) -> allowed tools

  relatedBackends: RelatedResource[];
  relatedPolicies: RelatedResource[];
  relatedRoutes: RelatedResource[];
  relatedGateways: RelatedResource[];
  relatedAgents: RelatedResource[];
  relatedServices: RelatedResource[];

  routedThroughGateway: boolean;
  hasTLS: boolean;
  hasAuth: boolean;
  hasJWT: boolean;
  jwtMode?: string;
  hasRBAC: boolean;
  hasCORS: boolean;
  hasRateLimit: boolean;
  hasPromptGuard: boolean;

  score: number;
  grade: string;
  status: 'compliant' | 'warning' | 'failing' | 'critical';
  findings: Finding[];
  scoreBreakdown: MCPServerScoreBreakdown;
  scoreExplanations?: ScoreExplanation[];
}

export interface MCPServerSummary {
  totalMCPServers: number;
  routedServers: number;
  unroutedServers: number;
  securedServers: number;
  atRiskServers: number;
  criticalServers: number;
  totalTools: number;
  exposedTools: number;
  averageScore: number;
}

export interface MCPServersResponse {
  servers: MCPServerView[];
  summary: MCPServerSummary;
}

// ---------- Verified Catalog Types (Inventory) ----------

export interface VerifiedCheck {
  id: string;
  name: string;
  category: string;
  passed: boolean;
  score: number;
  maxScore: number;
  description: string;
  detail: string;
}

export interface VerifiedFinding {
  severity: 'Critical' | 'High' | 'Medium' | 'Low';
  category: string;
  title: string;
  description: string;
  remediation: string;
}

export interface VerifiedScore {
  score: number;
  grade: string;
  status: 'Verified' | 'Unverified' | 'Rejected' | 'Pending';
  checks: VerifiedCheck[];
  findings: VerifiedFinding[];
  checksPassed: number;
  checksTotal: number;
  lastEvaluated: string;

  // Composite category scores (0–100)
  securityScore: number;
  trustScore: number;
  complianceScore: number;

  // Publisher trust sub-scores
  orgScore: number;
  publisherScore: number;
  verifiedOrg?: string;
  verifiedPublisher?: string;

  // Human-readable summary
  reason: string;

  // Flat convenience fields (set during transformation)
  resourceRef?: string;
  catalogName?: string;
  namespace?: string;
  mcpServerIDs?: string[];
  scoredAt?: string;
  toolNames?: string[];
  usedByAgents?: AgentUsage[];
}

export interface AgentUsage {
  name: string;
  namespace: string;
  toolNames?: string[];
}

export interface VerifiedResource {
  name: string;
  namespace: string;
  catalogName: string;
  title: string;
  description: string;
  version: string;
  sourceKind: string;
  sourceName: string;
  sourceNamespace: string;
  environment: string;
  cluster: string;
  published: boolean;
  deploymentReady: boolean;
  managementType: string;
  transport?: string;
  packageImage?: string;
  remoteURL?: string;
  toolNames?: string[];
  toolCount: number;
  usedByAgents?: AgentUsage[];
  verifiedScore: VerifiedScore;
  lastScored: string;
  resourceVersion: string;
}

export interface VerifiedSummary {
  totalCatalogs: number;
  totalScored: number;
  verifiedCount: number;
  unverifiedCount: number;
  rejectedCount: number;
  pendingCount: number;
  warningCount: number;
  criticalCount: number;
  averageScore: number;
  totalTools: number;
  totalAgentUsages: number;
  lastReconcile: string;
}

export interface VerifiedCatalogResponse {
  resources: VerifiedResource[];
  summary: VerifiedSummary;
}

// Shape expected by VerifiedCatalog component
export interface VerifiedInventory {
  items: VerifiedScore[];
  averageScore: number;
  totalScored: number;
  totalVerified: number;
  totalUnverified: number;
  totalRejected: number;
  totalPending: number;
}
