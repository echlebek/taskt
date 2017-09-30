package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"testing"
)

type Test struct {
	Input string
	Exp   *TaskResult
}

func init() {
	flag.Parse()
}

func TestTaskServer(t *testing.T) {
	tests := []Test{
		{
			Input: `{"command": ["echo", "hello, world!"], "timeout": 1000}`,
			Exp: &TaskResult{
				Command: []string{"ls"},
				Output:  "hello, world!\n",
			},
		},
		{
			Input: `{"command": ["asdf"]}`,
			Exp: &TaskResult{
				Error:    `exec: "asdf": executable file not found in $PATH`,
				ExitCode: -1,
			},
		},
		{
			Input: `{"command": ["cat", "doesnotexist"]}`,
			Exp: &TaskResult{
				ExitCode: -1,
				Error:    "exit status 1",
			},
		},
		{
			Input: `{"command": ["sleep", "5"], "timeout": 1}`,
			Exp: &TaskResult{
				ExitCode: -1,
				Error:    ErrTimeoutExceeded.Error(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			doTest(t, test)
		})
	}
}

func newServer(t *testing.T) *TaskServer {
	ln, err := net.Listen("tcp", fmt.Sprintf(":0"))
	if err != nil {
		t.Fatal(err)
	}
	server := NewTaskServer(ln)
	return server
}

func doTest(t *testing.T, test Test) {
	server := newServer(t)
	defer server.Close()
	addr := server.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, err = fmt.Fprintln(conn, test.Input)
	if err != nil {
		t.Fatal(err)
	}
	var result TaskResult
	if err := json.NewDecoder(conn).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if got, want := result.ExitCode, test.Exp.ExitCode; got != want {
		t.Errorf("bad exit code: got %d, want %d", got, want)
	}
	if got, want := result.Output, test.Exp.Output; got != want {
		t.Errorf("bad output: got %q, want %q", got, want)
	}
	if got, want := result.Error, test.Exp.Error; got != want {
		t.Errorf("bad error: got %q, want %q", got, want)
	}
}

func TestTaskServerConcurrent(t *testing.T) {
	server := newServer(t)
	defer server.Close()
	addr := server.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = fmt.Fprintln(conn, `{"command": ["sleep", "5"], "timeout": 1}`); err != nil {
		t.Fatal(err)
	}
	if _, err = fmt.Fprintln(conn, `{"command": ["echo", "hello?"]}`); err != nil {
		t.Fatal(err)
	}
	var result TaskResult
	if err := json.NewDecoder(conn).Decode(&result); err != nil {
		t.Fatal(err)
	}
	want := TaskResult{
		ExitCode: -1,
		Error:    ErrConcurrentExecution.Error(),
	}
	if got, want := result.ExitCode, want.ExitCode; got != want {
		t.Errorf("bad exit code: got %d, want %d", got, want)
	}
	if got, want := result.Error, want.Error; got != want {
		t.Errorf("bad error: got %q, want %q", got, want)
	}
}
