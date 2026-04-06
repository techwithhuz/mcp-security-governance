// Package skillscanner provides security pattern matching for SkillCatalog resources.
// Patterns are loaded from a Kubernetes ConfigMap (mcp-governance-skill-patterns)
// and fall back to built-in defaults when the ConfigMap is not present.
package skillscanner

import (
	"strings"
)

// PatternSet holds all compiled regex/keyword patterns for a single scan cycle.
type PatternSet struct {
	// SKL-SEC-001: prompt injection patterns (keywords/phrases in skill content)
	PromptInjection []string

	// SKL-SEC-002: privilege escalation language
	PrivilegeEscalation []string

	// SKL-SEC-003: data exfiltration patterns
	DataExfiltration []string

	// SKL-SEC-004: credential harvesting patterns
	CredentialHarvesting []string

	// SKL-SEC-005: scope creep rules (category → forbidden keywords)
	ScopeCreep []ScopeCreepRule

	// SKL-SEC-006: categories that MUST mention safety guardrail phrases
	SafetyGuardrails []SafetyGuardrailRule

	// SKL-SEC-007: suspicious download URL patterns
	SuspiciousDownloadURLs []string

	// SKL-SEC-008: hardcoded secret keyword patterns
	HardcodedSecrets []string

	// SKL-SEC-009: direct financial execution patterns
	FinancialExecution []string

	// SKL-SEC-010: untrusted third-party content ingestion
	UntrustedContent []string

	// SKL-SEC-011: unverifiable external runtime dependency
	ExternalRuntimeDependency []string

	// SKL-SEC-012: system service modification patterns
	SystemServiceModification []string

	// SKL-SEC-013: required SKILL.md frontmatter field names
	RequiredFrontmatterFields []string
}

// ScopeCreepRule flags skills whose content contains out-of-scope keywords
// for their declared category.
type ScopeCreepRule struct {
	// Category is the SkillCatalog spec.category value this rule applies to.
	Category string

	// ForbiddenKeywords are keywords that should NOT appear in a skill
	// belonging to this category.
	ForbiddenKeywords []string
}

// SafetyGuardrailRule asserts that skills in high-risk categories must contain
// at least one safety phrase.
type SafetyGuardrailRule struct {
	// Category is the SkillCatalog spec.category this rule applies to.
	Category string

	// RequiredPhrases — skill content must contain at least one of these.
	RequiredPhrases []string
}

// DefaultPatternSet returns the built-in pattern set used when the ConfigMap
// is absent or a key is missing.
func DefaultPatternSet() *PatternSet {
	return &PatternSet{
		PromptInjection: []string{
			"ignore previous instructions",
			"ignore all previous",
			"disregard your instructions",
			"forget your previous instructions",
			"your new instructions are",
			"override your system prompt",
			"new system prompt",
			"ignore the above",
			"do not follow",
			"bypass safety",
			"jailbreak",
			"DAN mode",
			"developer mode enabled",
			"pretend you are",
			"act as if you have no restrictions",
			"simulate being",
		},
		PrivilegeEscalation: []string{
			"sudo",
			"run as root",
			"escalate privileges",
			"become administrator",
			"gain elevated access",
			"bypass authentication",
			"disable security",
			"setuid",
			"chmod 777",
			"chmod +s",
			"privilege escalation",
			"run with elevated",
			"execute as administrator",
			"administrator privileges",
		},
		DataExfiltration: []string{
			"exfiltrate",
			"send data to external",
			"upload to external",
			"transmit to remote",
			"leak data",
			"send credentials",
			"forward secrets",
			"export sensitive",
			"copy to external storage",
			"transfer to third party",
			"webhook.site",
			"requestbin",
			"burpcollaborator",
			"oastify.com",
			"pipedream.net",
		},
		CredentialHarvesting: []string{
			"steal credentials",
			"harvest passwords",
			"capture tokens",
			"extract api keys",
			"collect secrets",
			"phishing",
			"credential stuffing",
			"brute force password",
			"enumerate users",
			"dump credentials",
			"keylogger",
			"password spraying",
			"capture authentication",
		},
		ScopeCreep: []ScopeCreepRule{
			{
				Category:          "data",
				ForbiddenKeywords: []string{"deploy", "provision infrastructure", "create cluster", "manage kubernetes"},
			},
			{
				Category:          "analytics",
				ForbiddenKeywords: []string{"delete database", "drop table", "truncate", "delete all records"},
			},
			{
				Category:          "communication",
				ForbiddenKeywords: []string{"execute code", "run shell", "bash -c", "system command"},
			},
			{
				Category:          "productivity",
				ForbiddenKeywords: []string{"access production", "modify live data", "delete records", "admin console"},
			},
		},
		SafetyGuardrails: []SafetyGuardrailRule{
			{
				Category: "database",
				RequiredPhrases: []string{
					"confirmation required",
					"dry run",
					"requires approval",
					"cannot be undone",
					"irreversible",
					"backup recommended",
				},
			},
			{
				Category: "infra",
				RequiredPhrases: []string{
					"confirmation required",
					"requires approval",
					"change management",
					"review before applying",
					"dry run",
				},
			},
			{
				Category: "admin",
				RequiredPhrases: []string{
					"admin access required",
					"requires elevated privileges",
					"audit logged",
					"requires approval",
					"multi-factor",
				},
			},
		},
		SuspiciousDownloadURLs: []string{
			".exe", ".bat", ".cmd", ".dmg", ".msi", ".ps1",
			"bit.ly/", "tinyurl.com/", "t.co/", "dropbox.com/s/", "mega.nz/",
			"releases/download/",
			"authtool", "osascript -e",
			"curl -fsSL", "curl -o /tmp", "wget -q -O", "wget --quiet",
		},
		HardcodedSecrets: []string{
			"sk-proj-", "sk-ant-", "ghp_", "gho_", "github_pat_", "glpat-", "AKIA",
			"xoxb-", "xoxp-", "xoxa-",
			"private_key_here", "insert_your_api_key", "replace_with_token",
			"seed phrase", "mnemonic phrase", "wallet mnemonic", "recovery phrase",
		},
		FinancialExecution: []string{
			"execute trade", "execute swap", "place order", "submit order",
			"confirm purchase", "initiate payment", "process payment",
			"wire transfer", "send funds", "transfer funds", "withdraw funds",
			"defi trading", "snipe token", "front-run",
			"payment processing", "billing execution", "charge customer",
			"private key", "wallet private key", "sign transaction",
		},
		UntrustedContent: []string{
			"browse any url", "fetch any url", "fetch any website", "fetch any link",
			"load any url", "read any website", "visit any url",
			"scrape twitter", "scrape reddit", "scrape any website",
			"read social media", "fetch user-supplied url", "follow any link",
		},
		ExternalRuntimeDependency: []string{
			"fetch instructions from", "load instructions from",
			"auto-update skill", "self-update",
			"fetch and execute", "download and run", "download and execute",
			"curl | python", "curl | bash", "curl | sh",
			"wget | python", "wget | bash",
			"pipe to shell", "execute downloaded", "run downloaded script",
		},
		SystemServiceModification: []string{
			"launchctl load", "launchctl start", "launchctl stop", "launchctl enable",
			"systemctl enable", "systemctl disable", "systemctl start", "systemctl stop",
			"service install", "install service",
			"crontab -e", "cron job",
			"/etc/rc.local", "/etc/init.d/", "/etc/sudoers",
			`HKLM\Software\Microsoft\Windows\CurrentVersion\Run`,
			"registry run key", "startup registry", "persistence mechanism",
		},
		RequiredFrontmatterFields: []string{
			"name",
			"description",
		},
	}
}

// ParsePatternSet builds a PatternSet from ConfigMap key→value pairs.
// Missing keys fall back to the built-in defaults.
func ParsePatternSet(data map[string]string) *PatternSet {
	defaults := DefaultPatternSet()

	ps := &PatternSet{}

	if raw, ok := data["prompt-injection"]; ok {
		ps.PromptInjection = parseLines(raw)
	} else {
		ps.PromptInjection = defaults.PromptInjection
	}

	if raw, ok := data["privilege-escalation"]; ok {
		ps.PrivilegeEscalation = parseLines(raw)
	} else {
		ps.PrivilegeEscalation = defaults.PrivilegeEscalation
	}

	if raw, ok := data["data-exfiltration"]; ok {
		ps.DataExfiltration = parseLines(raw)
	} else {
		ps.DataExfiltration = defaults.DataExfiltration
	}

	if raw, ok := data["credential-harvesting"]; ok {
		ps.CredentialHarvesting = parseLines(raw)
	} else {
		ps.CredentialHarvesting = defaults.CredentialHarvesting
	}

	if raw, ok := data["scope-creep"]; ok {
		ps.ScopeCreep = parseScopeCreepRules(raw)
	} else {
		ps.ScopeCreep = defaults.ScopeCreep
	}

	if raw, ok := data["safety-guardrails"]; ok {
		ps.SafetyGuardrails = parseSafetyGuardrails(raw)
	} else {
		ps.SafetyGuardrails = defaults.SafetyGuardrails
	}

	if raw, ok := data["suspicious-download-urls"]; ok {
		ps.SuspiciousDownloadURLs = parseLines(raw)
	} else {
		ps.SuspiciousDownloadURLs = defaults.SuspiciousDownloadURLs
	}

	if raw, ok := data["hardcoded-secrets"]; ok {
		ps.HardcodedSecrets = parseLines(raw)
	} else {
		ps.HardcodedSecrets = defaults.HardcodedSecrets
	}

	if raw, ok := data["financial-execution"]; ok {
		ps.FinancialExecution = parseLines(raw)
	} else {
		ps.FinancialExecution = defaults.FinancialExecution
	}

	if raw, ok := data["untrusted-content"]; ok {
		ps.UntrustedContent = parseLines(raw)
	} else {
		ps.UntrustedContent = defaults.UntrustedContent
	}

	if raw, ok := data["external-runtime-dependency"]; ok {
		ps.ExternalRuntimeDependency = parseLines(raw)
	} else {
		ps.ExternalRuntimeDependency = defaults.ExternalRuntimeDependency
	}

	if raw, ok := data["system-service-modification"]; ok {
		ps.SystemServiceModification = parseLines(raw)
	} else {
		ps.SystemServiceModification = defaults.SystemServiceModification
	}

	if raw, ok := data["required-frontmatter-fields"]; ok {
		ps.RequiredFrontmatterFields = parseLines(raw)
	} else {
		ps.RequiredFrontmatterFields = defaults.RequiredFrontmatterFields
	}

	return ps
}

// parseLines splits a multi-line string into non-empty trimmed lines,
// ignoring lines that start with '#' (comments).
func parseLines(raw string) []string {
	var result []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result = append(result, line)
	}
	return result
}

// parseScopeCreepRules parses the scope-creep ConfigMap value.
//
// Format:
//
//	# category: forbidden keyword1, keyword2
//	data: deploy, provision infrastructure
//	analytics: delete database, drop table
func parseScopeCreepRules(raw string) []ScopeCreepRule {
	var rules []ScopeCreepRule
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		category := strings.TrimSpace(parts[0])
		var keywords []string
		for _, kw := range strings.Split(parts[1], ",") {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				keywords = append(keywords, kw)
			}
		}
		if category != "" && len(keywords) > 0 {
			rules = append(rules, ScopeCreepRule{
				Category:          category,
				ForbiddenKeywords: keywords,
			})
		}
	}
	return rules
}

// parseSafetyGuardrails parses the safety-guardrails ConfigMap value.
//
// Format:
//
//	# category: required phrase1, required phrase2
//	database: confirmation required, dry run
//	infra: requires approval, change management
func parseSafetyGuardrails(raw string) []SafetyGuardrailRule {
	var rules []SafetyGuardrailRule
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		category := strings.TrimSpace(parts[0])
		var phrases []string
		for _, ph := range strings.Split(parts[1], ",") {
			ph = strings.TrimSpace(ph)
			if ph != "" {
				phrases = append(phrases, ph)
			}
		}
		if category != "" && len(phrases) > 0 {
			rules = append(rules, SafetyGuardrailRule{
				Category:        category,
				RequiredPhrases: phrases,
			})
		}
	}
	return rules
}
