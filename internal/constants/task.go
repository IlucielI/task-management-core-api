package constants

// Task workflow statuses
const (
	TaskStatusTodo       = "todo"
	TaskStatusInProgress = "in_progress"
	TaskStatusCodeReview = "code_review"
	TaskStatusReadyForQA = "ready_for_qa"
	TaskStatusDone       = "done"
)

// Task audit actions
const (
	TaskActionAssign       = "ASSIGN"
	TaskActionCreate       = "CREATE"
	TaskActionUpdate       = "UPDATE"
	TaskActionStatusUpdate = "STATUS_UPDATE"
	TaskActionDelete       = "DELETE"
)
