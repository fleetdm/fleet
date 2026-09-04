import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";

import PATHS from "router/paths";
import { IMdmConfig } from "interfaces/config";
import { DiskEncryptionSettingsPlatform } from "interfaces/platform";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";
import createMockTeam from "__mocks__/teamMock";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import {
  createGetDiskEncryptionSummaryHandler,
  createUpdateDiskEncryptionHandler,
} from "test/handlers/disk-encryption-handlers";

import DiskEncryption from "./DiskEncryption";

interface ITeamDiskEncryptionMdm {
  macOSEnabled?: boolean;
  macOSEscrowEnabled?: boolean;
  windowsEnabled?: boolean;
  windowsPINRequired?: boolean;
  linuxEscrowEnabled?: boolean;
}

const createGetTeamHandler = ({
  macOSEnabled = false,
  macOSEscrowEnabled = false,
  windowsEnabled = false,
  windowsPINRequired = false,
  linuxEscrowEnabled = false,
}: ITeamDiskEncryptionMdm = {}) => {
  return http.get(baseUrl("/fleets/:id"), () => {
    return HttpResponse.json({
      fleet: {
        ...createMockTeam(),
        mdm: createMockMdmConfig({
          apple_settings: {
            configuration_profiles: null,
            enable_disk_encryption: macOSEnabled,
            enable_escrow_disk_encryption_key: macOSEscrowEnabled,
          },
          windows_settings: {
            enable_disk_encryption: windowsEnabled,
            require_bitlocker_pin: windowsPINRequired,
          },
          linux_settings: {
            enable_escrow_disk_encryption_key: linuxEscrowEnabled,
          },
        }),
      },
    });
  });
};

interface IRenderOptions {
  teamId?: number;
  /** Omit to render the default (macOS) tab; pass `undefined` explicitly to
   * simulate a URL without a platform segment. */
  urlPlatformParam?: string;
  gitOpsModeEnabled?: boolean;
  isPremiumTier?: boolean;
  mdm?: Partial<IMdmConfig>;
}

const renderDiskEncryption = (options: IRenderOptions = {}) => {
  const {
    teamId = 1,
    gitOpsModeEnabled = false,
    isPremiumTier = true,
    mdm,
  } = options;
  const urlPlatformParam =
    "urlPlatformParam" in options ? options.urlPlatformParam : "macos";
  const router = createMockRouter();

  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier,
        config: createMockConfig({
          gitops: {
            gitops_mode_enabled: gitOpsModeEnabled,
            repository_url: "https://example.com/repo",
            exceptions: { labels: false, software: false, secrets: false },
          },
          mdm: createMockMdmConfig(mdm),
        }),
        setConfig: jest.fn(),
      },
    },
  });

  return {
    ...render(
      <DiskEncryption
        currentTeamId={teamId}
        onMutation={jest.fn()}
        router={router}
        urlPlatformParam={urlPlatformParam}
      />
    ),
    router,
  };
};

const platformTabPath = (
  platform: DiskEncryptionSettingsPlatform,
  teamId: number
) =>
  getPathWithQueryParams(PATHS.CONTROLS_DISK_ENCRYPTION_PLATFORM(platform), {
    fleet_id: teamId,
  });

const findEnforceCheckbox = () =>
  screen.findByRole("checkbox", { name: "Enable disk encryption" });
const findEscrowCheckbox = () =>
  screen.findByRole("checkbox", { name: "Escrow recovery key with Fleet" });
const findPINCheckbox = () =>
  screen.findByRole("checkbox", { name: "Require BitLocker PIN" });

describe("DiskEncryption", () => {
  it("renders the premium feature message for free tier", () => {
    renderDiskEncryption({ teamId: 0, isPremiumTier: false });

    expect(screen.getByText(/Fleet Premium/)).toBeInTheDocument();
    expect(screen.queryByRole("tab")).toBeNull();
  });

  it.each([
    [
      "macos",
      "macOS",
      ["Enable disk encryption", "Escrow recovery key with Fleet"],
      ["Require BitLocker PIN"],
    ],
    [
      "windows",
      "Windows",
      ["Enable disk encryption", "Require BitLocker PIN"],
      ["Escrow recovery key with Fleet"],
    ],
    [
      "linux",
      "Linux",
      ["Escrow recovery key with Fleet"],
      ["Enable disk encryption", "Require BitLocker PIN"],
    ],
  ] as const)(
    "selects the %s tab from the URL and renders its checkboxes",
    async (platform, tabName, present, absent) => {
      mockServer.use(createGetTeamHandler());
      mockServer.use(createGetDiskEncryptionSummaryHandler());
      const { router } = renderDiskEncryption({ urlPlatformParam: platform });

      expect(await screen.findByRole("tab", { name: tabName })).toHaveAttribute(
        "aria-selected",
        "true"
      );
      present.forEach((name) => {
        expect(screen.getByRole("checkbox", { name })).toBeInTheDocument();
      });
      absent.forEach((name) => {
        expect(screen.queryByRole("checkbox", { name })).toBeNull();
      });
      expect(router.replace).not.toHaveBeenCalled();
    }
  );

  it("navigates to the platform route when a tab is clicked", async () => {
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    const { user, router } = renderDiskEncryption();

    await user.click(await screen.findByRole("tab", { name: "Windows" }));
    expect(router.push).toHaveBeenCalledWith(platformTabPath("windows", 1));

    await user.click(screen.getByRole("tab", { name: "Linux" }));
    expect(router.push).toHaveBeenCalledWith(platformTabPath("linux", 1));
  });

  it("redirects to the macOS tab when the URL platform is missing or invalid", async () => {
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());

    const missing = renderDiskEncryption({ urlPlatformParam: undefined });
    await waitFor(() => {
      expect(missing.router.replace).toHaveBeenCalledWith(
        platformTabPath("macos", 1)
      );
    });
    missing.unmount();

    const invalid = renderDiskEncryption({
      teamId: 0,
      urlPlatformParam: "ios",
    });
    await waitFor(() => {
      expect(invalid.router.replace).toHaveBeenCalledWith(
        platformTabPath("macos", 0)
      );
    });
  });

  it("shows an error state when the fleet's settings fail to load", async () => {
    mockServer.use(
      http.get(baseUrl("/fleets/:id"), () =>
        HttpResponse.json({ message: "error" }, { status: 500 })
      )
    );
    renderDiskEncryption();

    expect(
      await screen.findByText(/Something.s gone wrong/)
    ).toBeInTheDocument();
    expect(screen.queryByRole("tab")).toBeNull();
  });

  it("disables the form and marks Save as loading while a save is in flight", async () => {
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    mockServer.use(
      http.post(baseUrl("/disk_encryption"), async () => {
        await delay(150);
        return HttpResponse.json({});
      })
    );
    const { user } = renderDiskEncryption();

    const enforceCheckbox = await findEnforceCheckbox();
    const saveButton = screen.getByRole("button", { name: "Save" });
    await user.click(saveButton);

    expect(saveButton).toBeDisabled();
    expect(enforceCheckbox).toHaveAttribute("aria-disabled", "true");

    await waitFor(() => expect(saveButton).toBeEnabled());
    expect(enforceCheckbox).toHaveAttribute("aria-disabled", "false");
  });

  it("toggles the macOS checkboxes independently and saves only macOS settings", async () => {
    let requestBody: Record<string, unknown> | undefined;
    mockServer.use(createGetTeamHandler({ macOSEnabled: true }));
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    mockServer.use(
      createUpdateDiskEncryptionHandler((body) => {
        requestBody = body;
      })
    );
    const { user } = renderDiskEncryption();

    const enforceCheckbox = await findEnforceCheckbox();
    const escrowCheckbox = await findEscrowCheckbox();
    expect(enforceCheckbox).toBeChecked();
    expect(escrowCheckbox).not.toBeChecked();

    // toggling one checkbox does not affect the other
    await user.click(escrowCheckbox);
    expect(enforceCheckbox).toBeChecked();
    expect(escrowCheckbox).toBeChecked();

    await user.click(enforceCheckbox);
    expect(enforceCheckbox).not.toBeChecked();
    expect(escrowCheckbox).toBeChecked();

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(requestBody).toEqual({
        macos_settings: {
          enable_disk_encryption: false,
          enable_escrow_disk_encryption_key: true,
        },
        fleet_id: 1,
      });
    });
  });

  it("saves only Windows settings from the Windows tab", async () => {
    let requestBody: Record<string, unknown> | undefined;
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    mockServer.use(
      createUpdateDiskEncryptionHandler((body) => {
        requestBody = body;
      })
    );
    const { user } = renderDiskEncryption({ urlPlatformParam: "windows" });

    await user.click(await findEnforceCheckbox());
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(requestBody).toEqual({
        windows_settings: {
          enable_disk_encryption: true,
          require_bitlocker_pin: false,
        },
        fleet_id: 1,
      });
    });
  });

  it("saves only Linux settings from the Linux tab", async () => {
    let requestBody: Record<string, unknown> | undefined;
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    mockServer.use(
      createUpdateDiskEncryptionHandler((body) => {
        requestBody = body;
      })
    );
    const { user } = renderDiskEncryption({ urlPlatformParam: "linux" });

    await user.click(await findEscrowCheckbox());
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(requestBody).toEqual({
        linux_settings: { enable_escrow_disk_encryption_key: true },
        fleet_id: 1,
      });
    });
  });

  it("omits fleet_id when saving for hosts with no fleet", async () => {
    let requestBody: Record<string, unknown> | undefined;
    mockServer.use(createGetConfigHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    mockServer.use(
      createUpdateDiskEncryptionHandler((body) => {
        requestBody = body;
      })
    );
    const { user } = renderDiskEncryption({ teamId: 0 });

    await user.click(await findEnforceCheckbox());
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(requestBody).toEqual({
        macos_settings: {
          enable_disk_encryption: true,
          enable_escrow_disk_encryption_key: false,
        },
      });
    });
  });

  it("ignores a saved BitLocker PIN requirement when Windows disk encryption is off", async () => {
    let requestBody: Record<string, unknown> | undefined;
    mockServer.use(
      createGetTeamHandler({ windowsEnabled: false, windowsPINRequired: true })
    );
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    mockServer.use(
      createUpdateDiskEncryptionHandler((body) => {
        requestBody = body;
      })
    );
    const { user } = renderDiskEncryption({ urlPlatformParam: "windows" });

    const pinCheckbox = await findPINCheckbox();
    expect(pinCheckbox).not.toBeChecked();
    expect(pinCheckbox).toHaveAttribute("aria-disabled", "true");

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(requestBody).toEqual({
        windows_settings: {
          enable_disk_encryption: false,
          require_bitlocker_pin: false,
        },
        fleet_id: 1,
      });
    });
  });

  it("disables and unchecks the BitLocker PIN checkbox when Windows disk encryption is turned off", async () => {
    let requestBody: Record<string, unknown> | undefined;
    mockServer.use(
      createGetTeamHandler({ windowsEnabled: true, windowsPINRequired: true })
    );
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    mockServer.use(
      createUpdateDiskEncryptionHandler((body) => {
        requestBody = body;
      })
    );
    const { user } = renderDiskEncryption({ urlPlatformParam: "windows" });

    const enforceCheckbox = await findEnforceCheckbox();
    const pinCheckbox = await findPINCheckbox();
    expect(enforceCheckbox).toBeChecked();
    expect(pinCheckbox).toBeChecked();
    expect(pinCheckbox).toHaveAttribute("aria-disabled", "false");

    await user.click(enforceCheckbox);
    expect(pinCheckbox).not.toBeChecked();
    expect(pinCheckbox).toHaveAttribute("aria-disabled", "true");

    // clicking the disabled PIN checkbox does nothing
    await user.click(pinCheckbox);
    expect(pinCheckbox).not.toBeChecked();

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(requestBody).toEqual({
        windows_settings: {
          enable_disk_encryption: false,
          require_bitlocker_pin: false,
        },
        fleet_id: 1,
      });
    });
  });

  it.each(["macos", "windows", "linux"] as const)(
    "disables all checkboxes and Save on the %s tab in GitOps mode",
    async (platform) => {
      mockServer.use(createGetTeamHandler());
      mockServer.use(createGetDiskEncryptionSummaryHandler());
      renderDiskEncryption({
        urlPlatformParam: platform,
        gitOpsModeEnabled: true,
      });

      const checkboxes = await screen.findAllByRole("checkbox");
      checkboxes.forEach((checkbox) => {
        expect(checkbox).toHaveAttribute("aria-disabled", "true");
      });
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    }
  );

  it.each([
    [
      "macos",
      { enabled_and_configured: false },
      /You must turn on Apple MDM/,
      `${LEARN_MORE_ABOUT_BASE_LINK}/turn-on-apple-mdm`,
    ],
    [
      "windows",
      { windows_enabled_and_configured: false },
      /You must turn on Windows MDM/,
      `${LEARN_MORE_ABOUT_BASE_LINK}/setup-windows-mdm`,
    ],
  ] as const)(
    "shows an empty state instead of the form on the %s tab when its MDM is turned off",
    async (platform, mdm, infoText, learnMoreUrl) => {
      mockServer.use(createGetTeamHandler());
      mockServer.use(createGetDiskEncryptionSummaryHandler());
      renderDiskEncryption({
        urlPlatformParam: platform,
        mdm,
      });

      expect(
        await screen.findByText("Turn on MDM to enforce disk encryption")
      ).toBeInTheDocument();
      expect(screen.getByText(infoText)).toBeInTheDocument();
      const link = screen.getByRole("link", { name: "Learn more" });
      expect(link).toHaveAttribute("href", learnMoreUrl);
      expect(link).toHaveAttribute("target", "_blank");

      expect(screen.queryByRole("checkbox")).toBeNull();
      expect(screen.queryByRole("button", { name: "Save" })).toBeNull();
    }
  );

  it("keeps the other tabs enabled when one platform's MDM is turned off", async () => {
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());

    const windows = renderDiskEncryption({
      urlPlatformParam: "windows",
      mdm: { enabled_and_configured: false },
    });
    expect(await findEnforceCheckbox()).toHaveAttribute(
      "aria-disabled",
      "false"
    );
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    windows.unmount();

    renderDiskEncryption({
      urlPlatformParam: "linux",
      mdm: {
        enabled_and_configured: false,
        windows_enabled_and_configured: false,
      },
    });
    expect(await findEscrowCheckbox()).toHaveAttribute(
      "aria-disabled",
      "false"
    );
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("renders the status table only for platforms with a setting enabled", async () => {
    mockServer.use(createGetTeamHandler({ windowsEnabled: true }));
    mockServer.use(createGetDiskEncryptionSummaryHandler());

    // macOS tab has no settings enabled, so no status table
    const macos = renderDiskEncryption({ urlPlatformParam: "macos" });
    expect(await findEnforceCheckbox()).toBeInTheDocument();
    expect(screen.queryByText("Verified")).toBeNull();
    macos.unmount();

    renderDiskEncryption({ urlPlatformParam: "windows" });
    expect(await screen.findByText("Verified")).toBeInTheDocument();
  });
});
