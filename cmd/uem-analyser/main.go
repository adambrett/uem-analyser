//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/adambrett/uem-analyser/internal/analyser"
)

var callbacks []js.Func

type inspectResponse struct {
	OK         bool                 `json:"ok"`
	Error      string               `json:"error,omitempty"`
	Inspection *analyser.Inspection `json:"inspection,omitempty"`
}

func main() {
	register("uemInspectFiles", inspectFiles)
	register("uemGenerateFiles", generateFiles)

	select {}
}

func register(name string, fn func(js.Value, []js.Value) any) {
	callback := js.FuncOf(fn)
	callbacks = append(callbacks, callback)
	js.Global().Set(name, callback)
}

func inspectFiles(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return marshalInspectResponse(inspectResponse{OK: false, Error: "No files were provided."})
	}

	files, err := inputFiles(args[0])
	if err != nil {
		return marshalInspectResponse(inspectResponse{OK: false, Error: err.Error()})
	}

	inspection, err := analyser.Inspect(files)
	if err != nil {
		return marshalInspectResponse(inspectResponse{OK: false, Error: err.Error()})
	}

	return marshalInspectResponse(inspectResponse{OK: true, Inspection: &inspection})
}

func generateFiles(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return objectError("Please choose files and select the VAS questions to include.")
	}

	files, err := inputFiles(args[0])
	if err != nil {
		return objectError(err.Error())
	}

	selected := selectedVAS(args[1])
	download, err := analyser.Generate(files, selected)
	if err != nil {
		return objectError(err.Error())
	}

	result := js.Global().Get("Object").New()
	result.Set("ok", true)
	result.Set("name", download.Name)
	result.Set("mimeType", download.MIMEType)

	data := js.Global().Get("Uint8Array").New(len(download.Data))
	js.CopyBytesToJS(data, download.Data)
	result.Set("data", data)

	return result
}

func inputFiles(value js.Value) ([]analyser.InputFile, error) {
	length := value.Get("length").Int()
	files := make([]analyser.InputFile, 0, length)

	for index := 0; index < length; index++ {
		item := value.Index(index)
		dataValue := item.Get("data")
		data := bytesFromJS(dataValue)

		files = append(files, analyser.InputFile{
			Name: item.Get("name").String(),
			Data: data,
		})
	}

	return files, nil
}

func bytesFromJS(value js.Value) []byte {
	byteLength := value.Get("byteLength")
	if !byteLength.IsUndefined() {
		data := make([]byte, byteLength.Int())
		js.CopyBytesToGo(data, value)

		return data
	}

	length := value.Get("length").Int()
	data := make([]byte, length)
	for index := 0; index < length; index++ {
		data[index] = byte(value.Index(index).Int())
	}

	return data
}

func selectedVAS(value js.Value) []string {
	length := value.Get("length").Int()
	selected := make([]string, 0, length)
	for index := 0; index < length; index++ {
		selected = append(selected, value.Index(index).String())
	}

	return selected
}

func marshalInspectResponse(response inspectResponse) string {
	data, err := json.Marshal(response)
	if err != nil {
		return `{"ok":false,"error":"Unable to encode response."}`
	}

	return string(data)
}

func objectError(message string) js.Value {
	result := js.Global().Get("Object").New()
	result.Set("ok", false)
	result.Set("error", message)

	return result
}
