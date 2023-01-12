package jira

import (
	"encoding/json"
	"fmt"
	"github.com/jfrog/jfrog-client-go/auth"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"net/http"
)

func NewJiraDetails() auth.ServiceDetails {
	return &jiraDetails{}
}

type jiraDetails struct {
	auth.CommonConfigFields
}

func (jd *jiraDetails) GetVersion() (string, error) {
	clientDetails := jd.CreateHttpClientDetails()
	resp, body, _, err := jd.GetClient().SendGet(jd.GetUrl()+"rest/api/3/serverInfo", false, &clientDetails)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusOK {
		info := &ServerInfo{}
		if err := json.Unmarshal(body, &info); err != nil {
			return "", err
		}
		return info.Version, nil
	} else {
		return "", errorutils.CheckErrorf(fmt.Sprintf("Response from Jira: %s.\n%s\n", resp.Status, body))
	}
}

type ServerInfo struct {
	Version        string `json:"version"`
	VersionNumbers []int  `json:"versionNumbers"`
}
