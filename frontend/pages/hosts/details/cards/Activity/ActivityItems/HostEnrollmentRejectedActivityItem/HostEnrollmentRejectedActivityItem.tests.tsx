import { screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";
import { createCustomRenderer } from "test/test-utils";

import HostEnrollmentRejectedActivityItem from "./HostEnrollmentRejectedActivityItem";

const activity = createMockHostPastActivity({
  type: ActivityType.HostEnrollmentRejected,
  actor_full_name: "",
  actor_id: 0,
  created_at: "2026-01-01T00:00:00Z",
  details: { reason: "one_time_secret_spent", enrollment_plane: "orbit" },
});

describe("HostEnrollmentRejectedActivityItem", () => {
  it("renders Fleet as the actor with a generic sentence", () => {
    const render = createCustomRenderer();
    render(
      <HostEnrollmentRejectedActivityItem
        activity={activity}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(screen.getByText("Fleet")).toBeInTheDocument();
    expect(
      screen.getByText(/rejected an enrollment for this host\./i)
    ).toBeInTheDocument();
  });

  it("explains an automatic enrollment rejected because IdP authentication is required", () => {
    const render = createCustomRenderer();
    render(
      <HostEnrollmentRejectedActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.HostEnrollmentRejected,
          actor_full_name: "",
          actor_id: 0,
          created_at: "2026-01-01T00:00:00Z",
          details: {
            reason: "end_user_authentication_required",
            enrollment_plane: "apple_mdm",
          },
        })}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(screen.getByText("Fleet")).toBeInTheDocument();
    expect(
      screen.getByText(
        /rejected an automatic enrollment for this host because IdP authentication is required\./i
      )
    ).toBeInTheDocument();
    const link = screen.getByRole("link", { name: /learn more/i });
    expect(link).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/rejected-automatic-enrollment"
    );
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("keeps the generic sentence and no link for other reasons", () => {
    const render = createCustomRenderer();
    render(
      <HostEnrollmentRejectedActivityItem
        activity={activity}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(
      screen.getByText(/rejected an enrollment for this host\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /learn more/i })
    ).not.toBeInTheDocument();
  });

  it("passes the activity to the details handler", async () => {
    const onShowDetails = jest.fn();
    const render = createCustomRenderer();
    const { user } = render(
      <HostEnrollmentRejectedActivityItem
        activity={activity}
        tab="past"
        onShowDetails={onShowDetails}
      />
    );

    await user.click(screen.getByRole("button", { name: /show info/i }));

    expect(onShowDetails).toHaveBeenCalledWith(
      expect.objectContaining({
        type: ActivityType.HostEnrollmentRejected,
        created_at: activity.created_at,
        details: activity.details,
      })
    );
  });
});
