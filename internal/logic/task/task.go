package task

import (
	"context"
	"encoding/json"
	"fmt"

	"gaap-api/internal/dao"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/mq"
	"gaap-api/internal/service"
	"gaap-api/internal/ws"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type sTask struct{}

func init() {
	service.RegisterTask(New())
}

func New() *sTask {
	return &sTask{}
}

// ListTasks returns a list of tasks for the current user
func (s *sTask) ListTasks(ctx context.Context, in model.TaskQueryInput) (out []model.Task, total int, err error) {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	m := dao.Tasks.Ctx(ctx)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	if in.Status != "" {
		m = m.Where("status", in.Status)
	}
	if in.Type != "" {
		m = m.Where("type", in.Type)
	}

	total, err = m.Count()
	if err != nil {
		return
	}

	var entities []entity.Tasks
	err = m.Order("created_at DESC").Page(in.Page, in.Limit).Scan(&entities)
	if err != nil {
		return
	}

	for _, e := range entities {
		task := s.entityToModel(&e)
		out = append(out, *task)
	}
	return
}

// GetTask returns a single task by ID
func (s *sTask) GetTask(ctx context.Context, id string) (out *model.Task, err error) {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	var e entity.Tasks
	m := dao.Tasks.Ctx(ctx).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	err = m.Scan(&e)
	if err != nil {
		return
	}
	if e.Id == "" {
		return nil, fmt.Errorf("task not found")
	}
	return s.entityToModel(&e), nil
}

// CreateTask creates a new task and publishes it to the queue
func (s *sTask) CreateTask(ctx context.Context, in model.TaskCreateInput) (out *model.Task, err error) {
	// Check if RabbitMQ is connected before proceeding
	rabbit := mq.GetRabbitMQ()
	fmt.Printf("CreateTask: RabbitMQ type: %T, IsConnected: %v\n", rabbit, rabbit.IsConnected())
	if !rabbit.IsConnected() {
		return nil, fmt.Errorf("task queue is not available, please try again later")
	}

	payloadBytes, err := json.Marshal(in.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Use Raw SQL with RETURNING to get the UUID
	var taskId string
	sql := `INSERT INTO tasks (user_id, type, status, payload) VALUES ($1, $2, $3, $4) RETURNING id`
	result, err := g.DB().GetOne(ctx, sql, in.UserId, in.Type, model.TaskStatusPending, string(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	taskId = result["id"].String()

	g.Log().Infof(ctx, "Created task with ID: %s", taskId)

	// Publish to RabbitMQ
	msg := &mq.Message{
		Type:    in.Type,
		Payload: payloadBytes,
	}
	// Add task ID to payload for worker
	msgPayload := g.Map{
		"taskId":  taskId,
		"payload": in.Payload,
	}
	msg.Payload, _ = json.Marshal(msgPayload)

	if err := mq.GetRabbitMQ().Publish(ctx, mq.QueueTasks, msg); err != nil {
		// Mark task as failed with the error reason
		failErr := fmt.Sprintf("failed to publish to queue: %v", err)
		s.FailTask(ctx, taskId, failErr)
		return nil, fmt.Errorf("failed to publish task to queue: %w", err)
	}

	return s.GetTask(ctx, taskId)
}

// CancelTask cancels a pending or running task
func (s *sTask) CancelTask(ctx context.Context, id string) error {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	m := dao.Tasks.Ctx(ctx).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}

	// Only allow cancelling pending or running tasks
	m = m.WhereIn("status", []string{model.TaskStatusPending, model.TaskStatusRunning})

	_, err := m.Data(g.Map{
		"status":       model.TaskStatusCancelled,
		"completed_at": gtime.Now(),
	}).Update()

	return err
}

// RetryTask retries a failed task
func (s *sTask) RetryTask(ctx context.Context, id string) (*model.Task, error) {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	// Get the failed task
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}
	if task.UserId != userId {
		return nil, fmt.Errorf("task not found")
	}
	if task.Status != model.TaskStatusFailed {
		return nil, fmt.Errorf("only failed tasks can be retried")
	}

	// Check if RabbitMQ is connected
	if !mq.GetRabbitMQ().IsConnected() {
		return nil, fmt.Errorf("task queue is not available, please try again later")
	}

	// Reset task status to pending
	_, err = dao.Tasks.Ctx(ctx).Where("id", id).Data(g.Map{
		"status":          model.TaskStatusPending,
		"progress":        0,
		"processed_items": 0,
		"result":          nil,
		"started_at":      nil,
		"completed_at":    nil,
	}).Update()
	if err != nil {
		return nil, fmt.Errorf("failed to reset task: %w", err)
	}

	// Re-publish to RabbitMQ
	msgPayload := g.Map{
		"taskId":  id,
		"payload": task.Payload,
	}
	msgBytes, _ := json.Marshal(msgPayload)
	msg := &mq.Message{
		Type:    task.Type,
		Payload: msgBytes,
	}

	if err := mq.GetRabbitMQ().Publish(ctx, mq.QueueTasks, msg); err != nil {
		// Mark task as failed again
		s.FailTask(ctx, id, fmt.Sprintf("failed to republish to queue: %v", err))
		return nil, fmt.Errorf("failed to retry task: %w", err)
	}

	g.Log().Infof(ctx, "Retried task with ID: %s", id)
	return s.GetTask(ctx, id)
}

// UpdateTaskProgress updates task progress
func (s *sTask) UpdateTaskProgress(ctx context.Context, id string, progress int, processedItems int) error {
	_, err := dao.Tasks.Ctx(ctx).Where("id", id).Data(g.Map{
		"progress":        progress,
		"processed_items": processedItems,
	}).Update()
	return err
}

// CompleteTask marks a task as completed
func (s *sTask) CompleteTask(ctx context.Context, id string, result interface{}) error {
	resultBytes, _ := json.Marshal(result)
	_, err := dao.Tasks.Ctx(ctx).Where("id", id).Data(g.Map{
		"status":       model.TaskStatusCompleted,
		"progress":     100,
		"result":       string(resultBytes),
		"completed_at": gtime.Now(),
	}).Update()
	if err != nil {
		return err
	}

	// Broadcast task update via WebSocket
	s.broadcastTaskUpdate(ctx, id, model.TaskStatusCompleted, result)
	return nil
}

// FailTask marks a task as failed
func (s *sTask) FailTask(ctx context.Context, id string, errMsg string) error {
	result := model.AccountMigrationResult{Error: errMsg}
	resultBytes, _ := json.Marshal(result)
	_, err := dao.Tasks.Ctx(ctx).Where("id", id).Data(g.Map{
		"status":       model.TaskStatusFailed,
		"result":       string(resultBytes),
		"completed_at": gtime.Now(),
	}).Update()
	if err != nil {
		return err
	}

	// Broadcast task update via WebSocket
	s.broadcastTaskUpdate(ctx, id, model.TaskStatusFailed, result)
	return nil
}

// broadcastTaskUpdate sends task status update to user via WebSocket
func (s *sTask) broadcastTaskUpdate(ctx context.Context, taskId string, status string, result interface{}) {
	task, err := s.GetTask(ctx, taskId)
	if err != nil || task == nil {
		g.Log().Warningf(ctx, "Failed to get task for WebSocket broadcast: %v", err)
		return
	}

	msg := &ws.Message{
		Type: ws.MessageTypeTaskUpdate,
		Payload: ws.TaskUpdatePayload{
			TaskId:   taskId,
			Status:   status,
			TaskType: task.Type,
			Result:   result,
		},
	}

	ws.GetHub().SendToUser(task.UserId, msg)
	g.Log().Infof(ctx, "Broadcasted task update via WebSocket: taskId=%s, status=%s, userId=%s", taskId, status, task.UserId)
}

// StartWorker starts the background task worker
func (s *sTask) StartWorker(ctx context.Context) error {
	return mq.GetRabbitMQ().Consume(ctx, mq.QueueTasks, func(ctx context.Context, msg *mq.Message) error {
		switch msg.Type {
		case model.TaskTypeAccountMigration:
			return s.processAccountMigration(ctx, msg.Payload)
		default:
			g.Log().Warningf(ctx, "Unknown task type: %s", msg.Type)
			return nil
		}
	})
}

// processAccountMigration handles account migration task
func (s *sTask) processAccountMigration(ctx context.Context, payload json.RawMessage) error {
	var data struct {
		TaskId  string                        `json:"taskId"`
		Payload model.AccountMigrationPayload `json:"payload"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	taskId := data.TaskId
	migrationPayload := data.Payload

	// Update task status to running
	_, err := dao.Tasks.Ctx(ctx).Where("id", taskId).Data(g.Map{
		"status":     model.TaskStatusRunning,
		"started_at": gtime.Now(),
	}).Update()
	if err != nil {
		return err
	}

	// Check if task was cancelled
	task, err := s.GetTask(ctx, taskId)
	if err != nil || task.Status == model.TaskStatusCancelled {
		return nil
	}

	// Execute migration in transaction
	result := model.AccountMigrationResult{}
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		return s.executeMigration(ctx, tx, taskId, &migrationPayload, &result)
	})

	if err != nil {
		s.FailTask(ctx, taskId, err.Error())
		return err
	}

	return s.CompleteTask(ctx, taskId, result)
}

// executeMigration performs the actual migration within a transaction
func (s *sTask) executeMigration(ctx context.Context, tx gdb.TX, taskId string, payload *model.AccountMigrationPayload, result *model.AccountMigrationResult) error {
	accountIds := append([]string{payload.AccountId}, payload.ChildAccountIds...)
	batchSize := 500

	// Count total transactions to migrate
	var totalCount int
	for _, accId := range accountIds {
		count, _ := tx.Model("transactions").
			Where("from_account_id = ? OR to_account_id = ?", accId, accId).
			Count()
		totalCount += count
	}

	// Update total items
	tx.Model("tasks").Where("id", taskId).Data(g.Map{"total_items": totalCount}).Update()

	processed := 0

	// Migrate transactions for each account
	for _, accId := range accountIds {
		// Get account currency
		var acc entity.Accounts
		tx.Model("accounts").Where("id", accId).Scan(&acc)

		targetId := payload.MigrationTargets[acc.Currency]
		if targetId == "" {
			continue
		}

		// Update from_account_id
		fromResult, err := tx.Model("transactions").
			Where("from_account_id", accId).
			Data(g.Map{"from_account_id": targetId}).
			Update()
		if err != nil {
			return err
		}
		fromCount, _ := fromResult.RowsAffected()
		result.TransactionsMigrated += int(fromCount)

		// Update to_account_id
		toResult, err := tx.Model("transactions").
			Where("to_account_id", accId).
			Data(g.Map{"to_account_id": targetId}).
			Update()
		if err != nil {
			return err
		}
		toCount, _ := toResult.RowsAffected()
		result.TransactionsMigrated += int(toCount)

		// Merge balance
		_, err = tx.Model("accounts").
			Where("id", targetId).
			Data(gdb.Raw(fmt.Sprintf("balance = balance + %f", acc.Balance))).
			Update()
		if err != nil {
			return err
		}
		result.BalancesMerged++

		// Soft delete account
		_, err = tx.Model("accounts").
			Where("id", accId).
			Data(g.Map{"deleted_at": gtime.Now()}).
			Update()
		if err != nil {
			return err
		}
		result.AccountsDeleted++

		// Update progress
		processed += int(fromCount) + int(toCount)
		progress := 0
		if totalCount > 0 {
			progress = (processed * 100) / totalCount
		}
		tx.Model("tasks").Where("id", taskId).Data(g.Map{
			"progress":        progress,
			"processed_items": processed,
		}).Update()

		// Check for cancellation periodically
		var taskStatus string
		tx.Model("tasks").Where("id", taskId).Fields("status").Scan(&taskStatus)
		if taskStatus == model.TaskStatusCancelled {
			return fmt.Errorf("task cancelled by user")
		}

		// Small batch delay to prevent overwhelming
		if processed%batchSize == 0 {
			g.Log().Debugf(ctx, "Migration progress: %d/%d", processed, totalCount)
		}
	}

	return nil
}

// entityToModel converts entity to model
func (s *sTask) entityToModel(e *entity.Tasks) *model.Task {
	var payload interface{}
	var result interface{}
	json.Unmarshal([]byte(e.Payload), &payload)
	if e.Result != "" {
		json.Unmarshal([]byte(e.Result), &result)
	}

	return &model.Task{
		Id:             e.Id,
		UserId:         e.UserId,
		Type:           e.Type,
		Status:         e.Status,
		Payload:        payload,
		Result:         result,
		Progress:       e.Progress,
		TotalItems:     e.TotalItems,
		ProcessedItems: e.ProcessedItems,
		StartedAt:      e.StartedAt,
		CompletedAt:    e.CompletedAt,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}
