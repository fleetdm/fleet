import React from "react";
import { screen } from "@testing-library/react";

import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import MdmSettings from "./MdmSettings";

jest.mock("services/entities/mdm_apple", () => ({
  __esModule: true,
  default: {
    getAppleAPNInfo: jest.fn().mockResolvedValue({}),
    getVppTokens: jest.fn().mockResolvedValue({ vpp_tokens: [] }),
  },
}));

jest.mock("services/entities/mdm", () => ({
  __esModule: true,
  default: {
    // 404 is how the API reports "no EULA uploaded yet".
    getEULAMetadata: jest.fn().mockRejectedValue({ status: 404 }),
  },
}));

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

describe("MdmSettings - EULA gating (#50757)", () => {
  it("shows the EULA section when only Windows MDM is configured", async () => {
    renderPage({
      enabled_and_configured: false,
      windows_enabled_and_configured: true,
      android_enabled_and_configured: false,
      apple_bm_enabled_and_configured: false,
    });

    expect(
      await screen.findByText("End user license agreement (EULA)")
    ).toBeInTheDocument();
  });

  it("shows the EULA section when only Android MDM is configured", async () => {
    renderPage({
      enabled_and_configured: false,
      android_enabled_and_configured: true,
      apple_bm_enabled_and_configured: false,
    });

    expect(
      await screen.findByText("End user license agreement (EULA)")
    ).toBeInTheDocument();
  });

  it("hides the EULA section when no MDM platform is configured", async () => {
    renderPage({
      enabled_and_configured: false,
      windows_enabled_and_configured: false,
      android_enabled_and_configured: false,
      apple_bm_enabled_and_configured: false,
    });

    expect(
      screen.queryByText("End user license agreement (EULA)")
    ).not.toBeInTheDocument();
  });

  it("hides the EULA section on Fleet Free", async () => {
    renderPage(
      {
        enabled_and_configured: false,
        windows_enabled_and_configured: true,
        android_enabled_and_configured: false,
        apple_bm_enabled_and_configured: false,
      },
      false
    );

    expect(
      screen.queryByText("End user license agreement (EULA)")
    ).not.toBeInTheDocument();
  });

  it("keeps the macOS migration section gated on Apple MDM", async () => {
    renderPage({
      enabled_and_configured: false,
      windows_enabled_and_configured: true,
      android_enabled_and_configured: false,
      apple_bm_enabled_and_configured: false,
    });

    // EULA renders, migration does not -- the two no longer share a gate.
    expect(
      await screen.findByText("End user license agreement (EULA)")
    ).toBeInTheDocument();
    expect(
      screen.queryByText("Migration workflow for macOS hosts")
    ).not.toBeInTheDocument();
  });
});
