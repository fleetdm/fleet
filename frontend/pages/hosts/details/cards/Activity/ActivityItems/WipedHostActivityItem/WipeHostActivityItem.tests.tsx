import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import WipeHostActivityItem from "./WipedHostActivityItem";

describe("WipeHostActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/wiped this host/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });

  it("renders guide and feature request links for Linux hosts", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          details: { host_platform: "linux" },
        })}
        tab="past"
      />
    );

    expect(
      screen.getByRole("link", { name: /learn more/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /file a feature request/i })
    ).toBeInTheDocument();
  });

  it("does not render guide and feature request links for non-Linux hosts", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          details: { host_platform: "darwin" },
        })}
        tab="past"
      />
    );

    expect(
      screen.queryByRole("link", { name: /learn more/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /file a feature request/i })
    ).not.toBeInTheDocument();
  });
});
