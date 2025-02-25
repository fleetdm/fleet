package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestMacosSetupAssistant(t *testing.T) {
	ctx := context.Background()
	ds := mysql.CreateMySQLDS(t)
	// call TruncateTables immediately as some DB migrations may create jobs
	mysql.TruncateTables(t, ds)

	org1Name := "org1"
	org2Name := "org2"

	mysql.SetTestABMAssets(t, ds, "fleet")
	tok := mysql.CreateAndSetABMToken(t, ds, org1Name)
	tok2 := mysql.CreateAndSetABMToken(t, ds, org2Name)

	// create a couple hosts for no team, team 1 and team 2 (none for team 3)
	hosts := make([]*fleet.Host, 6)
	for i := 0; i < len(hosts); i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:       fmt.Sprintf("test-host%d-name", i),
			OsqueryHostID:  ptr.String(fmt.Sprintf("osquery-%d", i)),
			NodeKey:        ptr.String(fmt.Sprintf("nodekey-%d", i)),
			UUID:           fmt.Sprintf("test-uuid-%d", i),
			Platform:       "darwin",
			HardwareSerial: fmt.Sprintf("serial-%d", i),
		})
		require.NoError(t, err)

		tokID := tok.ID
		if i%2 == 0 {
			tokID = tok2.ID
		}

		err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h}, tokID)
		require.NoError(t, err)
		hosts[i] = h
		t.Logf("host [%d]: %s - %s - %d", i, h.UUID, h.HardwareSerial, tokID)
	}

	// create teams
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	// hosts[0, 1] are no-team, hosts[2, 3] are team1, hosts[4, 5] are team2
	err = ds.AddHostsToTeam(ctx, &tm1.ID, []uint{hosts[2].ID, hosts[3].ID})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, &tm2.ID, []uint{hosts[4].ID, hosts[5].ID})
	require.NoError(t, err)

	logger := kitlog.NewNopLogger()
	depStorage, err := ds.NewMDMAppleDEPStorage()
	require.NoError(t, err)
	macosJob := &MacosSetupAssistant{
		Datastore:  ds,
		Log:        logger,
		DEPService: apple_mdm.NewDEPService(ds, depStorage, logger),
		DEPClient:  apple_mdm.NewDEPClient(depStorage, ds, logger),
	}

	const defaultProfileName = "Fleet default enrollment profile"

	// track the profile assigned to each device
	serialsToProfile := map[string]string{
		"serial-0": "",
		"serial-1": "",
		"serial-2": "",
		"serial-3": "",
		"serial-4": "",
		"serial-5": "",
	}

	// start the web server that will mock Apple DEP responses
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "auth123"})
			require.NoError(t, err)

		case "/profile":
			var reqProf godep.Profile
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			err = json.Unmarshal(b, &reqProf)
			require.NoError(t, err)

			// use the profile name as profile uuid, and append "+sso" if it was
			// registered with the sso url (end-user auth enabled).
			profUUID := reqProf.ProfileName
			if strings.HasSuffix(reqProf.ConfigurationWebURL, "/mdm/sso") {
				profUUID += "+sso"
			}
			err = encoder.Encode(godep.ProfileResponse{ProfileUUID: profUUID})
			require.NoError(t, err)

		case "/profile/devices":
			var reqProf godep.Profile
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			err = json.Unmarshal(b, &reqProf)
			require.NoError(t, err)

			for _, d := range reqProf.Devices {
				serialsToProfile[d] = reqProf.ProfileUUID
			}
			_, _ = w.Write([]byte(`{}`))

		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	err = depStorage.StoreConfig(ctx, "fleet", &nanodep_client.Config{BaseURL: srv.URL})
	require.NoError(t, err)
	err = depStorage.StoreConfig(ctx, org1Name, &nanodep_client.Config{BaseURL: srv.URL})
	require.NoError(t, err)
	err = depStorage.StoreConfig(ctx, org2Name, &nanodep_client.Config{BaseURL: srv.URL})
	require.NoError(t, err)

	w := NewWorker(ds, logger)
	w.Register(macosJob)

	runCheckDone := func() {
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)
		// no remaining jobs to process
		pending, err := ds.GetQueuedJobs(ctx, 10, time.Time{})
		require.NoError(t, err)
		require.Empty(t, pending)
	}

	// no jobs to process yet
	runCheckDone()

	// no default profile registered yet
	_, err = ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	require.ErrorIs(t, err, sql.ErrNoRows)

	start := time.Now().Truncate(time.Second)

	// enqueue a regenerate all and process the jobs
	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantUpdateAllProfiles, nil)
	require.NoError(t, err)
	runCheckDone()

	// all devices are assigned the default profile
	autoProf, err := ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	require.NoError(t, err)
	require.NotEmpty(t, autoProf.Token)

	getTeamID := func(tmID *uint) string {
		if tmID == nil {
			return "null"
		}

		return fmt.Sprint(*tmID)
	}

	tmIDs := []*uint{nil, ptr.Uint(tm1.ID), ptr.Uint(tm2.ID)}
	for _, tmID := range tmIDs {
		for _, org := range []string{org1Name, org2Name} {
			profUUID, modTime, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, tmID, org)
			require.NoError(t, err)
			require.Equal(t, defaultProfileName, profUUID, "tmID", getTeamID(tmID))
			require.False(t, modTime.Before(start))
		}
	}
	require.Equal(t, map[string]string{
		"serial-0": defaultProfileName,
		"serial-1": defaultProfileName,
		"serial-2": defaultProfileName,
		"serial-3": defaultProfileName,
		"serial-4": defaultProfileName,
		"serial-5": defaultProfileName,
	}, serialsToProfile)

	// create a custom setup assistant for team 1 and process the job
	tm1Asst, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
		TeamID:  &tm1.ID,
		Name:    "team1",
		Profile: json.RawMessage(`{"profile_name": "team1"}`),
	})
	require.NoError(t, err)
	require.NotZero(t, tm1Asst.ID)
	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantProfileChanged, &tm1.ID)
	require.NoError(t, err)
	runCheckDone()

	// default profile is unchanged
	for _, tmID := range tmIDs {
		for _, org := range []string{org1Name, org2Name} {
			profUUID, modTime, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, tmID, org)
			require.NoError(t, err)
			require.Equal(t, defaultProfileName, profUUID)
			require.False(t, modTime.Before(start))
		}
	}

	// team 1 setup assistant is registered for both tokens
	tm1Asst, err = ds.GetMDMAppleSetupAssistant(ctx, &tm1.ID)
	require.NoError(t, err)
	require.NotNil(t, tm1Asst)
	for _, org := range []string{org1Name, org2Name} {
		profUUID, modTime, err := ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm1.ID, org)
		require.NoError(t, err)
		require.Equal(t, "team1", profUUID)
		require.False(t, modTime.Before(start))
	}

	require.Equal(t, map[string]string{
		"serial-0": defaultProfileName,
		"serial-1": defaultProfileName,
		"serial-2": "team1",
		"serial-3": "team1",
		"serial-4": defaultProfileName,
		"serial-5": defaultProfileName,
	}, serialsToProfile)

	// enable end-user auth for team 2
	tm2.Config.MDM.MacOSSetup.EnableEndUserAuthentication = true
	tm2, err = ds.SaveTeam(ctx, tm2)
	require.NoError(t, err)

	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantUpdateProfile, &tm2.ID)
	require.NoError(t, err)
	runCheckDone()

	require.Equal(t, map[string]string{
		"serial-0": defaultProfileName,
		"serial-1": defaultProfileName,
		"serial-2": "team1",
		"serial-3": "team1",
		"serial-4": defaultProfileName + "+sso",
		"serial-5": defaultProfileName + "+sso",
	}, serialsToProfile)

	// create a custom setup assistant for teams 2 and 3, delete the one for team 1 and process the jobs
	tm2Asst, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
		TeamID:  &tm2.ID,
		Name:    "team2",
		Profile: json.RawMessage(`{"profile_name": "team2"}`),
	})
	require.NoError(t, err)
	require.NotZero(t, tm2Asst.ID)
	tm3Asst, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
		TeamID:  &tm3.ID,
		Name:    "team3",
		Profile: json.RawMessage(`{"profile_name": "team3"}`),
	})
	require.NoError(t, err)
	require.NotZero(t, tm3Asst.ID)
	err = ds.DeleteMDMAppleSetupAssistant(ctx, &tm1.ID)
	require.NoError(t, err)
	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantProfileChanged, &tm2.ID)
	require.NoError(t, err)
	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantProfileChanged, &tm3.ID)
	require.NoError(t, err)
	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantProfileDeleted, &tm1.ID)
	require.NoError(t, err)
	runCheckDone()

	require.Equal(t, map[string]string{
		"serial-0": defaultProfileName,
		"serial-1": defaultProfileName,
		"serial-2": defaultProfileName,
		"serial-3": defaultProfileName,
		"serial-4": "team2+sso",
		"serial-5": "team2+sso",
	}, serialsToProfile)

	// disable end-user auth for team 2
	tm2.Config.MDM.MacOSSetup.EnableEndUserAuthentication = false
	tm2, err = ds.SaveTeam(ctx, tm2)
	require.NoError(t, err)

	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantUpdateProfile, &tm2.ID)
	require.NoError(t, err)
	runCheckDone()

	require.Equal(t, map[string]string{
		"serial-0": defaultProfileName,
		"serial-1": defaultProfileName,
		"serial-2": defaultProfileName,
		"serial-3": defaultProfileName,
		"serial-4": "team2",
		"serial-5": "team2",
	}, serialsToProfile)

	// move hosts[2,4] to team 3, delete team 2
	err = ds.AddHostsToTeam(ctx, &tm3.ID, []uint{hosts[2].ID, hosts[4].ID})
	require.NoError(t, err)
	err = ds.DeleteTeam(ctx, tm2.ID)
	require.NoError(t, err)

	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantHostsTransferred, &tm3.ID, "serial-2", "serial-4")
	require.NoError(t, err)
	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantTeamDeleted, nil, "serial-5") // hosts[5] was in team 2
	require.NoError(t, err)
	runCheckDone()

	require.Equal(t, map[string]string{
		"serial-0": defaultProfileName,
		"serial-1": defaultProfileName,
		"serial-2": "team3",
		"serial-3": defaultProfileName,
		"serial-4": "team3",
		"serial-5": defaultProfileName,
	}, serialsToProfile)

	// create setup assistant for no-team
	noTmAsst, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
		TeamID:  nil,
		Name:    "no-team",
		Profile: json.RawMessage(`{"profile_name": "no-team"}`),
	})
	require.NoError(t, err)
	require.NotZero(t, noTmAsst.ID)

	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantProfileChanged, nil)
	require.NoError(t, err)
	runCheckDone()

	require.Equal(t, map[string]string{
		"serial-0": "no-team",
		"serial-1": "no-team",
		"serial-2": "team3",
		"serial-3": defaultProfileName,
		"serial-4": "team3",
		"serial-5": "no-team", // became a no-team host when team2 got deleted
	}, serialsToProfile)

	// check that profiles get re-generated (note that timestamps are not
	// impacted as the content of the profiles did not change)
	_, err = QueueMacosSetupAssistantJob(ctx, ds, logger, MacosSetupAssistantUpdateAllProfiles, nil)
	require.NoError(t, err)
	runCheckDone()

	// team 2 got deleted, update the list of team IDs
	tmIDs = []*uint{nil, ptr.Uint(tm1.ID), ptr.Uint(tm3.ID)}
	for i, tmID := range tmIDs {
		for _, org := range []string{org1Name, org2Name} {
			// no team and team 3 have a custom setup assistant
			switch i {
			case 0: // no team
				// custom profile defined for both orgs
				_, _, err := ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, tmID, org)
				require.NoError(t, err, "%v - %v", i, org)
			case 1: // tm1
				// team 1 uses the default setup assistant, and it is only defined for org1
				_, _, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, tmID, org)
				if org == org1Name {
					require.NoError(t, err, "%v - %v", i, org)
				} else {
					require.ErrorIs(t, err, sql.ErrNoRows, "%v - %v", i, org)
				}
			case 2: // tm3
				_, _, err := ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, tmID, org)
				// custom setup assistant only defined for org2
				if org == org2Name {
					require.NoError(t, err, "%v - %v", i, org)
				} else {
					require.ErrorIs(t, err, sql.ErrNoRows, "%v - %v", i, org)
				}
			}
		}
	}

	require.Equal(t, map[string]string{
		"serial-0": "no-team",
		"serial-1": "no-team",
		"serial-2": "team3",
		"serial-3": defaultProfileName,
		"serial-4": "team3",
		"serial-5": "no-team", // became a no-team host when team2 got deleted
	}, serialsToProfile)
}
