package commands

import (
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services"
	"github.com/marvelution/ext-build-info/services/common"
	"github.com/marvelution/ext-build-info/services/jira"
	"github.com/marvelution/ext-build-info/util"
	"strconv"
	"strings"
	"time"
)

type SendBuildInfoCommand struct {
	buildConfiguration *utils.BuildConfiguration
	jiraConfiguration  *JiraConfiguration
}

func NewSendBuildInfoCommand() *SendBuildInfoCommand {
	return &SendBuildInfoCommand{}
}

func (cmd *SendBuildInfoCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *SendBuildInfoCommand {
	cmd.buildConfiguration = buildConfiguration
	return cmd
}

func (cmd *SendBuildInfoCommand) SetJiraConfiguration(jiraConfiguration *JiraConfiguration) *SendBuildInfoCommand {
	cmd.jiraConfiguration = jiraConfiguration
	return cmd
}

func (cmd *SendBuildInfoCommand) Run() error {
	log.Info("Collecting build-info to send to Jira.")

	buildInfo, err := getBuildInfo(cmd.buildConfiguration, cmd.jiraConfiguration.serverDetails)
	if err != nil {
		return err
	}

	if buildInfo != nil && buildInfo.Issues != nil && len(buildInfo.Issues.AffectedIssues) > 0 {
		// We have issues, lets send the build-info
		client, err := services.NewOAuthJiraService(cmd.jiraConfiguration.jiraUrl, cmd.jiraConfiguration.jiraClientId,
			cmd.jiraConfiguration.jiraSecret, cmd.jiraConfiguration.dryRun)
		if err != nil {
			return err
		}

		buildNumber, err := strconv.ParseInt(buildInfo.Number, 10, 64)
		if err != nil {
			return err
		}

		var issueKeys []string
		for _, issue := range buildInfo.Issues.AffectedIssues {
			if !issue.Aggregated {
				log.Info("Including issue " + issue.Key)
				issueKeys = append(issueKeys, issue.Key)
			} else {
				log.Info("Skipping issue " + issue.Key + " since the issue is aggregated from a previous build")
			}
		}
		var references []jira.Reference
		for _, vcs := range buildInfo.VcsList {
			references = append(references, jira.Reference{
				Commit: &jira.Commit{
					Id:            vcs.Revision,
					RepositoryUri: util.GetHttpsVcsUrl(vcs.Url),
				},
			})
		}

		jiraBuildInfo := jira.BuildInfo{
			SchemaVersion:        "1.0",
			PipelineId:           buildInfo.Name,
			BuildNumber:          buildNumber,
			UpdateSequenceNumber: time.Now().UnixMilli(),
			DisplayName:          buildInfo.Name + " #" + buildInfo.Number,
			Url:                  buildInfo.BuildUrl,
			State:                common.Unknown,
			LastUpdated:          time.Now(),
			IssueKeys:            util.RemoveDuplicate(issueKeys),
			References:           references,
		}

		pipelinesService, err := services.NewPipelinesService(*cmd.jiraConfiguration.serverDetails)
		if err != nil {
			return err
		}
		pipelineReport, err := pipelinesService.GetPipelineReport(buildInfo.Properties["buildInfo.env.run_id"], cmd.jiraConfiguration.includePrePostRunSteps)
		if err != nil {
			return err
		}
		if pipelineReport != nil {
			jiraBuildInfo.State = pipelineReport.State
			jiraBuildInfo.TestInfo = &jira.TestInfo{
				TotalNumber:   pipelineReport.TestReport.TotalTests,
				NumberPassed:  pipelineReport.TestReport.TotalPassing,
				NumberFailed:  pipelineReport.TestReport.TotalFailures + pipelineReport.TestReport.TotalErrors,
				NumberSkipped: pipelineReport.TestReport.TotalSkipped,
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
		if len(response.RejectedBuilds) > 0 && cmd.jiraConfiguration.failOnReject {
			return errorutils.CheckErrorf("There are " + strconv.Itoa(len(response.RejectedBuilds)) + " rejected builds")
		}
	}

	return nil
}
