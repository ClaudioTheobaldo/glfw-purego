package glfw

// Key represents a keyboard key.
type Key int

// Action represents a key or mouse button action.
type Action int

// ModifierKey is a bitmask of modifier keys held during an input event.
type ModifierKey int

// MouseButton represents a mouse button.
type MouseButton int

// InputMode is an input configuration mode settable on a window.
type InputMode int

// cursorModeValue is an internal alias for the cursor mode value type.
// The public constants CursorNormal, CursorHidden, CursorDisabled are untyped ints.

// Hint is a window or context creation hint.
type Hint int

// OpenGLProfile specifies the OpenGL context profile.
type OpenGLProfile int

// ClientAPI specifies the client API to create a context for.
type ClientAPI int

// ContextCreationAPI specifies which API to use for context creation.
type ContextCreationAPI int

// PeripheralEvent is emitted when a monitor or joystick is connected/disconnected.
type PeripheralEvent int

// Joystick represents a joystick or gamepad device slot.
type Joystick int

// JoystickHatState is the state of a joystick hat (d-pad).
type JoystickHatState int

// GamepadAxis identifies a gamepad axis.
type GamepadAxis int

// GamepadButton identifies a gamepad button.
type GamepadButton int

// VidMode describes a monitor's video mode.
type VidMode struct {
	Width, Height                   int
	RedBits, GreenBits, BlueBits    int
	RefreshRate                     int
}

// GammaRamp represents the gamma ramp for a monitor.
type GammaRamp struct {
	Red, Green, Blue []uint16
}

// GamepadState holds the axis and button state for a connected gamepad.
type GamepadState struct {
	Buttons [15]Action
	Axes    [6]float32
}

// Image is an RGBA image used for window icons and custom cursors.
type Image struct {
	Width, Height int
	Pixels        []uint8
}

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

const (
	// Actions
	Release Action = 0
	Press   Action = 1
	Repeat  Action = 2

	// Modifier keys
	ModShift   ModifierKey = 0x0001
	ModControl ModifierKey = 0x0002
	ModAlt     ModifierKey = 0x0004
	ModSuper   ModifierKey = 0x0008

	// Mouse buttons
	MouseButtonLeft   MouseButton = 0
	MouseButtonRight  MouseButton = 1
	MouseButtonMiddle MouseButton = 2
	MouseButton4      MouseButton = 3
	MouseButton5      MouseButton = 4
	MouseButton6      MouseButton = 5
	MouseButton7      MouseButton = 6
	MouseButton8      MouseButton = 7
	MouseButtonLast               = MouseButton8

	// Input modes — the selector passed to SetInputMode / GetInputMode.
	CursorMode         InputMode = 0x00033001
	StickyKeys         InputMode = 0x00033002
	StickyMouseButtons InputMode = 0x00033003
	LockKeyMods        InputMode = 0x00033004
	RawMouseMotion     InputMode = 0x00033005

	// Cursor modes (values passed as the second argument to SetInputMode(CursorMode, ...))
	CursorNormal   = 0x00034001
	CursorHidden   = 0x00034002
	CursorDisabled = 0x00034003

	// Window / context hints
	Focused                Hint = 0x00020001
	Iconified              Hint = 0x00020002
	Resizable              Hint = 0x00020003
	Visible                Hint = 0x00020004
	Decorated              Hint = 0x00020005
	AutoIconify            Hint = 0x00020006
	Floating               Hint = 0x00020007
	Maximized              Hint = 0x00020008
	CenterCursor           Hint = 0x00020009
	TransparentFramebuffer Hint = 0x0002000A
	Hovered                Hint = 0x0002000B
	FocusOnShow            Hint = 0x0002000C
	RedBits                Hint = 0x00021001
	GreenBits              Hint = 0x00021002
	BlueBits               Hint = 0x00021003
	AlphaBits              Hint = 0x00021004
	DepthBits              Hint = 0x00021005
	StencilBits            Hint = 0x00021006
	Stereo                 Hint = 0x0002100C
	Samples                Hint = 0x0002100D
	SRGBCapable            Hint = 0x0002100E
	RefreshRate            Hint = 0x0002100F
	DoubleBuffer           Hint = 0x00021010
	ClientAPIs             Hint = 0x00022001 // selects the client API; value is a ClientAPI constant
	ContextVersionMajor    Hint = 0x00022002
	ContextVersionMinor    Hint = 0x00022003
	ContextRevision        Hint = 0x00022004
	ContextRobustness      Hint = 0x00022005
	OpenGLForwardCompatible Hint = 0x00022006
	OpenGLDebugContext     Hint = 0x00022007
	OpenGLProfileHint      Hint = 0x00022008
	ContextReleaseBehavior Hint = 0x00022009
	ContextNoError         Hint = 0x0002200A
	ContextCreationAPIHint Hint = 0x0002200B
	ScaleToMonitor         Hint = 0x0002200C

	// Client APIs
	OpenGLAPI   ClientAPI = 0x00030001
	OpenGLESAPI ClientAPI = 0x00030002
	NoAPI       ClientAPI = 0

	// OpenGL profiles
	AnyProfile            OpenGLProfile = 0
	CoreProfile           OpenGLProfile = 0x00032001
	CompatibilityProfile  OpenGLProfile = 0x00032002

	// Context creation APIs
	NativeContextAPI  ContextCreationAPI = 0x00036001
	EGLContextAPI     ContextCreationAPI = 0x00036002
	OSMesaContextAPI  ContextCreationAPI = 0x00036003

	// Peripheral events
	Connected    PeripheralEvent = 0x00040001
	Disconnected PeripheralEvent = 0x00040002

	// Joysticks
	Joystick1    Joystick = 0
	Joystick2    Joystick = 1
	Joystick3    Joystick = 2
	Joystick4    Joystick = 3
	Joystick5    Joystick = 4
	Joystick6    Joystick = 5
	Joystick7    Joystick = 6
	Joystick8    Joystick = 7
	Joystick9    Joystick = 8
	Joystick10   Joystick = 9
	Joystick11   Joystick = 10
	Joystick12   Joystick = 11
	Joystick13   Joystick = 12
	Joystick14   Joystick = 13
	Joystick15   Joystick = 14
	Joystick16   Joystick = 15
	JoystickLast           = Joystick16

	// Hat states
	HatCentered  JoystickHatState = 0
	HatUp        JoystickHatState = 1
	HatRight     JoystickHatState = 2
	HatDown      JoystickHatState = 4
	HatLeft      JoystickHatState = 8
	HatRightUp              = HatRight | HatUp
	HatRightDown            = HatRight | HatDown
	HatLeftUp               = HatLeft | HatUp
	HatLeftDown             = HatLeft | HatDown

	// Gamepad axes
	AxisLeftX        GamepadAxis = 0
	AxisLeftY        GamepadAxis = 1
	AxisRightX       GamepadAxis = 2
	AxisRightY       GamepadAxis = 3
	AxisLeftTrigger  GamepadAxis = 4
	AxisRightTrigger GamepadAxis = 5
	AxisLast                     = AxisRightTrigger

	// Gamepad buttons
	ButtonA           GamepadButton = 0
	ButtonB           GamepadButton = 1
	ButtonX           GamepadButton = 2
	ButtonY           GamepadButton = 3
	ButtonLeftBumper  GamepadButton = 4
	ButtonRightBumper GamepadButton = 5
	ButtonBack        GamepadButton = 6
	ButtonStart       GamepadButton = 7
	ButtonGuide       GamepadButton = 8
	ButtonLeftThumb   GamepadButton = 9
	ButtonRightThumb  GamepadButton = 10
	ButtonDpadUp      GamepadButton = 11
	ButtonDpadRight   GamepadButton = 12
	ButtonDpadDown    GamepadButton = 13
	ButtonDpadLeft    GamepadButton = 14
	ButtonLast                      = ButtonDpadLeft
	ButtonCross       = ButtonA
	ButtonCircle      = ButtonB
	ButtonSquare      = ButtonX
	ButtonTriangle    = ButtonY

	// Keys (subset — full list in keys.go generated file)
	KeyUnknown      Key = -1
	KeySpace        Key = 32
	KeyApostrophe   Key = 39
	KeyComma        Key = 44
	KeyMinus        Key = 45
	KeyPeriod       Key = 46
	KeySlash        Key = 47
	Key0            Key = 48
	Key1            Key = 49
	Key2            Key = 50
	Key3            Key = 51
	Key4            Key = 52
	Key5            Key = 53
	Key6            Key = 54
	Key7            Key = 55
	Key8            Key = 56
	Key9            Key = 57
	KeySemicolon    Key = 59
	KeyEqual        Key = 61
	KeyA            Key = 65
	KeyB            Key = 66
	KeyC            Key = 67
	KeyD            Key = 68
	KeyE            Key = 69
	KeyF            Key = 70
	KeyG            Key = 71
	KeyH            Key = 72
	KeyI            Key = 73
	KeyJ            Key = 74
	KeyK            Key = 75
	KeyL            Key = 76
	KeyM            Key = 77
	KeyN            Key = 78
	KeyO            Key = 79
	KeyP            Key = 80
	KeyQ            Key = 81
	KeyR            Key = 82
	KeyS            Key = 83
	KeyT            Key = 84
	KeyU            Key = 85
	KeyV            Key = 86
	KeyW            Key = 87
	KeyX            Key = 88
	KeyY            Key = 89
	KeyZ            Key = 90
	KeyEscape       Key = 256
	KeyEnter        Key = 257
	KeyTab          Key = 258
	KeyBackspace    Key = 259
	KeyInsert       Key = 260
	KeyDelete       Key = 261
	KeyRight        Key = 262
	KeyLeft         Key = 263
	KeyDown         Key = 264
	KeyUp           Key = 265
	KeyPageUp       Key = 266
	KeyPageDown     Key = 267
	KeyHome         Key = 268
	KeyEnd          Key = 269
	KeyCapsLock     Key = 280
	KeyScrollLock   Key = 281
	KeyNumLock      Key = 282
	KeyPrintScreen  Key = 283
	KeyPause        Key = 284
	KeyF1           Key = 290
	KeyF2           Key = 291
	KeyF3           Key = 292
	KeyF4           Key = 293
	KeyF5           Key = 294
	KeyF6           Key = 295
	KeyF7           Key = 296
	KeyF8           Key = 297
	KeyF9           Key = 298
	KeyF10          Key = 299
	KeyF11          Key = 300
	KeyF12          Key = 301
	KeyF13          Key = 302
	KeyF14          Key = 303
	KeyF15          Key = 304
	KeyF16          Key = 305
	KeyF17          Key = 306
	KeyF18          Key = 307
	KeyF19          Key = 308
	KeyF20          Key = 309
	KeyF21          Key = 310
	KeyF22          Key = 311
	KeyF23          Key = 312
	KeyF24          Key = 313
	KeyF25          Key = 314
	KeyLeftShift    Key = 340
	KeyLeftControl  Key = 341
	KeyLeftAlt      Key = 342
	KeyLeftSuper    Key = 343
	KeyRightShift   Key = 344
	KeyRightControl Key = 345
	KeyRightAlt     Key = 346
	KeyRightSuper   Key = 347
	KeyMenu         Key = 348

	// Numpad keys
	KeyKP0        Key = 320
	KeyKP1        Key = 321
	KeyKP2        Key = 322
	KeyKP3        Key = 323
	KeyKP4        Key = 324
	KeyKP5        Key = 325
	KeyKP6        Key = 326
	KeyKP7        Key = 327
	KeyKP8        Key = 328
	KeyKP9        Key = 329
	KeyKPDecimal  Key = 330
	KeyKPDivide   Key = 331
	KeyKPMultiply Key = 332
	KeyKPSubtract Key = 333
	KeyKPAdd      Key = 334
	KeyKPEnter    Key = 335
	KeyKPEqual    Key = 336

	// Extra printable keys
	KeyGraveAccent  Key = 96
	KeyLeftBracket  Key = 91
	KeyBackslash    Key = 92
	KeyRightBracket Key = 93
	KeyWorld1       Key = 161
	KeyWorld2       Key = 162

	KeyLast = KeyMenu
)

// StandardCursorShape identifies a built-in system cursor shape.
type StandardCursorShape int

const (
	ArrowCursor      StandardCursorShape = 0x00036001
	IBeamCursor      StandardCursorShape = 0x00036002
	CrosshairCursor  StandardCursorShape = 0x00036003
	HandCursor       StandardCursorShape = 0x00036004
	HResizeCursor    StandardCursorShape = 0x00036005
	VResizeCursor    StandardCursorShape = 0x00036006
)

// Cursor is an opaque cursor object created by CreateCursor or CreateStandardCursor.
type Cursor struct {
	handle uintptr // platform-specific handle (HCURSOR on Windows)
	system bool    // true = system cursor, must not be destroyed
}
