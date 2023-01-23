package table

/*
#cgo darwin CFLAGS: -DDARWIN -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa -framework Collaboration
#import <Collaboration/Collaboration.h>
#include <CoreFoundation/CoreFoundation.h>
void Image(CFDataRef *imageDataRef, char* user) {
	NSString *userName = [NSString stringWithFormat:@"%s", user];
	CBIdentity *identity = [CBIdentity identityWithName:userName authority:[CBIdentityAuthority defaultIdentityAuthority]];
	NSImage *userImage = [identity image];
	NSData *imageData = [userImage TIFFRepresentation];
	*imageDataRef = (CFDataRef)imageData;
}
const char * LocalUsers() {
	CSIdentityAuthorityRef defaultAuthority = CSGetDefaultIdentityAuthority();
	CSIdentityQueryRef query = CSIdentityQueryCreate(NULL, kCSIdentityClassUser, defaultAuthority);
	CFErrorRef error = NULL;
	CSIdentityQueryExecute(query, 0, &error);
	CFArrayRef results = CSIdentityQueryCopyResults(query);
	int numResults = CFArrayGetCount(results);
	NSMutableArray *usernames = [NSMutableArray array];
	for (int i = 0; i < numResults; ++i) {
 		CSIdentityRef identity = (CSIdentityRef)CFArrayGetValueAtIndex(results, i);
		CBIdentity *identityObject = [CBIdentity identityWithCSIdentity:identity];
		[usernames addObject:[identityObject posixName]];
	}
	NSString *usernamesString = [usernames componentsJoinedByString:@" "];
	CFRelease(results);
	CFRelease(query);

	const char *cString = [usernamesString UTF8String];
	return cString;
}
*/
import (
	"C"
)
import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"hash/crc64"
	"image"
	"image/png"
	"strings"
	"unsafe"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/nfnt/resize"
	"github.com/osquery/osquery-go/plugin/table"

	"golang.org/x/image/tiff"
)

var crcTable = crc64.MakeTable(crc64.ECMA)

func UserAvatar(logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("username"),
		table.TextColumn("thumbnail"),
		table.TextColumn("hash"),
	}
	t := &userAvatarTable{logger: logger}
	return table.NewPlugin("kolide_user_avatars", columns, t.generateAvatars)
}

type userAvatarTable struct {
	logger log.Logger
}

func (t *userAvatarTable) generateAvatars(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// use the username from the query context if provide, otherwise default to user created users
	var usernames []string
	q, ok := queryContext.Constraints["username"]
	if ok && len(q.Constraints) != 0 {
		for _, constraint := range q.Constraints {
			usernames = append(usernames, constraint.Expression)
		}
	} else {
		usernamesString := C.LocalUsers()
		for _, posixName := range strings.Split(C.GoString(usernamesString), " ") {
			usernames = append(usernames, posixName)
		}
	}

	var results []map[string]string
	for _, username := range usernames {
		image, hash, err := getUserAvatar(username)
		if err != nil {
			level.Debug(t.logger).Log(
				"msg", "error getting user avatar",
				"err", err,
			)
			continue
		}
		if image == nil {
			continue
		}

		var base64Buf bytes.Buffer
		encoder := base64.NewEncoder(base64.StdEncoding, &base64Buf)
		defer encoder.Close()
		thumbnail := resize.Thumbnail(150, 150, image, resize.Lanczos3)
		if err := png.Encode(encoder, thumbnail); err != nil {
			level.Debug(t.logger).Log(
				"msg", "error encoding resized user avatar to png",
				"err", err,
			)
			continue
		}

		results = append(results,
			map[string]string{
				"username":  username,
				"thumbnail": base64Buf.String(),
				"hash":      fmt.Sprintf("%x", hash),
			},
		)
	}

	return results, nil
}

func getUserAvatar(username string) (image.Image, uint64, error) {
	var data C.CFDataRef = 0
	C.Image(&data, C.CString(username))
	if data == 0 {
		return nil, 0, nil
	}
	defer C.CFRelease(C.CFTypeRef(data))

	goBytes := C.GoBytes(unsafe.Pointer(C.CFDataGetBytePtr(data)), C.int(C.CFDataGetLength(data)))
	if len(goBytes) == 0 {
		return nil, 0, nil
	}

	image, err := tiff.Decode(bytes.NewBuffer(goBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("decoding tiff data from C: %w", err)
	}
	hash := crc64.Checksum(goBytes, crcTable)
	return image, hash, nil
}
