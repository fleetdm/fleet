import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import UsersForm from "./UsersForm";

describe("UsersForm", () => {
  const defaultProps = {
    currentTeamId: 0,
    defaultIsEndUserAuthEnabled: false,
    defaultLockEndUserInfo: false,
    defaultEnableManagedLocalAccount: false,
  };

  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("renders the end user authentication and managed local account checkboxes", () => {
    render(<UsersForm {...defaultProps} />);
    expect(screen.getByText("End user authentication")).toBeInTheDocument();
    expect(screen.getByText("Managed local account")).toBeInTheDocument();
  });

  it("renders help text for end user authentication with IdP link", () => {
    render(<UsersForm {...defaultProps} />);
    expect(
      screen.getByText(/End users are required to authenticate/)
    ).toBeInTheDocument();
    expect(screen.getByText("identity provider (IdP)")).toBeInTheDocument();
  });

  it("renders help text for managed local account", () => {
    render(<UsersForm {...defaultProps} />);
    expect(
      screen.getByText(/Fleet generates a user \(_fleetadmin\)/)
    ).toBeInTheDocument();
  });

  it("hides lock end user info when end user auth is unchecked", () => {
    render(<UsersForm {...defaultProps} />);
    expect(screen.queryByText("Lock end user info")).not.toBeInTheDocument();
  });

  it("shows lock end user info inline when end user auth is checked", () => {
    render(<UsersForm {...defaultProps} defaultIsEndUserAuthEnabled />);
    expect(screen.getByText("Lock end user info")).toBeInTheDocument();
  });

  it("reveals lock end user info when end user auth is toggled on", async () => {
    const { user } = render(<UsersForm {...defaultProps} />);

    expect(screen.queryByText("Lock end user info")).not.toBeInTheDocument();

    await user.click(
      screen.getByRole("checkbox", { name: "End user authentication" })
    );

    expect(screen.getByText("Lock end user info")).toBeInTheDocument();
  });

  it("renders managed local account checkbox as unchecked by default", () => {
    render(<UsersForm {...defaultProps} />);
    expect(
      screen.getByRole("checkbox", { name: "Managed local account" })
    ).not.toBeChecked();
  });

  it("renders managed local account checkbox as checked when default is true", () => {
    render(<UsersForm {...defaultProps} defaultEnableManagedLocalAccount />);
    expect(
      screen.getByRole("checkbox", { name: "Managed local account" })
    ).toBeChecked();
  });

  it("does not show an 'Advanced options' reveal button", () => {
    render(<UsersForm {...defaultProps} />);
    expect(screen.queryByText("Advanced options")).not.toBeInTheDocument();
  });

  it("renders exactly one Save button", () => {
    render(<UsersForm {...defaultProps} />);
    const saveButtons = screen.getAllByRole("button", { name: "Save" });
    expect(saveButtons).toHaveLength(1);
  });
});
