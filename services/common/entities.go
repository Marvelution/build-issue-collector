package common

import (
	"github.com/jfrog/build-info-go/entities"
	"strings"
)

type State string

const (
	Successful State = "successful"
	Failed     State = "failed"
	Cancelled  State = "cancelled"
	InProgress State = "in_progress"
	Pending    State = "pending"
	Unknown    State = "unknown"
)

var BestToWorst = []State{Successful, Failed, Cancelled, InProgress, Pending, Unknown}

func GetState(code int64) State {
	if code == 4000 || code == 4005 {
		return Pending
	} else if code == 4001 {
		return InProgress
	} else if code == 4002 {
		return Successful
	} else if code == 4003 || code == 4004 || code == 4007 {
		return Failed
	} else if code == 4006 || code == 4008 {
		return Cancelled
	} else {
		return Unknown
	}
}

func GetStateByName(name string) State {
	if name == "" {
		return Successful
	} else {
		switch strings.ToLower(name) {
		case "successful":
			return Successful
		case "failed":
			return Failed
		case "cancelled":
			return Cancelled
		case "in_progress":
			return InProgress
		case "pending":
			return Pending
		default:
			return Unknown
		}
	}
}

func (s *State) Index() int {
	for index, state := range BestToWorst {
		if s == &state {
			return index
		}
	}
	return -1
}

func (s *State) IsWorstThan(state State) bool {
	return s.Index() > state.Index()
}

type BuildInfoByNumber []entities.BuildInfo

func (a BuildInfoByNumber) Len() int           { return len(a) }
func (a BuildInfoByNumber) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BuildInfoByNumber) Less(i, j int) bool { return a[i].Number < a[j].Number }
