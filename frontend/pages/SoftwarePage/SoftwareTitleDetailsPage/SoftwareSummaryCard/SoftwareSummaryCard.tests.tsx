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

// Mock the SoftwareIcon component since it makes API calls.
// We'll just check that it's called with the correct URL.
const mockSoftwareIcon = jest.fn();
jest.mock("../../components/icons/SoftwareIcon", () => {
  return {
    __esModule: true,
    default: ({ url }: { url: string }) => {
      mockSoftwareIcon({ url });
      return <div />;
    },
  };
});

describe("Software Summary Card", () => {
  beforeEach(() => {
    mockSoftwareIcon.mockClear();
  });
  it("Shows the correct basic info about a software title", async () => {
    const softwareTitle = createMockSoftwareTitle({
      icon_url: "https://example.com/icon.png",
    });
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
    // Check for type "Application (macOS)"
    expect(screen.getByText("Application (macOS)")).toBeInTheDocument();
    // Check that the icon component is called with the correct URL.
    expect(mockSoftwareIcon).toHaveBeenCalledWith({
      url: "https://example.com/icon.png",
    });
  });

  describe("Actions dropdown", () => {
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

    // Shared helper function to open the actions dropdown and retrieve all visible options.
    const getDropdownOptions = async (user: UserEvent): Promise<string[]> => {
      const actionsButton = screen.getByText("Actions");
      expect(actionsButton).toBeInTheDocument();

      await user.click(actionsButton);

      // Get all options from the dropdown menu
      const options = screen.getAllByTestId("dropdown-option");
      return options.map((option) => option.textContent || "");
    };

    it("displays Edit appearance and Edit software options for standard software packages", async () => {
      const { user } = render(
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

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit software");
      expect(options).not.toContain("Edit configuration");
      expect(options).not.toContain("Schedule auto updates");
    });

    it("displays Edit appearance, Edit software, and Schedule auto updates for iOS/iPadOS apps", async () => {
      const { user } = render(
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

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit software");
      expect(options).toContain("Schedule auto updates");
      expect(options).not.toContain("Edit configuration");
    });

    it("displays Edit appearance and Edit configuration (but not Edit software) for Android apps", async () => {
      const { user } = render(
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

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit configuration");
      expect(options).not.toContain("Edit software");
      expect(options).not.toContain("Schedule auto updates");
    });
  });
});
