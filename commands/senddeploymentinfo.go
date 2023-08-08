package commands

import (
	"fmt"
	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services"
	"github.com/marvelution/ext-build-info/services/jira"
	"github.com/marvelution/ext-build-info/services/pipelines"
	"github.com/marvelution/ext-build-info/util"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SendDeploymentInfoCommand struct {
	buildConfiguration *utils.BuildConfiguration
	jiraConfiguration  *JiraConfiguration
	deploymentInfo     *DeploymentInfo
}

func NewSendDeploymentInfoCommand() *SendDeploymentInfoCommand {
	return &SendDeploymentInfoCommand{}
}

func (cmd *SendDeploymentInfoCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *SendDeploymentInfoCommand {
	cmd.buildConfiguration = buildConfiguration
	return cmd
}

func (cmd *SendDeploymentInfoCommand) SetJiraConfiguration(jiraConfiguration *JiraConfiguration) *SendDeploymentInfoCommand {
	cmd.jiraConfiguration = jiraConfiguration
	return cmd
}

func (cmd *SendDeploymentInfoCommand) SetDeploymentInfo(deploymentInfo *DeploymentInfo) *SendDeploymentInfoCommand {
	cmd.deploymentInfo = deploymentInfo
	return cmd
}

func (cmd *SendDeploymentInfoCommand) Run() error {
	log.Info("Collecting deployment-info to send to Jira.")

	pipelinesService, err := services.NewPipelinesService(*cmd.jiraConfiguration.serverDetails)
	if err != nil {
		return err
	}

	// Get current run details
	currentRun, err := pipelinesService.GetRun(cmd.deploymentInfo.runId)
	if err != nil {
		return err
	}
	// Get current run resource version details
	triggeredByRunResourceVersionId := int64(currentRun.StaticPropertyBag["triggeredByRunResourceVersionId"].(float64))
	currentRunResourceVersion, err := pipelinesService.GetRunResourceVersion(triggeredByRunResourceVersionId)
	if err != nil {
		return err
	}
	// Get resource version details
	resourceVersion, err := pipelinesService.GetResourceVersion(currentRunResourceVersion.ResourceVersionId,
		currentRunResourceVersion.PipelineSourceBranch)
	if err != nil {
		return err
	}
	// Get CreatedBy Run details
	createdByRun, err := pipelinesService.GetRun(resourceVersion.CreatedByRunId)
	if err != nil {
		return err
	}
	// Get Previous CreatedBy Run
	attributes := map[string]string{
		"pipelineIds":       strconv.FormatInt(createdByRun.PipelineId, 10),
		"pipelineSourceIds": strconv.FormatInt(createdByRun.PipelineSourceId, 10),
		"createdBefore":     currentRun.CreatedAt.Add(-time.Minute * 1).Format(time.RFC3339),
		"limit":             "1",
		"sortBy":            "id",
		"sortOrder":         "-1",
	}
	previousRun, err := pipelinesService.FindRun(attributes)
	if err != nil {
		return err
	}
	attributes = map[string]string{
		"runIds":        strconv.FormatInt(previousRun.Id, 10),
		"resourceNames": currentRunResourceVersion.ResourceName,
	}
	previousRunResourceVersion, err := pipelinesService.FindRunResourceVersion(attributes)
	if err != nil {
		return err
	}

	firstExclusiveBuild := cmd.getBuildNumber(previousRunResourceVersion)
	lastInclusiveBuild := cmd.getBuildNumber(currentRunResourceVersion)

	log.Info(fmt.Sprintf("Collecting issues linked to build range %d (exclusive) and %d (inclusive)", firstExclusiveBuild,
		lastInclusiveBuild))

	var issueKeys []string

	buildInfoService, err := services.CreateExtBuildInfoService(cmd.jiraConfiguration.serverDetails)
	if err != nil {
		return err
	}

	buildInfos, err := buildInfoService.GetBuildInfosInRange(cmd.buildConfiguration, firstExclusiveBuild, lastInclusiveBuild, currentRunResourceVersion.PipelineSourceBranch)
	if err != nil {
		return err
	}
	var buildInfo = &buildinfo.BuildInfo{}
	for index, info := range *buildInfos {
		cmd.getIssueKeys(&info, &issueKeys)
		if index == len(*buildInfos)-1 {
			buildInfo = &info
		}
	}

	if len(issueKeys) > 0 {
		// We have issues, lets send the deployment-info
		client, err := services.NewOAuthJiraService(cmd.jiraConfiguration.jiraUrl, cmd.jiraConfiguration.jiraClientId,
			cmd.jiraConfiguration.jiraSecret, cmd.jiraConfiguration.dryRun)
		if err != nil {
			return err
		}

		_, _, state, _ := pipelinesService.GetRunSteps(currentRun.Id, cmd.jiraConfiguration.includePrePostRunSteps)

		jiraDeploymentInfo := jira.DeploymentInfo{
			SchemaVersion:            "1.0",
			DeploymentSequenceNumber: cmd.deploymentInfo.runNumber,
			UpdateSequenceNumber:     time.Now().UnixMilli(),
			Associations: []jira.Association{{
				AssociationType: jira.IssueIdOrKeysAssociation,
				Values:          issueKeys,
			}},
			DisplayName: cmd.deploymentInfo.GetDisplayName(),
			Url:         cmd.deploymentInfo.url,
			Description: "Deployment of " + buildInfo.Name + " #" + buildInfo.Number + " to " + cmd.deploymentInfo.environment,
			LastUpdated: time.Now(),
			State:       state,
			Pipeline:    cmd.deploymentInfo.GetPipeline(),
			Environment: cmd.deploymentInfo.GetEnvironment(),
		}

		response, err := client.SendDeploymentInfo(jiraDeploymentInfo)
		if err != nil {
			return err
		}
		if len(response.AcceptedDeployments) > 0 {
			for _, deployment := range response.AcceptedDeployments {
				log.Info("Deployment " + deployment.PipelineId + " #" + strconv.FormatInt(deployment.DeploymentSequenceNumber,
					10) + " was accepted by Jira")
			}
		}
		if len(response.RejectedDeployments) > 0 {
			for _, deployment := range response.RejectedDeployments {
				log.Warn("Deployment " + deployment.Key.PipelineId + " #" + strconv.FormatInt(deployment.Key.DeploymentSequenceNumber,
					10) + " was rejected by Jira")
				for _, deploymentError := range deployment.Errors {
					log.Warn(" - " + deploymentError.Message + " (" + deploymentError.ErrorTraceId + ")")
				}
			}
		}
		if len(response.UnknownIssueKeys) > 0 {
			log.Warn("The following issues are unknown by Jira: " + strings.Join(response.UnknownIssueKeys, ","))
		}
		if len(response.RejectedDeployments) > 0 && cmd.jiraConfiguration.failOnReject {
			return errorutils.CheckErrorf("There are " + strconv.Itoa(len(response.RejectedDeployments)) + " rejected deployments")
		}
	} else {
		log.Info("Nothing to send, no issue found")
	}

	return nil
}

func (cmd *SendDeploymentInfoCommand) getBuildNumber(resourceVersion *pipelines.RunResourceVersion) int64 {
	var buildNumber string
	if resourceVersion.ResourceVersionContentPropertyBag["buildNumber"] != nil {
		buildNumber = resourceVersion.ResourceVersionContentPropertyBag["buildNumber"].(string)
	} else {
		targetDeploymentPath := resourceVersion.ResourceVersionContentPropertyBag["targetDeploymentPath"].(string)
		regex := regexp.MustCompile("(.*)/([0-9]*)/")
		parts := regex.FindAllStringSubmatch(targetDeploymentPath, -1)
		buildNumber = parts[0][2]
	}
	number, err := strconv.ParseInt(buildNumber, 10, 64)
	if err != nil {
		panic(err)
	}
	return number
}

func (cmd *SendDeploymentInfoCommand) getIssueKeys(buildInfo *buildinfo.BuildInfo, issueKeys *[]string) {
	if buildInfo.Issues != nil && len(buildInfo.Issues.AffectedIssues) > 0 {
		for _, issue := range buildInfo.Issues.AffectedIssues {
			*issueKeys = append(*issueKeys, issue.Key)
		}
	}
}

type DeploymentInfo struct {
	name        string
	runId       int64
	runNumber   int64
	url         string
	environment string
}

func NewDeploymentInfo(environment string) *DeploymentInfo {
	runId, err := strconv.ParseInt(os.Getenv("run_id"), 10, 64)
	if err != nil {
		panic(err)
	}
	runNumber, err := strconv.ParseInt(os.Getenv("run_number"), 10, 64)
	if err != nil {
		panic(err)
	}
	return &DeploymentInfo{
		name:        os.Getenv("pipeline_name"),
		runId:       runId,
		runNumber:   runNumber,
		url:         os.Getenv("JFROG_CLI_BUILD_URL"),
		environment: environment,
	}
}

func (di *DeploymentInfo) GetDisplayName() string {
	return di.name + " #" + strconv.FormatInt(di.runNumber, 10)
}

func (di *DeploymentInfo) GetPipeline() jira.Pipeline {
	regex := regexp.MustCompile("(.*)/([0-9]*)/(.*)\\?(.*)")
	parts := regex.FindAllStringSubmatch(di.url, -1)
	url := parts[0][1] + "?" + parts[0][4]
	return jira.Pipeline{
		Id:          util.GenerateId(di.name),
		DisplayName: di.name,
		Url:         url,
	}
}

func (di *DeploymentInfo) GetEnvironment() jira.Environment {
	deploymentType := jira.Unmapped
	environmentLowercase := strings.ToLower(di.environment)
	if strings.EqualFold(environmentLowercase, "production") || strings.Contains(environmentLowercase, "prod") {
		deploymentType = jira.Production
	} else if strings.EqualFold(environmentLowercase, "staging") || strings.Contains(environmentLowercase, "stag") {
		deploymentType = jira.Staging
	} else if strings.Contains(environmentLowercase, "test") {
		deploymentType = jira.Testing
	} else {
		deploymentType = jira.Development
	}
	return jira.Environment{
		Id:          util.GenerateId(di.environment),
		DisplayName: di.environment,
		Type:        deploymentType,
	}
}
