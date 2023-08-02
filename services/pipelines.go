package services

import (
	"encoding/json"
	"fmt"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/auth"
	clientConfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/marvelution/ext-build-info/services/common"
	"github.com/marvelution/ext-build-info/services/pipelines"
	"net/http"
	"strconv"
	"strings"
)

type PipelinesService struct {
	client *jfroghttpclient.JfrogHttpClient
	auth.ServiceDetails
}

func NewPipelinesService(serverDetails utilsconfig.ServerDetails) (*PipelinesService, error) {
	pAuth, err := serverDetails.CreatePipelinesAuthConfig()
	if err != nil {
		return nil, err
	}
	config, err := clientConfig.NewConfigBuilder().
		SetServiceDetails(pAuth).
		SetDryRun(false).
		Build()
	if err != nil {
		return nil, err
	}

	client, err := jfroghttpclient.JfrogClientBuilder().
		SetCertificatesPath(config.GetCertificatesPath()).
		SetInsecureTls(config.IsInsecureTls()).
		SetClientCertPath(serverDetails.GetClientCertPath()).
		SetClientCertKeyPath(serverDetails.GetClientCertKeyPath()).
		AppendPreRequestInterceptor(config.GetServiceDetails().RunPreRequestFunctions).
		SetContext(config.GetContext()).
		SetRetries(config.GetHttpRetries()).
		SetRetryWaitMilliSecs(config.GetHttpRetryWaitMilliSecs()).
		Build()

	if err != nil {
		return nil, err
	}

	return &PipelinesService{
		client:         client,
		ServiceDetails: pAuth,
	}, nil
}

func (ps *PipelinesService) GetPipelineReport(runId string, includePrePostRunSteps bool) (*pipelines.PipelineReport, error) {
	parsedRunId, err := strconv.ParseInt(runId, 10, 64)
	if err != nil {
		return nil, err
	}
	run, err := ps.GetRun(parsedRunId)
	if err != nil {
		return nil, err
	}
	pipeline, err := ps.GetPipeline(run.PipelineId)
	if err != nil {
		return nil, err
	}

	pipelineReport := pipelines.PipelineReport{
		Name:      pipeline.Name,
		Branch:    pipeline.Branch,
		RunId:     parsedRunId,
		RunNumber: run.RunNumber,
		State:     common.Successful,
	}

	state, stepIds, err := ps.GetRunState(parsedRunId, includePrePostRunSteps)
	if err != nil {
		return nil, err
	}
	pipelineReport.State = state

	stepTestReports := &[]pipelines.StepTestReport{}
	err = ps.GetRequest("api/v1/stepTestReports?stepIds="+strings.Join(stepIds, ","), &stepTestReports)
	if err != nil {
		return nil, err
	}
	for _, stepTestReport := range *stepTestReports {
		pipelineReport.TotalPassing += stepTestReport.TotalPassing
		pipelineReport.TotalFailures += stepTestReport.TotalFailures
		pipelineReport.TotalErrors += stepTestReport.TotalErrors
		pipelineReport.TotalSkipped += stepTestReport.TotalSkipped
		pipelineReport.TotalTests += stepTestReport.TotalTests
	}

	return &pipelineReport, nil
}

func (ps *PipelinesService) GetRunState(runId int64, includePrePostRunSteps bool) (common.State, []string, error) {
	steps := &[]pipelines.Step{}
	err := ps.GetRequest("api/v1/steps?runIds="+strconv.FormatInt(runId, 10), &steps)
	if err != nil {
		return common.Unknown, nil, err
	}
	finalState := common.Successful
	var stepIds []string
	for _, step := range *steps {
		if (step.TypeCode != 2046 && step.TypeCode != 2047) || includePrePostRunSteps {
			stepIds = append(stepIds, strconv.FormatInt(step.Id, 10))
			state := common.GetState(step.StatusCode)
			if state.IsWorstThan(finalState) {
				finalState = state
			}
		}
	}
	return finalState, stepIds, nil
}

func (ps *PipelinesService) GetRun(runId int64) (*pipelines.Run, error) {
	runs := &[]pipelines.Run{}
	err := ps.GetRequest("api/v1/runs?runIds="+strconv.FormatInt(runId, 10), &runs)
	if err != nil {
		return nil, err
	}
	if len(*runs) == 1 {
		return &(*runs)[0], nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline run found with id %s\n", runId))
	}
}

func (ps *PipelinesService) FindRun(attributes map[string]string) (*pipelines.Run, error) {
	runs := &[]pipelines.Run{}
	params := ps.CreateParams(attributes)
	err := ps.GetRequest("api/v1/runs?"+params, &runs)
	if err != nil {
		return nil, err
	}
	if len(*runs) == 1 {
		return &(*runs)[0], nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline run found with %s\n", params))
	}
}

func (ps *PipelinesService) GetRunResourceVersion(versionId int64) (*pipelines.RunResourceVersion, error) {
	versions := &[]pipelines.RunResourceVersion{}
	err := ps.GetRequest("api/v1/runResourceVersions?runResourceVersionIds="+strconv.FormatInt(versionId, 10), versions)
	if err != nil {
		return nil, err
	}
	if len(*versions) == 1 {
		return &(*versions)[0], nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline run resource version found with id %d\n", versionId))
	}
}

func (ps *PipelinesService) FindRunResourceVersion(attributes map[string]string) (*pipelines.RunResourceVersion, error) {
	versions := &[]pipelines.RunResourceVersion{}
	params := ps.CreateParams(attributes)
	err := ps.GetRequest("api/v1/runResourceVersions?"+params, versions)
	if err != nil {
		return nil, err
	}
	if len(*versions) == 1 {
		return &(*versions)[0], nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline run resource version found with %s\n", params))
	}
}

func (ps *PipelinesService) GetResource(resourceName string, pipelineSourceId int64, pipelineSourceBranch string) (*pipelines.Resource, error) {
	resources := &[]pipelines.Resource{}
	err := ps.GetRequest("api/v1/resources?names="+resourceName+"&pipelineSourceIds="+strconv.FormatInt(pipelineSourceId,
		10)+"&pipelineSourceBranches="+pipelineSourceBranch, resources)
	if err != nil {
		return nil, err
	}
	if len(*resources) == 1 {
		return &(*resources)[0], nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline resource found with name: %s sourceId: %d branch: %s\n",
			resourceName, pipelineSourceId, pipelineSourceBranch))
	}
}

func (ps *PipelinesService) GetResourceVersion(versionId int64, pipelineSourceBranch string) (*pipelines.ResourceVersion, error) {
	versions := &[]pipelines.ResourceVersion{}
	err := ps.GetRequest("api/v1/resourceVersions?resourceVersionIds="+strconv.FormatInt(versionId, 10)+"&pipelineSourceBranches="+pipelineSourceBranch, versions)
	if err != nil {
		return nil, err
	}
	if len(*versions) == 1 {
		return &(*versions)[0], nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline resource version found with id: %d, branch: %s\n",
			versionId, pipelineSourceBranch))
	}
}

func (ps *PipelinesService) CreateParams(attributes map[string]string) string {
	var params strings.Builder
	for key, value := range attributes {
		params.WriteString(key)
		params.WriteString("=")
		params.WriteString(value)
		params.WriteString("&")
	}
	return params.String()
}

func (ps *PipelinesService) GetRequest(url string, response any) error {
	clientDetails := ps.CreateHttpClientDetails()
	utils.SetContentType("application/json", &clientDetails.Headers)
	fullUrl := ps.GetUrl() + url
	resp, body, _, err := ps.client.SendGet(fullUrl, false, &clientDetails)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return json.Unmarshal(body, response)
	} else {
		return errorutils.CheckErrorf(fmt.Sprintf("Response from Pipelines (%s): %s.\n%s\n", fullUrl, resp.Status, body))
	}
}

func (ps *PipelinesService) GetPipeline(pipelineId int64) (*pipelines.Pipeline, error) {
	pipeline := &pipelines.Pipeline{}
	err := ps.GetRequest("api/v1/pipelines/"+strconv.FormatInt(pipelineId, 10), pipeline)
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}
