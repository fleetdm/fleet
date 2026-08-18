import React from "react";
import { render } from "@testing-library/react";
import { noop } from "lodash";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import NotifiedEndUserBeforePatchingActivityItem from "./NotifiedEndUserBeforePatchingActivityItem";

const createNotifyActivity = (
  overrides: Partial<
    ReturnType<typeof createMockHostPastActivity>["details"]
  > = {}
) =>
  createMockHostPastActivity({
    type: ActivityType.NotifiedEndUserBeforePatching,
    actor_full_name: "Fleet",
    fleet_initiated: true,
    details: {
      host_display_name: "John's MacBook Pro",
      software_titles: ["1Password"],
      status: "success",
      time_before: 3600,
      ...overrides,
    },
  });

describe("NotifiedEndUserBeforePatchingActivityItem", () => {
  it("renders the single-app success sentence ending 'on this host.' with no host name", () => {
    const { container } = render(
      <NotifiedEndUserBeforePatchingActivityItem
        activity={createNotifyActivity()}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(container.textContent).toContain(
      "Fleet notified end user 1 hour before patching 1Password on this host."
    );
    // Host name from details is intentionally dropped on the host feed.
    expect(container.textContent).not.toContain("John's MacBook Pro");
  });

  it("renders three apps with Oxford comma", () => {
    const { container } = render(
      <NotifiedEndUserBeforePatchingActivityItem
        activity={createNotifyActivity({
          software_titles: ["1Password", "Slack", "Docker Desktop"],
        })}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(container.textContent).toContain(
      "1Password, Slack, and Docker Desktop on this host."
    );
  });

  it("truncates past three apps with pluralized 'more apps'", () => {
    const { container } = render(
      <NotifiedEndUserBeforePatchingActivityItem
        activity={createNotifyActivity({
          software_titles: [
            "1Password",
            "Slack",
            "Docker Desktop",
            "Zoom",
            "Chrome",
          ],
        })}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(container.textContent).toContain(
      "1Password, Slack, Docker Desktop, and 2 more apps"
    );
  });

  it("renders 5 minutes for the reminder", () => {
    const { container } = render(
      <NotifiedEndUserBeforePatchingActivityItem
        activity={createNotifyActivity({ time_before: 300 })}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(container.textContent).toContain(
      "notified end user 5 minutes before patching"
    );
  });

  it("renders the failed-to-notify sentence when status is failed", () => {
    const { container } = render(
      <NotifiedEndUserBeforePatchingActivityItem
        activity={createNotifyActivity({ status: "failed" })}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(container.textContent).toContain(
      "Fleet failed to notify end user 1 hour before patching 1Password on this host."
    );
  });
});
