import React from "react";
import { noop } from "lodash";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import createMockTeam from "__mocks__/teamMock";

import HostActionsDropdown from "./HostActionsDropdown";
import { HostMdmDeviceStatusUIState } from "../../helpers";

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

      expect(screen.getByText("Live report")).toBeInTheDocument();
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
        screen.getByText("Live report").parentElement?.parentElement
          ?.parentElement
      ).toHaveClass("actions-dropdown-select__option--is-disabled");

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Live report"));
        });

        expect(
          screen.getByText(/You can't run a live report on an offline host./i)
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
        screen.getByText("Live report").parentElement?.parentElement
          ?.parentElement
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

      expect(screen.getByText("Live report").parentElement).toHaveClass(
        "actions-dropdown-select__option--is-disabled"
      );
    });
  });

  describe("Show disk encryption key action", () => {
    it("hides the show disk encryption key action for macOS device when key is stored but device is not connected to Fleet MDM", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          isConnectedToFleetMdm={false}
          hostPlatform="darwin"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
    });

    it("hides the show disk encryption key action for iOS device", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          isConnectedToFleetMdm
          hostPlatform="ios"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
    });

    it("hides the show disk encryption key action for Android device", async () => {
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
          doesStoreEncryptionKey
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="android"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
    });

    it("includes the show disk encryption key action when key is stored and macOS device is connected to Fleet MDM", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          isConnectedToFleetMdm
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Show disk encryption key")).toBeInTheDocument();
    });

    it("includes the show disk encryption key action when key is stored for Linux device", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostPlatform="debian"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Show disk encryption key")).toBeInTheDocument();
    });

    it("includes the show disk encryption key action when key is stored for Windows device", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostPlatform="windows"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Show disk encryption key")).toBeInTheDocument();
    });
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

      const { user } = render(
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
            config: {
              server_settings: {
                scripts_disabled: false, // scriptsGloballyDisabled = false
              },
            },
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

    it("renders the Run script action as enabled when scripts_enabled is null", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
            config: {
              server_settings: {
                scripts_disabled: false,
              },
            },
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
            config: {
              server_settings: {
                scripts_disabled: false,
              },
            },
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

      await waitFor(() => user.hover(screen.getByText("Run script")));
      expect(
        screen.getByText(/fleetd agent with --enable-scripts/i)
      ).toBeInTheDocument();
    });

    it("renders the Run script action as disabled when scripts are disabled globally", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            currentUser: createMockUser(),
            config: {
              server_settings: {
                scripts_disabled: true, // scriptsGloballyDisabled = true
              },
            },
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
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByText("Run script"));
        });

        expect(
          screen.getByText(
            /Running scripts is disabled in organization settings./i
          )
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
          hostMdmDeviceStatus="unlocked"
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
    it("renders only the transfer, wipe, clear passcode, and delete options for iOS", async () => {
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
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Transfer")).toBeInTheDocument();
      expect(screen.queryByText("Wipe")).toBeInTheDocument();
      expect(screen.queryByText("Clear passcode")).toBeInTheDocument();
      expect(screen.queryByText("Delete")).toBeInTheDocument();

      expect(screen.queryByText("Live report")).not.toBeInTheDocument();
      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
    });

    it("renders only the transfer, wipe, clear passcode, and delete options for iPadOS", async () => {
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
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Transfer")).toBeInTheDocument();
      expect(screen.queryByText("Wipe")).toBeInTheDocument();
      expect(screen.queryByText("Clear passcode")).toBeInTheDocument();
      expect(screen.queryByText("Delete")).toBeInTheDocument();

      expect(screen.queryByText("Live report")).not.toBeInTheDocument();
      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
    });
  });

  describe("personally enrolled hosts (e.g. enrollment status => On (personal)", () => {
    it("render only the Transfer and Delete options for personally enrolled ios host", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
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
          hostMdmEnrollmentStatus={"On (personal)"}
          hostMdmDeviceStatus="unlocked"
          isConnectedToFleetMdm
          hostScriptsEnabled
          hostPlatform="ios"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Transfer")).toBeInTheDocument();
      expect(screen.getByText("Delete")).toBeInTheDocument();
      expect(screen.queryByText("Live report")).not.toBeInTheDocument();
      expect(screen.queryByText("Clear passcode")).not.toBeInTheDocument();
      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
      expect(screen.queryByText("Wipe")).not.toBeInTheDocument();
      expect(screen.queryByText("Lock")).not.toBeInTheDocument();
      expect(screen.queryByText("Unlock")).not.toBeInTheDocument();
      expect(screen.queryByText("Turn off MDM")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
    });

    it("render only the Transfer and Delete options for personally enrolled ipad host", async () => {
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
          hostMdmEnrollmentStatus={"On (personal)"}
          isConnectedToFleetMdm
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          hostPlatform="ipados"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Transfer")).toBeInTheDocument();
      expect(screen.getByText("Delete")).toBeInTheDocument();
      expect(screen.queryByText("Live report")).not.toBeInTheDocument();
      expect(screen.queryByText("Clear passcode")).not.toBeInTheDocument();
      expect(screen.queryByText("Run script")).not.toBeInTheDocument();
      expect(screen.queryByText("Wipe")).not.toBeInTheDocument();
      expect(screen.queryByText("Lock")).not.toBeInTheDocument();
      expect(screen.queryByText("Unlock")).not.toBeInTheDocument();
      expect(screen.queryByText("Turn off MDM")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Show disk encryption key")
      ).not.toBeInTheDocument();
    });
  });

  describe("Show Recovery Lock password action", () => {
    it("renders the action when recovery lock is enabled and host is macOS connected to Fleet MDM", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isRecoveryLockPasswordEnabled
        />
      );

      expect(
        screen.queryByText("Show Recovery Lock password")
      ).not.toBeInTheDocument();

      await user.click(screen.getByText("Actions"));

      expect(
        screen.getByText("Show Recovery Lock password")
      ).toBeInTheDocument();
    });

    it("hides the action when recovery lock is not enabled", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isRecoveryLockPasswordEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show Recovery Lock password")
      ).not.toBeInTheDocument();
    });

    it("hides the action for non-macOS hosts", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="windows"
          isRecoveryLockPasswordEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show Recovery Lock password")
      ).not.toBeInTheDocument();
    });

    it("hides the action when host is not connected to Fleet MDM", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm={false}
          hostPlatform="darwin"
          isRecoveryLockPasswordEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show Recovery Lock password")
      ).not.toBeInTheDocument();
    });

    it("disables the action when password is not available", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isRecoveryLockPasswordEnabled
          recoveryLockPasswordAvailable={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      const option = screen.getByText("Show Recovery Lock password");
      expect(option).toBeInTheDocument();
      expect(option).toHaveAttribute("aria-disabled", "true");

      await user.hover(option);
      await waitFor(() => {
        expect(
          screen.getByText(/Recovery Lock password is unavailable/i)
        ).toBeInTheDocument();
      });
    });
  });

  describe("Show managed account action", () => {
    it("renders the action when managed local account is enabled and host is ADE-enrolled macOS connected to Fleet MDM", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
          managedAccountStatus="verified"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Show managed account")).toBeInTheDocument();
    });

    it("hides the action when managed local account is not enabled", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled={false}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show managed account")
      ).not.toBeInTheDocument();
    });

    it("hides the action for non-ADE enrolled hosts (manual enrollment)", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (manual)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show managed account")
      ).not.toBeInTheDocument();
    });

    it("hides the action for non-macOS hosts", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="windows"
          isManagedLocalAccountEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show managed account")
      ).not.toBeInTheDocument();
    });

    it("hides the action when host is not connected to Fleet MDM", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm={false}
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show managed account")
      ).not.toBeInTheDocument();
    });

    it("hides the action on free tier", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            isPremiumTier: false,
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
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(
        screen.queryByText("Show managed account")
      ).not.toBeInTheDocument();
    });

    it("disables the action with 'still being created' tooltip when status is pending", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
          managedAccountStatus="pending"
        />
      );

      await user.click(screen.getByText("Actions"));

      const option = screen.getByText("Show managed account");
      expect(option).toBeInTheDocument();
      expect(option).toHaveAttribute("aria-disabled", "true");

      await user.hover(option);
      await waitFor(() => {
        expect(
          screen.getByText(/The managed account is still being/i)
        ).toBeInTheDocument();
      });
    });

    it("disables the action with 'failed' tooltip when status is failed", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
          managedAccountStatus="failed"
        />
      );

      await user.click(screen.getByText("Actions"));

      const option = screen.getByText("Show managed account");
      expect(option).toBeInTheDocument();
      expect(option).toHaveAttribute("aria-disabled", "true");

      await user.hover(option);
      await waitFor(() => {
        expect(
          screen.getByText(/The managed account failed to be/i)
        ).toBeInTheDocument();
      });
    });

    it("disables the action with 'next enrollment' tooltip when status is null (no record)", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (automatic)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
          managedAccountStatus={null}
        />
      );

      await user.click(screen.getByText("Actions"));

      const option = screen.getByText("Show managed account");
      expect(option).toBeInTheDocument();
      expect(option).toHaveAttribute("aria-disabled", "true");

      await user.hover(option);
      await waitFor(() => {
        expect(screen.getByText(/at the next enrollment/i)).toBeInTheDocument();
      });
    });

    it("renders the action for company-owned ADE enrollment status", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
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
          hostMdmEnrollmentStatus="On (company-owned)"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
          isConnectedToFleetMdm
          hostPlatform="darwin"
          isManagedLocalAccountEnabled
          managedAccountStatus="verified"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Show managed account")).toBeInTheDocument();
    });
  });

  describe("Clear passcode action", () => {
    it("renders the action when an iOS host is enrolled in MDM", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            isPremiumTier: true,
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
          hostMdmEnrollmentStatus="On (company-owned)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Clear passcode")).toBeInTheDocument();
    });

    it("does not render for below maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalTechnician: true,
            isGlobalAdmin: false,
            isTeamMaintainer: false,
            isTeamAdmin: false,
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: true,
            currentUser: createMockUser(),
          },
        },
      });

      render(
        <HostActionsDropdown
          hostTeamId={1}
          onSelect={noop}
          hostStatus="online"
          hostPlatform="ios"
          hostMdmEnrollmentStatus="On (company-owned)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      // Component returns null when no options are available for this role,
      // so neither the Actions button nor Clear passcode are rendered.
      expect(screen.queryByText("Actions")).not.toBeInTheDocument();
      expect(screen.queryByText("Clear passcode")).not.toBeInTheDocument();
    });

    it("does not render for non-iOS hosts", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            isPremiumTier: true,
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
          hostPlatform="darwin"
          hostMdmEnrollmentStatus="On (company-owned)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Clear passcode")).not.toBeInTheDocument();
    });

    it("is hidden if Apple MDM is not enabled", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            isPremiumTier: true,
            isMacMdmEnabledAndConfigured: false,
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
          hostMdmEnrollmentStatus="On (company-owned)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Clear passcode")).not.toBeInTheDocument();
    });

    it("is hidden on Fleet free", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            isPremiumTier: false,
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
          hostMdmEnrollmentStatus="On (company-owned)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Clear passcode")).not.toBeInTheDocument();
    });

    it.each<HostMdmDeviceStatusUIState>([
      "locked",
      "locking",
      "unlocking",
      "locating",
    ])("is disabled with tooltip when host status is %s", async (status) => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
            isPremiumTier: true,
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
          hostMdmEnrollmentStatus="On (company-owned)"
          isConnectedToFleetMdm
          hostMdmDeviceStatus={status}
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      const option = screen.getByText("Clear passcode");
      expect(option).toBeInTheDocument();
      expect(option).toHaveAttribute("aria-disabled", "true");

      await user.hover(option);

      await waitFor(() => {
        expect(
          screen.getByText(
            /Clear passcode is unavailable while host is in Lost Mode./i
          )
        ).toBeInTheDocument();
      });
    });

    it.each<HostMdmDeviceStatusUIState>(["wiping", "wiped"])(
      "is disabled with tooltip when pending wipe",
      async (status) => {
        const render = createCustomRenderer({
          context: {
            app: {
              isGlobalAdmin: true,
              isPremiumTier: true,
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
            hostMdmEnrollmentStatus="On (company-owned)"
            isConnectedToFleetMdm
            hostMdmDeviceStatus={status}
            hostScriptsEnabled
          />
        );

        await user.click(screen.getByText("Actions"));

        const option = screen.getByText("Clear passcode");
        expect(option).toBeInTheDocument();
        expect(option).toHaveAttribute("aria-disabled", "true");

        await user.hover(option);

        await waitFor(() => {
          expect(
            screen.getByText(
              /Clear passcode is unavailable while host is pending wipe./i
            )
          ).toBeInTheDocument();
        });
      }
    );
  });
});
