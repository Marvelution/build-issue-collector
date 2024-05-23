package services

import (
	"encoding/json"
	"fmt"
	"github.com/jfrog/jfrog-client-go/auth"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/marvelution/ext-build-info/services/jira"
	"net/http"
)

func NewJiraDetails() auth.ServiceDetails {
	return &jiraDetails{}
}

type jiraDetails struct {
	auth.CommonConfigFields
}

func (js *jiraDetails) GetVersion() (string, error) {
	info := &jira.ServerInfo{}
	if err := js.GetRequest("rest/api/3/serverInfo", &info); err != nil {
		return "", err
	}
	return info.Version, nil
}

func (js *jiraDetails) GetRequest(url string, request any) error {
	clientDetails := js.CreateHttpClientDetails()
	resp, body, _, err := js.GetClient().SendGet(js.GetUrl()+url, false, &clientDetails)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return json.Unmarshal(body, &request)
	} else {
		return errorutils.CheckErrorf(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))
	}
}

func NewBitbucketDetails() auth.ServiceDetails {
	return &bitbucketDetails{}
}

type bitbucketDetails struct {
	auth.CommonConfigFields
}

func (bs *bitbucketDetails) GetVersion() (string, error) {
	return "Cloud", nil
}
