import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { ANDROID_PLAY_STORE_URL } from "utilities/constants";

import AndroidLatestVersionWithTooltip from "./AndroidLatestVersionWithTooltip";

describe("AndroidLatestVersionWithTooltip", () => {
  const playStoreAppId = "com.example.app";
  const playStoreUrl = `${ANDROID_PLAY_STORE_URL}?id=${playStoreAppId}`;

  it('renders "Latest" text with tooltip with Play Store link', async () => {
    const { user } = renderWithSetup(
      <AndroidLatestVersionWithTooltip androidPlayStoreId={playStoreAppId} />
    );
    user.hover(screen.getByText("Latest"));
    await waitFor(() => {
      expect(
        screen.getByText(/See latest version on the/i)
      ).toBeInTheDocument();
      expect(screen.getByText(/Play Store/i)).toBeInTheDocument();
    });
    // Looks for <a> with correct href and text
    const playStoreLink = screen.getByRole("link", { name: /Play Store/i });
    expect(playStoreLink).toHaveAttribute("href", playStoreUrl);
    expect(playStoreLink).toHaveAttribute("target", "_blank");
    expect(playStoreLink).toHaveTextContent("Play Store");
  });
});
