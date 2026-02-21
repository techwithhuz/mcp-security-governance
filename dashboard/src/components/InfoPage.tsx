'use client';

import { useState } from 'react';
import Image from 'next/image';
import {
  Shield, Server, Route, Lock, Zap, Bot, Network, Database,
  Eye, GitBranch, CheckCircle2, AlertTriangle, ArrowRight,
  Cpu, Layers, Search, FileKey, Blocks, ShieldCheck, ShieldAlert,
  Star, Activity, ChevronDown, ChevronUp, ExternalLink, Info,
  BarChart3, RefreshCw, Scan, ArrowDown, Wrench, Github, Download, Settings
} from 'lucide-react';

const SCORING_CONTROLS = [
  {
    key: 'Gateway Routing',
    icon: Route,
    color: '#3b82f6',
    maxScore: 20,
    description: 'Is the MCP server routed through a Gateway API (Agentgateway)?',
    pass: 'Server is exposed through a governed gateway (+20)',
    fail: 'Server is directly reachable with no gateway layer (0)',
    why: 'Without a gateway, there is no central enforcement point. Any tool call can reach the server unfiltered.',
  },
  {
    key: 'Authentication',
    icon: FileKey,
    color: '#6366f1',
    maxScore: 20,
    description: 'Is JWT or mutual-TLS authentication enforced on incoming requests?',
    pass: 'JWT Strict mode enforced on the route (+20)',
    fail: 'No authentication required — anyone can call tools (0)',
    partial: 'JWT present but in permissive mode (+10)',
    why: 'Unauthenticated MCP servers let any agent or script invoke powerful tools without identity verification.',
  },
  {
    key: 'Authorization',
    icon: Lock,
    color: '#8b5cf6',
    maxScore: 15,
    description: 'Are tool calls filtered by an allow-list policy (mcp.tool.name in [...])?',
    pass: 'Tool allow-list enforced via AgentgatewayPolicy (+15)',
    fail: 'All tools exposed with no authorization filter (0)',
    why: 'Even authenticated agents should only access the specific tools they need (principle of least privilege).',
  },
  {
    key: 'TLS Encryption',
    icon: Shield,
    color: '#a855f7',
    maxScore: 15,
    description: 'Is TLS configured on backend connections (SNI / mutual-TLS)?',
    pass: 'TLS with SNI verification enabled (+15)',
    fail: 'Plaintext backend connection (0)',
    why: 'Without TLS, tool payloads and results can be intercepted in transit inside the cluster.',
  },
  {
    key: 'CORS Policy',
    icon: Blocks,
    color: '#ec4899',
    maxScore: 10,
    description: 'Is CORS configured to restrict browser-based cross-origin access?',
    pass: 'CORS policy restricts allowed origins (+10)',
    fail: 'No CORS policy — all origins allowed (0)',
    partial: 'CORS configured via attached policy (+8)',
    why: 'Without CORS, browser-based MCP clients (dashboards, agents) can be tricked into making tool calls from malicious origins.',
  },
  {
    key: 'Rate Limiting',
    icon: Zap,
    color: '#f59e0b',
    maxScore: 10,
    description: 'Is rate limiting applied to prevent API abuse or agent runaway loops?',
    pass: 'Rate limit policy applied (+10)',
    fail: 'No rate limit — unlimited tool calls allowed (0)',
    why: 'Agents in an agentic loop can make thousands of tool calls in seconds, causing service disruption or data exfiltration.',
  },
  {
    key: 'Prompt Guard',
    icon: ShieldCheck,
    color: '#10b981',
    maxScore: 10,
    description: 'Is AI prompt injection protection and sensitive data masking enabled?',
    pass: 'Prompt guard with injection detection + data masking (+10)',
    fail: 'No prompt protection (0)',
    partial: 'Partial prompt guard (request only or response only) (+5)',
    why: 'Prompt injection attacks trick AI agents into executing unauthorized actions. Sensitive data in responses (SSN, credit cards) must be masked.',
  },
  {
    key: 'Tool Scope',
    icon: Server,
    color: '#06b6d4',
    maxScore: 10,
    description: 'How tightly is the exposed tool surface restricted vs. all available tools?',
    pass: '≤25% of tools exposed (maximum restriction) (+10)',
    fail: '100% of tools exposed (no restriction) (0)',
    partial: 'Graduated score based on restriction percentage',
    why: 'Exposing all 50+ tools when an agent only needs 5 vastly increases the blast radius of a compromise.',
  },
];

const GRADE_THRESHOLDS = [
  { grade: 'A', min: 90, color: '#22c55e', label: 'Excellent' },
  { grade: 'B', min: 75, color: '#86efac', label: 'Good' },
  { grade: 'C', min: 60, color: '#eab308', label: 'Moderate Risk' },
  { grade: 'D', min: 40, color: '#f97316', label: 'High Risk' },
  { grade: 'F', min: 0, color: '#ef4444', label: 'Critical Risk' },
];

const ARCHITECTURE_LAYERS = [
  {
    layer: 'Dashboard & API',
    icon: BarChart3,
    color: '#6366f1',
    items: ['Next.js Dashboard', 'REST API Server', 'Real-time Updates'],
    description: 'User interface and API gateway for governance insights',
    position: 'top',
  },
  {
    layer: 'Governance Controller',
    icon: Cpu,
    color: '#0ea5e9',
    items: ['Resource Discovery', 'Scoring Engine', 'AI Agent', 'Evaluator'],
    description: 'Core brain that watches K8s, evaluates security, analyzes risks',
    position: 'middle',
  },
  {
    layer: 'Security Enforcement',
    icon: Shield,
    color: '#f59e0b',
    items: ['AgentGateway', 'HTTPRoute Rules', 'AgentgatewayPolicy', 'JWT & Rate Limit'],
    description: 'Active enforcement layer that gates and filters all MCP traffic',
    position: 'middle',
  },
  {
    layer: 'Kubernetes Resources',
    icon: Network,
    color: '#8b5cf6',
    items: ['MCP Servers', 'Agents', 'Gateways', 'Backends', 'Policies'],
    description: 'Native K8s resources that define the MCP ecosystem',
    position: 'bottom',
  },
];

const HOW_IT_WORKS = [
  {
    step: '01',
    title: 'Continuous Discovery',
    icon: Search,
    color: '#3b82f6',
    description: 'The controller watches Kubernetes for all MCP-related resources: RemoteMCPServers, AgentgatewayBackends, HTTPRoutes, Gateways, AgentgatewayPolicies, and kagent Agents. It reconciles every 30 seconds.',
  },
  {
    step: '02',
    title: 'Relationship Mapping',
    icon: GitBranch,
    color: '#6366f1',
    description: 'Resources are correlated: which backend serves which MCP server, which HTTPRoute targets which backend, which policy is attached to which route, and which agents consume which server.',
  },
  {
    step: '03',
    title: 'Security Evaluation',
    icon: ShieldAlert,
    color: '#8b5cf6',
    description: 'Each MCP server is evaluated across 8 security controls. Scores are calculated using weighted rubrics. Findings are generated for each failing control with specific remediation guidance.',
  },
  {
    step: '04',
    title: 'Tool Scope Analysis',
    icon: Layers,
    color: '#ec4899',
    description: 'The controller introspects AgentgatewayPolicies to find tool allow-lists per route path. It maps each HTTPRoute path (/ro, /rw, etc.) to its specific allowed tools from the actual HTTPRoute spec.',
  },
  {
    step: '05',
    title: 'AI-Powered Insights',
    icon: Cpu,
    color: '#f59e0b',
    description: 'Score explanations include human-readable reasons, improvement suggestions, and contributing resource references — making every score transparent and actionable.',
  },
  {
    step: '06',
    title: 'Live Dashboard',
    icon: Activity,
    color: '#10b981',
    description: 'Results are served via a REST API and rendered in this dashboard. Data refreshes every 30s automatically. You can also trigger an on-demand scan.',
  },
];

const COMPONENTS = [
  {
    name: 'Governance Controller',
    icon: Cpu,
    color: '#3b82f6',
    tech: 'Go 1.25 · Kubernetes client-go',
    description: 'The brain. Runs in-cluster, watches K8s resources, evaluates security posture, and serves the REST API on port 8090.',
    responsibilities: ['Resource discovery & correlation', 'Security scoring engine', 'Findings generation', 'REST API server'],
  },
  {
    name: 'Governance Dashboard',
    icon: BarChart3,
    color: '#6366f1',
    tech: 'Next.js 14 · TypeScript · Tailwind CSS',
    description: 'This UI. A Next.js app that proxies API calls to the controller and renders the posture data in real-time.',
    responsibilities: ['Real-time security posture view', 'MCP server deep-dive', 'Per-path tool visualization', 'Findings & remediation'],
  },
];

export default function InfoPage() {
  const [expandedControl, setExpandedControl] = useState<string | null>(null);

  return (
    <div className="space-y-16 pb-16">

      {/* ── Hero ── */}
      <div className="relative rounded-3xl overflow-hidden border border-gov-border bg-gov-surface">
        <div className="absolute inset-0 bg-gradient-to-br from-blue-500/5 via-indigo-500/5 to-purple-500/5" />
        <div className="absolute top-0 right-0 w-96 h-96 bg-blue-500/5 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2" />
        <div className="absolute bottom-0 left-0 w-64 h-64 bg-purple-500/5 rounded-full blur-3xl translate-y-1/2 -translate-x-1/4" />
        <div className="relative px-8 py-12 md:px-16 md:py-16">
          <div className="max-w-4xl">
            <div className="flex flex-col md:flex-row items-start md:items-center gap-8 mb-8">
              {/* Logo */}
              <div className="flex-shrink-0">
                <Image
                  src="/logo.svg"
                  alt="MCP-G Logo"
                  width={100}
                  height={100}
                  className="drop-shadow-lg"
                />
              </div>
              {/* Title */}
              <div>
                <div className="text-xs font-bold text-blue-400 uppercase tracking-widest mb-3">Kubernetes-Native Governance</div>
                <h1 className="text-3xl md:text-5xl font-black text-gov-text mb-2">
                  MCP-G<span className="text-gov-text-3"> (MCP Governance)</span>
                </h1>
                <p className="text-base text-gov-text-2">
                  <span className="bg-clip-text text-transparent bg-gradient-to-r from-purple-400 via-pink-400 to-red-400 font-bold">AI-Powered</span> security governance for Model Context Protocol infrastructure
                </p>
              </div>
            </div>
            <p className="text-lg text-gov-text-2 leading-relaxed mb-8">
              Monitors <a href="https://agentgateway.dev" target="_blank" rel="noopener noreferrer" className="text-blue-400 font-semibold hover:underline">AgentGateway</a> and <a href="https://kagent.dev" target="_blank" rel="noopener noreferrer" className="text-blue-400 font-semibold hover:underline">Kagent</a> resources, evaluates security posture with an <span className="font-bold text-blue-300">MCP-Server-centric scoring model</span>, and provides AI-powered governance insights at scale.
            </p>
            <div className="flex flex-wrap gap-3 mb-6">
              {[
                { icon: Eye, label: 'Full MCP Visibility' },
                { icon: Shield, label: 'Automated Security Scoring' },
                { icon: Layers, label: 'Path-Based Tool Control' },
                { icon: Bot, label: 'Agent Relationship Mapping' },
                { icon: Activity, label: 'Real-Time Monitoring' },
              ].map(({ icon: Icon, label }) => (
                <div key={label} className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-gov-bg border border-gov-border text-sm text-gov-text-2">
                  <Icon size={14} className="text-blue-400" />
                  {label}
                </div>
              ))}
            </div>
            <a
              href="https://github.com/techwithhuz/mcp-security-governance"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-blue-500/10 border border-blue-500/30 text-blue-300 hover:bg-blue-500/20 transition-all"
            >
              <Github size={16} />
              <span className="font-semibold">View on GitHub</span>
              <ExternalLink size={14} />
            </a>
          </div>
        </div>
      </div>

      {/* ── The Problem ── */}
      <section>
        <SectionHeader
          icon={AlertTriangle}
          color="#f97316"
          title="The Problem MCP-G Solves"
          subtitle="Why governing MCP servers is critical in agentic AI systems"
        />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-6">
          {[
            {
              icon: Eye,
              color: '#ef4444',
              title: 'Zero Visibility',
              body: 'AI agents silently call MCP tools in the background. Without governance, you have no idea which tools were called, by whom, with what data — until something goes wrong.',
            },
            {
              icon: Lock,
              color: '#f97316',
              title: 'Unconstrained Tool Access',
              body: 'MCP servers typically expose every tool to every caller. An agent that needs only "read dashboard" also gets access to "delete dashboard", "drop database", and beyond.',
            },
            {
              icon: ShieldAlert,
              color: '#eab308',
              title: 'Prompt Injection Risk',
              body: 'Malicious content in tool responses can hijack an AI agent\'s next actions. Without prompt guards, your agent can be tricked into calling destructive tools.',
            },
          ].map(({ icon: Icon, color, title, body }) => (
            <div key={title} className="bg-gov-surface rounded-2xl border border-gov-border p-6 hover:border-gov-border-light transition-all">
              <div className="p-2.5 rounded-xl mb-4 w-fit" style={{ backgroundColor: `${color}15` }}>
                <Icon size={20} style={{ color }} />
              </div>
              <h3 className="text-base font-bold text-gov-text mb-2">{title}</h3>
              <p className="text-sm text-gov-text-3 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* ── Prerequisites ── */}
      <section>
        <SectionHeader
          icon={CheckCircle2}
          color="#22c55e"
          title="Prerequisites"
          subtitle="Required components that must be installed before MCP-G"
        />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-6">
          {[
            {
              icon: Route,
              color: '#8b5cf6',
              title: 'AgentGateway',
              description: 'The enforcement plane that routes and secures all MCP traffic through policies, JWT authentication, rate limiting, and tool allow-lists.',
              link: 'https://agentgateway.dev/docs/kubernetes/latest/install/helm/',
              linkText: 'Install AgentGateway',
            },
            {
              icon: Bot,
              color: '#10b981',
              title: 'Kagent',
              description: 'The MCP agent platform that discovers, registers, and executes MCP servers. MCP-G monitors Kagent agents and MCPServer resources.',
              link: 'https://kagent.dev/docs/kagent/introduction/installation',
              linkText: 'Install Kagent',
            },
            {
              icon: Database,
              color: '#f59e0b',
              title: 'Agent Registry',
              description: 'A centralized catalog of verified MCP servers. MCP-G scores catalog entries and enables agents to use only approved servers.',
              link: 'https://github.com/den-vasyliev/agentregistry-inventory',
              linkText: 'Deploy Agent Registry',
            },
          ].map(({ icon: Icon, color, title, description, link, linkText }) => (
            <div key={title} className="bg-gov-surface rounded-2xl border border-gov-border p-6 hover:border-gov-border-light transition-all flex flex-col">
              <div className="p-2.5 rounded-xl mb-4 w-fit" style={{ backgroundColor: `${color}15` }}>
                <Icon size={20} style={{ color }} />
              </div>
              <h3 className="text-base font-bold text-gov-text mb-2">{title}</h3>
              <p className="text-sm text-gov-text-3 leading-relaxed mb-4 flex-grow">{description}</p>
              <a
                href={link}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 text-sm font-semibold text-blue-400 hover:text-blue-300 transition-colors"
              >
                {linkText}
                <ExternalLink size={12} />
              </a>
            </div>
          ))}
        </div>
      </section>

      {/* ── Installation ── */}
      <section>
        <SectionHeader
          icon={Download}
          color="#3b82f6"
          title="Installation"
          subtitle="Deploy MCP-G using Helm — fastest way to get started"
        />
        
        {/* Prerequisites Alert */}
        <div className="mt-6 p-5 bg-blue-500/10 border border-blue-500/30 rounded-2xl mb-6">
          <p className="text-sm text-gov-text-2">
            <strong>📋 Prerequisites:</strong> Ensure <strong>AgentGateway</strong>, <strong>Kagent</strong>, and <strong>Agent Registry</strong> are already installed in your cluster before deploying MCP-G.
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
          {/* Quick Install */}
          <div className="bg-gov-surface rounded-2xl border border-gov-border p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2.5 rounded-xl bg-green-500/20 border border-green-500/30">
                <Zap size={18} className="text-green-400" />
              </div>
              <h3 className="text-lg font-bold text-gov-text">Quick Start</h3>
            </div>
            <p className="text-sm text-gov-text-3 mb-4">Deploy MCP-G with default settings in one command:</p>
            <div className="bg-gov-bg rounded-lg border border-gov-border p-4 mb-4">
              <pre className="text-xs text-gov-text-2 font-mono overflow-x-auto whitespace-pre-wrap break-words">
helm repo add techwithhuz https://charts.techwithhuz.dev
helm repo update
helm install mcp-governance techwithhuz/mcp-governance \
  --create-namespace \
  --namespace mcp-governance
              </pre>
            </div>
            <p className="text-xs text-gov-text-3">
              <strong>Dashboard:</strong> <code className="bg-gov-bg px-2 py-1 rounded">http://localhost:3000</code>
            </p>
          </div>

          {/* Production Install */}
          <div className="bg-gov-surface rounded-2xl border border-gov-border p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2.5 rounded-xl bg-purple-500/20 border border-purple-500/30">
                <Layers size={18} className="text-purple-400" />
              </div>
              <h3 className="text-lg font-bold text-gov-text">Production Setup</h3>
            </div>
            <p className="text-sm text-gov-text-3 mb-4">Deploy with AI scoring and custom configuration:</p>
            <div className="bg-gov-bg rounded-lg border border-gov-border p-4 mb-4">
              <pre className="text-xs text-gov-text-2 font-mono overflow-x-auto whitespace-pre-wrap break-words">
helm install mcp-governance techwithhuz/mcp-governance \
  --create-namespace \
  --namespace mcp-governance \
  --set samples.install=true \
  --set aiAgent.enabled=true \
  --set aiAgent.provider=gemini
              </pre>
            </div>
            <p className="text-xs text-gov-text-3">
              Set <code className="bg-gov-bg px-2 py-1 rounded">GOOGLE_API_KEY</code> env var for Gemini AI
            </p>
          </div>
        </div>

        {/* Verification Section */}
        <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-gov-surface rounded-2xl border border-gov-border p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2.5 rounded-xl bg-blue-500/20 border border-blue-500/30">
                <CheckCircle2 size={18} className="text-blue-400" />
              </div>
              <h3 className="text-lg font-bold text-gov-text">Verify Installation</h3>
            </div>
            <div className="bg-gov-bg rounded-lg border border-gov-border p-4">
              <pre className="text-xs text-gov-text-2 font-mono overflow-x-auto whitespace-pre-wrap break-words">
{`# Check deployment status
kubectl get pods -n mcp-governance

# View dashboard
kubectl port-forward -n mcp-governance \\
  svc/mcp-governance-dashboard 3000:3000

# Check API health
curl http://localhost:8090/api/health`}
              </pre>
            </div>
          </div>

          <div className="bg-gov-surface rounded-2xl border border-gov-border p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2.5 rounded-xl bg-amber-500/20 border border-amber-500/30">
                <Settings size={18} className="text-amber-400" />
              </div>
              <h3 className="text-lg font-bold text-gov-text">Customize Installation</h3>
            </div>
            <p className="text-sm text-gov-text-3 mb-3">For advanced configuration, use a values file:</p>
            <div className="bg-gov-bg rounded-lg border border-gov-border p-4">
              <pre className="text-xs text-gov-text-2 font-mono overflow-x-auto whitespace-pre-wrap break-words">
helm install mcp-governance techwithhuz/mcp-governance \
  --create-namespace \
  --namespace mcp-governance \
  -f custom-values.yaml
              </pre>
            </div>
            <p className="text-xs text-gov-text-3 mt-3">
              <a href="https://github.com/techwithhuz/mcp-security-governance/blob/main/charts/mcp-governance/values.yaml" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:underline">
                View all available Helm values →
              </a>
            </p>
          </div>
        </div>

        {/* Common Options */}
        <div className="mt-6 bg-gov-surface rounded-2xl border border-gov-border p-6">
          <h3 className="text-lg font-bold text-gov-text mb-4">Common Helm Options</h3>
          <div className="space-y-3">
            {[
              { flag: 'samples.install=true', desc: 'Install sample MCPGovernancePolicy' },
              { flag: 'aiAgent.enabled=true', desc: 'Enable AI-powered governance scoring' },
              { flag: 'aiAgent.provider=ollama', desc: 'Use local Ollama instead of Gemini' },
              { flag: 'dashboard.service.type=LoadBalancer', desc: 'Expose dashboard via LoadBalancer' },
              { flag: 'controller.replicas=3', desc: 'Scale controller for high availability' },
            ].map(({ flag, desc }) => (
              <div key={flag} className="flex items-start gap-3 p-3 bg-gov-bg rounded-lg border border-gov-border/50">
                <ArrowRight size={16} className="text-blue-400 mt-0.5 flex-shrink-0" />
                <div>
                  <p className="text-xs font-mono text-blue-300">{flag}</p>
                  <p className="text-xs text-gov-text-3 mt-1">{desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ── Architecture ── */}
      <section>
        <SectionHeader
          icon={Network}
          color="#3b82f6"
          title="Architecture"
          subtitle="How MCP-G discovers, evaluates, and governs your MCP infrastructure"
        />
        
        {/* Architecture Diagram - Enhanced */}
        <div className="mt-8 bg-gradient-to-br from-slate-900/50 to-slate-800/50 rounded-3xl border border-gov-border p-8 overflow-x-auto">
          <svg viewBox="0 0 1400 820" className="w-full" style={{ minHeight: '720px' }}>
            <defs>
              <linearGradient id="grad-ui" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style={{ stopColor: '#6366f1', stopOpacity: 0.35 }} />
                <stop offset="100%" style={{ stopColor: '#4f46e5', stopOpacity: 0.12 }} />
              </linearGradient>
              <linearGradient id="grad-ctrl-outer" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style={{ stopColor: '#0ea5e9', stopOpacity: 0.18 }} />
                <stop offset="100%" style={{ stopColor: '#0284c7', stopOpacity: 0.06 }} />
              </linearGradient>
              <linearGradient id="grad-ctrl-inner" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style={{ stopColor: '#0ea5e9', stopOpacity: 0.4 }} />
                <stop offset="100%" style={{ stopColor: '#0284c7', stopOpacity: 0.18 }} />
              </linearGradient>
              <linearGradient id="grad-ai-inner" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style={{ stopColor: '#a855f7', stopOpacity: 0.45 }} />
                <stop offset="100%" style={{ stopColor: '#9333ea', stopOpacity: 0.18 }} />
              </linearGradient>
              <linearGradient id="grad-enforce" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style={{ stopColor: '#f59e0b', stopOpacity: 0.3 }} />
                <stop offset="100%" style={{ stopColor: '#d97706', stopOpacity: 0.1 }} />
              </linearGradient>
              <marker id="arrow-blue" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
                <polygon points="0 0, 10 3, 0 6" fill="#06b6d4" />
              </marker>
              <marker id="arrow-purple" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
                <polygon points="0 0, 10 3, 0 6" fill="#a855f7" />
              </marker>
              <marker id="arrow-red" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
                <polygon points="0 0, 10 3, 0 6" fill="#ef4444" />
              </marker>
              <marker id="arrow-pink" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
                <polygon points="0 0, 10 3, 0 6" fill="#ec4899" />
              </marker>
            </defs>

            {/* ===== K8s Cluster Border ===== */}
            <rect x="20" y="20" width="1360" height="780" fill="none" stroke="#64748b" strokeWidth="2.5" strokeDasharray="10,5" rx="18" />
            <rect x="20" y="20" width="230" height="30" fill="#1e293b" rx="6" />
            <text x="35" y="41" fill="#94a3b8" fontSize="14" fontWeight="bold">☸ Kubernetes Cluster</text>

            {/* ===== DASHBOARD BOX ===== */}
            <rect x="50" y="80" width="220" height="140" fill="url(#grad-ui)" stroke="#4f46e5" strokeWidth="2.5" rx="14" />
            {/* Dashboard label bar */}
            <rect x="50" y="80" width="220" height="30" fill="#4f46e5" fillOpacity="0.35" rx="14" />
            <rect x="50" y="94" width="220" height="16" fill="#4f46e5" fillOpacity="0.35" />
            <text x="160" y="99" fill="#e0e7ff" fontSize="11" fontWeight="bold" textAnchor="middle">🖥️ Dashboard</text>
            <text x="160" y="128" fill="#c7d2fe" fontSize="13" fontWeight="bold" textAnchor="middle">Next.js</text>
            <text x="160" y="147" fill="#a5b4fc" fontSize="10" textAnchor="middle">Port :3000</text>
            <text x="160" y="168" fill="#818cf8" fontSize="9" textAnchor="middle" fontStyle="italic">Auto-refresh every 15s</text>
            <text x="160" y="207" fill="#818cf8" fontSize="8" textAnchor="middle">polls via REST API</text>

            {/* ===== GOVERNANCE CONTROLLER OUTER BOX ===== */}
            <rect x="330" y="65" width="850" height="175" fill="url(#grad-ctrl-outer)" stroke="#0284c7" strokeWidth="3" rx="16" />
            {/* Controller label bar */}
            <rect x="330" y="65" width="850" height="30" fill="#0284c7" fillOpacity="0.35" rx="16" />
            <rect x="330" y="79" width="850" height="16" fill="#0284c7" fillOpacity="0.35" />
            <text x="755" y="85" fill="#e0f2fe" fontSize="13" fontWeight="bold" textAnchor="middle">⚙️ Governance Controller</text>

            {/* ─ sub-box: Go API Server ─ */}
            <rect x="350" y="105" width="175" height="115" fill="url(#grad-ctrl-inner)" stroke="#38bdf8" strokeWidth="2" rx="10" />
            <text x="437" y="128" fill="#bae6fd" fontSize="11" fontWeight="bold" textAnchor="middle">Go API Server</text>
            <text x="437" y="146" fill="#7dd3fc" fontSize="10" textAnchor="middle">Port :8090</text>
            <line x1="360" y1="155" x2="514" y2="155" stroke="#38bdf8" strokeWidth="1" strokeOpacity="0.4" />
            <text x="437" y="170" fill="#93c5fd" fontSize="8.5" textAnchor="middle">REST endpoints</text>
            <text x="437" y="185" fill="#93c5fd" fontSize="8.5" textAnchor="middle">CORS middleware</text>
            <text x="437" y="200" fill="#7dd3fc" fontSize="8" textAnchor="middle" fontStyle="italic">:8090</text>

            {/* ─ sub-box: Scoring Engine ─ */}
            <rect x="545" y="105" width="175" height="115" fill="url(#grad-ctrl-inner)" stroke="#38bdf8" strokeWidth="2" rx="10" />
            <text x="632" y="128" fill="#bae6fd" fontSize="11" fontWeight="bold" textAnchor="middle">Scoring Engine</text>
            <text x="632" y="146" fill="#7dd3fc" fontSize="10" textAnchor="middle">8 Categories</text>
            <line x1="555" y1="155" x2="709" y2="155" stroke="#38bdf8" strokeWidth="1" strokeOpacity="0.4" />
            <text x="632" y="170" fill="#93c5fd" fontSize="8.5" textAnchor="middle">Weighted scores</text>
            <text x="632" y="185" fill="#93c5fd" fontSize="8.5" textAnchor="middle">0–100 per server</text>
            <text x="632" y="200" fill="#7dd3fc" fontSize="8" textAnchor="middle" fontStyle="italic">Evaluator</text>

            {/* ─ sub-box: AI Agent ─ */}
            <rect x="740" y="105" width="175" height="115" fill="url(#grad-ai-inner)" stroke="#c084fc" strokeWidth="2" rx="10" />
            <text x="827" y="128" fill="#e9d5ff" fontSize="11" fontWeight="bold" textAnchor="middle">🧠 AI Agent</text>
            <text x="827" y="146" fill="#d8b4fe" fontSize="10" textAnchor="middle">Gemini / Ollama</text>
            <line x1="750" y1="155" x2="904" y2="155" stroke="#c084fc" strokeWidth="1" strokeOpacity="0.4" />
            <text x="827" y="170" fill="#c4b5fd" fontSize="8.5" textAnchor="middle">Risk analysis</text>
            <text x="827" y="185" fill="#c4b5fd" fontSize="8.5" textAnchor="middle">AI suggestions</text>
            <text x="827" y="200" fill="#d8b4fe" fontSize="8" textAnchor="middle" fontStyle="italic">Optional</text>

            {/* ─ sub-box: Inventory ─ */}
            <rect x="935" y="105" width="175" height="115" fill="url(#grad-ctrl-inner)" stroke="#38bdf8" strokeWidth="2" rx="10" />
            <text x="1022" y="128" fill="#bae6fd" fontSize="11" fontWeight="bold" textAnchor="middle">Inventory</text>
            <text x="1022" y="146" fill="#7dd3fc" fontSize="10" textAnchor="middle">Status update for Agent Registry</text>
            <line x1="945" y1="155" x2="1099" y2="155" stroke="#38bdf8" strokeWidth="1" strokeOpacity="0.4" />
            <text x="1022" y="170" fill="#93c5fd" fontSize="8.5" textAnchor="middle">Status update for MCPServerCatalog</text>
            <text x="1022" y="185" fill="#93c5fd" fontSize="8.5" textAnchor="middle">Catalog Inventory Score</text>
            <text x="1022" y="200" fill="#7dd3fc" fontSize="8" textAnchor="middle" fontStyle="italic">Patch MCPServerCatalog</text>

            {/* ===== LLM PROVIDER BOX ===== */}
            <rect x="1240" y="65" width="130" height="175" fill="#ec4899" fillOpacity="0.12" stroke="#db2777" strokeWidth="2.5" rx="14" />
            <rect x="1240" y="65" width="130" height="30" fill="#db2777" fillOpacity="0.35" rx="14" />
            <rect x="1240" y="79" width="130" height="16" fill="#db2777" fillOpacity="0.35" />
            <text x="1305" y="86" fill="#fce7f3" fontSize="10" fontWeight="bold" textAnchor="middle">🌐 LLM</text>
            <text x="1305" y="130" fill="#f9a8d4" fontSize="11" fontWeight="bold" textAnchor="middle">Gemini</text>
            <text x="1305" y="150" fill="#fbcfe8" fontSize="9" textAnchor="middle">Google AI</text>
            <line x1="1255" y1="165" x2="1355" y2="165" stroke="#db2777" strokeWidth="1" strokeOpacity="0.4" />
            <text x="1305" y="185" fill="#f9a8d4" fontSize="11" fontWeight="bold" textAnchor="middle">Ollama</text>
            <text x="1305" y="205" fill="#fbcfe8" fontSize="9" textAnchor="middle">Local LLM</text>
            <text x="1305" y="225" fill="#f472b6" fontSize="8" textAnchor="middle" fontStyle="italic">Optional</text>

            {/* ===== KUBERNETES API BOX ===== */}
            <rect x="50" y="310" width="1300" height="80" fill="#64748b" fillOpacity="0.1" stroke="#475569" strokeWidth="2.5" rx="14" />
            <rect x="50" y="310" width="1300" height="30" fill="#475569" fillOpacity="0.3" rx="14" />
            <rect x="50" y="324" width="1300" height="16" fill="#475569" fillOpacity="0.3" />
            <text x="700" y="330" fill="#e2e8f0" fontSize="12" fontWeight="bold" textAnchor="middle">🔌 Kubernetes API</text>
            <text x="700" y="365" fill="#94a3b8" fontSize="9" textAnchor="middle">list / watch — continuous discovery of MCP-related resources across all namespaces</text>

            {/* ===== DISCOVERED RESOURCES (bottom row) ===== */}
            {/* AgentGateway */}
            <rect x="50" y="470" width="230" height="150" fill="url(#grad-enforce)" stroke="#d97706" strokeWidth="2.5" rx="14" />
            <rect x="50" y="470" width="230" height="30" fill="#d97706" fillOpacity="0.4" rx="14" />
            <rect x="50" y="484" width="230" height="16" fill="#d97706" fillOpacity="0.4" />
            <text x="165" y="490" fill="#fef3c7" fontSize="11" fontWeight="bold" textAnchor="middle">🛡️ AgentGateway</text>
            <text x="165" y="525" fill="#fcd34d" fontSize="10" fontWeight="bold" textAnchor="middle">agentgateway.dev</text>
            <text x="165" y="545" fill="#fde68a" fontSize="9" textAnchor="middle">• AgentgatewayBackend</text>
            <text x="165" y="560" fill="#fde68a" fontSize="9" textAnchor="middle">• AgentgatewayPolicy</text>
            <text x="165" y="575" fill="#fde68a" fontSize="9" textAnchor="middle">• AgentgatewayParameters</text>
            <text x="165" y="600" fill="#f59e0b" fontSize="8" textAnchor="middle" fontStyle="italic">Enforcement Layer</text>

            {/* Kagent */}
            <rect x="320" y="470" width="230" height="150" fill="#10b981" fillOpacity="0.15" stroke="#059669" strokeWidth="2.5" rx="14" />
            <rect x="320" y="470" width="230" height="30" fill="#059669" fillOpacity="0.4" rx="14" />
            <rect x="320" y="484" width="230" height="16" fill="#059669" fillOpacity="0.4" />
            <text x="435" y="490" fill="#d1fae5" fontSize="11" fontWeight="bold" textAnchor="middle">🤖 Kagent</text>
            <text x="435" y="525" fill="#86efac" fontSize="10" fontWeight="bold" textAnchor="middle">kagent.dev</text>
            <text x="435" y="545" fill="#bbf7d0" fontSize="9" textAnchor="middle">• Agent</text>
            <text x="435" y="560" fill="#bbf7d0" fontSize="9" textAnchor="middle">• MCPServer</text>
            <text x="435" y="575" fill="#bbf7d0" fontSize="9" textAnchor="middle">• RemoteMCPServer</text>
            <text x="435" y="600" fill="#10b981" fontSize="8" textAnchor="middle" fontStyle="italic">AI Agent Workloads</text>

            {/* Gateway API */}
            <rect x="590" y="470" width="230" height="150" fill="#8b5cf6" fillOpacity="0.15" stroke="#7c3aed" strokeWidth="2.5" rx="14" />
            <rect x="590" y="470" width="230" height="30" fill="#7c3aed" fillOpacity="0.4" rx="14" />
            <rect x="590" y="484" width="230" height="16" fill="#7c3aed" fillOpacity="0.4" />
            <text x="705" y="490" fill="#ede9fe" fontSize="11" fontWeight="bold" textAnchor="middle">🌐 Gateway API</text>
            <text x="705" y="525" fill="#c4b5fd" fontSize="10" fontWeight="bold" textAnchor="middle">gateway.networking.k8s.io</text>
            <text x="705" y="545" fill="#ddd6fe" fontSize="9" textAnchor="middle">• Gateway</text>
            <text x="705" y="560" fill="#ddd6fe" fontSize="9" textAnchor="middle">• HTTPRoute</text>
            <text x="705" y="575" fill="#ddd6fe" fontSize="9" textAnchor="middle">• GatewayClass</text>
            <text x="705" y="600" fill="#8b5cf6" fontSize="8" textAnchor="middle" fontStyle="italic">Traffic Routing</text>

            {/* Governance CRDs */}
            <rect x="860" y="470" width="230" height="150" fill="#ef4444" fillOpacity="0.15" stroke="#dc2626" strokeWidth="2.5" rx="14" />
            <rect x="860" y="470" width="230" height="30" fill="#dc2626" fillOpacity="0.4" rx="14" />
            <rect x="860" y="484" width="230" height="16" fill="#dc2626" fillOpacity="0.4" />
            <text x="975" y="490" fill="#fee2e2" fontSize="11" fontWeight="bold" textAnchor="middle">📋 Governance CRDs</text>
            <text x="975" y="525" fill="#fca5a5" fontSize="10" fontWeight="bold" textAnchor="middle">governance.mcp.io</text>
            <text x="975" y="545" fill="#fecaca" fontSize="9" textAnchor="middle">• MCPGovernancePolicy</text>
            <text x="975" y="560" fill="#fecaca" fontSize="9" textAnchor="middle">• GovernanceEvaluation</text>
            <text x="975" y="575" fill="#fecaca" fontSize="9" textAnchor="middle">• Findings</text>
            <text x="975" y="600" fill="#ef4444" fontSize="8" textAnchor="middle" fontStyle="italic">Config &amp; Results</text>

            {/* Agent Registry */}
            <rect x="1130" y="470" width="230" height="150" fill="#ec4899" fillOpacity="0.12" stroke="#db2777" strokeWidth="2.5" rx="14" />
            <rect x="1130" y="470" width="230" height="30" fill="#db2777" fillOpacity="0.35" rx="14" />
            <rect x="1130" y="484" width="230" height="16" fill="#db2777" fillOpacity="0.35" />
            <text x="1245" y="490" fill="#fce7f3" fontSize="11" fontWeight="bold" textAnchor="middle">🗂️ Agent Registry</text>
            <text x="1245" y="525" fill="#f9a8d4" fontSize="10" fontWeight="bold" textAnchor="middle">agentregistry.dev</text>
            <text x="1245" y="545" fill="#fbcfe8" fontSize="9" textAnchor="middle">• MCPServerCatalog</text>
            <text x="1245" y="560" fill="#fbcfe8" fontSize="9" textAnchor="middle">• AgentCatalog</text>
            <text x="1245" y="575" fill="#fbcfe8" fontSize="9" textAnchor="middle">• SkillCatalog</text>
            <text x="1245" y="600" fill="#ec4899" fontSize="8" textAnchor="middle" fontStyle="italic">Verified Catalog</text>

            {/* ===== CONNECTIONS ===== */}

            {/* Dashboard → API Server (polls REST API) */}
            <path d="M 270 152 L 350 152" stroke="#06b6d4" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <text x="275" y="143" fill="#06b6d4" fontSize="8.5" fontWeight="bold">REST API polls</text>

            {/* API → Scoring Engine (internal) */}
            <path d="M 525 162 L 545 162" stroke="#38bdf8" strokeWidth="2" markerEnd="url(#arrow-blue)" />

            {/* API → AI Agent (internal) */}
            <path d="M 720 162 L 740 162" stroke="#c084fc" strokeWidth="2" markerEnd="url(#arrow-purple)" />

            {/* Scoring Engine → Inventory (internal) */}
            <path d="M 915 162 L 935 162" stroke="#38bdf8" strokeWidth="2" markerEnd="url(#arrow-blue)" />

            {/* AI Agent → LLM (dashed) */}
            <path d="M 1115 152 L 1240 152" stroke="#a855f7" strokeWidth="2.5" strokeDasharray="6,4" markerEnd="url(#arrow-purple)" />
            <text x="1143" y="143" fill="#a855f7" fontSize="8.5" fontWeight="bold">LLM inference</text>

            {/* Inventory → K8s API (centered) */}
            <path d="M 755 240 L 755 310" stroke="#0ea5e9" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <text x="800" y="280" fill="#0ea5e9" fontSize="8.5" fontWeight="bold">list / watch</text>

            {/* K8s API → Resources (fan down) */}
            <path d="M 165 390 L 165 470" stroke="#475569" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <path d="M 435 390 L 435 470" stroke="#475569" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <path d="M 705 390 L 705 470" stroke="#475569" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <path d="M 975 390 L 975 470" stroke="#475569" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <path d="M 1245 390 L 1245 470" stroke="#475569" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />

            {/* Scoring Engine → Governance CRDs (writes status) */}
            {/*<path d="M 632 240 Q 632 410 975 470" stroke="#ef4444" strokeWidth="2" strokeDasharray="6,4" markerEnd="url(#arrow-red)" />
            <text x="680" y="380" fill="#ef4444" fontSize="8.5" fontWeight="bold">writes status</text>*/}

            {/* ===== LEGEND ===== */}
            <rect x="50" y="650" width="1300" height="140" fill="#0f172a" fillOpacity="0.7" stroke="#475569" strokeWidth="1.5" rx="12" />
            <text x="70" y="675" fill="#94a3b8" fontSize="11" fontWeight="bold">Legend — Data Flows</text>

            <line x1="70" y1="695" x2="130" y2="695" stroke="#06b6d4" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <text x="145" y="700" fill="#cbd5e1" fontSize="9">REST API calls &amp; resource discovery</text>

            <line x1="70" y1="720" x2="130" y2="720" stroke="#a855f7" strokeWidth="2.5" strokeDasharray="6,4" markerEnd="url(#arrow-purple)" />
            <text x="145" y="725" fill="#cbd5e1" fontSize="9">AI / LLM inference (async)</text>

            <line x1="450" y1="695" x2="510" y2="695" stroke="#ef4444" strokeWidth="2" strokeDasharray="6,4" markerEnd="url(#arrow-red)" />
            <text x="525" y="700" fill="#cbd5e1" fontSize="9">Evaluation results written to Governance CRDs</text>

            <line x1="450" y1="720" x2="510" y2="720" stroke="#475569" strokeWidth="2.5" markerEnd="url(#arrow-blue)" />
            <text x="525" y="725" fill="#cbd5e1" fontSize="9">K8s API list / watch (continuous)</text>

            {/* Color key boxes */}
            <rect x="70" y="740" width="12" height="12" fill="#4f46e5" rx="3" />
            <text x="88" y="751" fill="#a5b4fc" fontSize="9">Dashboard UI</text>
            <rect x="200" y="740" width="12" height="12" fill="#0284c7" rx="3" />
            <text x="218" y="751" fill="#7dd3fc" fontSize="9">Governance Controller</text>
            <rect x="380" y="740" width="12" height="12" fill="#d97706" rx="3" />
            <text x="398" y="751" fill="#fcd34d" fontSize="9">AgentGateway</text>
            <rect x="490" y="740" width="12" height="12" fill="#059669" rx="3" />
            <text x="508" y="751" fill="#86efac" fontSize="9">Kagent</text>
            <rect x="570" y="740" width="12" height="12" fill="#7c3aed" rx="3" />
            <text x="588" y="751" fill="#c4b5fd" fontSize="9">Gateway API</text>
            <rect x="680" y="740" width="12" height="12" fill="#dc2626" rx="3" />
            <text x="698" y="751" fill="#fca5a5" fontSize="9">Governance CRDs</text>
            <rect x="820" y="740" width="12" height="12" fill="#db2777" rx="3" />
            <text x="838" y="751" fill="#f9a8d4" fontSize="9">Agent Registry / LLM</text>
          </svg>
        </div>

        {/* Key Flows */}
        <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
          {[
            {
              title: 'Request Flow',
              icon: ArrowRight,
              color: '#f59e0b',
              items: ['Agent calls tool', 'Gateway validates JWT', 'Policy checks allow-list', 'Tool executes', 'Response masked'],
            },
            {
              title: 'Discovery Flow',
              icon: Search,
              color: '#0ea5e9',
              items: ['Watch K8s API', 'List all MCP CRDs', 'Discover Backends', 'Find HTTPRoutes', 'Read Policies'],
            },
            {
              title: 'Evaluation Flow',
              icon: BarChart3,
              color: '#10b981',
              items: ['Score 8 categories', 'Calculate per-tool risk', 'Analyze prompt injection', 'Generate findings', 'Update dashboard'],
            },
          ].map((flow, idx) => {
            const Icon = flow.icon;
            return (
              <div key={idx} className="bg-gov-surface rounded-2xl border border-gov-border p-4">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-2 rounded-lg" style={{ backgroundColor: `${flow.color}15` }}>
                    <Icon size={16} style={{ color: flow.color }} />
                  </div>
                  <h4 className="text-sm font-bold text-gov-text">{flow.title}</h4>
                </div>
                <ol className="space-y-2">
                  {flow.items.map((item, i) => (
                    <li key={i} className="text-xs text-gov-text-3 flex items-start gap-2">
                      <span className="inline-block w-1.5 h-1.5 rounded-full flex-shrink-0 mt-1.5" style={{ backgroundColor: flow.color }}></span>
                      {item}
                    </li>
                  ))}
                </ol>
              </div>
            );
          })}
        </div>
      </section>

      {/* ── Components ── */}
      <section>
        <SectionHeader
          icon={Blocks}
          color="#8b5cf6"
          title="Key Components"
          subtitle="Everything that makes up the MCP-G platform"
        />
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mt-6">
          {COMPONENTS.map(comp => {
            const Icon = comp.icon;
            return (
              <div key={comp.name} className="bg-gov-surface rounded-2xl border border-gov-border p-5 hover:border-gov-border-light transition-all group">
                <div className="flex items-start gap-4 mb-4">
                  <div className="p-2.5 rounded-xl flex-shrink-0 group-hover:scale-110 transition-transform" style={{ backgroundColor: `${comp.color}15` }}>
                    <Icon size={20} style={{ color: comp.color }} />
                  </div>
                  <div>
                    <h3 className="text-sm font-bold text-gov-text">{comp.name}</h3>
                    <span className="text-[10px] font-mono text-gov-text-3 px-1.5 py-0.5 bg-gov-bg rounded border border-gov-border">{comp.tech}</span>
                  </div>
                </div>
                <p className="text-xs text-gov-text-3 leading-relaxed mb-4">{comp.description}</p>
                <ul className="space-y-1.5">
                  {comp.responsibilities.map(r => (
                    <li key={r} className="flex items-center gap-2 text-xs text-gov-text-2">
                      <CheckCircle2 size={11} style={{ color: comp.color }} className="flex-shrink-0" />
                      {r}
                    </li>
                  ))}
                </ul>
              </div>
            );
          })}
        </div>
      </section>

      {/* ── How It Works ── */}
      <section>
        <SectionHeader
          icon={RefreshCw}
          color="#10b981"
          title="How It Works"
          subtitle="The six steps MCP-G performs on every reconciliation cycle"
        />
        <div className="mt-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {HOW_IT_WORKS.map(step => {
            const Icon = step.icon;
            return (
              <div key={step.step} className="bg-gov-surface rounded-2xl border border-gov-border p-5 hover:border-gov-border-light transition-all relative overflow-hidden group">
                <div
                  className="absolute top-4 right-4 text-5xl font-black opacity-[0.04] group-hover:opacity-[0.07] transition-opacity select-none"
                  style={{ color: step.color }}
                >
                  {step.step}
                </div>
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-2 rounded-xl" style={{ backgroundColor: `${step.color}15` }}>
                    <Icon size={18} style={{ color: step.color }} />
                  </div>
                  <span className="text-xs font-black uppercase tracking-widest" style={{ color: step.color }}>Step {step.step}</span>
                </div>
                <h3 className="text-sm font-bold text-gov-text mb-2">{step.title}</h3>
                <p className="text-xs text-gov-text-3 leading-relaxed">{step.description}</p>
              </div>
            );
          })}
        </div>
      </section>

      {/* ── Scoring System ── */}
      <section>
        <SectionHeader
          icon={Star}
          color="#f59e0b"
          title="Security Scoring System"
          subtitle="How the 0–100 security score is calculated for each MCP server"
        />

        {/* Grade thresholds */}
        <div className="mt-6 grid grid-cols-2 md:grid-cols-5 gap-3 mb-6">
          {GRADE_THRESHOLDS.map(g => (
            <div
              key={g.grade}
              className="text-center rounded-2xl border p-4"
              style={{ borderColor: `${g.color}30`, backgroundColor: `${g.color}06` }}
            >
              <div className="text-3xl font-black mb-1" style={{ color: g.color }}>{g.grade}</div>
              <div className="text-xs font-bold text-gov-text-2 mb-1">{g.min}+</div>
              <div className="text-[10px] text-gov-text-3">{g.label}</div>
            </div>
          ))}
        </div>

        {/* Score formula */}
        <div className="mb-6 bg-gov-surface rounded-2xl border border-gov-border p-5">
          <div className="flex items-center gap-2 mb-4">
            <BarChart3 size={16} className="text-yellow-400" />
            <span className="text-sm font-bold text-gov-text-2 uppercase tracking-wider">Score Formula</span>
          </div>
          <div className="overflow-x-auto">
            <div className="flex flex-wrap items-center gap-3 min-w-max">
              {SCORING_CONTROLS.map((ctrl, i) => {
                const Icon = ctrl.icon;
                return (
                  <span key={ctrl.key} className="flex items-center gap-2">
                    <span
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold border"
                      style={{ backgroundColor: `${ctrl.color}10`, color: ctrl.color, borderColor: `${ctrl.color}25` }}
                    >
                      <Icon size={11} />
                      {ctrl.key}
                      <span className="font-mono opacity-70">/{ctrl.maxScore}</span>
                    </span>
                    {i < SCORING_CONTROLS.length - 1 && <span className="text-gov-text-3 font-bold">+</span>}
                  </span>
                );
              })}
              <span className="text-gov-text-3 font-bold">=</span>
              <span className="px-3 py-1.5 rounded-xl text-xs font-black bg-gov-accent/10 text-blue-400 border border-blue-500/20">
                Score (0–110, capped at 100)
              </span>
            </div>
          </div>
          <p className="text-xs text-gov-text-3 mt-3 flex items-start gap-1.5">
            <Info size={12} className="flex-shrink-0 mt-0.5 text-gov-text-3" />
            Scores are capped at 100. Having all controls at maximum gives 110 — the extra 10 acts as a buffer so you can still achieve 100 even if one minor control is slightly below max.
          </p>
        </div>

        {/* Controls accordion */}
        <div className="space-y-2">
          <p className="text-xs font-bold text-gov-text-3 uppercase tracking-wider px-1 mb-3">Click any control to see how it's scored</p>
          {SCORING_CONTROLS.map(ctrl => {
            const Icon = ctrl.icon;
            const isOpen = expandedControl === ctrl.key;
            return (
              <div
                key={ctrl.key}
                className="bg-gov-surface rounded-xl border border-gov-border overflow-hidden transition-all"
              >
                <button
                  className="w-full px-5 py-4 flex items-center gap-4 hover:bg-gov-bg/50 transition-colors text-left"
                  onClick={() => setExpandedControl(isOpen ? null : ctrl.key)}
                >
                  <div className="p-2 rounded-lg flex-shrink-0" style={{ backgroundColor: `${ctrl.color}15` }}>
                    <Icon size={16} style={{ color: ctrl.color }} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-bold text-gov-text">{ctrl.key}</span>
                      <span
                        className="px-2 py-0.5 rounded-md text-[10px] font-bold border"
                        style={{ backgroundColor: `${ctrl.color}10`, color: ctrl.color, borderColor: `${ctrl.color}25` }}
                      >
                        max {ctrl.maxScore} pts
                      </span>
                    </div>
                    <p className="text-xs text-gov-text-3 mt-0.5 truncate">{ctrl.description}</p>
                  </div>
                  {isOpen ? <ChevronUp size={16} className="text-gov-text-3 flex-shrink-0" /> : <ChevronDown size={16} className="text-gov-text-3 flex-shrink-0" />}
                </button>
                {isOpen && (
                  <div className="px-5 pb-5 border-t border-gov-border pt-4 space-y-3 bg-gov-bg/30">
                    {/* Why it matters */}
                    <div className="flex items-start gap-2.5 p-3 rounded-xl bg-blue-500/5 border border-blue-500/15">
                      <Info size={13} className="text-blue-400 flex-shrink-0 mt-0.5" />
                      <div>
                        <div className="text-[10px] font-bold text-blue-400 uppercase tracking-wider mb-1">Why It Matters</div>
                        <p className="text-xs text-gov-text-2 leading-relaxed">{ctrl.why}</p>
                      </div>
                    </div>
                    {/* Pass condition */}
                    <div className="flex items-start gap-2.5 p-3 rounded-xl bg-green-500/5 border border-green-500/15">
                      <CheckCircle2 size={13} className="text-green-400 flex-shrink-0 mt-0.5" />
                      <div>
                        <div className="text-[10px] font-bold text-green-400 uppercase tracking-wider mb-1">Full Score</div>
                        <p className="text-xs text-gov-text-2 leading-relaxed">{ctrl.pass}</p>
                      </div>
                    </div>
                    {/* Partial condition */}
                    {ctrl.partial && (
                      <div className="flex items-start gap-2.5 p-3 rounded-xl bg-yellow-500/5 border border-yellow-500/15">
                        <AlertTriangle size={13} className="text-yellow-400 flex-shrink-0 mt-0.5" />
                        <div>
                          <div className="text-[10px] font-bold text-yellow-400 uppercase tracking-wider mb-1">Partial Score</div>
                          <p className="text-xs text-gov-text-2 leading-relaxed">{ctrl.partial}</p>
                        </div>
                      </div>
                    )}
                    {/* Fail condition */}
                    <div className="flex items-start gap-2.5 p-3 rounded-xl bg-red-500/5 border border-red-500/15">
                      <AlertTriangle size={13} className="text-red-400 flex-shrink-0 mt-0.5" />
                      <div>
                        <div className="text-[10px] font-bold text-red-400 uppercase tracking-wider mb-1">Zero Score</div>
                        <p className="text-xs text-gov-text-2 leading-relaxed">{ctrl.fail}</p>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </section>

      {/* ── Key Concepts ── */}
      <section>
        <SectionHeader
          icon={Info}
          color="#06b6d4"
          title="Key Concepts"
          subtitle="Important terms and ideas to understand in MCP-G"
        />
        <div className="mt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
          {[
            {
              term: 'Path-Based Tool Restriction',
              color: '#3b82f6',
              icon: Route,
              body: 'An HTTPRoute can have multiple path rules (e.g. /mcp/grafana/ro and /mcp/grafana/rw). Each path has its own AgentgatewayPolicy with a different tool allow-list. This means read-only agents only ever see read tools, and write agents only see write tools — even though they talk to the same MCP server.',
            },
            {
              term: 'AgentgatewayPolicy',
              color: '#8b5cf6',
              icon: Shield,
              body: 'A Kubernetes CRD (agentgateway.dev/v1alpha1) that attaches security rules to an HTTPRoute. It has two sections: traffic (applies at ingress: JWT, CORS, rate limit) and backend (applies at egress: MCP tool authorization, prompt guard). Both can carry tool allow-lists.',
            },
            {
              term: 'Effective Tool Count',
              color: '#10b981',
              icon: Layers,
              body: 'When a server has multiple paths with different tool sets, MCP-G shows the most-restrictive set as the "effective" count. This prevents inflated numbers — if /ro allows 10 tools and /rw allows 10 different tools, the effective count is 10 (the tightest single-path exposure), not 20.',
            },
            {
              term: 'Tool Scope Score',
              color: '#f59e0b',
              icon: Server,
              body: 'Calculated as: restriction_ratio = 1 - (effective_tool_count / total_tools). A server with 57 tools but only 10 exposed has a ratio of 82.5% → full 10 points. The score is graduated: >75% restriction = full marks, 50–75% = partial, <50% = lower score.',
            },
            {
              term: 'Verified Catalog',
              color: '#ec4899',
              icon: Database,
              body: 'An MCPServerCatalog CRD that curates which MCP servers are approved for use in your organization. Each catalog entry has its own security score. Agents should only be allowed to use servers that appear in the verified catalog with a passing score.',
            },
            {
              term: 'Prompt Guard',
              color: '#06b6d4',
              icon: ShieldCheck,
              body: 'Agentgateway can inspect both the request (tool inputs) and response (tool outputs) for prompt injection patterns (e.g. "ignore previous instructions") and sensitive data (SSN, credit cards, emails). Matches can be blocked or masked before reaching/leaving the agent.',
            },
          ].map(({ term, color, icon: Icon, body }) => (
            <div key={term} className="bg-gov-surface rounded-2xl border border-gov-border p-5 hover:border-gov-border-light transition-all">
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2 rounded-lg" style={{ backgroundColor: `${color}15` }}>
                  <Icon size={16} style={{ color }} />
                </div>
                <h3 className="text-sm font-bold text-gov-text">{term}</h3>
              </div>
              <p className="text-xs text-gov-text-3 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* ── Quick Reference ── */}
      <section>
        <SectionHeader
          icon={Scan}
          color="#22c55e"
          title="Security Checklist"
          subtitle="What a fully-secured MCP server looks like in MCP-G"
        />
        <div className="mt-6 bg-gov-surface rounded-2xl border border-gov-border overflow-hidden">
          <div className="divide-y divide-gov-border">
            {[
              { check: 'Routed through Agentgateway via HTTPRoute', required: true },
              { check: 'JWT authentication in Strict mode on all routes', required: true },
              { check: 'Tool allow-list policy on every HTTPRoute path', required: true },
              { check: 'TLS with SNI verification to MCP backend', required: true },
              { check: 'CORS policy restricting allowed origins', required: true },
              { check: 'Rate limiting on all route paths', required: true },
              { check: 'Prompt injection guard on request + response', required: true },
              { check: 'Less than 25% of total tools exposed per path', required: false },
              { check: 'Separate /ro and /rw paths with distinct tool sets', required: false },
              { check: 'Server present in Verified Catalog with passing score', required: false },
            ].map(({ check, required }) => (
              <div key={check} className="px-5 py-3.5 flex items-center gap-4">
                <CheckCircle2 size={16} className="text-green-400 flex-shrink-0" />
                <span className="text-sm text-gov-text flex-1">{check}</span>
                <span
                  className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md ${
                    required
                      ? 'bg-red-500/10 text-red-400 border border-red-500/20'
                      : 'bg-gov-bg text-gov-text-3 border border-gov-border'
                  }`}
                >
                  {required ? 'Required' : 'Best Practice'}
                </span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ── Kubernetes CRDs & Configuration ── */}
      <section>
        <SectionHeader
          icon={Database}
          color="#ec4899"
          title="MCP-G Resources & Configuration"
          subtitle="Custom resources that power MCP-G governance"
        />

        {/* CRDs Overview */}
        <div className="mt-6 grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
          {[
            {
              name: 'MCPGovernancePolicy',
              group: 'governance.mcp.io/v1alpha1',
              icon: Shield,
              color: '#ec4899',
              description: 'Defines security policies and evaluation criteria for MCP servers. Specifies what controls to check, thresholds, weights, and remediation guidance.',
              examples: ['Authentication requirements', 'Tool scope thresholds', 'TLS enforcement', 'Rate limit policies'],
            },
            {
              name: 'MCPGovernanceEvaluation',
              group: 'governance.mcp.io/v1alpha1',
              icon: BarChart3,
              color: '#06b6d4',
              description: 'Written by the controller to store evaluation results. Contains per-server scores, findings, and governance status. Updated every 30 seconds.',
              examples: ['Server scores', 'Security findings', 'Compliance status', 'Tool exposure metrics'],
            },
          ].map(crd => {
            const Icon = crd.icon;
            return (
              <div key={crd.name} className="bg-gov-surface rounded-2xl border border-gov-border p-6 hover:border-gov-border-light transition-all">
                <div className="flex items-start gap-4 mb-4">
                  <div className="p-3 rounded-xl flex-shrink-0" style={{ backgroundColor: `${crd.color}15` }}>
                    <Icon size={20} style={{ color: crd.color }} />
                  </div>
                  <div>
                    <h3 className="text-base font-bold text-gov-text">{crd.name}</h3>
                    <span className="text-[10px] font-mono text-gov-text-3 px-1.5 py-0.5 bg-gov-bg rounded border border-gov-border mt-1 inline-block">{crd.group}</span>
                  </div>
                </div>
                <p className="text-xs text-gov-text-2 leading-relaxed mb-3">{crd.description}</p>
                <div className="space-y-1.5">
                  {crd.examples.map(ex => (
                    <div key={ex} className="flex items-center gap-2 text-xs text-gov-text-3">
                      <span className="w-1.5 h-1.5 rounded-full flex-shrink-0" style={{ backgroundColor: crd.color }}></span>
                      {ex}
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>

        {/* MCPGovernancePolicy Example */}
        <div className="mb-8">
          <h3 className="text-base font-bold text-gov-text mb-4 flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-pink-500/10">
              <Shield size={16} className="text-pink-400" />
            </div>
            MCPGovernancePolicy Example
          </h3>
          <div className="bg-gov-bg rounded-2xl border border-gov-border overflow-hidden">
            <div className="px-4 py-3 bg-gov-surface border-b border-gov-border flex items-center gap-2">
              <span className="text-xs font-bold text-gov-text-2 uppercase tracking-wider">YAML Configuration</span>
              <span className="ml-auto text-[10px] font-mono text-gov-text-3">governance.mcp.io/v1alpha1</span>
            </div>
            <pre className="p-4 text-xs font-mono text-gov-text-2 overflow-x-auto">
{`apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernancePolicy
metadata:
  name: enterprise-mcp-policy
  namespace: default
spec:
  # Evaluation scope - which MCP servers to score
  selector:
    matchLabels:
      governance: enabled
  aiAgent:
    enabled: true
    # Provider: "gemini" (requires GOOGLE_API_KEY env var) or "ollama" (local model)
    provider: gemini
    # Model name: e.g. "gemini-2.5-flash" for Gemini, "llama3.1" or "qwen2.5" for Ollama
    model: "gemini-2.5-flash"
    # Ollama endpoint (only used when provider is "ollama")
    # ollamaEndpoint: "http://localhost:11434"
    # Interval between periodic AI evaluations (e.g. "5m", "10m", "1h")
    scanInterval: "5m"
    # Set to false to disable periodic scanning (manual-only via dashboard button)
    scanEnabled: true
  # Control weights (0-100 each)
  weights:
    gatewayRouting: 20    # Must route through AgentGateway
    authentication: 20    # JWT auth required
    authorization: 15     # Tool allow-lists via policy
    tls: 15              # Backend TLS encryption
    cors: 10             # CORS policy configured
    rateLimit: 10        # Rate limiting enforced
    promptGuard: 10      # Prompt injection protection
    toolScope: 10        # Tool exposure restricted
  
  # Thresholds for what constitutes a pass/fail
  thresholds:
    minServerScore: 60           # F grade below this
    minCategoryScore: 50         # Control fails if below
    maxToolExposure: 0.25        # Max 25% of tools exposed
    criticalFinding: 85          # Critical severity if score below
  
  # Findings severity levels
  severityPenalties:
    Critical: 20   # Deduct 20 points per finding
    High: 10
    Medium: 5
    Low: 1
  
  # Verified Catalog Scoring configuration
  verifiedCatalogScoring:
    # Category weights (should sum to 100)
    # Controls how the final composite score is computed
    securityWeight: 50      # Weight for security score
    trustWeight: 30         # Weight for trust score
    complianceWeight: 20    # Weight for compliance score
    # Status thresholds
    verifiedThreshold: 70   # Score >= 70 → "Verified"
    unverifiedThreshold: 50 # Score >= 50 → "Unverified", < 50 → "Rejected"
    # Per-check maximum scores (customize severity of each check)
    checkMaxScores:
      PUB-001: 20  # Publisher Verification
      PUB-002: 15  # Code Signing
      PUB-003: 15  # Publisher Trust
      SEC-001: 15  # Transport Security
      SEC-002: 10  # Deployment Security
      DEP-001: 10  # Dependency Analysis
      DEP-002: 10  # Vulnerability Scanning
      DEP-003: 10  # License Compliance
      TOOL-001: 5  # Tool Scope
      USE-001: 5   # Usage Patterns
  # targetNamespaces is empty – scan all namespaces by default
  targetNamespaces: []`}
            </pre>
          </div>
          <p className="text-xs text-gov-text-3 mt-3">
            📝 This policy defines the "rules of the game" — what the controller should evaluate, how much each control matters (weights), and what triggers critical findings.
          </p>
        </div>

        {/* MCPGovernanceEvaluation Example */}
        <div className="mb-8">
          <h3 className="text-base font-bold text-gov-text mb-4 flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-cyan-500/10">
              <BarChart3 size={16} className="text-cyan-400" />
            </div>
            MCPGovernanceEvaluation (Status Output)
          </h3>
          <div className="bg-gov-bg rounded-2xl border border-gov-border overflow-hidden">
            <div className="px-4 py-3 bg-gov-surface border-b border-gov-border flex items-center gap-2">
              <span className="text-xs font-bold text-gov-text-2 uppercase tracking-wider">Generated by Controller</span>
              <span className="ml-auto text-[10px] font-mono text-gov-text-3">Written every 30 seconds</span>
            </div>
            <pre className="p-4 text-xs font-mono text-gov-text-2 overflow-x-auto">
{`apiVersion: governance.mcp.io/v1alpha1
kind: MCPGovernanceEvaluation
metadata:
  name: evaluation-timestamp-001
  namespace: default
spec:
  policyRef:
    name: enterprise-mcp-policy
    namespace: default
  evaluatedAt: "2026-02-21T10:30:00Z"

status:
  # Overall cluster score
  overallScore: 78
  overallGrade: B
  phase: Compliant  # Compliant | AtRisk | Critical
  
  # Per-category cluster scores (average across all servers)
  categoryScores:
    gatewayRouting: 85
    authentication: 90
    authorization: 75
    tls: 80
    cors: 70
    rateLimit: 65
    promptGuard: 75
    toolScope: 70
  
  # Per-server evaluation results
  serverEvaluations:
    - name: grafana-mcp
      namespace: kagent
      score: 85
      grade: B
      categories:
        gatewayRouting:
          score: 100
          status: Pass
          evidence: "Routed via AgentgatewayBackend"
        authentication:
          score: 100
          status: Pass
          evidence: "JWT Strict mode enabled"
        authorization:
          score: 75
          status: Pass
          evidence: "10 tools allowed of 57 total"
        # ... more categories ...
  
  # Aggregated findings
  findings:
    - id: TLS-BACKEND-001
      severity: High
      category: TLS Encryption
      title: Backend TLS not configured
      description: MCP backend uses plaintext connection
      serverRef: ["postgres-mcp"]
      remediation: "Enable TLS with SNI in AgentgatewayBackend"
    - id: AUTH-STRICT-001
      severity: Medium
      category: Authentication
      title: JWT in Permissive mode
      description: JWT auth allows requests without token
      serverRef: ["redis-mcp", "cache-mcp"]
      remediation: "Change jwtAuthentication.mode to Strict"
  
  # Summary statistics
  summary:
    totalServersEvaluated: 3
    serversAtRisk: 1        # Score < 60
    serversCritical: 0      # Score < 40
    criticalFindings: 1
    highFindings: 2
    mediumFindings: 3
    lowFindings: 5`}
            </pre>
          </div>
          <p className="text-xs text-gov-text-3 mt-3">
            📊 This is the <strong>result</strong> of evaluation — what the controller found. It's stored in etcd and referenced by the dashboard.
          </p>
        </div>

        {/* Configuration Deep Dive removed */}
      </section>
    </div>
  );
}

function SectionHeader({ icon: Icon, color, title, subtitle }: {
  icon: typeof Shield;
  color: string;
  title: string;
  subtitle: string;
}) {
  return (
    <div className="flex items-start gap-4">
      <div className="p-2.5 rounded-xl flex-shrink-0 mt-0.5" style={{ backgroundColor: `${color}15` }}>
        <Icon size={20} style={{ color }} />
      </div>
      <div>
        <h2 className="text-xl font-black text-gov-text">{title}</h2>
        <p className="text-sm text-gov-text-3 mt-0.5">{subtitle}</p>
      </div>
    </div>
  );
}
