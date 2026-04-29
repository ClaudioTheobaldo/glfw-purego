//go:build windows

// 24_joystick polls an XInput gamepad each frame and displays axes, buttons,
// hats and gamepad state to stdout.  Press ESC to quit.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 24_joystick.exe .
package main

import (
	"fmt"
	"log"
	"strings"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

const joy = glfw.Joystick(0) // Joystick1 / first XInput slot

var buttonNames = []string{"A", "B", "X", "Y", "LB", "RB", "Back", "Start", "L3", "R3"}

func hatName(state glfw.JoystickHatState) string {
	if state == glfw.HatCentered {
		return "CENTER"
	}
	var parts []string
	if state&glfw.HatUp != 0 {
		parts = append(parts, "UP")
	}
	if state&glfw.HatDown != 0 {
		parts = append(parts, "DOWN")
	}
	if state&glfw.HatLeft != 0 {
		parts = append(parts, "LEFT")
	}
	if state&glfw.HatRight != 0 {
		parts = append(parts, "RIGHT")
	}
	return strings.Join(parts, "+")
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(800, 600, "Joystick — glfw-purego", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	win.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	fw, fh := win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(fw), int32(fh))

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	fmt.Println("Plug in an XInput gamepad (e.g. Xbox controller). Press ESC to quit.")
	fmt.Println()

	for !win.ShouldClose() {
		gl.ClearColor(0.05, 0.05, 0.15, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		if !glfw.JoystickPresent(joy) {
			win.SetTitle("Joystick | No controller detected (plug in XInput gamepad)")
		} else {
			name := glfw.GetJoystickName(joy)
			win.SetTitle("Joystick | " + name)

			// Build axis string
			axes := glfw.GetJoystickAxes(joy)
			axisLabels := []string{"LX", "LY", "RX", "RY", "LT", "RT"}
			var axisBuf strings.Builder
			axisBuf.WriteString("Axes:")
			for i, label := range axisLabels {
				val := float32(0)
				if i < len(axes) {
					val = axes[i]
				}
				fmt.Fprintf(&axisBuf, " %s:%+.2f", label, val)
			}
			axisStr := axisBuf.String()

			// Build button string
			buttons := glfw.GetJoystickButtons(joy)
			var pressedBtns []string
			for i, name := range buttonNames {
				if i < len(buttons) && buttons[i] == glfw.Press {
					pressedBtns = append(pressedBtns, name)
				}
			}
			btnStr := "Btns: " + strings.Join(pressedBtns, " ")
			if len(pressedBtns) == 0 {
				btnStr = "Btns: (none)"
			}

			// Build hat string
			hats := glfw.GetJoystickHats(joy)
			hatStr := "DPad:"
			if len(hats) > 0 {
				hatStr += " " + hatName(hats[0])
			} else {
				hatStr += " -"
			}

			// Gamepad state
			gpStr := ""
			if glfw.JoystickIsGamepad(joy) {
				var gs glfw.GamepadState
				if glfw.GetGamepadState(joy, &gs) {
					gpStr = fmt.Sprintf(" | Gamepad OK (LeftX:%+.2f LeftY:%+.2f)",
						gs.Axes[glfw.AxisLeftX], gs.Axes[glfw.AxisLeftY])
				}
			}

			line := fmt.Sprintf("%s | %s | %s%s", axisStr, btnStr, hatStr, gpStr)
			fmt.Printf("\r%-120s", line)
		}

		win.SwapBuffers()
		glfw.PollEvents()
	}
	fmt.Println()
}
