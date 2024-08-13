import React from "react";
import { noop } from "lodash";
import { screen, within } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import {
  createMockAppStoreApp,
  createMockSoftwarePackage,
  createMockSoftwareTitleDetails,
} from "__mocks__/softwareMock";

import SoftwarePackageCard from "./SoftwarePackageCard";
import { getPackageCardInfo } from "../helpers";

const softwareTitleAsPackageMock = createMockSoftwareTitleDetails({
  software_package: createMockSoftwarePackage({
    self_service: true,
    labels_include_any: [{ name: "Mock label", id: 20 }],
  }),
});

const softwareTitleAsAppStoreAppMock = createMockSoftwareTitleDetails({
  app_store_app: createMockAppStoreApp(),
});

describe("Software package section", () => {
  describe("shows package data", () => {
    it("show name, version, self service, and number of labels targeted", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      render(
        <SoftwarePackageCard
          title={softwareTitleAsPackageMock}
          softwareId={123}
          teamId={1}
          onDelete={noop}
        />
      );
      expect(
        screen.getByText(new RegExp(softwareTitleAsPackageMock.name, "i"))
      ).toBeInTheDocument();
      expect(
        screen.getByText(
          new RegExp(
            `Version\\s+${softwareTitleAsPackageMock.software_package?.version}\\s+•`,
            "i"
          )
        )
      ).toBeInTheDocument();
      expect(screen.getByText("Self-service")).toBeInTheDocument();
      expect(screen.getByText("1 label")).toBeInTheDocument();
    });
  });
  describe("shows correct host status counts", () => {
    it("show all 5 status counts for software packages", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      render(
        <SoftwarePackageCard
          title={softwareTitleAsPackageMock}
          softwareId={123}
          teamId={1}
          onDelete={noop}
        />
      );

      const verifiedElement = screen.getByText("Verified");
      let statusDiv =
        verifiedElement.parentElement?.parentElement?.parentElement;
      if (!statusDiv) {
        throw new Error("Verified div not found");
      }
      expect(within(statusDiv).getByText("1 host")).toBeInTheDocument();

      const verifyingElement = screen.getByText("Verifying");
      statusDiv = verifyingElement.parentElement?.parentElement?.parentElement;
      if (!statusDiv) {
        throw new Error("Verifying div not found");
      }
      expect(within(statusDiv).getByText("4 hosts")).toBeInTheDocument();

      const pendingElement = screen.getByText("Pending");
      statusDiv = pendingElement.parentElement?.parentElement?.parentElement;
      if (!statusDiv) {
        throw new Error("Pending div not found");
      }
      expect(within(statusDiv).getByText("2 hosts")).toBeInTheDocument();

      const blockedElement = screen.getByText("Blocked");
      statusDiv = blockedElement.parentElement?.parentElement?.parentElement;
      if (!statusDiv) {
        throw new Error("Blocked div not found");
      }
      expect(within(statusDiv).getByText("0 hosts")).toBeInTheDocument();

      const failedElement = screen.getByText("Failed");
      statusDiv = failedElement.parentElement?.parentElement?.parentElement;
      if (!statusDiv) {
        throw new Error("Failed div not found");
      }
      expect(within(statusDiv).getByText("3 hosts")).toBeInTheDocument();
    });
    it("omit blocked status only if app store app", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      render(
        <SoftwarePackageCard
          title={softwareTitleAsAppStoreAppMock}
          softwareId={123}
          teamId={1}
          onDelete={noop}
        />
      );
      expect(screen.queryByText("Verified")).toBeInTheDocument();
      expect(screen.queryByText("Verifying")).toBeInTheDocument();
      expect(screen.queryByText("Pending")).toBeInTheDocument();
      expect(screen.queryByText("Failed")).toBeInTheDocument();
      expect(screen.queryByText("Blocked")).not.toBeInTheDocument();
    });
  });
  describe("shows correct actions in action dropdown", () => {
    it("show download, options, and delete actions for software packages", async () => {
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
        <SoftwarePackageCard
          title={softwareTitleAsPackageMock}
          softwareId={123}
          teamId={1}
          onDelete={noop}
        />
      );

      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Delete")).toBeInTheDocument();
      expect(screen.queryByText("Show options")).toBeInTheDocument();
      expect(screen.queryByText("Download")).toBeInTheDocument();
    });
    it("only shows delete action for app store app", async () => {
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
        <SoftwarePackageCard
          title={softwareTitleAsAppStoreAppMock}
          softwareId={123}
          teamId={1}
          onDelete={noop}
        />
      );
      await user.click(screen.getByText("Actions"));

      expect(screen.queryByText("Delete")).toBeInTheDocument();
      expect(screen.queryByText("Show options")).not.toBeInTheDocument();
      expect(screen.queryByText("Download")).not.toBeInTheDocument();
    });
  });
});
