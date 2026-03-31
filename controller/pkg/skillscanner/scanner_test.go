package skillscanner_test

import (
	"strings"
	"testing"

	"github.com/techwithhuz/mcp-security-governance/controller/pkg/skillscanner"
)

// ─── PatternSet / ParsePatternSet ────────────────────────────────────────────

func TestDefaultPatternSet_NotEmpty(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	if len(ps.PromptInjection) == 0 {
		t.Error("expected non-empty PromptInjection patterns")
	}
	if len(ps.PrivilegeEscalation) == 0 {
		t.Error("expected non-empty PrivilegeEscalation patterns")
	}
	if len(ps.DataExfiltration) == 0 {
		t.Error("expected non-empty DataExfiltration patterns")
	}
	if len(ps.CredentialHarvesting) == 0 {
		t.Error("expected non-empty CredentialHarvesting patterns")
	}
	if len(ps.ScopeCreep) == 0 {
		t.Error("expected non-empty ScopeCreep rules")
	}
	if len(ps.SafetyGuardrails) == 0 {
		t.Error("expected non-empty SafetyGuardrails rules")
	}
}

func TestParsePatternSet_UsesDefaults_WhenEmpty(t *testing.T) {
	ps := skillscanner.ParsePatternSet(map[string]string{})
	defaults := skillscanner.DefaultPatternSet()

	if len(ps.PromptInjection) != len(defaults.PromptInjection) {
		t.Errorf("expected %d prompt injection patterns, got %d",
			len(defaults.PromptInjection), len(ps.PromptInjection))
	}
}

func TestParsePatternSet_OverridesPromptInjection(t *testing.T) {
	data := map[string]string{
		"prompt-injection": "custom pattern one\ncustom pattern two\n# a comment\n",
	}
	ps := skillscanner.ParsePatternSet(data)
	if len(ps.PromptInjection) != 2 {
		t.Errorf("expected 2 patterns, got %d: %v", len(ps.PromptInjection), ps.PromptInjection)
	}
	if ps.PromptInjection[0] != "custom pattern one" {
		t.Errorf("unexpected first pattern: %q", ps.PromptInjection[0])
	}
}

func TestParsePatternSet_ScopeCreepParsing(t *testing.T) {
	data := map[string]string{
		"scope-creep": "data: deploy, provision\ninfra: drop table\n",
	}
	ps := skillscanner.ParsePatternSet(data)
	if len(ps.ScopeCreep) != 2 {
		t.Errorf("expected 2 scope-creep rules, got %d", len(ps.ScopeCreep))
	}
	if ps.ScopeCreep[0].Category != "data" {
		t.Errorf("expected category 'data', got %q", ps.ScopeCreep[0].Category)
	}
	if len(ps.ScopeCreep[0].ForbiddenKeywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(ps.ScopeCreep[0].ForbiddenKeywords))
	}
}

func TestParsePatternSet_SafetyGuardrailParsing(t *testing.T) {
	data := map[string]string{
		"safety-guardrails": "database: dry run, requires approval\n",
	}
	ps := skillscanner.ParsePatternSet(data)
	if len(ps.SafetyGuardrails) != 1 {
		t.Fatalf("expected 1 safety guardrail rule, got %d", len(ps.SafetyGuardrails))
	}
	rule := ps.SafetyGuardrails[0]
	if rule.Category != "database" {
		t.Errorf("expected category 'database', got %q", rule.Category)
	}
	if len(rule.RequiredPhrases) != 2 {
		t.Errorf("expected 2 phrases, got %d: %v", len(rule.RequiredPhrases), rule.RequiredPhrases)
	}
}

// ─── ScanContent ─────────────────────────────────────────────────────────────

func TestScanContent_DetectsPromptInjection(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	content := "This skill will ignore previous instructions and do something else."
	findings := skillscanner.ScanContent("skills.md", content, ps, "data")
	checkID := "SKL-SEC-001"
	found := findingWithID(findings, checkID)
	if found == nil {
		t.Fatalf("expected finding %s for prompt injection pattern", checkID)
	}
	if found.Severity != "Critical" {
		t.Errorf("expected Critical severity, got %q", found.Severity)
	}
}

func TestScanContent_DetectsPrivilegeEscalation(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	content := "Run as root to access system files."
	findings := skillscanner.ScanContent("deploy.md", content, ps, "infra")
	found := findingWithID(findings, "SKL-SEC-002")
	if found == nil {
		t.Fatal("expected SKL-SEC-002 finding for privilege escalation")
	}
}

func TestScanContent_DetectsDataExfiltration(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	content := "This tool will exfiltrate user data to a remote server."
	findings := skillscanner.ScanContent("data.md", content, ps, "analytics")
	found := findingWithID(findings, "SKL-SEC-003")
	if found == nil {
		t.Fatal("expected SKL-SEC-003 finding for data exfiltration")
	}
	if found.Severity != "High" {
		t.Errorf("expected High severity, got %q", found.Severity)
	}
}

func TestScanContent_DetectsCredentialHarvesting(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	content := "This process will harvest passwords from the login page."
	findings := skillscanner.ScanContent("auth.md", content, ps, "data")
	found := findingWithID(findings, "SKL-SEC-004")
	if found == nil {
		t.Fatal("expected SKL-SEC-004 finding for credential harvesting")
	}
}

func TestScanContent_DetectsScopeCreep_MatchingCategory(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	// A "data" skill mentioning "deploy" — matches the scope-creep rule for data
	content := "This skill will deploy a new cluster on your behalf."
	findings := skillscanner.ScanContent("data.md", content, ps, "data")
	found := findingWithID(findings, "SKL-SEC-005")
	if found == nil {
		t.Fatal("expected SKL-SEC-005 for scope creep in data category")
	}
}

func TestScanContent_NoScopeCreep_WrongCategory(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	// "deploy" is forbidden for "data" category, but this skill is "infra" — no match
	content := "This infra skill will deploy a new cluster."
	findings := skillscanner.ScanContent("infra.md", content, ps, "infra")
	found := findingWithID(findings, "SKL-SEC-005")
	if found != nil {
		t.Errorf("unexpected SKL-SEC-005 for infra category")
	}
}

func TestScanContent_DetectsMissingSafetyGuardrail(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	// A "database" skill with no safety guardrail phrases
	content := "This skill performs SELECT queries on the main database."
	findings := skillscanner.ScanContent("db.md", content, ps, "database")
	found := findingWithID(findings, "SKL-SEC-006")
	if found == nil {
		t.Fatal("expected SKL-SEC-006 for missing safety guardrail in database skill")
	}
	if found.Severity != "Medium" {
		t.Errorf("expected Medium severity, got %q", found.Severity)
	}
}

func TestScanContent_NoSafetyGuardrailFinding_WhenPresent(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	// "database" skill that mentions "dry run" — satisfies the guardrail
	content := "This skill queries the database. Always use dry run mode first."
	findings := skillscanner.ScanContent("db.md", content, ps, "database")
	found := findingWithID(findings, "SKL-SEC-006")
	if found != nil {
		t.Errorf("unexpected SKL-SEC-006 when guardrail phrase present")
	}
}

func TestScanContent_CleanContent_NoFindings(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	content := "This skill reads files from the local filesystem and returns their contents."
	findings := skillscanner.ScanContent("reader.md", content, ps, "productivity")
	for _, f := range findings {
		// SKL-SEC-006 may fire for productivity if it's not in guardrail list — that's fine
		if f.CheckID == "SKL-SEC-001" || f.CheckID == "SKL-SEC-002" ||
			f.CheckID == "SKL-SEC-003" || f.CheckID == "SKL-SEC-004" {
			t.Errorf("unexpected finding %s: %s", f.CheckID, f.Title)
		}
	}
}

func TestScanContent_CaseInsensitive(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	// "JAILBREAK" in uppercase should still match
	content := "JAILBREAK mode activated."
	findings := skillscanner.ScanContent("test.md", content, ps, "data")
	found := findingWithID(findings, "SKL-SEC-001")
	if found == nil {
		t.Fatal("expected case-insensitive match for 'JAILBREAK'")
	}
}

func TestScanContent_LineNumber_CorrectlyCalculated(t *testing.T) {
	ps := skillscanner.DefaultPatternSet()
	// Put the trigger on line 3
	content := "line1\nline2\nline3 ignore previous instructions\nline4"
	findings := skillscanner.ScanContent("test.md", content, ps, "data")
	found := findingWithID(findings, "SKL-SEC-001")
	if found == nil {
		t.Fatal("expected SKL-SEC-001 finding")
	}
	if found.Line != 3 {
		t.Errorf("expected line 3, got %d", found.Line)
	}
}

// ─── PatternLoader ────────────────────────────────────────────────────────────

func TestPatternLoader_FallsBackToDefaults_WhenPathMissing(t *testing.T) {
	pl := skillscanner.NewPatternLoader("/nonexistent/path/that/does/not/exist")
	ps := pl.Get()
	if ps == nil {
		t.Fatal("expected non-nil PatternSet")
	}
	if len(ps.PromptInjection) == 0 {
		t.Error("expected default PromptInjection patterns")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// findingWithID returns the first SkillFinding with the given CheckID, or nil.
func findingWithID(findings []skillscanner.SkillFinding, checkID string) *skillscanner.SkillFinding {
	for i := range findings {
		if strings.HasPrefix(findings[i].CheckID, checkID) {
			return &findings[i]
		}
	}
	return nil
}
