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

    render(<ActivityItem activity={activity} />);

    // waiting for the activity data to render
    await screen.findByText("Rachel");

    expect(screen.getByRole("img")).toHaveAttribute("alt", "User avatar");
    expect(screen.getByText("Rachel")).toBeInTheDocument();
    expect(screen.getByText("2 days ago")).toBeInTheDocument();
  });

  it("renders a default activity for activities without a specific message", () => {
    const activity = createMockActivity({
      type: ActivityType.CreatedPack,
    });
    render(<ActivityItem activity={activity} />);

    expect(screen.getByText("created pack.")).toBeInTheDocument();
  });

  it("renders a default activity for activities with a named property", () => {
    const activity = createMockActivity({
      type: ActivityType.CreatedPack,
      details: { pack_name: "Test pack" },
    });
    render(<ActivityItem activity={activity} />);

    expect(screen.getByText("created pack .")).toBeInTheDocument();
    expect(screen.getByText("Test pack")).toBeInTheDocument();
  });

  it("renders a live_query type activity", () => {
    const activity = createMockActivity({ type: ActivityType.LiveQuery });
    render(<ActivityItem activity={activity} />);

    expect(screen.getByText("ran a live query .")).toBeInTheDocument();
  });

  it("renders a live_query type activity with host count details", () => {
    const activity = createMockActivity({
      type: ActivityType.LiveQuery,
      details: {
        targets_count: 10,
      },
    });
    render(<ActivityItem activity={activity} />);

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
    render(<ActivityItem activity={activity} />);

    expect(
      screen.getByText("ran the query as a live query .")
    ).toBeInTheDocument();
    expect(screen.getByText("Test Query")).toBeInTheDocument();
    expect(screen.getByText("Show query")).toBeInTheDocument();
  });

  it("renders an applied_spec_pack type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecPack,
    });
    render(<ActivityItem activity={activity} />);

    expect(
      screen.getByText("edited a pack using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_policy type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecPolicy,
    });
    render(<ActivityItem activity={activity} />);

    expect(
      screen.getByText("edited policies using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_saved_query type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecSavedQuery,
    });
    render(<ActivityItem activity={activity} />);

    expect(
      screen.getByText("edited a query using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_saved_query type activity when run on multiple queries", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecSavedQuery,
      details: { specs: [createMockQuery(), createMockQuery()] },
    });
    render(<ActivityItem activity={activity} />);

    expect(
      screen.getByText("edited queries using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an applied_spec_team type activity for a single team", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecTeam,
      details: { teams: [createMockTeamSummary()] },
    });
    render(<ActivityItem activity={activity} />);

    expect(screen.getByText("edited team using fleetctl.")).toBeInTheDocument();
    expect(screen.getByText("Team 1")).toBeInTheDocument();
  });

  it("renders an applied_spec_team type activity for multiple team", () => {
    const activity = createMockActivity({
      type: ActivityType.AppliedSpecTeam,
      details: {
        teams: [createMockTeamSummary(), createMockTeamSummary()],
      },
    });
    render(<ActivityItem activity={activity} />);

    expect(
      screen.getByText("edited multiple teams using fleetctl.")
    ).toBeInTheDocument();
  });

  it("renders an user_added_by_sso type activity", () => {
    const activity = createMockActivity({
      type: ActivityType.UserAddedBySSO,
    });
    render(<ActivityItem activity={activity} />);

    expect(screen.getByText("was added to Fleet by SSO.")).toBeInTheDocument();
  });

  it("renders an edited_agent_options type activity for a team", () => {
    const activity = createMockActivity({
      type: ActivityType.EditedAgentOptions,
      details: { team_name: "Test Team 1" },
    });
    render(<ActivityItem activity={activity} />);

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
    render(<ActivityItem activity={activity} />);

    expect(screen.getByText("edited agent options.")).toBeInTheDocument();
  });
});
