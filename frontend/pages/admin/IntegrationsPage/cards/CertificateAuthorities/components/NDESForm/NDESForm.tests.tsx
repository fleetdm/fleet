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

    // data is valid, so submit should be enabled
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

  it("submit button is disabled if isDirty is false", () => {
    render(
      <NDESForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        isDirty={false}
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("submit button is enabled if isDirty", () => {
    render(
      <NDESForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        isDirty
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();
  });
});
