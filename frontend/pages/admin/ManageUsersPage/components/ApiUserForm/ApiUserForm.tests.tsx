import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer } from "test/test-utils";
import createMockTeam from "__mocks__/teamMock";

import ApiUserForm from "./ApiUserForm";

jest.mock("services/entities/api_endpoints");

const renderWithSetup = createCustomRenderer({ withBackendMock: true });

const FLEET_ASSIGNED = {
  name: "CI token",
  global_role: null,
  fleets: [createMockTeam({ id: 1, name: "Fleet 1", role: "observer" })],
};

const ENDPOINTS_SELECTED = {
  name: "CI token",
  global_role: null,
  fleets: [createMockTeam({ id: 1, name: "Fleet 1", role: "observer" })],
  api_endpoints: [{ method: "GET", path: "/api/v1/fleet/hosts" }],
};

describe("ApiUserForm - component", () => {
  const defaultProps = {
    onCancel: noop,
    onSubmit: noop,
    availableTeams: [createMockTeam({ id: 1, name: "Fleet 1" })],
    isPremiumTier: true,
  };

  it("keeps the submit button enabled while the form is invalid", async () => {
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <ApiUserForm {...defaultProps} onSubmit={onSubmit} />
    );

    const submitButton = screen.getByRole("button", { name: "Add" });
    expect(submitButton).not.toBeDisabled();

    await user.click(submitButton);

    expect(submitButton).not.toBeDisabled();
    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("Enter a name")).toBeInTheDocument();
  });

  it("does not show the Name error until the field is dirty", async () => {
    const { user } = renderWithSetup(<ApiUserForm {...defaultProps} />);

    // Pristine blur stays silent.
    await user.click(screen.getByLabelText("Name"));
    await user.tab();
    expect(screen.queryByText("Enter a name")).not.toBeInTheDocument();

    // Dirty then emptied, so the field is dirty with an invalid value.
    await user.type(screen.getByLabelText("Name"), "a");
    await user.clear(screen.getByLabelText("Name"));
    await user.tab();

    expect(screen.getByText("Enter a name")).toBeInTheDocument();
  });

  it("clears the Name error on focus", async () => {
    const { user } = renderWithSetup(<ApiUserForm {...defaultProps} />);

    await user.click(screen.getByRole("button", { name: "Add" }));
    expect(screen.getByText("Enter a name")).toBeInTheDocument();

    // The error replaces the label, so it is the field's accessible name while
    // it shows.
    await user.click(screen.getByLabelText("Enter a name"));

    expect(screen.queryByText("Enter a name")).not.toBeInTheDocument();
  });

  it("renders the 'at least one fleet' error inline", async () => {
    const { user } = renderWithSetup(<ApiUserForm {...defaultProps} />);

    await user.type(screen.getByLabelText("Name"), "CI token");
    await user.click(screen.getByRole("radio", { name: "Assign to fleet(s)" }));
    await user.click(screen.getByRole("button", { name: "Add" }));

    expect(screen.getByText("Select at least one fleet")).toBeInTheDocument();
  });

  it("clears the fleet error when the user switches back to global", async () => {
    const { user } = renderWithSetup(<ApiUserForm {...defaultProps} />);

    await user.type(screen.getByLabelText("Name"), "CI token");
    await user.click(screen.getByRole("radio", { name: "Assign to fleet(s)" }));
    await user.click(screen.getByRole("button", { name: "Add" }));
    expect(screen.getByText("Select at least one fleet")).toBeInTheDocument();

    // The requirement no longer applies, so its error must go immediately.
    await user.click(screen.getByRole("radio", { name: "Global user" }));

    expect(
      screen.queryByText("Select at least one fleet")
    ).not.toBeInTheDocument();
  });

  it("requires an endpoint when 'Specific API endpoints' is selected", async () => {
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <ApiUserForm {...defaultProps} onSubmit={onSubmit} />
    );

    await user.type(screen.getByLabelText("Name"), "CI token");
    await user.click(
      screen.getByRole("radio", { name: "Specific API endpoints" })
    );
    await user.click(screen.getByRole("button", { name: "Add" }));

    expect(
      screen.getByText("Select at least one API endpoint")
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();

    // Switching back to "All" removes the requirement, so the error goes too.
    await user.click(screen.getByRole("radio", { name: "All API endpoints" }));

    expect(
      screen.queryByText("Select at least one API endpoint")
    ).not.toBeInTheDocument();
  });

  it("submits a trimmed name and omits api_endpoints on free tier", async () => {
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <ApiUserForm
        {...defaultProps}
        isPremiumTier={false}
        onSubmit={onSubmit}
      />
    );

    await user.type(screen.getByLabelText("Name"), "  CI token  ");
    await user.click(screen.getByRole("button", { name: "Add" }));

    expect(onSubmit).toHaveBeenCalledWith({
      name: "CI token",
      global_role: "observer",
      fleets: [],
      api_endpoints: undefined,
    });
  });

  it("does not apply premium-only rules on free tier", async () => {
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <ApiUserForm
        {...defaultProps}
        isPremiumTier={false}
        onSubmit={onSubmit}
      />
    );

    // The fleet and endpoint selectors are not rendered on free, so their rules
    // could never be satisfied and must not block submission.
    expect(
      screen.queryByRole("radio", { name: "Assign to fleet(s)" })
    ).not.toBeInTheDocument();

    await user.type(screen.getByLabelText("Name"), "CI token");
    await user.click(screen.getByRole("button", { name: "Add" }));

    expect(onSubmit).toHaveBeenCalled();
  });

  it("disables the fields and submit button while a submission is in flight", () => {
    renderWithSetup(<ApiUserForm {...defaultProps} isSubmitting />);

    expect(screen.getByLabelText("Name")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();
    // Cancel stays enabled so the user can leave.
    expect(screen.getByRole("button", { name: "Cancel" })).not.toBeDisabled();
  });

  it("disables the permissions and API access controls while in flight", () => {
    renderWithSetup(<ApiUserForm {...defaultProps} isSubmitting />);

    expect(screen.getByRole("radio", { name: "Global user" })).toBeDisabled();
    expect(
      screen.getByRole("radio", { name: "Assign to fleet(s)" })
    ).toBeDisabled();
    expect(
      screen.getByRole("radio", { name: "All API endpoints" })
    ).toBeDisabled();
    expect(
      screen.getByRole("radio", { name: "Specific API endpoints" })
    ).toBeDisabled();
  });

  it("disables the fleet checkboxes while in flight", () => {
    renderWithSetup(
      <ApiUserForm
        {...defaultProps}
        defaultData={FLEET_ASSIGNED}
        isSubmitting
      />
    );

    // Checkbox renders a div[role=checkbox], not a native input, so disabled
    // shows up as aria-disabled rather than the disabled attribute.
    expect(screen.getByRole("checkbox", { name: "Fleet 1" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );
  });

  it("disables the endpoint search while in flight", () => {
    renderWithSetup(
      <ApiUserForm
        {...defaultProps}
        defaultData={ENDPOINTS_SELECTED}
        isSubmitting
      />
    );

    expect(
      screen.getByPlaceholderText("Search by name or path")
    ).toBeDisabled();
  });
});
