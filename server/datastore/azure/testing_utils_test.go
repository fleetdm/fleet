package azure

import (
	"context"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/require"
)

// Well-known Azurite (the Azure Storage emulator) development account and
// key, documented at
// https://learn.microsoft.com/en-us/azure/storage/common/storage-use-azurite#well-known-storage-account-and-key.
const (
	testAccountName = "devstoreaccount1"
	testAccountKey  = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
)

var testContainerURL = func() string {
	port := "10000"
	if p := os.Getenv("FLEET_AZURITE_PORT"); p != "" {
		port = p
	}
	return "http://127.0.0.1:" + port + "/" + testAccountName
}()

func setupTestSoftwareInstallerStore(tb testing.TB, container string) *SoftwareInstallerStore {
	return setupTestStore(tb, container, NewSoftwareInstallerStore)
}

func setupTestBootstrapPackageStore(tb testing.TB, container string) *BootstrapPackageStore {
	return setupTestStore(tb, container, NewBootstrapPackageStore)
}

func setupTestOrgLogoStore(tb testing.TB, container string) *OrgLogoStore {
	return setupTestStore(tb, container, NewOrgLogoStore)
}

func setupTestSoftwareTitleIconStore(tb testing.TB, container string) *SoftwareTitleIconStore {
	return setupTestStore(tb, container, NewSoftwareTitleIconStore)
}

type testStoreIface interface {
	CreateTestContainer(ctx context.Context, name string) error
	CleanupTestContainer(ctx context.Context) error
}

func setupTestStore[T testStoreIface](tb testing.TB, container string, newFn func(config.AzureConfig) (T, error)) T {
	checkTestEnv(tb)

	store, err := newFn(config.AzureConfig{
		SoftwareInstallersAccountName:  testAccountName,
		SoftwareInstallersAccountKey:   testAccountKey,
		SoftwareInstallersContainer:    container,
		SoftwareInstallersContainerURL: testContainerURL,
	})
	require.NoError(tb, err)

	err = store.CreateTestContainer(context.Background(), container)
	require.NoError(tb, err)

	tb.Cleanup(func() {
		if err := store.CleanupTestContainer(context.Background()); err != nil {
			tb.Errorf("cleanup azure container %q: %v", container, err)
		}
	})

	return store
}

func checkTestEnv(tb testing.TB) {
	if _, ok := os.LookupEnv("AZURE_STORAGE_TEST"); !ok {
		tb.Skip("set AZURE_STORAGE_TEST environment variable to run Azure-based tests")
	}
}
