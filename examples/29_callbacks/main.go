// 29_callbacks demonstrates that SetXxxCallback returns the previous callback,
// enabling callback chaining — a common pattern for middleware/logging layers.
//
// A "logger" callback is installed first.  A "handler" callback is installed
// second; it receives the previous (logger) callback and calls it after its
// own work, so both fire on every key press.
//
// Controls:
//
//	T — toggle the logger layer on/off (re-installs callbacks)
//	Any other key — handled by the chained pair
//	ESC — quit
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 29_callbacks.exe .
package main

import (
	"fmt"
	"log"

	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

type keyCB = func(*glfw.Window, glfw.Key, int, glfw.Action, glfw.ModifierKey)
type sizeCB = func(*glfw.Window, int, int)

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)

	win, err := glfw.CreateWindow(800, 600, "Callback Chaining — glfw-purego", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	pressCount := 0
	loggerOn := true

	// handlerCB is the primary key handler.
	handlerCB := keyCB(func(w *glfw.Window, key glfw.Key, sc int, action glfw.Action, mods glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		pressCount++
		name := glfw.GetKeyName(key, sc)
		if name == "" {
			name = "?"
		}
		fmt.Printf("[HANDLER #%d] key=%v scancode=%d name=%q mods=%v\n", pressCount, key, sc, name, mods)
	})

	// installCallbacks chains logger → handler using the returned previous value.
	var installCallbacks func()
	installCallbacks = func() {
		// Start clean.
		win.SetKeyCallback(nil)

		// Install base handler first.
		win.SetKeyCallback(handlerCB)

		if loggerOn {
			// var-declared so the closure can reference it safely.
			var prev keyCB
			prev = win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, sc int, action glfw.Action, mods glfw.ModifierKey) {
				if key == glfw.KeyEscape && action == glfw.Press {
					w.SetShouldClose(true)
					return
				}
				if key == glfw.KeyT && action == glfw.Press {
					loggerOn = !loggerOn
					fmt.Printf("Logger %s — reinstalling callbacks\n", map[bool]string{true: "ON", false: "OFF"}[loggerOn])
					installCallbacks()
					return
				}
				// Logger layer runs first.
				if action == glfw.Press {
					fmt.Printf("[LOGGER     ] key=%v action=Press\n", key)
				}
				// Chain to the previous (handler) callback.
				if prev != nil {
					prev(w, key, sc, action, mods)
				}
			})
		} else {
			// No logger: just the bare handler + ESC/T handling.
			win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, sc int, action glfw.Action, mods glfw.ModifierKey) {
				if key == glfw.KeyEscape && action == glfw.Press {
					w.SetShouldClose(true)
					return
				}
				if key == glfw.KeyT && action == glfw.Press {
					loggerOn = !loggerOn
					fmt.Printf("Logger %s — reinstalling callbacks\n", map[bool]string{true: "ON", false: "OFF"}[loggerOn])
					installCallbacks()
					return
				}
				handlerCB(w, key, sc, action, mods)
			})
		}

		// ── Size callback chaining (independent of T toggle) ──────────────────
		// layer1 is installed first (becomes "prev" for layer2).
		var prevSize sizeCB
		prevSize = win.SetSizeCallback(func(w *glfw.Window, width, height int) {
			fmt.Printf("[SIZE layer1] %dx%d — forwarding…\n", width, height)
			if prevSize != nil {
				prevSize(w, width, height)
			}
		})
		// layer2 is installed second; it chains back to layer1.
		win.SetSizeCallback(func(w *glfw.Window, width, height int) {
			fmt.Printf("[SIZE layer2] %dx%d\n", width, height)
			if prevSize != nil {
				prevSize(w, width, height)
			}
		})

		// ── Cursor-position callback ───────────────────────────────────────────
		win.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
			w.SetTitle(fmt.Sprintf("Callback Chaining | %d presses | cursor=(%.0f,%.0f) | T=toggle logger",
				pressCount, x, y))
		})

		// ── Focus callbacks ────────────────────────────────────────────────────
		win.SetFocusCallback(func(w *glfw.Window, focused bool) {
			if focused {
				fmt.Println("[FOCUS] window gained focus")
			} else {
				fmt.Println("[FOCUS] window lost focus")
			}
		})
	}

	installCallbacks()

	fmt.Println("Controls: T = toggle logger layer, ESC = quit")
	fmt.Println("Resize window to see chained size callbacks.")
	fmt.Println("Each key press fires: [LOGGER] then [HANDLER] — in that order.")

	for !win.ShouldClose() {
		win.SwapBuffers()
		glfw.PollEvents()
	}
}
