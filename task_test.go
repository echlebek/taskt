package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestMSDurationMarshalJSON(t *testing.T) {
	tests := []struct {
		Input MSDuration
		Want  string
	}{
		{
			Input: MSDuration(3500 * time.Microsecond),
			Want:  "3.5",
		},
		{
			Input: MSDuration(time.Hour),
			Want:  "3600000",
		},
		{
			Input: MSDuration(time.Microsecond),
			Want:  "0.001",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			b, err := json.Marshal(test.Input)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(b), test.Want; got != want {
				t.Errorf("bad MSDuration.MarshalJSON: got %q, want %q", got, want)
			}
		})
	}
}

func TestMSDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		Input    []byte
		Want     MSDuration
		ExpError bool
	}{
		{
			Input: []byte("3.5"),
			Want:  MSDuration(3500 * time.Microsecond),
		},
		{
			Input: []byte("3600000"),
			Want:  MSDuration(time.Hour),
		},
		{
			Input: []byte("0.001"),
			Want:  MSDuration(time.Microsecond),
		},
		{
			Input:    []byte("3 seconds"),
			ExpError: true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			var d MSDuration
			err := json.Unmarshal(test.Input, &d)
			if err != nil && !test.ExpError {
				t.Fatal(err)
			} else if err == nil && test.ExpError {
				t.Error("expected non-nil error")
			}
			if got, want := d, test.Want; got != want {
				t.Errorf("bad MSDuration.UnmarshalJSON: got %d, want %d", got, want)
			}
		})
	}
}
