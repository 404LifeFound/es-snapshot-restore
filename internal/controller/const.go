package controller

const (
	AnnotationRestoreTaskID     = "restore.elastic.co/task-id"
	AnnotationRestoreTaskStatus = "restore.elastic.co/state"

	ElasticsearchKind       = "Elasticsearch"
	ElasticsearchAPIVersion = "elasticsearch.k8s.elastic.co/v1"

	RestoreStatusPending = "pending"
	RestoreStatusRunning = "running"
	RestoreStatusDone    = "done"
	RestoreStatusFailed  = "failed"
)
