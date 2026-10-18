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

void setAppIcon(const void* data, int length) {
	Class cls = objc_getClass("NSApplication");
	SEL sel = sel_registerName("sharedApplication");
	id (*objc_msgSend_id)(id, SEL) = (id(*)(id, SEL))objc_msgSend;
	id app = objc_msgSend_id((id)cls, sel);

	Class nsDataCls = objc_getClass("NSData");
	SEL dataSel = sel_registerName("dataWithBytes:length:");
	id (*objc_msgSend_data)(Class, SEL, const void*, unsigned long) = (id(*)(Class, SEL, const void*, unsigned long))objc_msgSend;
	id nsData = objc_msgSend_data(nsDataCls, dataSel, data, (unsigned long)length);

	Class nsImageCls = objc_getClass("NSImage");
	SEL allocSel = sel_registerName("alloc");
	id (*objc_msgSend_alloc)(Class, SEL) = (id(*)(Class, SEL))objc_msgSend;
	id nsImageAlloc = objc_msgSend_alloc(nsImageCls, allocSel);

	SEL initSel = sel_registerName("initWithData:");
	id (*objc_msgSend_init)(id, SEL, id) = (id(*)(id, SEL, id))objc_msgSend;
	id nsImage = objc_msgSend_init(nsImageAlloc, initSel, nsData);

	if (nsImage) {
		SEL setIconSel = sel_registerName("setApplicationIconImage:");
		void (*objc_msgSend_set_icon)(id, SEL, id) = (void(*)(id, SEL, id))objc_msgSend;
		objc_msgSend_set_icon(app, setIconSel, nsImage);
	}
}
*/
import "C"
import "unsafe"

func HideDockIcon() {
	C.setAppActivationPolicy(1)
}

func ShowDockIcon(iconBytes []byte) {
	C.setAppActivationPolicy(0)
	if len(iconBytes) > 0 {
		C.setAppIcon(unsafe.Pointer(&iconBytes[0]), C.int(len(iconBytes)))
	}
	C.activateApp()
}


