package commands

import (
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"os"
)

type CleanSlateCommand struct {
	buildConfiguration *utils.BuildConfiguration
}

func NewCleanSlateCommand() *CleanSlateCommand {
	return &CleanSlateCommand{}
}

func (config *CleanSlateCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *CleanSlateCommand {
	config.buildConfiguration = buildConfiguration
	return config
}

func (config *CleanSlateCommand) Run() error {
	log.Info("Clearing all existing build-info to start from a clean slate.")

	buildName, err := config.buildConfiguration.GetBuildName()
	if err != nil {
		return err
	}
	buildNumber, err := config.buildConfiguration.GetBuildNumber()
	if err != nil {
		return err
	}

	buildDir, err := utils.GetBuildDir(buildName, buildNumber, config.buildConfiguration.GetProject())
	if err != nil {
		return err
	}
	log.Info("Removing build-info directory: " + buildDir)
	err = os.RemoveAll(buildDir)
	if err != nil {
		return err
	}

	return nil
}
