import React from "react";

import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockAppStoreApp,
  createMockAppStoreAppAndroid,
} from "__mocks__/softwareMock";

import { render as defaultRender, screen } from "@testing-library/react";
import { UserEvent } from "@testing-library/user-event";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareSummaryCard from "./SoftwareSummaryCard";

const router = createMockRouter();

jest.mock("../../components/icons/SoftwareIcon", () => {
  return {
    __esModule: true,
    default: () => {
      return <div />;
    },
  };
});

describe("Software Summary Card", () => {
  // beforeAll(() => {});
  it("Shows the correct basic info about a software title", async () => {
    const softwareTitle = createMockSoftwareTitle();
    defaultRender(
      <SoftwareSummaryCard
        softwareTitle={softwareTitle}
        softwareId={1}
        router={router}
        refetchSoftwareTitle={jest.fn()}
        onToggleViewYaml={jest.fn()}
      />
    );
    // Get the text with aria label "software display name"
    const displayNameElement = screen.getByLabelText("software display name");
    expect(displayNameElement).toHaveTextContent(softwareTitle.name);
    // Check for type "Applicaiton (macOS)"
    expect(screen.getByText("Application (macOS)")).toBeInTheDocument();
  });

  describe("Actions dropdown", () => {
    let user: UserEvent;

    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          config: {
            gitops: {
              gitops_mode_enabled: false,
              repository_url: "",
            },
          },
        },
      },
    });

    /**
     * Shared helper function to open the actions dropdown and retrieve all visible options
     * @returns Array of action option text labels
     */
    const getDropdownOptions = async (): Promise<string[]> => {
      const actionsButton = screen.getByText("Actions");
      expect(actionsButton).toBeInTheDocument();

      await user.click(actionsButton);

      // Get all options from the dropdown menu
      const options = screen.getAllByTestId("dropdown-option");
      return options.map((option) => option.textContent || "");
    };

    it("displays Edit appearance and Edit software options for standard software packages", async () => {
      const result = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage(),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onToggleViewYaml={jest.fn()}
        />
      );

      user = result.user;
      const options = await getDropdownOptions();

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit software");
      expect(options).not.toContain("Edit configuration");
      expect(options).not.toContain("Schedule auto updates");
    });

    it("displays Edit appearance, Edit software, and Schedule auto updates for iOS/iPadOS apps", async () => {
      const result = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "ios_apps",
            app_store_app: createMockAppStoreApp(),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onToggleViewYaml={jest.fn()}
        />
      );

      user = result.user;

      const options = await getDropdownOptions();

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit software");
      expect(options).toContain("Schedule auto updates");
      expect(options).not.toContain("Edit configuration");
    });

    it("displays Edit appearance and Edit configuration (but not Edit software) for Android apps", async () => {
      const result = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "android_apps",
            app_store_app: createMockAppStoreAppAndroid(),
            software_package: null,
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onToggleViewYaml={jest.fn()}
        />
      );

      user = result.user;

      const options = await getDropdownOptions();

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit configuration");
      expect(options).not.toContain("Edit software");
      expect(options).not.toContain("Schedule auto updates");
    });
  });
});
