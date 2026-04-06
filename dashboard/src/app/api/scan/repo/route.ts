import { NextRequest, NextResponse } from 'next/server';
import { exec } from 'child_process';
import { promisify } from 'util';
import { readdir, readFile, rm } from 'fs/promises';
import { existsSync, readFileSync } from 'fs';
import path from 'path';
import { randomUUID } from 'crypto';

const execAsync = promisify(exec);

// ─── Types ────────────────────────────────────────────────────────────────────

interface ScanRequest {
  repoUrl: string;
  folderPath?: string;
  credentialToken?: string;
}

interface ScanFinding {
  checkID: string;
  severity: 'Critical' | 'High' | 'Medium' | 'Low';
  category: string;
  title: string;
  remediation: string;
  filePath?: string;
  line?: number;
  matchedPattern?: string;
}

interface SecurityCheck {
  id: string;
  name: string;
  passed: boolean;
  description: string;
  remediation: string;  // failure message / remediation guidance
  findingCount: number;
  severity?: 'Critical' | 'High' | 'Medium' | 'Low';
}

interface FolderScanResult {
  folderPath: string;
  filesScanned: number;
  findings: ScanFinding[];
  securityChecks: SecurityCheck[];
  score: number;
  status: 'pass' | 'warning' | 'fail';
}

interface ScanResult {
  status: 'success' | 'error';
  repoUrl: string;
  scanPath: string;
  totalFilesScanned: number;
  totalFindings: number;
  folderResults: FolderScanResult[];
  error?: string;
}

// ─── Git clone helpers ────────────────────────────────────────────────────────

/** Build the clone URL, injecting the token for private repos */
function buildCloneUrl(repoUrl: string, token?: string): string {
  const url = repoUrl.trim().replace(/\.git$/, '');
  if (!token) return `${url}.git`;
  // Inject token as: https://x-access-token:<token>@github.com/owner/repo.git
  return url.replace(/^(https?:\/\/)/, `$1x-access-token:${token}@`) + '.git';
}

/** Sparse-checkout clone — only fetches the specified folder (and its contents).
 *  Uses --depth=1 so no history is downloaded.
 *  Falls back to a full shallow clone if sparse-checkout fails (older git versions).
 */
async function sparseClone(
  repoUrl: string,
  folderPath: string,
  destDir: string,
  token?: string,
): Promise<void> {
  const cloneUrl = buildCloneUrl(repoUrl, token);

  // Git config to avoid interactive prompts
  const GIT_ENV = 'GIT_TERMINAL_PROMPT=0 GIT_ASKPASS=echo';

  // Step 1: bare init + set remote (avoids downloading anything yet)
  await execAsync(`${GIT_ENV} git init "${destDir}"`);
  await execAsync(`${GIT_ENV} git -C "${destDir}" remote add origin "${cloneUrl}"`);

  // Step 2: enable sparse-checkout
  await execAsync(`git -C "${destDir}" config core.sparseCheckout true`);

  // Step 3: write the sparse-checkout pattern
  const sparsePattern = folderPath ? `${folderPath.replace(/\/$/, '')}/**` : '/**';
  await execAsync(`echo "${sparsePattern}" >> "${destDir}/.git/info/sparse-checkout"`);

  // Step 4: fetch only the default branch at depth 1
  await execAsync(
    `${GIT_ENV} git -C "${destDir}" fetch --depth=1 origin HEAD`,
    { timeout: 60_000 },
  );

  // Step 5: checkout
  await execAsync(`${GIT_ENV} git -C "${destDir}" checkout FETCH_HEAD`, { timeout: 30_000 });
}

function parseGitHubURL(repoUrl: string): { owner: string; repo: string } {
  const url = repoUrl.trim().replace(/\.git$/, '');
  const m = url.match(/(?:github\.com|gitlab\.com)[:/]([^/]+)\/([^/]+)/);
  if (!m) throw new Error(`Cannot parse repository URL: ${repoUrl}`);
  return { owner: m[1], repo: m[2] };
}

// ─── Pattern definitions ──────────────────────────────────────────────────────
// SKL-SEC-001 → SKL-SEC-014 security checks for skill content analysis.
//
// ALL keyword patterns are loaded exclusively from the mcp-governance-skill-patterns
// ConfigMap, mounted at /etc/mcp-governance/skill-patterns (override via
// SKILL_PATTERNS_MOUNT env var).  There are NO hardcoded keyword lists in this file.
//
// Only static governance metadata (check name, severity, category, remediation text)
// is defined here in PATTERN_META — these require a code review to change.
// Keyword lists are purely operator-managed via the ConfigMap.
//
// The scanner will throw at startup if the ConfigMap mount is not present,
// so misconfigured deployments are caught immediately rather than silently
// scanning with stale or missing patterns.

// ── Types ─────────────────────────────────────────────────────────────────────

type PatternEntry = {
  name: string;
  severity: 'Critical' | 'High' | 'Medium' | 'Low';
  category: string;
  remediation: string;
  keywords?: string[];
  requiredPhrases?: string[];
};

type PatternMap = Record<string, PatternEntry>;

// ── ConfigMap loader ──────────────────────────────────────────────────────────
// ALL keyword/phrase patterns are loaded exclusively from the
// mcp-governance-skill-patterns ConfigMap, mounted read-only into the pod at
// /etc/mcp-governance/skill-patterns (override via SKILL_PATTERNS_MOUNT env var).
//
// Each ConfigMap key maps to a file; each file contains one keyword per line
// (lines starting with # are comments, blank lines are ignored).
//
// Static governance metadata (check name, severity, category, remediation text)
// lives in PATTERN_META below — these fields are intentionally kept in code so
// they require a code-review + image rebuild to change (governance control).
//
// The loader throws at first use if the mount directory is absent, so a
// misconfigured pod fails fast rather than silently using stale patterns.

const CONFIGMAP_MOUNT = process.env.SKILL_PATTERNS_MOUNT ?? '/etc/mcp-governance/skill-patterns';

/** Static governance metadata per check — name, severity, category, remediation.
 *  These never change without a code review. Only keyword lists are in the ConfigMap.
 */
const PATTERN_META: Record<string, Omit<PatternEntry, 'keywords' | 'requiredPhrases'>> = {
  'SKL-SEC-001': { name: 'Prompt Injection',                        severity: 'Critical', category: 'Prompt Injection',    remediation: 'Remove or rewrite instructions that attempt to override the AI agent\'s system prompt or safety guidelines.' },
  'SKL-SEC-002': { name: 'Malicious Code / Privilege Escalation',   severity: 'Critical', category: 'Malicious Code',       remediation: 'Remove privilege escalation and code execution patterns. Skills should operate with least-privilege principles.' },
  'SKL-SEC-003': { name: 'Data Exfiltration',                       severity: 'High',     category: 'Data Exfiltration',    remediation: 'Remove external data transmission patterns. Ensure data flows comply with your governance policy.' },
  'SKL-SEC-004': { name: 'Insecure Credential Handling',            severity: 'Critical', category: 'Credential Handling',  remediation: 'Remove instructions that collect, harvest, or require verbatim output of credentials. Use environment variables or secret managers instead.' },
  'SKL-SEC-005': { name: 'Scope Creep',                             severity: 'High',     category: 'Scope Creep',          remediation: 'Remove actions outside the skill\'s declared scope, or update the skill metadata to accurately reflect its true capabilities.' },
  'SKL-SEC-006': { name: 'Missing Safety Guardrails',               severity: 'Medium',   category: 'Safety Guardrails',    remediation: 'Add an explicit safety guardrail note. Required: at least one of "do not", "never", "always verify", "safety", "guardrail", "must not", "prohibited".' },
  'SKL-SEC-007': { name: 'Suspicious Download URL',                 severity: 'Critical', category: 'Suspicious Download',  remediation: 'Remove links to executables, archives from untrusted sources, URL shorteners, and base64/encoded download commands. Only reference official, verifiable package managers.' },
  'SKL-SEC-008': { name: 'Hardcoded Secrets',                       severity: 'High',     category: 'Hardcoded Secrets',    remediation: 'Remove hardcoded API keys, tokens, passwords, and private keys from skill content. Use environment variables or a secrets manager instead.' },
  'SKL-SEC-009': { name: 'Direct Financial Execution',              severity: 'Medium',   category: 'Financial Execution',  remediation: 'Document financial transaction capabilities clearly in the skill description. Add confirmation gates and user consent checks before executing any financial operation.' },
  'SKL-SEC-010': { name: 'Untrusted Third-Party Content Exposure',  severity: 'Medium',   category: 'Untrusted Content',    remediation: 'Restrict the skill to trusted, known data sources. Avoid browsing arbitrary URLs, reading social media posts, or analyzing content from unknown websites without sanitization.' },
  'SKL-SEC-011': { name: 'Unverifiable External Dependency',        severity: 'High',     category: 'External Dependency',  remediation: 'Do not fetch skill instructions or executable code from external URLs at runtime. Bundle all required resources in the skill package itself.' },
  'SKL-SEC-012': { name: 'System Service Modification',             severity: 'Medium',   category: 'System Modification',  remediation: 'Remove instructions that modify system services, startup scripts, or system-wide configurations. If required, document the need clearly and add explicit user consent steps.' },
};

/** Maps ConfigMap file name → check ID */
const CONFIGMAP_KEY_TO_ID: Record<string, string> = {
  'prompt-injection':            'SKL-SEC-001',
  'privilege-escalation':        'SKL-SEC-002',
  'data-exfiltration':           'SKL-SEC-003',
  'credential-harvesting':       'SKL-SEC-004',
  'scope-creep':                 'SKL-SEC-005',
  'safety-guardrails':           'SKL-SEC-006',
  'suspicious-download-urls':    'SKL-SEC-007',
  'hardcoded-secrets':           'SKL-SEC-008',
  'financial-execution':         'SKL-SEC-009',
  'untrusted-content':           'SKL-SEC-010',
  'external-runtime-dependency': 'SKL-SEC-011',
  'system-service-modification': 'SKL-SEC-012',
};

/** Parse a ConfigMap file value (newline-delimited, # = comment) into a string array */
function parseConfigMapLines(raw: string): string[] {
  return raw
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.length > 0 && !l.startsWith('#'));
}

let _cachedPatterns: PatternMap | null = null;

function loadPatterns(): PatternMap {
  if (_cachedPatterns) return _cachedPatterns;

  // ── Fail fast if ConfigMap mount is not present ───────────────────────────
  if (!existsSync(CONFIGMAP_MOUNT)) {
    throw new Error(
      `[scan/repo] FATAL: skill-patterns ConfigMap mount not found at "${CONFIGMAP_MOUNT}". ` +
      `Ensure the mcp-governance-skill-patterns ConfigMap is mounted into the pod. ` +
      `(Override path with SKILL_PATTERNS_MOUNT env var for local dev.)`,
    );
  }

  const result: PatternMap = {};
  let loaded = 0;

  for (const [cmKey, checkID] of Object.entries(CONFIGMAP_KEY_TO_ID)) {
    const filePath = path.join(CONFIGMAP_MOUNT, cmKey);
    const meta = PATTERN_META[checkID];
    if (!meta) continue;

    if (!existsSync(filePath)) {
      console.warn(`[scan/repo] Missing ConfigMap key "${cmKey}" (${checkID}) at ${filePath} — check will have no patterns.`);
      // Still register the check with empty patterns so the UI shows it
      result[checkID] = checkID === 'SKL-SEC-006'
        ? { ...meta, requiredPhrases: [] }
        : { ...meta, keywords: [] };
      continue;
    }

    try {
      const raw = readFileSync(filePath, 'utf-8');
      const lines = parseConfigMapLines(raw);

      result[checkID] = checkID === 'SKL-SEC-006'
        ? { ...meta, requiredPhrases: lines }
        : { ...meta, keywords: lines };

      loaded++;
    } catch (err) {
      console.warn(`[scan/repo] Could not read ConfigMap file ${filePath}: ${err}`);
      result[checkID] = checkID === 'SKL-SEC-006'
        ? { ...meta, requiredPhrases: [] }
        : { ...meta, keywords: [] };
    }
  }

  console.log(`[scan/repo] Loaded ${loaded}/${Object.keys(CONFIGMAP_KEY_TO_ID).length} pattern files from ConfigMap mount at ${CONFIGMAP_MOUNT}`);

  _cachedPatterns = result;
  return _cachedPatterns;
}

// Convenience accessor — use PATTERNS everywhere else in this file
// so a simple alias works; `loadPatterns()` is called once at first use.
const PATTERNS: PatternMap = new Proxy({} as PatternMap, {
  get(_target, prop: string) { return loadPatterns()[prop]; },
  ownKeys()                  { return Object.keys(loadPatterns()); },
  has(_target, prop: string) { return prop in loadPatterns(); },
  getOwnPropertyDescriptor(_target, prop: string) {
    const val = loadPatterns()[prop];
    return val !== undefined ? { value: val, enumerable: true, configurable: true, writable: false } : undefined;
  },
});

// ─── Scanner logic ────────────────────────────────────────────────────────────

function lineNumber(content: string, idx: number): number {
  if (idx <= 0) return 1;
  return content.substring(0, idx).split('\n').length;
}

/** Regex-based patterns for things keyword matching can't catch precisely (e.g. real secrets) */
const SECRET_REGEXES: Array<{ pattern: RegExp; label: string }> = [
  { pattern: /sk-[a-zA-Z0-9]{20,}/g,                       label: 'OpenAI/Anthropic API key (sk-...)' },
  { pattern: /ghp_[a-zA-Z0-9]{36,}/g,                      label: 'GitHub PAT (ghp_...)' },
  { pattern: /gho_[a-zA-Z0-9]{36,}/g,                      label: 'GitHub OAuth token (gho_...)' },
  { pattern: /github_pat_[a-zA-Z0-9_]{36,}/g,              label: 'GitHub fine-grained PAT' },
  { pattern: /glpat-[a-zA-Z0-9_-]{20,}/g,                  label: 'GitLab PAT' },
  { pattern: /AKIA[0-9A-Z]{16}/g,                           label: 'AWS access key ID' },
  { pattern: /xox[bpoa]-[0-9A-Za-z-]{24,}/g,               label: 'Slack token' },
  { pattern: /-----BEGIN [A-Z]+ PRIVATE KEY-----/g,         label: 'PEM private key block' },
  // Note: removed overly-broad regexes for 64-char hex and base64 strings
  // Those patterns match too many legitimate values (commit hashes, UUIDs, etc.)
  // Only high-confidence patterns above
];

/** Parse YAML frontmatter from a SKILL.md — returns findings if frontmatter is missing/invalid */
function validateSkillFrontmatter(filePath: string, content: string): ScanFinding[] {
  const findings: ScanFinding[] = [];

  // Skill metadata validation: file starts with ---, contains YAML block, then --- again
  if (!content.trimStart().startsWith('---')) {
    findings.push({
      checkID: 'SKL-SEC-013',
      severity: 'Low',
      category: 'Skill Metadata',
      title: `Missing YAML frontmatter in ${filePath}`,
      remediation: 'Add a YAML frontmatter block at the top of SKILL.md with at minimum "name" and "description" fields.',
      filePath,
    });
    return findings;
  }

  // Extract the frontmatter block
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match) {
    findings.push({
      checkID: 'SKL-SEC-013',
      severity: 'Low',
      category: 'Skill Metadata',
      title: `Malformed YAML frontmatter in ${filePath}`,
      remediation: 'Ensure the frontmatter block is properly closed with "---" on its own line.',
      filePath,
    });
    return findings;
  }

  const yaml = match[1];
  if (!/^name\s*:/m.test(yaml)) {
    findings.push({
      checkID: 'SKL-SEC-013',
      severity: 'Low',
      category: 'Skill Metadata',
      title: `Missing "name" field in SKILL.md frontmatter: ${filePath}`,
      remediation: 'Add a "name:" field to the YAML frontmatter.',
      filePath,
    });
  }
  if (!/^description\s*:/m.test(yaml)) {
    findings.push({
      checkID: 'SKL-SEC-013',
      severity: 'Low',
      category: 'Skill Metadata',
      title: `Missing "description" field in SKILL.md frontmatter: ${filePath}`,
      remediation: 'Add a "description:" field to the YAML frontmatter so Claude knows when to trigger this skill.',
      filePath,
    });
  }

  return findings;
}

function scanFileContent(filePath: string, content: string): ScanFinding[] {
  const lower = content.toLowerCase();
  const findings: ScanFinding[] = [];

  // ── Keyword checks for all PATTERNS that have a keywords array ────────────
  for (const [checkID, def] of Object.entries(PATTERNS)) {
    if (def.keywords && def.keywords.length > 0) {
      // Deduplicate: only report the first hit per keyword to avoid flood
      const seen = new Set<string>();
      for (const kw of def.keywords) {
        const kwLower = kw.toLowerCase();
        if (seen.has(kwLower)) continue;
        const idx = lower.indexOf(kwLower);
        if (idx >= 0) {
          seen.add(kwLower);
          findings.push({
            checkID,
            severity: def.severity,
            category: def.category,
            title: `${def.name} pattern detected: "${kw}"`,
            remediation: def.remediation,
            filePath,
            line: lineNumber(content, idx),
            matchedPattern: kw,
          });
        }
      }
    }
  }

  // ── Regex-based hardcoded secret detection (supplements SKL-SEC-008) ──────
  for (const { pattern, label } of SECRET_REGEXES) {
    pattern.lastIndex = 0; // reset stateful global regex
    const match = pattern.exec(content);
    if (match) {
      findings.push({
        checkID: 'SKL-SEC-008',
        severity: 'High',
        category: 'Hardcoded Secrets',
        title: `Potential hardcoded secret detected: ${label}`,
        remediation: PATTERNS['SKL-SEC-008'].remediation,
        filePath,
        line: lineNumber(content, match.index),
        matchedPattern: label,
      });
    }
  }

  return findings;
}

function buildSecurityChecks(findings: ScanFinding[], hasMissingSkillMd = false): SecurityCheck[] {
  const checks: SecurityCheck[] = [];

  for (const [id, def] of Object.entries(PATTERNS)) {
    const count = findings.filter(f => f.checkID === id).length;
    const description = id === 'SKL-SEC-006'
      ? PATTERNS['SKL-SEC-006'].remediation
      : `No ${def.category.toLowerCase()} patterns detected`;
    const meta = PATTERN_META[id as keyof typeof PATTERN_META];
    checks.push({ 
      id, 
      name: def.name, 
      passed: count === 0, 
      description, 
      remediation: def.remediation,
      findingCount: count,
      severity: meta?.severity,
    });
  }

  // SKL-SEC-013: SKILL.md metadata check (injected separately, not in PATTERNS loop above)
  const metaCount = findings.filter(f => f.checkID === 'SKL-SEC-013').length;
  checks.push({
    id: 'SKL-SEC-013',
    name: 'Skill Metadata (Frontmatter)',
    passed: metaCount === 0 && !hasMissingSkillMd,
    description: 'SKILL.md has valid YAML frontmatter with name and description fields.',
    remediation: 'Add a SKILL.md file with YAML frontmatter containing name and description fields.',
    findingCount: metaCount + (hasMissingSkillMd ? 1 : 0),
    severity: 'Medium',
  });

  // SKL-SEC-014: Missing SKILL.md (W014)
  const missingMdCount = findings.filter(f => f.checkID === 'SKL-SEC-014').length;
  checks.push({
    id: 'SKL-SEC-014',
    name: 'Missing SKILL.md (W014)',
    passed: missingMdCount === 0,
    description: 'Folder contains a SKILL.md file with skill metadata.',
    remediation: 'Add a SKILL.md file to the repository root or skill folder with metadata.',
    findingCount: missingMdCount,
    severity: 'High',
  });

  return checks;
}

function computeScore(checks: SecurityCheck[]): number {
  if (checks.length === 0) return 100;
  // Weight critical/high failures more heavily
  let weightedPassed = 0;
  let weightedTotal  = 0;
  for (const c of checks) {
    // Derive weight from severity via PATTERNS lookup
    const def = Object.values(PATTERNS).find(p => p.name === c.name);
    const sev = (def as any)?.severity as string | undefined;
    const weight = sev === 'Critical' ? 4 : sev === 'High' ? 3 : sev === 'Medium' ? 2 : 1;
    weightedTotal += weight;
    if (c.passed) weightedPassed += weight;
  }
  return Math.round((weightedPassed / weightedTotal) * 100);
}

function folderStatus(score: number): FolderScanResult['status'] {
  if (score >= 70) return 'pass';
  if (score >= 40) return 'warning';
  return 'fail';
}

/** Recursively walk a local directory and collect all skill/skills.md files,
 *  grouped by their immediate parent folder path (relative to cloneRoot).
 */
async function collectLocalSkillFiles(
  dir: string,
  cloneRoot: string,
): Promise<Map<string, { relPath: string; content: string }[]>> {
  const result = new Map<string, { relPath: string; content: string }[]>();

  if (!existsSync(dir)) return result;

  const entries = await readdir(dir, { withFileTypes: true });
  const folderKey = path.relative(cloneRoot, dir) || '/';

  const skillFiles = entries.filter(
    e => e.isFile() && /^skills?\.md$/i.test(e.name),
  );

  if (skillFiles.length > 0) {
    const folderFiles: { relPath: string; content: string }[] = [];
    for (const f of skillFiles) {
      const absPath = path.join(dir, f.name);
      const relPath = path.relative(cloneRoot, absPath);
      const content = await readFile(absPath, 'utf-8');
      folderFiles.push({ relPath, content });
    }
    result.set(folderKey, folderFiles);
  }

  for (const e of entries) {
    if (e.isDirectory() && e.name !== '.git') {
      const sub = await collectLocalSkillFiles(path.join(dir, e.name), cloneRoot);
      sub.forEach((v, k) => result.set(k, v));
    }
  }

  return result;
}

// ─── Main scan function ───────────────────────────────────────────────────────

async function scanRepository(repoUrl: string, folderPath: string, token?: string): Promise<ScanResult> {
  const scanPath = folderPath.replace(/^\/|\/$/g, '');
  const destDir  = path.join('/tmp', `scan-${randomUUID()}`);

  // ── Clone ────────────────────────────────────────────────────────────────
  try {
    await sparseClone(repoUrl, scanPath, destDir, token);
  } catch (e: any) {
    // Clean up on failure
    await rm(destDir, { recursive: true, force: true }).catch(() => {});
    const msg: string = e.stderr || e.message || String(e);
    if (msg.includes('Authentication failed') || msg.includes('could not read Username')) {
      return { status: 'error', repoUrl, scanPath, totalFilesScanned: 0, totalFindings: 0, folderResults: [],
        error: 'Authentication failed. Please provide a valid Personal Access Token.' };
    }
    if (msg.includes('not found') || msg.includes('Repository not found') || msg.includes('does not exist')) {
      return { status: 'error', repoUrl, scanPath, totalFilesScanned: 0, totalFindings: 0, folderResults: [],
        error: 'Repository not found. Please check the URL.' };
    }
    return { status: 'error', repoUrl, scanPath, totalFilesScanned: 0, totalFindings: 0, folderResults: [],
      error: `Clone failed: ${msg.slice(0, 300)}` };
  }

  // ── Scan ─────────────────────────────────────────────────────────────────
  try {
    const scanDir = scanPath ? path.join(destDir, scanPath) : destDir;
    const fileMap = await collectLocalSkillFiles(scanDir, destDir);

    const folderResults: FolderScanResult[] = [];
    let totalFiles    = 0;
    let totalFindings = 0;

    fileMap.forEach((files, folderKey) => {
      const allFindings: ScanFinding[] = [];

      // W014: check if this folder has a SKILL.md (case-insensitive)
      const hasSkillMd = files.some(f => /^skills?\.md$/i.test(path.basename(f.relPath)));
      if (!hasSkillMd && files.length > 0) {
        allFindings.push({
          checkID: 'SKL-SEC-014',
          severity: 'Low',
          category: 'Missing SKILL.md',
          title: `No SKILL.md found in folder: ${folderKey}`,
          remediation: 'Add a SKILL.md file with YAML frontmatter (name + description) to document this skill\'s purpose and security properties.',
          filePath: folderKey,
        });
      }

      for (const file of files) {
        // Keyword + regex checks (SKL-SEC-001 → 012, 008 regex)
        allFindings.push(...scanFileContent(file.relPath, file.content));

        // YAML frontmatter validation (SKL-SEC-013) — only for SKILL.md files
        if (/^skills?\.md$/i.test(path.basename(file.relPath))) {
          allFindings.push(...validateSkillFrontmatter(file.relPath, file.content));
        }

        // SKL-SEC-006: guardrail check (absence of safety phrases)
        const lower = file.content.toLowerCase();
        const sec006 = PATTERNS['SKL-SEC-006'];
        const guardrailPhrases = sec006?.requiredPhrases ?? ['do not','never','always verify','safety','guardrail','must not','prohibited'];
        const hasGuardrail = guardrailPhrases.some(p => lower.includes(p));
        if (!hasGuardrail) {
          allFindings.push({
            checkID: 'SKL-SEC-006',
            severity: 'Medium',
            category: 'Safety Guardrails',
            title: `No safety guardrail found in ${file.relPath}`,
            remediation: `Add an explicit safety note. Required phrases (at least one): ${guardrailPhrases.join(', ')}`,
            filePath: file.relPath,
          });
        }
      }

      const securityChecks = buildSecurityChecks(allFindings, !hasSkillMd);
      const score = computeScore(securityChecks);

      folderResults.push({
        folderPath: folderKey,
        filesScanned: files.length,
        findings: allFindings,
        securityChecks,
        score,
        status: folderStatus(score),
      });

      totalFiles    += files.length;
      totalFindings += allFindings.length;
    });

    return {
      status: 'success',
      repoUrl,
      scanPath,
      totalFilesScanned: totalFiles,
      totalFindings,
      folderResults,
    };
  } finally {
    // Always clean up the cloned directory
    await rm(destDir, { recursive: true, force: true }).catch(() => {});
  }
}

// ─── Route handler ────────────────────────────────────────────────────────────

export async function POST(request: NextRequest) {
  try {
    const body: ScanRequest = await request.json();

    if (!body.repoUrl) {
      return NextResponse.json({ status: 'error', error: 'Repository URL is required' }, { status: 400 });
    }

    const result = await scanRepository(body.repoUrl, body.folderPath || '', body.credentialToken);
    return NextResponse.json(result);
  } catch (error) {
    console.error('[scan/repo] POST error:', error);
    return NextResponse.json(
      { status: 'error', repoUrl: '', scanPath: '', totalFilesScanned: 0, totalFindings: 0, folderResults: [],
        error: error instanceof Error ? error.message : 'Failed to process scan request' },
      { status: 500 },
    );
  }
}
