import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import createMockConfig from "__mocks__/configMock";
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
        id: 1,
        name: "Team 1",
        mdm: {
          enable_disk_encryption: false,
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
        },
      },
    });
  });
};

const renderDiskEncryption = ({
  teamId = 1,
  gitOpsModeEnabled = false,
  isPremiumTier = true,
}: {
  teamId?: number;
  gitOpsModeEnabled?: boolean;
  isPremiumTier?: boolean;
} = {}) => {
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
        }),
        setConfig: jest.fn(),
      },
    },
  });

  return render(
    <DiskEncryption
      currentTeamId={teamId}
      onMutation={jest.fn()}
      router={createMockRouter()}
    />
  );
};

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

  it("switches between the macOS, Windows, and Linux tabs", async () => {
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    const { user } = renderDiskEncryption();

    // macOS tab is selected by default and has both checkboxes
    expect(await findEnforceCheckbox()).toBeInTheDocument();
    expect(await findEscrowCheckbox()).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "Require BitLocker PIN" })
    ).toBeNull();

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    expect(await findEnforceCheckbox()).toBeInTheDocument();
    expect(await findPINCheckbox()).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "Escrow recovery key with Fleet" })
    ).toBeNull();

    await user.click(screen.getByRole("tab", { name: "Linux" }));
    expect(await findEscrowCheckbox()).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "Enable disk encryption" })
    ).toBeNull();
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
    const { user } = renderDiskEncryption();

    await user.click(await screen.findByRole("tab", { name: "Windows" }));
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
    const { user } = renderDiskEncryption();

    await user.click(await screen.findByRole("tab", { name: "Linux" }));
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
    const { user } = renderDiskEncryption();

    await user.click(await screen.findByRole("tab", { name: "Windows" }));

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

  it("disables all checkboxes and Save on every tab in GitOps mode", async () => {
    mockServer.use(createGetTeamHandler());
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    const { user } = renderDiskEncryption({ gitOpsModeEnabled: true });

    const assertAllDisabled = async () => {
      const checkboxes = await screen.findAllByRole("checkbox");
      checkboxes.forEach((checkbox) => {
        expect(checkbox).toHaveAttribute("aria-disabled", "true");
      });
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    };

    await assertAllDisabled();
    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await assertAllDisabled();
    await user.click(screen.getByRole("tab", { name: "Linux" }));
    await assertAllDisabled();
  });

  it("renders the status table only for platforms with a setting enabled", async () => {
    mockServer.use(createGetTeamHandler({ windowsEnabled: true }));
    mockServer.use(createGetDiskEncryptionSummaryHandler());
    const { user } = renderDiskEncryption();

    // macOS tab has no settings enabled, so no status table
    expect(await findEnforceCheckbox()).toBeInTheDocument();
    expect(screen.queryByText("Verified")).toBeNull();

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    expect(await screen.findByText("Verified")).toBeInTheDocument();
  });
});
