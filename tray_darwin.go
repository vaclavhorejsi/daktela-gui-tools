package main

/*
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

void trayInit(const unsigned char* png, int pngLen);
void trayUpdateHistory(const char** titles, int count);
*/
import "C"
import (
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxHistorySlots = 10

var _trayApp *App

//export goTrayCallback
func goTrayCallback(tag C.int) {
	a := _trayApp
	if a == nil {
		return
	}
	t := int(tag)
	switch {
	case t == 1:
		go a.ShowConnectDialog()
	case t == 2:
		go a.ShowMountDialog()
	case t == 99:
		runtime.Quit(a.ctx)
	case t >= 100 && t < 110:
		idx := t - 100
		if idx < len(a.mountHistory) {
			go a.ExecuteMount(a.mountHistory[idx])
		}
	case t >= 200 && t < 210:
		idx := t - 200
		if idx < len(a.mountHistory) {
			go a.ExecuteConnect(a.mountHistory[idx])
		}
	}
}

func (a *App) setupTray() {
	_trayApp = a
	C.trayInit((*C.uchar)(unsafe.Pointer(&icon[0])), C.int(len(icon)))

	a.trayUpdate = func() {
		history := a.mountHistory
		limit := len(history)
		if limit > maxHistorySlots {
			limit = maxHistorySlots
		}
		cstrs := make([]*C.char, limit)
		for i, s := range history[:limit] {
			cstrs[i] = C.CString(s)
		}
		var ptr **C.char
		if limit > 0 {
			ptr = &cstrs[0]
		}
		C.trayUpdateHistory(ptr, C.int(limit))
		for _, cs := range cstrs {
			C.free(unsafe.Pointer(cs))
		}
	}
	a.trayUpdate()
}
