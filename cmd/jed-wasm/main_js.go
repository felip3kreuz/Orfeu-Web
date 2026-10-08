//go:build js && wasm

package main

import (
	"syscall/js"

	"jed-simulador/internal/wasmbridge"
)

var exported []js.Func

func expose(obj js.Value, name string, fn func([]js.Value) any) {
	wrapped := js.FuncOf(func(this js.Value, args []js.Value) any {
		return fn(args)
	})
	exported = append(exported, wrapped)
	obj.Set(name, wrapped)
}

func argString(args []js.Value, index int) string {
	if len(args) <= index {
		return ""
	}
	return args[index].String()
}

func argInt(args []js.Value, index int) int {
	if len(args) <= index {
		return 0
	}
	return args[index].Int()
}

func main() {
	svc := wasmbridge.NewService()
	api := js.Global().Get("Object").New()
	api.Set("version", wasmbridge.Version)

	expose(api, "info", func(args []js.Value) any { return svc.Info() })
	expose(api, "createSimulator", func(args []js.Value) any {
		return svc.CreateSimulator(argString(args, 0))
	})
	expose(api, "destroySimulator", func(args []js.Value) any {
		return svc.DestroySimulator(argInt(args, 0))
	})
	expose(api, "processWeek", func(args []js.Value) any {
		return svc.ProcessWeek(argInt(args, 0), argString(args, 1))
	})
	expose(api, "indicators", func(args []js.Value) any {
		return svc.Indicators(argString(args, 0))
	})
	expose(api, "score", func(args []js.Value) any {
		return svc.Score(argString(args, 0))
	})
	expose(api, "review", func(args []js.Value) any {
		return svc.Review(argString(args, 0))
	})
	expose(api, "placeInputOrder", func(args []js.Value) any {
		return svc.PlaceInputOrder(argInt(args, 0), argString(args, 1), argString(args, 2), argString(args, 3), argString(args, 4), argString(args, 5))
	})
	expose(api, "buyStock", func(args []js.Value) any {
		return svc.BuyStock(argString(args, 0), argString(args, 1), argString(args, 2))
	})
	expose(api, "journeyStep", func(args []js.Value) any {
		return svc.JourneyStep(argString(args, 0))
	})
	expose(api, "canvasExplanations", func(args []js.Value) any {
		return svc.CanvasExplanations(argString(args, 0))
	})
	expose(api, "digitalChannels", func(args []js.Value) any { return svc.DigitalChannels() })
	expose(api, "digitalTools", func(args []js.Value) any { return svc.DigitalTools() })

	js.Global().Set("JEDCore", api)
	select {}
}
