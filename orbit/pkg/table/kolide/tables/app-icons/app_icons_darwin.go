//go:build darwin
// +build darwin

package appicons

/*
#cgo darwin CFLAGS: -DDARWIN -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa
#import <Appkit/AppKit.h>
void Icon(CFDataRef *iconDataRef, char* path) {
	NSString *appPath = [[NSString stringWithUTF8String:path] stringByStandardizingPath];
	NSImage *img = [[NSWorkspace sharedWorkspace] iconForFile:appPath];

	//request 128x128 since we are going to resize the icon
	NSRect targetFrame = NSMakeRect(0, 0, 128, 128);
	CGImageRef cgref = [img CGImageForProposedRect:&targetFrame context:nil hints:nil];
	NSBitmapImageRep *brep = [[NSBitmapImageRep alloc] initWithCGImage:cgref];
	NSData *imageData = [brep TIFFRepresentation];
	*iconDataRef = (CFDataRef)imageData;
}
*/
import (
	"C"
)
import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"

	"fmt"
	"hash/crc64"
	"image"
	"image/png"
	"unsafe"

	"github.com/nfnt/resize"
	"github.com/osquery/osquery-go/plugin/table"

	"golang.org/x/image/tiff"
)

var crcTable = crc64.MakeTable(crc64.ECMA)

func AppIcons() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("path"),
		table.TextColumn("icon"),
		table.TextColumn("hash"),
	}
	return table.NewPlugin("kolide_app_icons", columns, generateAppIcons)
}

func generateAppIcons(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	q, ok := queryContext.Constraints["path"]
	if !ok || len(q.Constraints) == 0 {
		return nil, errors.New("The kolide_app_icons table requires that you specify a constraint WHERE path =")
	}
	path := q.Constraints[0].Expression
	img, hash, err := getAppIcon(path, queryContext)
	if err != nil {
		return nil, err
	}

	var results []map[string]string
	buf := new(bytes.Buffer)
	img = resize.Resize(128, 128, img, resize.Bilinear)
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}
	results = append(results, map[string]string{
		"path": path,
		"icon": base64.StdEncoding.EncodeToString(buf.Bytes()),
		"hash": fmt.Sprintf("%x", hash),
	})

	return results, nil
}

func getAppIcon(appPath string, queryContext table.QueryContext) (image.Image, uint64, error) {
	var data C.CFDataRef = 0
	C.Icon(&data, C.CString(appPath))
	defer C.CFRelease(C.CFTypeRef(data))

	tiffBytes := C.GoBytes(unsafe.Pointer(C.CFDataGetBytePtr(data)), C.int(C.CFDataGetLength(data)))
	img, err := tiff.Decode(bytes.NewBuffer(tiffBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("decoding tiff bytes: %w", err)
	}
	checksum := crc64.Checksum(tiffBytes, crcTable)

	return img, checksum, nil
}
