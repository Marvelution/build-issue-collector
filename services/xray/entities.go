package xray

import "time"

type BuildSummary struct {
	Build            Build             `json:"build"`
	Issues           []Issue           `json:"issues"`
	Licenses         []License         `json:"licenses"`
	OperationalRisks []OperationalRisk `json:"operational_risks"`
	Errors           []interface{}     `json:"errors"`
}

type Build struct {
	Name        string `json:"name"`
	Number      string `json:"number,omitempty"`
	ComponentId string `json:"component_id"`
	PkgType     string `json:"pkg_type"`
	Path        string `json:"path"`
	Sha256      string `json:"sha256"`
}

type Issue struct {
	IssueId                string      `json:"issue_id"`
	Summary                string      `json:"summary"`
	Description            string      `json:"description"`
	IssueType              string      `json:"issue_type"`
	Severity               string      `json:"severity"`
	Provider               string      `json:"provider"`
	Cves                   []Cves      `json:"cves"`
	Created                time.Time   `json:"created"`
	ImpactPath             []string    `json:"impact_path"`
	Applicability          interface{} `json:"applicability"`
	ComponentPhysicalPaths []string    `json:"component_physical_paths"`
}

type Cves struct {
	Cve    string   `json:"cve"`
	Cwe    []string `json:"cwe"`
	CvssV3 string   `json:"cvss_v3"`
}

type License struct {
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	MoreInfoUrl []string `json:"more_info_url"`
	Components  []string `json:"components"`
}

type OperationalRisk struct {
	ComponentId   string      `json:"component_id"`
	Risk          string      `json:"risk"`
	RiskReason    string      `json:"risk_reason"`
	IsEol         bool        `json:"is_eol"`
	EolMessage    string      `json:"eol_message"`
	LatestVersion string      `json:"latest_version"`
	NewerVersions int64       `json:"newer_versions"`
	Cadence       int64       `json:"cadence"`
	Commits       interface{} `json:"commits"`
	Committers    interface{} `json:"committers"`
	Released      time.Time   `json:"released"`
}

type IgnoredViolationsRequest struct {
	Artifacts []Scope `json:"artifacts"`
}

type Scope struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type IgnoredViolationsResponse struct {
	Data       []IgnoredViolation `json:"data"`
	TotalCount int64              `json:"total_count"`
}

type IgnoredViolation struct {
	ViolationId       string            `json:"violation_id"`
	IssueId           string            `json:"issue_id"`
	Type              string            `json:"type"`
	Created           time.Time         `json:"created"`
	WatchName         string            `json:"watch_name"`
	Provider          string            `json:"provider"`
	Description       string            `json:"description"`
	Severity          string            `json:"severity"`
	Properties        []Properties      `json:"properties"`
	ImpactedArtifact  ImpactedArtifact  `json:"impacted_artifact"`
	MatchedPolicies   []MatchedPolicies `json:"matched_policies"`
	IgnoreRuleDetails IgnoreRuleDetails `json:"ignore_rule_details"`
	Applicability     interface{}       `json:"applicability"`
}

type Properties struct {
	Cve    string   `json:"Cve"`
	Cwe    []string `json:"Cwe"`
	CvssV2 string   `json:"CvssV2"`
	CvssV3 string   `json:"CvssV3"`
}

type ImpactedArtifact struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type MatchedPolicies struct {
	Policy     string `json:"policy"`
	Rule       string `json:"rule"`
	IsBlocking bool   `json:"is_blocking"`
}
type IgnoreRuleDetails struct {
	Id        string    `json:"id"`
	Author    string    `json:"author"`
	Created   time.Time `json:"created"`
	Notes     string    `json:"notes"`
	IsExpired bool      `json:"is_expired"`
}
