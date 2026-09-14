package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdmtest "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
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
		{
			name:     "name at max length is accepted",
			teamName: new(strings.Repeat("a", fleet.MaxTeamNameLength)),
			wantName: strings.Repeat("a", fleet.MaxTeamNameLength),
		},
		{
			name:     "name over max length is rejected",
			teamName: new(strings.Repeat("a", fleet.MaxTeamNameLength+1)),
			wantErr:  fmt.Sprintf("may not exceed %d characters", fleet.MaxTeamNameLength),
		},
		{
			// Guards against regressing to byte-based length checks, which
			// would reject multibyte names that fit within the character cap.
			name:     "multibyte name at max character length is accepted",
			teamName: new(strings.Repeat("日", fleet.MaxTeamNameLength)),
			wantName: strings.Repeat("日", fleet.MaxTeamNameLength),
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
	// A successful rename reconciles the fleet names copied into the app config.
	ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFunc = func(context.Context, *uint) ([]string, error) {
		return nil, nil
	}
	ds.GetVPPTokenByTeamIDFunc = func(context.Context, *uint) (*fleet.VPPTokenDB, error) {
		return nil, &notFoundError{}
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
		{
			name:     "name at max length is accepted",
			teamName: new(strings.Repeat("a", fleet.MaxTeamNameLength)),
			wantName: strings.Repeat("a", fleet.MaxTeamNameLength),
		},
		{
			name:     "name over max length is rejected",
			teamName: new(strings.Repeat("a", fleet.MaxTeamNameLength+1)),
			wantErr:  fmt.Sprintf("may not exceed %d characters", fleet.MaxTeamNameLength),
		},
		{
			name:     "multibyte name at max character length is accepted",
			teamName: new(strings.Repeat("日", fleet.MaxTeamNameLength)),
			wantName: strings.Repeat("日", fleet.MaxTeamNameLength),
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
		{
			name:     "name at max length is accepted",
			teamName: strings.Repeat("a", fleet.MaxTeamNameLength),
			wantName: strings.Repeat("a", fleet.MaxTeamNameLength),
		},
		{
			name:     "name over max length is rejected",
			teamName: strings.Repeat("a", fleet.MaxTeamNameLength+1),
			wantErr:  fmt.Sprintf("may not exceed %d characters", fleet.MaxTeamNameLength),
		},
		{
			name:     "multibyte name at max character length is accepted",
			teamName: strings.Repeat("日", fleet.MaxTeamNameLength),
			wantName: strings.Repeat("日", fleet.MaxTeamNameLength),
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
	// A case-only rename is still a rename, so it reconciles the fleet names
	// copied into the app config.
	ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFunc = func(context.Context, *uint) ([]string, error) {
		return nil, nil
	}
	ds.GetVPPTokenByTeamIDFunc = func(context.Context, *uint) (*fleet.VPPTokenDB, error) {
		return nil, &notFoundError{}
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
		// A rename reconciles the fleet names copied into the app config.
		ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFunc = func(context.Context, *uint) ([]string, error) {
			return nil, nil
		}
		ds.GetVPPTokenByTeamIDFunc = func(context.Context, *uint) (*fleet.VPPTokenDB, error) {
			return nil, &notFoundError{}
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
// (Apple) and BitLocker (Windows), so the change must be allowed when either
// platform's MDM is configured. The Apple FileVault profile is created only
// when Apple MDM is configured.
func TestModifyTeamMDMEnableDiskEncryption(t *testing.T) {
	testCases := []struct {
		name           string
		appleEnabled   bool
		windowsEnabled bool
		wantErr        string
	}{
		{
			name:           "windows MDM only succeeds without invoking FileVault (issue #44194)",
			windowsEnabled: true,
		},
		{
			name:    "neither MDM platform configured rejects the change",
			wantErr: "mdm.enable_disk_encryption",
		},
		{
			name:         "apple MDM configured invokes FileVault profile creation",
			appleEnabled: true,
		},
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
		return nil
	}
	ctx := test.UserContext(context.Background(),
		&fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{MDM: fleet.MDM{
					EnabledAndConfigured:        tc.appleEnabled,
					WindowsEnabledAndConfigured: tc.windowsEnabled,
				}}, nil
			}
			ds.TeamWithExtrasFunc = func(_ context.Context, tid uint) (*fleet.Team, error) {
				return &fleet.Team{ID: tid, Name: "team-1"}, nil
			}
			var savedTeam *fleet.Team
			ds.SaveTeamFunc = func(_ context.Context, team *fleet.Team) (*fleet.Team, error) {
				savedTeam = team
				return team, nil
			}
			// Wire the Apple FileVault mocks only when Apple MDM is configured;
			// a regression that drops the gate would call into them on a
			// Windows-only deployment and panic on the nil mock function.
			if tc.appleEnabled {
				ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, _ []fleet.MDMAssetName,
					_ sqlx.QueryerContext,
				) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
					return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
						fleet.MDMAssetCACert: {Value: []byte(testCert)},
					}, nil
				}
				// the reconciler reads the stored settings back, so serve what
				// SaveTeam just captured
				ds.TeamMDMConfigFunc = func(_ context.Context, _ uint) (*fleet.TeamMDM, error) {
					require.NotNil(t, savedTeam, "reconcile ran before the team was saved")
					mdm := savedTeam.Config.MDM
					return &mdm, nil
				}
				ds.UpsertMDMAppleFleetConfigProfileFunc = func(_ context.Context, _ fleet.MDMAppleConfigProfile) error {
					return nil
				}
			}

			svc := &Service{
				Service: mockSvc,
				ds:      ds,
				config:  config.FleetConfig{Server: config.ServerConfig{PrivateKey: "something"}},
				authz:   authorizer,
			}

			payload := fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{
				EnableDiskEncryption: optjson.SetBool(true),
			}}
			team, err := svc.ModifyTeam(ctx, 1, payload)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				require.Nil(t, team)
				require.False(t, ds.SaveTeamFuncInvoked, "team should not have been saved")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, team)
			require.True(t, team.Config.MDM.EnableDiskEncryption)
			require.NotNil(t, savedTeam)
			require.True(t, savedTeam.Config.MDM.EnableDiskEncryption)
			require.Equal(t, tc.appleEnabled, ds.UpsertMDMAppleFleetConfigProfileFuncInvoked,
				"FileVault profile write must match Apple MDM configuration")
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
			// per-platform fields set to match the flat toggle, as the
			// unmarshal gap-fill guarantees for teams loaded from the DB
			mdmConfig: fleet.TeamMDM{
				EnableDiskEncryption: true,
				MacOSSettings: fleet.MacOSSettings{
					EnableDiskEncryption:          optjson.SetBool(true),
					EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
				},
				WindowsSettings: fleet.WindowsSettings{
					EnableDiskEncryption: optjson.SetBool(true),
				},
				LinuxSettings: fleet.LinuxSettings{
					EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
				},
				RequireBitLockerPIN: true,
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
				MacOSSettings: fleet.MacOSSettings{
					EnableDiskEncryption:          optjson.SetBool(true),
					EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
				},
				WindowsSettings: fleet.WindowsSettings{
					EnableDiskEncryption: optjson.SetBool(true),
				},
				LinuxSettings: fleet.LinuxSettings{
					EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
				},
				RequireBitLockerPIN: false,
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

		// the flat toggle fans out to every per-platform setting, as
		// ResolvePerPlatform does for the endpoint payload
		changes := fleet.DiskEncryptionSettingsChanges{
			MacOSEnable:   tC.diskEncryption,
			MacOSEscrow:   tC.diskEncryption,
			WindowsEnable: tC.diskEncryption,
			LinuxEscrow:   tC.diskEncryption,
		}
		err := svc.updateTeamMDMDiskEncryption(
			ctx,
			&team,
			changes,
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

	t.Run("user is global gitops", func(t *testing.T) {
		// gitops can write teams but is not allowed to read enroll secrets, so
		// the secrets must be masked even in write responses.
		user := &fleet.User{GlobalRole: new(fleet.RoleGitOps)}
		teams := buildTeams(3)

		err := obfuscateSecrets(user, teams)
		require.NoError(t, err)

		for _, team := range teams {
			for _, s := range team.Secrets {
				require.Equal(t, fleet.MaskedPassword, s.Secret)
			}
		}
	})

	t.Run("user is global maintainer", func(t *testing.T) {
		user := &fleet.User{GlobalRole: new(fleet.RoleMaintainer)}
		teams := buildTeams(3)

		err := obfuscateSecrets(user, teams)
		require.NoError(t, err)

		for _, team := range teams {
			for _, s := range team.Secrets {
				require.NotEqual(t, fleet.MaskedPassword, s.Secret)
			}
		}
	})

	t.Run("user is gitops/maintainer in some teams", func(t *testing.T) {
		teams := buildTeams(3)

		// Team gitops can modify the team but must not read its enroll secrets,
		// while team maintainer can. The user is not a member of team 0.
		user := &fleet.User{Teams: []fleet.UserTeam{
			{
				Team: *teams[1],
				Role: fleet.RoleGitOps,
			},
			{
				Team: *teams[2],
				Role: fleet.RoleMaintainer,
			},
		}}

		err := obfuscateSecrets(user, teams)
		require.NoError(t, err)

		for i, team := range teams {
			for _, s := range team.Secrets {
				// Only team 2 (maintainer) should be visible; team 0 (no
				// membership) and team 1 (gitops) must be masked.
				require.Equal(t, fleet.MaskedPassword == s.Secret, i == 0 || i == 1)
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

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
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

// TestApplyTeamSpecsCustomSettingsWithoutMDMConfigured verifies that GitOps
// team edits can add Windows or Android configuration profiles without being
// rejected by an MDM-configured check on the team-edit path. The previous
// check fired when the AppConfig's *EnabledAndConfigured flag was false, which
// surfaced as a customer bug when the cached AppConfig lagged behind a recent
// platform-enable. createTeamFromSpec has never gated this, so we mirror it.
func TestApplyTeamSpecsCustomSettingsWithoutMDMConfigured(t *testing.T) {
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	adminUser := &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}
	ctx := test.UserContext(context.Background(), adminUser)

	const teamName = "Mobile"

	newSvc := func(t *testing.T, mdmConfigured bool) (*Service, *mock.Store, **fleet.Team) {
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			// mdmConfigured=false simulates the bug condition: the cached
			// AppConfig still reports MDM as off even though it has actually
			// been enabled. The fix must let the edit proceed regardless.
			ac.MDM.AndroidEnabledAndConfigured = mdmConfigured
			ac.MDM.WindowsEnabledAndConfigured = mdmConfigured
			return ac, nil
		}
		ds.TeamByFilenameFunc = func(ctx context.Context, _ string) (*fleet.Team, error) {
			return nil, &notFoundError{}
		}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
			// Existing team has no Android/Windows custom settings — the spec
			// is adding the first profile.
			return &fleet.Team{ID: 42, Name: name}, nil
		}
		ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
			return nil, nil
		}
		var saved *fleet.Team
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			saved = team
			return team, nil
		}

		mockSvc := &svcmock.Service{}
		mockSvc.NewActivityFunc = func(ctx context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
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
		return svc, ds, &saved
	}

	t.Run("adds android profile when AppConfig reports Android MDM off", func(t *testing.T) {
		svc, ds, saved := newSvc(t, false)
		spec := &fleet.TeamSpec{
			Name: teamName,
			MDM: fleet.TeamSpecMDM{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: "profiles/android.json"},
					}),
				},
			},
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.NotNil(t, *saved)
		require.Len(t, (*saved).Config.MDM.AndroidSettings.CustomSettings.Value, 1)
		require.Equal(t, "profiles/android.json", (*saved).Config.MDM.AndroidSettings.CustomSettings.Value[0].Path)
	})

	t.Run("adds windows profile when AppConfig reports Windows MDM off", func(t *testing.T) {
		svc, ds, saved := newSvc(t, false)
		spec := &fleet.TeamSpec{
			Name: teamName,
			MDM: fleet.TeamSpecMDM{
				WindowsSettings: fleet.WindowsSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: "profiles/windows.xml"},
					}),
				},
			},
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.NotNil(t, *saved)
		require.Len(t, (*saved).Config.MDM.WindowsSettings.CustomSettings.Value, 1)
		require.Equal(t, "profiles/windows.xml", (*saved).Config.MDM.WindowsSettings.CustomSettings.Value[0].Path)
	})

	t.Run("edit persists and disables the windows managed local account toggle", func(t *testing.T) {
		svc, ds, saved := newSvc(t, true)
		existing := &fleet.Team{ID: 42, Name: teamName}
		ds.TeamByNameFunc = func(context.Context, string) (*fleet.Team, error) { return existing, nil }
		spec := &fleet.TeamSpec{
			Name: teamName,
			MDM: fleet.TeamSpecMDM{
				WindowsSettings: fleet.WindowsSettings{
					EnableManagedLocalAccount: optjson.SetBool(true),
				},
			},
		}
		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, (*saved).Config.MDM.WindowsSettings.EnableManagedLocalAccount.Value)

		// an explicit false disables the managed local account again
		spec.MDM.WindowsSettings.EnableManagedLocalAccount = optjson.SetBool(false)
		_, err = svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.False(t, (*saved).Config.MDM.WindowsSettings.EnableManagedLocalAccount.Value)
	})

	t.Run("adds android profile when AppConfig reports Android MDM on (happy path)", func(t *testing.T) {
		svc, ds, saved := newSvc(t, true)
		spec := &fleet.TeamSpec{
			Name: teamName,
			MDM: fleet.TeamSpecMDM{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: "profiles/android.json"},
					}),
				},
			},
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.NotNil(t, *saved)
		require.Len(t, (*saved).Config.MDM.AndroidSettings.CustomSettings.Value, 1)
	})
}

// TestApplyTeamSpecsClearBootstrapPackageAlreadyDeleted verifies that clearing
// a bootstrap package via GitOps succeeds even when the actual package row has
// already been deleted (e.g. via the GUI), leaving a stale URL in team config.
func TestApplyTeamSpecsClearBootstrapPackageAlreadyDeleted(t *testing.T) {
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	adminUser := &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}
	ctx := test.UserContext(context.Background(), adminUser)

	const teamName = "TestTeam"
	const teamID = uint(42)

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, _ string) (*fleet.Team, error) {
		return nil, &notFoundError{}
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{
			ID:   teamID,
			Name: name,
			Config: fleet.TeamConfig{
				MDM: fleet.TeamMDM{
					MacOSSetup: fleet.MacOSSetup{
						// Stale URL: the DB row is gone but team config still has it.
						BootstrapPackage: optjson.SetString("https://example.com/bootstrap.pkg"),
					},
				},
			},
		}, nil
	}
	ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
		return nil, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}
	// The bootstrap package row was already deleted via the GUI.
	ds.TeamWithExtrasFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: teamName}, nil
	}
	ds.GetMDMAppleBootstrapPackageMetaFunc = func(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackage, error) {
		return nil, &bootstrapNotFoundError{msg: "bootstrap package not found"}
	}

	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(ctx context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
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

	spec := &fleet.TeamSpec{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSetup: fleet.MacOSSetup{
				// Clearing the bootstrap package (Set=true, Value="").
				BootstrapPackage: optjson.SetString(""),
			},
		},
	}

	_, err = svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{})
	require.NoError(t, err)
	require.True(t, ds.SaveTeamFuncInvoked)
}

// TestModifyTeamMDMManagedLocalAccountRequiresMDM covers the MDM-off gates for both platform toggles, which the
// integration suite can't exercise since it always runs with MDM configured, plus the Windows managed local account toggle's
// persistence and activity.
func TestModifyTeamMDMManagedLocalAccountRequiresMDM(t *testing.T) {
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	ctx := test.UserContext(context.Background(),
		&fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	windowsMDMConfigured := false
	ds := new(mock.Store)
	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{
			EnabledAndConfigured:        false,
			WindowsEnabledAndConfigured: windowsMDMConfigured,
		}}, nil
	}
	ds.TeamWithExtrasFunc = func(_ context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "team-1"}, nil
	}
	ds.SaveTeamFunc = func(_ context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}

	var activities []string
	mockSvc := &svcmock.Service{}
	// Reached via validateEndUserAuthenticationAndSetupAssistant when MacOSSetup is set.
	mockSvc.HasCustomSetupAssistantConfigurationWebURLFunc = func(context.Context, *uint) (bool, error) {
		return false, nil
	}
	mockSvc.NewActivityFunc = func(_ context.Context, _ *fleet.User, act fleet.ActivityDetails) error {
		switch a := act.(type) {
		case fleet.ActivityTypeEnabledManagedLocalAccount:
			activities = append(activities, a.ActivityName()+":"+a.Platform)
		case fleet.ActivityTypeDisabledManagedLocalAccount:
			activities = append(activities, a.ActivityName()+":"+a.Platform)
		}
		return nil
	}

	svc := &Service{
		Service: mockSvc,
		ds:      ds,
		config:  config.FleetConfig{Server: config.ServerConfig{PrivateKey: "something"}},
		authz:   authorizer,
		logger:  slog.New(slog.DiscardHandler),
	}

	windowsPayload := fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{
		WindowsSettings: &fleet.TeamPayloadWindowsSettings{
			EnableManagedLocalAccount: optjson.SetBool(true),
		},
	}}

	t.Run("macOS enable admin account requires Apple MDM", func(t *testing.T) {
		_, err := svc.ModifyTeam(ctx, 1, fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{
			MacOSSetup: &fleet.MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(true)},
		}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "setup_experience.enable_managed_local_account")
		require.False(t, ds.SaveTeamFuncInvoked, "team should not have been saved")
	})

	t.Run("windows enable admin account requires Windows MDM", func(t *testing.T) {
		_, err := svc.ModifyTeam(ctx, 1, windowsPayload)
		require.Error(t, err)
		require.Contains(t, err.Error(), "windows_settings.enable_managed_local_account")
		require.False(t, ds.SaveTeamFuncInvoked, "team should not have been saved")
	})

	t.Run("windows enable admin toggle persists and fires activity", func(t *testing.T) {
		windowsMDMConfigured = true
		team, err := svc.ModifyTeam(ctx, 1, windowsPayload)
		require.NoError(t, err)
		require.True(t, team.Config.MDM.WindowsSettings.EnableManagedLocalAccount.Value)
		require.False(t, team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
		require.Equal(t, []string{"enabled_managed_local_account:windows"}, activities)
	})
}

func TestDeleteTeamWindowsEnrollmentDefaultFleet(t *testing.T) {
	deletedTeamID, otherTeamID := uint(42), uint(43)

	testCases := []struct {
		name           string
		defaultFleetID *uint
		wantCleared    bool
	}{
		{name: "deleted fleet is the configured default", defaultFleetID: &deletedTeamID, wantCleared: true},
		{name: "another fleet is the configured default", defaultFleetID: &otherTeamID},
		{name: "no default configured"},
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	ctx := test.UserContext(context.Background(),
		&fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.TeamLiteFunc = func(_ context.Context, tid uint) (*fleet.TeamLite, error) {
				return &fleet.TeamLite{ID: tid, Name: "team-1"}, nil
			}
			ds.ListHostsFunc = func(context.Context, fleet.TeamFilter, fleet.HostListOptions) ([]*fleet.Host, error) {
				return nil, nil
			}
			ds.GetCertificateTemplatesByTeamIDFunc = func(context.Context, uint, fleet.ListOptions) (
				[]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error,
			) {
				return nil, nil, nil
			}
			// Deleting a fleet also scrubs its name from the app config copies.
			ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{}, nil
			}
			ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFunc = func(context.Context, *uint) ([]string, error) {
				return nil, nil
			}
			ds.GetVPPTokenByTeamIDFunc = func(context.Context, *uint) (*fleet.VPPTokenDB, error) {
				return nil, &notFoundError{}
			}
			ds.GetWindowsEnrollmentDefaultFleetFunc = func(context.Context) (*uint, string, error) {
				return tc.defaultFleetID, "default-fleet", nil
			}
			var clearedTo *uint
			ds.SetWindowsEnrollmentDefaultFleetFunc = func(_ context.Context, fleetID *uint) error {
				clearedTo = fleetID
				return nil
			}
			ds.DeleteTeamFunc = func(context.Context, uint) error { return nil }

			var activities []string
			mockSvc := &svcmock.Service{}
			mockSvc.NewActivityFunc = func(_ context.Context, _ *fleet.User, act fleet.ActivityDetails) error {
				if a, ok := act.(fleet.ActivityTypeEditedWindowsEnrollmentDefaultFleet); ok {
					require.Nil(t, a.FleetID, "cleared default must not name a fleet")
					require.Nil(t, a.FleetName, "cleared default must not name a fleet")
				}
				activities = append(activities, act.ActivityName())
				return nil
			}

			svc := &Service{
				Service: mockSvc,
				ds:      ds,
				authz:   authorizer,
				logger:  slog.New(slog.DiscardHandler),
			}

			require.NoError(t, svc.DeleteTeam(ctx, deletedTeamID))

			// The deleted fleet activity always fires; the enrollment one only when the default was actually cleared.
			require.Contains(t, activities, fleet.ActivityTypeDeletedTeam{}.ActivityName())
			clearedActivity := fleet.ActivityTypeEditedWindowsEnrollmentDefaultFleet{}.ActivityName()
			if tc.wantCleared {
				require.True(t, ds.SetWindowsEnrollmentDefaultFleetFuncInvoked)
				require.Nil(t, clearedTo, "default fleet should be cleared, not reassigned")
				require.Contains(t, activities, clearedActivity)
			} else {
				require.False(t, ds.SetWindowsEnrollmentDefaultFleetFuncInvoked)
				require.NotContains(t, activities, clearedActivity)
			}
		})
	}
}

func TestModifyTeamOSUpdatesDeadlineDays(t *testing.T) {
	// A deadline_days-only edit must be treated as a change: the setting has to be
	// stored and the OS update declaration regenerated. Before deadline_days was
	// part of the change detection, both were silently skipped.
	testCases := []struct {
		name         string
		storedDays   optjson.Int
		payloadDays  optjson.Int
		wantSaved    int
		wantRedeploy bool
	}{
		{
			name:         "deadline_days changed",
			storedDays:   optjson.SetInt(14),
			payloadDays:  optjson.SetInt(21),
			wantSaved:    21,
			wantRedeploy: true,
		},
		{
			name:         "deadline_days set from unset",
			storedDays:   optjson.Int{},
			payloadDays:  optjson.SetInt(14),
			wantSaved:    14,
			wantRedeploy: true,
		},
		{
			name:         "deadline_days unchanged",
			storedDays:   optjson.SetInt(14),
			payloadDays:  optjson.SetInt(14),
			wantSaved:    14,
			wantRedeploy: false,
		},
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	ctx := test.UserContext(context.Background(),
		&fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var gotActivities []fleet.ActivityDetails
			mockSvc := &svcmock.Service{}
			mockSvc.NewActivityFunc = func(_ context.Context, _ *fleet.User, a fleet.ActivityDetails) error {
				gotActivities = append(gotActivities, a)
				return nil
			}

			ds := new(mock.Store)
			ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
			}
			ds.TeamWithExtrasFunc = func(_ context.Context, tid uint) (*fleet.Team, error) {
				return &fleet.Team{ID: tid, Name: "team-1", Config: fleet.TeamConfig{
					MDM: fleet.TeamMDM{
						MacOSUpdates: fleet.AppleOSUpdateSettings{
							MinimumVersion: optjson.SetString(fleet.AppleOSUpdateLatestVersion),
							DeadlineDays:   tc.storedDays,
						},
					},
				}}, nil
			}
			var savedTeam *fleet.Team
			ds.SaveTeamFunc = func(_ context.Context, team *fleet.Team) (*fleet.Team, error) {
				savedTeam = team
				return team, nil
			}
			ds.HasAppleUpdateConfigProfileConfiguredFunc = func(_ context.Context, teamID uint) (bool, error) {
				return false, nil
			}
			ds.LabelIDsByNameFunc = func(_ context.Context, names []string, _ fleet.TeamFilter) (map[string]uint, error) {
				ids := make(map[string]uint, len(names))
				for i, name := range names {
					ids[name] = uint(i + 1) //nolint:gosec
				}
				return ids, nil
			}
			var gotDecl *fleet.MDMAppleDeclaration
			var gotVars []fleet.FleetVarName
			ds.SetOrUpdateMDMAppleDeclarationFunc = func(_ context.Context, decl *fleet.MDMAppleDeclaration,
				usesFleetVars []fleet.FleetVarName, activationAction fleet.MDMAppleActivationAction,
			) (*fleet.MDMAppleDeclaration, error) {
				gotDecl = decl
				gotVars = usesFleetVars
				decl.DeclarationUUID = "decl-uuid"
				return decl, nil
			}

			svc := &Service{
				Service: mockSvc,
				ds:      ds,
				config:  config.FleetConfig{Server: config.ServerConfig{PrivateKey: "something"}},
				authz:   authorizer,
			}

			payload := fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{
				MacOSUpdates: &fleet.AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString(fleet.AppleOSUpdateLatestVersion),
					DeadlineDays:   tc.payloadDays,
				},
			}}
			team, err := svc.ModifyTeam(ctx, 1, payload)
			require.NoError(t, err)
			require.NotNil(t, team)

			// The outer Set guard also controls whether the value is stored at all.
			require.NotNil(t, savedTeam)
			require.Equal(t, tc.wantSaved, savedTeam.Config.MDM.MacOSUpdates.DeadlineDays.Value)

			require.Equal(t, tc.wantRedeploy, ds.SetOrUpdateMDMAppleDeclarationFuncInvoked,
				"declaration regeneration must follow the change detection")
			if tc.wantRedeploy {
				require.NotNil(t, gotDecl)
				require.Contains(t, string(gotDecl.RawJSON), "$FLEET_VAR_HOST_TARGET_OS_VERSION")
				require.Len(t, gotVars, 2)
			}

			// The activity feed renders "updated macOS version to latest" from
			// minimum_version, so the payload has to carry the sentinel through.
			// Deadline stays empty in latest mode, which is what makes the
			// renderer drop its "(deadline: ...)" clause.
			var osUpdateActivities []fleet.ActivityTypeEditedMacOSMinVersion
			for _, a := range gotActivities {
				if edited, ok := a.(fleet.ActivityTypeEditedMacOSMinVersion); ok {
					osUpdateActivities = append(osUpdateActivities, edited)
				}
			}
			if !tc.wantRedeploy {
				require.Empty(t, osUpdateActivities, "an unchanged setting must not emit an activity")
				return
			}
			require.Len(t, osUpdateActivities, 1)
			require.Equal(t, fleet.AppleOSUpdateLatestVersion, osUpdateActivities[0].MinimumVersion)
			require.Empty(t, osUpdateActivities[0].Deadline)
			require.NotNil(t, osUpdateActivities[0].TeamID)
			require.Equal(t, uint(1), *osUpdateActivities[0].TeamID)
		})
	}
}

// ModifyTeam validates the incoming payload and then replaces the whole
// AppleOSUpdateSettings struct, so a stored deadline field can't leak into the
// validated value. ModifyAppConfig merges the payload over the stored config
// instead, which is why it needed clearStaleAppleOSUpdateDeadline and this
// doesn't. These cases lock that in for both directions: a sparse PATCH that
// switches mode must succeed and must not persist the outgoing mode's deadline.
func TestModifyTeamSwitchingOSUpdateModes(t *testing.T) {
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	ctx := test.UserContext(context.Background(),
		&fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	storedLatest := fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString(fleet.AppleOSUpdateLatestVersion),
		DeadlineDays:   optjson.SetInt(14),
	}

	setup := func(t *testing.T, stored fleet.AppleOSUpdateSettings) (*Service, func() *fleet.Team) {
		// ModifyTeam checks minimum_version against GDMF, so serve Apple's asset
		// list from the local fixture rather than reaching out to Apple.
		mdmtest.StartNewAppleGDMFTestServer(t)

		mockSvc := &svcmock.Service{}
		mockSvc.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
			return nil
		}

		ds := new(mock.Store)
		ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
		}
		ds.TeamWithExtrasFunc = func(_ context.Context, tid uint) (*fleet.Team, error) {
			return &fleet.Team{ID: tid, Name: "team-1", Config: fleet.TeamConfig{
				MDM: fleet.TeamMDM{MacOSUpdates: stored},
			}}, nil
		}
		var savedTeam *fleet.Team
		ds.SaveTeamFunc = func(_ context.Context, team *fleet.Team) (*fleet.Team, error) {
			savedTeam = team
			return team, nil
		}
		ds.HasAppleUpdateConfigProfileConfiguredFunc = func(context.Context, uint) (bool, error) {
			return false, nil
		}
		ds.LabelIDsByNameFunc = func(_ context.Context, names []string, _ fleet.TeamFilter) (map[string]uint, error) {
			ids := make(map[string]uint, len(names))
			for i, name := range names {
				ids[name] = uint(i + 1) //nolint:gosec // G115: small test values
			}
			return ids, nil
		}
		ds.SetOrUpdateMDMAppleDeclarationFunc = func(_ context.Context, decl *fleet.MDMAppleDeclaration,
			_ []fleet.FleetVarName, activationAction fleet.MDMAppleActivationAction,
		) (*fleet.MDMAppleDeclaration, error) {
			decl.DeclarationUUID = "decl-uuid"
			return decl, nil
		}
		ds.DeleteMDMAppleDeclarationByNameFunc = func(context.Context, *uint, string) error {
			return nil
		}

		return &Service{
			Service: mockSvc,
			ds:      ds,
			config:  config.FleetConfig{Server: config.ServerConfig{PrivateKey: "something"}},
			authz:   authorizer,
		}, func() *fleet.Team { return savedTeam }
	}

	t.Run("switching to a specific version", func(t *testing.T) {
		svc, saved := setup(t, storedLatest)

		// deadline_days is deliberately absent, as a sparse PATCH would leave it.
		_, err := svc.ModifyTeam(ctx, 1, fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("14.6.1"),
				Deadline:       optjson.SetString("2026-09-01"),
			},
		}})
		require.NoError(t, err)

		require.NotNil(t, saved())
		require.Equal(t, "14.6.1", saved().Config.MDM.MacOSUpdates.MinimumVersion.Value)
		require.False(t, saved().Config.MDM.MacOSUpdates.DeadlineDays.Valid,
			"the stored deadline_days must not survive the mode change")
	})

	t.Run("clearing enforcement entirely", func(t *testing.T) {
		svc, saved := setup(t, storedLatest)

		_, err := svc.ModifyTeam(ctx, 1, fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(""),
				Deadline:       optjson.SetString(""),
			},
		}})
		require.NoError(t, err)

		require.NotNil(t, saved())
		require.Empty(t, saved().Config.MDM.MacOSUpdates.MinimumVersion.Value)
		require.False(t, saved().Config.MDM.MacOSUpdates.DeadlineDays.Valid)
	})

	t.Run("switching into latest mode from a specific version", func(t *testing.T) {
		// the mirror direction: a stored deadline is the stale field here, and the
		// wholesale replace has to drop it just the same.
		svc, saved := setup(t, fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("14.6.1"),
			Deadline:       optjson.SetString("2026-09-01"),
		})

		// deadline is deliberately absent, as a sparse PATCH would leave it.
		_, err := svc.ModifyTeam(ctx, 1, fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(fleet.AppleOSUpdateLatestVersion),
				DeadlineDays:   optjson.SetInt(14),
			},
		}})
		require.NoError(t, err)

		require.NotNil(t, saved())
		require.Equal(t, fleet.AppleOSUpdateLatestVersion, saved().Config.MDM.MacOSUpdates.MinimumVersion.Value)
		require.Equal(t, 14, saved().Config.MDM.MacOSUpdates.DeadlineDays.Value)
		require.Empty(t, saved().Config.MDM.MacOSUpdates.Deadline.Value,
			"the stored deadline must not survive the mode change")
	})
}

func TestApplyTeamSpecsOSUpdatesValidation(t *testing.T) {
	// GitOps applies team settings through editTeamFromSpec, which validates each
	// Apple platform's OS update settings. All three must reject invalid settings,
	// keyed by the platform that is at fault.
	latest := func(days optjson.Int) fleet.AppleOSUpdateSettings {
		return fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString(fleet.AppleOSUpdateLatestVersion),
			DeadlineDays:   days,
		}
	}
	valid := latest(optjson.SetInt(14))
	missingDays := latest(optjson.Int{})

	testCases := []struct {
		name    string
		mdm     fleet.TeamSpecMDM
		wantErr string
	}{
		{
			name: "all platforms valid",
			mdm:  fleet.TeamSpecMDM{MacOSUpdates: valid, IOSUpdates: valid, IPadOSUpdates: valid},
		},
		{
			name:    "macos missing deadline_days",
			mdm:     fleet.TeamSpecMDM{MacOSUpdates: missingDays},
			wantErr: "macos_updates",
		},
		{
			name:    "ios missing deadline_days",
			mdm:     fleet.TeamSpecMDM{IOSUpdates: missingDays},
			wantErr: "ios_updates",
		},
		{
			name:    "ipados missing deadline_days",
			mdm:     fleet.TeamSpecMDM{IPadOSUpdates: missingDays},
			wantErr: "ipados_updates",
		},
		{
			name:    "macos deadline with latest",
			mdm:     fleet.TeamSpecMDM{MacOSUpdates: fleet.AppleOSUpdateSettings{MinimumVersion: optjson.SetString(fleet.AppleOSUpdateLatestVersion), Deadline: optjson.SetString("2026-09-01"), DeadlineDays: optjson.SetInt(14)}},
			wantErr: "macos_updates",
		},
		{
			// Not a "latest" case: a half-configured block was accepted before iOS
			// was validated here, then enforced nothing because Configured() needs
			// both fields. Existing fleet files like this now fail the apply.
			name:    "ios version without deadline",
			mdm:     fleet.TeamSpecMDM{IOSUpdates: fleet.AppleOSUpdateSettings{MinimumVersion: optjson.SetString("17.5")}},
			wantErr: "ios_updates",
		},
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	ctx := test.UserContext(context.Background(),
		&fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &svcmock.Service{}
			mockSvc.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
				return nil
			}

			ds := new(mock.Store)
			ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
			}
			ds.TeamByNameFunc = func(_ context.Context, name string) (*fleet.Team, error) {
				return &fleet.Team{ID: 1, Name: name}, nil
			}
			ds.TeamConflictsWithNameFunc = func(context.Context, string, uint) (*fleet.Team, error) {
				return nil, nil
			}
			ds.IsEnrollSecretAvailableFunc = func(context.Context, string, bool, *uint) (bool, error) {
				return true, nil
			}
			ds.SaveTeamFunc = func(_ context.Context, team *fleet.Team) (*fleet.Team, error) {
				return team, nil
			}
			ds.HasAppleUpdateConfigProfileConfiguredFunc = func(context.Context, uint) (bool, error) {
				return false, nil
			}
			ds.LabelIDsByNameFunc = func(_ context.Context, names []string, _ fleet.TeamFilter) (map[string]uint, error) {
				ids := make(map[string]uint, len(names))
				for i, name := range names {
					ids[name] = uint(i + 1) //nolint:gosec
				}
				return ids, nil
			}
			ds.SetOrUpdateMDMAppleDeclarationFunc = func(_ context.Context, decl *fleet.MDMAppleDeclaration,
				_ []fleet.FleetVarName, activationAction fleet.MDMAppleActivationAction,
			) (*fleet.MDMAppleDeclaration, error) {
				decl.DeclarationUUID = "decl-uuid"
				return decl, nil
			}

			svc := &Service{
				Service: mockSvc,
				ds:      ds,
				config:  config.FleetConfig{Server: config.ServerConfig{PrivateKey: "something"}},
				authz:   authorizer,
			}

			_, err := svc.ApplyTeamSpecs(ctx,
				[]*fleet.TeamSpec{{Name: "team-1", MDM: tc.mdm}},
				fleet.ApplyTeamSpecOptions{})

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.ErrorContains(t, err, tc.wantErr,
				"the error must name the platform whose settings are invalid")
			require.False(t, ds.SaveTeamFuncInvoked, "an invalid spec must not be persisted")
		})
	}
}

// GitOps creating a brand-new team has to persist the Apple OS update settings
// the same way editing an existing one does; they were previously dropped.
func TestApplyTeamSpecsCreateAppliesAppleOSUpdates(t *testing.T) {
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
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, newB bool, teamID *uint) (bool, error) {
		return true, nil
	}

	var created *fleet.Team
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		created = team
		team.ID = 1
		return team, nil
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
	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	svc.Service = mockSvc

	ctx := test.UserContext(t.Context(), &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	spec := &fleet.TeamSpec{Name: "Engineering"}
	spec.MDM.MacOSUpdates = fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("15.0"),
		Deadline:       optjson.SetString("2026-09-01"),
	}
	spec.MDM.IOSUpdates = fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("18.1"),
		Deadline:       optjson.SetString("2026-09-02"),
	}
	spec.MDM.IPadOSUpdates = fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("18.2"),
		Deadline:       optjson.SetString("2026-09-03"),
	}

	_, err = svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{})
	require.NoError(t, err)
	require.True(t, ds.NewTeamFuncInvoked)
	require.NotNil(t, created)

	// macOS already worked; it's here so a regression that drops all of them is
	// distinguishable from one that drops only the iOS/iPadOS pair.
	require.Equal(t, "15.0", created.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "18.1", created.Config.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2026-09-02", created.Config.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "18.2", created.Config.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2026-09-03", created.Config.MDM.IPadOSUpdates.Deadline.Value)
}

// staleTeamNamesFixture is the state a rename or a delete has to reconcile: the
// fleet names copied into mdm.apple_business and mdm.volume_purchasing_program,
// plus the abm_tokens and vpp_token_teams rows that are the source of truth for
// which of those entries may be touched.
type staleTeamNamesFixture struct {
	abmStored []fleet.MDMAppleABMAssignmentInfo
	abmOrgs   []string // ABM tokens defaulting to this fleet
	vppStored []fleet.MDMAppleVolumePurchasingProgramInfo
	vppToken  *fleet.VPPTokenDB // nil means the fleet has no VPP token
}

// newStaleTeamNamesService wires a service over f, and returns a live count of
// app config writes so a test can pin that the ABM and VPP corrections share one.
func newStaleTeamNamesService(t *testing.T, teamID uint, f staleTeamNamesFixture) (*Service, *mock.Store, *fleet.AppConfig, *int) {
	appCfg := &fleet.AppConfig{}
	appCfg.MDM.AppleBusinessManager = optjson.SetSlice(f.abmStored)
	appCfg.MDM.VolumePurchasingProgram = optjson.SetSlice(f.vppStored)

	var saves int
	ds := new(mock.Store)
	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) { return appCfg, nil }
	ds.TeamConflictsWithNameFunc = func(context.Context, string, uint) (*fleet.Team, error) { return nil, nil }
	ds.SaveTeamFunc = func(_ context.Context, team *fleet.Team) (*fleet.Team, error) { return team, nil }
	ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFunc = func(_ context.Context, tid *uint) ([]string, error) {
		require.NotNil(t, tid)
		require.Equal(t, teamID, *tid, "must look up ABM defaults for this fleet")
		return f.abmOrgs, nil
	}
	ds.GetVPPTokenByTeamIDFunc = func(_ context.Context, tid *uint) (*fleet.VPPTokenDB, error) {
		require.NotNil(t, tid)
		require.Equal(t, teamID, *tid, "must look up the VPP token for this fleet")
		if f.vppToken == nil {
			return nil, &notFoundError{}
		}
		return f.vppToken, nil
	}
	ds.SaveAppConfigFunc = func(context.Context, *fleet.AppConfig) error {
		saves++
		return nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	mockSvc := &svcmock.Service{}
	mockSvc.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error { return nil }

	return &Service{
		Service: mockSvc,
		ds:      ds,
		config:  config.FleetConfig{Server: config.ServerConfig{PrivateKey: "something"}},
		authz:   authorizer,
		logger:  slog.New(slog.DiscardHandler),
	}, ds, appCfg, &saves
}

func TestApplyTeamSpecsRenameUpdatesStaleAppConfigTeamNames(t *testing.T) {
	ctx := test.UserContext(context.Background(), &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	const filename = "workstations.yml"

	newSvc := func() (*Service, *mock.Store, *fleet.AppConfig, *int) {
		svc, ds, appCfg, saves := newStaleTeamNamesService(t, 7, staleTeamNamesFixture{
			abmStored: []fleet.MDMAppleABMAssignmentInfo{
				{OrganizationName: "Acme Inc", MacOSTeam: "Workstations", IpadOSTeam: "Tablets"},
			},
			abmOrgs:   []string{"Acme Inc"},
			vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations", "Tablets"}}},
			vppToken:  &fleet.VPPTokenDB{ID: 1, Location: "Acme HQ"},
		})
		ds.IsEnrollSecretAvailableFunc = func(context.Context, string, bool, *uint) (bool, error) { return true, nil }
		ds.TeamByNameFunc = func(context.Context, string) (*fleet.Team, error) { return nil, &notFoundError{} }
		ds.TeamByFilenameFunc = func(context.Context, string) (*fleet.Team, error) {
			return &fleet.Team{ID: 7, Name: "Workstations", Filename: new(filename)}, nil
		}
		return svc, ds, appCfg, saves
	}

	t.Run("rename via filename-matched spec rewrites both ABM and VPP entries", func(t *testing.T) {
		svc, _, appCfg, saves := newSvc()

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
			{Name: "Laptops", Filename: new(filename)},
		}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)

		require.Equal(t, 1, *saves)
		require.Equal(t, []fleet.MDMAppleABMAssignmentInfo{
			{OrganizationName: "Acme Inc", MacOSTeam: "Laptops", IpadOSTeam: "Tablets"},
		}, appCfg.MDM.AppleBusinessManager.Value)
		require.Equal(t, []fleet.MDMAppleVolumePurchasingProgramInfo{
			{Location: "Acme HQ", Teams: []string{"Laptops", "Tablets"}},
		}, appCfg.MDM.VolumePurchasingProgram.Value)
	})

	t.Run("dry run does not touch the ABM or VPP config", func(t *testing.T) {
		svc, ds, appCfg, saves := newSvc()

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{
			{Name: "Laptops", Filename: new(filename)},
		}, fleet.ApplyTeamSpecOptions{ApplySpecOptions: fleet.ApplySpecOptions{DryRun: true}})
		require.NoError(t, err)

		require.False(t, ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFuncInvoked)
		require.False(t, ds.GetVPPTokenByTeamIDFuncInvoked)
		require.Zero(t, *saves)
		require.Equal(t, "Workstations", appCfg.MDM.AppleBusinessManager.Value[0].MacOSTeam)
		require.Equal(t, []string{"Workstations", "Tablets"}, appCfg.MDM.VolumePurchasingProgram.Value[0].Teams)
	})
}

func TestModifyTeamRenameUpdatesStaleAppConfigTeamNames(t *testing.T) {
	const teamID = uint(5)

	acmeToken := &fleet.VPPTokenDB{ID: 1, Location: "Acme HQ"}

	testCases := []struct {
		name       string
		newName    string
		fixture    staleTeamNamesFixture
		wantLookup bool
		wantSaves  int
		wantABM    []fleet.MDMAppleABMAssignmentInfo
		wantVPP    []fleet.MDMAppleVolumePurchasingProgramInfo
	}{
		{
			name:    "renames the fleet everywhere it appears, in a single write",
			newName: "Laptops",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{
					{OrganizationName: "Acme Inc", MacOSTeam: "Workstations", IOSTeam: "Phones"},
				},
				abmOrgs:   []string{"Acme Inc"},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations", "Servers"}}},
				vppToken:  acmeToken,
			},
			wantLookup: true,
			wantSaves:  1,
			wantABM: []fleet.MDMAppleABMAssignmentInfo{
				{OrganizationName: "Acme Inc", MacOSTeam: "Laptops", IOSTeam: "Phones"},
			},
			wantVPP: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Laptops", "Servers"}}},
		},
		{
			name:    "other ABM tokens and VPP locations are left alone",
			newName: "Laptops",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{
					{OrganizationName: "Acme Inc", MacOSTeam: "Workstations"},
					{OrganizationName: "Beta Corp", MacOSTeam: "Workstations"},
				},
				abmOrgs: []string{"Acme Inc"},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{
					{Location: "Acme HQ", Teams: []string{"Workstations"}},
					{Location: "Beta HQ", Teams: []string{"Workstations"}},
				},
				vppToken: acmeToken,
			},
			wantLookup: true,
			wantSaves:  1,
			wantABM: []fleet.MDMAppleABMAssignmentInfo{
				{OrganizationName: "Acme Inc", MacOSTeam: "Laptops"},
				{OrganizationName: "Beta Corp", MacOSTeam: "Workstations"},
			},
			wantVPP: []fleet.MDMAppleVolumePurchasingProgramInfo{
				{Location: "Acme HQ", Teams: []string{"Laptops"}},
				{Location: "Beta HQ", Teams: []string{"Workstations"}},
			},
		},
		{
			name:    "one side matching still writes only once",
			newName: "Laptops",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Workstations"}},
				abmOrgs:   []string{"Acme Inc"},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations"}}},
				vppToken:  nil, // fleet has no VPP token, so the VPP copy must not move
			},
			wantLookup: true,
			wantSaves:  1,
			wantABM:    []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Laptops"}},
			wantVPP:    []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations"}}},
		},
		{
			name:    "fleet is referenced by neither",
			newName: "Laptops",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Servers"}},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Servers"}}},
			},
			wantLookup: true,
			wantABM:    []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Servers"}},
			wantVPP:    []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Servers"}}},
		},
		{
			name:    "same name is not a rename",
			newName: "Workstations",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Workstations"}},
				abmOrgs:   []string{"Acme Inc"},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations"}}},
				vppToken:  acmeToken,
			},
			wantABM: []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Workstations"}},
			wantVPP: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations"}}},
		},
	}

	ctx := test.UserContext(context.Background(), &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, ds, appCfg, saves := newStaleTeamNamesService(t, teamID, tc.fixture)
			ds.TeamWithExtrasFunc = func(_ context.Context, tid uint) (*fleet.Team, error) {
				return &fleet.Team{ID: tid, Name: "Workstations"}, nil
			}

			team, err := svc.ModifyTeam(ctx, teamID, fleet.TeamPayload{Name: new(tc.newName)})
			require.NoError(t, err)
			require.Equal(t, tc.newName, team.Name)

			require.Equal(t, tc.wantLookup, ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFuncInvoked,
				"a no-op rename must not query ABM defaults")
			require.Equal(t, tc.wantLookup, ds.GetVPPTokenByTeamIDFuncInvoked,
				"a no-op rename must not query the VPP token")

			// Both corrections share one app config write, so a case that
			// changes ABM and VPP together must still save exactly once.
			require.Equal(t, tc.wantSaves, *saves)
			require.Equal(t, tc.wantABM, appCfg.MDM.AppleBusinessManager.Value)
			require.Equal(t, tc.wantVPP, appCfg.MDM.VolumePurchasingProgram.Value)
		})
	}
}

func TestDeleteTeamCleansStaleAppConfigTeamNames(t *testing.T) {
	const teamID = uint(42)

	acmeToken := &fleet.VPPTokenDB{ID: 1, Location: "Acme HQ"}

	testCases := []struct {
		name      string
		fixture   staleTeamNamesFixture
		wantSaves int
		wantABM   []fleet.MDMAppleABMAssignmentInfo
		wantVPP   []fleet.MDMAppleVolumePurchasingProgramInfo
	}{
		{
			name: "clears the fleet everywhere it appears, in a single write",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{
					{OrganizationName: "Acme Inc", MacOSTeam: "Workstations", IOSTeam: "Phones"},
				},
				abmOrgs:   []string{"Acme Inc"},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations", "Servers"}}},
				vppToken:  acmeToken,
			},
			wantSaves: 1,
			wantABM: []fleet.MDMAppleABMAssignmentInfo{
				{OrganizationName: "Acme Inc", MacOSTeam: "", IOSTeam: "Phones"},
			},
			wantVPP: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Servers"}}},
		},
		{
			name: "other ABM tokens and VPP locations are left alone",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{
					{OrganizationName: "Acme Inc", MacOSTeam: "Workstations"},
					{OrganizationName: "Beta Corp", MacOSTeam: "Workstations"},
				},
				abmOrgs: []string{"Acme Inc"},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{
					{Location: "Acme HQ", Teams: []string{"Workstations"}},
					{Location: "Beta HQ", Teams: []string{"Workstations"}},
				},
				vppToken: acmeToken,
			},
			wantSaves: 1,
			wantABM: []fleet.MDMAppleABMAssignmentInfo{
				{OrganizationName: "Acme Inc", MacOSTeam: ""},
				{OrganizationName: "Beta Corp", MacOSTeam: "Workstations"},
			},
			wantVPP: []fleet.MDMAppleVolumePurchasingProgramInfo{
				{Location: "Acme HQ", Teams: []string{}},
				{Location: "Beta HQ", Teams: []string{"Workstations"}},
			},
		},
		{
			name: "one side matching still writes only once",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Workstations"}},
				abmOrgs:   []string{"Acme Inc"},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations"}}},
				vppToken:  nil, // fleet had no VPP token, so the VPP copy must not move
			},
			wantSaves: 1,
			wantABM:   []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: ""}},
			wantVPP:   []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Workstations"}}},
		},
		{
			name: "fleet is referenced by neither",
			fixture: staleTeamNamesFixture{
				abmStored: []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Servers"}},
				vppStored: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Servers"}}},
			},
			wantABM: []fleet.MDMAppleABMAssignmentInfo{{OrganizationName: "Acme Inc", MacOSTeam: "Servers"}},
			wantVPP: []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "Acme HQ", Teams: []string{"Servers"}}},
		},
	}

	ctx := test.UserContext(context.Background(), &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, ds, appCfg, saves := newStaleTeamNamesService(t, teamID, tc.fixture)
			ds.TeamLiteFunc = func(_ context.Context, tid uint) (*fleet.TeamLite, error) {
				return &fleet.TeamLite{ID: tid, Name: "Workstations"}, nil
			}
			ds.ListHostsFunc = func(context.Context, fleet.TeamFilter, fleet.HostListOptions) ([]*fleet.Host, error) {
				return nil, nil
			}
			ds.GetCertificateTemplatesByTeamIDFunc = func(context.Context, uint, fleet.ListOptions) (
				[]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error,
			) {
				return nil, nil, nil
			}
			ds.GetWindowsEnrollmentDefaultFleetFunc = func(context.Context) (*uint, string, error) { return nil, "", nil }
			ds.DeleteTeamFunc = func(context.Context, uint) error { return nil }

			require.NoError(t, svc.DeleteTeam(ctx, teamID))

			require.True(t, ds.GetABMTokenOrgNamesAssociatedByDefaultTeamsFuncInvoked)
			require.True(t, ds.GetVPPTokenByTeamIDFuncInvoked)

			// Both cleanups share one app config write, so a case that clears
			// ABM and VPP together must still save exactly once.
			require.Equal(t, tc.wantSaves, *saves)
			require.Equal(t, tc.wantABM, appCfg.MDM.AppleBusinessManager.Value)
			require.Equal(t, tc.wantVPP, appCfg.MDM.VolumePurchasingProgram.Value)
		})
	}
}
