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
	"github.com/marvelution/ext-build-info/services/jira"
	"net/http"
	"strings"
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

	if !dryRun {
		request := &jira.AccessTokenRequest{
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
			response := &jira.AccessTokenResponse{}
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, err
			}
			details.SetAccessToken(response.AccessToken)
		} else {
			return nil, errorutils.CheckErrorf(fmt.Sprintf("Failed getting an access token: %s.\n%s\n", resp.Status, body))
		}
	}
	return &JiraService{client: client, ServiceDetails: details, dryRun: dryRun}, nil
}

func (js *JiraService) GetVersion() (string, error) {
	info := &jira.ServerInfo{}
	if err := js.GetRequest("rest/api/3/serverInfo", &info); err != nil {
		return "", err
	}
	return info.Version, nil
}

func (js *JiraService) GetCloudId() (string, error) {
	if js.cloudId != "" {
		return js.cloudId, nil
	} else {
		info := &jira.CloudIdResponse{}
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

	request := &jira.SearchRequest{
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
		searchResult := &jira.SearchResult{}
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

func (js *JiraService) SendBuildInfo(buildInfo jira.BuildInfo) (*jira.BuildInfoResponse, error) {
	request := jira.BuildInfoRequest{
		Properties: map[string]string{},
		Builds: []jira.BuildInfo{
			buildInfo,
		},
		ProviderMetadata: jira.ProviderMetadata{
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
		return &jira.BuildInfoResponse{
			AcceptedBuilds: []jira.BuildKey{{
				PipelineId:  buildInfo.PipelineId,
				BuildNumber: buildInfo.BuildNumber,
			}},
			RejectedBuilds:   nil,
			UnknownIssueKeys: nil,
		}, nil
	} else {
		log.Debug("Sending build-info to Jira using request ("+url+"):", string(content))
		resp, body, err := js.client.SendPost(url, content, &clientDetails)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusAccepted {
			response := &jira.BuildInfoResponse{}
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

func (js *JiraService) SendDeploymentInfo(deploymentInfo jira.DeploymentInfo) (*jira.DeploymentInfoResponse, error) {
	request := jira.DeploymentInfoRequest{
		Properties: map[string]string{},
		Deployments: []jira.DeploymentInfo{
			deploymentInfo,
		},
		ProviderMetadata: jira.ProviderMetadata{
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

	url := "https://api.atlassian.com/jira/deployments/0.1/cloud/" + cloudId + "/bulk"
	if js.dryRun {
		log.Info("Dry-running request to Jira ("+url+"):", string(content))
		return &jira.DeploymentInfoResponse{
			AcceptedDeployments: []jira.DeploymentKey{{
				PipelineId:               deploymentInfo.Pipeline.Id,
				EnvironmentId:            deploymentInfo.Environment.Id,
				DeploymentSequenceNumber: deploymentInfo.DeploymentSequenceNumber,
			}},
			RejectedDeployments: nil,
			UnknownIssueKeys:    nil,
			UnknownAssociations: nil,
		}, nil
	} else {
		log.Debug("Sending deployment-info to Jira using request ("+url+"):", string(content))
		resp, body, err := js.client.SendPost(url, content, &clientDetails)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusAccepted {
			response := &jira.DeploymentInfoResponse{}
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
