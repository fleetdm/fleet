import { screen } from "@testing-library/react";
import React from "react";

import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import mdmAPI from "services/entities/mdm";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import MdmSettings from "./MdmSettings";

jest.mock("services/entities/mdm_apple", () => ({
  __esModule: true,
  default: {
    getAppleAPNInfo: jest.fn().mockResolvedValue({}),
    getVppTokens: jest.fn().mockResolvedValue({ vpp_tokens: [] }),
  },
}));

jest.mock("services/entities/microsoft_graph_credentials", () => ({
  __esModule: true,
  default: {
    getCredentials: jest
      .fn()
      .mockResolvedValue({ microsoft_graph_credentials: [] }),
  },
}));

jest.mock("services/entities/mdm", () => ({
  __esModule: true,
  default: {
    // 404 is how the API reports that nothing has been uploaded yet.
    getEULAMetadata: jest.fn().mockRejectedValue({ status: 404 }),
    getWindowsEULAMetadata: jest.fn().mockRejectedValue({ status: 404 }),
    deleteWindowsEULA: jest.fn().mockResolvedValue({}),
  },
}));

const SECTION_TITLE = "End user license agreement (EULA)";

const renderPage = (
  mdmOverrides: Partial<IConfig["mdm"]>,
  isPremiumTier = true
) => {
  const render = createCustomRenderer({
    context: { app: { isPremiumTier, config: createMockConfig() } },
    withBackendMock: true,
  });

  return render(
    <MdmSettings
      router={createMockRouter()}
      isPremiumTier={isPremiumTier}
      appConfig={createMockConfig({
        mdm: createMockMdmConfig(mdmOverrides),
      })}
    />
  );
};

describe("MdmSettings end user agreement section", () => {
  it("shows the section on a Windows-only server", async () => {
    renderPage({
      enabled_and_configured: false,
      apple_bm_enabled_and_configured: false,
      windows_enabled_and_configured: true,
      android_enabled_and_configured: false,
    });

    expect(await screen.findByText(SECTION_TITLE)).toBeInTheDocument();
    expect(
      screen.queryByText("Migration workflow for macOS hosts")
    ).not.toBeInTheDocument();
  });

  it("shows the section when Apple MDM and Apple Business Manager are set up", async () => {
    renderPage({
      enabled_and_configured: true,
      apple_bm_enabled_and_configured: true,
      windows_enabled_and_configured: false,
    });

    expect(await screen.findByText(SECTION_TITLE)).toBeInTheDocument();
  });

  it("hides the section when neither platform can show an agreement", async () => {
    renderPage({
      enabled_and_configured: true,
      apple_bm_enabled_and_configured: false,
      windows_enabled_and_configured: false,
      android_enabled_and_configured: true,
    });

    expect(
      await screen.findByText("Mobile device management (MDM)")
    ).toBeInTheDocument();
    expect(screen.queryByText(SECTION_TITLE)).not.toBeInTheDocument();
  });

  it("hides the section on Fleet Free", async () => {
    renderPage(
      {
        enabled_and_configured: false,
        apple_bm_enabled_and_configured: false,
        windows_enabled_and_configured: true,
      },
      false
    );

    expect(
      await screen.findByText("Mobile device management (MDM)")
    ).toBeInTheDocument();
    expect(screen.queryByText(SECTION_TITLE)).not.toBeInTheDocument();
  });

  it("shows the uploader again after the agreement is deleted", async () => {
    jest.mocked(mdmAPI.getWindowsEULAMetadata).mockResolvedValueOnce({
      name: "terms.md",
      token: "win-token",
      created_at: "2026-10-01T00:00:00Z",
    });
    const { user } = renderPage({
      enabled_and_configured: false,
      apple_bm_enabled_and_configured: false,
      windows_enabled_and_configured: true,
    });

    expect(await screen.findByText("terms.md")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Delete EULA" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    // The refetch returns 404, while react-query keeps the last metadata.
    expect(
      await screen.findByText(
        "Export your document as a markdown (.md) file and upload it."
      )
    ).toBeInTheDocument();
    expect(screen.queryByText("terms.md")).not.toBeInTheDocument();
  });
});
