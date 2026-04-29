package glfw

import (
	"sync"
	"unsafe"

	base "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── Type aliases ──────────────────────────────────────────────────────────────
type (
	Window              = base.Window
	Monitor             = base.Monitor
	VidMode             = base.VidMode
	Image               = base.Image
	GamepadState        = base.GamepadState
	Cursor              = base.Cursor
	StandardCursorShape = base.StandardCursorShape
)

// ── Re-exported types for callback signatures ─────────────────────────────────
type (
	Key                = base.Key
	Action             = base.Action
	ModifierKey        = base.ModifierKey
	MouseButton        = base.MouseButton
	Joystick           = base.Joystick
	JoystickHatState   = base.JoystickHatState
	PeripheralEvent    = base.PeripheralEvent
	InputMode          = base.InputMode
	Hint               = base.Hint
	ClientAPI          = base.ClientAPI
	ContextCreationAPI = base.ContextCreationAPI
	OpenGLProfile      = base.OpenGLProfile
	ErrorCode          = base.ErrorCode
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	// Actions
	Release Action = base.Release
	Press   Action = base.Press
	Repeat  Action = base.Repeat

	// Modifier keys
	ModShift   ModifierKey = base.ModShift
	ModControl ModifierKey = base.ModControl
	ModAlt     ModifierKey = base.ModAlt
	ModSuper   ModifierKey = base.ModSuper

	// Mouse buttons
	MouseButtonLeft   MouseButton = base.MouseButtonLeft
	MouseButtonRight  MouseButton = base.MouseButtonRight
	MouseButtonMiddle MouseButton = base.MouseButtonMiddle
	MouseButton4      MouseButton = base.MouseButton4
	MouseButton5      MouseButton = base.MouseButton5
	MouseButton6      MouseButton = base.MouseButton6
	MouseButton7      MouseButton = base.MouseButton7
	MouseButton8      MouseButton = base.MouseButton8
	MouseButtonLast               = base.MouseButtonLast

	// Input modes
	CursorMode         InputMode = base.CursorMode
	StickyKeys         InputMode = base.StickyKeys
	StickyMouseButtons InputMode = base.StickyMouseButtons

	// Cursor modes
	CursorNormal   = base.CursorNormal
	CursorHidden   = base.CursorHidden
	CursorDisabled = base.CursorDisabled

	// Standard cursor shapes
	ArrowCursor     StandardCursorShape = base.ArrowCursor
	IBeamCursor     StandardCursorShape = base.IBeamCursor
	CrosshairCursor StandardCursorShape = base.CrosshairCursor
	HandCursor      StandardCursorShape = base.HandCursor
	HResizeCursor   StandardCursorShape = base.HResizeCursor
	VResizeCursor   StandardCursorShape = base.VResizeCursor

	// Window / context hints (v3.2 adds Maximized, ContextCreationAPIHint, etc.)
	Focused                 Hint = base.Focused
	Iconified               Hint = base.Iconified
	Resizable               Hint = base.Resizable
	Visible                 Hint = base.Visible
	Decorated               Hint = base.Decorated
	AutoIconify             Hint = base.AutoIconify
	Floating                Hint = base.Floating
	Maximized               Hint = base.Maximized
	RedBits                 Hint = base.RedBits
	GreenBits               Hint = base.GreenBits
	BlueBits                Hint = base.BlueBits
	AlphaBits               Hint = base.AlphaBits
	DepthBits               Hint = base.DepthBits
	StencilBits             Hint = base.StencilBits
	Stereo                  Hint = base.Stereo
	Samples                 Hint = base.Samples
	SRGBCapable             Hint = base.SRGBCapable
	RefreshRate             Hint = base.RefreshRate
	DoubleBuffer            Hint = base.DoubleBuffer
	ClientAPIs              Hint = base.ClientAPIs
	ContextVersionMajor     Hint = base.ContextVersionMajor
	ContextVersionMinor     Hint = base.ContextVersionMinor
	ContextRevision         Hint = base.ContextRevision
	ContextRobustness       Hint = base.ContextRobustness
	OpenGLForwardCompatible Hint = base.OpenGLForwardCompatible
	OpenGLDebugContext      Hint = base.OpenGLDebugContext
	OpenGLProfileHint       Hint = base.OpenGLProfileHint
	ContextCreationAPIHint  Hint = base.ContextCreationAPIHint

	// DontCare
	DontCare = -1

	// Client APIs
	OpenGLAPI   ClientAPI = base.OpenGLAPI
	OpenGLESAPI ClientAPI = base.OpenGLESAPI
	NoAPI       ClientAPI = base.NoAPI

	// OpenGL profiles
	AnyProfile           OpenGLProfile = base.AnyProfile
	CoreProfile          OpenGLProfile = base.CoreProfile
	CompatibilityProfile OpenGLProfile = base.CompatibilityProfile

	// Context creation APIs (added in 3.2)
	NativeContextAPI ContextCreationAPI = base.NativeContextAPI
	EGLContextAPI    ContextCreationAPI = base.EGLContextAPI

	// Peripheral events
	Connected    PeripheralEvent = base.Connected
	Disconnected PeripheralEvent = base.Disconnected

	// Joysticks
	Joystick1    Joystick = base.Joystick1
	Joystick2    Joystick = base.Joystick2
	Joystick3    Joystick = base.Joystick3
	Joystick4    Joystick = base.Joystick4
	Joystick5    Joystick = base.Joystick5
	Joystick6    Joystick = base.Joystick6
	Joystick7    Joystick = base.Joystick7
	Joystick8    Joystick = base.Joystick8
	Joystick9    Joystick = base.Joystick9
	Joystick10   Joystick = base.Joystick10
	Joystick11   Joystick = base.Joystick11
	Joystick12   Joystick = base.Joystick12
	Joystick13   Joystick = base.Joystick13
	Joystick14   Joystick = base.Joystick14
	Joystick15   Joystick = base.Joystick15
	Joystick16   Joystick = base.Joystick16
	JoystickLast           = base.JoystickLast

	// Hat states
	HatCentered  JoystickHatState = base.HatCentered
	HatUp        JoystickHatState = base.HatUp
	HatRight     JoystickHatState = base.HatRight
	HatDown      JoystickHatState = base.HatDown
	HatLeft      JoystickHatState = base.HatLeft
	HatRightUp                    = base.HatRightUp
	HatRightDown                  = base.HatRightDown
	HatLeftUp                     = base.HatLeftUp
	HatLeftDown                   = base.HatLeftDown

	// Error codes
	NotInitialized     ErrorCode = base.NotInitialized
	NoCurrentContext   ErrorCode = base.NoCurrentContext
	InvalidEnum        ErrorCode = base.InvalidEnum
	InvalidValue       ErrorCode = base.InvalidValue
	OutOfMemory        ErrorCode = base.OutOfMemory
	APIUnavailable     ErrorCode = base.APIUnavailable
	VersionUnavailable ErrorCode = base.VersionUnavailable
	PlatformError      ErrorCode = base.PlatformError
	FormatUnavailable  ErrorCode = base.FormatUnavailable

	// Keys
	KeyUnknown      Key = base.KeyUnknown
	KeySpace        Key = base.KeySpace
	KeyApostrophe   Key = base.KeyApostrophe
	KeyComma        Key = base.KeyComma
	KeyMinus        Key = base.KeyMinus
	KeyPeriod       Key = base.KeyPeriod
	KeySlash        Key = base.KeySlash
	Key0            Key = base.Key0
	Key1            Key = base.Key1
	Key2            Key = base.Key2
	Key3            Key = base.Key3
	Key4            Key = base.Key4
	Key5            Key = base.Key5
	Key6            Key = base.Key6
	Key7            Key = base.Key7
	Key8            Key = base.Key8
	Key9            Key = base.Key9
	KeySemicolon    Key = base.KeySemicolon
	KeyEqual        Key = base.KeyEqual
	KeyA            Key = base.KeyA
	KeyB            Key = base.KeyB
	KeyC            Key = base.KeyC
	KeyD            Key = base.KeyD
	KeyE            Key = base.KeyE
	KeyF            Key = base.KeyF
	KeyG            Key = base.KeyG
	KeyH            Key = base.KeyH
	KeyI            Key = base.KeyI
	KeyJ            Key = base.KeyJ
	KeyK            Key = base.KeyK
	KeyL            Key = base.KeyL
	KeyM            Key = base.KeyM
	KeyN            Key = base.KeyN
	KeyO            Key = base.KeyO
	KeyP            Key = base.KeyP
	KeyQ            Key = base.KeyQ
	KeyR            Key = base.KeyR
	KeyS            Key = base.KeyS
	KeyT            Key = base.KeyT
	KeyU            Key = base.KeyU
	KeyV            Key = base.KeyV
	KeyW            Key = base.KeyW
	KeyX            Key = base.KeyX
	KeyY            Key = base.KeyY
	KeyZ            Key = base.KeyZ
	KeyEscape       Key = base.KeyEscape
	KeyEnter        Key = base.KeyEnter
	KeyTab          Key = base.KeyTab
	KeyBackspace    Key = base.KeyBackspace
	KeyInsert       Key = base.KeyInsert
	KeyDelete       Key = base.KeyDelete
	KeyRight        Key = base.KeyRight
	KeyLeft         Key = base.KeyLeft
	KeyDown         Key = base.KeyDown
	KeyUp           Key = base.KeyUp
	KeyPageUp       Key = base.KeyPageUp
	KeyPageDown     Key = base.KeyPageDown
	KeyHome         Key = base.KeyHome
	KeyEnd          Key = base.KeyEnd
	KeyCapsLock     Key = base.KeyCapsLock
	KeyScrollLock   Key = base.KeyScrollLock
	KeyNumLock      Key = base.KeyNumLock
	KeyPrintScreen  Key = base.KeyPrintScreen
	KeyPause        Key = base.KeyPause
	KeyF1           Key = base.KeyF1
	KeyF2           Key = base.KeyF2
	KeyF3           Key = base.KeyF3
	KeyF4           Key = base.KeyF4
	KeyF5           Key = base.KeyF5
	KeyF6           Key = base.KeyF6
	KeyF7           Key = base.KeyF7
	KeyF8           Key = base.KeyF8
	KeyF9           Key = base.KeyF9
	KeyF10          Key = base.KeyF10
	KeyF11          Key = base.KeyF11
	KeyF12          Key = base.KeyF12
	KeyF13          Key = base.KeyF13
	KeyF14          Key = base.KeyF14
	KeyF15          Key = base.KeyF15
	KeyF16          Key = base.KeyF16
	KeyF17          Key = base.KeyF17
	KeyF18          Key = base.KeyF18
	KeyF19          Key = base.KeyF19
	KeyF20          Key = base.KeyF20
	KeyF21          Key = base.KeyF21
	KeyF22          Key = base.KeyF22
	KeyF23          Key = base.KeyF23
	KeyF24          Key = base.KeyF24
	KeyF25          Key = base.KeyF25
	KeyLeftShift    Key = base.KeyLeftShift
	KeyLeftControl  Key = base.KeyLeftControl
	KeyLeftAlt      Key = base.KeyLeftAlt
	KeyLeftSuper    Key = base.KeyLeftSuper
	KeyRightShift   Key = base.KeyRightShift
	KeyRightControl Key = base.KeyRightControl
	KeyRightAlt     Key = base.KeyRightAlt
	KeyRightSuper   Key = base.KeyRightSuper
	KeyMenu         Key = base.KeyMenu
	KeyKP0          Key = base.KeyKP0
	KeyKP1          Key = base.KeyKP1
	KeyKP2          Key = base.KeyKP2
	KeyKP3          Key = base.KeyKP3
	KeyKP4          Key = base.KeyKP4
	KeyKP5          Key = base.KeyKP5
	KeyKP6          Key = base.KeyKP6
	KeyKP7          Key = base.KeyKP7
	KeyKP8          Key = base.KeyKP8
	KeyKP9          Key = base.KeyKP9
	KeyKPDecimal    Key = base.KeyKPDecimal
	KeyKPDivide     Key = base.KeyKPDivide
	KeyKPMultiply   Key = base.KeyKPMultiply
	KeyKPSubtract   Key = base.KeyKPSubtract
	KeyKPAdd        Key = base.KeyKPAdd
	KeyKPEnter      Key = base.KeyKPEnter
	KeyKPEqual      Key = base.KeyKPEqual
	KeyGraveAccent  Key = base.KeyGraveAccent
	KeyLeftBracket  Key = base.KeyLeftBracket
	KeyBackslash    Key = base.KeyBackslash
	KeyRightBracket Key = base.KeyRightBracket
	KeyWorld1       Key = base.KeyWorld1
	KeyWorld2       Key = base.KeyWorld2
	KeyLast                = base.KeyLast
)

// ── Window user pointer (from v3.1) ───────────────────────────────────────────
var (
	userPtrMu sync.RWMutex
	userPtrs  = make(map[*Window]unsafe.Pointer)
)

// SetWindowUserPointer stores an arbitrary pointer associated with the window.
func SetWindowUserPointer(w *Window, ptr unsafe.Pointer) {
	userPtrMu.Lock()
	userPtrs[w] = ptr
	userPtrMu.Unlock()
}

// GetWindowUserPointer retrieves the pointer previously set by SetWindowUserPointer.
func GetWindowUserPointer(w *Window) unsafe.Pointer {
	userPtrMu.RLock()
	p := userPtrs[w]
	userPtrMu.RUnlock()
	return p
}

// GetWindowFrameSize returns the size of each edge of the frame around the
// window's client area. This is a stub returning zeros.
func GetWindowFrameSize(w *Window) (left, top, right, bottom int) {
	return 0, 0, 0, 0
}

// ── Functions ─────────────────────────────────────────────────────────────────

func Init() error       { return base.Init() }
func Terminate()        { base.Terminate() }
func GetTime() float64  { return base.GetTime() }
func SetTime(t float64) { base.SetTime(t) }

func CreateWindow(width, height int, title string, monitor, share *Monitor) (*Window, error) {
	return base.CreateWindow(width, height, title, monitor, share)
}

func PollEvents()                          { base.PollEvents() }
func WaitEvents()                          { base.WaitEvents() }
func WaitEventsTimeout(timeout float64)    { base.WaitEventsTimeout(timeout) }
func PostEmptyEvent()                      { base.PostEmptyEvent() }

func WindowHint(target Hint, hint int) { base.WindowHint(target, hint) }
func DefaultWindowHints()              { base.DefaultWindowHints() }

func GetMonitors() ([]*Monitor, error)  { return base.GetMonitors() }
func GetPrimaryMonitor() *Monitor       { return base.GetPrimaryMonitor() }
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {
	base.SetMonitorCallback(cb)
}

func GetClipboardString() string  { return base.GetClipboardString() }
func SetClipboardString(s string) { base.SetClipboardString(s) }

func JoystickPresent(joy Joystick) bool              { return base.JoystickPresent(joy) }
func GetJoystickAxes(joy Joystick) []float32          { return base.GetJoystickAxes(joy) }
func GetJoystickButtons(joy Joystick) []Action        { return base.GetJoystickButtons(joy) }
func GetJoystickHats(joy Joystick) []JoystickHatState { return base.GetJoystickHats(joy) }
func GetJoystickName(joy Joystick) string             { return base.GetJoystickName(joy) }
func GetJoystickGUID(joy Joystick) string             { return base.GetJoystickGUID(joy) }
func SetJoystickCallback(cb func(joy Joystick, event PeripheralEvent)) {
	base.SetJoystickCallback(cb)
}

func CreateCursor(image *Image, xhot, yhot int) (*Cursor, error) {
	return base.CreateCursor(image, xhot, yhot)
}
func CreateStandardCursor(shape StandardCursorShape) (*Cursor, error) {
	return base.CreateStandardCursor(shape)
}
func DestroyCursor(cursor *Cursor) { base.DestroyCursor(cursor) }

func GetKeyName(key Key, scancode int) string { return base.GetKeyName(key, scancode) }
func GetKeyScancode(key Key) int              { return base.GetKeyScancode(key) }

func GetTimerFrequency() uint64 { return base.GetTimerFrequency() }
func GetTimerValue() uint64     { return base.GetTimerValue() }

// ── Context management ────────────────────────────────────────────────────────

func SwapInterval(interval int)                 { base.SwapInterval(interval) }
func GetProcAddress(name string) unsafe.Pointer { return base.GetProcAddress(name) }
func ExtensionSupported(extension string) bool  { return base.ExtensionSupported(extension) }
func GetCurrentContext() *Window                { return base.GetCurrentContext() }
func DetachCurrentContext()                     { base.DetachCurrentContext() }
