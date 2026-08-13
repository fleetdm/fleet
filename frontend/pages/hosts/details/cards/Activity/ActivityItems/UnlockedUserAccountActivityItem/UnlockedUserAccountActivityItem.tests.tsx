import React from "react";

import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { ActivityType, IHostPastActivity } from "interfaces/activity";

import UnlockedUserAccountActivityItem from "./UnlockedUserAccountActivityItem";

describe("UnlockedUserAccountActivityItem", () => {
  it("renders the username and exposes command details", async () => {
    const onShowDetails = jest.fn();
    const activity: IHostPastActivity = {
      id: 1,
      created_at: "2026-08-12T12:00:00Z",
      actor_full_name: "Jay Moore",
      actor_id: 1,
      actor_gravatar: "",
      actor_api_only: false,
      fleet_initiated: false,
      type: ActivityType.UnlockedUserAccount,
      details: {
        username: "anna",
        command_uuid: "command-uuid",
        host_uuid: "host-uuid",
      },
    };
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(
      <UnlockedUserAccountActivityItem
        activity={activity}
        tab="past"
        onShowDetails={onShowDetails}
      />
    );

    expect(screen.getByText("anna")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "show info" }));
    expect(onShowDetails).toHaveBeenCalledWith(
      expect.objectContaining({
        type: ActivityType.UnlockedUserAccount,
        details: activity.details,
      })
    );
  });
});
