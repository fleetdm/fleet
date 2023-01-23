//nolint:unconvert
package table

/*
#cgo darwin CFLAGS: -DDARWIN -x objective-c
#include <CoreFoundation/CoreFoundation.h>
*/
import (
	"C"
)
import (
	"fmt"
	"unsafe"
)

// Functions with "Create" or "Copy" in the name return references that need to
// be CFReleased. See
// https://developer.apple.com/library/archive/documentation/CoreFoundation/Conceptual/CFMemoryMgmt/Concepts/Ownership.html#//apple_ref/doc/uid/20001148-103029

func copyPreferenceValue(key, domain, username string) interface{} {
	keyCFString := cFStringRef(key)
	defer C.CFRelease((C.CFTypeRef)(keyCFString))
	domainCFString := cFStringRef(domain)
	defer C.CFRelease((C.CFTypeRef)(domainCFString))
	usernameCFString := cFStringRef(username)
	defer C.CFRelease((C.CFTypeRef)(usernameCFString))

	val := C.CFPreferencesCopyValue(
		keyCFString, domainCFString, usernameCFString, C.kCFPreferencesAnyHost,
	)
	if C.CFTypeRef(val) != 0 {
		// will panic if the is NULL
		defer C.CFRelease((C.CFTypeRef)(val))
	}
	return goValueFromCFPlistRef(val)
}

// cFStringRef returns a C.CFStringRef which must be released with C.CFRelease
func cFStringRef(s string) C.CFStringRef {
	return C.CFStringCreateWithCString(C.kCFAllocatorDefault, C.CString(s), C.kCFStringEncodingUTF8)
}

func goBoolean(ref C.CFBooleanRef) bool {
	return ref == C.kCFBooleanTrue
}

func goInt(ref C.CFNumberRef) int {
	var n int
	C.CFNumberGetValue(ref, C.CFNumberGetType(ref), unsafe.Pointer(&n))
	return n
}

func goString(ref C.CFStringRef) string {
	length := C.CFStringGetLength(ref)
	if length == 0 {
		// empty string
		return ""
	}
	cfRange := C.CFRange{0, length}
	enc := C.CFStringEncoding(C.kCFStringEncodingUTF8)
	var usedBufLen C.CFIndex
	if C.CFStringGetBytes(ref, cfRange, enc, 0, C.false, nil, 0, &usedBufLen) > 0 {
		bytes := make([]byte, usedBufLen)
		buffer := (*C.UInt8)(unsafe.Pointer(&bytes[0]))
		if C.CFStringGetBytes(ref, cfRange, enc, 0, C.false, buffer, usedBufLen, nil) > 0 {
			return *(*string)(unsafe.Pointer(&bytes))
		}
	}

	return ""
}

func goValueFromCFPlistRef(ref C.CFPropertyListRef) interface{} {
	if C.CFTypeRef(ref) == 0 {
		return "Unknown"
	}
	switch typeID := C.CFGetTypeID(C.CFTypeRef(ref)); typeID {
	case C.CFBooleanGetTypeID():
		return goBoolean(C.CFBooleanRef(ref))
	case C.CFNumberGetTypeID():
		return goInt(C.CFNumberRef(ref))
	case C.CFStringGetTypeID():
		return goString(C.CFStringRef(ref))
	default:
		panic(fmt.Sprintf("unknown CF type id %v", typeID))
	}
}
