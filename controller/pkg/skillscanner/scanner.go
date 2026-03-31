// Package skillscanner provides GitHub API-based content scanning for SkillCatalog resources.
package skillscanner

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SkillFile represents a single file fetched from a GitHub repository.
type SkillFile struct {
	Path    string
	Content string
}

// SkillFinding is a security issue found while pattern-matching a skill file.
type SkillFinding struct {
	// CheckID is e.g. "SKL-SEC-001"
	CheckID string

	// Severity is one of "Critical", "High", "Medium", "Low"
	Severity string

	// Category is the check category label
	Category string

	// FilePath is the repository-relative path of the offending file
	FilePath string

	// Line is the approximate line number (1-based; 0 = whole file)
	Line int

	// MatchedPattern is the keyword/phrase that triggered this finding
	MatchedPattern string

	// Title is a one-line description
	Title string

	// Remediation is an actionable suggestion
	Remediation string
}

// PatternLoader loads PatternSet from a mounted ConfigMap directory and
// refreshes it every reloadTTL.
type PatternLoader struct {
	mu         sync.RWMutex
	mountPath  string
	patterns   *PatternSet
	loadedAt   time.Time
	reloadTTL  time.Duration
}

// NewPatternLoader creates a PatternLoader that reads from mountPath.
// mountPath should be the directory where the ConfigMap is mounted
// (e.g. /etc/mcp-governance/skill-patterns).
func NewPatternLoader(mountPath string) *PatternLoader {
	pl := &PatternLoader{
		mountPath: mountPath,
		reloadTTL: 30 * time.Second,
	}
	pl.load()
	return pl
}

// Get returns the current PatternSet, reloading from disk when the TTL has expired.
func (pl *PatternLoader) Get() *PatternSet {
	pl.mu.RLock()
	needsReload := pl.patterns == nil || time.Since(pl.loadedAt) > pl.reloadTTL
	pl.mu.RUnlock()

	if needsReload {
		pl.load()
	}

	pl.mu.RLock()
	defer pl.mu.RUnlock()
	return pl.patterns
}

// load reads the ConfigMap mount directory and parses all key files.
// Falls back to DefaultPatternSet() on any error.
func (pl *PatternLoader) load() {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	data := make(map[string]string)

	keys := []string{
		"prompt-injection",
		"privilege-escalation",
		"data-exfiltration",
		"credential-harvesting",
		"scope-creep",
		"safety-guardrails",
	}

	for _, key := range keys {
		path := filepath.Join(pl.mountPath, key)
		content, err := os.ReadFile(path)
		if err != nil {
			// File missing or unreadable – skip this key (defaults will be used)
			continue
		}
		data[key] = string(content)
	}

	if len(data) == 0 {
		log.Printf("[skillscanner] ConfigMap mount not found at %s, using built-in defaults", pl.mountPath)
		pl.patterns = DefaultPatternSet()
	} else {
		log.Printf("[skillscanner] Loaded skill patterns from %s (%d keys)", pl.mountPath, len(data))
		pl.patterns = ParsePatternSet(data)
	}
	pl.loadedAt = time.Now()
}

// githubContentsAPIResponse maps the GitHub Contents API response.
type githubContentsAPIResponse struct {
	Type     string `json:"type"` // "file" or "dir"
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"` // "base64"
	HTMLURL  string `json:"html_url"`
}

// FetchSkillFiles fetches Markdown files from the GitHub repository referenced
// in a SkillCatalog's spec.repository.url.
// repoURL is expected to be a GitHub HTTPS URL, e.g.:
//
//	https://github.com/anthropics/skills.git
//	https://github.com/anthropics/skills
//
// token is an optional personal access token for private repos.
// Returns at most maxFiles file contents to avoid excessive API calls.
func FetchSkillFiles(repoURL string, token string) ([]SkillFile, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, err
	}

	// List the root directory to find Markdown files
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/", owner, repo)
	entries, err := listGitHubContents(apiURL, token)
	if err != nil {
		return nil, fmt.Errorf("list root contents: %w", err)
	}

	var files []SkillFile
	const maxFiles = 20

	for _, entry := range entries {
		if len(files) >= maxFiles {
			break
		}

		if entry.Type == "file" && isMarkdown(entry.Name) {
			content, err := fetchFileContent(entry.Path, owner, repo, token)
			if err != nil {
				log.Printf("[skillscanner] skip %s: %v", entry.Path, err)
				continue
			}
			files = append(files, SkillFile{
				Path:    entry.Path,
				Content: content,
			})
		} else if entry.Type == "dir" && isSkillDir(entry.Name) {
			// One level of subdirectory traversal for common skill dirs
			subURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, entry.Path)
			subEntries, err := listGitHubContents(subURL, token)
			if err != nil {
				continue
			}
			for _, sub := range subEntries {
				if len(files) >= maxFiles {
					break
				}
				if sub.Type == "file" && isMarkdown(sub.Name) {
					content, err := fetchFileContent(sub.Path, owner, repo, token)
					if err != nil {
						log.Printf("[skillscanner] skip %s: %v", sub.Path, err)
						continue
					}
					files = append(files, SkillFile{
						Path:    sub.Path,
						Content: content,
					})
				}
			}
		}
	}

	return files, nil
}

// ScanContent runs all pattern checks against a single file's content.
// category is the SkillCatalog.spec.category value.
func ScanContent(filePath string, content string, ps *PatternSet, skillCategory string) []SkillFinding {
	lower := strings.ToLower(content)
	var findings []SkillFinding

	// SKL-SEC-001: Prompt injection
	for _, pat := range ps.PromptInjection {
		if idx := strings.Index(lower, strings.ToLower(pat)); idx >= 0 {
			line := lineNumber(content, idx)
			findings = append(findings, SkillFinding{
				CheckID:        "SKL-SEC-001",
				Severity:       "Critical",
				Category:       "Prompt Injection",
				FilePath:       filePath,
				Line:           line,
				MatchedPattern: pat,
				Title:          fmt.Sprintf("Prompt injection pattern detected: %q", pat),
				Remediation:    "Remove or rewrite the instruction that could override the AI agent's system prompt.",
			})
		}
	}

	// SKL-SEC-002: Privilege escalation
	for _, pat := range ps.PrivilegeEscalation {
		if idx := strings.Index(lower, strings.ToLower(pat)); idx >= 0 {
			line := lineNumber(content, idx)
			findings = append(findings, SkillFinding{
				CheckID:        "SKL-SEC-002",
				Severity:       "Critical",
				Category:       "Privilege Escalation",
				FilePath:       filePath,
				Line:           line,
				MatchedPattern: pat,
				Title:          fmt.Sprintf("Privilege escalation language detected: %q", pat),
				Remediation:    "Remove privilege escalation language. Skills should operate with least-privilege principles.",
			})
		}
	}

	// SKL-SEC-003: Data exfiltration
	for _, pat := range ps.DataExfiltration {
		if idx := strings.Index(lower, strings.ToLower(pat)); idx >= 0 {
			line := lineNumber(content, idx)
			findings = append(findings, SkillFinding{
				CheckID:        "SKL-SEC-003",
				Severity:       "High",
				Category:       "Data Exfiltration",
				FilePath:       filePath,
				Line:           line,
				MatchedPattern: pat,
				Title:          fmt.Sprintf("Data exfiltration pattern detected: %q", pat),
				Remediation:    "Remove references to external data transmission. Ensure data flows comply with your data governance policy.",
			})
		}
	}

	// SKL-SEC-004: Credential harvesting
	for _, pat := range ps.CredentialHarvesting {
		if idx := strings.Index(lower, strings.ToLower(pat)); idx >= 0 {
			line := lineNumber(content, idx)
			findings = append(findings, SkillFinding{
				CheckID:        "SKL-SEC-004",
				Severity:       "Critical",
				Category:       "Credential Harvesting",
				FilePath:       filePath,
				Line:           line,
				MatchedPattern: pat,
				Title:          fmt.Sprintf("Credential harvesting language detected: %q", pat),
				Remediation:    "Remove any instructions that collect, capture, or forward credentials or secrets.",
			})
		}
	}

	// SKL-SEC-005: Scope creep — only check rules matching this skill's category
	for _, rule := range ps.ScopeCreep {
		if !strings.EqualFold(rule.Category, skillCategory) {
			continue
		}
		for _, kw := range rule.ForbiddenKeywords {
			if idx := strings.Index(lower, strings.ToLower(kw)); idx >= 0 {
				line := lineNumber(content, idx)
				findings = append(findings, SkillFinding{
					CheckID:        "SKL-SEC-005",
					Severity:       "High",
					Category:       "Scope Creep",
					FilePath:       filePath,
					Line:           line,
					MatchedPattern: kw,
					Title:          fmt.Sprintf("Out-of-scope keyword %q found in %s skill", kw, skillCategory),
					Remediation:    fmt.Sprintf("Remove actions outside the declared category %q, or update the category to reflect the skill's true scope.", skillCategory),
				})
			}
		}
	}

	// SKL-SEC-006: Missing safety guardrails — only check rules matching this category
	for _, rule := range ps.SafetyGuardrails {
		if !strings.EqualFold(rule.Category, skillCategory) {
			continue
		}
		found := false
		for _, phrase := range rule.RequiredPhrases {
			if strings.Contains(lower, strings.ToLower(phrase)) {
				found = true
				break
			}
		}
		if !found {
			findings = append(findings, SkillFinding{
				CheckID:  "SKL-SEC-006",
				Severity: "Medium",
				Category: "Safety Guardrails",
				FilePath: filePath,
				Title:    fmt.Sprintf("No safety guardrail found in %s skill file %s", skillCategory, filePath),
				Remediation: fmt.Sprintf(
					"Add an explicit safety note to the skill file. Required phrases (at least one): %s",
					strings.Join(rule.RequiredPhrases, ", "),
				),
			})
		}
	}

	return findings
}

// --- internal helpers ---

// parseGitHubURL extracts the owner and repo name from a GitHub URL.
func parseGitHubURL(repoURL string) (owner, repo string, err error) {
	// Remove .git suffix
	repoURL = strings.TrimSuffix(strings.TrimSpace(repoURL), ".git")

	// Handle https://github.com/owner/repo
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "git@github.com:"} {
		if strings.HasPrefix(repoURL, prefix) {
			path := strings.TrimPrefix(repoURL, prefix)
			parts := strings.SplitN(path, "/", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], nil
			}
		}
	}

	return "", "", fmt.Errorf("unsupported GitHub URL format: %q", repoURL)
}

// listGitHubContents calls the GitHub Contents API and returns directory entries.
func listGitHubContents(apiURL string, token string) ([]githubContentsAPIResponse, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository or path not found (404)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var entries []githubContentsAPIResponse
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal directory listing: %w", err)
	}

	return entries, nil
}

// fetchFileContent fetches and decodes a single file from the GitHub Contents API.
func fetchFileContent(path, owner, repo, token string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var fileResp githubContentsAPIResponse
	if err := json.Unmarshal(body, &fileResp); err != nil {
		return "", fmt.Errorf("unmarshal file response: %w", err)
	}

	if fileResp.Encoding != "base64" {
		return fileResp.Content, nil
	}

	// GitHub wraps base64 lines with \n — strip them before decoding
	cleaned := strings.ReplaceAll(fileResp.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", fmt.Errorf("base64 decode %s: %w", path, err)
	}

	return string(decoded), nil
}

// isMarkdown returns true for common Markdown file extensions.
func isMarkdown(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".mdx") ||
		strings.HasSuffix(lower, ".markdown")
}

// isSkillDir returns true if a directory name looks like it contains skill files.
func isSkillDir(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range []string{"skill", "skills", "agent", "prompt", "prompts", "docs", "instructions"} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// lineNumber returns the 1-based line number of the byte offset idx in content.
func lineNumber(content string, idx int) int {
	if idx <= 0 {
		return 1
	}
	if idx >= len(content) {
		idx = len(content) - 1
	}
	return strings.Count(content[:idx], "\n") + 1
}
