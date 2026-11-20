import { screen } from "@testing-library/react";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import WipeModal from "./WipeModal";

const MOCK_PROPS = {
  id: 7,
  hostName: "host-1",
  onSuccess: jest.fn(),
  onClose: jest.fn(),
};

const REPURPOSE_OR_WIPE =
  "Repurpose the host for another end user, or wipe lost or stolen hosts.";

describe("WipeModal", () => {
  it("points Windows hosts to Reset for repurposing", () => {
    const render = createCustomRenderer();
    render(
      <WipeModal
        {...MOCK_PROPS}
        hostPlatform="windows"
        isWindowsHost
        isLinuxHost={false}
      />
    );

    expect(
      screen.getByText(/Wipe lost or stolen hosts. To use the host again/)
    ).toBeInTheDocument();
    expect(screen.getByText("Actions > Reset")).toBeInTheDocument();
    expect(screen.queryByText(REPURPOSE_OR_WIPE)).not.toBeInTheDocument();
  });

  it("explains the script-based wipe for Linux hosts", () => {
    const render = createCustomRenderer();
    render(
      <WipeModal
        {...MOCK_PROPS}
        hostPlatform="ubuntu"
        isWindowsHost={false}
        isLinuxHost
      />
    );

    expect(
      screen.getByText(/To repurpose the host for another end user, you will/)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/This will run a script to erase all data/)
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Learn more/ })).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/linux-wipe"
    );
    expect(screen.queryByText("Actions > Reset")).not.toBeInTheDocument();
  });

  it.each(["darwin", "ios", "ipados", "android"])(
    "offers repurposing through Wipe on %s",
    (hostPlatform) => {
      const render = createCustomRenderer();
      render(
        <WipeModal
          {...MOCK_PROPS}
          hostPlatform={hostPlatform}
          isWindowsHost={false}
          isLinuxHost={false}
        />
      );

      expect(screen.getByText(REPURPOSE_OR_WIPE)).toBeInTheDocument();
      expect(screen.queryByText("Actions > Reset")).not.toBeInTheDocument();
    }
  );
});
