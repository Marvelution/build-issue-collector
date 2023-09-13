package services

import (
	"encoding/json"
	buildinfo "github.com/jfrog/build-info-go/entities"
	artutils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/auth"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/marvelution/ext-build-info/services/common"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strconv"
)

func CreateExtBuildInfoService(serverDetails *utilsconfig.ServerDetails) (*ExtBuildInfoService, error) {
	sm, err := artutils.CreateServiceManager(serverDetails, -1, 0, false)
	if err != nil {
		return nil, err
	}
	return NewExtBuildInfoService(sm.GetConfig().GetServiceDetails(), sm.Client()), nil
}

type ExtBuildInfoService struct {
	client     *jfroghttpclient.JfrogHttpClient
	artDetails *auth.ServiceDetails
	DryRun     bool
}

func NewExtBuildInfoService(artDetails auth.ServiceDetails, client *jfroghttpclient.JfrogHttpClient) *ExtBuildInfoService {
	return &ExtBuildInfoService{artDetails: &artDetails, client: client}
}

func (bis *ExtBuildInfoService) GetArtifactoryDetails() auth.ServiceDetails {
	return *bis.artDetails
}

func (bis *ExtBuildInfoService) GetJfrogHttpClient() *jfroghttpclient.JfrogHttpClient {
	return bis.client
}

func (bis *ExtBuildInfoService) IsDryRun() bool {
	return bis.DryRun
}

func (bis *ExtBuildInfoService) GetBuildInfosInRange(buildConfig *artutils.BuildConfiguration, startExclusive, endInclusive int64, branch string) (*[]buildinfo.BuildInfo, error) {
	buildName, err := buildConfig.GetBuildName()
	if err != nil {
		return nil, err
	}
	buildRuns, err := bis.GetBuildRuns(buildName, buildConfig.GetProject())
	if err != nil {
		return nil, err
	}
	buildInfoService := bis.getBuildInfoService()
	buildInfos := &[]buildinfo.BuildInfo{}
	var nonNumericRegex = regexp.MustCompile(`[^0-9]+`)
	for _, build := range buildRuns.BuildsNumbers {
		buildNumber, err := strconv.ParseInt(nonNumericRegex.ReplaceAllString(build.Uri, ""), 10, 64)
		if err != nil {
			log.Debug("Excluding build "+build.Uri+"as it cannot be parsed to a build number", err)
		} else if buildNumber > startExclusive && buildNumber <= endInclusive {
			buildInfoParams := services.BuildInfoParams{
				BuildName:   buildName,
				BuildNumber: strconv.FormatInt(buildNumber, 10),
				ProjectKey:  buildConfig.GetProject(),
			}
			publishedBuildInfo, found, err := buildInfoService.GetBuildInfo(buildInfoParams)
			if err != nil {
				log.Warn("Excluding build-info "+buildInfoParams.BuildName+" #"+buildInfoParams.BuildNumber+" because of error:", err)
			} else if !found {
				log.Debug("Excluding build-info " + buildInfoParams.BuildName + " #" + buildInfoParams.BuildNumber + " it was not found")
			} else {
				for _, vcs := range publishedBuildInfo.BuildInfo.VcsList {
					if vcs.Branch == branch {
						log.Info("Including build-info " + buildInfoParams.BuildName + " #" + buildInfoParams.BuildNumber)
						*buildInfos = append(*buildInfos, publishedBuildInfo.BuildInfo)
						break
					} else {
						log.Debug("Excluding build-info " + buildInfoParams.BuildName + " #" + buildInfoParams.BuildNumber + " branch doesn't match")
					}
				}
			}
		}
	}
	sort.Sort(common.BuildInfoByNumber(*buildInfos))
	return buildInfos, nil
}

func (bis *ExtBuildInfoService) GetBuildRuns(buildName, projectKey string) (*BuildRuns, error) {
	httpClientsDetails := bis.GetArtifactoryDetails().CreateHttpClientDetails()
	restApi := path.Join("api/build/", buildName)

	queryParams := make(map[string]string)
	if projectKey != "" {
		queryParams["project"] = projectKey
	}

	requestFullUrl, err := utils.BuildArtifactoryUrl(bis.GetArtifactoryDetails().GetUrl(), restApi, queryParams)
	if err != nil {
		return nil, err
	}

	httpClient := bis.GetJfrogHttpClient()
	log.Debug("Getting build runs from: ", requestFullUrl)
	resp, body, _, err := httpClient.SendGet(requestFullUrl, true, &httpClientsDetails)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		log.Debug("Artifactory response: " + resp.Status + "\n" + clientutils.IndentJson(body))
		return nil, nil
	}
	if err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusOK); err != nil {
		return nil, err
	}

	// Build BuildInfo struct from json.
	runs := &BuildRuns{}
	if err := json.Unmarshal(body, runs); err != nil {
		return nil, err
	}
	log.Debug("Found runs: ", runs)
	return runs, nil
}

func (bis *ExtBuildInfoService) getBuildInfoService() *services.BuildInfoService {
	return services.NewBuildInfoService(bis.GetArtifactoryDetails(), bis.GetJfrogHttpClient())
}

type BuildRuns struct {
	Uri           string        `json:"uri"`
	BuildsNumbers []BuildNumber `json:"buildsNumbers"`
}

type BuildNumber struct {
	Uri string `json:"uri"`
	//Started time.Time `json:"started"`
}
