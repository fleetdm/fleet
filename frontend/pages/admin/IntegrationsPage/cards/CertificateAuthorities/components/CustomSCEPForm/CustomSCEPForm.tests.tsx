import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import CustomSCEPForm, { ICustomSCEPFormData } from "./CustomSCEPForm";

const createTestFormData = (overrides?: Partial<ICustomSCEPFormData>) => ({
  name: "TEST_NAME",
  scepURL: "https://test.com",
  challenge: "test-challenge",
  ...overrides,
});

describe("CustomSCEPForm", () => {
  it("render the custom button text", () => {
    render(
      <CustomSCEPForm
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

  it("enables and disabled form submittion depending on the form validation", async () => {
    const { user } = renderWithSetup(
      <CustomSCEPForm
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
      <CustomSCEPForm
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
