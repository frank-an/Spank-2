//go:build darwin
// +build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation

#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <unistd.h>

void keyDown(int keyCode) {
	CGEventRef event = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, true);
	if (event) {
		CGEventPost(kCGHIDEventTap, event);
		CFRelease(event);
	}
}

void keyUp(int keyCode) {
	CGEventRef event = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, false);
	if (event) {
		CGEventPost(kCGHIDEventTap, event);
		CFRelease(event);
	}
}

void pressKey(int keyCode) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, true);
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, false);
	if (down) {
		CGEventPost(kCGHIDEventTap, down);
		CFRelease(down);
	}
	usleep(15000);
	if (up) {
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(up);
	}
}

void pressCmdV(void) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)0x09, true);
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)0x09, false);
	if (down) {
		CGEventSetFlags(down, kCGEventFlagMaskCommand);
		CGEventPost(kCGHIDEventTap, down);
		CFRelease(down);
	}
	usleep(15000);
	if (up) {
		CGEventSetFlags(up, kCGEventFlagMaskCommand);
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(up);
	}
}

void pressKeyShifted(int keyCode) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, true);
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, false);
	if (down) {
		CGEventSetFlags(down, kCGEventFlagMaskShift);
		CGEventPost(kCGHIDEventTap, down);
		CFRelease(down);
	}
	usleep(15000);
	if (up) {
		CGEventSetFlags(up, kCGEventFlagMaskShift);
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(up);
	}
}

static CGPoint getCurrentMouseLocation(void) {
	CGEventRef event = CGEventCreate(NULL);
	if (!event) return CGPointZero;
	CGPoint point = CGEventGetLocation(event);
	CFRelease(event);
	return point;
}

void clickMouse(int button) {
	CGPoint loc = getCurrentMouseLocation();
	CGEventRef down = CGEventCreateMouseEvent(NULL,
		button == 1 ? kCGEventRightMouseDown : kCGEventLeftMouseDown,
		loc,
		(CGMouseButton)button);
	CGEventRef up = CGEventCreateMouseEvent(NULL,
		button == 1 ? kCGEventRightMouseUp : kCGEventLeftMouseUp,
		loc,
		(CGMouseButton)button);
	if (down) {
		CGEventPost(kCGHIDEventTap, down);
		CFRelease(down);
	}
	usleep(10000);
	if (up) {
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(up);
	}
}
*/
import "C"

func pressKey(keyCode int) {
	C.pressKey(C.int(keyCode))
}

func pressCmdV() {
	C.pressCmdV()
}

func pressKeyShifted(keyCode int) {
	C.pressKeyShifted(C.int(keyCode))
}

func clickMouse(button int) {
	C.clickMouse(C.int(button))
}
