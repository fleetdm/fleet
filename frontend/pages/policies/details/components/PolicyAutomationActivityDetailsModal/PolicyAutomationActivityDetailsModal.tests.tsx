import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ActivityType } from "interfaces/activity";
import { IPolicyAutomationActivity } from "interfaces/policy";

import PolicyAutomationActivityDetailsModal from "./PolicyAutomationActivityDetailsModal";

const failedSoftwareActivity: IPolicyAutomationActivity = {
  id: 1,
  created_at: "2026-06-12T15:04:05Z",
  type: ActivityType.InstalledSoftware,
  fleet_initiated: true,
  details: { policy_id: 123, software_title: "1Password" },
  host_id: 42,
  host_display_name: "Rachael's MacBook Pro",
  status: "error",
  output: "Failed installer: Package name is Zoom Workplace",
  pre_install_output: null,
  post_install_output: null,
};

describe("PolicyAutomationActivityDetailsModal", () => {
  it("renders the host, status, and details", () => {
    render(
      <PolicyAutomationActivityDetailsModal
        activity={failedSoftwareActivity}
        onCancel={jest.fn()}
      />
    );

    expect(
      screen.getByText("Details", { selector: ".modal__header span" })
    ).toBeInTheDocument();
    expect(screen.getByText("Rachael's MacBook Pro")).toBeInTheDocument();
    expect(screen.getByText("Software failed (1Password)")).toBeInTheDocument();
    expect(
      screen.getByText("Failed installer: Package name is Zoom Workplace")
    ).toBeInTheDocument();
  });

  it("shows the Reset policy action only when provided and invokes it", async () => {
    const onResetPolicy = jest.fn();
    const { rerender } = render(
      <PolicyAutomationActivityDetailsModal
        activity={failedSoftwareActivity}
        onCancel={jest.fn()}
      />
    );
    expect(
      screen.queryByRole("button", { name: /reset policy/i })
    ).not.toBeInTheDocument();

    rerender(
      <PolicyAutomationActivityDetailsModal
        activity={failedSoftwareActivity}
        onCancel={jest.fn()}
        onResetPolicy={onResetPolicy}
      />
    );
    await userEvent.click(
      screen.getByRole("button", { name: /reset policy/i })
    );
    expect(onResetPolicy).toHaveBeenCalledTimes(1);
  });

  it("renders separate pre-install, install, and post-install output sections for software installs", () => {
    render(
      <PolicyAutomationActivityDetailsModal
        activity={{
          ...failedSoftwareActivity,
          pre_install_output: "pre-install query returned no rows",
          output: "install script exited 1",
          post_install_output: "post-install verification failed",
        }}
        onCancel={jest.fn()}
      />
    );

    expect(screen.getByText("Pre-install query output")).toBeInTheDocument();
    expect(
      screen.getByText("pre-install query returned no rows")
    ).toBeInTheDocument();
    // The install-script section uses the "Details" label (shared with the modal
    // title), so assert on its unique output value rather than the label.
    expect(screen.getByText("install script exited 1")).toBeInTheDocument();
    expect(screen.getByText("Post-install script output")).toBeInTheDocument();
    expect(
      screen.getByText("post-install verification failed")
    ).toBeInTheDocument();
  });

  it("omits an install output section that is empty", () => {
    render(
      <PolicyAutomationActivityDetailsModal
        activity={{
          ...failedSoftwareActivity,
          pre_install_output: "pre-install query failed",
          output: null,
          post_install_output: null,
        }}
        onCancel={jest.fn()}
      />
    );

    // Only the stage that produced output is shown; empty sections (the
    // install-script and post-install stages here) are omitted.
    expect(screen.getByText("Pre-install query output")).toBeInTheDocument();
    expect(screen.getByText("pre-install query failed")).toBeInTheDocument();
    expect(
      screen.queryByText("Post-install script output")
    ).not.toBeInTheDocument();
  });

  it("omits the details box when there is no output or error", () => {
    render(
      <PolicyAutomationActivityDetailsModal
        activity={{
          ...failedSoftwareActivity,
          type: ActivityType.RanAutomationWebhook,
          status: "success",
          details: { policy_id: 123, status_code: 200 },
          output: null,
        }}
        onCancel={jest.fn()}
      />
    );

    expect(screen.getByText("Webhook queued")).toBeInTheDocument();
    // No details box (and therefore no copy button) when there's nothing to show.
    expect(screen.queryByTestId("copy-icon")).toBeNull();
  });
});
