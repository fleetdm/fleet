import React from "react";
import { render, screen } from "@testing-library/react";

import createMockActivity from "__mocks__/activityMock";
import createMockQuery from "__mocks__/queryMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import { ActivityType } from "interfaces/activity";

import ActivityItem from ".";

describe("Activity Feed", () => {
  it("renders avatar, actor name, timestamp", async () => {
    const currentDate = new Date();
    currentDate.setDate(currentDate.getDate() - 2);

    const activity = createMockActivity({
      created_at: currentDate.toISOString(),
    });

    render(<ActivityItem activity={activity} isPremiumTier />);

    // waiting for the activity data to render
    await screen.findByText("Test User");

    expect(screen.getByRole("img")).toHaveAttribute("alt", "User avatar");
    expect(screen.getByText("Test User")).toBeInTheDocument();
    expect(screen.getByText("2 days ago")).toBeInTheDocument();
  });

  it("renders a default activity for activities without a specific message", () => {
    const activity = createMockActivity({
      type: ActivityType.CreatedPack,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("created pack.")).toBeInTheDocument();
  });

  it("renders a default activity for activities with a named property", () => {
    const activity = createMockActivity({
      type: ActivityType.CreatedPack,
      details: { pack_name: "Test pack" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("created pack .")).toBeInTheDocument();
    expect(screen.getByText("Test pack")).toBeInTheDocument();
  });

  it("renders a live_query type activity", () => {
    const activity = createMockActivity({ type: ActivityType.LiveQuery });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("ran a live query .")).toBeInTheDocument();
  });

  it("renders a live_query type activity with host count details", () => {
    const activity = createMockActivity({
      type: ActivityType.LiveQuery,
      details: {
        targets_count: 10,
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("ran a live query on 10 hosts.")
    ).toBeInTheDocument();
  });

  it("renders a live_query type activity for a saved live query with targets", () => {
    const activity = createMockActivity({
      type: ActivityType.LiveQuery,
      details: {
        query_name: "Test Query",
        query_sql: "SELECT * FROM users",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText(/ran the/)).toBeInTheDocument();
    expect(screen.getByText("Test Query")).toBeInTheDocument();
    expect(screen.getByText("Show query")).toBeInTheDocument();
  });
  it("renders a live_query type activity for a saved live query with targets and performance impact", () => {
    const activity = createMockActivity({
      type: ActivityType.LiveQuery,
      details: {
        query_name: "Test Query",
        query_sql: "SELECT * FROM users",
        targets_count: 10,
        stats: {
          system_time_p50: 0,
          system_time_p95: 50.4923,
          total_executions: 345,
        },
      },
    });

    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText(/ran the/)).toBeInTheDocument();
    expect(screen.getByText("Test Query")).toBeInTheDocument();
    expect(
      screen.getByText(/with excessive performance impact on 10 hosts\./)
    ).toBeInTheDocument();
    expect(screen.getByText("Show query")).toBeInTheDocument();
  });

  it("renders a live_query type activity for a saved live query with targets and no performance impact", () => {
    const activity = createMockActivity({
      type: ActivityType.LiveQuery,
      details: {
        query_name: "Test Query",
        query_sql: "SELECT * FROM users",
        targets_count: 10,
        stats: {
          system_time_p50: 0,
          system_time_p95: 0,
          total_executions: 0,
        },
      },
    });

    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText(/ran the/)).toBeInTheDocument();
    expect(screen.getByText("Test Query")).toBeInTheDocument();
    expect(screen.queryByText(/Undetermined/)).toBeNull();
    expect(screen.getByText("Show query")).toBeInTheDocument();
  });

  it("renders an applied_spec_pack type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecPack,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited a pack using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_policy type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecPolicy,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited policies using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_saved_query type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecSavedQuery,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited a query using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_saved_query type activity when run on multiple queries", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecSavedQuery,
      details: { specs: [createMockQuery(), createMockQuery()] },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited queries using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_team type activity for a single team", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecTeam,
      details: { teams: [createMockTeamSummary()] },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited the team using fleetctl.")
    ).toBeInTheDocument();
    expect(screen.getByText("Team 1")).toBeInTheDocument();
  });

  it("renders an applied_spec_team type activity for multiple team", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecTeam,
      details: {
        teams: [createMockTeamSummary(), createMockTeamSummary()],
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited multiple teams using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an user_added_by_sso type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.UserAddedBySSO,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("was added to Fleet by SSO.")).toBeInTheDocument();
  });

  it("renders an edited_agent_options type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedAgentOptions,
      details: { team_name: "Test Team 1" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited agent options on team.")
    ).toBeInTheDocument();
    expect(screen.getByText("Test Team 1")).toBeInTheDocument();
  });

  it("renders an edited_agent_options type activity globally", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedAgentOptions,
      details: { global: true },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("edited agent options.")).toBeInTheDocument();
  });

  it("renders a user_logged_in type activity globally", () => {
    const activity = createMockActivity({
      type: ActivityType.UserLoggedIn,
      details: { public_ip: "192.168.0.1" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("successfully logged in from public IP 192.168.0.1.")
    ).toBeInTheDocument();
  });
  it("renders a user_logged_in type activity without public IP", () => {
    const activity = createMockActivity({
      type: ActivityType.UserLoggedIn,
      details: {},
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("successfully logged in.")).toBeInTheDocument();
  });

  it("renders a user_failed_login type activity globally", () => {
    const activity = createMockActivity({
      type: ActivityType.UserFailedLogin,
      details: { email: "foo@example.com", public_ip: "192.168.0.1" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(" failed to log in from public IP 192.168.0.1.", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(
      screen.getByText("foo@example.com", { exact: false })
    ).toBeInTheDocument();
  });

  // // // // // // // // // // // //
  // created_user tests
  // // // // // //// // // // // //

  it("renders a created_user type activity globally", () => {
    const activity = createMockActivity({
      type: ActivityType.UserCreated,
      details: { user_email: "newuser@example.com" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("created a user", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
  });

  it("correctly renders a created_user type activity for a premium SSO user created by JIT provisioning", () => {
    const activity = createMockActivity({
      actor_full_name: "Jit Sso",
      actor_id: 3,
      type: ActivityType.UserCreated,
      details: {
        user_id: 3,
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    //  If actor_id is the same as user_id:
    // "<name> activated their account."
    expect(screen.getByText("Jit Sso")).toBeInTheDocument();
    expect(screen.getByText(/activated their account\./)).toBeInTheDocument();
  });
  // // // // // //// // // // // //

  it("renders a deleted_user type activity globally", () => {
    const activity = createMockActivity({
      type: ActivityType.UserDeleted,
      details: { user_email: "newuser@example.com" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted a user", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
  });

  // // // // // // // // // // // //
  // changed_user_global_role  tests
  // // // // // //// // // // // //

  it("renders a changed_user_global_role type activity globally for premium users", () => {
    const activity = createMockActivity({
      type: ActivityType.UserChangedGlobalRole,
      details: { user_email: "newuser@example.com", role: "maintainer" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("changed", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    expect(
      screen.getByText("for all teams.", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders a changed_user_global_role type activity globally for free users", () => {
    const activity = createMockActivity({
      type: ActivityType.UserChangedGlobalRole,
      details: { user_email: "newuser@example.com", role: "maintainer" },
    });
    render(<ActivityItem activity={activity} isPremiumTier={false} />);

    expect(screen.getByText("changed", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    const forAllTeams = screen.queryByText("for all teams.");
    expect(forAllTeams).toBeNull();
  });

  it("correctly renders a changed_user_global_role type activity for a premium SSO user created by JIT provisioning", () => {
    const activity = createMockActivity({
      actor_id: 3,
      type: ActivityType.UserChangedGlobalRole,
      details: {
        user_id: 3,
        user_email: "jit@sso.com",
        role: "observer",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    //  If actor_id is the same as user_id:
    // "<user_email> was assigned the <role> for all teams."
    expect(screen.getByText("jit@sso.com")).toBeInTheDocument();
    expect(screen.getByText(/was assigned the/)).toBeInTheDocument();
    expect(screen.getByText("observer")).toBeInTheDocument();
    expect(screen.getByText(/role for all teams./)).toBeInTheDocument();
  });

  it("correctly renders a changed_user_global_role type activity when changing an existing user's global role, premium", () => {
    const activity = createMockActivity({
      actor_id: 1,
      actor_full_name: "Ally Admin",
      type: ActivityType.UserChangedGlobalRole,
      details: {
        user_id: 3,
        user_email: "user@example.com",
        role: "maintainer",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    //  If actor_id is different from user_id on premium:
    // "<actor_full_name> changed <user_email> to <role> for all teams."
    expect(screen.getByText("Ally Admin")).toBeInTheDocument();
    expect(screen.getByText(/changed/)).toBeInTheDocument();
    expect(screen.getByText("user@example.com")).toBeInTheDocument();
    expect(screen.getByText(/to/)).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    expect(screen.getByText(/for all teams/)).toBeInTheDocument();
  });

  it("correctly renders a changed_user_global_role type activity when changing an existing user's global role, free", () => {
    const activity = createMockActivity({
      actor_id: 1,
      actor_full_name: "Ally Admin",
      type: ActivityType.UserChangedGlobalRole,
      details: {
        user_id: 3,
        user_email: "user@example.com",
        role: "maintainer",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier={false} />);

    //  If actor_id is different from user_id on free:
    // "<actor_full_name> changed <user_email> to <role>."
    expect(screen.getByText("Ally Admin")).toBeInTheDocument();
    expect(screen.getByText("changed", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("user@example.com")).toBeInTheDocument();
    expect(screen.getByText("to", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    const forAllTeams = screen.queryByText("for all teams.");
    expect(forAllTeams).toBeNull();
  });

  // // // // // // // // // // // //
  // changed_user_team_role  tests
  // // // // // //// // // // // //

  it("renders a changed_user_team_role type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.UserChangedTeamRole,
      details: {
        user_email: "newuser@example.com",
        role: "maintainer",
        team_name: "Test Team",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("changed", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    expect(screen.getByText("Test Team")).toBeInTheDocument();
  });

  it("correctly renders a changed_user_team_role type activity when a new SSO team user is created via JIT provisioning", () => {
    const activity = createMockActivity({
      actor_id: 1,
      actor_full_name: "Ally Admin",
      type: ActivityType.UserChangedTeamRole,
      details: {
        user_id: 1,
        user_email: "jit@sso.com",
        role: "maintainer",
        team_name: "Test Team",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    // If actor_id is the same as user_id:
    // "<user_email> was assigned the <role> role for the <team_name> team."
    expect(screen.getByText("jit@sso.com")).toBeInTheDocument();
    expect(screen.getByText(/was assigned the/)).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    expect(screen.getByText(/role for the/)).toBeInTheDocument();
    expect(screen.getByText(/Test Team/)).toBeInTheDocument();
    expect(screen.getByText(/team\./)).toBeInTheDocument();

    expect(screen.queryByText("Ally Admin")).toBeNull();
    const forAllTeams = screen.queryByText("for all teams.");
    expect(forAllTeams).toBeNull();
  });

  it("correctly renders a changed_user_team_role type activity when changing an existing user's team role", () => {
    const activity = createMockActivity({
      actor_id: 1,
      actor_full_name: "Ally Admin",
      type: ActivityType.UserChangedTeamRole,
      details: {
        user_id: 3,
        user_email: "user@example.com",
        role: "maintainer",
        team_name: "Test Team",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    //  If actor_id is different from user_id:
    // "<actor_full_name> changed <user_email> to <role> for the <team_name> team."
    expect(screen.getByText("Ally Admin")).toBeInTheDocument();
    expect(screen.getByText(/changed/)).toBeInTheDocument();
    expect(screen.getByText("user@example.com")).toBeInTheDocument();
    expect(screen.getByText(/to/)).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    expect(screen.getByText(/for the/)).toBeInTheDocument();
    expect(screen.getByText(/Test Team/)).toBeInTheDocument();
    expect(screen.getByText(/team\./)).toBeInTheDocument();
    expect(screen.queryByText("for all teams.")).toBeNull();
  });

  // // // // // // // // // // // //

  it("renders a deleted_user_team_role type activity globally", () => {
    const activity = createMockActivity({
      type: ActivityType.UserDeletedTeamRole,
      details: {
        user_email: "newuser@example.com",
        team_name: "Test Team",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("removed", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
    expect(screen.getByText("Test Team")).toBeInTheDocument();
  });

  it("renders a deleted_user_global_role type activity globally for premium users", () => {
    const activity = createMockActivity({
      type: ActivityType.UserDeletedGlobalRole,
      details: { user_email: "newuser@example.com", role: "maintainer" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("removed", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    expect(
      screen.getByText("for all teams.", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders a deleted_user_global_role type activity globally for free users", () => {
    const activity = createMockActivity({
      type: ActivityType.UserDeletedGlobalRole,
      details: { user_email: "newuser@example.com", role: "maintainer" },
    });
    render(<ActivityItem activity={activity} isPremiumTier={false} />);

    expect(screen.getByText("removed", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("newuser@example.com")).toBeInTheDocument();
    expect(screen.getByText("maintainer")).toBeInTheDocument();
    const forAllTeams = screen.queryByText("for all teams.");
    expect(forAllTeams).toBeNull();
  });

  it("renders an 'enabled_disk_encryption' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.EnabledDiskEncryption,
      details: { team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("enforced disk encryption for hosts assigned to the", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("with no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders an 'enabled_macos_disk_encryption' type activity for a team", () => {
    // Test deprecated activity type
    const activity = createMockActivity({
      type: ActivityType.EnabledMacDiskEncryption,
      details: { team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("enforced disk encryption for hosts assigned to the", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("with no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'disabled_disk_encryption' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.DisabledMacDiskEncryption,
      details: { team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "removed disk encryption enforcement for hosts assigned to the",
        {
          exact: false,
        }
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("with no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'disabled_macos_disk_encryption' type activity for a team", () => {
    // Test deprecated activity type
    const activity = createMockActivity({
      type: ActivityType.DisabledDiskEncryption,
      details: { team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "removed disk encryption enforcement for hosts assigned to the",
        {
          exact: false,
        }
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("with no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders an 'enabled_disk_encryption' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.EnabledDiskEncryption,
      details: {},
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("enforced disk encryption for hosts with no team.")
    ).toBeInTheDocument();
    expect(screen.queryByText("assigned to the")).toBeNull();
  });

  it("renders an 'enabled_macos_disk_encryption' type activity for hosts with no team.", () => {
    // Test deprecated activity type
    const activity = createMockActivity({
      type: ActivityType.EnabledMacDiskEncryption,
      details: {},
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("enforced disk encryption for hosts with no team.")
    ).toBeInTheDocument();
    expect(screen.queryByText("assigned to the")).toBeNull();
  });

  it("renders a 'disabled_disk_encryption' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.DisabledDiskEncryption,
      details: {},
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "removed disk encryption enforcement for hosts with no team.",
        {
          exact: false,
        }
      )
    ).toBeInTheDocument();
    expect(screen.queryByText("assigned to the")).toBeNull();
  });

  it("renders a 'disabled_macos_disk_encryption' type activity for hosts with no team.", () => {
    // Test deprecated activity type
    const activity = createMockActivity({
      type: ActivityType.DisabledMacDiskEncryption,
      details: {},
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "removed disk encryption enforcement for hosts with no team.",
        {
          exact: false,
        }
      )
    ).toBeInTheDocument();
    expect(screen.queryByText("assigned to the")).toBeNull();
  });

  it("renders a 'changed_macos_setup_assistant' type activity for no team", () => {
    const activity = createMockActivity({
      type: ActivityType.ChangedMacOSSetupAssistant,
      details: { name: "dep-profile.json" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText((content, node) => {
        return (
          node?.innerHTML ===
          "<b>Test User </b> changed the macOS Setup Assistant (added <b>dep-profile.json</b>) for hosts that automatically enroll to no team."
        );
      })
    ).toBeInTheDocument();
  });

  it("renders a 'changed_macos_setup_assistant' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.ChangedMacOSSetupAssistant,
      details: { name: "dep-profile.json", team_name: "Workstations" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText((content, node) => {
        return (
          node?.innerHTML ===
          "<b>Test User </b> changed the macOS Setup Assistant (added <b>dep-profile.json</b>) for hosts  that automatically enroll to the <b>Workstations</b> team."
        );
      })
    ).toBeInTheDocument();
  });

  it("renders a 'deleted_macos_setup_assistant' type activity for no team", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedMacOSSetupAssistant,
      details: { name: "dep-profile.json" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText((content, node) => {
        return (
          node?.innerHTML ===
          "<b>Test User </b> changed the macOS Setup Assistant (deleted <b>dep-profile.json</b>) for hosts that automatically enroll to no team."
        );
      })
    ).toBeInTheDocument();
  });

  it("renders a 'deleted_macos_setup_assistant' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedMacOSSetupAssistant,
      details: { name: "dep-profile.json", team_name: "Workstations" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText((content, node) => {
        return (
          node?.innerHTML ===
          "<b>Test User </b> changed the macOS Setup Assistant (deleted <b>dep-profile.json</b>) for hosts  that automatically enroll to the <b>Workstations</b> team."
        );
      })
    ).toBeInTheDocument();
  });

  it("renders a 'added_bootstrap_package' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.AddedBootstrapPackage,
      details: { bootstrap_package_name: "foo.pkg", team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("added a bootstrap package (", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.pkg", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText(") for macOS hosts that automatically enroll to the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("automatically enroll to no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'deleted_bootstrap_package' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedBootstrapPackage,
      details: { bootstrap_package_name: "foo.pkg", team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted a bootstrap package (", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.pkg", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText(") for macOS hosts that automatically enroll to the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("automatically enroll to no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'added_bootstrap_package' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.AddedBootstrapPackage,
      details: { bootstrap_package_name: "foo.pkg" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("added a bootstrap package (", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.pkg", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText(
        ") for macOS hosts that automatically enroll to no team.",
        { exact: false }
      )
    ).toBeInTheDocument();
  });

  it("renders a 'deleted_bootstrap_package' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedBootstrapPackage,
      details: { bootstrap_package_name: "foo.pkg" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted a bootstrap package (", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.pkg", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText(
        ") for macOS hosts that automatically enroll to no team.",
        { exact: false }
      )
    ).toBeInTheDocument();
  });

  it("renders a 'enabled_macos_setup_end_user_auth' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.EnabledMacOSSetupEndUserAuth,
      details: { team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "required end user authentication for macOS hosts that automatically enroll to",
        { exact: false }
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'enabled_macos_setup_end_user_auth' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.EnabledMacOSSetupEndUserAuth,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "required end user authentication for macOS hosts that automatically enroll to no team.",
        { exact: false }
      )
    ).toBeInTheDocument();
  });

  it("renders a 'disabled_macos_setup_end_user_auth' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.DisabledMacOSSetupEndUserAuth,
      details: { team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "removed end user authentication requirement for macOS hosts that automatically enroll to",
        { exact: false }
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'disabled_macos_setup_end_user_auth' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.DisabledMacOSSetupEndUserAuth,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText(
        "removed end user authentication requirement for macOS hosts that automatically enroll to no team.",
        { exact: false }
      )
    ).toBeInTheDocument();
  });

  it("renders a 'transferred_hosts' type activity for one host transferred to no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.TransferredHosts,
      details: {
        host_ids: [1],
        host_display_names: ["foo"],
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("transferred host", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("no team", { exact: false })).toBeInTheDocument();
  });

  it("renders a 'transferred_hosts' type activity for one host transferred to a team.", () => {
    const activity = createMockActivity({
      type: ActivityType.TransferredHosts,
      details: {
        host_ids: [1],
        host_display_names: ["foo"],
        team_name: "Alphas",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("transferred host", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("Alphas", { exact: false })).toBeInTheDocument();
  });

  it("renders a 'transferred_hosts' type activity for multiple hosts transferred to no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.TransferredHosts,
      details: {
        host_ids: [1, 2, 3],
        host_display_names: ["foo", "bar", "baz"],
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("transferred 3 hosts", { exact: false })
    ).toBeInTheDocument();
    expect(screen.queryByText("foo")).toBeNull();
    expect(screen.queryByText("bar")).toBeNull();
    expect(screen.queryByText("baz")).toBeNull();
    expect(screen.getByText("no team", { exact: false })).toBeInTheDocument();
  });

  it("renders a 'transferred_hosts' type activity for multiple hosts transferred to a team.", () => {
    const activity = createMockActivity({
      type: ActivityType.TransferredHosts,
      details: {
        host_ids: [1, 2, 3],
        host_display_names: ["foo", "bar", "baz"],
        team_name: "Alphas",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("transferred 3 hosts", { exact: false })
    ).toBeInTheDocument();
    expect(screen.queryByText("foo")).toBeNull();
    expect(screen.queryByText("bar")).toBeNull();
    expect(screen.queryByText("baz")).toBeNull();
    expect(screen.getByText("Alphas", { exact: false })).toBeInTheDocument();
  });

  it("renders a 'mdm_enrolled' type for apple if mdm_platform is not provided", () => {
    const activity = createMockActivity({
      type: ActivityType.MdmEnrolled,
      details: {
        host_serial: "ABCD",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText((content, node) => {
        return (
          node?.innerHTML ===
          "<b>Test User </b>An end user turned on MDM features for a host with serial number <b>ABCD (manual)</b>."
        );
      })
    ).toBeInTheDocument();
  });

  it("renders a 'mdm_enrolled' type for apple with all details provided", () => {
    const activity = createMockActivity({
      type: ActivityType.MdmEnrolled,
      details: {
        host_serial: "ABCD",
        installed_from_dep: true,
        mdm_platform: "apple",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText((content, node) => {
        return (
          node?.innerHTML ===
          "<b>Test User </b>An end user turned on MDM features for a host with serial number <b>ABCD (automatic)</b>."
        );
      })
    ).toBeInTheDocument();
  });

  it("renders a 'mdm_enrolled' type activity for windows hosts.", () => {
    const activity = createMockActivity({
      type: ActivityType.MdmEnrolled,
      details: {
        mdm_platform: "microsoft",
        host_display_name: "ABCD",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText((content, node) => {
        console.log(node?.innerHTML);
        return (
          node?.innerHTML ===
          "<b>Test User </b>Mobile device management (MDM) was turned on for <b>ABCD (manual)</b>."
        );
      })
    ).toBeInTheDocument();
  });

  it("renders an 'added_script' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.AddedScript,
      details: { script_name: "foo.sh", team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("added script ", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.sh", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText(" to the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders an 'edited_script' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedScript,
      details: { team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited scripts", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText(" for the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(
      screen.getByText(" team via fleetctl.", { exact: false })
    ).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'deleted_script' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedScript,
      details: { script_name: "foo.sh", team_name: "Alphas" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted script ", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.sh", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText(" from the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders an 'added_script' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.AddedScript,
      details: { script_name: "foo.sh" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("added script ", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.sh", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText("to no team.", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders an 'edited_script' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedScript,
      details: {},
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited scripts", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("for no team via fleetctl.", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders a 'deleted_script' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedScript,
      details: { script_name: "foo.sh" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted script ", { exact: false })
    ).toBeInTheDocument();
    expect(screen.getByText("foo.sh", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByText("from no team.", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders an 'added_software' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.AddedSoftware,
      details: {
        software_title: "Foo bar",
        software_package: "foobar.pkg",
        team_name: "Alphas",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("added software ", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("foobar.pkg", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText(" to the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders an 'edited_software' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedSoftware,
      details: {
        software_title: "Foo bar",
        software_package: "foobar.pkg",
        team_name: "Alphas",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited software", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText(" on the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders a 'deleted_software' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedSoftware,
      details: {
        software_title: "Foo bar",
        software_package: "foobar.pkg",
        team_name: "Alphas",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted software ", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("foobar.pkg", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText(" from the ", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Alphas")).toBeInTheDocument();
    expect(screen.getByText(" team.", { exact: false })).toBeInTheDocument();
    const withNoTeams = screen.queryByText("no team");
    expect(withNoTeams).toBeNull();
  });

  it("renders an 'added_software' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.AddedSoftware,
      details: { software_title: "Foo bar", software_package: "foobar.pkg" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("added software ", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("foobar.pkg", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("to no team.", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders an 'edited_software' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedSoftware,
      details: {
        software_title: "Foo bar",
        software_package: "foobar.pkg",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("edited software", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("on no team", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders a 'deleted_software' type activity for hosts with no team.", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedSoftware,
      details: { software_title: "Foo bar", software_package: "foobar.pkg" },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted software ", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("foobar.pkg", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText("from no team.", { exact: false })
    ).toBeInTheDocument();
  });

  it("renders a pluralized 'deleted_multiple_saved_query' type activity when deleting multiple queries.", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedMultipleSavedQuery,
      details: {
        query_ids: [1, 2, 3],
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(
      screen.getByText("deleted multiple queries", { exact: false })
    ).toBeInTheDocument();
  });
  // test for wipe activity
  it("renders a 'wiped_host' type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.WipedHost,
      details: {
        host_display_name: "Foo Host",
      },
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText("wiped", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("Foo Host", { exact: false })).toBeInTheDocument();
  });

  it("renders the correct actor for a installed_software activity without self_service", () => {
    const activity = createMockActivity({
      type: ActivityType.InstalledSoftware,
      actor_id: 1,
      actor_full_name: "Test Admin",
      details: {
        software_title: "Foo Software",
        host_display_name: "Foo Host",
      },
    });

    render(<ActivityItem activity={activity} isPremiumTier />);
    expect(screen.getByText("Test Admin")).toBeInTheDocument();
  });

  it("renders the correct actor for a installed_software activity that was self_service", () => {
    const activity = createMockActivity({
      type: ActivityType.InstalledSoftware,
      actor_id: 1,
      details: {
        software_title: "Foo Software",
        self_service: true,
        host_display_name: "Foo Host",
      },
    });

    render(<ActivityItem activity={activity} isPremiumTier />);
    expect(screen.getByText("An end user")).toBeInTheDocument();
  });

  it("renders the correct actor for a installed_app_store_app activity without self_service", () => {
    const activity = createMockActivity({
      type: ActivityType.InstalledAppStoreApp,
      actor_id: 1,
      actor_full_name: "Test Admin",
      details: {
        software_title: "Foo Software",
        host_display_name: "Foo Host",
      },
    });

    render(<ActivityItem activity={activity} isPremiumTier />);
    expect(screen.getByText("Test Admin")).toBeInTheDocument();
  });

  it("renders the correct actor for a installed_app_store_app activity that was self_service", () => {
    const activity = createMockActivity({
      type: ActivityType.InstalledAppStoreApp,
      actor_id: 1,
      details: {
        software_title: "Foo Software",
        self_service: true,
        host_display_name: "Foo Host",
      },
    });

    render(<ActivityItem activity={activity} isPremiumTier />);
    expect(screen.getByText("An end user")).toBeInTheDocument();
  });

  it("renders addedNdesScepProxy activity correctly", () => {
    const activity = createMockActivity({
      type: ActivityType.AddedNdesScepProxy,
    });
    render(<ActivityItem activity={activity} isPremiumTier={false} />);

    expect(screen.getByText(/Test User/)).toBeInTheDocument();
    expect(
      screen.getByText(
        /added Microsoft's Network Device Enrollment Service \(NDES\) as your SCEP server/
      )
    ).toBeInTheDocument();
  });

  it("renders editedNdesScepProxy activity correctly", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedNdesScepProxy,
    });
    render(<ActivityItem activity={activity} isPremiumTier={false} />);

    expect(screen.getByText(/Test User/)).toBeInTheDocument();
    expect(
      screen.getByText(
        /edited configurations for Microsoft's Network Device Enrollment Service \(NDES\) as your SCEP server/
      )
    ).toBeInTheDocument();
  });

  it("renders deletedNdesScepProxy activity correctly", () => {
    const activity = createMockActivity({
      type: ActivityType.DeletedNdesScepProxy,
    });
    render(<ActivityItem activity={activity} isPremiumTier={false} />);

    expect(screen.getByText(/Test User/)).toBeInTheDocument();
    expect(
      screen.getByText(
        /removed Microsoft's Network Device Enrollment Service \(NDES\) as your SCEP server/
      )
    ).toBeInTheDocument();
  });

  it("renders setup experience installed software correctly", () => {
    const activity = createMockActivity({
      type: ActivityType.InstalledSoftware,
      actor_full_name: "",
      actor_email: "",
      actor_id: undefined,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText(/Fleet/)).toBeInTheDocument();
  });

  it("renders setup experience ran script correctly", () => {
    const activity = createMockActivity({
      type: ActivityType.RanScript,
      actor_full_name: "",
      actor_email: "",
      actor_id: undefined,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText(/Fleet/)).toBeInTheDocument();
  });

  it("renders setup experience installed VPP app correctly", () => {
    const activity = createMockActivity({
      type: ActivityType.RanScript,
      actor_full_name: "",
      actor_email: "",
      actor_id: undefined,
    });
    render(<ActivityItem activity={activity} isPremiumTier />);

    expect(screen.getByText(/Fleet/)).toBeInTheDocument();
  });
});
