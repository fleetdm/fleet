import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import HydrantForm, { IHydrantFormData } from "./HydrantForm";

const createTestFormData = (overrides?: Partial<IHydrantFormData>) => ({
  name: "TEST_NAME",
  url: "https://test.com",
  clientId: "123",
  clientSecret: "test secret",
  ...overrides,
});

describe("DigicertForm", () => {
  it("render the custom button text", () => {
    render(
      <HydrantForm
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
      <HydrantForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    // data is valid, submit should be enabled
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();

    // name input is invalidated, submit should be disabled
    await user.clear(screen.getByLabelText("Name"));
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("disables submit when isSubmitting is set to true", () => {
    render(
      <HydrantForm
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
});
