#include "sus.h"

#import <Cocoa/Cocoa.h>
#import <SUScanController.h>
#import <SUSharedPrefs.h>
#import <SUUpdateProduct.h>

void getSoftwareUpdateConfiguration(int os_version,
                                    int* isAutomaticallyCheckForUpdatesManaged,
                                    int* isAutomaticallyCheckForUpdatesEnabled,
                                    int* doesBackgroundDownload,
                                    int* doesAppStoreAutoUpdates,
                                    int* doesOSXAutoUpdates,
                                    int* doesAutomaticCriticalUpdateInstall,
                                    int* lastCheckTimestamp) {
  NSBundle* bundle;
  bundle = [NSBundle
      bundleWithPath:
          @"/System/Library/PrivateFrameworks/SoftwareUpdate.framework"];
  [bundle load];

  Class SUSharedPrefs = [bundle classNamed:@"SUSharedPrefs"];
  id manager = [SUSharedPrefs sharedPrefManager];

  BOOL val = [manager isAutomaticallyCheckForUpdatesManaged];
  if (val) {
    *isAutomaticallyCheckForUpdatesManaged = 1;
  }

  val = [manager isAutomaticallyCheckForUpdatesEnabled];
  if (val) {
    *isAutomaticallyCheckForUpdatesEnabled = 1;
  }

  val = [manager doesBackgroundDownload];
  if (val) {
    *doesBackgroundDownload = 1;
  }

  val = [manager doesAppStoreAutoUpdates];
  if (val) {
    *doesAppStoreAutoUpdates = 1;
  }

  // before 10.13 (build ver 17) it's called doesMacOSAutoUpdate.
  if (os_version >= 18) {
    val = [manager doesMacOSAutoUpdate];
    if (val) {
      *doesOSXAutoUpdates = 1;
    }
  } else {
    val = [manager doesOSXAutoUpdates];
    if (val) {
      *doesOSXAutoUpdates = 1;
    }
  }

  val = [manager doesAutomaticCriticalUpdateInstall];
  if (val) {
    *doesAutomaticCriticalUpdateInstall = 1;
  }
  NSDate* lastCheckSuccessfulDate = (NSDate*)[manager lastCheckSuccessfulDate];
  *lastCheckTimestamp = [lastCheckSuccessfulDate timeIntervalSince1970];
  return;
}

void getRecommendedUpdates() {
  NSBundle* bundle;
  bundle = [NSBundle
      bundleWithPath:
          @"/System/Library/PrivateFrameworks/SoftwareUpdate.framework"];
  [bundle load];

  Class SUSharedPrefs = [bundle classNamed:@"SUSharedPrefs"];
  id manager = [SUSharedPrefs sharedPrefManager];

  NSArray* updates = [manager recommendedUpdates];
  unsigned int i = 0;

  updatesFound([updates count]);

  for (id update in updates) {
    for (NSString* key in update) {
      NSString* value = [update objectForKey:key];
      updateKeyValueFound(
          i, (char*)[key UTF8String], (char*)[[value description] UTF8String]);
    }
    ++i;
  }

  return;
}

void getAvailableProducts() {
  NSBundle* bundle;
  bundle = [NSBundle
      bundleWithPath:
          @"/System/Library/PrivateFrameworks/SoftwareUpdate.framework"];
  [bundle load];

  Class SUScanController = [bundle classNamed:@"SUScanController"];
  id scanController = [SUScanController sharedScanController];

  dispatch_semaphore_t dsema = dispatch_semaphore_create(0);
  void (^replyBlock)(id) = ^(NSArray* products) {
    // refreshAvailableProductsInForeground will invoke this block once
    // completed, now we can signal the main thread
    dispatch_semaphore_signal(dsema);
  };

  // This ridiculously long function signature has been reverse engineered and
  // these argument values have been chosen primarily via trial and error
  [scanController refreshAvailableProductsInForeground:YES
                                      limitedToChanged:NO
                            evenIfConfigurationChanged:YES
                           initiatedByDeviceConnection:YES
                                  limitedToProductKeys:nil
                                 limitedToProductTypes:nil
                               forCurrentConfiguration:YES
                               distributionEnvironment:nil
                         distributionEvalutionMetainfo:nil
                                     installedPrinters:nil
                                preferredLocalizations:nil
                                                finish:replyBlock];

  dispatch_time_t timeout =
      dispatch_time(DISPATCH_TIME_NOW, (uint64_t)(3 * 60 * NSEC_PER_SEC));
  // Wait until either refreshAvailableProductsInForeground completes or timeout
  intptr_t err = dispatch_semaphore_wait(dsema, timeout);
  if (err != 0) {
    // Timed out waiting for results, nothing to do
    return;
  }

  id availableProducts = [scanController availableProducts];
  unsigned int i = 0;

  productsFound([availableProducts count]);

  for (id product in availableProducts) {
    Class SUUpdateProduct = [bundle classNamed:@"SUUpdateProduct"];
    id updateProduct = [[SUUpdateProduct alloc] initWithSUProduct:product];

    // Build a list of keys we're interested in getting the values of
    NSArray* keys =
        [NSArray arrayWithObjects:@"title",
                                  @"versionString",
                                  @"action",
                                  @"currentLocalization",
                                  @"productKey",
                                  @"serverState",
                                  @"type",
                                  @"auxInfo",
                                  @"identifierForProductLabel",
                                  @"versionForProductLabel",
                                  @"allowedToUseInstallLater",
                                  @"shouldAuthenticateReboot",
                                  @"isABaseSystemUpdate",
                                  @"isMajorOSUpdate",
                                  @"isMajorOSUpdateInternal",
                                  @"majorProduct",
                                  @"adminDeferred",
                                  @"adminDeferralDate",
                                  @"isFirmwareUpdate",
                                  @"productType",
                                  @"productBuildVersion",
                                  @"productVersion",
                                  @"autoUpdateEligible",
                                  @"postDate",
                                  @"deferredEnablementDate",
                                  @"updateInfo",
                                  @"shouldAutoInstallWithDelayInHours",
                                  nil];
    NSDictionary* dict = [updateProduct dictionaryWithValuesForKeys:keys];

    [dict enumerateKeysAndObjectsUsingBlock:^(id key, id object, BOOL* stop) {
      if ([object isKindOfClass:[NSDictionary class]]) {
        // This is a nested dictionary, build a nested object
        [object enumerateKeysAndObjectsUsingBlock:^(
                    id nestedKey, id nestedObject, BOOL* nestedStop) {
          // To keep things simple, only support one level of nesting
          productNestedKeyValueFound(
              i,
              (char*)[key UTF8String],
              (char*)[nestedKey UTF8String],
              (nestedObject == (id)[NSNull null])
                  ? NULL
                  : (char*)[[nestedObject description] UTF8String]);
        }];
      } else {
        // This is a basic key-value pair
        productKeyValueFound(i,
                             (char*)[key UTF8String],
                             (object == (id)[NSNull null])
                                 ? NULL
                                 : (char*)[[object description] UTF8String]);
      }
    }];

    ++i;
  }
}