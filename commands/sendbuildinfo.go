package commands

import (
	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	artservices "github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services"
	"github.com/marvelution/ext-build-info/util"
	"os"
	"strconv"
	"strings"
	"time"
)

type SendBuildInfoCommand struct {
	buildConfiguration *utils.BuildConfiguration
	JiraConfiguration  *JiraConfiguration
}

func NewSendBuildInfoCommand() *SendBuildInfoCommand {
	return &SendBuildInfoCommand{}
}

func (config *SendBuildInfoCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *SendBuildInfoCommand {
	config.buildConfiguration = buildConfiguration
	return config
}

func (config *SendBuildInfoCommand) SetJiraConfiguration(jiraConfiguration *JiraConfiguration) *SendBuildInfoCommand {
	config.JiraConfiguration = jiraConfiguration
	return config
}

func (config *SendBuildInfoCommand) Run() error {
	log.Info("Collecting build-info to send to Jira.")

	buildInfo, err := config.getBuildInfo(config.JiraConfiguration)
	if err != nil {
		return err
	}

	if buildInfo.Issues != nil && len(buildInfo.Issues.AffectedIssues) > 0 {
		// We have issues, lets send the build-info
		client, err := services.NewOAuthJiraService(config.JiraConfiguration.jiraUrl, config.JiraConfiguration.jiraClientId,
			config.JiraConfiguration.jiraSecret, config.JiraConfiguration.dryRun)
		if err != nil {
			return err
		}

		buildNumber, err := strconv.ParseInt(buildInfo.Number, 10, 64)
		if err != nil {
			return err
		}

		var issueKeys []string
		for _, issue := range buildInfo.Issues.AffectedIssues {
			issueKeys = append(issueKeys, issue.Key)
		}
		var references []services.Reference
		for _, vcs := range buildInfo.VcsList {
			references = append(references, services.Reference{
				Commit: &services.Commit{
					Id:            vcs.Revision,
					RepositoryUri: util.GetHttpsVcsUrl(vcs.Url),
				},
			})
		}

		jiraBuildInfo := services.BuildInfo{
			SchemaVersion:        "1.0",
			PipelineId:           buildInfo.Name,
			BuildNumber:          buildNumber,
			UpdateSequenceNumber: time.Now().UnixMilli(),
			DisplayName:          buildInfo.Name + " #" + buildInfo.Number,
			Url:                  buildInfo.BuildUrl,
			State:                services.Unknown,
			LastUpdated:          time.Now(),
			IssueKeys:            issueKeys,
			References:           references,
		}

		pipelinesService, err := services.NewPipelinesService(*config.JiraConfiguration.serverDetails)
		if err != nil {
			return err
		}
		pipelineReport, err := pipelinesService.GetPipelineReport(buildInfo.Properties["buildInfo.env.run_id"], config.JiraConfiguration.includePrePostRunSteps)
		if err != nil {
			return err
		}
		if pipelineReport != nil {
			jiraBuildInfo.State = pipelineReport.State
			jiraBuildInfo.TestInfo = &services.TestInfo{
				TotalNumber:   pipelineReport.TotalTests,
				NumberPassed:  pipelineReport.TotalPassing,
				NumberFailed:  pipelineReport.TotalFailures + pipelineReport.TotalErrors,
				NumberSkipped: pipelineReport.TotalSkipped,
			}
		}

		response, err := client.SendBuildInfo(jiraBuildInfo)
		if err != nil {
			return err
		}
		if len(response.AcceptedBuilds) > 0 {
			for _, build := range response.AcceptedBuilds {
				log.Info("Build " + build.PipelineId + " #" + strconv.FormatInt(build.BuildNumber, 10) + " was accepted by Jira")
			}
		}
		if len(response.RejectedBuilds) > 0 {
			for _, build := range response.RejectedBuilds {
				log.Warn("Build " + build.Key.PipelineId + " #" + strconv.FormatInt(build.Key.BuildNumber, 10) + " was rejected by Jira")
				for _, buildError := range build.Errors {
					log.Warn(" - " + buildError.Message + " (" + buildError.ErrorTraceId + ")")
				}
			}
		}
		if len(response.UnknownIssueKeys) > 0 {
			log.Warn("The following issues are unknown by Jira: " + strings.Join(response.UnknownIssueKeys, ","))
		}
		if len(response.RejectedBuilds) > 0 && config.JiraConfiguration.failOnReject {
			return errorutils.CheckErrorf("There are " + strconv.Itoa(len(response.RejectedBuilds)) + " rejected builds")
		}
	}

	return nil
}

// Returns build info, or empty build info struct if not found.
func (config *SendBuildInfoCommand) getBuildInfo(jiraConfig *JiraConfiguration) (*buildinfo.BuildInfo, error) {
	// Create services manager to get build-info from Artifactory.
	sm, err := utils.CreateServiceManager(jiraConfig.serverDetails, -1, 0, false)
	if err != nil {
		return nil, err
	}

	buildName, err := config.buildConfiguration.GetBuildName()
	if err != nil {
		return nil, err
	}
	buildNumber, err := config.buildConfiguration.GetBuildNumber()
	if err != nil {
		return nil, err
	}

	buildInfoParams := artservices.BuildInfoParams{BuildName: buildName, BuildNumber: buildNumber}
	publishedBuildInfo, found, err := sm.GetBuildInfo(buildInfoParams)
	if err != nil {
		return nil, err
	}
	if !found {
		return &buildinfo.BuildInfo{}, nil
	}

	return &publishedBuildInfo.BuildInfo, nil
}

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
		jc.jiraID = os.Getenv("int_" + jc.jiraID + "_url")
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
