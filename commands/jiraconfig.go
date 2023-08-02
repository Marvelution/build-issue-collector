package commands

import (
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"os"
)

type JiraConfiguration struct {
	serverID               string
	serverDetails          *utilsconfig.ServerDetails
	jiraID                 string
	jiraUrl                string
	jiraClientId           string
	jiraSecret             string
	dryRun                 bool
	includePrePostRunSteps bool
	failOnReject           bool
}

func (jc *JiraConfiguration) SetServerID(serverID string) *JiraConfiguration {
	jc.serverID = serverID
	return jc
}

func (jc *JiraConfiguration) SetJiraID(jiraID string) *JiraConfiguration {
	jc.jiraID = jiraID
	return jc
}

func (jc *JiraConfiguration) SetJiraDetails(url, clientId, secret string) *JiraConfiguration {
	jc.jiraUrl = url
	jc.jiraClientId = clientId
	jc.jiraSecret = secret
	return jc
}

func (jc *JiraConfiguration) SetDryRun(dryRun bool) *JiraConfiguration {
	jc.dryRun = dryRun
	return jc
}

func (jc *JiraConfiguration) SetIncludePrePostRunSteps(includePrePostRunSteps bool) *JiraConfiguration {
	jc.includePrePostRunSteps = includePrePostRunSteps
	return jc
}

func (jc *JiraConfiguration) SetFailOnReject(failOnReject bool) *JiraConfiguration {
	jc.failOnReject = failOnReject
	return jc
}

func (jc *JiraConfiguration) ValidateJiraConfiguration() (err error) {
	if jc.jiraUrl == "" {
		log.Debug("Loading Jira details from integration ", jc.jiraID)
		jc.jiraUrl = os.Getenv("int_" + jc.jiraID + "_url")
		jc.jiraClientId = os.Getenv("int_" + jc.jiraID + "_username")
		jc.jiraSecret = os.Getenv("int_" + jc.jiraID + "_token")
	}

	if jc.jiraUrl == "" || jc.jiraClientId == "" || jc.jiraSecret == "" {
		return errorutils.CheckErrorf("Missing Jira details")
	}

	// If no server-id provided, use default server.
	serverDetails, err := utilsconfig.GetSpecificConfig(jc.serverID, true, false)
	if err != nil {
		return err
	}
	jc.serverDetails = serverDetails
	return nil
}
