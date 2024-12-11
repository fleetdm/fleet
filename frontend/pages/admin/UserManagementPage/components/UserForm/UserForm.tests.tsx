import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import { renderWithSetup, createMockRouter } from "test/test-utils";
import { DEFAULT_USER_FORM_ERRORS } from "utilities/constants";
import UserForm from "./UserForm";

// Note: Happy path is tested e2e so these integration tests are only edge cases
describe("UserForm - component", () => {
  const defaultProps = {
    availableTeams: [],
    onCancel: noop,
    onSubmit: noop,
    submitText: "Add",
    isModifiedByGlobalAdmin: true,
    isPremiumTier: true,
    smtpConfigured: true,
    sesConfigured: true,
    canUseSso: false,
    isNewUser: true,
    router: createMockRouter(),
    userFormErrors: DEFAULT_USER_FORM_ERRORS,
  };

  it("displays error messages for invalid inputs", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    const submitButton = screen.getByText("Add");
    await user.click(submitButton);

    expect(
      screen.getByText("Name field must be completed")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Email field must be completed")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Password field must be completed")
    ).toBeInTheDocument();
  });

  it("renders SSO option when canUseSso is true", () => {
    render(<UserForm {...defaultProps} canUseSso />);

    expect(screen.getByLabelText("Enable single sign-on")).toBeInTheDocument();
  });

  it("disables invite user option when SMTP and SES are not configured", () => {
    render(
      <UserForm
        {...defaultProps}
        smtpConfigured={false}
        sesConfigured={false}
      />
    );

    const inviteUserRadio = screen.getByLabelText("Invite user");
    expect(inviteUserRadio).toBeDisabled();
  });

  it("does not render premium sections when isPremiumTier is false", async () => {
    renderWithSetup(<UserForm {...defaultProps} isPremiumTier={false} />);

    // Check that premium-specific elements are not present
    expect(screen.queryByText("Global user")).not.toBeInTheDocument();
    expect(screen.queryByText("Assign team(s)")).not.toBeInTheDocument();
    expect(
      screen.queryByText("Enable two-factor authentication")
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/team/i)).not.toBeInTheDocument();

    // Verify that non-premium elements are still present
    expect(screen.getByLabelText("Full name")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
  });

  it("does not render password and 2FA sections when SSO is enabled", () => {
    render(<UserForm {...defaultProps} canUseSso />);

    // Enable SSO
    const ssoRadio = screen.getByLabelText("Enable single sign-on");
    ssoRadio.click();

    // Check that password and 2FA sections are not present
    expect(screen.queryByText("Password")).not.toBeInTheDocument();
    expect(
      screen.queryByText("Enable two-factor authentication")
    ).not.toBeInTheDocument();
  });
});
