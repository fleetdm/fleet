import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import DebugLoggingModal from "./DebugLoggingModal";

const MOCK_PROPS = {
  hostId: 7,
  hostName: "macos-host-1",
  isCurrentlyActive: false,
  onSuccess: jest.fn(),
  onClose: jest.fn(),
};

describe("DebugLoggingModal", () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  describe("enable mode", () => {
    it("renders enable copy and the duration dropdown", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<DebugLoggingModal {...MOCK_PROPS} />);

      expect(screen.getByText(/Enable debug logging/i)).toBeInTheDocument();
      expect(
        screen.getByText(/Turn on orbit debug logging for/i)
      ).toBeInTheDocument();
      expect(screen.getByText(/macos-host-1/i)).toBeInTheDocument();
      // Default duration is 24h.
      expect(screen.getByText(/24 hours \(default\)/i)).toBeInTheDocument();
    });

    it("renders an Enable CTA and a Cancel button", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<DebugLoggingModal {...MOCK_PROPS} />);

      expect(screen.getByRole("button", { name: "Enable" })).toBeEnabled();
      expect(screen.getByRole("button", { name: "Cancel" })).toBeEnabled();
    });

    it("calls onClose when Cancel is clicked", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<DebugLoggingModal {...MOCK_PROPS} />);

      await user.click(screen.getByRole("button", { name: "Cancel" }));
      expect(MOCK_PROPS.onClose).toHaveBeenCalled();
    });
  });

  describe("disable mode", () => {
    it("renders disable copy and hides the duration dropdown", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<DebugLoggingModal {...MOCK_PROPS} isCurrentlyActive />);

      expect(screen.getByText(/Disable debug logging/i)).toBeInTheDocument();
      expect(
        screen.getByText(/Turn off orbit debug logging on/i)
      ).toBeInTheDocument();
      expect(screen.getByText(/macos-host-1/i)).toBeInTheDocument();
      expect(
        screen.queryByText(/24 hours \(default\)/i)
      ).not.toBeInTheDocument();
    });

    it("renders a Disable CTA and a Cancel button", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<DebugLoggingModal {...MOCK_PROPS} isCurrentlyActive />);

      expect(screen.getByRole("button", { name: "Disable" })).toBeEnabled();
      expect(screen.getByRole("button", { name: "Cancel" })).toBeEnabled();
    });
  });
});
