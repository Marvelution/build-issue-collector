package util

import (
	"regexp"
	"strings"
)

func GetSshVcsUrl(vcsUrl string) string {
	if strings.HasPrefix(vcsUrl, "git@") {
		return vcsUrl
	}
	re := regexp.MustCompile(`^https://(.*)/(.*)$`)
	return re.ReplaceAllString(vcsUrl, "git@$1:$2")
}

func GetHttpsVcsUrl(vcsUrl string) string {
	if strings.HasPrefix(vcsUrl, "https://") {
		return vcsUrl
	}
	re := regexp.MustCompile(`^git@(.*):(.*)$`)
	return re.ReplaceAllString(vcsUrl, "https://$1/$2")
}
