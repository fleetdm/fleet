import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import { renderWithSetup, createMockRouter } from "test/test-utils";
import UserForm from "./UserForm";

// Note: A lot of this flow is tested e2e so these integration tests are only nuanced tests
describe("UserForm - component", () => {
  const defaultProps = {
    availableTeams: [],
    onCancel: noop,
    onSubmit: noop,
    submitText: "Submit",
    isPremiumTier: false,
    smtpConfigured: true,
    sesConfigured: true,
    canUseSso: false,
    isNewUser: true,
    router: createMockRouter(),
  };

  it("displays error messages for invalid inputs", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    const submitButton = screen.getByText("Submit");
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
});
