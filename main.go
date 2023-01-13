package main

import (
	"errors"
	"fmt"
	artifactoryUtils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/plugins"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/marvelution/ext-build-info/commands"
	"github.com/marvelution/ext-build-info/docs/collectissues"
	"strconv"
)

func main() {
	plugins.PluginMain(components.App{
		Name:        "ext-build-info",
		Description: "Extended build info.",
		Version:     "v1.1.0",
		Commands: []components.Command{
			{
				Name:        "collect-issues",
				Description: collectissues.GetDescription(),
				Aliases:     []string{"ci"},
				Flags: []components.Flag{
					components.StringFlag{
						Name:        "server-id",
						Description: "Server ID configured using the config command.",
					},
					components.StringFlag{
						Name:        "project",
						Description: "Server ID configured using the config command.",
					},
					components.StringFlag{
						Name:        "tracker",
						Description: "Tracker to use to collect related issue from.",
					},
					components.StringFlag{
						Name:        "tracker-url",
						Description: "Tracker base url to use to collect related issue from.",
					},
					components.StringFlag{
						Name:        "tracker-username",
						Description: "Tracker username to use to collect related issue from.",
					},
					components.StringFlag{
						Name:        "tracker-token",
						Description: "Tracker token to use to collect related issue from.",
					},
					components.StringFlag{
						Name:        "regexp",
						Description: "A regular expression used for matching the git commit messages.",
					},
					components.StringFlag{
						Name:         "key-group-index",
						Description:  "The capturing group index in the regular expression used for retrieving the issue key.",
						DefaultValue: "1",
					},
					components.BoolFlag{
						Name:         "aggregate",
						Description:  "Set to true, if you wish all builds to include issues from previous builds.",
						DefaultValue: false,
					},
					components.StringFlag{
						Name: "aggregation-status",
						Description: "If aggregate is set to true, " +
							"this property indicates how far in time should the issues be aggregated. In the above example, " +
							"issues will be aggregated from previous builds, until a build with a RELEASE status is found. " +
							"Build statuses are set when a build is promoted using the jf rt build-promote command.",
					},
				},
				Arguments: []components.Argument{
					{
						Name:        "build name",
						Description: "The name of the build.",
					},
					{
						Name:        "build number",
						Description: "The number of the build.",
					},
					{
						Name:        "path to .git",
						Description: "Path to a directory containing the .git directory. If not specified, the .git directory is assumed to be in the current directory or in one of the parent directories.",
					},
				},
				Action: func(c *components.Context) error {
					return collectIssuesCmd(c)
				},
			},
		},
	})
}

func collectIssuesCmd(c *components.Context) error {
	nargs := len(c.Arguments)
	if nargs > 3 {
		return errors.New(fmt.Sprintf("Wrong number of arguments (%d).", nargs))
	}
	buildConfiguration := CreateBuildConfiguration(c)
	if err := buildConfiguration.ValidateBuildParams(); err != nil {
		return err
	}

	issueConfiguration, err := CreateIssueConfiguration(c)
	if err != nil {
		return err
	}
	if err := issueConfiguration.ValidateIssueConfiguration(); err != nil {
		return err
	}

	collectIssueCommand := commands.NewCollectIssueCommand().SetBuildConfiguration(buildConfiguration).SetIssuesConfig(issueConfiguration)
	if nargs == 3 {
		collectIssueCommand.SetDotGitPath(c.Arguments[2])
	} else if nargs == 1 {
		collectIssueCommand.SetDotGitPath(c.Arguments[0])
	}
	return collectIssueCommand.Run()
}

func CreateBuildConfiguration(c *components.Context) *artifactoryUtils.BuildConfiguration {
	buildConfiguration := new(artifactoryUtils.BuildConfiguration)
	buildNameArg, buildNumberArg := "", ""
	if len(c.Arguments) >= 2 {
		buildNameArg, buildNumberArg = c.Arguments[0], c.Arguments[1]
	}
	if buildNameArg == "" || buildNumberArg == "" {
		buildNameArg = ""
		buildNumberArg = ""
	}
	buildConfiguration.SetBuildName(buildNameArg).SetBuildNumber(buildNumberArg).SetProject(c.GetStringFlagValue("project"))
	return buildConfiguration
}

func CreateIssueConfiguration(c *components.Context) (*commands.IssuesConfiguration, error) {
	issueConfiguration := new(commands.IssuesConfiguration)
	issueConfiguration.SetServerID(c.GetStringFlagValue("server-id"))
	issueConfiguration.SetTracker(c.GetStringFlagValue("tracker"))
	if url := c.GetStringFlagValue("tracker-url"); url != "" {
		issueConfiguration.SetTrackerDetails(url, c.GetStringFlagValue("tracker-username"), c.GetStringFlagValue("tracker-token"))
	}
	issueConfiguration.SetRegexp(c.GetStringFlagValue("regexp"))
	if index := c.GetStringFlagValue("key-group-index"); index != "" {
		groupIndex, err := strconv.Atoi(index)
		if err != nil {
			return nil, err
		}
		issueConfiguration.SetKeyGroupIndex(groupIndex)
	}
	issueConfiguration.SetAggregate(c.GetBoolFlagValue("aggregate"))
	issueConfiguration.SetAggregationStatus(c.GetStringFlagValue("aggregation-status"))
	return issueConfiguration, nil
}
