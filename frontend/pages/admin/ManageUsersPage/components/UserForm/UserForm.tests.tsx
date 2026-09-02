import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import { renderWithSetup, createMockRouter } from "test/test-utils";
import createMockTeam from "__mocks__/teamMock";
import UserForm from "./UserForm";

// Note: Happy path is tested e2e so these integration tests are only edge cases
describe("UserForm - component", () => {
  const defaultProps = {
    availableTeams: [],
    onCancel: noop,
    onSubmit: noop,
    isModifiedByGlobalAdmin: true,
    isPremiumTier: true,
    smtpConfigured: true,
    sesConfigured: true,
    canUseSso: false,
    isNewUser: true,
    router: createMockRouter(),
    serverErrors: {},
  };

  it("reveals every error on submit, even for untouched fields", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    const submitButton = screen.getByText("Add");
    await user.click(submitButton);

    expect(screen.getByText("Enter a name")).toBeInTheDocument();
    expect(screen.getByText("Enter an email")).toBeInTheDocument();
    expect(screen.getByText("Enter a password")).toBeInTheDocument();
  });

  it("keeps the submit button enabled while the form is invalid", async () => {
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <UserForm {...defaultProps} onSubmit={onSubmit} />
    );

    const submitButton = screen.getByText("Add");
    expect(submitButton).not.toBeDisabled();

    await user.click(submitButton);

    expect(submitButton).not.toBeDisabled();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("renders the 'at least one fleet' error inline rather than as a toast", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    await user.click(screen.getByText("Add"));

    expect(screen.getByText("Select at least one fleet")).toBeInTheDocument();
  });

  it("renders SSO option when canUseSso is true", () => {
    render(<UserForm {...defaultProps} canUseSso />);

    expect(screen.getByLabelText("Single sign-on")).toBeInTheDocument();
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
    expect(
      screen.queryByRole("radio", { name: "Password" })
    ).toBeInTheDocument();
  });

  it("does not render password and 2FA sections when SSO is selected", () => {
    render(<UserForm {...defaultProps} canUseSso />);

    // Enable SSO
    const ssoRadio = screen.getByLabelText("Single sign-on");
    ssoRadio.click();

    // Check that the password radio is present
    const passwordRadio = screen.getByRole("radio", { name: "Password" });
    expect(passwordRadio).not.toBeDisabled();

    // Check that password input field and 2FA sections are not present
    expect(
      screen.queryByRole("input", { name: "Password" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("Enable two-factor authentication")
    ).not.toBeInTheDocument();
  });

  // #40410: blurring one field must not surface errors on fields the user has
  // not yet touched (previously, blurring the autofocused Name field flagged the
  // empty Email and Password fields).
  it("does not surface an Email error when only the Name field is blurred", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    await user.type(screen.getByLabelText("Full name"), "Alice");
    await user.tab();

    expect(screen.queryByText("Enter an email")).not.toBeInTheDocument();
    expect(screen.queryByText("Enter a password")).not.toBeInTheDocument();
  });

  it("does not show an error when a pristine field is blurred", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    await user.click(screen.getByLabelText("Email"));
    await user.tab();

    expect(screen.queryByText("Enter an email")).not.toBeInTheDocument();
  });

  it("shows the Email error on blur once the Email field is dirty", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    const emailField = screen.getByLabelText("Email");
    await user.type(emailField, "nope");
    await user.tab();

    expect(screen.getByText("Enter a valid email")).toBeInTheDocument();
  });

  it("clears the Email error on focus, not on typing", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    await user.click(screen.getByText("Add"));
    expect(screen.getByText("Enter an email")).toBeInTheDocument();

    // Querying the input by the error string is deliberate, not a typo for
    // "Email": FormField renders the error *in place of* the label, so while an
    // error shows, the error text IS the field's accessible label. Asserting it
    // here doubles as coverage of that rule (patterns.md, Visual affordances).
    await user.click(screen.getByLabelText("Enter an email"));

    expect(screen.queryByText("Enter an email")).not.toBeInTheDocument();
    // Clearing is per-field: the other errors are still on screen.
    expect(screen.getByText("Enter a name")).toBeInTheDocument();
  });

  it("does not validate the email when the field is read-only", async () => {
    // No SMTP/SES on an edit form makes email read-only. A field the user can't
    // edit must never hold an error that blocks submit.
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <UserForm
        {...defaultProps}
        isNewUser={false}
        smtpConfigured={false}
        sesConfigured={false}
        defaultName="User 1"
        defaultEmail="not-a-valid-email"
        defaultGlobalRole="observer"
        onSubmit={onSubmit}
      />
    );

    expect(screen.getByLabelText("Email")).toBeDisabled();

    await user.click(screen.getByText("Save"));

    expect(screen.queryByText("Enter a valid email")).not.toBeInTheDocument();
    expect(onSubmit).toHaveBeenCalled();
  });

  it("disables the permissions selectors while a submission is in flight", () => {
    render(
      <UserForm
        {...defaultProps}
        availableTeams={[createMockTeam({ id: 1, name: "Fleet 1" })]}
        defaultGlobalRole={null}
        defaultTeams={[createMockTeam({ id: 1, name: "Fleet 1" })]}
        isUpdatingUsers
      />
    );

    expect(screen.getByRole("checkbox", { name: "Fleet 1" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );
    expect(screen.getByRole("radio", { name: "Global user" })).toBeDisabled();
  });

  it("uses password error copy that matches the help text", async () => {
    const { user } = renderWithSetup(<UserForm {...defaultProps} />);

    const passwordField = document.querySelector(
      'input[name="password"]'
    ) as HTMLInputElement;
    await user.type(passwordField, "abcdefghijkl");
    await user.tab();

    expect(
      screen.getByText("Enter a password with at least 1 number and 1 symbol")
    ).toBeInTheDocument();
  });

  it("does not submit mfa_enabled for a new user who switches to SSO", async () => {
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <UserForm
        {...defaultProps}
        canUseSso
        defaultGlobalRole="observer"
        onSubmit={onSubmit}
      />
    );

    await user.type(screen.getByLabelText("Full name"), "User 1");
    await user.type(screen.getByLabelText("Email"), "user@example.com");

    // Ticking MFA and then switching to SSO hides the checkbox but leaves its
    // value in form state, so the payload must drop it.
    await user.click(screen.getByRole("checkbox", { name: "mfa_enabled" }));
    await user.click(screen.getByRole("radio", { name: "Single sign-on" }));

    await user.click(screen.getByText("Add"));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ sso_enabled: true, mfa_enabled: false })
    );
  });

  it("displays disabled SSO option when SSO is globally disabled but was previously enabled for the user", async () => {
    const props = {
      ...defaultProps,
      defaultName: "User 1",
      defaultEmail: "user@example.com",
      currentUserId: 1,
      canUseSso: false,
      isSsoEnabled: true,
      isNewUser: false,
    };

    const { user } = renderWithSetup(<UserForm {...props} />);

    // Check that the SSO radio is disabled
    const ssoRadio = screen.getByLabelText("Single sign-on");
    expect(ssoRadio).toBeDisabled();

    await user.click(screen.getByText("Save"));
    expect(screen.getByText("Enter a password")).toBeInTheDocument();
  });
});
