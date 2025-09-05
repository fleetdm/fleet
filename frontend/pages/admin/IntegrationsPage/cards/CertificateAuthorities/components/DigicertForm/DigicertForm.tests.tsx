import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import DigicertForm, { IDigicertFormData } from "./DigicertForm";

const createTestFormData = (overrides?: Partial<IDigicertFormData>) => ({
  name: "TEST_NAME",
  url: "https://test.com",
  apiToken: "test-api-123",
  profileId: "test-id-123",
  commonName: "test-common",
  userPrincipalName: "test-principal",
  certificateSeatId: "test-seat-123",
  ...overrides,
});

describe("DigicertForm", () => {
  it("render the custom button text", () => {
    render(
      <DigicertForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeVisible();
  });

  it("enables and disables form submission depending on the form validation", async () => {
    const { user } = renderWithSetup(
      <DigicertForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    // data is valid, but no changes have been made so submit should be disabled
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();

    // Name is valid and now changed so submit should be enabled
    await user.type(screen.getByLabelText("Name"), "Updated_Name");
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();

    // name input is invalidated, submit should be disabled
    await user.clear(screen.getByLabelText("Name"));
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("disables submit when isSubmitting is set to true", () => {
    render(
      <DigicertForm
        formData={createTestFormData()}
        isSubmitting
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("has submit disabled when no changes have been made", async () => {
    const { user } = renderWithSetup(
      <DigicertForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();

    // Update a field
    await user.type(screen.getByLabelText("Name"), "Updated_Name");
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();
  });
});
