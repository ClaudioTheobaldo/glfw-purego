//go:build windows

package glfw

import "unsafe"

// ----------------------------------------------------------------------------
// Win32 structs
// ----------------------------------------------------------------------------

type _WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type _MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      _POINT
}

type _POINT struct{ X, Y int32 }

type _RECT struct{ Left, Top, Right, Bottom int32 }

type _PIXELFORMATDESCRIPTOR struct {
	NSize           uint16
	NVersion        uint16
	DwFlags         uint32
	IPixelType      byte
	CColorBits      byte
	CRedBits        byte
	CRedShift       byte
	CGreenBits      byte
	CGreenShift     byte
	CBlueBits       byte
	CBlueShift      byte
	CAlphaBits      byte
	CAlphaShift     byte
	CAccumBits      byte
	CAccumRedBits   byte
	CAccumGreenBits byte
	CAccumBlueBits  byte
	CAccumAlphaBits byte
	CDepthBits      byte
	CStencilBits    byte
	CAuxBuffers     byte
	ILayerType      byte
	BReserved       byte
	DwLayerMask     uint32
	DwVisibleMask   uint32
	DwDamageMask    uint32
}

type _MONITORINFOEXW struct {
	CbSize    uint32
	RcMonitor _RECT
	RcWork    _RECT
	DwFlags   uint32
	SzDevice  [32]uint16
}

type _DEVMODEW struct {
	DmDeviceName    [32]uint16
	DmSpecVersion   uint16
	DmDriverVersion uint16
	DmSize          uint16
	DmDriverExtra   uint16
	DmFields        uint32
	// Position/display union fields
	DmPositionX        int32
	DmPositionY        int32
	DmDisplayOrientation uint32
	DmDisplayFixedOutput uint32
	DmColor            int16
	DmDuplex           int16
	DmYResolution      int16
	DmTTOption         int16
	DmCollate          int16
	DmFormName         [32]uint16
	DmLogPixels        uint16
	DmBitsPerPel       uint32
	DmPelsWidth        uint32
	DmPelsHeight       uint32
	DmDisplayFlags     uint32
	DmDisplayFrequency uint32
	DmICMMethod        uint32
	DmICMIntent        uint32
	DmMediaType        uint32
	DmDitherType       uint32
	DmReserved1        uint32
	DmReserved2        uint32
	DmPanningWidth     uint32
	DmPanningHeight    uint32
}

// Ensure struct sizes are sane at compile time.
var _ = unsafe.Sizeof(_PIXELFORMATDESCRIPTOR{}) // 40 bytes

// ----------------------------------------------------------------------------
// Window class styles
// ----------------------------------------------------------------------------

const (
	_CS_HREDRAW = 0x0002
	_CS_VREDRAW = 0x0001
	_CS_OWNDC   = 0x0020
)

// ----------------------------------------------------------------------------
// Window styles
// ----------------------------------------------------------------------------

const (
	_WS_OVERLAPPED   = uintptr(0x00000000)
	_WS_POPUP        = uintptr(0x80000000)
	_WS_CAPTION      = uintptr(0x00C00000)
	_WS_SYSMENU      = uintptr(0x00080000)
	_WS_THICKFRAME   = uintptr(0x00040000)
	_WS_MINIMIZEBOX  = uintptr(0x00020000)
	_WS_MAXIMIZEBOX  = uintptr(0x00010000)
	_WS_BORDER       = uintptr(0x00800000)
	_WS_VISIBLE      = uintptr(0x10000000)
	_WS_MAXIMIZE     = uintptr(0x01000000)
	_WS_CLIPCHILDREN = uintptr(0x02000000)
	_WS_CLIPSIBLINGS = uintptr(0x04000000)

	_WS_OVERLAPPEDWINDOW = _WS_OVERLAPPED | _WS_CAPTION | _WS_SYSMENU |
		_WS_THICKFRAME | _WS_MINIMIZEBOX | _WS_MAXIMIZEBOX
)

// Extended window styles
const (
	_WS_EX_APPWINDOW   = uintptr(0x00040000)
	_WS_EX_TOPMOST     = uintptr(0x00000008)
	_WS_EX_ACCEPTFILES = uintptr(0x00000010)
	_WS_EX_LAYERED     = uintptr(0x00080000)
)

// SetLayeredWindowAttributes flags
const (
	_LWA_ALPHA    = uintptr(0x2)
	_LWA_COLORKEY = uintptr(0x1)
)

// ----------------------------------------------------------------------------
// Window messages
// ----------------------------------------------------------------------------

const (
	_WM_NULL         = 0x0000
	_WM_CLOSE        = 0x0010
	_WM_DESTROY      = 0x0002
	_WM_QUIT         = 0x0012
	_WM_SIZE         = 0x0005
	_WM_MOVE         = 0x0003
	_WM_SETFOCUS     = 0x0007
	_WM_KILLFOCUS    = 0x0008
	_WM_PAINT        = 0x000F
	_WM_ERASEBKGND   = 0x0014
	_WM_CHAR         = 0x0102
	_WM_SYSCHAR      = 0x0106
	_WM_KEYDOWN      = 0x0100
	_WM_KEYUP        = 0x0101
	_WM_SYSKEYDOWN   = 0x0104
	_WM_SYSKEYUP     = 0x0105
	_WM_UNICHAR      = 0x0109
	_WM_LBUTTONDOWN  = 0x0201
	_WM_LBUTTONUP    = 0x0202
	_WM_RBUTTONDOWN  = 0x0204
	_WM_RBUTTONUP    = 0x0205
	_WM_MBUTTONDOWN  = 0x0207
	_WM_MBUTTONUP    = 0x0208
	_WM_XBUTTONDOWN  = 0x020B
	_WM_XBUTTONUP    = 0x020C
	_WM_MOUSEMOVE    = 0x0200
	_WM_MOUSEWHEEL   = 0x020A
	_WM_MOUSEHWHEEL  = 0x020E
	_WM_ENTERSIZEMOVE = 0x0231
	_WM_EXITSIZEMOVE  = 0x0232
	_WM_DROPFILES    = 0x0233
	_WM_DPICHANGED   = 0x02E0
	_WM_NCCREATE     = 0x0081
	_WM_NCDESTROY    = 0x0082
	_WM_SETICON        = 0x0080
	_WM_DISPLAYCHANGE  = 0x007E
	_WM_INPUT          = uint32(0x00FF)
)

// WM_SIZE wParam values
const (
	_SIZE_RESTORED  = 0
	_SIZE_MINIMIZED = 1
	_SIZE_MAXIMIZED = 2
)

// WM_XBUTTON wParam high word
const (
	_XBUTTON1 = 0x0001
	_XBUTTON2 = 0x0002
)

// ----------------------------------------------------------------------------
// ShowWindow commands
// ----------------------------------------------------------------------------

const (
	_SW_HIDE        = 0
	_SW_SHOW        = 5
	_SW_SHOWDEFAULT = 10
	_SW_MAXIMIZE    = 3
	_SW_MINIMIZE    = 6
	_SW_RESTORE     = 9
)

// ----------------------------------------------------------------------------
// SetWindowPos flags
// ----------------------------------------------------------------------------

const (
	_SWP_NOSIZE     = 0x0001
	_SWP_NOMOVE     = 0x0002
	_SWP_NOZORDER   = 0x0004
	_SWP_NOACTIVATE = 0x0010
	_SWP_FRAMECHANGED = 0x0020
	_HWND_TOP       = uintptr(0)
)

// ----------------------------------------------------------------------------
// PeekMessage flags
// ----------------------------------------------------------------------------

const (
	_PM_NOREMOVE = 0x0000
	_PM_REMOVE   = 0x0001
)

// MsgWaitForMultipleObjectsEx flags
const (
	_QS_ALLINPUT         = 0x04FF
	_MWMO_INPUTAVAILABLE = 0x0004
)

// GetWindowLong / SetWindowLong index
const (
	_GWL_STYLE   = -16
	_GWL_EXSTYLE = -20
)

// ----------------------------------------------------------------------------
// CW_USEDEFAULT — sentinel for CreateWindowEx position/size.
// This is the int32 value 0x80000000 = -2147483648.
// ----------------------------------------------------------------------------

const _CW_USEDEFAULT int = -2147483648

// ----------------------------------------------------------------------------
// Pixel format descriptor flags
// ----------------------------------------------------------------------------

const (
	_PFD_DRAW_TO_WINDOW = 0x00000004
	_PFD_SUPPORT_OPENGL = 0x00000020
	_PFD_DOUBLEBUFFER   = 0x00000001
	_PFD_TYPE_RGBA      = byte(0)
	_PFD_MAIN_PLANE     = byte(0)
)

// ----------------------------------------------------------------------------
// WGL context attribs (wglCreateContextAttribsARB)
// ----------------------------------------------------------------------------

const (
	_WGL_CONTEXT_MAJOR_VERSION_ARB             = 0x2091
	_WGL_CONTEXT_MINOR_VERSION_ARB             = 0x2092
	_WGL_CONTEXT_FLAGS_ARB                     = 0x2094
	_WGL_CONTEXT_PROFILE_MASK_ARB              = 0x9126
	_WGL_CONTEXT_CORE_PROFILE_BIT_ARB          = 0x00000001
	_WGL_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB = 0x00000002
	_WGL_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB    = 0x00000002
	_WGL_CONTEXT_DEBUG_BIT_ARB                 = 0x00000001
)

// WGL pixel format attribs (wglChoosePixelFormatARB)
const (
	_WGL_DRAW_TO_WINDOW_ARB         = 0x2001
	_WGL_SUPPORT_OPENGL_ARB         = 0x2010
	_WGL_DOUBLE_BUFFER_ARB          = 0x2011
	_WGL_PIXEL_TYPE_ARB             = 0x2013
	_WGL_COLOR_BITS_ARB             = 0x2014
	_WGL_RED_BITS_ARB               = 0x2015
	_WGL_GREEN_BITS_ARB             = 0x2017
	_WGL_BLUE_BITS_ARB              = 0x2019
	_WGL_ALPHA_BITS_ARB             = 0x201B
	_WGL_DEPTH_BITS_ARB             = 0x2022
	_WGL_STENCIL_BITS_ARB           = 0x2023
	_WGL_SAMPLE_BUFFERS_ARB         = 0x2041
	_WGL_SAMPLES_ARB                = 0x2042
	_WGL_TYPE_RGBA_ARB              = 0x202B
	_WGL_FRAMEBUFFER_SRGB_CAPABLE_ARB = 0x20A9
	_WGL_ACCELERATION_ARB           = 0x2003
	_WGL_FULL_ACCELERATION_ARB      = 0x2027
)

// ----------------------------------------------------------------------------
// Monitor / display constants
// ----------------------------------------------------------------------------

const (
	_MONITORINFOF_PRIMARY    = 0x00000001
	_ENUM_CURRENT_SETTINGS   = ^uint32(0) // 0xFFFFFFFF
	_MDT_EFFECTIVE_DPI       = 0
)

// DPI awareness values for SetProcessDpiAwareness
const (
	_PROCESS_DPI_UNAWARE           = 0
	_PROCESS_SYSTEM_DPI_AWARE      = 1
	_PROCESS_PER_MONITOR_DPI_AWARE = 2
)

// ----------------------------------------------------------------------------
// Virtual key codes
// ----------------------------------------------------------------------------

const (
	_VK_BACK    = 0x08
	_VK_TAB     = 0x09
	_VK_RETURN  = 0x0D
	_VK_SHIFT   = 0x10
	_VK_CONTROL = 0x11
	_VK_MENU    = 0x12 // Alt
	_VK_PAUSE   = 0x13
	_VK_CAPITAL = 0x14 // CapsLock
	_VK_ESCAPE  = 0x1B
	_VK_SPACE   = 0x20
	_VK_PRIOR   = 0x21 // PageUp
	_VK_NEXT    = 0x22 // PageDown
	_VK_END     = 0x23
	_VK_HOME    = 0x24
	_VK_LEFT    = 0x25
	_VK_UP      = 0x26
	_VK_RIGHT   = 0x27
	_VK_DOWN    = 0x28
	_VK_PRINT   = 0x2A
	_VK_SNAPSHOT= 0x2C // PrintScreen
	_VK_INSERT  = 0x2D
	_VK_DELETE  = 0x2E
	// 0x30–0x39: '0'–'9'
	// 0x41–0x5A: 'A'–'Z'
	_VK_LWIN        = 0x5B
	_VK_RWIN        = 0x5C
	_VK_APPS        = 0x5D // Menu key
	_VK_NUMPAD0     = 0x60
	_VK_NUMPAD9     = 0x69
	_VK_MULTIPLY    = 0x6A
	_VK_ADD         = 0x6B
	_VK_SEPARATOR   = 0x6C
	_VK_SUBTRACT    = 0x6D
	_VK_DECIMAL     = 0x6E
	_VK_DIVIDE      = 0x6F
	_VK_F1          = 0x70
	_VK_F25         = 0x88
	_VK_NUMLOCK     = 0x90
	_VK_SCROLL      = 0x91 // ScrollLock
	_VK_LSHIFT      = 0xA0
	_VK_RSHIFT      = 0xA1
	_VK_LCONTROL    = 0xA2
	_VK_RCONTROL    = 0xA3
	_VK_LMENU       = 0xA4
	_VK_RMENU       = 0xA5
	_VK_OEM_1       = 0xBA // ;: on US
	_VK_OEM_PLUS    = 0xBB // =+
	_VK_OEM_COMMA   = 0xBC // ,<
	_VK_OEM_MINUS   = 0xBD // -_
	_VK_OEM_PERIOD  = 0xBE // .>
	_VK_OEM_2       = 0xBF // /? on US
	_VK_OEM_3       = 0xC0 // `~ on US
	_VK_OEM_4       = 0xDB // [{ on US
	_VK_OEM_5       = 0xDC // \| on US
	_VK_OEM_6       = 0xDD // ]} on US
	_VK_OEM_7       = 0xDE // '" on US
)

// IDC / IDI for LoadCursor / LoadIcon
const (
	_IDC_ARROW       = uintptr(32512)
	_IDI_APPLICATION = uintptr(32512)
)

// ----------------------------------------------------------------------------
// lParam bit helpers
// ----------------------------------------------------------------------------

func loword(v uintptr) int16 { return int16(uint16(v)) }
func hiword(v uintptr) int16 { return int16(uint16(v >> 16)) }
func getXLParam(lp uintptr) int { return int(int16(uint16(lp))) }
func getYLParam(lp uintptr) int { return int(int16(uint16(lp >> 16))) }
func getWheelDelta(wp uintptr) float64 {
	return float64(int16(uint16(wp>>16))) / 120.0
}
func getXButton(wp uintptr) uint16 { return uint16(wp >> 16) }

// ----------------------------------------------------------------------------
// BITMAPV5HEADER for cursor and icon creation with alpha channel.
// ----------------------------------------------------------------------------

type _BITMAPV5HEADER struct {
	BV5Size          uint32
	BV5Width         int32
	BV5Height        int32
	BV5Planes        uint16
	BV5BitCount      uint16
	BV5Compression   uint32
	BV5SizeImage     uint32
	BV5XPelsPerMeter int32
	BV5YPelsPerMeter int32
	BV5ClrUsed       uint32
	BV5ClrImportant  uint32
	BV5RedMask       uint32
	BV5GreenMask     uint32
	BV5BlueMask      uint32
	BV5AlphaMask     uint32
	BV5CSType        uint32
	BV5Endpoints     [36]byte // CIEXYZTRIPLE
	BV5GammaRed      uint32
	BV5GammaGreen    uint32
	BV5GammaBlue     uint32
	BV5Intent        uint32
	BV5ProfileData   uint32
	BV5ProfileSize   uint32
	BV5Reserved      uint32
}

// _ICONINFO is used with CreateIconIndirect to create cursors and icons.
type _ICONINFO struct {
	FIcon    int32  // 1 = icon, 0 = cursor
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

// ----------------------------------------------------------------------------
// ChangeDisplaySettingsExW flags
// ----------------------------------------------------------------------------

const (
	_CDS_FULLSCREEN         = uintptr(0x00000004)
	_DISP_CHANGE_SUCCESSFUL = int32(0)
)

// DEVMODEW dmFields bits
const (
	_DM_BITSPERPEL       = uint32(0x00040000)
	_DM_PELSWIDTH        = uint32(0x00080000)
	_DM_PELSHEIGHT       = uint32(0x00100000)
	_DM_DISPLAYFREQUENCY = uint32(0x00400000)
)

// Bitmap compression
const _BI_BITFIELDS = uint32(3)
const _LCS_WINDOWS_COLOR_SPACE = uint32(0x57696E20)
const _DIB_RGB_COLORS = uint32(0)

// WM_SETICON wParam (supplement — _WM_SETICON message already defined above)
const (
	_ICON_SMALL = uintptr(0)
	_ICON_BIG   = uintptr(1)
)

// WM_SETCURSOR
const (
	_WM_SETCURSOR = uint32(0x0020)
	_HTCLIENT     = uintptr(1)
)

// Additional IDC cursor shapes
const (
	_IDC_IBEAM    = uintptr(32513)
	_IDC_CROSS    = uintptr(32515)
	_IDC_HAND     = uintptr(32649)
	_IDC_SIZEWE   = uintptr(32644)
	_IDC_SIZENS   = uintptr(32645)
	_IDC_SIZENWSE = uintptr(32642)
	_IDC_SIZENESW = uintptr(32643)
	_IDC_SIZEALL  = uintptr(32646)
	_IDC_NO       = uintptr(32648)
)

// WM_GETMINMAXINFO / WM_SIZING
const (
	_WM_GETMINMAXINFO = uint32(0x0024)
	_WM_SIZING        = uint32(0x0214)

	// wParam values for WM_SIZING
	_WMSZ_LEFT        = 1
	_WMSZ_RIGHT       = 2
	_WMSZ_TOP         = 3
	_WMSZ_TOPLEFT     = 4
	_WMSZ_TOPRIGHT    = 5
	_WMSZ_BOTTOM      = 6
	_WMSZ_BOTTOMLEFT  = 7
	_WMSZ_BOTTOMRIGHT = 8
)

// MINMAXINFO — lParam of WM_GETMINMAXINFO
type _MINMAXINFO struct {
	PtReserved     _POINT
	PtMaxSize      _POINT
	PtMaxPosition  _POINT
	PtMinTrackSize _POINT
	PtMaxTrackSize _POINT
}

// FlashWindowEx constants
const (
	_FLASHW_STOP      = uint32(0)
	_FLASHW_CAPTION   = uint32(0x00000001)
	_FLASHW_TRAY      = uint32(0x00000002)
	_FLASHW_ALL       = uint32(0x00000003)
	_FLASHW_TIMER     = uint32(0x00000004)
	_FLASHW_TIMERNOEDIT = uint32(0x0000000C)
)

// FLASHWINFO — parameter for FlashWindowEx
type _FLASHWINFO struct {
	CbSize    uint32
	Hwnd      uintptr
	DwFlags   uint32
	UCount    uint32
	DwTimeout uint32
}

// MapVirtualKey mapping types
const (
	_MAPVK_VK_TO_VSC    = uint32(0)
	_MAPVK_VSC_TO_VK    = uint32(1)
	_MAPVK_VK_TO_CHAR   = uint32(2)
	_MAPVK_VSC_TO_VK_EX = uint32(3)
)

// Extended key flag for GetKeyNameTextW
const _KF_EXTENDED = uint32(0x0100)

// ----------------------------------------------------------------------------
// Raw input (WM_INPUT / RegisterRawInputDevices) — Group 6
// ----------------------------------------------------------------------------

// _RAWINPUTDEVICE registers a device class for raw input delivery.
// 64-bit layout: usUsagePage(2)+usUsage(2)+dwFlags(4)+hwndTarget(8) = 16 bytes
type _RAWINPUTDEVICE struct {
	UsUsagePage uint16
	UsUsage     uint16
	DwFlags     uint32
	HwndTarget  uintptr
}

// _RAWINPUTHEADER is the header prepended to every RAWINPUT structure.
// 64-bit layout: dwType(4)+dwSize(4)+hDevice(8)+wParam(8) = 24 bytes
type _RAWINPUTHEADER struct {
	DwType  uint32
	DwSize  uint32
	HDevice uintptr
	WParam  uintptr
}

// _RAWMOUSE holds relative or absolute mouse movement data.
// 64-bit layout: usFlags(2)+pad(2)+usButtonFlags(2)+usButtonData(2)+
//                ulRawButtons(4)+lLastX(4)+lLastY(4)+ulExtraInfo(4) = 24 bytes
type _RAWMOUSE struct {
	UsFlags       uint16
	_             uint16 // padding before button union
	UsButtonFlags uint16
	UsButtonData  uint16
	UlRawButtons  uint32
	LLastX        int32
	LLastY        int32
	UlExtraInformation uint32
}

// _RAWINPUT is the complete raw input packet for a mouse event.
// 64-bit layout: header(24)+mouse(24) = 48 bytes
type _RAWINPUT struct {
	Header _RAWINPUTHEADER
	Mouse  _RAWMOUSE
}

// Raw input usage constants (HID usage page 1 = Generic Desktop).
const (
	_HID_USAGE_PAGE_GENERIC = uint16(0x01)
	_HID_USAGE_GENERIC_MOUSE = uint16(0x02)
)

// RegisterRawInputDevices flags.
const (
	_RIDEV_INPUTSINK = uint32(0x00000100) // receive input even when not in focus
	_RIDEV_REMOVE    = uint32(0x00000001) // unregister a device
)

// GetRawInputData command.
const _RID_INPUT = uint32(0x10000003)

// RAWINPUTHEADER.dwType values.
const _RIM_TYPEMOUSE = uint32(0)

// RAWMOUSE.usFlags: when bit 0 is set the values are absolute, not relative.
const _MOUSE_MOVE_ABSOLUTE = uint16(0x0001)
