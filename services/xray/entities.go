package xray

import "time"

import "github.com/jfrog/jfrog-client-go/xray/services"

type BuildScanResult struct {
	BuildName       string                   `json:"build_name"`
	BuildNumber     string                   `json:"build_number"`
	Status          string                   `json:"status"`
	MoreDetailsUrl  string                   `json:"more_details_url"`
	FailBuild       bool                     `json:"fail_build"`
	Violations      []services.Violation     `json:"violations,omitempty"`
	Vulnerabilities []services.Vulnerability `json:"vulnerabilities,omitempty"`
}

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
