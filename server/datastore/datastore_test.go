package datastore

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kolide/kolide/server/kolide"
)

func functionName(f func(*testing.T, kolide.Datastore)) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	elements := strings.Split(fullName, ".")
	return elements[len(elements)-1]
}

var testFunctions = [...]func(*testing.T, kolide.Datastore){
	testOrgInfo,
	testCreateInvite,
	testInviteByEmail,
	testInviteByToken,
	testListInvites,
	testDeleteInvite,
	testSaveInvite,
	testDeleteQuery,
	testDeleteQueries,
	testSaveQuery,
	testListQuery,
	testDeletePack,
	testEnrollHost,
	testAuthenticateHost,
	testLabels,
	testManagingLabelsOnPacks,
	testPasswordResetRequests,
	testCreateUser,
	testSaveUser,
	testUserByID,
	testPasswordResetRequests,
	testSearchHosts,
	testSearchHostsLimit,
	testSearchLabels,
	testSearchLabelsLimit,
	testListHostsInLabel,
	testListUniqueHostsInLabels,
	testDistributedQueriesForHost,
	testSaveHosts,
	testDeleteHost,
	testListHost,
	testGetHostsInPack,
	testDistributedQueryCampaign,
	testCleanupDistributedQueryCampaigns,
	testBuiltInLabels,
	testLoadPacksForQueries,
	testScheduledQuery,
	testDeleteScheduledQuery,
	testListScheduledQueriesInPack,
	testSaveScheduledQuery,
	testOptions,
	testNewScheduledQuery,
	testOptionsToConfig,
	testGetPackByName,
	testGetQueryByName,
	testDecorators,
	testFileIntegrityMonitoring,
	testYARAStore,
	testAddLabelToPackTwice,
	testGenerateHostStatusStatistics,
	testMarkHostSeen,
	testDuplicateNewQuery,
	testIdempotentDeleteHost,
	testChangeEmail,
	testLicense,
	testSaveLabel,
	testFlappingNetworkInterfaces,
	testReplaceDeletedLabel,
	testMigrationStatus,
	testUnicode,
	testCountHostsInTargets,
	testResetOptions,
	testIdentityProvider,
}
