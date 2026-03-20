package main

import "github.com/FlipsideCrypto/near-intents-cli/internal/output"

// Type aliases so existing code that references Envelope/ErrorInfo still compiles.
type Envelope = output.Envelope
type ErrorInfo = output.ErrorInfo

var prettyOutput bool
var exitCode int

func PrintSuccess(data interface{}) {
	output.PrettyOutput = prettyOutput
	output.PrintSuccess(data)
}

func PrintErrorResponse(code, message string) {
	output.PrettyOutput = prettyOutput
	output.PrintErrorResponse(code, message)
	exitCode = output.ExitCode
}
