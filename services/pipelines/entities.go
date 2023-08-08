package pipelines

import (
	"github.com/marvelution/ext-build-info/services/common"
	"time"
)

type Pipeline struct {
	Id                    int64     `json:"id"`
	Name                  string    `json:"name"`
	Branch                string    `json:"pipelineSourceBranch"`
	ProjectId             int64     `json:"projectId"`
	PipelineSourceId      int64     `json:"pipelineSourceId"`
	LatestRunNumber       int64     `json:"latestRunNumber"`
	LatestRunId           int64     `json:"latestRunId"`
	LatestSuccessfulRunId int64     `json:"latestSuccessfulRunId"`
	LatestFailedRunId     int64     `json:"latestFailedRunId"`
	LatestErrorRunId      int64     `json:"latestErrorRunId"`
	LatestSkippedRunId    int64     `json:"latestSkippedRunId"`
	LatestTimedOutRunId   int64     `json:"latestTimedOutRunId"`
	LatestCancelledRunId  int64     `json:"latestCancelledRunId"`
	LatestUnstableRunId   int64     `json:"latestUnstableRunId"`
	LatestCompletedRunId  int64     `json:"latestCompletedRunId"`
	RetentionMinRuns      int64     `json:"retentionMinRuns"`
	RetentionMaxAgeDays   int64     `json:"retentionMaxAgeDays"`
	SyntaxVersion         string    `json:"syntaxVersion"`
	IsDeleted             bool      `json:"isDeleted"`
	DeletedAt             time.Time `json:"deletedAt"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type PipelineRunReport struct {
	Name               string             `json:"name"`
	Branch             string             `json:"branch"`
	RunId              int64              `json:"runId"`
	RunNumber          int64              `json:"runNumber"`
	EndedAt            time.Time          `json:"endedAt"`
	StartedAt          time.Time          `json:"startedAt"`
	State              common.State       `json:"state"`
	TestReport         PipelineTestReport `json:"tests"`
	RunResourceVersion RunResourceVersion `json:"run-resource-version"`
}

type PipelineTestReport struct {
	TotalTests    int64 `json:"totalTests"`
	TotalPassing  int64 `json:"totalPassing"`
	TotalFailures int64 `json:"totalFailures"`
	TotalErrors   int64 `json:"totalErrors"`
	TotalSkipped  int64 `json:"totalSkipped"`
}

type Step struct {
	ConfigPropertyBag            map[string]any `json:"configPropertyBag"`
	StaticPropertyBag            map[string]any `json:"staticPropertyBag"`
	SystemPropertyBag            map[string]any `json:"systemPropertyBag"`
	ExecPropertyBag              map[string]any `json:"execPropertyBag"`
	Id                           int64          `json:"id"`
	MasterResourceId             int64          `json:"masterResourceId"`
	PipelineId                   int64          `json:"pipelineId"`
	PipelineSourceId             int64          `json:"pipelineSourceId"`
	PipelineStepId               int64          `json:"pipelineStepId"`
	ProjectId                    int64          `json:"projectId"`
	Name                         string         `json:"name"`
	RunId                        int64          `json:"runId"`
	StatusCode                   int64          `json:"statusCode"`
	TypeCode                     int64          `json:"typeCode"`
	AffinityGroup                string         `json:"affinityGroup"`
	GroupInProgress              bool           `json:"groupInProgress"`
	GroupStartedAt               time.Time      `json:"groupStartedAt"`
	PendingLogsComplete          bool           `json:"pendingLogsComplete"`
	IsConsoleArchived            bool           `json:"isConsoleArchived"`
	FileStoreProvider            interface{}    `json:"fileStoreProvider"`
	PipelineStateArtifactName    interface{}    `json:"pipelineStateArtifactName"`
	TriggeredByResourceVersionId int64          `json:"triggeredByResourceVersionId"`
	TriggeredByStepId            interface{}    `json:"triggeredByStepId"`
	TriggeredByIdentityId        interface{}    `json:"triggeredByIdentityId"`
	TriggeredAt                  time.Time      `json:"triggeredAt"`
	TimeoutAt                    time.Time      `json:"timeoutAt"`
	ReadyAt                      time.Time      `json:"readyAt"`
	QueuedAt                     time.Time      `json:"queuedAt"`
	EndedAt                      time.Time      `json:"endedAt"`
	StartedAt                    time.Time      `json:"startedAt"`
	LastHeartbeatAt              time.Time      `json:"lastHeartbeatAt"`
	ApprovalRequestedAt          interface{}    `json:"approvalRequestedAt"`
	ExternalBuildId              interface{}    `json:"externalBuildId"`
	ExternalBuildUrl             interface{}    `json:"externalBuildUrl"`
	RequiresApproval             bool           `json:"requiresApproval"`
	IsApproved                   interface{}    `json:"isApproved"`
	CreatedAt                    time.Time      `json:"createdAt"`
	UpdatedAt                    time.Time      `json:"updatedAt"`
}

type StepTestReport struct {
	Id               int64 `json:"id"`
	ProjectId        int64 `json:"projectId"`
	PipelineSourceId int64 `json:"pipelineSourceId"`
	StepId           int64 `json:"stepId"`
	DurationSeconds  int64 `json:"durationSeconds"`
	TotalTests       int64 `json:"totalTests"`
	TotalPassing     int64 `json:"totalPassing"`
	TotalFailures    int64 `json:"totalFailures"`
	TotalErrors      int64 `json:"totalErrors"`
	TotalSkipped     int64 `json:"totalSkipped"`
}

type Run struct {
	StaticPropertyBag map[string]any `json:"staticPropertyBag"`
	SystemPropertyBag map[string]any `json:"systemPropertyBag"`
	Id                int64          `json:"id"`
	Name              string         `json:"name"`
	PipelineId        int64          `json:"pipelineId"`
	PipelineSourceId  int64          `json:"pipelineSourceId"`
	ProjectId         int64          `json:"projectId"`
	ParentRunId       int64          `json:"parentRunId"`
	RunNumber         int64          `json:"runNumber"`
	DurationSeconds   int64          `json:"durationSeconds"`
	StatusCode        int64          `json:"statusCode"`
	Description       string         `json:"description"`
	PubKey            string         `json:"pubKey"`
	MerkleLeaves      string         `json:"merkleLeaves"`
	MerkleRoot        string         `json:"merkleRoot"`
	EndedAt           time.Time      `json:"endedAt"`
	StartedAt         time.Time      `json:"startedAt"`
	TriggerId         int64          `json:"triggerId"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type RunResourceVersion struct {
	ResourceStaticPropertyBag         map[string]any `json:"resourceStaticPropertyBag"`
	ResourceConfigPropertyBag         map[string]any `json:"resourceConfigPropertyBag"`
	ResourceVersionContentPropertyBag map[string]any `json:"resourceVersionContentPropertyBag"`
	Id                                int64          `json:"id"`
	ProjectId                         int64          `json:"projectId"`
	PipelineSourceId                  int64          `json:"pipelineSourceId"`
	PipelineSourceBranch              string         `json:"pipelineSourceBranch"`
	MasterResourceId                  int64          `json:"masterResourceId"`
	PipelineId                        int64          `json:"pipelineId"`
	RunId                             int64          `json:"runId"`
	StepId                            int64          `json:"stepId"`
	ResourceName                      string         `json:"resourceName"`
	ResourceTypeCode                  int64          `json:"resourceTypeCode"`
	ResourceVersionId                 int64          `json:"resourceVersionId"`
	ResourceVersionCreatedByStepId    int64          `json:"resourceVersionCreatedByStepId"`
	CreatedAt                         time.Time      `json:"createdAt"`
	UpdatedAt                         time.Time      `json:"updatedAt"`
}

type Resource struct {
	YmlConfigPropertyBag        map[string]any `json:"ymlConfigPropertyBag"`
	SystemPropertyBag           map[string]any `json:"systemPropertyBag"`
	StaticPropertyBag           map[string]any `json:"staticPropertyBag"`
	Yml                         ResourceYaml   `json:"yml"`
	Id                          int64          `json:"id"`
	Name                        string         `json:"name"`
	PipelineSourceBranch        string         `json:"pipelineSourceBranch"`
	ProjectIntegrationId        int64          `json:"projectIntegrationId"`
	TypeCode                    int64          `json:"typeCode"`
	MasterResourceId            int64          `json:"masterResourceId"`
	LatestResourceVersionId     int64          `json:"latestResourceVersionId"`
	PinnedVersionId             int64          `json:"pinnedVersionId"`
	ProjectId                   int64          `json:"projectId"`
	PipelineSourceId            int64          `json:"pipelineSourceId"`
	IsDeleted                   bool           `json:"isDeleted"`
	DeletedAt                   time.Time      `json:"deletedAt"`
	IsConsistent                bool           `json:"isConsistent"`
	IsInternal                  bool           `json:"isInternal"`
	SyntaxVersion               string         `json:"syntaxVersion"`
	NextTriggerTime             time.Time      `json:"nextTriggerTime"`
	GitRepoFullName             string         `json:"gitRepoFullName"`
	GitIncludeBranchPattern     string         `json:"gitIncludeBranchPattern"`
	GitExcludeBranchPattern     string         `json:"gitExcludeBranchPattern"`
	GitBuildOnCommit            bool           `json:"gitBuildOnCommit"`
	GitBuildOnPullRequestCreate bool           `json:"gitBuildOnPullRequestCreate"`
	GitBuildOnPullRequestClose  bool           `json:"gitBuildOnPullRequestClose"`
	GitBuildOnReleaseCreate     bool           `json:"gitBuildOnReleaseCreate"`
	GitBuildOnTagCreate         bool           `json:"gitBuildOnTagCreate"`
	StoreVersionHistory         bool           `json:"storeVersionHistory"`
	GitBuildOnBranchDelete      bool           `json:"gitBuildOnBranchDelete"`
	CreatedAt                   time.Time      `json:"createdAt"`
	UpdatedAt                   time.Time      `json:"updatedAt"`
}

type ResourceYaml struct {
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Configuration map[string]any `json:"configuration"`
}

type ResourceVersion struct {
	ContentPropertyBag map[string]any `json:"contentPropertyBag"`
	Id                 int64          `json:"id"`
	ProjectId          int64          `json:"projectId"`
	PipelineSourceId   int64          `json:"pipelineSourceId"`
	ResourceId         int64          `json:"resourceId"`
	CreatedByStepId    int64          `json:"createdByStepId"`
	CreatedByRunId     int64          `json:"createdByRunId"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}
