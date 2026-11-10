//go:build darwin && cgo

package entratoken

/*
#include <unistd.h>
#include <uuid/uuid.h>

static int fleetHostUUID(char *out) {
	uuid_t u;
	struct timespec wait = {5, 0};
	if (gethostuuid(u, &wait) != 0) {
		return -1;
	}
	uuid_unparse_upper(u, out);
	return 0;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// hostUUID returns the hardware UUID (IOPlatformUUID), which osquery reports as the host UUID.
func hostUUID() (string, error) {
	var buf [37]C.char
	if C.fleetHostUUID(&buf[0]) != 0 {
		return "", errors.New("get host UUID")
	}
	return C.GoString((*C.char)(unsafe.Pointer(&buf[0]))), nil
}
