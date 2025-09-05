import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import NDESForm, { INDESFormData } from "./NDESForm";

const createTestFormData = (overrides?: Partial<INDESFormData>) => ({
  scepURL: "https://test.com",
  adminURL: "https://test.com",
  username: "test user",
  password: "password123",
  ...overrides,
});

describe("NDESForm", () => {
  it("render the custom button text", () => {
    render(
      <NDESForm
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
      <NDESForm
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

    // scepURL is valid and now changed so submit should be enabled
    await user.type(screen.getByLabelText("SCEP URL"), "https://updated.com");
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();

    // scepURL input is invalidated, submit should be disabled
    await user.clear(screen.getByLabelText("SCEP URL"));
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("disables submit when isSubmitting is set to true", () => {
    render(
      <NDESForm
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
      <NDESForm
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
    await user.type(screen.getByLabelText("SCEP URL"), "https://updated.com");
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();
  });
});
