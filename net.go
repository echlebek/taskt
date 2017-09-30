package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sync"
	"time"
)

// TaskServer reads TaskRequests, dispatches tasks, and writes TaskResults.
type TaskServer struct {
	net.Listener
	pool chan struct{}
	done chan struct{}
}

// NewTaskServer creates a new TaskServer that listens on ln for TaskRequests.
func NewTaskServer(ln net.Listener) *TaskServer {
	t := &TaskServer{
		Listener: ln,
		pool:     make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	t.unlock()
	go t.run()
	log.Printf("listening on %v", ln.Addr())
	return t
}

func (t *TaskServer) lock() error {
	select {
	case <-t.pool:
		return nil
	default:
		return ErrConcurrentExecution
	}
}

func (t *TaskServer) unlock() {
	select {
	case t.pool <- struct{}{}:
	default:
		panic("double unlock")
	}
}

func (t *TaskServer) run() {
	for {
		select {
		case <-t.done:
			log.Println("shutting down")
			return
		default:
		}
		conn, err := t.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go t.serve(conn)
	}
}

func (t *TaskServer) serve(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println(err)
		}
	}()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		go func() {
			b := scanner.Bytes()
			var task TaskRequest
			var result *TaskResult
			if err := json.Unmarshal(b, &task); err != nil {
				result = &TaskResult{
					ExitCode: -1,
					Error:    err.Error(),
				}
			} else {
				result = t.runTask(conn, &task)
			}
			m, _ := json.Marshal(result)
			if _, err := fmt.Fprintln(conn, string(m)); err != nil {
				log.Println(err)
			}
		}()
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

func (t *TaskServer) runTask(conn net.Conn, task *TaskRequest) *TaskResult {
	if err := t.lock(); err != nil {
		return &TaskResult{
			Error:    err.Error(),
			ExitCode: -1,
		}
	}
	defer t.unlock()
	if len(task.Command) == 0 {
		return &TaskResult{
			Error:    "no command specified",
			ExitCode: -1,
		}
	}
	ctx := context.Background()
	var cancel context.CancelFunc
	now := time.Now()
	if task.Timeout > 0 {
		deadline := now.Add(time.Second * time.Duration(task.Timeout))
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, task.Command[0], task.Command[1:]...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	status := 0
	err := cmd.Start()
	if err != nil {
		return &TaskResult{
			Error:    err.Error(),
			ExitCode: -1,
		}
	}
	done := make(chan struct{})
	var ctxerr error
	go func() {
		err = cmd.Wait()
		done <- struct{}{}
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		select {
		case <-done:
		case <-ctx.Done():
			cmd.Process.Kill()
			ctxerr = ctx.Err()
		}
		wg.Done()
	}()
	wg.Wait()
	if ctxerr != nil {
		return &TaskResult{
			Error:    ErrTimeoutExceeded.Error(),
			ExitCode: -1,
		}
	}
	if err != nil {
		return &TaskResult{
			Error:    err.Error(),
			ExitCode: -1,
		}
	}
	output := stdout.String()
	outerr := stderr.String()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return &TaskResult{
				Error:    err.Error(),
				ExitCode: -1,
			}
		}
		status = -1
		output = ""
	}
	return &TaskResult{
		Command:    task.Command,
		ExecutedAt: now.Unix(),
		DurationMS: MSDuration(time.Now().Sub(now) / time.Millisecond),
		ExitCode:   status,
		Output:     output,
		Error:      outerr,
	}
}

func (t *TaskServer) Close() {
	close(t.done)
}
