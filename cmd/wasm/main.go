// Command wasm is the WebAssembly entry point for epubverify.
// It exposes a validateEPUB(Uint8Array) function to JavaScript
// that validates EPUB bytes and returns a JSON report.
package main

import (
	"bytes"
	"encoding/json"
	"syscall/js"

	"github.com/adammathes/epubverify/pkg/report"
	"github.com/adammathes/epubverify/pkg/validate"
)

func main() {
	js.Global().Set("validateEPUB", js.FuncOf(validateEPUB))

	// Block forever so the WASM module stays alive.
	select {}
}

// validateEPUB is called from JavaScript: validateEPUB(uint8Array) -> Promise<string>
// It accepts a Uint8Array of EPUB bytes and returns a JSON string with the report.
func validateEPUB(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return errorJSON("no data provided")
	}

	// Copy bytes from JS Uint8Array into Go slice.
	jsArray := args[0]
	length := jsArray.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, jsArray)

	// Run validation.
	r, err := validate.ValidateBytes(data)
	if err != nil {
		return errorJSON("validation failed: " + err.Error())
	}

	// Serialize the report to JSON.
	out := report.JSONOutput{
		Valid:        r.IsValid(),
		Messages:     r.Messages,
		FatalCount:   r.FatalCount(),
		ErrorCount:   r.ErrorCount(),
		WarningCount: r.WarningCount(),
	}
	if out.Messages == nil {
		out.Messages = []report.Message{}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return errorJSON("json encoding failed: " + err.Error())
	}

	return buf.String()
}

func errorJSON(msg string) string {
	out := map[string]interface{}{
		"valid":         false,
		"error":         msg,
		"messages":      []interface{}{},
		"fatal_count":   1,
		"error_count":   0,
		"warning_count": 0,
	}
	b, _ := json.Marshal(out)
	return string(b)
}
