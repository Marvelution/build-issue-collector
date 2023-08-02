package jira

import (
	"github.com/marvelution/ext-build-info/services/common"
	"time"
)

type ServerInfo struct {
	Version        string `json:"version"`
	VersionNumbers []int  `json:"versionNumbers"`
}

type CloudIdResponse struct {
	CloudId string `json:"cloudId"`
}

type SearchRequest struct {
	Jql           string   `json:"jql,omitempty"`
	StartAt       int      `json:"startAt,omitempty"`
	MaxResults    int      `json:"maxResults,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	ValidateQuery string   `json:"validateQuery,omitempty"`
}

type SearchResult struct {
	Expand          string   `json:"expand,omitempty"`
	StartAt         int      `json:"startAt,omitempty"`
	MaxResults      int      `json:"maxResults,omitempty"`
	Total           int      `json:"total,omitempty"`
	Issues          []Issue  `json:"issues,omitempty"`
	WarningMessages []string `json:"warningMessages,omitempty"`
}

type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary string `json:"summary"`
}

type AccessTokenRequest struct {
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type BuildInfoRequest struct {
	Properties       map[string]string `json:"properties,omitempty"`
	Builds           []BuildInfo       `json:"builds"`
	ProviderMetadata ProviderMetadata  `json:"providerMetadata"`
}

type BuildInfo struct {
	SchemaVersion        string       `json:"schemaVersion,omitempty"`
	PipelineId           string       `json:"pipelineId"`
	BuildNumber          int64        `json:"buildNumber"`
	UpdateSequenceNumber int64        `json:"updateSequenceNumber"`
	DisplayName          string       `json:"displayName"`
	Description          string       `json:"description,omitempty"`
	Label                string       `json:"label,omitempty"`
	Url                  string       `json:"url"`
	State                common.State `json:"state"`
	LastUpdated          time.Time    `json:"lastUpdated"`
	IssueKeys            []string     `json:"issueKeys"`
	TestInfo             *TestInfo    `json:"testInfo,omitempty"`
	References           []Reference  `json:"references,omitempty"`
}

type TestInfo struct {
	TotalNumber   int64 `json:"totalNumber"`
	NumberPassed  int64 `json:"numberPassed"`
	NumberFailed  int64 `json:"numberFailed"`
	NumberSkipped int64 `json:"numberSkipped,omitempty"`
}

type Reference struct {
	Commit *Commit `json:"commit,omitempty"`
	Ref    *Ref    `json:"ref,omitempty"`
}

type Commit struct {
	Id            string `json:"id"`
	RepositoryUri string `json:"repositoryUri"`
}

type Ref struct {
	Name string `json:"name"`
	Uri  string `json:"uri"`
}

type ProviderMetadata struct {
	Product string `json:"product"`
}

type BuildInfoResponse struct {
	AcceptedBuilds   []BuildKey      `json:"acceptedBuilds"`
	RejectedBuilds   []RejectedBuild `json:"rejectedBuilds"`
	UnknownIssueKeys []string        `json:"unknownIssueKeys"`
}

type BuildKey struct {
	PipelineId  string `json:"pipelineId"`
	BuildNumber int64  `json:"buildNumber"`
}

type RejectedBuild struct {
	Key    BuildKey `json:"key"`
	Errors []Error  `json:"errors"`
}

type Error struct {
	Message      string `json:"message"`
	ErrorTraceId string `json:"errorTraceId"`
}

type DeploymentInfoRequest struct {
	Properties       map[string]string `json:"properties,omitempty"`
	Deployments      []DeploymentInfo  `json:"deployments"`
	ProviderMetadata ProviderMetadata  `json:"providerMetadata"`
}

type DeploymentInfo struct {
	SchemaVersion            string        `json:"schemaVersion,omitempty"`
	DeploymentSequenceNumber int64         `json:"deploymentSequenceNumber"`
	UpdateSequenceNumber     int64         `json:"updateSequenceNumber"`
	Associations             []Association `json:"associations"`
	DisplayName              string        `json:"displayName"`
	Url                      string        `json:"url"`
	Description              string        `json:"description"`
	LastUpdated              time.Time     `json:"lastUpdated"`
	Label                    string        `json:"label,omitempty"`
	State                    common.State  `json:"state"`
	Pipeline                 Pipeline      `json:"pipeline"`
	Environment              Environment   `json:"environment"`
	Commands                 []Command     `json:"commands,omitempty"`
}

type AssociationType string

const (
	IssueKeysAssociation       AssociationType = "issueKeys"
	IssueIdOrKeysAssociation   AssociationType = "issueIdOrKeys"
	ServiceIdOrKeysAssociation AssociationType = "serviceIdOrKeys"
)

type Association struct {
	AssociationType AssociationType `json:"associationType"`
	Values          []string        `json:"values"`
}

type Pipeline struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayName"`
	Url         string `json:"url"`
}

type Environment struct {
	Id          string          `json:"id"`
	DisplayName string          `json:"displayName"`
	Type        EnvironmentType `json:"type"`
}

type EnvironmentType string

const (
	Unmapped    EnvironmentType = "unmapped"
	Development EnvironmentType = "development"
	Testing     EnvironmentType = "testing"
	Staging     EnvironmentType = "staging"
	Production  EnvironmentType = "production"
)

type Command struct {
	Command string `json:"command"`
}

type DeploymentInfoResponse struct {
	AcceptedDeployments []DeploymentKey      `json:"acceptedDeployments"`
	RejectedDeployments []RejectedDeployment `json:"rejectedDeployments"`
	UnknownIssueKeys    []string             `json:"unknownIssueKeys"`
	UnknownAssociations []Association        `json:"unknownAssociations"`
}

type DeploymentKey struct {
	PipelineId               string `json:"pipelineId"`
	EnvironmentId            string `json:"environmentId"`
	DeploymentSequenceNumber int64  `json:"deploymentSequenceNumber"`
}

type RejectedDeployment struct {
	Key    DeploymentKey `json:"key"`
	Errors []Error       `json:"errors"`
}
