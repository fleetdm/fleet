import React from "react";
import { noop } from "lodash";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import createMockTeam from "__mocks__/teamMock";

import HostActionsDropdown from "./HostActionsDropdown";

describe("Host Actions Dropdown", () => {
  describe("Transfer action", () => {
    it("renders the Transfer action when on premium tier and the user is a global admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Transfer")).toBeInTheDocument();
    });

    it("renders the Transfer action when on premium tier and the user is a global maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalMaintainer: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Transfer")).toBeInTheDocument();
    });
  });
  describe("Query action", () => {
    it("renders the Query action when the user is a global admin and the host is online", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Query")).toBeInTheDocument();
    });

    it("renders the Query action as disabled with a tooltip when a host is offline", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="offline"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.getByText("Query").parentElement?.parentElement?.parentElement
      ).toHaveClass("actions-dropdown-select__option--is-disabled");

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Query"));
        });

        expect(
          screen.getByText(/You can't query an offline host./i)
        ).toBeInTheDocument();
      });
    });

    it("renders the Query action as disabled when a host is locked", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="offline"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));
      expect(
        screen.getByText("Query").parentElement?.parentElement?.parentElement
      ).toHaveClass("actions-dropdown-select__option--is-disabled");
    });

    it("renders the Query action as disabled when a host is updating", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="locking"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Query").parentElement).toHaveClass(
        "actions-dropdown-select__option--is-disabled"
      );
    });
  });

  it("renders the Show Disk Encryption Key action when on premium tier and we store the disk encryption key", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          currentUser: createMockUser(),
        },
      },
    });

    const { user } = render(
      <HostActionsDropdown
        hostTeamId={null}
        onSelect={noop}
        hostStatus="online"
        hostMdmEnrollmentStatus={null}
        doesStoreEncryptionKey
        hostMdmDeviceStatus="unlocked"
        hostScriptsEnabled
      />
    );

    await user.click(screen.getByText("Actions"));

    expect(screen.getByText("Show disk encryption key")).toBeInTheDocument();
  });

  describe("Turn off MDM action", () => {
    it("renders the action when the host is enrolled in mdm and the user is a global admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("renders the action when the host is enrolled in mdm and the user is a global maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            isGlobalMaintainer: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("renders the action when the host is enrolled in mdm and the user is a team admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            currentUser: createMockUser({
              teams: [createMockTeam({ id: 1, role: "admin" })],
            }),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={1}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("renders the action when the host is enrolled in mdm and the user is at least a team maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            currentUser: createMockUser({
              teams: [createMockTeam({ id: 1, role: "maintainer" })],
            }),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={1}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("does not render the action when the host is enrolled in a non Fleet MDM solution", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm={false}
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Turn off MDM")).not.toBeInTheDocument();
    });

    it("renders as disabled when the host is offline", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user, debug } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="offline"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      debug();

      expect(screen.getByText("Turn off MDM").parentElement).toHaveClass(
        "actions-dropdown-select__option--is-disabled"
      );
    });

    it("does not render the action when the host platform is not darwin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostTeamId={1}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="windows"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Turn off MDM")).not.toBeInTheDocument();
    });
  });

  describe("Delete action", () => {
    it("renders when the user is a global admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Delete")).toBeInTheDocument();
    });

    it("renders when the user is a global maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalMaintainer: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Delete")).toBeInTheDocument();
    });

    it("renders when the user is a team admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            currentUser: createMockUser({
              teams: [createMockTeam({ id: 1, role: "admin" })],
            }),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={1}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Delete")).toBeInTheDocument();
    });

    it("renders when the user is a team maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            currentUser: createMockUser({
              teams: [createMockTeam({ id: 1, role: "maintainer" })],
            }),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={1}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Delete")).toBeInTheDocument();
    });
  });

  describe("Lock action", () => {
    it("renders when the host is enrolled in mdm and the mdm is enabled and host is unlocked", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Lock")).toBeInTheDocument();
    });

    it("renders as disabled with a tooltip when scripts_enabled is set to false for windows/linux", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="debian"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.getByText("Lock").parentElement?.parentElement?.parentElement
      ).toHaveClass("actions-dropdown-select__option--is-disabled");

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Lock"));
        });

        expect(
          screen.getByText(/fleetd agent with --enable-scripts/i)
        ).toBeInTheDocument();
      });
    });

    it("does not render when the host is not enrolled in mdm", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="Off"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Lock")).not.toBeInTheDocument();
    });

    it("does not render when the host is not enrolled in a Fleet MDM solution", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm={false}
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Lock")).not.toBeInTheDocument();
    });
  });

  describe("Unlock action", () => {
    it("renders when the host is enrolled in mdm and the mdm is enabled and host is locked", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Unlock")).toBeInTheDocument();
    });

    it("renders when the host is enrolled in mdm and the mdm is enabled and host is unlocking", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocking"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Unlock")).toBeInTheDocument();
    });

    it("does not render when the host is not enrolled in mdm", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="Off"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Unlock")).not.toBeInTheDocument();
    });

    it("does not render when the host is not enrolled in a Fleet MDM solution", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm={false}
          hostPlatform="darwin"
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Unlock")).not.toBeInTheDocument();
    });

    it("does not renders when a macOS host but does not have Fleet mac mdm enabled and configured", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: false,
            isWindowsMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Unlock")).not.toBeInTheDocument();
    });

    it("renders as disabled with a tooltip when scripts_enabled is set to false for windows/linux", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="offline"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="windows"
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.getByText("Unlock").parentElement?.parentElement?.parentElement
      ).toHaveClass("actions-dropdown-select__option--is-disabled");

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Unlock"));
        });

        expect(
          screen.getByText(/fleetd agent with --enable-scripts/i)
        ).toBeInTheDocument();
      });
    });
  });

  describe("Wipe action", () => {
    it("renders only when the host is unlocked", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Wipe")).toBeInTheDocument();
    });

    it("does not renders when a windows host but does not have Fleet windows mdm enabled and configured", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isWindowsMdmEnabledAndConfigured: false,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="windows"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Wipe")).not.toBeInTheDocument();
    });

    it("does not render for a macOS host if Fleet Apple MDM is not enabled and configured", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: false,
            isWindowsMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Wipe")).not.toBeInTheDocument();
    });

    it("renders as disabled with a tooltip when scripts_enabled is set to false for linux", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostPlatform="debian"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.getByText("Wipe").parentElement?.parentElement?.parentElement
      ).toHaveClass("actions-dropdown-select__option--is-disabled");

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Wipe"));
        });

        expect(
          screen.getByText(/fleetd agent with --enable-scripts/i)
        ).toBeInTheDocument();
      });
    });
  });

  describe("Run script action", () => {
    it("renders the Run script action when scripts_enabled is set to true", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="offline"
          isConnectedToFleetMdm
          hostPlatform="windows"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Run script")).toBeInTheDocument();
    });

    it("renders the Run script action as enabled when `scripts_enabled` is `null`", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="offline"
          isConnectedToFleetMdm
          hostPlatform="windows"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled={null}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Run script")).toBeInTheDocument();

      expect(
        screen
          .getByText("Run script")
          .parentElement?.parentElement?.parentElement?.classList.contains(
            "actions-dropdown-select__option--is-disabled"
          )
      ).toBeFalsy();

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Run script"));
        });

        expect(
          screen.queryByText(/fleetd agent with --enable-scripts/i)
        ).toBeNull();
      });
    });

    it("renders the Run script action as disabled with a tooltip when scripts_enabled is set to false", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          isConnectedToFleetMdm
          hostPlatform="darwin"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.getByText("Run script").parentElement?.parentElement
          ?.parentElement
      ).toHaveClass("actions-dropdown-select__option--is-disabled");

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Run script"));
        });

        expect(
          screen.getByText(/fleetd agent with --enable-scripts/i)
        ).toBeInTheDocument();
      });
    });

    it("does not render the Run script action for ChromeOS", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostPlatform="chrome"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus={"unlocked"}
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
    });
    it("does not render the Run script action for global observers/+", async () => {
      // Global observer
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalObserver: true,
            currentUser: createMockUser(),
          },
        },
      });
      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="offline"
          isConnectedToFleetMdm
          hostPlatform="windows"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
    });
    it("does not render the Run script action for team observers/+", async () => {
      // team observer
      const render = createCustomRenderer({
        context: {
          app: {
            isTeamObserver: true,
            currentUser: createMockUser(),
          },
        },
      });
      const { user } = render(
        <HostActionsDropdown
          hostTeamId={1}
          onSelect={noop}
          hostStatus="offline"
          isConnectedToFleetMdm
          hostPlatform="windows"
          hostMdmEnrollmentStatus={null}
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
    });
  });

  describe("Render options only available for iOS and iPadOS", () => {
    it("renders only the transfer, wipe, and delete options for iOS", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            isMacMdmEnabledAndConfigured: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostPlatform="ios"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus={"unlocked"}
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Transfer")).toBeInTheDocument();
      expect(screen.queryByText("Wipe")).toBeInTheDocument();
      expect(screen.queryByText("Delete")).toBeInTheDocument();

      expect(screen.queryByText("Query")).not.toBeInTheDocument();
      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
      expect(screen.queryByText("Turn off MDM")).not.toBeInTheDocument();
      expect(screen.queryByText("Lock")).not.toBeInTheDocument();
    });

    it("renders only the transfer, wipe, and delete options for iPadOS", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            isMacMdmEnabledAndConfigured: true,
            currentUser: createMockUser(),
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          hostTeamId={null}
          onSelect={noop}
          hostStatus="online"
          hostPlatform="ipados"
          hostMdmEnrollmentStatus="On (automatic)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus={"unlocked"}
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Transfer")).toBeInTheDocument();
      expect(screen.queryByText("Wipe")).toBeInTheDocument();
      expect(screen.queryByText("Delete")).toBeInTheDocument();

      expect(screen.queryByText("Query")).not.toBeInTheDocument();
      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
      expect(screen.queryByText("Turn off MDM")).not.toBeInTheDocument();
      expect(screen.queryByText("Lock")).not.toBeInTheDocument();
    });
  });
});
