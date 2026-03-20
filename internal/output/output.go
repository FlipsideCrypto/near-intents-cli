package output

import (
	"encoding/json"
	"fmt"
	"os"
)

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var PrettyOutput bool
var ExitCode int

func PrintSuccess(data interface{}) {
	env := Envelope{Success: true, Data: data}
	printEnvelope(env)
}

func PrintErrorResponse(code, message string) {
	env := Envelope{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message},
	}
	ExitCode = 1
	printEnvelope(env)
}

func printEnvelope(env Envelope) {
	var out []byte
	var err error
	if PrettyOutput {
		out, err = json.MarshalIndent(env, "", "  ")
	} else {
		out, err = json.Marshal(env)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
