import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockConfig } from "__mocks__/configMock";
import { createMockHostMdmProfile } from "__mocks__/hostMock";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

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
  props: Partial<React.ComponentProps<typeof Controls>> = {},
  { oneTimeEnrollSecrets = false }: { oneTimeEnrollSecrets?: boolean } = {}
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: createMockConfig(
          oneTimeEnrollSecrets
            ? { auth: { mdm_apple_one_time_enroll_secrets: true } }
            : {}
        ),
      },
    },
  });

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
  describe("Resend while verifying", () => {
    const verifyingFleetd = control({
      profile_uuid: "a-fleetd",
      name: "Fleetd configuration",
      status: "verifying",
    });

    it("offers Resend on the Fleetd configuration profile when one-time enroll secrets are on", () => {
      renderControls(
        { controls: [verifyingFleetd] },
        { oneTimeEnrollSecrets: true }
      );
      expect(
        screen.getByRole("button", { name: "Resend" })
      ).toBeInTheDocument();
    });

    it("disables it when one-time enroll secrets are off", () => {
      renderControls({ controls: [verifyingFleetd] });
      expect(screen.getByRole("button", { name: "Resend" })).toBeDisabled();
    });

    it("disables it to the end user", () => {
      renderControls(
        { controls: [verifyingFleetd], isDeviceUser: true },
        { oneTimeEnrollSecrets: true }
      );
      expect(screen.getByRole("button", { name: "Resend" })).toBeDisabled();
    });

    it("disables it on other verifying profiles", () => {
      renderControls(
        {
          controls: [
            control({ profile_uuid: "a", name: "Custom", status: "verifying" }),
          ],
        },
        { oneTimeEnrollSecrets: true }
      );
      expect(screen.getByRole("button", { name: "Resend" })).toBeDisabled();
    });
  });

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

  it("shows sort indicators on the sortable Name and Status columns", () => {
    renderControls({
      controls: [control({ profile_uuid: "a", name: "A", status: "verified" })],
    });

    const sortArrows = (label: string) =>
      screen
        .getAllByRole("columnheader")
        .find((th) => th.textContent === label)
        ?.querySelector(".sort-arrows");

    expect(sortArrows("Name")).toBeTruthy();
    expect(sortArrows("Status")).toBeTruthy();
    expect(sortArrows("Details")).toBeFalsy();
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
      renderControls({ canAddControls: true, isConnectedToFleetMdm: true });

      expect(screen.getByText("No controls")).toBeInTheDocument();
      expect(
        screen.getByText("No controls have been added for this host.")
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Add controls" })
      ).toBeInTheDocument();
    });

    it("hides the link for a user who can't reach the Controls page", () => {
      renderControls({ canAddControls: false, isConnectedToFleetMdm: true });

      expect(screen.getByText("No controls")).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Add controls" })
      ).not.toBeInTheDocument();
    });

    // A host Fleet has no MDM connection to can't receive controls at all, so
    // "none have been added" would point at the wrong problem.
    describe("a host not connected to Fleet MDM", () => {
      it("says the host isn't talking to Fleet for MDM features", () => {
        renderControls({ canAddControls: true, isConnectedToFleetMdm: false });

        expect(
          screen.getByText(
            "No controls available. This host isn't talking to Fleet for MDM features."
          )
        ).toBeInTheDocument();
      });

      it("says the device isn't talking to Fleet for MDM features on My device", () => {
        renderControls({ isDeviceUser: true, isConnectedToFleetMdm: false });

        expect(
          screen.getByText(
            "No controls available. Your device isn't talking to Fleet for MDM features."
          )
        ).toBeInTheDocument();
      });

      it("hides the Add controls link, which wouldn't fix anything", () => {
        renderControls({ canAddControls: true, isConnectedToFleetMdm: false });

        expect(
          screen.queryByRole("button", { name: "Add controls" })
        ).not.toBeInTheDocument();
      });
    });

    it.each([
      ["a fleet", 3, "fleet_id=3"],
      // A no-team host reports team_id: null. Dropping the param entirely
      // would land the user on whatever fleet they were last viewing.
      ["no team", null, "fleet_id=0"],
    ])("links to the Controls page for %s", (_label, teamId, expected) => {
      const router = createMockRouter();
      const { user } = renderControls({
        canAddControls: true,
        isConnectedToFleetMdm: true,
        teamId,
        router,
      });

      user.click(screen.getByRole("button", { name: "Add controls" }));

      return waitFor(() => {
        expect(router.push).toHaveBeenCalledWith(
          expect.stringContaining(expected)
        );
      });
    });

    it("hides the link on My device, which has no Controls page", () => {
      renderControls({
        isDeviceUser: true,
        canAddControls: true,
        isConnectedToFleetMdm: true,
      });

      expect(
        screen.getByText("No controls have been added for your device.")
      ).toBeInTheDocument();
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

  describe("self-service profiles", () => {
    const notInstalled = control({
      profile_uuid: "ss-1",
      name: "Opt-in",
      status: null,
      self_service: true,
    });
    const installed = control({
      profile_uuid: "ss-2",
      name: "Opted in",
      status: "verified",
      self_service: true,
    });
    const selfServiceProps = {
      isMacOSHost: true,
      canManageSelfServiceProfiles: true,
      installRequest: jest.fn(() => Promise.resolve()),
      uninstallRequest: jest.fn(() => Promise.resolve()),
    };

    it("sorts not-installed profiles first with a --- status", () => {
      renderControls({
        ...selfServiceProps,
        controls: [
          control({ profile_uuid: "f", name: "Failed", status: "failed" }),
          notInstalled,
        ],
      });
      expect(rowStatuses()).toEqual(["---", "Failed"]);
    });

    it("installs without opening the details modal", async () => {
      const installRequest = jest.fn(() => Promise.resolve());
      const { user } = renderControls({
        ...selfServiceProps,
        installRequest,
        controls: [notInstalled],
      });
      await user.click(screen.getByRole("button", { name: "Install" }));
      expect(installRequest).toHaveBeenCalledWith("ss-1");
      expect(
        screen.queryByRole("button", { name: "Close" })
      ).not.toBeInTheDocument();
    });

    it("flips Install to a disabled Resend until the host reports a new status", async () => {
      const onProfileResent = jest.fn();
      const { user } = renderControls({
        ...selfServiceProps,
        onProfileResent,
        controls: [notInstalled],
      });
      await user.click(screen.getByRole("button", { name: "Install" }));
      await waitFor(() => expect(onProfileResent).toHaveBeenCalled());
      // The refetch comes back before the reconciler has run.
      expect(screen.getByRole("button", { name: "Resend" })).toBeDisabled();
      expect(
        screen.queryByRole("button", { name: "Install" })
      ).not.toBeInTheDocument();
    });

    it("keeps Uninstall disabled until the host reports a new status", async () => {
      const onProfileResent = jest.fn();
      const props = { ...selfServiceProps, onProfileResent };
      const { user, rerender } = renderControls({
        ...props,
        controls: [installed],
      });
      await user.click(screen.getByRole("button", { name: "Uninstall" }));
      await user.click(screen.getByRole("checkbox"));
      await user.click(
        screen
          .getAllByRole("button", { name: "Uninstall" })
          .pop() as HTMLElement
      );
      await waitFor(() => expect(onProfileResent).toHaveBeenCalled());
      expect(screen.getByRole("button", { name: "Uninstall" })).toBeDisabled();

      rerender(
        <Controls
          hostDisplayName="Anna's MacBook Pro"
          canResendProfiles
          resendRequest={jest.fn()}
          router={createMockRouter()}
          {...props}
          controls={[{ ...installed, status: "failed" }]}
        />
      );
      expect(screen.getByRole("button", { name: "Uninstall" })).toBeEnabled();
    });

    it.each([true, false])(
      "requires the checkbox before uninstalling (isDeviceUser: %s)",
      async (isDeviceUser) => {
        const uninstallRequest = jest.fn(() => Promise.resolve());
        const { user } = renderControls({
          ...selfServiceProps,
          uninstallRequest,
          isDeviceUser,
          controls: [installed],
        });
        await user.click(screen.getByRole("button", { name: "Uninstall" }));
        expect(
          screen.getByText("Uninstall configuration profile")
        ).toBeInTheDocument();
        // The row button stays behind the modal; the modal's is last.
        const confirm = screen
          .getAllByRole("button", { name: "Uninstall" })
          .pop();
        expect(confirm).toBeDisabled();
        await user.click(screen.getByRole("checkbox"));
        await user.click(confirm as HTMLElement);
        expect(uninstallRequest).toHaveBeenCalledWith("ss-2");
      }
    );

    it("shows Resend and Uninstall once installed", () => {
      renderControls({ ...selfServiceProps, controls: [installed] });
      expect(screen.getByRole("button", { name: "Resend" })).toBeEnabled();
      expect(screen.getByRole("button", { name: "Uninstall" })).toBeEnabled();
    });

    it("shows only a disabled Resend while the install is pending", () => {
      renderControls({
        ...selfServiceProps,
        controls: [{ ...installed, status: "pending" }],
      });
      expect(screen.getByRole("button", { name: "Resend" })).toBeDisabled();
      expect(
        screen.queryByRole("button", { name: "Install" })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Uninstall" })
      ).not.toBeInTheDocument();
    });

    it("disables Uninstall while removing enforcement", () => {
      renderControls({
        ...selfServiceProps,
        controls: [
          { ...installed, status: "pending", operation_type: "remove" },
        ],
      });
      expect(screen.getByRole("button", { name: "Uninstall" })).toBeDisabled();
      expect(screen.getByRole("button", { name: "Resend" })).toBeDisabled();
    });

    it("has no search box on My device", () => {
      renderControls({
        ...selfServiceProps,
        isDeviceUser: true,
        controls: [installed],
      });
      expect(
        screen.queryByPlaceholderText("Search by name")
      ).not.toBeInTheDocument();
    });

    it("only flags hidden profiles with an icon on Host details", async () => {
      const hiddenRow = { ...installed, hidden: true };
      renderControls({ ...selfServiceProps, controls: [hiddenRow] });
      expect(screen.getByTestId("eye-slash-icon")).toBeInTheDocument();
    });

    it("doesn't flag hidden profiles on My device", async () => {
      const { user } = renderControls({
        ...selfServiceProps,
        isDeviceUser: true,
        controls: [{ ...installed, hidden: true }],
      });
      await user.click(
        screen.getByRole("switch", { name: "Show hidden profiles" })
      );
      expect(screen.queryByTestId("eye-slash-icon")).not.toBeInTheDocument();
    });

    it("shows the hidden profiles tooltip on the My device toggle", async () => {
      const { user } = renderControls({
        ...selfServiceProps,
        isDeviceUser: true,
        controls: [installed],
      });
      await user.hover(screen.getByText("Show hidden profiles"));
      expect(
        await screen.findByText(
          /These include automatically installed profiles that don't require action from you/
        )
      ).toBeInTheDocument();
    });

    it("hides Install and Uninstall without permission", () => {
      renderControls({
        ...selfServiceProps,
        canManageSelfServiceProfiles: false,
        controls: [notInstalled, installed],
      });
      expect(
        screen.queryByRole("button", { name: "Install" })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Uninstall" })
      ).not.toBeInTheDocument();
    });

    it("hides hidden profiles on My device until toggled on", async () => {
      const { user } = renderControls({
        ...selfServiceProps,
        isDeviceUser: true,
        controls: [{ ...installed, hidden: true }],
      });
      expect(screen.queryAllByText("Opted in")).toHaveLength(0);
      await user.click(
        screen.getByRole("switch", { name: "Show hidden profiles" })
      );
      expect(screen.getAllByText("Opted in").length).toBeGreaterThan(0);
    });
  });

  it("disables Resend with a tooltip when the user can't resend", async () => {
    const { user } = renderControls({
      canResendProfiles: false,
      controls: [control({ profile_uuid: "a", status: "verified" })],
    });
    const resend = screen.getByRole("button", { name: "Resend" });
    expect(resend).toBeDisabled();
    await user.hover(resend);
    expect(
      await screen.findByText(
        "You don't have permission to resend this profile."
      )
    ).toBeInTheDocument();
  });
});
