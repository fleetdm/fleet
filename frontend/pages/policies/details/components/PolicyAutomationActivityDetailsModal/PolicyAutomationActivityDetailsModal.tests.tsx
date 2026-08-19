import React from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { baseUrl, createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";

import { ActivityType } from "interfaces/activity";
import { IPolicyAutomationActivity } from "interfaces/policy";

import PolicyAutomationActivityDetailsModal from "./PolicyAutomationActivityDetailsModal";

const stubScriptResult = (
  overrides: { exit_code?: number; output?: string } = {}
) =>
  mockServer.use(
    http.get(baseUrl("/scripts/results/:executionId"), () =>
      HttpResponse.json({
        hostname: "Rachael's MacBook Pro",
        host_id: 42,
        execution_id: "exec-1",
        script_contents: "#!/bin/bash\nnotify ...",
        script_id: null,
        exit_code: overrides.exit_code ?? 0,
        output: overrides.output ?? "",
        message: "",
        runtime: 1,
        host_timeout: false,
        created_at: "2026-01-01T00:00:00Z",
      })
    )
  );

// The notify branch's useQuery needs a QueryClientProvider.
const render = createCustomRenderer({ withBackendMock: true });

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

  describe("notify activities", () => {
    const notifyActivity: IPolicyAutomationActivity = {
      id: 2,
      created_at: "2026-06-12T15:04:05Z",
      type: ActivityType.NotifiedEndUserBeforePatching,
      fleet_initiated: true,
      details: {
        policy_id: 123,
        software_title: "1Password",
        time_before: 3600,
        script_execution_id: "exec-1",
      },
      host_id: 42,
      host_display_name: "Rachael's MacBook Pro",
      status: "success",
      output: null,
      pre_install_output: null,
      post_install_output: null,
    };

    it("shows the offline caveat prose on success and hides the script output behind Details", async () => {
      stubScriptResult({ exit_code: 0, output: "Notification displayed." });
      render(
        <PolicyAutomationActivityDetailsModal
          activity={notifyActivity}
          onCancel={jest.fn()}
        />
      );

      expect(
        await screen.findByText(
          /End user was notified\. Patch will be forced in 1 hour\./
        )
      ).toBeInTheDocument();
      expect(
        screen.queryByText("Notification script output:")
      ).not.toBeInTheDocument();

      // Reveal button appears once the async script fetch resolves.
      await userEvent.click(
        await screen.findByRole("button", { name: /Details/ })
      );
      expect(
        screen.getByText("Notification script output:")
      ).toBeInTheDocument();
      expect(screen.getByText("Notification displayed.")).toBeInTheDocument();
    });

    it("swaps in the 5-minute copy on the reminder", async () => {
      stubScriptResult({ exit_code: 0 });
      render(
        <PolicyAutomationActivityDetailsModal
          activity={{
            ...notifyActivity,
            details: { ...notifyActivity.details, time_before: 300 },
          }}
          onCancel={jest.fn()}
        />
      );

      expect(
        await screen.findByText(/Patch will be forced in 5 minutes\./)
      ).toBeInTheDocument();
    });

    it("appends the End user experience link for exit codes 100 and 101", async () => {
      stubScriptResult({ exit_code: 100 });
      const { unmount } = render(
        <PolicyAutomationActivityDetailsModal
          activity={{ ...notifyActivity, status: "error" }}
          onCancel={jest.fn()}
        />
      );

      const link = (await screen.findByRole("link", {
        name: /End user experience/i,
      })) as HTMLAnchorElement;
      expect(link.href).toBe(
        "https://fleetdm.com/learn-more-about/patching-end-user-experience"
      );
      expect(link.target).toBe("_blank");
      unmount();

      // Non-Fleet-Desktop failures don't get the link.
      stubScriptResult({ exit_code: 41 });
      render(
        <PolicyAutomationActivityDetailsModal
          activity={{ ...notifyActivity, status: "error" }}
          onCancel={jest.fn()}
        />
      );
      await screen.findByText(/screen was locked/i);
      expect(
        screen.queryByRole("link", { name: /End user experience/i })
      ).not.toBeInTheDocument();
    });

    it("shows the exit-code failure sentence on a notify failure", async () => {
      stubScriptResult({
        exit_code: 41,
        output: "The screen is locked.",
      });
      render(
        <PolicyAutomationActivityDetailsModal
          activity={{ ...notifyActivity, status: "error" }}
          onCancel={jest.fn()}
        />
      );

      expect(
        await screen.findByText(/The screen was locked/)
      ).toBeInTheDocument();
      await userEvent.click(screen.getByRole("button", { name: /Details/ }));
      expect(screen.getByText("The screen is locked.")).toBeInTheDocument();
    });
  });

  describe("skipped install (notify variant)", () => {
    it("shows the notify explanation and reveals the pre-install output under Details", async () => {
      const skipActivity: IPolicyAutomationActivity = {
        id: 3,
        created_at: "2026-06-12T15:04:05Z",
        type: ActivityType.InstalledSoftware,
        fleet_initiated: true,
        details: {
          policy_id: 123,
          software_title: "1Password",
          skipped_install: true,
        },
        host_id: 42,
        host_display_name: "Rachael's MacBook Pro",
        status: "error",
        output: null,
        pre_install_output:
          "Query didn't return result or failed\nThe app was open. Fleet notifies the end user 1 hour before the patch is forced.",
        post_install_output: null,
      };

      render(
        <PolicyAutomationActivityDetailsModal
          activity={skipActivity}
          onCancel={jest.fn()}
        />
      );

      expect(
        screen.getByText(
          /The app was open\. The end user will be notified before the patch is forced\./
        )
      ).toBeInTheDocument();
      expect(
        screen.queryByText("Pre-install query output:")
      ).not.toBeInTheDocument();

      await userEvent.click(screen.getByRole("button", { name: /Details/ }));
      expect(screen.getByText("Pre-install query output:")).toBeInTheDocument();
      expect(
        screen.getByText(/Fleet notifies the end user 1 hour/)
      ).toBeInTheDocument();
    });

    it("keeps the patch_when_closed skip on the old rendering (no notify explanation)", () => {
      const patchWhenClosedSkip: IPolicyAutomationActivity = {
        id: 4,
        created_at: "2026-06-12T15:04:05Z",
        type: ActivityType.InstalledSoftware,
        fleet_initiated: true,
        details: {
          policy_id: 123,
          software_title: "1Password",
          skipped_install: true,
        },
        host_id: 42,
        host_display_name: "Rachael's MacBook Pro",
        status: "error",
        output: null,
        pre_install_output:
          "Query didn't return result or failed\nThe app was open.",
        post_install_output: null,
      };

      render(
        <PolicyAutomationActivityDetailsModal
          activity={patchWhenClosedSkip}
          onCancel={jest.fn()}
        />
      );

      expect(
        screen.queryByText(/The end user will be notified/)
      ).not.toBeInTheDocument();
      // Old rendering exposes pre_install_output directly (no reveal).
      expect(screen.getByText("Pre-install query output")).toBeInTheDocument();
    });
  });
});
