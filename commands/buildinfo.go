package commands

import (
	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	artservices "github.com/jfrog/jfrog-client-go/artifactory/services"
)

// Returns build info, or empty build info struct if not found.
func getBuildInfo(buildConfig *utils.BuildConfiguration, serverDetails *utilsconfig.ServerDetails) (*buildinfo.BuildInfo, error) {
	// Create services manager to get build-info from Artifactory.
	sm, err := utils.CreateServiceManager(serverDetails, -1, 0, false)
	if err != nil {
		return nil, err
	}

	buildName, err := buildConfig.GetBuildName()
	if err != nil {
		return nil, err
	}
	buildNumber, err := buildConfig.GetBuildNumber()
	if err != nil {
		return nil, err
	}

	bis := artservices.NewBuildInfoService(sm.GetConfig().GetServiceDetails(), sm.Client())

	publishedBuildInfo, found, err := bis.GetBuildInfo(artservices.BuildInfoParams{BuildName: buildName, BuildNumber: buildNumber})
	if err != nil {
		return nil, err
	}
	if !found {
		return &buildinfo.BuildInfo{}, nil
	}

	return &publishedBuildInfo.BuildInfo, nil
}
