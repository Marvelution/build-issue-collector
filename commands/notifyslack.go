package commands

import (
	"encoding/json"
	"fmt"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/http/httpclient"
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/httputils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services"
	"github.com/marvelution/ext-build-info/services/common"
	"net/http"
	"os"
	"strings"
)

type NotifySlackCommand struct {
	buildConfiguration *utils.BuildConfiguration
	slackConfiguration *SlackConfiguration
}

func NewNotifySlackCommand() *NotifySlackCommand {
	return &NotifySlackCommand{}
}

func (cmd *NotifySlackCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *NotifySlackCommand {
	cmd.buildConfiguration = buildConfiguration
	return cmd
}

func (cmd *NotifySlackCommand) SetSlackConfiguration(slackConfiguration *SlackConfiguration) *NotifySlackCommand {
	cmd.slackConfiguration = slackConfiguration
	return cmd
}

func (cmd *NotifySlackCommand) Run() error {
	log.Info("Collecting build-info to send to Slack.")

	pipelinesService, err := services.NewPipelinesService(*cmd.slackConfiguration.serverDetails)
	if err != nil {
		return err
	}
	pipelineReport, err := pipelinesService.GetPipelineReport(os.Getenv("run_id"), cmd.slackConfiguration.includePrePostRunSteps)
	if err != nil {
		return err
	}

	icon := ""
	if pipelineReport.State == common.Failed {
		if pipelineReport.TestReport.TotalFailures > 0 || pipelineReport.TestReport.TotalErrors > 0 {
			icon = ":bangbang:"
		} else {
			icon = ":interrobang:"
		}
	}

	message := SlackMessage{
		Blocks: []SlackBlock{{
			Type: "section",
			Text: SlackText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("%s <%s|%s #%d> *%s*",
					icon, os.Getenv("JFROG_CLI_BUILD_URL"), pipelineReport.Name, pipelineReport.RunNumber, pipelineReport.State),
			},
		}},
		Attachments: []SlackAttachment{},
	}

	var vcsInfo []string
	runResourceVersions := pipelineReport.GetGitRepoRunResourceVersions()
	if len(*runResourceVersions) > 0 {
		for _, runResourceVersion := range *runResourceVersions {
			log.Debug("Collecting vcs information from resource: " + runResourceVersion.ResourceName)
			shaData := runResourceVersion.ResourceVersionContentPropertyBag["shaData"]
			shaDataMap := shaData.(map[string]any)
			repo := runResourceVersion.ResourceVersionContentPropertyBag["path"]
			branch := shaDataMap["branchName"].(string)
			commitUrl := shaDataMap["commitUrl"].(string)
			commitMessage := shaDataMap["commitMessage"].(string)
			commitSha := shaDataMap["commitSha"].(string)[0:8]
			vcsInfo = append(vcsInfo, fmt.Sprintf("`<%s|%s>` %s%s @ %s", commitUrl, commitSha, commitMessage, repo, branch))
		}
	} else {
		buildInfo, err := getBuildInfo(cmd.buildConfiguration, cmd.slackConfiguration.serverDetails)
		if err == nil {
			log.Debug(fmt.Sprintf("Collecting vcs information from buildInfo: %s #%s", buildInfo.Name, buildInfo.Number))
			revisions := map[string]struct{}{}
			for _, vcs := range buildInfo.VcsList {
				_, processed := revisions[vcs.Revision]
				if vcs.Revision != "" && vcs.Branch != "" && !processed {
					revisions[vcs.Revision] = struct{}{}
					vcsInfo = append(vcsInfo, fmt.Sprintf("`%s` @ `%s`", vcs.Branch, vcs.Revision[0:8]))
				}
			}
			// Look again to add any revisions without a branch name
			for _, vcs := range buildInfo.VcsList {
				_, processed := revisions[vcs.Revision]
				if !processed {
					revisions[vcs.Revision] = struct{}{}
					vcsInfo = append(vcsInfo, fmt.Sprintf("`%s`", vcs.Revision[0:8]))
				}
			}
		}
	}
	if len(vcsInfo) > 0 {
		message.Blocks = append(message.Blocks, SlackBlock{
			Type: "section",
			Text: SlackText{
				Type: "mrkdwn",
				Text: strings.Join(vcsInfo, "\n"),
			},
		})
	}

	testReport := pipelineReport.TestReport
	if testReport.TotalTests > 0 {
		icon := ""
		if pipelineReport.TestReport.TotalFailures > 0 || pipelineReport.TestReport.TotalErrors > 0 {
			icon = ":exclamation: "
		}
		message.Blocks = append(message.Blocks, SlackBlock{
			Type: "section",
			Text: SlackText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("%s%d tests; %d succeeded, %d skipped, %d failed, %d errored", icon, testReport.TotalTests, testReport.TotalPassing, testReport.TotalSkipped, testReport.TotalFailures, testReport.TotalErrors),
			},
		})
	}

	xrayService, err := services.NewXrayService(*cmd.slackConfiguration.serverDetails)
	if err != nil {
		return err
	}
	scanResult, _ := xrayService.GetBuildScanResult(cmd.buildConfiguration)
	if scanResult != nil {
		summary, _ := xrayService.GetBuildSummary(cmd.buildConfiguration)
		if summary != nil {
			if len(scanResult.Vulnerabilities) > 0 {
				var text string
				if len(scanResult.Violations) > 0 {
					text = fmt.Sprintf(":exclamation: <%s|%d violations>, %d security issues, %d operational risks", scanResult.MoreDetailsUrl, len(scanResult.Violations), len(scanResult.Vulnerabilities), len(summary.OperationalRisks))
				} else {
					text = fmt.Sprintf("%d security issues, %d operational risks", len(scanResult.Vulnerabilities), len(summary.OperationalRisks))
				}
				message.Blocks = append(message.Blocks, SlackBlock{
					Type: "section",
					Text: SlackText{
						Type: "mrkdwn",
						Text: text,
					},
				})
			}
		}
	}

	content, err := json.Marshal(message)
	if err != nil {
		return err
	}

	url := os.Getenv("int_" + cmd.slackConfiguration.slack + "_url")
	log.Debug("Posting message to " + url + "\n" + clientUtils.IndentJson(content))

	client, err := httpclient.ClientBuilder().Build()
	if err != nil {
		return err
	}
	httpClientDetails := httputils.HttpClientDetails{
		Headers: map[string]string{"Content-Type": "application/json"},
	}
	resp, body, err := client.SendPost(url, content, httpClientDetails, "")
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		log.Info("Successfully posted message to Slack")
		return nil
	} else {
		return errorutils.CheckErrorf(fmt.Sprintf("Failed posting message to Slack: %s.\n%s\n", resp.Status, body))
	}
}

type SlackConfiguration struct {
	serverID               string
	serverDetails          *utilsconfig.ServerDetails
	slack                  string
	includePrePostRunSteps bool
	failOnReject           bool
}

func (sc *SlackConfiguration) SetServerID(serverID string) *SlackConfiguration {
	sc.serverID = serverID
	return sc
}

func (sc *SlackConfiguration) SetIncludePrePostRunSteps(includePrePostRunSteps bool) *SlackConfiguration {
	sc.includePrePostRunSteps = includePrePostRunSteps
	return sc
}

func (sc *SlackConfiguration) SetFailOnReject(failOnReject bool) *SlackConfiguration {
	sc.failOnReject = failOnReject
	return sc
}

func (sc *SlackConfiguration) ValidateSlackConfiguration() (err error) {
	// If no server-id provided, use default server.
	serverDetails, err := utilsconfig.GetSpecificConfig(sc.serverID, true, false)
	if err != nil {
		return err
	}
	sc.serverDetails = serverDetails
	return nil
}

func (sc *SlackConfiguration) SetSlack(slack string) {
	sc.slack = slack
}

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackBlock struct {
	Type string    `json:"type"`
	Text SlackText `json:"text"`
}

type SlackAttachment struct {
	Color  string       `json:"color"`
	Blocks []SlackBlock `json:"blocks"`
}

type SlackMessage struct {
	Blocks      []SlackBlock      `json:"blocks"`
	Attachments []SlackAttachment `json:"attachments"`
}
