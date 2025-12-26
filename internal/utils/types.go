package utils

type TaskStatus string

type Stag string

var (
	TaskPending  TaskStatus = "PENDING"
	TaskRunning  TaskStatus = "RUNNING"
	TaskSuccess  TaskStatus = "SUCCESS"
	TaskFailed   TaskStatus = "FAILED"
	TaskTimeout  TaskStatus = "TIMEOUT"
	TaskCanceled TaskStatus = "CANCELED"

	StagInit         Stag = "INIT"
	StagCreateESNode Stag = "CREATE_ES_NODE"
	StageCheckESNode Stag = "CHECK_ES_NODE"
	StagRestoreIndex Stag = "RESTORE_INDEX"
)
