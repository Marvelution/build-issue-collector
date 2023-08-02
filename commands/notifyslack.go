package commands

import (
	"encoding/json"
	"fmt"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/http/httpclient"
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

	var color string
	switch pipelineReport.State {
	case common.Successful:
		color = "#40be46"
		break
	case common.Failed:
		color = "#fc8675"
		break
	case common.Cancelled:
	case common.InProgress:
	case common.Pending:
		color = "#5183a0"
		break
	default:
		color = ""
		break
	}

	var testReport string
	if pipelineReport.TotalFailures > 0 || pipelineReport.TotalErrors > 0 {
		testReport = fmt.Sprintf("\\n%d / %d tests failed", pipelineReport.TotalFailures+pipelineReport.TotalErrors,
			pipelineReport.TotalTests)
	} else {
		testReport = ""
	}

	var vcsReport string
	buildInfo, err := getBuildInfo(cmd.buildConfiguration, cmd.slackConfiguration.serverDetails)
	if err != nil {
		buildName, _ := cmd.buildConfiguration.GetBuildName()
		buildNumber, _ := cmd.buildConfiguration.GetBuildNumber()
		log.Info("No build-info found for " + buildName + " #" + buildNumber + ", no data from it will ne available for the notification.")
		vcsReport = ""
	} else {
		revisions := map[string]struct{}{}
		var vcsInfo []string
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
		vcsReport = strings.Join(vcsInfo, " ")
	}

	attachment := SlackAttachment{
		Color: color,
		Blocks: []SlackBlock{{
			Type: "section",
			Text: SlackText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("<%s|%s #%d> *%s* for %s%s",
					os.Getenv("JFROG_CLI_BUILD_URL"), pipelineReport.Name, pipelineReport.RunNumber, pipelineReport.State, vcsReport, testReport),
			},
		}},
	}

	content, err := json.Marshal(SlackMessage{
		Attachments: []SlackAttachment{attachment},
	})
	if err != nil {
		return err
	}

	url := os.Getenv("int_" + cmd.slackConfiguration.slack + "_url")
	log.Debug("Posting message to "+url, string(content))

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
	Attachments []SlackAttachment `json:"attachments"`
}
