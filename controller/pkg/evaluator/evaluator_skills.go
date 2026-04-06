package evaluator

import (
	"fmt"
	"log"
	"strings"

	"github.com/techwithhuz/mcp-security-governance/controller/pkg/skillscanner"
)

// checkSkillCatalogs is the entry-point for all SkillCatalog governance checks.
// It runs metadata checks (SKL-001 to SKL-008) against every discovered
// SkillCatalog CR, and optionally fetches + scans repository content
// (SKL-SEC-001 to SKL-SEC-006) when policy.SkillGovernance.ScanRepoContent is true.
//
// The function returns a []Finding (for the global findings list) and also builds
// a []SkillCatalogScore (written into result.SkillCatalogScores).
func checkSkillCatalogs(state *ClusterState, policy Policy, patternLoader *skillscanner.PatternLoader) ([]Finding, []SkillCatalogScore) {
	if !policy.SkillGovernance.Enabled || len(state.SkillCatalogs) == 0 {
		return nil, nil
	}

	var findings []Finding
	var scores []SkillCatalogScore

	for _, skill := range state.SkillCatalogs {
		ref := fmt.Sprintf("SkillCatalog/%s/%s", skill.Namespace, skill.Name)

		// --- Metadata checks (SKL-001 to SKL-008) ---
		metaFindings := checkSkillMetadata(skill, ref)
		findings = append(findings, metaFindings...)

		// --- Repo content scanning (SKL-SEC-001 to SKL-SEC-006) ---
		var contentFindings []Finding
		scannedFiles := 0
		securityScanned := false

		if policy.SkillGovernance.ScanRepoContent && skill.RepoSource == "github" && skill.RepoURL != "" {
			ps := patternLoader.Get()
			contentFindings, scannedFiles = scanSkillRepo(skill, policy, ps)
			findings = append(findings, contentFindings...)
			securityScanned = true
		}

		// --- Build per-catalog score ---
		score := scoreSkillCatalog(skill, metaFindings, contentFindings, policy)
		score.ScannedFiles = scannedFiles
		score.SecurityScanned = securityScanned
		scores = append(scores, score)
	}

	return findings, scores
}

// checkSkillMetadata performs static (no network) governance checks on a SkillCatalog CR.
func checkSkillMetadata(skill SkillCatalogResource, ref string) []Finding {
	var findings []Finding

	// SKL-001: Missing version
	if skill.Version == "" {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-001-%s", skill.Name),
			Severity:    SeverityMedium,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("SkillCatalog '%s' has no version", skill.Name),
			Description: fmt.Sprintf("The SkillCatalog '%s' (namespace: %s) does not declare a version in spec.version.", skill.Name, skill.Namespace),
			Impact:      "Without a version pin, the skill cannot be audited or rolled back reliably.",
			Remediation: "Set spec.version to a semantic version string (e.g. '1.0.0').",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	}

	// SKL-002: Unknown or missing repository source
	knownSources := map[string]bool{"github": true, "gitlab": true, "bitbucket": true}
	if skill.RepoSource == "" {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-002-%s", skill.Name),
			Severity:    SeverityLow,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("SkillCatalog '%s' has no repository source", skill.Name),
			Description: "spec.repository.source is empty.",
			Impact:      "Without a known source, automated scanning and provenance checks cannot be performed.",
			Remediation: "Set spec.repository.source to 'github', 'gitlab', or 'bitbucket'.",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	} else if !knownSources[strings.ToLower(skill.RepoSource)] {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-002-%s", skill.Name),
			Severity:    SeverityLow,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("SkillCatalog '%s' uses unknown repository source '%s'", skill.Name, skill.RepoSource),
			Description: fmt.Sprintf("spec.repository.source is '%s', which is not a recognised source.", skill.RepoSource),
			Impact:      "Automated repo scanning only supports github/gitlab/bitbucket.",
			Remediation: "Use a supported source value or extend the scanner for this VCS.",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	}

	// SKL-003: HTTP (non-HTTPS) repository URL — critical
	if skill.RepoURL != "" && strings.HasPrefix(strings.ToLower(skill.RepoURL), "http://") {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-003-%s", skill.Name),
			Severity:    SeverityCritical,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("SkillCatalog '%s' uses plain HTTP repository URL", skill.Name),
			Description: fmt.Sprintf("spec.repository.url is '%s', which uses unencrypted HTTP.", skill.RepoURL),
			Impact:      "An HTTP URL allows man-in-the-middle attacks that could inject malicious skill content.",
			Remediation: "Change spec.repository.url to use HTTPS.",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	}

	// SKL-004: Missing resource-uid label
	if skill.ResourceUID == "" {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-004-%s", skill.Name),
			Severity:    SeverityLow,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("SkillCatalog '%s' missing resource-uid label", skill.Name),
			Description: "The label 'agentregistry.dev/resource-uid' is not set.",
			Impact:      "Without a unique resource UID, deduplication and audit trails are unreliable.",
			Remediation: "Add the label 'agentregistry.dev/resource-uid' with a unique value.",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	}

	// SKL-005: Missing category
	if skill.Category == "" {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-005-%s", skill.Name),
			Severity:    SeverityLow,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("SkillCatalog '%s' has no category", skill.Name),
			Description: "spec.category is empty.",
			Impact:      "Without a category, scope-creep and safety guardrail checks cannot be applied.",
			Remediation: "Set spec.category to a meaningful value (e.g. 'data', 'analytics', 'infra', 'admin').",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	}

	// SKL-006: Empty or too-short description (< 20 chars)
	if len(strings.TrimSpace(skill.Description)) < 20 {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-006-%s", skill.Name),
			Severity:    SeverityLow,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("SkillCatalog '%s' has an insufficient description", skill.Name),
			Description: fmt.Sprintf("spec.description is %d characters, which is below the 20-character minimum.", len(strings.TrimSpace(skill.Description))),
			Impact:      "Poor descriptions make it difficult to understand skill purpose during security reviews.",
			Remediation: "Provide a meaningful description (at least 20 characters) in spec.description.",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	}

	// SKL-007: Production environment without a version pin
	env := strings.ToLower(skill.Environment)
	if (env == "prod" || env == "production") && skill.Version == "" {
		findings = append(findings, Finding{
			ID:          fmt.Sprintf("SKL-007-%s", skill.Name),
			Severity:    SeverityHigh,
			Category:    "Skill Governance",
			Title:       fmt.Sprintf("Production SkillCatalog '%s' has no version pin", skill.Name),
			Description: "The skill is deployed in a production environment but has no version specified.",
			Impact:      "Unpinned skills in production can receive unexpected updates, leading to unaudited behaviour changes.",
			Remediation: "Pin the skill to a specific version (spec.version) before deploying to production.",
			ResourceRef: ref,
			Namespace:   skill.Namespace,
		})
	}

	// SKL-008: Personal GitHub account URL (not an org/enterprise URL)
	// Heuristic: github.com/[single-word-with-no-dashes] is often a personal account
	if skill.RepoURL != "" {
		if owner, _, err := parseGitHubOwnerRepo(skill.RepoURL); err == nil {
			if isLikelyPersonalAccount(owner) {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("SKL-008-%s", skill.Name),
					Severity:    SeverityMedium,
					Category:    "Skill Governance",
					Title:       fmt.Sprintf("SkillCatalog '%s' may reference a personal GitHub account (%s)", skill.Name, owner),
					Description: "The repository URL appears to reference a personal GitHub account rather than an organisation.",
					Impact:      "Personal repositories have fewer security controls and may not meet enterprise provenance requirements.",
					Remediation: "Host production skills in a GitHub organisation or enterprise account with branch protection and code review policies.",
					ResourceRef: ref,
					Namespace:   skill.Namespace,
				})
			}
		}
	}

	return findings
}

// scanSkillRepo fetches and pattern-scans the GitHub repository content
// for a SkillCatalog. Returns governance findings and the number of files scanned.
func scanSkillRepo(skill SkillCatalogResource, policy Policy, ps *skillscanner.PatternSet) ([]Finding, int) {
	log.Printf("[skillscanner] scanning repo content for SkillCatalog %s/%s (%s)", skill.Namespace, skill.Name, skill.RepoURL)

	files, err := skillscanner.FetchSkillFiles(skill.RepoURL, policy.SkillGovernance.GitHubToken)
	if err != nil {
		log.Printf("[skillscanner] failed to fetch files for %s: %v", skill.Name, err)
		return nil, 0
	}

	ref := fmt.Sprintf("SkillCatalog/%s/%s", skill.Namespace, skill.Name)
	var findings []Finding

	for _, file := range files {
		skillFindings := skillscanner.ScanContent(file.Path, file.Content, ps, skill.Category)
		for _, sf := range skillFindings {
			// Apply policy suppressions
			if sf.CheckID == "SKL-SEC-001" && !policy.SkillGovernance.FailOnPromptInjection {
				continue
			}
			if sf.CheckID == "SKL-SEC-002" && !policy.SkillGovernance.FailOnPrivilegeEscalation {
				continue
			}
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("%s-%s-%s", sf.CheckID, skill.Name, sanitiseID(file.Path)),
				Severity:    sf.Severity,
				Category:    "Skill Security",
				Title:       sf.Title,
				Description: fmt.Sprintf("Pattern '%s' found in file '%s' (line %d) of SkillCatalog '%s'.", sf.MatchedPattern, sf.FilePath, sf.Line, skill.Name),
				Impact:      "Malicious or misuse-enabling patterns in skill content can compromise the AI agent.",
				Remediation: sf.Remediation,
				ResourceRef: ref,
				Namespace:   skill.Namespace,
			})
		}
	}

	return findings, len(files)
}

// scoreSkillCatalog computes a 0–100 score for a single SkillCatalog based on findings.
func scoreSkillCatalog(skill SkillCatalogResource, metaFindings, contentFindings []Finding, policy Policy) SkillCatalogScore {
	score := 100
	penalty := 0

	allFindings := append(metaFindings, contentFindings...)

	for _, f := range allFindings {
		switch f.Severity {
		case SeverityCritical:
			penalty += policy.SeverityPenalties.Critical
		case SeverityHigh:
			penalty += policy.SeverityPenalties.High
		case SeverityMedium:
			penalty += policy.SeverityPenalties.Medium
		case SeverityLow:
			penalty += policy.SeverityPenalties.Low
		}
	}
	score -= penalty
	if score < 0 {
		score = 0
	}

	status := "pass"
	switch {
	case score < 50:
		status = "fail"
	case score < 80:
		status = "warning"
	}

	// Build serialisable finding list
	var sFindings []SkillCatalogFinding
	for _, f := range allFindings {
		sFindings = append(sFindings, SkillCatalogFinding{
			CheckID:     f.ID,
			Severity:    f.Severity,
			Category:    f.Category,
			Title:       f.Title,
			Remediation: f.Remediation,
		})
	}

	return SkillCatalogScore{
		Name:       skill.Name,
		Namespace:  skill.Namespace,
		Version:    skill.Version,
		Category:   skill.Category,
		RepoURL:    skill.RepoURL,
		WebsiteURL: skill.WebsiteURL,
		Score:      score,
		Status:     status,
		Findings:   sFindings,
	}
}

// --- helpers ---

// parseGitHubOwnerRepo is a lightweight URL parser used only for SKL-008.
func parseGitHubOwnerRepo(repoURL string) (owner, repo string, err error) {
	url := strings.TrimSuffix(strings.TrimSpace(repoURL), ".git")
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "git@github.com:"} {
		if strings.HasPrefix(url, prefix) {
			path := strings.TrimPrefix(url, prefix)
			parts := strings.SplitN(path, "/", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], nil
			}
		}
	}
	return "", "", fmt.Errorf("not a GitHub URL")
}

// isLikelyPersonalAccount heuristic: owner with no hyphens and no uppercase
// often indicates a personal GitHub handle rather than an org name like
// "my-company" or "MyOrg".
func isLikelyPersonalAccount(owner string) bool {
	// Known orgs / bots to exclude from the check
	knownOrgs := map[string]bool{
		"anthropics": true, "openai": true, "google": true, "microsoft": true,
		"hashicorp": true, "kubernetes": true, "helm": true,
	}
	if knownOrgs[strings.ToLower(owner)] {
		return false
	}
	// Short, all-lowercase, no-hyphen names are the most common personal accounts
	return !strings.Contains(owner, "-") && !strings.Contains(owner, "_") && owner == strings.ToLower(owner)
}

// sanitiseID replaces path separators with dashes for use in finding IDs.
func sanitiseID(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return s
}

// ── Exported wrappers for use by cmd/api/main.go ─────────────────────────────

// CheckSkillMetadataExported is an exported wrapper for checkSkillMetadata.
func CheckSkillMetadataExported(skill SkillCatalogResource, ref string) []Finding {
	return checkSkillMetadata(skill, ref)
}

// ScanSkillRepoExported is an exported wrapper for scanSkillRepo.
func ScanSkillRepoExported(skill SkillCatalogResource, policy Policy, ps *skillscanner.PatternSet) ([]Finding, int) {
	return scanSkillRepo(skill, policy, ps)
}

// ScoreSkillCatalogExported is an exported wrapper for scoreSkillCatalog.
func ScoreSkillCatalogExported(skill SkillCatalogResource, metaFindings, contentFindings []Finding, policy Policy) SkillCatalogScore {
	return scoreSkillCatalog(skill, metaFindings, contentFindings, policy)
}
