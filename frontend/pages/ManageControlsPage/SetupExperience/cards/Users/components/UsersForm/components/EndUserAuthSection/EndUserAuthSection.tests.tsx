import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import EndUserAuthSection from "./EndUserAuthSection";

describe("EndUserAuthSection", () => {
  const onEndUserAuthChangeMock = jest.fn();
  const onLockEndUserInfoChangeMock = jest.fn();
  beforeEach(() => {
    onEndUserAuthChangeMock.mockClear();
    onLockEndUserInfoChangeMock.mockClear();
  });

  const defaultProps = {
    endUserAuthEnabled: false,
    lockEndUserInfo: false,
    onEndUserAuthChange: onEndUserAuthChangeMock,
    onLockEndUserInfoChange: onLockEndUserInfoChangeMock,
    isIdPConfigured: true,
    isMacMdmEnabledAndConfigured: true,
    gitOpsModeEnabled: false,
  };

  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("renders the end user authentication checkbox", () => {
    render(<EndUserAuthSection {...defaultProps} />);
    expect(screen.getByText("Require IdP authentication")).toBeInTheDocument();
  });

  it("renders help text for end user authentication with IdP link", () => {
    render(<EndUserAuthSection {...defaultProps} />);
    expect(
      screen.getByText(/End users are required to authenticate/)
    ).toBeInTheDocument();
    expect(screen.getByText("identity provider (IdP)")).toBeInTheDocument();
  });

  it("hides lock end user info when end user auth is unchecked", () => {
    render(<EndUserAuthSection {...defaultProps} />);
    expect(screen.queryByText("Lock end user info")).not.toBeInTheDocument();
  });

  it("shows lock end user info inline when end user auth is checked", () => {
    render(<EndUserAuthSection {...defaultProps} endUserAuthEnabled />);
    expect(screen.getByText("Lock end user info")).toBeInTheDocument();
  });

  it("calls onEndUserAuthChange when the EUA checkbox is toggled", async () => {
    const { user } = render(<EndUserAuthSection {...defaultProps} />);

    await user.click(
      screen.getByRole("checkbox", { name: "Require IdP authentication" })
    );

    expect(onEndUserAuthChangeMock).toHaveBeenCalledWith(true);
  });

  it("calls onLockEndUserInfoChange when lock end user info is toggled", async () => {
    const { user } = render(
      <EndUserAuthSection {...defaultProps} endUserAuthEnabled />
    );

    await user.click(
      screen.getByRole("checkbox", { name: "Lock end user info" })
    );

    expect(onLockEndUserInfoChangeMock).toHaveBeenCalledWith(true);
  });

  describe("disabled states", () => {
    it("disables end user authentication checkbox when IdP is not configured", () => {
      render(<EndUserAuthSection {...defaultProps} isIdPConfigured={false} />);
      expect(
        screen.getByRole("checkbox", { name: "Require IdP authentication" })
      ).toHaveAttribute("aria-disabled", "true");
    });

    it("enables end user authentication checkbox when IdP is configured", () => {
      render(<EndUserAuthSection {...defaultProps} />);
      expect(
        screen.getByRole("checkbox", { name: "Require IdP authentication" })
      ).toHaveAttribute("aria-disabled", "false");
    });

    it("disables lock end user info when IdP is not configured", () => {
      render(
        <EndUserAuthSection
          {...defaultProps}
          isIdPConfigured={false}
          endUserAuthEnabled
        />
      );
      expect(
        screen.getByRole("checkbox", { name: "Lock end user info" })
      ).toHaveAttribute("aria-disabled", "true");
    });

    it("disables lock end user info when Apple MDM is not configured", () => {
      render(
        <EndUserAuthSection
          {...defaultProps}
          isMacMdmEnabledAndConfigured={false}
          endUserAuthEnabled
        />
      );
      expect(
        screen.getByRole("checkbox", { name: "Lock end user info" })
      ).toHaveAttribute("aria-disabled", "true");
    });

    it("enables lock end user info when Apple MDM and IdP are configured", () => {
      render(<EndUserAuthSection {...defaultProps} endUserAuthEnabled />);
      expect(
        screen.getByRole("checkbox", { name: "Lock end user info" })
      ).toHaveAttribute("aria-disabled", "false");
    });

    it("does not call onEndUserAuthChange when IdP is not configured", async () => {
      const { user } = render(
        <EndUserAuthSection {...defaultProps} isIdPConfigured={false} />
      );

      await user.click(
        screen.getByRole("checkbox", { name: "Require IdP authentication" })
      );

      expect(onEndUserAuthChangeMock).not.toHaveBeenCalled();
    });
  });
});
