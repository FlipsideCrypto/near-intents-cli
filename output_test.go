package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintSuccess(t *testing.T) {
	exitCode = 0
	out := captureStdout(func() {
		PrintSuccess(map[string]string{"key": "value"})
	})
	var env Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !env.Success {
		t.Error("expected success=true")
	}
	if env.Error != nil {
		t.Error("expected error=nil")
	}
}

func TestPrintErrorOutput(t *testing.T) {
	exitCode = 0
	out := captureStdout(func() {
		PrintErrorResponse("TOKEN_NOT_FOUND", "USDC not found on chain solana")
	})
	var env Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Success {
		t.Error("expected success=false")
	}
	if env.Error == nil || env.Error.Code != "TOKEN_NOT_FOUND" {
		t.Error("expected error code TOKEN_NOT_FOUND")
	}
	if exitCode != 1 {
		t.Error("expected exitCode=1")
	}
}
