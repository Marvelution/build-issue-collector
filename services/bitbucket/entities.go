package bitbucket

import (
	"github.com/marvelution/ext-build-info/services/common"
	"time"
)

type CreateCommitStatus struct {
	Uuid        string           `json:"uuid,omitempty"`
	Key         string           `json:"key"`
	Refname     string           `json:"refname,omitempty"`
	Url         string           `json:"url"`
	State       CommitBuildState `json:"state"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	CreatedOn   time.Time        `json:"created_on"`
	UpdatedOn   time.Time        `json:"updated_on"`
}

type CommitBuildState string

const (
	Successful CommitBuildState = "SUCCESSFUL"
	Failed     CommitBuildState = "FAILED"
	Stopped    CommitBuildState = "STOPPED"
	InProgress CommitBuildState = "INPROGRESS"
)

func GetState(state common.State) CommitBuildState {
	switch state {
	case common.Successful:
		return Successful
	case common.Failed:
		return Failed
	case common.Cancelled:
	case common.Unknown:
		return Stopped
	case common.InProgress:
	case common.Pending:
	default:
		return InProgress
	}
	return ""
}
