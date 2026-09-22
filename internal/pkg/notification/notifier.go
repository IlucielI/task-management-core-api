package notification

import (
	"context"
	"log"

	"task-management/internal/models"
)

// Notifier defines the contract for sending notifications upon domain actions.
type Notifier interface {
	SendTaskAssignedNotification(ctx context.Context, task *models.Task, assignee *models.User) error
}

// LogNotifier is the default implementation that logs notification events.
type LogNotifier struct{}

// NewLogNotifier creates a new LogNotifier instance.
func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

// SendTaskAssignedNotification logs the assignment notification event.
func (n *LogNotifier) SendTaskAssignedNotification(ctx context.Context, task *models.Task, assignee *models.User) error {
	log.Printf("[NOTIFICATION] Task '%s' (ID: %s) assigned to user '%s' (ID: %s, Email: %s)",
		task.Title, task.ID, assignee.Name, assignee.ID, assignee.Email)
	return nil
}
