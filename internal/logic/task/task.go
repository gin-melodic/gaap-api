package task

import (
	"context"
	"encoding/json"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/mq"
	"gaap-api/internal/service"
	"gaap-api/internal/ws"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

type sTask struct{}

func init() {
	service.RegisterTask(New())
}

func New() *sTask {
	return &sTask{}
}

// ListTasks returns a list of tasks for the current user
func (s *sTask) ListTasks(ctx context.Context, in model.TaskQueryInput) (out []model.TaskOutput[any, any], total int, err error) {
	userId := utils.RequireUserId(ctx)

	m := dao.Tasks.Ctx(ctx)
	if userId != "" {
		m = m.Where(dao.Tasks.Columns().UserId, userId)
	}
	if in.Status != 0 {
		m = m.Where(dao.Tasks.Columns().Status, in.Status)
	}
	if in.Type != 0 {
		m = m.Where(dao.Tasks.Columns().Type, in.Type)
	}

	total, err = m.Count()
	if err != nil {
		return nil, 0, gerror.Wrap(err, "failed to count tasks")
	}

	var entities []entity.Tasks
	err = m.Order("created_at DESC").Page(in.Page, in.Limit).Scan(&entities)
	if err != nil {
		return nil, 0, gerror.Wrap(err, "failed to list tasks")
	}

	for _, e := range entities {
		task := s.entityToModel(&e)
		out = append(out, *task)
	}
	return
}

// GetTask returns a single task by ID
func (s *sTask) GetTask(ctx context.Context, id uuid.UUID) (out *model.TaskOutput[any, any], err error) {
	userId := utils.RequireUserId(ctx)

	var e entity.Tasks
	m := dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, id)
	if userId != "" {
		m = m.Where(dao.Tasks.Columns().UserId, userId)
	}
	err = m.Scan(&e)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get task")
	}
	if e.Id == uuid.Nil {
		return nil, gerror.New("task not found")
	}
	return s.entityToModel(&e), nil
}

// CreateTask creates a new task and publishes it to the queue
func (s *sTask) CreateTask(ctx context.Context, in model.TaskCreateInput[any]) (out *model.TaskOutput[any, any], err error) {
	// Check if RabbitMQ is connected before proceeding
	rabbit := mq.GetRabbitMQ()
	g.Log().Debugf(ctx, "CreateTask: RabbitMQ type: %T, IsConnected: %v", rabbit, rabbit.IsConnected())
	if !rabbit.IsConnected() {
		return nil, gerror.New("task queue is not available, please try again later")
	}

	payloadBytes, err := json.Marshal(in.Payload)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to marshal payload")
	}

	// Generate UUID7 for the new task
	taskId, err := uuid.NewV7()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to generate UUID7 for new task")
	}

	taskEntity := entity.Tasks{
		Id:      taskId,
		UserId:  in.UserId,
		Type:    in.Type,
		Status:  model.TaskStatusPending,
		Payload: string(payloadBytes),
	}

	_, err = dao.Tasks.Ctx(ctx).Data(taskEntity).Insert()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to create task")
	}

	g.Log().Infof(ctx, "Created task with ID: %s", taskId)

	// Publish to RabbitMQ
	msgPayload := g.Map{
		"taskId":  taskId.String(),
		"payload": in.Payload,
	}
	msgBytes, _ := json.Marshal(msgPayload)
	msg := &mq.Message{
		Type:    in.Type,
		Payload: msgBytes,
	}

	if err := mq.GetRabbitMQ().Publish(ctx, mq.QueueTasks, msg); err != nil {
		// Mark task as failed with the error reason
		failErr := gerror.Newf("failed to publish to queue: %v", err)
		s.FailTask(ctx, taskId, failErr.Error())
		return nil, gerror.Wrap(err, "failed to publish task to queue")
	}

	return s.GetTask(ctx, taskId)
}

// CancelTask cancels a pending or running task
func (s *sTask) CancelTask(ctx context.Context, id uuid.UUID) error {
	userId := utils.RequireUserId(ctx)

	m := dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, id)
	if userId != "" {
		m = m.Where(dao.Tasks.Columns().UserId, userId)
	}

	// Only allow cancelling pending or running tasks
	m = m.WhereIn(dao.Tasks.Columns().Status, []int{model.TaskStatusPending, model.TaskStatusRunning})

	_, err := m.Data(entity.Tasks{
		Status:      model.TaskStatusCancelled,
		CompletedAt: gtime.Now(),
	}).Update()

	if err != nil {
		return gerror.Wrap(err, "failed to cancel task")
	}
	return nil
}

// RetryTask retries a failed task
func (s *sTask) RetryTask(ctx context.Context, id uuid.UUID) (*model.TaskOutput[any, any], error) {
	userId := utils.RequireUserId(ctx)

	// Get the failed task
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, gerror.New("task not found")
	}
	if task.UserId.String() != userId {
		return nil, gerror.New("task not found")
	}
	if task.Status != model.TaskStatusFailed {
		return nil, gerror.New("only failed tasks can be retried")
	}

	// Check if RabbitMQ is connected
	if !mq.GetRabbitMQ().IsConnected() {
		return nil, gerror.New("task queue is not available, please try again later")
	}

	// Reset task status to pending
	_, err = dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, id).Data(entity.Tasks{
		Status:         model.TaskStatusPending,
		Progress:       0,
		ProcessedItems: 0,
	}).Update()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to reset task")
	}

	// Re-publish to RabbitMQ
	msgPayload := g.Map{
		"taskId":  id.String(),
		"payload": task.Payload,
	}
	msgBytes, _ := json.Marshal(msgPayload)
	msg := &mq.Message{
		Type:    task.Type,
		Payload: msgBytes,
	}

	if err := mq.GetRabbitMQ().Publish(ctx, mq.QueueTasks, msg); err != nil {
		// Mark task as failed again
		s.FailTask(ctx, id, gerror.Newf("failed to republish to queue: %v", err).Error())
		return nil, gerror.Wrap(err, "failed to retry task")
	}

	g.Log().Infof(ctx, "Retried task with ID: %s", id)
	return s.GetTask(ctx, id)
}

// UpdateTaskProgress updates task progress
func (s *sTask) UpdateTaskProgress(ctx context.Context, id uuid.UUID, progress int, processedItems int) error {
	_, err := dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, id).Data(entity.Tasks{
		Progress:       progress,
		ProcessedItems: processedItems,
	}).Update()
	if err != nil {
		return gerror.Wrap(err, "failed to update task progress")
	}
	return nil
}

// CompleteTask marks a task as completed
func (s *sTask) CompleteTask(ctx context.Context, id uuid.UUID, result interface{}) error {
	resultBytes, _ := json.Marshal(result)
	_, err := dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, id).Data(entity.Tasks{
		Status:      model.TaskStatusCompleted,
		Progress:    100,
		Result:      string(resultBytes),
		CompletedAt: gtime.Now(),
	}).Update()
	if err != nil {
		return gerror.Wrap(err, "failed to complete task")
	}

	// Broadcast task update via WebSocket
	s.broadcastTaskUpdate(ctx, id, model.TaskStatusCompleted, result)
	return nil
}

// FailTask marks a task as failed
func (s *sTask) FailTask(ctx context.Context, id uuid.UUID, errMsg string) error {
	result := model.AccountMigrationResult{Error: errMsg}
	resultBytes, _ := json.Marshal(result)
	_, err := dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, id).Data(entity.Tasks{
		Status:      model.TaskStatusFailed,
		Result:      string(resultBytes),
		CompletedAt: gtime.Now(),
	}).Update()
	if err != nil {
		return gerror.Wrap(err, "failed to mark task as failed")
	}

	// Broadcast task update via WebSocket
	s.broadcastTaskUpdate(ctx, id, model.TaskStatusFailed, result)
	return nil
}

// broadcastTaskUpdate sends task status update to user via WebSocket
func (s *sTask) broadcastTaskUpdate(ctx context.Context, taskId uuid.UUID, status model.TaskStatus, result interface{}) {
	task, err := s.GetTask(ctx, taskId)
	if err != nil || task == nil {
		g.Log().Warningf(ctx, "Failed to get task for WebSocket broadcast: %v", err)
		return
	}

	msg := &ws.Message{
		Type: ws.MessageTypeTaskUpdate,
		Payload: ws.TaskUpdatePayload{
			TaskId:   taskId.String(),
			Status:   status,
			TaskType: task.Type,
			Result:   result,
		},
	}

	ws.GetHub().SendToUser(task.UserId.String(), msg)
	g.Log().Infof(ctx, "Broadcasted task update via WebSocket: taskId=%s, status=%d, userId=%s", taskId, status, task.UserId)
}

// StartWorker starts the background task worker
func (s *sTask) StartWorker(ctx context.Context) error {
	return mq.GetRabbitMQ().Consume(ctx, mq.QueueTasks, func(ctx context.Context, msg *mq.Message) error {
		switch msg.Type {
		case model.TaskTypeAccountMigration:
			return s.processAccountMigration(ctx, msg.Payload)
		case model.TaskTypeDataExport:
			return s.processDataExport(ctx, msg.Payload)
		case model.TaskTypeDataImport:
			return s.processDataImport(ctx, msg.Payload)
		default:
			g.Log().Warningf(ctx, "Unknown task type: %d", msg.Type)
			return nil
		}
	})
}

// entityToModel converts entity to model
func (s *sTask) entityToModel(e *entity.Tasks) *model.TaskOutput[any, any] {
	var payload interface{}
	var result interface{}
	json.Unmarshal([]byte(e.Payload), &payload)
	if e.Result != "" {
		json.Unmarshal([]byte(e.Result), &result)
	}

	return &model.TaskOutput[any, any]{
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
