package datastore

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
)

func functionName(f func(*testing.T, kolide.Datastore)) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	elements := strings.Split(fullName, ".")
	return elements[len(elements)-1]
}

var testFunctions = [...]func(*testing.T, kolide.Datastore){
	testOrgInfo,
	testCreateInvite,
	testDeleteQuery,
	testSaveQuery,
	testDeletePack,
	testAddAndRemoveQueryFromPack,
	testEnrollHost,
	testAuthenticateHost,
	testLabels,
	testManagingLabelsOnPacks,
	testPasswordResetRequests,
	testCreateUser,
	testSaveUser,
	testUserByID,
	testPasswordResetRequests,
}
