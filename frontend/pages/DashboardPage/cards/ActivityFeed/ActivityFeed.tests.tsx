import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  activityHandler2DaysAgo,
  activityHandler9Activities,
} from "test/handlers/activity-handlers";

import ActivityFeed from "./ActivityFeed";

describe("Activity Feed", () => {
  it("renders the correct number of activities", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<ActivityFeed setShowActivityFeedTitle={noop} />);

    // waiting for the activity data to render
    await screen.findByText("Rachel");

    expect(screen.getByText("Rachel")).toBeInTheDocument();
    expect(screen.getByText("Gabe")).toBeInTheDocument();
    expect(screen.getByText("Luke")).toBeInTheDocument();
  });
  it("disables pagination when there are no more activities", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<ActivityFeed setShowActivityFeedTitle={noop} />);

    // waiting for the activity data to render
    await screen.findByText("Rachel");

    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
  });

  it("enables next pagination when there are more activities", async () => {
    mockServer.use(activityHandler9Activities);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<ActivityFeed setShowActivityFeedTitle={noop} />);

    // waiting for the activity data to render
    await screen.findAllByText("Rachel");

    expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
  });

  it("disables previous pagination on initial page", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<ActivityFeed setShowActivityFeedTitle={noop} />);

    // waiting for the activity data to render
    await screen.findAllByText("Rachel");

    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
  });

  it("enables previous pagination when on subsequent pages", async () => {
    mockServer.use(activityHandler9Activities);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { user } = render(<ActivityFeed setShowActivityFeedTitle={noop} />);

    // waiting for the activity data to render
    await screen.findAllByText("Rachel");

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled();
  });

  it("renders avatar, actor name, timestamp", async () => {
    mockServer.use(activityHandler2DaysAgo);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<ActivityFeed setShowActivityFeedTitle={noop} />);

    // waiting for the activity data to render
    await screen.findByText("Rachel");

    expect(screen.getByRole("img")).toHaveAttribute("alt", "User avatar");
    expect(screen.getByText("Rachel")).toBeInTheDocument();
    expect(screen.getByText("2 days ago")).toBeInTheDocument();
  });
  // TODO: Create unit size component for individual activities and
  // test each activity type with different details at the unit level
});
