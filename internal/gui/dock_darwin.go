//go:build darwin

package gui

/*
#cgo LDFLAGS: -framework Cocoa
#include <objc/objc.h>
#include <objc/runtime.h>
#include <objc/message.h>

void setAppActivationPolicy(long policy) {
	Class cls = objc_getClass("NSApplication");
	SEL sel = sel_registerName("sharedApplication");
	id (*objc_msgSend_id)(id, SEL) = (id(*)(id, SEL))objc_msgSend;
	id app = objc_msgSend_id((id)cls, sel);

	SEL selPolicy = sel_registerName("setActivationPolicy:");
	void (*objc_msgSend_void_int)(id, SEL, long) = (void(*)(id, SEL, long))objc_msgSend;
	objc_msgSend_void_int(app, selPolicy, policy);
}

void activateApp() {
	Class cls = objc_getClass("NSApplication");
	SEL sel = sel_registerName("sharedApplication");
	id (*objc_msgSend_id)(id, SEL) = (id(*)(id, SEL))objc_msgSend;
	id app = objc_msgSend_id((id)cls, sel);

	SEL selActivate = sel_registerName("activateIgnoringOtherApps:");
	void (*objc_msgSend_void_bool)(id, SEL, bool) = (void(*)(id, SEL, bool))objc_msgSend;
	objc_msgSend_void_bool(app, selActivate, true);
}
*/
import "C"

func HideDockIcon() {
	// NSApplicationActivationPolicyAccessory = 1 (Hides from Dock, keeps menu bar/tray)
	C.setAppActivationPolicy(1)
}

func ShowDockIcon() {
	// NSApplicationActivationPolicyRegular = 0 (Shows in Dock, normal app window)
	C.setAppActivationPolicy(0)
	C.activateApp()
}

