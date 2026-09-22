package constants

// TaskStatus represents valid workflow status codes for a task.
type TaskStatus string

// Task workflow statuses
const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCodeReview TaskStatus = "code_review"
	TaskStatusReadyForQA TaskStatus = "ready_for_qa"
	TaskStatusDone       TaskStatus = "done"
)

// TaskAction represents valid action types for task audit logs.
type TaskAction string

// Task audit actions
const (
	TaskActionAssign       TaskAction = "ASSIGN"
	TaskActionCreate       TaskAction = "CREATE"
	TaskActionUpdate       TaskAction = "UPDATE"
	TaskActionStatusUpdate TaskAction = "STATUS_UPDATE"
	TaskActionDelete       TaskAction = "DELETE"
)
