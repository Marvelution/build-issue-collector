package services

import (
	"encoding/json"
	"fmt"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/auth"
	clientConfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services/bitbucket"
	"net/http"
)

type BitbucketService struct {
	client *jfroghttpclient.JfrogHttpClient
	dryRun bool
	auth.ServiceDetails
}

func NewBitbucketService(Url, Username, Token string, dryRun bool) (*BitbucketService, error) {
	details := NewBitbucketDetails()
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
	return &BitbucketService{client: client, ServiceDetails: details, dryRun: dryRun}, nil
}

func (bs *BitbucketService) SendCommitStatus(repoSlug, commitSha string, message bitbucket.CreateCommitStatus) error {
	content, err := json.Marshal(message)
	if err != nil {
		return err
	}

	clientDetails := bs.CreateHttpClientDetails()
	utils.SetContentType("application/json", &clientDetails.Headers)

	url := bs.ServiceDetails.GetUrl() + "2.0/repositories/" + repoSlug + "/commit/" + commitSha + "/statuses/build"
	if bs.dryRun {
		log.Info("Dry-running request to Bitbucket ("+url+"):", string(content))
		return nil
	} else {
		log.Debug("Sending build-info to Jira using request ("+url+"):", string(content))
		resp, body, err := bs.client.SendPost(url, content, &clientDetails)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusCreated {
			log.Debug(fmt.Sprintf("Response from Bitbucket: %s.\n%s\n", resp.Status, body))
			return nil
		} else {
			return errorutils.CheckErrorf(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))
		}
	}
}
