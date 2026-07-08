import React from "react";

import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockSoftwarePackageIos,
  createMockAppStoreApp,
  createMockAppStoreAppAndroid,
  createMockAppStoreAppIos,
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
        onClickVersions={jest.fn()}
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

    it("collapses to a single pencil-icon Edit (appearance) button for standard custom software packages (#48400)", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage(),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
          canActivateMultiplePackages
        />
      );

      // Custom non-FMA macOS/Linux/Windows titles drop the Actions dropdown.
      // Per-installer Edit moves to the Library accordion row; the page-level
      // CTA collapses to a single pencil-icon "Edit" that opens the Edit
      // Appearance modal directly.
      expect(screen.queryByText("Actions")).not.toBeInTheDocument();
      const editButton = screen.getByRole("button", { name: /Edit/ });
      expect(editButton).toBeInTheDocument();
    });

    it("displays Edit appearance, Edit software, Edit configuration, and Schedule auto updates for iOS/iPadOS apps", async () => {
      const { user } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "ios_apps",
            app_store_app: createMockAppStoreAppIos(),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit software");
      expect(options).toContain("Edit configuration");
      expect(options).toContain("Schedule auto updates");
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
          onClickVersions={jest.fn()}
        />
      );

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit configuration");
      expect(options).not.toContain("Edit software");
      expect(options).not.toContain("Schedule auto updates");
    });

    it("displays Edit appearance (but not Edit configuration nor Edit software) for Android web apps", async () => {
      const { user } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "android_apps",
            app_store_app: createMockAppStoreAppAndroid({
              app_store_id: "com.google.enterprise.webapp.myapp",
            }),
            software_package: null,
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).not.toContain("Edit software");
      expect(options).not.toContain("Edit configuration");
      expect(options).not.toContain("Schedule auto updates");
    });

    it("displays Edit configuration for iOS/iPadOS in-house (.ipa) apps", async () => {
      const { user } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "ios_apps",
            software_package: createMockSoftwarePackageIos(),
            app_store_app: null,
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit software");
      expect(options).toContain("Edit configuration");
    });

    it("collapses macOS .pkg titles to the single-Edit button (no Edit configuration, no Actions dropdown) (#48400)", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "apps",
            software_package: createMockSoftwarePackage(),
            app_store_app: null,
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
          canActivateMultiplePackages
        />
      );

      // macOS in-house .pkg is a custom non-FMA, non-iOS title — collapses
      // to the pencil Edit button. Edit configuration never applied here
      // and the dropdown is gone entirely.
      expect(screen.queryByText("Actions")).not.toBeInTheDocument();
      expect(screen.queryByText("Edit configuration")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: /Edit/ })).toBeInTheDocument();
    });

    it("does not display Edit configuration for macOS VPP apps", async () => {
      const { user } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "apps",
            // Explicit null — `createMockSoftwareTitle` defaults to a custom
            // package, which would otherwise hide the dropdown under #48400.
            software_package: null,
            app_store_app: createMockAppStoreApp({ platform: "darwin" }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      const options = await getDropdownOptions(user);

      expect(options).toContain("Edit appearance");
      expect(options).toContain("Edit software");
      expect(options).not.toContain("Edit configuration");
    });

    it("adds Versions option after Patch for a Premium Fleet-maintained app", async () => {
      const { user } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      const options = await getDropdownOptions(user);
      const patchIdx = options.indexOf("Patch");
      const versionsIdx = options.indexOf("Versions");

      expect(patchIdx).toBeGreaterThan(-1);
      expect(versionsIdx).toBeGreaterThan(-1);
      expect(versionsIdx).toBe(patchIdx + 1);
    });

    it("hides Versions option on Fleet Free even for a Fleet-maintained app", async () => {
      const freeRender = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: false,
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

      const { user } = freeRender(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      const options = await getDropdownOptions(user);
      expect(options).not.toContain("Versions");
    });

    it("hides the Actions dropdown (and therefore Versions) for non-FMA custom installers (#48400)", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage(),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
          canActivateMultiplePackages
        />
      );

      // Non-FMA custom titles no longer use the Actions dropdown at all,
      // so Versions is implicitly hidden — the whole dropdown is gone.
      // (The `<dt>Versions</dt>` stat row in the description list remains;
      // we're asserting against the dropdown item, not that stat.)
      expect(screen.queryByText("Actions")).not.toBeInTheDocument();
      expect(
        screen.queryByRole("menuitem", { name: /Versions/ })
      ).not.toBeInTheDocument();
    });

    it("still renders the Versions option when GitOps mode is on", async () => {
      const gitopsRender = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            config: {
              gitops: {
                gitops_mode_enabled: true,
                repository_url: "https://example.com/repo",
              },
            },
          },
        },
      });

      const { user } = gitopsRender(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      await user.click(screen.getByText("Actions"));
      const options = screen
        .getAllByTestId("dropdown-option")
        .map((opt) => opt.textContent);
      expect(options).toContain("Versions");
    });

    it("hides Versions option for observers without manage permission", async () => {
      const observerRender = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalObserver: true,
            config: {
              gitops: {
                gitops_mode_enabled: false,
                repository_url: "",
              },
            },
          },
        },
      });

      const { user } = observerRender(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      // Observers don't get an Actions dropdown at all when canManageSoftware
      // is false — confirming Versions is unreachable from this role.
      expect(screen.queryByText("Actions")).not.toBeInTheDocument();
      expect(user).toBeDefined();
    });
  });

  describe("Header pills", () => {
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

    it("renders the Fleet-maintained pill for an FMA", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("Fleet-maintained")).toBeInTheDocument();
    });

    it("renders the App Store (VPP) pill for an Apple VPP app (macOS)", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            source: "apps",
            app_store_app: createMockAppStoreApp({ platform: "darwin" }),
            software_package: null,
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("App Store (VPP)")).toBeInTheDocument();
    });

    it("renders the Play Store pill for an Android Play Store app", () => {
      render(
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
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("Play Store")).toBeInTheDocument();
    });

    it("renders the Custom package pill for a non-FMA software package", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage(),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("Custom package")).toBeInTheDocument();
    });

    it("renders the Self-service pill when self_service is true", () => {
      // FMA mock — custom packages hide the title-level Self-service /
      // Auto install / Patch chips under #48400 (per-row icons take over).
      // FMA titles are single-package and keep the chips.
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              self_service: true,
              fleet_maintained_app_id: 7,
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("Self-service")).toBeInTheDocument();
    });

    it("hides the Self-service / Auto install / Patch chips for custom packages (#48400)", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              self_service: true,
              automatic_install_policies: [
                { id: 1, name: "Policy A", type: "dynamic" },
              ],
              patch_policy: { id: 42, name: "Outdated Postman" },
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
          canActivateMultiplePackages
        />
      );

      // Per-row icons on the Library accordion replace these for multi-
      // package custom titles; the title-level chips would be misleading.
      expect(screen.queryByText("Self-service")).not.toBeInTheDocument();
      expect(screen.queryByText("Auto install")).not.toBeInTheDocument();
      expect(screen.queryByText("Patch policy")).not.toBeInTheDocument();
    });

    it("does not render the Self-service pill when self_service is false", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              self_service: false,
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.queryByText("Self-service")).not.toBeInTheDocument();
    });

    it("renders the Auto install pill when the title has linked auto-install policies", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
              automatic_install_policies: [
                { id: 1, name: "Policy A", type: "dynamic" },
                { id: 2, name: "Policy B", type: "dynamic" },
              ],
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("Auto install")).toBeInTheDocument();
    });

    it("renders the Patch policy pill when only a patch_policy is linked", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
              patch_policy: { id: 42, name: "Outdated Postman" },
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("Patch policy")).toBeInTheDocument();
      expect(screen.queryByText("Auto install")).not.toBeInTheDocument();
    });

    it("renders the Auto install pill when both auto-install and patch policies are linked", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
              automatic_install_policies: [
                { id: 1, name: "Policy A", type: "dynamic" },
              ],
              patch_policy: { id: 42, name: "Outdated Postman" },
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.getByText("Auto install")).toBeInTheDocument();
      expect(screen.queryByText("Patch policy")).not.toBeInTheDocument();
    });

    it("does not render the Auto install pill when no policies are linked", () => {
      render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage(),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(screen.queryByText("Auto install")).not.toBeInTheDocument();
    });

    it("navigates directly to the policy when only one policy is linked", async () => {
      const pushedRouter = createMockRouter();
      const { user } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
              automatic_install_policies: [
                { id: 99, name: "Solo policy", type: "dynamic" },
              ],
            }),
          })}
          softwareId={1}
          teamId={3}
          router={pushedRouter}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      await user.click(screen.getByText("Auto install"));

      expect(pushedRouter.push).toHaveBeenCalledWith(
        expect.stringMatching(/\/policies\/99.*fleet_id=3/)
      );
    });

    it("opens the Policies modal when more than one policy is linked", async () => {
      const { user } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              fleet_maintained_app_id: 7,
              automatic_install_policies: [
                { id: 1, name: "Policy A", type: "dynamic" },
                { id: 2, name: "Policy B", type: "dynamic" },
              ],
            }),
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      await user.click(screen.getByText("Auto install"));

      expect(screen.getAllByText("Policy A").length).toBeGreaterThan(0);
      expect(screen.getAllByText("Policy B").length).toBeGreaterThan(0);
    });

    it("does not render the pills row when there is no installer", () => {
      const { container } = render(
        <SoftwareSummaryCard
          softwareTitle={createMockSoftwareTitle({
            software_package: null,
            app_store_app: null,
          })}
          softwareId={1}
          teamId={1}
          router={router}
          refetchSoftwareTitle={jest.fn()}
          onClickVersions={jest.fn()}
        />
      );

      expect(
        container.querySelector(".software-details-summary__header-pills")
      ).toBeNull();
    });
  });
});
