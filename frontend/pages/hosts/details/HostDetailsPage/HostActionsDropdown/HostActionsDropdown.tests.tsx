import React from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import HostActionsDropdown from "./HostActionsDropdown";

describe("Host Actions Dropdown", () => {
  describe("Transfer action", () => {
    it("renders the Transfer action when on premium tier and the user is a global admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus={null}
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
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus={null}
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
        },
      },
    });

    const { user } = render(
      <HostActionsDropdown
        onSelect={noop}
        hostStatus="online"
        hostMdmEnrollemntStatus={null}
        doesStoreEncryptionKey
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
            isMdmEnabledAndConfigured: true,
            isGlobalAdmin: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
          mdmName="Fleet"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("renders the action when the host is enrolled in mdm and the user is a global maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMdmEnabledAndConfigured: true,
            isGlobalMaintainer: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
          mdmName="Fleet"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("renders the action when the host is enrolled in mdm and the user is a team admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMdmEnabledAndConfigured: true,
            isTeamAdmin: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
          mdmName="Fleet"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("renders the action when the host is enrolled in mdm and the user is a team maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMdmEnabledAndConfigured: true,
            isTeamMaintainer: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
          mdmName="Fleet"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Turn off MDM")).toBeInTheDocument();
    });

    it("does not render the action when the host is enrolled in a non Fleet MDM solution", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMdmEnabledAndConfigured: true,
            isTeamMaintainer: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
          mdmName="Non Fleet MDM"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Turn off MDM")).not.toBeInTheDocument();
    });

    it("renders as disabled when the host is offline", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isMdmEnabledAndConfigured: true,
            isTeamMaintainer: true,
          },
        },
      });

      const { user, debug } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="offline"
          hostMdmEnrollemntStatus="On (automatic)"
          mdmName="Fleet"
        />
      );

      await user.click(screen.getByText("Actions"));

      debug();

      expect(screen.getByText("Turn off MDM").parentNode).toHaveClass(
        "is-disabled"
      );
    });
  });

  describe("Delete action", () => {
    it("renders when the user is a global admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isGlobalAdmin: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
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
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Delete")).toBeInTheDocument();
    });

    it("renders when the user is a team admin", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isTeamAdmin: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Delete")).toBeInTheDocument();
    });

    it("renders when the user is a team maintainer", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isTeamMaintainer: true,
          },
        },
      });

      const { user } = render(
        <HostActionsDropdown
          onSelect={noop}
          hostStatus="online"
          hostMdmEnrollemntStatus="On (automatic)"
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.getByText("Delete")).toBeInTheDocument();
    });
  });
});
