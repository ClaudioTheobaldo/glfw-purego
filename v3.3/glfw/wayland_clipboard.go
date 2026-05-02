//go:build linux && wayland

package glfw

import (
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ── Clipboard via wl_data_device ─────────────────────────────────────────────
//
// Protocol sketch:
//   wl_data_device_manager.create_data_source  → wl_data_source  (we offer)
//   wl_data_device_manager.get_data_device     → wl_data_device  (listener hub)
//   wl_data_device.set_selection(source,serial)                   (claim clipboard)
//   wl_data_offer.receive(mime,fd)                                (paste from peer)
//
// Opcodes (wl_data_device_manager):
//   0 = create_data_source  (→ wl_data_source)
//   1 = get_data_device     (seat → wl_data_device)
//
// Opcodes (wl_data_source):
//   0 = offer   "s"       — announce MIME type
//   1 = destroy ""
//
// Opcodes (wl_data_device):
//   1 = set_selection  "?ou"  — (source, serial)
//
// Opcodes (wl_data_offer):
//   1 = receive  "sh"     — (mime_type, pipe_fd)
//   2 = destroy  ""

// preferredMIME is the clipboard MIME type we advertise and prefer to receive.
const preferredMIME = "text/plain;charset=utf-8"

// altMIME is the fallback MIME type.
const altMIME = "text/plain"

// wlInitDataDevice creates the wl_data_device and wires up its listener.
// Called from Init() when both wl_seat and wl_data_device_manager are available.
func wlInitDataDevice() {
	if wlDDDeviceIfaceAddr == 0 {
		// Interface address not available — clipboard not supported.
		return
	}
	// get_data_device opcode=1, signature="no" (new_id + wl_seat)
	wl.dataDevice = wlProxyMarshalFlags(wl.ddManager, 1,
		wlDDDeviceIfaceAddr, 3, 0,
		wl.seat, 0)
	if wl.dataDevice == 0 {
		return
	}
	// Attach listener for 6 events (data_offer, enter, leave, motion, drop, selection).
	wl.ddListener = new([6]uintptr)
	wl.ddListener[0] = purego.NewCallback(wlOnDataDeviceDataOffer)
	wl.ddListener[1] = purego.NewCallback(wlOnDataDeviceEnter)
	wl.ddListener[2] = purego.NewCallback(wlOnDataDeviceLeave)
	wl.ddListener[3] = purego.NewCallback(wlOnDataDeviceMotion)
	wl.ddListener[4] = purego.NewCallback(wlOnDataDeviceDrop)
	wl.ddListener[5] = purego.NewCallback(wlOnDataDeviceSelection)
	wlProxyAddListener(wl.dataDevice, uintptr(unsafe.Pointer(wl.ddListener)), 0)
}

// ── wl_data_device event handlers ────────────────────────────────────────────
// C signatures (all pointers as uintptr):
//   data_offer:  (data, device, offer)
//   enter:       (data, device, serial, surface, x, y, offer)
//   leave:       (data, device)
//   motion:      (data, device, time, x, y)
//   drop:        (data, device)
//   selection:   (data, device, offer_or_null)

func wlOnDataDeviceDataOffer(data, device, offer uintptr) {
	// A new wl_data_offer proxy is about to be used; store it so the selection
	// event can promote it to the clipboard offer.
	wl.pendingOffer = offer
}

func wlOnDataDeviceEnter(data, device uintptr, serial uint32, surface uintptr, x, y int32, offer uintptr) {
	// Drag-and-drop enter — not implemented.
}

func wlOnDataDeviceLeave(data, device uintptr) {}

func wlOnDataDeviceMotion(data, device uintptr, time uint32, x, y int32) {}

func wlOnDataDeviceDrop(data, device uintptr) {}

func wlOnDataDeviceSelection(data, device, offer uintptr) {
	// A new clipboard selection arrived.  Destroy the old offer (if any) and
	// store the new one.  offer == 0 means the clipboard was cleared.
	if wl.lastOffer != 0 && wl.lastOffer != offer {
		// wl_data_offer.destroy opcode=2
		wlProxyMarshalFlags(wl.lastOffer, 2, 0, 1, 1) // destroy + free
	}
	wl.lastOffer = offer
	// If we just set the clipboard ourselves, keep clipboardText as-is.
	if offer == 0 || offer == wl.clipboardSrc {
		return
	}
}

// ── SetClipboardString ────────────────────────────────────────────────────────

// SetClipboardString places text into the system clipboard.
func SetClipboardString(str string) {
	if wl.dataDevice == 0 || wlDDSourceIfaceAddr == 0 {
		return
	}
	// Destroy any previous data source we owned.
	if wl.clipboardSrc != 0 {
		wlProxyMarshalFlags(wl.clipboardSrc, 1, 0, 1, 1) // destroy
		wl.clipboardSrc = 0
	}
	wl.clipboardText = str

	// create_data_source opcode=0, signature="n"
	src := wlProxyMarshalFlags(wl.ddManager, 0, wlDDSourceIfaceAddr, 3, 0, 0)
	if src == 0 {
		return
	}
	wl.clipboardSrc = src

	// Attach a 3-event source listener (target, send, cancelled).
	// Store in wl struct so the GC never collects it while the source is live.
	wl.clipboardSrcListener = new([3]uintptr)
	wl.clipboardSrcListener[0] = purego.NewCallback(func(data, source, mimePtr uintptr) {})
	wl.clipboardSrcListener[1] = purego.NewCallback(wlOnDataSourceSend)
	wl.clipboardSrcListener[2] = purego.NewCallback(wlOnDataSourceCancelled)
	wlProxyAddListener(src, uintptr(unsafe.Pointer(wl.clipboardSrcListener)), 0)

	// Offer both MIME types.
	for _, mime := range [2]string{preferredMIME, altMIME} {
		mimeC := append([]byte(mime), 0)
		// wl_data_source.offer opcode=0, signature="s"
		wlProxyMarshalFlags(src, 0, 0, 1, 0,
			uintptr(unsafe.Pointer(&mimeC[0])))
		wlDisplayFlush(wl.display)
	}

	// set_selection opcode=1, signature="?ou" (source, serial)
	wlProxyMarshalFlags(wl.dataDevice, 1, 0, 1, 0,
		src, uintptr(wl.kbSerial))
	wlDisplayFlush(wl.display)
}

func wlOnDataSourceSend(data, source, mimePtr uintptr, fd int32) {
	// Another client is requesting our clipboard contents.
	text := wl.clipboardText
	if len(text) > 0 {
		buf := []byte(text)
		for len(buf) > 0 {
			n, err := syscall.Write(int(fd), buf)
			if err != nil {
				break
			}
			buf = buf[n:]
		}
	}
	syscall.Close(int(fd))
}

func wlOnDataSourceCancelled(data, source uintptr) {
	// Another client claimed the clipboard; our source is no longer current.
	if source == wl.clipboardSrc {
		wl.clipboardSrc = 0
	}
}

// ── GetClipboardString ────────────────────────────────────────────────────────

// GetClipboardString returns the current clipboard text.
func GetClipboardString() string {
	if wl.dataDevice == 0 {
		return ""
	}
	// Short-circuit: we own the clipboard.
	if wl.clipboardSrc != 0 {
		return wl.clipboardText
	}
	offer := wl.lastOffer
	if offer == 0 || wlDDOfferIfaceAddr == 0 {
		return ""
	}

	// Create a pipe to receive the data.
	var pipeFDs [2]int
	if err := syscall.Pipe2(pipeFDs[:], syscall.O_CLOEXEC); err != nil {
		return ""
	}
	readFD, writeFD := pipeFDs[0], pipeFDs[1]

	// wl_data_offer.receive opcode=1, signature="sh" (mime_type, fd)
	mimeC := append([]byte(preferredMIME), 0)
	wlProxyMarshalFlags(offer, 1, 0, 1, 0,
		uintptr(unsafe.Pointer(&mimeC[0])),
		uintptr(writeFD))
	wlDisplayFlush(wl.display)
	syscall.Close(writeFD)

	// Read the clipboard data from the pipe.
	var buf [4096]byte
	var text []byte
	for {
		n, err := syscall.Read(readFD, buf[:])
		if n > 0 {
			text = append(text, buf[:n]...)
		}
		if err != nil || n == 0 {
			break
		}
	}
	syscall.Close(readFD)

	return string(text)
}
