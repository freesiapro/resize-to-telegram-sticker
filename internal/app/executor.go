package app

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

type TaskType string

const TaskTypeVideoSticker TaskType = "video_sticker"

type Task struct {
	ID      int
	Type    TaskType
	Label   string
	Payload any
}

type TaskHandler interface {
	Handle(ctx context.Context, task Task) Result
}

type TaskEventType int

const (
	TaskStarted TaskEventType = iota
	TaskFinished
)

type TaskEvent struct {
	Type   TaskEventType
	Task   Task
	Result Result
}

type Executor struct {
	Concurrency int
	Handlers    map[TaskType]TaskHandler
}

func (e Executor) Run(ctx context.Context, tasks []Task, events chan<- TaskEvent) error {
	if events == nil {
		return fmt.Errorf("events channel is nil")
	}
	if len(tasks) == 0 {
		close(events)
		return nil
	}
	defer close(events)

	concurrency := e.concurrency(len(tasks))
	workCh := make(chan Task)
	var wg sync.WaitGroup

	handlerLookup := e.Handlers
	if handlerLookup == nil {
		handlerLookup = map[TaskType]TaskHandler{}
	}

	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case task, ok := <-workCh:
				if !ok {
					return
				}
				if ctx.Err() != nil {
					return
				}
				events <- TaskEvent{Type: TaskStarted, Task: task}
				handler := handlerLookup[task.Type]
				result := runTask(ctx, task, handler)
				events <- TaskEvent{Type: TaskFinished, Task: task, Result: result}
			}
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			close(workCh)
			wg.Wait()
			return ctx.Err()
		case workCh <- task:
		}
	}
	close(workCh)
	wg.Wait()
	return nil
}

func (e Executor) concurrency(taskCount int) int {
	if taskCount <= 0 {
		return 0
	}
	concurrency := e.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.GOMAXPROCS(0)
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > taskCount {
		concurrency = taskCount
	}
	return concurrency
}

func runTask(ctx context.Context, task Task, handler TaskHandler) Result {
	if handler == nil {
		return Result{
			InputPath: task.Label,
			Err:       fmt.Errorf("no handler for task type %s", task.Type),
		}
	}
	result := handler.Handle(ctx, task)
	if result.InputPath == "" {
		result.InputPath = task.Label
	}
	return result
}
