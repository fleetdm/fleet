package service

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestNewTeamNameValidation(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		team.ID = 1
		return team, nil
	}
	ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
		return nil, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	svc := &Service{
		Service: mockSvc,
		ds:      ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: "something"},
		},
		authz: authorizer,
	}

	adminUser := &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	ctx := test.UserContext(context.Background(), adminUser)

	testCases := []struct {
		name     string
		teamName *string
		wantErr  string
		wantName string
	}{
		{
			name:    "nil name",
			wantErr: "missing required argument",
		},
		{
			name:     "empty string",
			teamName: ptr.String(""),
			wantErr:  "may not be empty",
		},
		{
			name:     "only spaces",
			teamName: ptr.String("     "),
			wantErr:  "may not be empty",
		},
		{
			name:     "only tabs",
			teamName: ptr.String("\t\t\t"),
			wantErr:  "may not be empty",
		},
		{
			name:     "only newlines",
			teamName: ptr.String("\n\n\n"),
			wantErr:  "may not be empty",
		},
		{
			name:     "only carriage returns",
			teamName: ptr.String("\r\r\r"),
			wantErr:  "may not be empty",
		},
		{
			name:     "mixed whitespace",
			teamName: ptr.String(" \t\n\r "),
			wantErr:  "may not be empty",
		},
		{
			name:     "single space",
			teamName: ptr.String(" "),
			wantErr:  "may not be empty",
		},
		{
			name:     "single tab",
			teamName: ptr.String("\t"),
			wantErr:  "may not be empty",
		},
		{
			name:     "single newline",
			teamName: ptr.String("\n"),
			wantErr:  "may not be empty",
		},
		{
			name:     "leading spaces are trimmed",
			teamName: ptr.String("   myteam"),
			wantName: "myteam",
		},
		{
			name:     "trailing spaces are trimmed",
			teamName: ptr.String("myteam   "),
			wantName: "myteam",
		},
		{
			name:     "inner spaces preserved",
			teamName: ptr.String("my team"),
			wantName: "my team",
		},
		{
			name:     "leading and trailing trimmed with inner preserved",
			teamName: ptr.String("  my team  "),
			wantName: "my team",
		},
		{
			name:     "valid name no trimming needed",
			teamName: ptr.String("Engineering"),
			wantName: "Engineering",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := fleet.TeamPayload{Name: tc.teamName}

			team, err := svc.NewTeam(ctx, payload)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				require.Nil(t, team)
			} else {
				require.NoError(t, err)
				require.NotNil(t, team)
				require.Equal(t, tc.wantName, team.Name)
			}
		})
	}
}

func TestModifyTeamNameValidation(t *testing.T) {
	ds := new(mock.Store)
	ds.TeamWithExtrasFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "existing-team"}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}
	ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
		return nil, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	svc := &Service{
		Service: mockSvc,
		ds:      ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: "something"},
		},
		authz: authorizer,
	}

	adminUser := &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	ctx := test.UserContext(context.Background(), adminUser)

	testCases := []struct {
		name     string
		teamName *string
		wantErr  string
		wantName string
	}{
		{
			name:     "only spaces",
			teamName: ptr.String("     "),
			wantErr:  "may not be empty",
		},
		{
			name:     "only tabs",
			teamName: ptr.String("\t\t\t"),
			wantErr:  "may not be empty",
		},
		{
			name:     "only newlines",
			teamName: ptr.String("\n\n\n"),
			wantErr:  "may not be empty",
		},
		{
			name:     "only carriage returns",
			teamName: ptr.String("\r\r\r"),
			wantErr:  "may not be empty",
		},
		{
			name:     "mixed whitespace",
			teamName: ptr.String(" \t\n\r "),
			wantErr:  "may not be empty",
		},
		{
			name:     "empty string",
			teamName: ptr.String(""),
			wantErr:  "may not be empty",
		},
		{
			name:     "single space",
			teamName: ptr.String(" "),
			wantErr:  "may not be empty",
		},
		{
			name:     "nil name keeps existing name",
			wantName: "existing-team",
		},
		{
			name:     "valid name",
			teamName: ptr.String("new-name"),
			wantName: "new-name",
		},
		{
			name:     "leading spaces are trimmed",
			teamName: ptr.String("  new-name"),
			wantName: "new-name",
		},
		{
			name:     "trailing spaces are trimmed",
			teamName: ptr.String("new-name  "),
			wantName: "new-name",
		},
		{
			name:     "inner spaces preserved",
			teamName: ptr.String("my team"),
			wantName: "my team",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := fleet.TeamPayload{Name: tc.teamName}

			team, err := svc.ModifyTeam(ctx, 1, payload)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				require.Nil(t, team)
			} else {
				require.NoError(t, err)
				require.NotNil(t, team)
				require.Equal(t, tc.wantName, team.Name)
			}
		})
	}
}

func TestApplyTeamSpecsNameValidation(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, &notFoundError{}
	}
	ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
		return nil, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	svc := &Service{
		ds: ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: "something"},
		},
		authz: authorizer,
	}

	adminUser := &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	ctx := test.UserContext(context.Background(), adminUser)

	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, newB bool, teamID *uint) (bool, error) {
		return true, nil
	}

	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	svc.Service = mockSvc

	testCases := []struct {
		name     string
		teamName string
		wantErr  string
		wantName string
	}{
		{
			name:     "empty string",
			teamName: "",
			wantErr:  "may not be empty",
		},
		{
			name:     "only spaces",
			teamName: "     ",
			wantErr:  "may not be empty",
		},
		{
			name:     "only tabs",
			teamName: "\t\t\t",
			wantErr:  "may not be empty",
		},
		{
			name:     "only newlines",
			teamName: "\n\n\n",
			wantErr:  "may not be empty",
		},
		{
			name:     "only carriage returns",
			teamName: "\r\r\r",
			wantErr:  "may not be empty",
		},
		{
			name:     "mixed whitespace",
			teamName: " \t\n\r ",
			wantErr:  "may not be empty",
		},
		{
			name:     "single space",
			teamName: " ",
			wantErr:  "may not be empty",
		},
		{
			name:     "valid name",
			teamName: "Engineering",
			wantName: "Engineering",
		},
		{
			name:     "leading and trailing spaces are trimmed",
			teamName: "  Engineering  ",
			wantName: "Engineering",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
				{Name: tc.teamName},
			}, fleet.ApplyTeamSpecOptions{ApplySpecOptions: fleet.ApplySpecOptions{DryRun: true}})
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				require.Contains(t, result, tc.wantName)
			}
		})
	}
}

// TestNewTeamCollationEqualConflict covers the case where the requested name
// collides with an existing team under MySQL's collation.
func TestNewTeamCollationEqualConflict(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		t.Fatalf("NewTeam should not be called when a conflict is detected")
		return nil, nil
	}
	ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
		require.Equal(t, uint(0), excludeID)
		return &fleet.Team{ID: 42, Name: "ABC"}, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	svc := &Service{
		Service: mockSvc,
		ds:      ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: "something"},
		},
		authz: authorizer,
	}

	adminUser := &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}
	ctx := test.UserContext(context.Background(), adminUser)

	_, err = svc.NewTeam(ctx, fleet.TeamPayload{Name: new("abc")})
	require.Error(t, err)
	var conflict *fleet.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Contains(t, err.Error(), `"ABC"`)
	require.Contains(t, err.Error(), "must differ by more than letter case")
}

// TestModifyTeamCaseOnlyRenameAndConflict covers two ModifyTeam scenarios:
//  1. Case-only self-rename succeeds (the team is excluded from the conflict
//     check by id).
//  2. Rename into another team's name returns a ConflictError naming that
//     team.
func TestModifyTeamCaseOnlyRenameAndConflict(t *testing.T) {
	ds := new(mock.Store)
	ds.TeamWithExtrasFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "ABC"}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	svc := &Service{
		Service: mockSvc,
		ds:      ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: "something"},
		},
		authz: authorizer,
	}

	adminUser := &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}
	ctx := test.UserContext(context.Background(), adminUser)

	t.Run("case-only self rename succeeds", func(t *testing.T) {
		ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
			require.Equal(t, uint(5), excludeID)
			return nil, nil
		}

		team, err := svc.ModifyTeam(ctx, 5, fleet.TeamPayload{Name: new("abc")})
		require.NoError(t, err)
		require.NotNil(t, team)
		require.Equal(t, "abc", team.Name)
	})

	t.Run("rename into another team's name conflicts", func(t *testing.T) {
		ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
			require.Equal(t, uint(5), excludeID)
			return &fleet.Team{ID: 6, Name: "def"}, nil
		}

		team, err := svc.ModifyTeam(ctx, 5, fleet.TeamPayload{Name: new("DEF")})
		require.Error(t, err)
		require.Nil(t, team)
		var conflict *fleet.ConflictError
		require.ErrorAs(t, err, &conflict)
		require.Contains(t, err.Error(), `"def"`)
		require.Contains(t, err.Error(), "must differ by more than letter case")
	})
}

// TestApplyTeamSpecsCollationEqualConflict covers the three GitOps scenarios
// that were inconsistently handled:
//   - Single-team case-only rename should succeed.
//   - Cross-file conflict (new filename, colliding name) should return
//     ConflictError.
//   - Intra-batch conflict should be detected before any DB writes.
func TestApplyTeamSpecsCollationEqualConflict(t *testing.T) {
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	adminUser := &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}
	ctx := test.UserContext(context.Background(), adminUser)

	newSvc := func() (*Service, *mock.Store) {
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, newB bool, teamID *uint) (bool, error) {
			return true, nil
		}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
			return nil, &notFoundError{}
		}
		ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
			return nil, &notFoundError{}
		}
		ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
			return nil, nil
		}

		mockSvc := &svcmock.Service{}
		mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}
		svc := &Service{
			Service: mockSvc,
			ds:      ds,
			config: config.FleetConfig{
				Server: config.ServerConfig{PrivateKey: "something"},
			},
			authz:  authorizer,
			logger: slog.New(slog.DiscardHandler),
		}
		return svc, ds
	}

	t.Run("case-only rename of existing team succeeds and persists new name", func(t *testing.T) {
		svc, ds := newSvc()
		filename := "abc.yml"
		existing := &fleet.Team{ID: 7, Name: "ABC", Filename: new(filename)}
		ds.TeamByFilenameFunc = func(ctx context.Context, f string) (*fleet.Team, error) {
			require.Equal(t, filename, f)
			return existing, nil
		}
		conflictCalls := 0
		ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
			conflictCalls++
			require.Equal(t, uint(7), excludeID,
				"conflict check must exclude the team matched by filename so a case-only rename succeeds")
			return nil, nil
		}
		var savedTeam *fleet.Team
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			savedTeam = team
			return team, nil
		}

		// Run without DryRun so editTeamFromSpec actually rewrites team.Name
		// and SaveTeam is called — otherwise we'd only be asserting that the
		// conflict check didn't trip.
		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
			{Name: "abc", Filename: new(filename)},
		}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.Equal(t, 1, conflictCalls, "TeamConflictsWithName must be called once per spec")
		require.True(t, ds.SaveTeamFuncInvoked, "SaveTeam must be called to persist the rename")
		require.NotNil(t, savedTeam)
		require.Equal(t, "abc", savedTeam.Name, "rename must persist the spec's new case form")
		require.Equal(t, uint(7), savedTeam.ID, "rename must target the same team id")
	})

	t.Run("filename-matched rename into another team's name conflicts", func(t *testing.T) {
		// Regression: team "ABC" is managed by "abc.yml". User tries to
		// rename it to "DEF" via the same file, but another team "def"
		// already exists under a different file. This must 409.
		svc, ds := newSvc()
		existing := &fleet.Team{ID: 7, Name: "ABC", Filename: new("abc.yml")}
		ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
			return existing, nil
		}
		ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
			require.Equal(t, uint(7), excludeID, "must exclude the filename-matched team")
			return &fleet.Team{ID: 8, Name: "def"}, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			t.Fatalf("SaveTeam must not be called when a conflict is detected")
			return nil, nil
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
			{Name: "DEF", Filename: new("abc.yml")},
		}, fleet.ApplyTeamSpecOptions{ApplySpecOptions: fleet.ApplySpecOptions{DryRun: true}})
		require.Error(t, err)
		var conflict *fleet.ConflictError
		require.ErrorAs(t, err, &conflict)
		require.Contains(t, err.Error(), `"def"`)
		require.Contains(t, err.Error(), "must differ by more than letter case")
	})

	t.Run("adopt existing team via new filename succeeds", func(t *testing.T) {
		// Regression for TestIntegrationsEnterpriseGitops: a spec with a new
		// filename that matches an existing team by name must adopt it
		// (possibly taking over management from another YAML file). The
		// pre-fix behavior was adoption; the fix must not break it.
		svc, ds := newSvc()
		existing := &fleet.Team{ID: 12, Name: "Adoptable", Filename: new("old.yml")}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
			return existing, nil
		}
		var savedTeam *fleet.Team
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			savedTeam = team
			return team, nil
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
			{Name: "Adoptable", Filename: new("new.yml")},
		}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked, "SaveTeam must be called to adopt the team")
		require.NotNil(t, savedTeam)
		require.Equal(t, uint(12), savedTeam.ID)
		require.NotNil(t, savedTeam.Filename)
		require.Equal(t, "new.yml", *savedTeam.Filename, "adoption must set the new filename")
	})

	t.Run("intra-batch conflict short-circuits before any DB conflict check", func(t *testing.T) {
		svc, ds := newSvc()
		ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
			t.Fatalf("TeamConflictsWithName must not be called when the pre-pass catches the conflict (got name=%q, excludeID=%d)", name, excludeID)
			return nil, nil
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
			{Name: "ABC", Filename: new("foo.yml")},
			{Name: "abc", Filename: new("bar.yml")},
		}, fleet.ApplyTeamSpecOptions{ApplySpecOptions: fleet.ApplySpecOptions{DryRun: true}})
		require.Error(t, err)
		var conflict *fleet.ConflictError
		require.ErrorAs(t, err, &conflict)
		require.Contains(t, err.Error(), "foo.yml")
		require.Contains(t, err.Error(), "bar.yml")
		require.Contains(t, err.Error(), "must differ by more than letter case")
	})

	t.Run("no-filename spec with collation-equal name preserves DB canonical name", func(t *testing.T) {
		// Regression: without a filename, a spec whose name is a case variant
		// of an existing team must NOT silently rename that team. The DB's
		// canonical form wins; users who want to rename must supply a
		// filename.
		svc, ds := newSvc()
		existing := &fleet.Team{ID: 11, Name: "Workstations"}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
			// TeamByName is collation-aware in production, so "workstations"
			// matches "Workstations" here.
			return existing, nil
		}
		var savedTeam *fleet.Team
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			savedTeam = team
			return team, nil
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
			{Name: "workstations"}, // no filename
		}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.NotNil(t, savedTeam)
		require.Equal(t, "Workstations", savedTeam.Name,
			"no-filename spec must preserve the DB's canonical name, not silently case-rename it")
	})
}

// TestModifyTeamMDMEnableDiskEncryption covers the team-level PATCH endpoint
// validation for `mdm.enable_disk_encryption`. The flag governs both FileVault
// (Apple) and BitLocker (Windows) enforcement, so the change must be allowed
// when either platform's MDM is configured. Issue #44194 reported that the
// previous validation gated solely on Apple MDM and rejected Windows-only
// deployments.
func TestModifyTeamMDMEnableDiskEncryption(t *testing.T) {
	testCases := []struct {
		name                 string
		appleEnabled         bool
		windowsEnabled       bool
		wantErr              string
		wantFileVaultProfile bool
	}{
		{
			name:                 "windows MDM only succeeds without invoking FileVault (issue #44194)",
			appleEnabled:         false,
			windowsEnabled:       true,
			wantFileVaultProfile: false,
		},
		{
			name:           "neither MDM platform configured rejects the change",
			appleEnabled:   false,
			windowsEnabled: false,
			wantErr:        "mdm.enable_disk_encryption",
		},
		{
			name:                 "apple MDM only invokes FileVault profile creation",
			appleEnabled:         true,
			windowsEnabled:       false,
			wantFileVaultProfile: true,
		},
		{
			name:                 "both MDM platforms configured invokes FileVault profile creation",
			appleEnabled:         true,
			windowsEnabled:       true,
			wantFileVaultProfile: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						EnabledAndConfigured:        tc.appleEnabled,
						WindowsEnabledAndConfigured: tc.windowsEnabled,
					},
				}, nil
			}
			ds.TeamWithExtrasFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
				return &fleet.Team{
					ID:     tid,
					Name:   "team-1",
					Config: fleet.TeamConfig{MDM: fleet.TeamMDM{EnableDiskEncryption: false}},
				}, nil
			}
			var savedTeam *fleet.Team
			ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
				savedTeam = team
				return team, nil
			}
			// CA cert + profile mocks are exercised by the FileVault path when
			// Apple MDM is configured. Both are wired regardless because the
			// invocation flag is what we assert against.
			ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, _ []fleet.MDMAssetName,
				_ sqlx.QueryerContext,
			) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
				return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
					fleet.MDMAssetCACert: {Value: []byte(testCert)},
				}, nil
			}
			ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile,
				_ []fleet.FleetVarName,
			) (*fleet.MDMAppleConfigProfile, error) {
				return &p, nil
			}

			authorizer, err := authz.NewAuthorizer()
			require.NoError(t, err)

			mockSvc := &svcmock.Service{}
			mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
				return nil
			}

			svc := &Service{
				Service: mockSvc,
				ds:      ds,
				config: config.FleetConfig{
					Server: config.ServerConfig{PrivateKey: "something"},
				},
				authz: authorizer,
			}

			adminUser := &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)}
			ctx := test.UserContext(context.Background(), adminUser)

			payload := fleet.TeamPayload{
				MDM: &fleet.TeamPayloadMDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
			}
			team, err := svc.ModifyTeam(ctx, 1, payload)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				require.Nil(t, team)
				require.False(t, ds.SaveTeamFuncInvoked, "team should not have been saved")
				require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked, "FileVault profile should not have been created")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, team)
			require.True(t, team.Config.MDM.EnableDiskEncryption, "team payload should reflect EnableDiskEncryption=true")
			require.NotNil(t, savedTeam)
			require.True(t, savedTeam.Config.MDM.EnableDiskEncryption, "EnableDiskEncryption must be persisted")
			require.Equal(t, tc.wantFileVaultProfile, ds.NewMDMAppleConfigProfileFuncInvoked,
				"FileVault profile creation should match Apple MDM configuration")
		})
	}
}

func TestUpdateTeamMDMDiskEncryption(t *testing.T) {
	testCases := []struct {
		name           string
		mdmConfig      fleet.TeamMDM
		diskEncryption *bool
		requireTPMPIN  *bool
		expectedError  string
	}{
		{
			name: "try to disable disk encryption with TPM PIN enabled",
			mdmConfig: fleet.TeamMDM{
				EnableDiskEncryption: true,
				RequireBitLockerPIN:  true,
			},
			diskEncryption: ptr.Bool(false),
			requireTPMPIN:  ptr.Bool(true),

			expectedError: fleet.CantDisableDiskEncryptionIfPINRequiredErrMsg,
		},
		{
			name: "try to enable disk encryption with TPM PIN enabled",
			mdmConfig: fleet.TeamMDM{
				EnableDiskEncryption: false,
				RequireBitLockerPIN:  false,
			},
			diskEncryption: ptr.Bool(false),
			requireTPMPIN:  ptr.Bool(true),
			expectedError:  fleet.CantEnablePINRequiredIfDiskEncryptionEnabled,
		},
		{
			name: "try to disable disk encryption with TPM PIN enabled when disk encryption prev enabled",
			mdmConfig: fleet.TeamMDM{
				EnableDiskEncryption: true,
				RequireBitLockerPIN:  false,
			},
			diskEncryption: ptr.Bool(false),
			requireTPMPIN:  ptr.Bool(true),
			expectedError:  fleet.CantDisableDiskEncryptionIfPINRequiredErrMsg,
		},
	}

	ds := new(mock.Store)

	svc := &Service{
		ds: ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: "something",
			},
		},
	}

	ctx := context.Background()

	for _, tC := range testCases {
		team := fleet.Team{
			Config: fleet.TeamConfig{
				MDM: tC.mdmConfig,
			},
		}

		err := svc.updateTeamMDMDiskEncryption(
			ctx,
			&team,
			tC.diskEncryption,
			tC.requireTPMPIN,
		)

		if tC.expectedError != "" {
			require.NotNil(t, err)
			require.True(
				t,
				strings.Contains(err.Error(), tC.expectedError),
				"Expected '%s' to contain '%s'",
				err.Error(), tC.expectedError)
		}
	}
}

func TestObfuscateSecrets(t *testing.T) {
	buildTeams := func(n int) []*fleet.Team {
		r := make([]*fleet.Team, 0, n)
		for i := 1; i <= n; i++ {
			r = append(r, &fleet.Team{
				ID: uint(i), //nolint:gosec // dismiss G115
				Secrets: []*fleet.EnrollSecret{
					{Secret: "abc"},
					{Secret: "123"},
				},
			})
		}
		return r
	}

	t.Run("no user", func(t *testing.T) {
		err := obfuscateSecrets(nil, nil)
		require.Error(t, err)
	})

	t.Run("no teams", func(t *testing.T) {
		user := fleet.User{}
		err := obfuscateSecrets(&user, nil)
		require.NoError(t, err)
	})

	t.Run("user is not a global observer", func(t *testing.T) {
		user := fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
		teams := buildTeams(3)

		err := obfuscateSecrets(&user, teams)
		require.NoError(t, err)

		for _, team := range teams {
			for _, s := range team.Secrets {
				require.NotEqual(t, fleet.MaskedPassword, s.Secret)
			}
		}
	})

	t.Run("user is global observer/observer+/technician", func(t *testing.T) {
		roles := []*string{
			ptr.String(fleet.RoleObserver),
			ptr.String(fleet.RoleObserverPlus),
			ptr.String(fleet.RoleTechnician),
		}
		for _, r := range roles {
			user := &fleet.User{GlobalRole: r}
			teams := buildTeams(3)

			err := obfuscateSecrets(user, teams)
			require.NoError(t, err)

			for _, team := range teams {
				for _, s := range team.Secrets {
					require.Equal(t, fleet.MaskedPassword, s.Secret)
				}
			}
		}
	})

	t.Run("user is observer/technician in some teams", func(t *testing.T) {
		teams := buildTeams(5)

		// Make user an observer in the 'even' teams
		user := &fleet.User{Teams: []fleet.UserTeam{
			{
				Team: *teams[1],
				Role: fleet.RoleObserver,
			},
			{
				Team: *teams[2],
				Role: fleet.RoleAdmin,
			},
			{
				Team: *teams[3],
				Role: fleet.RoleObserverPlus,
			},
			{
				Team: *teams[4],
				Role: fleet.RoleTechnician,
			},
		}}

		err := obfuscateSecrets(user, teams)
		require.NoError(t, err)

		for i, team := range teams {
			for _, s := range team.Secrets {
				require.Equal(t, fleet.MaskedPassword == s.Secret, i == 0 || i == 1 || i == 3 || i == 4)
			}
		}
	})
}

type bootstrapNotFoundError struct {
	msg string
}

func (e *bootstrapNotFoundError) Error() string {
	return e.msg
}

func (e *bootstrapNotFoundError) IsNotFound() bool {
	return true
}

func TestUpdateTeamMDMAppleSetupManualAgent(t *testing.T) {
	cases := []struct {
		Name            string
		Count           fleet.SetupExperienceCount
		Error           string
		MacOSSetup      fleet.MacOSSetup
		MDMSetupPayload fleet.MDMAppleSetupPayload
	}{
		{
			Name: "good case",
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
		},
		{
			Name: "no bootstrap package",
			Count: fleet.SetupExperienceCount{
				Installers: 0,
				VPP:        0,
				Scripts:    0,
			},
			Error: "bootstrap_package",
		},
		{
			Name: "installers exist",
			Count: fleet.SetupExperienceCount{
				Installers: 1,
				VPP:        0,
				Scripts:    0,
			},
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
			Error: "disable setup experience software",
		},
		{
			Name: "vpp apps exist",
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
			Count: fleet.SetupExperienceCount{
				VPP: 1,
			},
			Error: "disable setup experience software",
		},
		{
			Name: "script exists",
			Count: fleet.SetupExperienceCount{
				Scripts: 1,
			},
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
			Error: "remove your setup experience script",
		},
	}

	ds := new(mock.Store)

	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
		return nil
	}

	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return &fleet.Team{}, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	svc := &Service{
		ds: ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: "something",
			},
		},
		authz: authorizer,
	}

	// Add admin user to context
	adminUser := &fleet.User{
		ID:         2,
		GlobalRole: ptr.String(fleet.RoleAdmin),
		Email:      "useradmin@example.com",
	}
	ctx := test.UserContext(context.Background(), adminUser)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ds.GetMDMAppleBootstrapPackageMetaFunc = func(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackage, error) {
				if tc.MacOSSetup.BootstrapPackage.Value == "" {
					return nil, &bootstrapNotFoundError{msg: "bootstrap package not found"}
				}
				return &fleet.MDMAppleBootstrapPackage{
					Name: tc.MacOSSetup.BootstrapPackage.Value,
				}, nil
			}

			ds.GetSetupExperienceCountFunc = func(ctx context.Context, platform string, teamID *uint) (*fleet.SetupExperienceCount, error) {
				return &tc.Count, nil
			}

			tm := &fleet.Team{}
			tm.Config.MDM.MacOSSetup = tc.MacOSSetup

			payload := fleet.MDMAppleSetupPayload{
				ManualAgentInstall: ptr.Bool(true),
			}

			err := svc.updateTeamMDMAppleSetup(ctx, tm, payload)
			if tc.Error == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.Error)
			}
		})
		t.Run(tc.Name+" no team", func(t *testing.T) {
			ds.GetSetupExperienceCountFunc = func(ctx context.Context, platform string, teamID *uint) (*fleet.SetupExperienceCount, error) {
				return &tc.Count, nil
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				appConfig := &fleet.AppConfig{}
				appConfig.MDM.MacOSSetup = tc.MacOSSetup
				return appConfig, nil
			}

			tm := &fleet.Team{}
			tm.Config.MDM.MacOSSetup = tc.MacOSSetup

			payload := fleet.MDMAppleSetupPayload{
				ManualAgentInstall: ptr.Bool(true),
			}

			err := svc.updateAppConfigMDMAppleSetup(ctx, payload)
			if tc.Error == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.Error)
			}
		})

	}
}
