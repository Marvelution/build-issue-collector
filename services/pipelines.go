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

func (ps *PipelinesService) GetPipelineReport(runId string, includePrePostRunSteps bool) (*PipelineReport, error) {
	pipelineReport := PipelineReport{
		State: Successful,
	}

	steps := &[]Step{}
	err := ps.GetRequest("api/v1/steps?runIds="+runId, &steps)
	if err != nil {
		return nil, err
	}
	var stepIds []string
	for _, step := range *steps {
		if (step.TypeCode != 2046 && step.TypeCode != 2047) || includePrePostRunSteps {
			stepIds = append(stepIds, strconv.FormatInt(step.Id, 10))
			state := GetState(step.StatusCode)
			if state.IsWorstThan(pipelineReport.State) {
				pipelineReport.State = state
			}
		}
	}

	stepTestReports := &[]StepTestReport{}
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

type PipelineReport struct {
	State         State
	TotalTests    int64 `json:"totalTests"`
	TotalPassing  int64 `json:"totalPassing"`
	TotalFailures int64 `json:"totalFailures"`
	TotalErrors   int64 `json:"totalErrors"`
	TotalSkipped  int64 `json:"totalSkipped"`
}

type Step struct {
	Id         int64  `json:"id"`
	PipelineId int64  `json:"pipelineId"`
	RunId      int64  `json:"runId"`
	StatusCode int64  `json:"statusCode"`
	TypeCode   int    `json:"typeCode"`
	Name       string `json:"name"`
}

type StepTestReport struct {
	Id               int64 `json:"id"`
	ProjectId        int64 `json:"projectId"`
	PipelineSourceId int64 `json:"pipelineSourceId"`
	StepId           int64 `json:"stepId"`
	DurationSeconds  int64 `json:"durationSeconds"`
	TotalTests       int64 `json:"totalTests"`
	TotalPassing     int64 `json:"totalPassing"`
	TotalFailures    int64 `json:"totalFailures"`
	TotalErrors      int64 `json:"totalErrors"`
	TotalSkipped     int64 `json:"totalSkipped"`
}
