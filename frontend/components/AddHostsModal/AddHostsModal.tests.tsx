import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import AddHostsModal from "./AddHostsModal";

const ENROLL_SECRET = "abcdefg12345678";

// Joins every path's geometry rather than picking one out, so the comparison
// does not depend on how many paths qrcode.react emits or in what order.
const getQrCodeData = () => {
  const paths = screen.getByTestId("enroll-qr-code").querySelectorAll("path");
  expect(paths.length).toBeGreaterThan(0);
  return Array.from(paths)
    .map((path) => path.getAttribute("d"))
    .join("|");
};

describe("AddHostsModal", () => {
  it("renders loading state", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    render(
      <AddHostsModal isAnyTeamSelected={false} isLoading onCancel={noop} />
    );
    // Spinner has a built-in anti-flash delay, so wait for it to appear.
    const loadingSpinner = await screen.findByTestId("spinner");
    expect(loadingSpinner).toBeVisible();
  });

  it("renders platform tabs", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "macOS" }));
    const macOSText = screen.getByText(/--type=pkg/i);
    expect(macOSText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    const windowsText = screen.getByText(/--type=msi/i);
    expect(windowsText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Linux" }));
    const linuxDebText = screen.getByText(/--type=deb/i);
    expect(linuxDebText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).toBeInTheDocument();
    expect(screen.queryByText(/--type=rpm/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "ChromeOS" }));
    const extensionId = screen.getByDisplayValue(
      /fleeedmmihkfkeemmipgmhhjemlljidg/i
    );
    expect(extensionId).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "iOS & iPadOS" }));
    expect(screen.queryByText(/Turn on Apple MDM/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Android" }));
    expect(screen.queryByText(/Turn on Android MDM/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Advanced" }));
    const advancedText = screen.getByText(/--type=YOUR_TYPE/i);
    expect(advancedText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByText(/Plain osquery/i));
    const downloadEnrollSecret = screen.getByText(
      /Download your enroll secret/i
    );
    expect(downloadEnrollSecret).toBeInTheDocument();
    const osquerydCommand = screen.getByDisplayValue(
      /osqueryd --flagfile=flagfile.txt --verbose/i
    );
    expect(osquerydCommand).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();
  });

  it("renders enroll url input for macOS if mac mdm is enabled", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isMacMdmEnabledAndConfigured: true,
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "macOS" }));
    expect(screen.getByLabelText("Personal (BYOD)")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Company-owned (fully-managed)")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Share this link with your end users:")
    ).toBeInTheDocument();

    // Company-owned is selected by default — URL has no byod param
    const urlInput = screen.getByDisplayValue(
      new RegExp(`/enroll\\?enroll_secret=${ENROLL_SECRET}$`)
    );
    expect(urlInput).toBeInTheDocument();

    // Switching to Personal (BYOD) appends byod=true
    await user.click(screen.getByLabelText("Personal (BYOD)"));
    const byodUrlInput = screen.getByDisplayValue(
      new RegExp(`/enroll\\?enroll_secret=${ENROLL_SECRET}&byod=true`)
    );
    expect(byodUrlInput).toBeInTheDocument();
  });

  it("renders enroll url input for ios & ipadOS if mac mdm is enabled", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isMacMdmEnabledAndConfigured: true,
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "iOS & iPadOS" }));
    expect(screen.getByText("Enrollment instructions")).toBeInTheDocument();
    expect(
      screen.getByText("Share this link with your end users:")
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Personal (BYOD)")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Company-owned (fully-managed)")
    ).toBeInTheDocument();
    expect(screen.getByText("To test, scan the QR code:")).toBeInTheDocument();
    expect(screen.getByTestId("enroll-qr-code")).toBeInTheDocument();
  });

  it("renders enroll url input for android if android mdm is enabled", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isAndroidMdmEnabledAndConfigured: true,
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "Android" }));
    expect(screen.getByText("Enrollment instructions")).toBeInTheDocument();
    expect(
      screen.getByText("Share this link with your end users:")
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Personal (BYOD)")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Company-owned (fully-managed)")
    ).toBeInTheDocument();
    expect(screen.getByText("To test, scan the QR code:")).toBeInTheDocument();
    expect(screen.getByTestId("enroll-qr-code")).toBeInTheDocument();
  });

  it("updates the android qr code when the enrollment type changes", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isAndroidMdmEnabledAndConfigured: true,
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "Android" }));

    // Personal (BYOD) is selected by default — URL has no fully_managed param.
    expect(
      screen.getByDisplayValue(
        new RegExp(`/enroll\\?enroll_secret=${ENROLL_SECRET}$`)
      )
    ).toBeInTheDocument();
    const workProfileQrData = getQrCodeData();
    expect(workProfileQrData).toBeTruthy();

    await user.click(screen.getByLabelText("Company-owned (fully-managed)"));

    expect(
      screen.getByDisplayValue(
        new RegExp(
          `/enroll\\?enroll_secret=${ENROLL_SECRET}&fully_managed=true$`
        )
      )
    ).toBeInTheDocument();
    expect(getQrCodeData()).not.toEqual(workProfileQrData);
  });

  it("updates the ios & ipadOS qr code when the enrollment type changes", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isMacMdmEnabledAndConfigured: true,
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "iOS & iPadOS" }));

    // Personal (BYOD) is selected by default — URL carries byod=true.
    expect(
      screen.getByDisplayValue(
        new RegExp(`/enroll\\?enroll_secret=${ENROLL_SECRET}&byod=true$`)
      )
    ).toBeInTheDocument();
    const personalQrData = getQrCodeData();
    expect(personalQrData).toBeTruthy();

    await user.click(screen.getByLabelText("Company-owned (fully-managed)"));

    expect(
      screen.getByDisplayValue(
        new RegExp(`/enroll\\?enroll_secret=${ENROLL_SECRET}$`)
      )
    ).toBeInTheDocument();
    expect(getQrCodeData()).not.toEqual(personalQrData);
  });

  it("renders no qr code while the mdm gate is showing", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "iOS & iPadOS" }));
    expect(screen.getByText(/Turn on Apple MDM/i)).toBeInTheDocument();
    expect(screen.queryByTestId("enroll-qr-code")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Android" }));
    expect(screen.getByText(/Turn on Android MDM/i)).toBeInTheDocument();
    expect(screen.queryByTestId("enroll-qr-code")).not.toBeInTheDocument();
  });

  it("renders installer with secret", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    const regex = new RegExp(`${ENROLL_SECRET}`);
    const text = screen.getByDisplayValue(regex);

    expect(text).toBeInTheDocument();
  });
  it("renders no enroll secret cta", async () => {
    const onCancel = jest.fn();
    const openEnrollSecretModal = jest.fn();

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected={false}
        isLoading={false}
        onCancel={onCancel}
        openEnrollSecretModal={openEnrollSecretModal}
      />
    );

    expect(screen.getByText("Something's gone wrong.")).toBeInTheDocument();
    expect(
      screen.getByText(/you have no enroll secrets\./i)
    ).toBeInTheDocument();

    const cta = screen.getByText(/manage enroll secrets/i);
    expect(cta).toBeInTheDocument();

    await user.click(cta);

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(openEnrollSecretModal).toHaveBeenCalledTimes(1);
  });

  it("excludes `--enable-scripts` flag if `config.server_settings.scripts-disabled` is `true`", async () => {
    const mockConfig = createMockConfig();
    mockConfig.server_settings.scripts_disabled = true;

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: mockConfig,
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "macOS" }));
    const macOSText = screen.getByText(/--type=pkg/i);
    expect(macOSText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    const windowsText = screen.getByText(/--type=msi/i);
    expect(windowsText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Linux" }));
    const linuxRPMText = screen.getByText(/--type=rpm/i);
    expect(linuxRPMText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "ChromeOS" }));
    const extensionId = screen.getByDisplayValue(
      /fleeedmmihkfkeemmipgmhhjemlljidg/i
    );
    expect(extensionId).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Advanced" }));
    const advancedText = screen.getByText(/--type=YOUR_TYPE/i);
    expect(advancedText).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByText(/Plain osquery/i));
    const downloadEnrollSecret = screen.getByText(
      /Download your enroll secret/i
    );
    expect(downloadEnrollSecret).toBeInTheDocument();
    const osquerydCommand = screen.getByDisplayValue(
      /osqueryd --flagfile=flagfile.txt --verbose/i
    );
    expect(osquerydCommand).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();
  });
});
