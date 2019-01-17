package datastore

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kolide/fleet/server/kolide"
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
	testSaveLabel,
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
	testListHostsInPack,
	testListPacksForHost,
	testHostIDsByName,
	testListPacks,
	testDistributedQueryCampaign,
	testCleanupDistributedQueryCampaigns,
	testBuiltInLabels,
	testLoadPacksForQueries,
	testScheduledQuery,
	testDeleteScheduledQuery,
	testNewScheduledQuery,
	testListScheduledQueriesInPack,
	testCascadingDeletionOfQueries,
	testOptions,
	testOptionsToConfig,
	testGetPackByName,
	testGetQueryByName,
	testFileIntegrityMonitoring,
	testYARAStore,
	testAddLabelToPackTwice,
	testGenerateHostStatusStatistics,
	testMarkHostSeen,
	testDuplicateNewQuery,
	testIdempotentDeleteHost,
	testChangeEmail,
	testChangeLabelDetails,
	testFlappingNetworkInterfaces,
	testMigrationStatus,
	testUnicode,
	testCountHostsInTargets,
	testHostStatus,
	testResetOptions,
	testApplyOsqueryOptions,
	testApplyOsqueryOptionsNoOverrides,
	testOsqueryOptionsForHost,
	testApplyQueries,
	testApplyPackSpecRoundtrip,
	testApplyPackSpecMissingQueries,
	testApplyPackSpecMissingName,
	testGetPackSpec,
	testApplyLabelSpecsRoundtrip,
	testGetLabelSpec,
	testLabelIDsByName,
	testListLabelsForPack,
}
