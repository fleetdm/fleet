#import <Foundation/Foundation.h>
#import <MSAL/MSAL.h>

void run(void) {
        NSError *error = nil;
        MSALPublicClientApplicationConfig *config = [[MSALPublicClientApplicationConfig alloc]
            initWithClientId:@"<CLIENT_ID>"
            redirectUri:nil
            authority:nil];

        MSALPublicClientApplication *application = [[MSALPublicClientApplication alloc] initWithConfiguration:config error:&error];

        if (error) {
            NSLog(@"Failed to create application: %@", error);
            return;
        }
    
        [application getDeviceInformationWithParameters:nil
                                        completionBlock:^(MSALDeviceInformation * _Nullable deviceInformation, __unused NSError * _Nullable error) {
            NSString *deviceId = deviceInformation.extraDeviceInformation[MSAL_PRIMARY_REGISTRATION_DEVICE_ID];
            NSString *upn = deviceInformation.extraDeviceInformation[MSAL_PRIMARY_REGISTRATION_UPN];
            
            NSLog(@"deviceId = %s, upn = %s", (char*)[deviceId UTF8String], (char*)[upn UTF8String]);
            
         }];

}

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        run();
    }
    return 0;
}
