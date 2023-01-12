package collectissues

var Usage = []string{"ext-build-info collect-issues [command options] <build name> <build number> [Path To .git]"}

func GetDescription() string {
	return "Collect issue details from git and add them to a build."
}
