'use client';

import { Shield, Server, Route, Bot, Plug, Globe, Lock, Zap } from 'lucide-react';

interface ResourceSummary {
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

interface ResourceCardsProps {
  resources: ResourceSummary;
}

const resourceItems = [
  { key: 'gatewaysFound', label: 'Agent Gateways', icon: Shield, color: '#3b82f6', group: 'agentgateway' },
  { key: 'agentgatewayBackends', label: 'AGW Backends', icon: Server, color: '#6366f1', group: 'agentgateway' },
  { key: 'agentgatewayPolicies', label: 'AGW Policies', icon: Lock, color: '#8b5cf6', group: 'agentgateway' },
  { key: 'httpRoutes', label: 'HTTP Routes', icon: Route, color: '#a855f7', group: 'agentgateway' },
  { key: 'kagentAgents', label: 'Kagent Agents', icon: Bot, color: '#ec4899', group: 'kagent' },
  { key: 'kagentMCPServers', label: 'MCP Servers', icon: Plug, color: '#f43f5e', group: 'kagent' },
  { key: 'kagentRemoteMCPServers', label: 'Remote MCP', icon: Globe, color: '#f97316', group: 'kagent' },
  { key: 'totalMCPEndpoints', label: 'MCP Endpoints', icon: Zap, color: '#eab308', group: 'kagent' },
];

export default function ResourceCards({ resources }: ResourceCardsProps) {
  const total = resources.compliantResources + resources.nonCompliantResources;
  const complianceRate = total > 0 ? Math.round((resources.compliantResources / total) * 100) : 0;

  return (
    <div className="space-y-4">
      {/* Compliance summary bar */}
      <div className="bg-gov-surface rounded-2xl border border-gov-border p-5 card-hover">
        <div className="flex items-center justify-between mb-3">
          <span className="text-sm font-semibold text-gov-text-2">Resource Compliance</span>
          <span className="text-2xl font-bold text-gov-text">{complianceRate}%</span>
        </div>
        <div className="w-full h-3 bg-gov-bg rounded-full overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-1000 ease-out"
            style={{
              width: `${complianceRate}%`,
              background: `linear-gradient(90deg, #22c55e, ${complianceRate >= 70 ? '#22c55e' : complianceRate >= 50 ? '#eab308' : '#ef4444'})`,
            }}
          />
        </div>
        <div className="flex justify-between mt-2 text-xs text-gov-text-3">
          <span className="text-green-400">{resources.compliantResources} compliant</span>
          <span className="text-red-400">{resources.nonCompliantResources} non-compliant</span>
        </div>
      </div>

      {/* AgentGateway Resources */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gov-text-3 mb-2 px-1">
          AgentGateway Resources
        </h3>
        <div className="grid grid-cols-2 gap-3">
          {resourceItems.filter(r => r.group === 'agentgateway').map((item) => {
            const count = (resources as any)[item.key] || 0;
            const Icon = item.icon;
            return (
              <div
                key={item.key}
                className="bg-gov-surface rounded-xl border border-gov-border p-4 card-hover"
              >
                <div className="flex items-center gap-3">
                  <div
                    className="p-2 rounded-lg"
                    style={{ backgroundColor: `${item.color}15` }}
                  >
                    <Icon size={18} style={{ color: item.color }} />
                  </div>
                  <div>
                    <div className="text-2xl font-bold">{count}</div>
                    <div className="text-xs text-gov-text-3">{item.label}</div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Kagent Resources */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gov-text-3 mb-2 px-1">
          Kagent Resources
        </h3>
        <div className="grid grid-cols-2 gap-3">
          {resourceItems.filter(r => r.group === 'kagent').map((item) => {
            const count = (resources as any)[item.key] || 0;
            const Icon = item.icon;
            return (
              <div
                key={item.key}
                className="bg-gov-surface rounded-xl border border-gov-border p-4 card-hover"
              >
                <div className="flex items-center gap-3">
                  <div
                    className="p-2 rounded-lg"
                    style={{ backgroundColor: `${item.color}15` }}
                  >
                    <Icon size={18} style={{ color: item.color }} />
                  </div>
                  <div>
                    <div className="text-2xl font-bold">{count}</div>
                    <div className="text-xs text-gov-text-3">{item.label}</div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
