package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/yurika0211/luckyagent-runtime/tool"
	"github.com/yurika0211/luckyagent-tools/autonomy"
	taskstore "github.com/yurika0211/luckyagent-tools/task"
)

func (a *Agent) executeQueuedDelegate(ctx context.Context, description, input string, options tool.AgentExecutorOptions) (string, error) {
	if a.autonomy == nil {
		return "", fmt.Errorf("autonomy queue is not initialized")
	}
	parentID := tool.DelegateTaskID(ctx)
	q := a.autonomy.Queue()
	queued, _, err := q.AddWithAcceptance(description, input, autonomy.PriorityNormal, []string{"delegate"}, map[string]string{
		"idempotency_key":          "delegate:" + parentID,
		"delegate_task_id":         parentID,
		"delegate_workspace":       tool.ExtractDelegateWorkspace(input),
		"delegate_allow_recursive": strconv.FormatBool(options.AllowRecursiveDelegate),
	}, []string{"Complete the delegated task and verify the requested deliverable"})
	if err != nil {
		return "", err
	}
	if a.taskStore != nil {
		_ = a.taskStore.AppendEvent(taskstore.Event{Type: taskstore.EventChildCreated, TaskID: parentID, ChildID: queued.ID,
			ExecutionSessionID: queued.SessionID, Message: "delegated to durable autonomy queue"})
		if record, ok, err := a.taskStore.Get(parentID); err == nil && ok {
			if record.Metadata == nil {
				record.Metadata = make(map[string]string)
			}
			record.Metadata["autonomy_task_id"] = queued.ID
			record.Metadata["autonomy_execution_session_id"] = queued.SessionID
			_ = a.taskStore.Update(record)
		}
	}
	if err := a.StartAutonomyNow(ctx); err != nil {
		_ = q.Block(queued.ID, "could not start workers: "+err.Error())
		return "", err
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, ok := q.Get(queued.ID)
		if !ok {
			return "", fmt.Errorf("queued delegate disappeared")
		}
		switch current.State {
		case autonomy.TaskDone:
			if !current.Verified {
				return current.Result, fmt.Errorf("queued delegate was not verified")
			}
			return current.Result, nil
		case autonomy.TaskBlocked:
			return current.Result, fmt.Errorf("queued delegate blocked: %s", current.BlockReason)
		}
		select {
		case <-ctx.Done():
			if err := a.autonomy.Pool().CancelTask(queued.ID, "delegating task ended: "+ctx.Err().Error()); err != nil {
				return "", fmt.Errorf("%w; cancel queued task: %v", ctx.Err(), err)
			}
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}
