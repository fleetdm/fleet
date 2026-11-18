import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import {
  activityHandlerHasMoreActivities,
  activityHandlerHasPreviousActivities,
} from "test/handlers/activity-handlers";
import mockServer from "test/mock-server";
import {
  createCustomRenderer,
  createMockRouter,
  renderWithSetup,
} from "test/test-utils";

import ActivityFeed from "./ActivityFeed";

describe("Activity Feed", () => {
  it("renders the correct number of activities", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render
    await screen.findByText("Test User");

    expect(screen.getByText("Test User")).toBeInTheDocument();
    expect(screen.getByText("Test User 2")).toBeInTheDocument();
    expect(screen.getByText("Test User 3")).toBeInTheDocument();
  });

  it("hides pagination when there are only one page of activities", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render
    await screen.findByText("Test User");

    expect(screen.queryByText(/previous/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/next/i)).not.toBeInTheDocument();
  });

  it("enables next pagination when there are more activities", async () => {
    mockServer.use(activityHandlerHasMoreActivities);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render and pagination to be present
    await screen.findByRole("button", { name: "Next" });

    expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
  });

  it("enables previous pagination when there are more previous activities", async () => {
    mockServer.use(activityHandlerHasPreviousActivities);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { user } = render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render
    await screen.findAllByText("Test User");

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled();
  });

  it("shows the card title when activities come from a fresh cache on return to the page", async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const renderFeed = (setShowActivityFeedTitle: (show: boolean) => void) =>
      renderWithSetup(
        <QueryClientProvider client={client}>
          <ActivityFeed
            setShowActivityFeedTitle={setShowActivityFeedTitle}
            setRefetchActivities={noop}
            isPremiumTier
            router={createMockRouter()}
          />
        </QueryClientProvider>
      );

    const firstVisit = renderFeed(noop);
    await screen.findByText("Test User");
    firstVisit.unmount();

    const setShowActivityFeedTitle = jest.fn();
    renderFeed(setShowActivityFeedTitle);

    await waitFor(() =>
      expect(setShowActivityFeedTitle).toHaveBeenCalledWith(true)
    );
  });
});
