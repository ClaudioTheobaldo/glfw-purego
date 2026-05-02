//go:build darwin

// darwin_clipboard.go — NSPasteboard clipboard support.

package glfw

import (
	"github.com/ebitengine/purego/objc"
)

// ── SEL cache (clipboard-specific) ───────────────────────────────────────────

var (
	selGeneralPasteboard     = objc.RegisterName("generalPasteboard")
	selClearContents         = objc.RegisterName("clearContents")
	selWriteObjects          = objc.RegisterName("writeObjects:")
	selReadObjectsForClasses = objc.RegisterName("readObjectsForClasses:options:")
)

// ── SetClipboardString ────────────────────────────────────────────────────────

// SetClipboardString writes s to the system clipboard via NSPasteboard.
func SetClipboardString(s string) {
	pb := objc.ID(objc.GetClass("NSPasteboard")).Send(selGeneralPasteboard)
	pb.Send(selClearContents)

	nsStr := nsStringFromGoString(s)
	// Build a one-element NSArray containing the NSString.
	arr := nsArrayWithID(nsStr)
	pb.Send(selWriteObjects, arr)
}

// ── GetClipboardString ────────────────────────────────────────────────────────

// GetClipboardString reads the system clipboard as a plain-text string.
// Returns "" if the clipboard is empty or contains no string data.
func GetClipboardString() string {
	pb := objc.ID(objc.GetClass("NSPasteboard")).Send(selGeneralPasteboard)

	// Ask for NSString objects only.
	classArr := nsArrayWithID(objc.ID(objc.GetClass("NSString")))
	result := pb.Send(selReadObjectsForClasses, classArr, objc.ID(0))
	if result == 0 {
		return ""
	}
	count := objc.Send[uint64](result, selCount)
	if count == 0 {
		return ""
	}
	nsStr := result.Send(selObjectAtIndex, uint64(0))
	return goStringFromNS(nsStr)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// nsArrayWithID returns a single-element NSArray containing obj.
// Uses +[NSArray arrayWithObject:] which is simpler than arrayWithObjects:count:.
func nsArrayWithID(obj objc.ID) objc.ID {
	return objc.ID(objc.GetClass("NSArray")).Send(selArrayWithObject, obj)
}
