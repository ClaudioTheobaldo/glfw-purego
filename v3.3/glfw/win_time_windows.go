//go:build windows

package glfw

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modWinmm         = windows.NewLazySystemDLL("winmm.dll")
	procTimeBeginPeriod = modWinmm.NewProc("timeBeginPeriod")

	procQueryPerformanceCounter  = modKernel32.NewProc("QueryPerformanceCounter")
	procQueryPerformanceFrequency = modKernel32.NewProc("QueryPerformanceFrequency")
)

var (
	timerOnce  sync.Once
	timerFreq  int64
	timerBase  int64   // raw counter value at reset point
	timerBaseT float64 // time value at reset point
	timerMu    sync.Mutex
)

func initTimer() {
	timerOnce.Do(func() {
		var freq int64
		procQueryPerformanceFrequency.Call(uintptr(unsafe.Pointer(&freq)))
		atomic.StoreInt64(&timerFreq, freq)

		var cnt int64
		procQueryPerformanceCounter.Call(uintptr(unsafe.Pointer(&cnt)))
		timerBase = cnt
		timerBaseT = 0

		// Request 1ms timer resolution for accurate sleep/wait.
		procTimeBeginPeriod.Call(1)
	})
}

func timeFrequency() int64 {
	initTimer()
	return atomic.LoadInt64(&timerFreq)
}

func timeGetTicks() int64 {
	initTimer()
	var cnt int64
	procQueryPerformanceCounter.Call(uintptr(unsafe.Pointer(&cnt)))
	timerMu.Lock()
	base := timerBase
	timerMu.Unlock()
	return cnt - base
}

func timeSetBase(t float64) {
	initTimer()
	var cnt int64
	procQueryPerformanceCounter.Call(uintptr(unsafe.Pointer(&cnt)))
	timerMu.Lock()
	timerBase = cnt
	timerBaseT = t
	timerMu.Unlock()
}
