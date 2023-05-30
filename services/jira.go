package services

import (
	"encoding/json"
	"fmt"
	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/auth"
	clientConfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"net/http"
	"strings"
	"time"
)

type JiraService struct {
	client  *jfroghttpclient.JfrogHttpClient
	cloudId string
	dryRun  bool
	auth.ServiceDetails
}

func NewJiraService(Url, Username, Token string) (*JiraService, error) {
	details := NewJiraDetails()
	details.SetUrl(clientutils.AddTrailingSlashIfNeeded(Url))
	details.SetUser(Username)
	details.SetPassword(Token)
	configBuilder := clientConfig.NewConfigBuilder().SetServiceDetails(details)

	config, err := configBuilder.Build()
	if err != nil {
		return nil, err
	}

	client, err := jfroghttpclient.JfrogClientBuilder().
		SetTimeout(config.GetHttpTimeout()).
		SetRetries(config.GetHttpRetries()).
		SetRetryWaitMilliSecs(config.GetHttpRetryWaitMilliSecs()).
		SetHttpClient(config.GetHttpClient()).
		Build()
	if err != nil {
		return nil, err
	}
	return &JiraService{client: client, ServiceDetails: details}, nil
}

func NewOAuthJiraService(Url, ClientId, Secret string, dryRun bool) (*JiraService, error) {
	details := NewJiraDetails()
	details.SetUrl(clientutils.AddTrailingSlashIfNeeded(Url))
	configBuilder := clientConfig.NewConfigBuilder().SetServiceDetails(details)

	config, err := configBuilder.Build()
	if err != nil {
		return nil, err
	}

	client, err := jfroghttpclient.JfrogClientBuilder().
		SetTimeout(config.GetHttpTimeout()).
		SetRetries(config.GetHttpRetries()).
		SetRetryWaitMilliSecs(config.GetHttpRetryWaitMilliSecs()).
		SetHttpClient(config.GetHttpClient()).
		Build()
	if err != nil {
		return nil, err
	}
	if details.GetClient() == nil {
		details.SetClient(client)
	}

	request := &AccessTokenRequest{
		Audience:     "api.atlassian.com",
		GrantType:    "client_credentials",
		ClientId:     ClientId,
		ClientSecret: Secret,
	}

	content, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	clientDetails := details.CreateHttpClientDetails()
	utils.SetContentType("application/json", &clientDetails.Headers)
	resp, body, err := client.SendPost("https://api.atlassian.com/oauth/token", content, &clientDetails)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		response := &AccessTokenResponse{}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, err
		}
		details.SetAccessToken(response.AccessToken)

		return &JiraService{client: client, ServiceDetails: details, dryRun: dryRun}, nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("Failed getting an access token: %s.\n%s\n", resp.Status, body))
	}
}

func (js *JiraService) GetVersion() (string, error) {
	info := &ServerInfo{}
	if err := js.GetRequest("rest/api/3/serverInfo", &info); err != nil {
		return "", err
	}
	return info.Version, nil
}

func (js *JiraService) GetCloudId() (string, error) {
	if js.cloudId != "" {
		return js.cloudId, nil
	} else {
		info := &CloudIdResponse{}
		if err := js.GetRequest("_edge/tenant_info", &info); err != nil {
			return "", err
		}
		js.cloudId = info.CloudId
		return js.cloudId, nil
	}
}

func (js *JiraService) GetRequest(url string, request any) error {
	clientDetails := js.CreateHttpClientDetails()
	resp, body, _, err := js.client.SendGet(js.GetUrl()+url, false, &clientDetails)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return json.Unmarshal(body, &request)
	} else {
		return errorutils.CheckErrorf(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))
	}
}

func (js *JiraService) GetIssues(foundIssueKeys []string) ([]buildinfo.AffectedIssue, error) {
	if len(foundIssueKeys) == 0 {
		return []buildinfo.AffectedIssue{}, nil
	}

	request := &SearchRequest{
		Jql:           "issue IN (" + strings.Join(foundIssueKeys[:], ",") + ")",
		Fields:        []string{"key", "summary"},
		StartAt:       0,
		MaxResults:    100,
		ValidateQuery: "warn",
	}

	content, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	log.Info("Searching Jira using request:", string(content))

	clientDetails := js.CreateHttpClientDetails()
	utils.SetContentType("application/json", &clientDetails.Headers)
	resp, body, err := js.client.SendPost(js.GetUrl()+"rest/api/3/search", content, &clientDetails)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		searchResult := &SearchResult{}
		if err := json.Unmarshal(body, &searchResult); err != nil {
			return nil, err
		}

		var foundIssues []buildinfo.AffectedIssue
		for _, issue := range searchResult.Issues {
			log.Info("Found Jira issue: ", issue)
			foundIssues = append(foundIssues, buildinfo.AffectedIssue{
				Key:        issue.Key,
				Summary:    issue.Fields.Summary,
				Url:        js.GetUrl() + "browse/" + issue.Key,
				Aggregated: false,
			})
		}

		return foundIssues, nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))
	}
}

func (js *JiraService) SendBuildInfo(buildInfo BuildInfo) (*BuildInfoResponse, error) {
	request := BuildInfoRequest{
		Properties: map[string]string{},
		Builds: []BuildInfo{
			buildInfo,
		},
		ProviderMetadata: ProviderMetadata{
			Product: "Jfrog Pipelines",
		},
	}
	content, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	clientDetails := js.CreateHttpClientDetails()
	utils.SetContentType("application/json", &clientDetails.Headers)

	cloudId, err := js.GetCloudId()
	if err != nil {
		return nil, err
	}

	url := "https://api.atlassian.com/jira/builds/0.1/cloud/" + cloudId + "/bulk"
	if js.dryRun {
		log.Info("Dry-running request to Jira ("+url+"):", string(content))
		return nil, nil
	} else {
		log.Debug("Sending build-info to Jira using request ("+url+"):", string(content))
		resp, body, err := js.client.SendPost(url, content, &clientDetails)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusAccepted {
			response := &BuildInfoResponse{}
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, err
			}

			log.Debug(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))

			return response, nil
		} else {
			return nil, errorutils.CheckErrorf(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))
		}
	}
}

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
	SchemaVersion        string      `json:"schemaVersion,omitempty"`
	PipelineId           string      `json:"pipelineId"`
	BuildNumber          int64       `json:"buildNumber"`
	UpdateSequenceNumber int64       `json:"updateSequenceNumber"`
	DisplayName          string      `json:"displayName"`
	Description          string      `json:"description,omitempty"`
	Label                string      `json:"label,omitempty"`
	Url                  string      `json:"url"`
	State                State       `json:"state"`
	LastUpdated          time.Time   `json:"lastUpdated"`
	IssueKeys            []string    `json:"issueKeys"`
	TestInfo             *TestInfo   `json:"testInfo,omitempty"`
	References           []Reference `json:"references,omitempty"`
}

type State string

const (
	Successful State = "successful"
	Failed     State = "failed"
	Cancelled  State = "cancelled"
	InProgress State = "in_progress"
	Pending    State = "pending"
	Unknown    State = "unknown"
)

var BestToWorst = []State{Successful, Failed, Cancelled, InProgress, Pending, Unknown}

func GetState(code int64) State {
	if code == 4000 || code == 4005 {
		return Pending
	} else if code == 4001 {
		return InProgress
	} else if code == 4002 {
		return Successful
	} else if code == 4003 || code == 4004 || code == 4007 {
		return Failed
	} else if code == 4006 || code == 4008 {
		return Cancelled
	} else {
		return Unknown
	}
}

func (s *State) Index() int {
	for index, state := range BestToWorst {
		if s == &state {
			return index
		}
	}
	return -1
}

func (s *State) IsWorstThan(state State) bool {
	return s.Index() > state.Index()
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
