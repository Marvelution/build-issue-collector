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
	"github.com/jfrog/jfrog-client-go/utils/log"
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

func (ps *PipelinesService) GetPipelineReport(runId string, includePrePostRunSteps bool) (*pipelines.PipelineRunReport, error) {
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

	pipelineReport := pipelines.PipelineRunReport{
		Name:       pipeline.Name,
		Branch:     pipeline.Branch,
		RunId:      parsedRunId,
		RunNumber:  run.RunNumber,
		EndedAt:    run.EndedAt,
		StartedAt:  run.StartedAt,
		State:      common.Unknown,
		TestReport: pipelines.PipelineTestReport{},
	}

	_, stepIds, state, err := ps.GetRunSteps(parsedRunId, includePrePostRunSteps)
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
		pipelineReport.TestReport.TotalPassing += stepTestReport.TotalPassing
		pipelineReport.TestReport.TotalFailures += stepTestReport.TotalFailures
		pipelineReport.TestReport.TotalErrors += stepTestReport.TotalErrors
		pipelineReport.TestReport.TotalSkipped += stepTestReport.TotalSkipped
		pipelineReport.TestReport.TotalTests += stepTestReport.TotalTests
	}

	runResourceVersions, err := ps.GetRunResourceVersions(parsedRunId)
	if err != nil {
		return nil, err
	}
	pipelineReport.RunResourceVersions = *runResourceVersions

	return &pipelineReport, nil
}

func (ps *PipelinesService) GetRunSteps(runId int64, includePrePostRunSteps bool) (*[]pipelines.Step, []string, common.State, error) {
	steps := &[]pipelines.Step{}
	err := ps.GetRequest("api/v1/steps?runIds="+strconv.FormatInt(runId, 10), &steps)
	if err != nil {
		return nil, nil, common.Unknown, err
	}
	finalState := common.Successful
	var stepIds []string
	for _, step := range *steps {
		if (step.TypeCode != 2046 && step.TypeCode != 2047) || includePrePostRunSteps {
			stepIds = append(stepIds, strconv.FormatInt(step.Id, 10))
			state := common.GetState(step.StatusCode)
			log.Debug("Step "+step.Name+" resulted", step.StatusCode, state)
			if state.IsWorstThan(finalState) {
				finalState = state
			}
		}
	}
	return steps, stepIds, finalState, nil
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
	} else if len(*runs) > 1 {
		for _, run := range *runs {
			if run.StatusCode == 4002 {
				return &run, nil
			}
		}
	}
	return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline run found with %s\n", params))
}

func (ps *PipelinesService) GetRunResourceVersion(versionId string) (*pipelines.RunResourceVersion, error) {
	versions := &[]pipelines.RunResourceVersion{}
	err := ps.GetRequest("api/v1/runResourceVersions?runResourceVersionIds="+versionId, versions)
	if err != nil {
		return nil, err
	}
	if len(*versions) == 1 {
		return &(*versions)[0], nil
	} else {
		return nil, errorutils.CheckErrorf(fmt.Sprintf("No pipeline run resource version found with id %d\n", versionId))
	}
}

func (ps *PipelinesService) GetRunResourceVersions(runId int64) (*[]pipelines.RunResourceVersion, error) {
	versions := &[]pipelines.RunResourceVersion{}
	err := ps.GetRequest("api/v1/runResourceVersions?runIds="+strconv.FormatInt(runId, 10), versions)
	if err != nil {
		return nil, err
	}
	return versions, nil
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
