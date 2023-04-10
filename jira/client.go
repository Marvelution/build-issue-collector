package jira

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
)

type Client struct {
	client *jfroghttpclient.JfrogHttpClient
	config auth.ServiceDetails
}

func NewClient(Url, Username, Token string) (*Client, error) {
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
	if details.GetClient() == nil {
		details.SetClient(client)
	}
	return &Client{client: client, config: details}, nil
}

func (c *Client) GetIssues(foundIssueKeys []string) ([]buildinfo.AffectedIssue, error) {
	if len(foundIssueKeys) == 0 {
		return []buildinfo.AffectedIssue{}, nil
	}
	clientDetails := c.config.CreateHttpClientDetails()
	utils.SetContentType("application/json", &clientDetails.Headers)

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

	resp, body, err := c.client.SendPost(c.config.GetUrl()+"rest/api/3/search", content, &clientDetails)
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
				Url:        c.config.GetUrl() + "browse/" + issue.Key,
				Aggregated: false,
			})
		}

		return foundIssues, nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))
	}
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
