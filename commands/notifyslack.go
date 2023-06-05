package commands

import (
	"encoding/json"
	"fmt"
	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	artservices "github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/http/httpclient"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/httputils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services"
	"net/http"
	"os"
	"strings"
)

type NotifySlackCommand struct {
	buildConfiguration *utils.BuildConfiguration
	SlackConfiguration *SlackConfiguration
}

func NewNotifySlackCommand() *NotifySlackCommand {
	return &NotifySlackCommand{}
}

func (cmd *NotifySlackCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *NotifySlackCommand {
	cmd.buildConfiguration = buildConfiguration
	return cmd
}

func (cmd *NotifySlackCommand) SetSlackConfiguration(slackConfiguration *SlackConfiguration) *NotifySlackCommand {
	cmd.SlackConfiguration = slackConfiguration
	return cmd
}

func (cmd *NotifySlackCommand) Run() error {
	log.Info("Collecting build-info to send to Slack.")

	buildInfo, err := cmd.getBuildInfo(cmd.SlackConfiguration)
	if err != nil {
		return err
	}

	pipelinesService, err := services.NewPipelinesService(*cmd.SlackConfiguration.serverDetails)
	if err != nil {
		return err
	}
	pipelineReport, err := pipelinesService.GetPipelineReport(buildInfo.Properties["buildInfo.env.run_id"], cmd.SlackConfiguration.includePrePostRunSteps)
	if err != nil {
		return err
	}

	var color string
	switch pipelineReport.State {
	case services.Successful:
		color = "#40be46"
		break
	case services.Failed:
		color = "#fc8675"
		break
	case services.Cancelled:
	case services.InProgress:
	case services.Pending:
		color = "#5183a0"
		break
	default:
		color = ""
		break
	}

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

	var testReport string
	if pipelineReport.TotalFailures > 0 || pipelineReport.TotalErrors > 0 {
		testReport = fmt.Sprintf("\\n%d / %d tests failed", pipelineReport.TotalFailures+pipelineReport.TotalErrors,
			pipelineReport.TotalTests)
	} else {
		testReport = ""
	}

	vcsReport := strings.Join(vcsInfo, " ")

	attachment := SlackAttachment{
		Color: color,
		Blocks: []SlackBlock{{
			Type: "section",
			Text: SlackText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("<%s|%s #%s> *%s* for %s%s",
					buildInfo.BuildUrl, buildInfo.Name, buildInfo.Number, pipelineReport.State, vcsReport, testReport),
			},
		}},
	}

	content, err := json.Marshal(SlackMessage{
		Attachments: []SlackAttachment{attachment},
	})
	if err != nil {
		return err
	}

	url := os.Getenv("int_" + cmd.SlackConfiguration.slack + "_url")
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

// Returns build info, or empty build info struct if not found.
func (cmd *NotifySlackCommand) getBuildInfo(jiraConfig *SlackConfiguration) (*buildinfo.BuildInfo, error) {
	// Create services manager to get build-info from Artifactory.
	sm, err := utils.CreateServiceManager(jiraConfig.serverDetails, -1, 0, false)
	if err != nil {
		return nil, err
	}

	buildName, err := cmd.buildConfiguration.GetBuildName()
	if err != nil {
		return nil, err
	}
	buildNumber, err := cmd.buildConfiguration.GetBuildNumber()
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
