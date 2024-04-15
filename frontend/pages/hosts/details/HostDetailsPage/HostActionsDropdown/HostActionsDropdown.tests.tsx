import React from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";
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
          mdmName="Fleet"
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
          mdmName="Fleet"
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
          mdmName="Fleet"
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
          mdmName="Fleet"
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
          mdmName="Non Fleet MDM"
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
          mdmName="Fleet"
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      debug();

      expect(screen.getByText("Turn off MDM").parentNode).toHaveClass(
        "is-disabled"
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
          mdmName="Fleet"
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
          mdmName="Fleet"
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Lock")).toBeInTheDocument();
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
          mdmName="Fleet"
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
          mdmName="Non Fleet MDM"
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
          mdmName="Fleet"
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
          mdmName="Fleet"
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
          mdmName="Fleet"
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
          mdmName="Non Fleet MDM"
          hostPlatform="darwin"
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Unlock")).not.toBeInTheDocument();
    });

    it("does not renders when a mac host but does not have Fleet mac mdm enabled and configured", async () => {
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
          mdmName="Fleet"
          hostPlatform="darwin"
          hostMdmDeviceStatus="locked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Unlock")).not.toBeInTheDocument();
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
          mdmName="Fleet"
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
          mdmName="Fleet"
          hostPlatform="windows"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Wipe")).not.toBeInTheDocument();
    });

    it("does not renders when a mac host but does not have Fleet mac mdm enabled and configured", async () => {
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
          mdmName="Fleet"
          hostPlatform="darwin"
          hostMdmDeviceStatus="unlocked"
          hostScriptsEnabled
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Wipe")).not.toBeInTheDocument();
    });
  });
});
