//go:build darwin && cgo

package entratoken

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#include <stdlib.h>

typedef struct {
	char *redirectURL;
	char *body;
	int bodyLen;
	char *err;
} fleetAuthorizeResult;

fleetAuthorizeResult fleetEntraAuthorize(const char *url, double timeoutSeconds);

#import <Foundation/Foundation.h>

// Follows redirects within login.microsoftonline.com and stops at the first redirect
// elsewhere (the app's redirect URI), recording its URL.
@interface FleetEntraRedirectStopper : NSObject <NSURLSessionTaskDelegate>
@property (copy) NSString *redirectURL;
@end

@implementation FleetEntraRedirectStopper
- (void)URLSession:(NSURLSession *)session
              task:(NSURLSessionTask *)task
willPerformHTTPRedirection:(NSHTTPURLResponse *)response
        newRequest:(NSURLRequest *)request
 completionHandler:(void (^)(NSURLRequest *))completionHandler {
	if ([request.URL.host isEqualToString:@"login.microsoftonline.com"]) {
		completionHandler(request);
		return;
	}
	self.redirectURL = request.URL.absoluteString;
	completionHandler(nil);
}
@end

static char *fleetCopyString(NSString *s) {
	if (s == nil) {
		return NULL;
	}
	return strdup([s UTF8String]);
}

fleetAuthorizeResult fleetEntraAuthorize(const char *url, double timeoutSeconds) {
	__block fleetAuthorizeResult result = {0};
	@autoreleasepool {
		NSURLSessionConfiguration *config = [NSURLSessionConfiguration ephemeralSessionConfiguration];
		config.timeoutIntervalForRequest = timeoutSeconds;
		config.timeoutIntervalForResource = timeoutSeconds;
		FleetEntraRedirectStopper *delegate = [FleetEntraRedirectStopper new];
		NSURLSession *session = [NSURLSession sessionWithConfiguration:config delegate:delegate delegateQueue:nil];
		dispatch_semaphore_t done = dispatch_semaphore_create(0);
		NSURL *u = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
		if (u == nil) {
			result.err = strdup("invalid authorize URL");
			return result;
		}
		[[session dataTaskWithURL:u completionHandler:^(NSData *data, NSURLResponse *response, NSError *error) {
			NSString *redirectURL = delegate.redirectURL;
			// A redirect to a custom scheme can also surface as an "unsupported URL" error.
			if (redirectURL == nil && error != nil) {
				redirectURL = [error.userInfo[NSURLErrorFailingURLErrorKey] absoluteString];
				if (redirectURL != nil && [redirectURL hasPrefix:@"https://login.microsoftonline.com/"]) {
					redirectURL = nil;
				}
			}
			if (redirectURL != nil) {
				result.redirectURL = fleetCopyString(redirectURL);
			} else if (error != nil) {
				result.err = fleetCopyString(error.localizedDescription);
			}
			// The page holding the code is a few KB; cap what's copied to Go.
			NSUInteger n = MIN(data.length, (NSUInteger)(1 << 20));
			if (n > 0) {
				result.body = malloc(n);
				if (result.body != NULL) {
					memcpy(result.body, data.bytes, n);
					result.bodyLen = (int)n;
				}
			}
			dispatch_semaphore_signal(done);
		}] resume];
		dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
		[session finishTasksAndInvalidate];
	}
	return result;
}
*/
import "C"

import (
	"context"
	"errors"
	"time"
	"unsafe"
)

func authorize(ctx context.Context, authorizeURL string) (*authorizeResponse, error) {
	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	if timeout <= 0 {
		return nil, context.DeadlineExceeded
	}

	cURL := C.CString(authorizeURL)
	defer C.free(unsafe.Pointer(cURL))

	res := C.fleetEntraAuthorize(cURL, C.double(timeout.Seconds()))
	defer func() {
		C.free(unsafe.Pointer(res.redirectURL))
		C.free(unsafe.Pointer(res.body))
		C.free(unsafe.Pointer(res.err))
	}()

	if res.err != nil {
		return nil, errors.New("authorize request: " + C.GoString(res.err))
	}
	resp := &authorizeResponse{}
	if res.redirectURL != nil {
		resp.redirectURL = C.GoString(res.redirectURL)
	}
	if res.body != nil {
		resp.body = C.GoBytes(unsafe.Pointer(res.body), res.bodyLen)
	}
	return resp, nil
}
