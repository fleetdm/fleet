import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { createMockHostMdmProfile } from "__mocks__/hostMock";

import Controls from "./Controls";
import { IHostMdmProfileWithAddedStatus } from "./OSSettingsTableConfig";

const control = (
  overrides: Partial<IHostMdmProfileWithAddedStatus>
): IHostMdmProfileWithAddedStatus =>
  createMockHostMdmProfile({
    platform: "darwin",
    operation_type: "install",
    detail: "",
    ...overrides,
  } as Parameters<typeof createMockHostMdmProfile>[0]);

const renderControls = (
  props: Partial<React.ComponentProps<typeof Controls>> = {}
) => {
  const render = createCustomRenderer({ withBackendMock: true });

  return render(
    <Controls
      controls={[]}
      hostDisplayName="Anna's MacBook Pro"
      canResendProfiles
      resendRequest={jest.fn()}
      onProfileResent={jest.fn()}
      router={createMockRouter()}
      {...props}
    />
  );
};

/** Status text of each rendered row, in render order. */
const rowStatuses = () =>
  Array.from(
    document.querySelectorAll(".os-settings-status-cell__status-text")
  ).map((el) => el.textContent);

const rowCount = () => screen.getAllByRole("row").length - 1;

describe("Controls card", () => {
  it("counts the controls", () => {
    renderControls({
      controls: [
        control({ profile_uuid: "a", name: "A", status: "verified" }),
        control({ profile_uuid: "b", name: "B", status: "verified" }),
      ],
    });

    expect(screen.getByText("2 controls")).toBeInTheDocument();
  });

  describe("Details column", () => {
    it("shows the detail when the control has one", () => {
      renderControls({
        controls: [
          control({
            profile_uuid: "a",
            name: "Edge policy",
            status: "failed",
            detail: "Error.ConfigurationCannotBeApplied",
          }),
        ],
      });

      expect(
        screen.getByText("Error.ConfigurationCannotBeApplied")
      ).toBeInTheDocument();
    });

    it("shows the detail on a control that hasn't failed", () => {
      renderControls({
        controls: [
          control({
            profile_uuid: "a",
            name: "Wi-Fi",
            status: "pending",
            detail: "Waiting for certificate to be installed on the host.",
          }),
        ],
      });

      expect(
        screen.getByText("Waiting for certificate to be installed on the host.")
      ).toBeInTheDocument();
    });

    it("shows a placeholder when the control has no detail", () => {
      renderControls({
        controls: [
          control({ profile_uuid: "a", name: "Passcode", status: "verified" }),
        ],
      });

      expect(screen.getByText("---")).toBeInTheDocument();
    });
  });

  it("sorts by status priority: failed, action required, enforcing, removing enforcement, verifying, verified", () => {
    renderControls({
      controls: [
        control({ profile_uuid: "1", name: "Verified", status: "verified" }),
        control({ profile_uuid: "2", name: "Verifying", status: "verifying" }),
        control({
          profile_uuid: "3",
          name: "Removing enforcement",
          status: "pending",
          operation_type: "remove",
        }),
        control({ profile_uuid: "4", name: "Enforcing", status: "pending" }),
        control({
          profile_uuid: "5",
          name: "Action required",
          status: "action_required",
        }),
        control({ profile_uuid: "6", name: "Failed", status: "failed" }),
      ],
    });

    expect(rowStatuses()).toEqual([
      "Failed",
      "Action required",
      "Enforcing",
      "Removing enforcement",
      "Verifying",
      "Verified",
    ]);
  });

  it("opens the details modal when a row is clicked", async () => {
    const { user } = renderControls({
      controls: [
        control({
          profile_uuid: "a",
          name: "Okta Verify settings",
          status: "verified",
        }),
      ],
    });

    await user.click(
      screen.getByText("Okta Verify settings", {
        selector: ".data-table__tooltip-truncated-text",
      })
    );

    expect(screen.getByText(/applied/, { selector: "span" })).toHaveTextContent(
      "Anna's MacBook Pro applied Okta Verify settings. Fleet verified."
    );
  });

  it("does not open the details modal when the row's Resend action is clicked", async () => {
    const resendRequest = jest.fn().mockResolvedValue(undefined);
    const { user } = renderControls({
      resendRequest,
      controls: [
        control({
          profile_uuid: "a",
          name: "Okta Verify settings",
          status: "failed",
          detail: "Something went wrong",
        }),
      ],
    });

    await user.click(screen.getByRole("button", { name: /Resend/ }));

    expect(resendRequest).toHaveBeenCalledWith("a");
    expect(screen.queryByText("Details:")).not.toBeInTheDocument();
  });

  describe("empty state", () => {
    it("offers a link to the Controls page when the user can reach it", () => {
      renderControls({ canAddControls: true });

      expect(screen.getByText("No controls")).toBeInTheDocument();
      expect(
        screen.getByText("No controls have been added for this host.")
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Add controls" })
      ).toBeInTheDocument();
    });

    it("hides the link for a user who can't reach the Controls page", () => {
      renderControls({ canAddControls: false });

      expect(screen.getByText("No controls")).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Add controls" })
      ).not.toBeInTheDocument();
    });

    it("hides the link on My device, which has no Controls page", () => {
      renderControls({ isDeviceUser: true, canAddControls: true });

      expect(
        screen.queryByRole("button", { name: "Add controls" })
      ).not.toBeInTheDocument();
    });
  });

  it("paginates after 20 controls", () => {
    renderControls({
      controls: Array.from({ length: 21 }, (_, i) =>
        control({
          profile_uuid: `p${i}`,
          name: `Profile ${i}`,
          status: "verified",
        })
      ),
    });

    expect(screen.getByText("21 controls")).toBeInTheDocument();
    expect(rowCount()).toBe(20);
    expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
  });
});
