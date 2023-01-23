// Go callbacks
extern void updatesFound(unsigned int);
extern void updateKeyValueFound(unsigned int, char*, char*);
extern void productsFound(unsigned int);
extern void productKeyValueFound(unsigned int, char*, char*);
extern void productNestedKeyValueFound(unsigned int, char*, char*, char*);

// Gets software update config flags from SUSharedPrefs API
void getSoftwareUpdateConfiguration(int os_version,
                                    int* isAutomaticallyCheckForUpdatesManaged,
                                    int* isAutomaticallyCheckForUpdatesEnabled,
                                    int* doesBackgroundDownload,
                                    int* doesAppStoreAutoUpdates,
                                    int* doesOSXAutoUpdates,
                                    int* doesAutomaticCriticalUpdateInstall,
                                    int* lastCheckTimestamp);

// Gets recommended updates from the SUSharedPrefs API
void getRecommendedUpdates();

// Gets the available products via the SUScanController API
void getAvailableProducts();