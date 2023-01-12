package commands

import (
	"errors"
	"github.com/marvelution/ext-build-info/jira"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	utilsconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	artclientutils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	gofrogcmd "github.com/marvelution/ext-build-info/io"

	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const (
	GitLogLimit   = 100
	IssueKeyRegex = "(((?:\\p{Lu}[\\p{Lu}\\p{N}_]+|\\p{Ll}[\\p{Ll}\\p{N}_]+))-\\p{N}+)"
)

type CollectIssueCommand struct {
	buildConfiguration  *utils.BuildConfiguration
	dotGitPath          string
	issuesConfiguration *IssuesConfiguration
}

func NewCollectIssueCommand() *CollectIssueCommand {
	return &CollectIssueCommand{}
}

func (config *CollectIssueCommand) SetBuildConfiguration(buildConfiguration *utils.BuildConfiguration) *CollectIssueCommand {
	config.buildConfiguration = buildConfiguration
	return config
}

func (config *CollectIssueCommand) SetDotGitPath(dotGitPath string) *CollectIssueCommand {
	config.dotGitPath = dotGitPath
	return config
}

func (config *CollectIssueCommand) SetIssuesConfig(issuesConfiguration *IssuesConfiguration) *CollectIssueCommand {
	config.issuesConfiguration = issuesConfiguration
	return config
}

func (config *CollectIssueCommand) Run() error {
	log.Info("Reading the git branch, revision and remote URL and adding them to the build-info.")

	buildName, err := config.buildConfiguration.GetBuildName()
	if err != nil {
		return err
	}
	buildNumber, err := config.buildConfiguration.GetBuildNumber()
	if err != nil {
		return err
	}

	err = utils.SaveBuildGeneralDetails(buildName, buildNumber, config.buildConfiguration.GetProject())
	if err != nil {
		return err
	}

	// Find .git if it wasn't provided in the command.
	if config.dotGitPath == "" {
		var exists bool
		config.dotGitPath, exists, err = fileutils.FindUpstream(".git", fileutils.Any)
		if err != nil {
			return err
		}
		if !exists {
			return errorutils.CheckErrorf("Could not find .git")
		}
	}

	// Collect URL, branch and revision into GitManager.
	gitManager := clientutils.NewGitManager(config.dotGitPath)
	err = gitManager.ReadConfig()
	if err != nil {
		return err
	}

	// Collect issues if required.
	var issues []buildinfo.AffectedIssue
	if config.issuesConfiguration.tracker != nil {
		log.Debug("Collecting issues hosted on ", config.issuesConfiguration.tracker.Name)
		issues, err = config.collectBuildIssues(gitManager.GetUrl())
		if err != nil {
			return err
		}
	}

	// Populate partials with VCS info.
	populateFunc := func(partial *buildinfo.Partial) {
		partial.VcsList = append(partial.VcsList, buildinfo.Vcs{
			Url:      gitManager.GetUrl(),
			Revision: gitManager.GetRevision(),
			Branch:   gitManager.GetBranch(),
			Message:  gitManager.GetMessage(),
		})

		if config.issuesConfiguration.tracker != nil {
			partial.Issues = &buildinfo.Issues{
				Tracker:                &buildinfo.Tracker{Name: config.issuesConfiguration.tracker.Name, Version: ""},
				AggregateBuildIssues:   config.issuesConfiguration.aggregate,
				AggregationBuildStatus: config.issuesConfiguration.aggregationStatus,
				AffectedIssues:         issues,
			}
		}
	}
	err = utils.SavePartialBuildInfo(buildName, buildNumber, config.buildConfiguration.GetProject(), populateFunc)
	if err != nil {
		return err
	}

	// Done.
	log.Info("Collected", len(issues), "issue details for", buildName+"/"+buildNumber+".")
	return nil
}

func (config *CollectIssueCommand) collectBuildIssues(vcsUrl string) ([]buildinfo.AffectedIssue, error) {
	log.Info("Collecting build issues from VCS...")

	// Check that git exists in path.
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, errorutils.CheckError(err)
	}

	// Get latest build's VCS revision from Artifactory.
	lastVcsRevision, err := config.getLatestVcsRevision(vcsUrl)
	if err != nil {
		return nil, err
	}

	// Run issues collection.
	return config.DoCollect(config.issuesConfiguration, lastVcsRevision)
}

func (config *CollectIssueCommand) DoCollect(issuesConfig *IssuesConfiguration, lastVcsRevision string) ([]buildinfo.AffectedIssue, error) {
	var foundIssueKeys []string
	logRegExp, err := createLogRegExpHandler(issuesConfig, &foundIssueKeys)
	if err != nil {
		return nil, err
	}

	errRegExp, err := createErrRegExpHandler(lastVcsRevision)
	if err != nil {
		return nil, err
	}

	// Get log with limit, starting from the latest commit.
	logCmd := &LogCmd{logLimit: issuesConfig.logLimit, lastVcsRevision: lastVcsRevision}

	// Change working dir to where .git is.
	wd, err := os.Getwd()
	if errorutils.CheckError(err) != nil {
		return nil, err
	}
	defer os.Chdir(wd)
	err = os.Chdir(config.dotGitPath)
	if errorutils.CheckError(err) != nil {
		return nil, err
	}

	// Run git command.
	_, _, exitOk, err := gofrogcmd.RunCmdWithOutputParser(logCmd, false, logRegExp, errRegExp)
	if err != nil {
		if _, ok := err.(RevisionRangeError); ok {
			// Revision not found in range. Ignore and don't collect new issues.
			log.Info(err.Error())
			return []buildinfo.AffectedIssue{}, nil
		}
		return nil, errorutils.CheckError(err)
	}
	if !exitOk {
		// May happen when trying to run git log for non-existing revision.
		return nil, errorutils.CheckErrorf("failed executing git log command")
	}

	if len(foundIssueKeys) > 0 {
		// Return found issues.
		return issuesConfig.tracker.ProcessorFunc(foundIssueKeys)
	} else {
		return []buildinfo.AffectedIssue{}, nil
	}
}

// Creates a regexp handler to parse and fetch issues from the output of the git log command.
func createLogRegExpHandler(issuesConfig *IssuesConfiguration, foundIssues *[]string) (*gofrogcmd.CmdOutputPattern, error) {
	// Create regex pattern.
	issueRegexp, err := clientutils.GetRegExp(issuesConfig.regexp)
	if err != nil {
		return nil, err
	}

	// Create handler with exec function.
	logRegExp := gofrogcmd.CmdOutputPattern{
		RegExp: issueRegexp,
		ExecFunc: func(pattern *gofrogcmd.CmdOutputPattern) (string, error) {
			// Reached here - means no error occurred.
			var found []string
			for _, matches := range pattern.MatchedResults {
				if len(matches)-1 < issuesConfig.keyGroupIndex {
					return "", errors.New("unexpected result while parsing issues from git log. " +
						"Make sure that the regular expression used to find issues, includes a capturing group, for the issue ID")
				}
				found = append(found, matches[issuesConfig.keyGroupIndex])
			}
			log.Debug("Found issues: ", found)
			*foundIssues = append(*foundIssues, found...)
			return "", nil
		},
	}
	return &logRegExp, nil
}

// Error to be thrown when revision could not be found in the git revision range.
type RevisionRangeError struct {
	ErrorMsg string
}

func (err RevisionRangeError) Error() string {
	return err.ErrorMsg
}

// Creates a regexp handler to handle the event of revision missing in the git revision range.
func createErrRegExpHandler(lastVcsRevision string) (*gofrogcmd.CmdOutputPattern, error) {
	// Create regex pattern.
	invalidRangeExp, err := clientutils.GetRegExp(`fatal: Invalid revision range [a-fA-F0-9]+\.\.`)
	if err != nil {
		return nil, err
	}

	// Create handler with exec function.
	errRegExp := gofrogcmd.CmdOutputPattern{
		RegExp: invalidRangeExp,
		ExecFunc: func(pattern *gofrogcmd.CmdOutputPattern) (string, error) {
			// Revision could not be found in the revision range, probably due to a squash / revert. Ignore and don't collect new issues.
			errMsg := "Revision: '" + lastVcsRevision + "' that was fetched from latest build info does not exist in the git revision range. No new issues are added."
			return "", RevisionRangeError{ErrorMsg: errMsg}
		},
	}
	return &errRegExp, nil
}

func (config *CollectIssueCommand) getLatestVcsRevision(vcsUrl string) (string, error) {
	// Get latest build's build-info from Artifactory
	buildInfo, err := config.getLatestBuildInfo(config.issuesConfiguration)
	if err != nil {
		return "", err
	}

	sshVcsUrl := config.getSshVcsUrl(vcsUrl)
	httpsVcsUrl := config.getHttpsVcsUrl(vcsUrl)

	// Get previous VCS Revision from BuildInfo.
	lastVcsRevision := ""
	for _, vcs := range buildInfo.VcsList {
		if vcs.Url == sshVcsUrl || vcs.Url == httpsVcsUrl {
			lastVcsRevision = vcs.Revision
			break
		}
	}

	return lastVcsRevision, nil
}

func (config *CollectIssueCommand) getSshVcsUrl(vcsUrl string) string {
	if strings.HasPrefix(vcsUrl, "git@") {
		return vcsUrl
	}
	re := regexp.MustCompile(`^https://(.*)/(.*)$`)
	return re.ReplaceAllString(vcsUrl, "git@$1:$2")
}

func (config *CollectIssueCommand) getHttpsVcsUrl(vcsUrl string) string {
	if strings.HasPrefix(vcsUrl, "https://") {
		return vcsUrl
	}
	re := regexp.MustCompile(`^git@(.*):(.*)$`)
	return re.ReplaceAllString(vcsUrl, "https://$1/$2")
}

// Returns build info, or empty build info struct if not found.
func (config *CollectIssueCommand) getLatestBuildInfo(issuesConfig *IssuesConfiguration) (*buildinfo.BuildInfo, error) {
	// Create services manager to get build-info from Artifactory.
	sm, err := utils.CreateServiceManager(issuesConfig.serverDetails, -1, 0, false)
	if err != nil {
		return nil, err
	}

	// Get latest build-info from Artifactory.
	buildName, err := config.buildConfiguration.GetBuildName()
	if err != nil {
		return nil, err
	}
	buildInfoParams := services.BuildInfoParams{BuildName: buildName, BuildNumber: artclientutils.LatestBuildNumberKey}
	publishedBuildInfo, found, err := sm.GetBuildInfo(buildInfoParams)
	if err != nil {
		return nil, err
	}
	if !found {
		return &buildinfo.BuildInfo{}, nil
	}

	return &publishedBuildInfo.BuildInfo, nil
}

type IssuesConfiguration struct {
	serverID          string
	serverDetails     *utilsconfig.ServerDetails
	logLimit          int
	tracker           *Tracker
	regexp            string
	keyGroupIndex     int
	aggregate         bool
	aggregationStatus string
}

func (ic *IssuesConfiguration) SetServerID(serverID string) *IssuesConfiguration {
	ic.serverID = serverID
	return ic
}

func (ic *IssuesConfiguration) SetLogLimit(logLimit int) *IssuesConfiguration {
	ic.logLimit = logLimit
	return ic
}

func (ic *IssuesConfiguration) SetTracker(tracker string) *IssuesConfiguration {
	if tracker != "" {
		ic.tracker = new(Tracker)
		ic.tracker.Name = tracker
	}
	return ic
}

func (ic *IssuesConfiguration) SetTrackerDetails(url, username, token string) *IssuesConfiguration {
	ic.tracker.Url = url
	ic.tracker.Username = username
	ic.tracker.Token = token
	return ic
}

func (ic *IssuesConfiguration) SetRegexp(regexp string) *IssuesConfiguration {
	ic.regexp = regexp
	return ic
}

func (ic *IssuesConfiguration) SetKeyGroupIndex(keyGroupIndex int) *IssuesConfiguration {
	ic.keyGroupIndex = keyGroupIndex
	return ic
}

func (ic *IssuesConfiguration) SetAggregate(aggregate bool) *IssuesConfiguration {
	ic.aggregate = aggregate
	return ic
}

func (ic *IssuesConfiguration) SetAggregationStatus(aggregationStatus string) *IssuesConfiguration {
	ic.aggregationStatus = aggregationStatus
	return ic
}

func (ic *IssuesConfiguration) ValidateIssueConfiguration() (err error) {
	if ic.logLimit >= 0 {
		ic.logLimit = GitLogLimit
	}

	if ic.tracker != nil {
		if strings.EqualFold(ic.tracker.Name, "JIRA") {
			ic.tracker.ProcessorFunc = func(foundIssueKeys []string) ([]buildinfo.AffectedIssue, error) {
				client, err := jira.NewClient(ic.tracker.Url, ic.tracker.Username, ic.tracker.Token)
				if err != nil {
					return nil, err
				}
				return client.GetIssues(foundIssueKeys)
			}
			if ic.tracker.Url == "" {
				log.Debug("Loading Jira details from integration ", ic.tracker.Name)
				ic.tracker.Url = os.Getenv("int_" + ic.tracker.Name + "_url")
				ic.tracker.Username = os.Getenv("int_" + ic.tracker.Name + "_username")
				ic.tracker.Token = os.Getenv("int_" + ic.tracker.Name + "_token")
			}

			if ic.tracker.Url == "" || ic.tracker.Username == "" || ic.tracker.Token == "" {
				return errorutils.CheckErrorf("Missing Jira details")
			}

			ic.tracker.Name = "JIRA"
			ic.regexp = IssueKeyRegex
			ic.keyGroupIndex = 1
		} else {
			return errorutils.CheckErrorf("Unsupported tracker: %s", "tracker", err.Error())
		}
	}

	// If no server-id provided, use default server.
	serverDetails, err := utilsconfig.GetSpecificConfig(ic.serverID, true, false)
	if err != nil {
		return err
	}
	ic.serverDetails = serverDetails
	return nil
}

type Tracker struct {
	Name          string
	Url           string
	Username      string
	Token         string
	ProcessorFunc func(foundIssueKeys []string) ([]buildinfo.AffectedIssue, error)
}

type LogCmd struct {
	logLimit        int
	lastVcsRevision string
}

func (logCmd *LogCmd) GetCmd() *exec.Cmd {
	var cmd []string
	cmd = append(cmd, "git")
	cmd = append(cmd, "log", "--pretty='format:%d%s'", "-"+strconv.Itoa(logCmd.logLimit))
	if logCmd.lastVcsRevision != "" {
		cmd = append(cmd, logCmd.lastVcsRevision+"..")
	}
	log.Debug("Fetching git log: ", cmd)
	return exec.Command(cmd[0], cmd[1:]...)
}

func (logCmd *LogCmd) GetEnv() map[string]string {
	return map[string]string{}
}

func (logCmd *LogCmd) GetStdWriter() io.WriteCloser {
	return nil
}

func (logCmd *LogCmd) GetErrWriter() io.WriteCloser {
	return nil
}
