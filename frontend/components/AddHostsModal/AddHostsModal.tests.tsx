import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import AddHostsModal from "./AddHostsModal";

const ENROLL_SECRET = "abcdefg12345678";

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
    const loadingSpinner = screen.getByTestId("spinner");
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
    expect(
      screen.queryByText(/CentOS, Red Hat, and Fedora Linux, use --type=rpm/i)
    ).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "ChromeOS" }));
    const extensionId = screen.getByDisplayValue(
      /fleeedmmihkfkeemmipgmhhjemlljidg/i
    );
    expect(extensionId).toBeInTheDocument();
    expect(screen.queryByText(/--enable-scripts/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "iOS & iPadOS" }));
    expect(
      screen.queryByText(/Send this to your end users:/i)
    ).toBeInTheDocument();

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
        isAnyTeamSelected={false}
        currentTeamName="Apples"
        isLoading={false}
        onCancel={noop}
        openEnrollSecretModal={noop}
      />
    );

    const text = screen.getByText("Something's gone wrong.");
    const ctaButton = screen.getByRole("button", {
      name: "Manage enroll secrets",
    });

    expect(text).toBeInTheDocument();
    expect(ctaButton).toBeEnabled();
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
