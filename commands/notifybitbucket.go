package commands

import (
	"fmt"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services"
	"github.com/marvelution/ext-build-info/services/bitbucket"
	"os"
)

type NotifyBitbucketCommand struct {
	buildConfiguration     *utils.BuildConfiguration
	bitbucketConfiguration *BitbucketConfiguration
}

func NewNotifyBitbucketCommand() *NotifyBitbucketCommand {
	return &NotifyBitbucketCommand{}
}

func (cmd *NotifyBitbucketCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *NotifyBitbucketCommand {
	cmd.buildConfiguration = buildConfiguration
	return cmd
}

func (cmd *NotifyBitbucketCommand) SetBitbucketConfiguration(bitbucketConfiguration *BitbucketConfiguration) *NotifyBitbucketCommand {
	cmd.bitbucketConfiguration = bitbucketConfiguration
	return cmd
}

func (cmd *NotifyBitbucketCommand) Run() error {
	log.Info("Collecting build-info to send to Bitbucket.")

	pipelinesService, err := services.NewPipelinesService(*cmd.bitbucketConfiguration.serverDetails)
	if err != nil {
		return err
	}
	pipelineReport, err := pipelinesService.GetPipelineReport(os.Getenv("run_id"), cmd.bitbucketConfiguration.includePrePostRunSteps)
	if err != nil {
		return err
	}

	runResourceVersions := pipelineReport.GetGitRepoRunResourceVersions()
	if len(*runResourceVersions) > 0 {
		for _, runResourceVersion := range *runResourceVersions {
			log.Debug("Collecting vcs information from resource: " + runResourceVersion.ResourceName)
			shaData := runResourceVersion.ResourceVersionContentPropertyBag["shaData"]
			shaDataMap := shaData.(map[string]any)

			repo := runResourceVersion.ResourceVersionContentPropertyBag["path"].(string)
			commitSha := shaDataMap["commitSha"].(string)[0:8]

			testReport := pipelineReport.TestReport
			message := bitbucket.CreateCommitStatus{
				Key:  pipelineReport.Name,
				Name: fmt.Sprintf("%s %d", pipelineReport.Name, pipelineReport.RunNumber),
				Description: fmt.Sprintf("%d tests; %d succeeded, %d skipped, %d failed, %d errored",
					testReport.TotalTests, testReport.TotalPassing, testReport.TotalSkipped, testReport.TotalFailures, testReport.TotalErrors),
				Refname:   shaDataMap["branchName"].(string),
				Url:       os.Getenv("JFROG_CLI_BUILD_URL"),
				State:     bitbucket.GetState(pipelineReport.State),
				CreatedOn: pipelineReport.StartedAt,
				UpdatedOn: pipelineReport.EndedAt,
			}

			bitbucketService, err := services.NewBitbucketService(cmd.bitbucketConfiguration.bitbucketUrl,
				cmd.bitbucketConfiguration.bitbucketUsername, cmd.bitbucketConfiguration.bitbucketToken, cmd.bitbucketConfiguration.dryRun)
			if err != nil {
				return err
			}
			err = bitbucketService.SendCommitStatus(repo, commitSha, message)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type BitbucketConfiguration struct {
	serverID               string
	serverDetails          *utilsconfig.ServerDetails
	bitbucketID            string
	bitbucketUrl           string
	bitbucketUsername      string
	bitbucketToken         string
	dryRun                 bool
	includePrePostRunSteps bool
}

func (jc *BitbucketConfiguration) SetServerID(serverID string) *BitbucketConfiguration {
	jc.serverID = serverID
	return jc
}

func (jc *BitbucketConfiguration) SetBitbucketID(bitbucketID string) *BitbucketConfiguration {
	jc.bitbucketID = bitbucketID
	return jc
}

func (jc *BitbucketConfiguration) SetBitbucketDetails(url, username, token string) *BitbucketConfiguration {
	jc.bitbucketUrl = url
	jc.bitbucketUsername = username
	jc.bitbucketToken = token
	return jc
}

func (jc *BitbucketConfiguration) SetDryRun(dryRun bool) *BitbucketConfiguration {
	jc.dryRun = dryRun
	return jc
}

func (jc *BitbucketConfiguration) SetIncludePrePostRunSteps(includePrePostRunSteps bool) *BitbucketConfiguration {
	jc.includePrePostRunSteps = includePrePostRunSteps
	return jc
}

func (jc *BitbucketConfiguration) ValidateBitbucketConfiguration() (err error) {
	if jc.bitbucketUrl == "" {
		log.Debug("Loading Bitbucket details from integration ", jc.bitbucketID)
		jc.bitbucketUrl = os.Getenv("int_" + jc.bitbucketID + "_url")
		jc.bitbucketUsername = os.Getenv("int_" + jc.bitbucketID + "_username")
		jc.bitbucketToken = os.Getenv("int_" + jc.bitbucketID + "_token")
	}

	if jc.bitbucketUrl == "" || jc.bitbucketUsername == "" || jc.bitbucketToken == "" {
		return errorutils.CheckErrorf("Missing Bitbucket details")
	}

	// If no server-id provided, use default server.
	serverDetails, err := utilsconfig.GetSpecificConfig(jc.serverID, true, false)
	if err != nil {
		return err
	}
	jc.serverDetails = serverDetails
	return nil
}
