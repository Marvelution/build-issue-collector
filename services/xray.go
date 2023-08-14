package services

import (
	"encoding/json"
	"fmt"
	artutils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/auth"
	clientConfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/marvelution/ext-build-info/services/xray"
	"net/http"
)

type XrayService struct {
	client *jfroghttpclient.JfrogHttpClient
	auth.ServiceDetails
}

func NewXrayService(serverDetails utilsconfig.ServerDetails) (*XrayService, error) {
	pAuth, err := serverDetails.CreateXrayAuthConfig()
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

	return &XrayService{
		client:         client,
		ServiceDetails: pAuth,
	}, nil
}

func (xr *XrayService) GetBuildSummary(buildConfig *artutils.BuildConfiguration) (*xray.BuildSummary, error) {
	name, err := buildConfig.GetBuildName()
	if err != nil {
		return nil, err
	}
	number, err := buildConfig.GetBuildNumber()
	if err != nil {
		return nil, err
	}
	buildSummary := &xray.BuildSummary{}
	err = xr.GetRequest("api/v1/summary/build?build_name="+name+"&build_number="+number, buildSummary)
	if err != nil {
		return nil, err
	}
	buildSummary.Build.Number = number
	return buildSummary, nil
}

func (xr *XrayService) GetBuildScanResult(buildConfig *artutils.BuildConfiguration) (*xray.BuildScanResult, error) {
	name, err := buildConfig.GetBuildName()
	if err != nil {
		return nil, err
	}
	number, err := buildConfig.GetBuildNumber()
	if err != nil {
		return nil, err
	}
	buildScanResult := &xray.BuildScanResult{}
	err = xr.GetRequest("api/v2/ci/build/"+name+"/"+number+"?include_vulnerabilities=true", buildScanResult)
	if err != nil {
		return nil, err
	}
	return buildScanResult, nil
}

func (xr *XrayService) GetRequest(url string, response any) error {
	clientDetails := xr.CreateHttpClientDetails()
	utils.SetContentType("application/json", &clientDetails.Headers)
	fullUrl := xr.GetUrl() + url
	resp, body, _, err := xr.client.SendGet(fullUrl, false, &clientDetails)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return json.Unmarshal(body, response)
	} else {
		return errorutils.CheckErrorf(fmt.Sprintf("Response from Xray (%s): %s.\n%s\n", fullUrl, resp.Status, body))
	}
}
