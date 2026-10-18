package gui

/*
#cgo LDFLAGS: -framework Cocoa
#include <objc/objc.h>
#include <objc/runtime.h>
#include <objc/message.h>

void hideDockIcon() {
	Class cls = objc_getClass("NSApplication");
	SEL sel = sel_registerName("sharedApplication");
	id (*objc_msgSend_id)(id, SEL) = (id(*)(id, SEL))objc_msgSend;
	id app = objc_msgSend_id((id)cls, sel);

	SEL selPolicy = sel_registerName("setActivationPolicy:");
	void (*objc_msgSend_void_int)(id, SEL, long) = (void(*)(id, SEL, long))objc_msgSend;
	// NSApplicationActivationPolicyAccessory = 1
	objc_msgSend_void_int(app, selPolicy, 1);
}
*/
import "C"

func init() {
	C.hideDockIcon()
}
