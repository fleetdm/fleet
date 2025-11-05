import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import CustomESTForm, { ICustomESTFormData } from "./CustomESTForm";

const createTestFormData = (overrides?: Partial<ICustomESTFormData>) => ({
  name: "TEST_NAME",
  url: "https://test.com",
  username: "testuser",
  password: "testpassword",
  ...overrides,
});

describe("CustomESTForm", () => {
  it("render the custom button text", () => {
    render(
      <CustomESTForm
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

  it("enables submission depending on the form validation", async () => {
    const testData = createTestFormData();
    render(
      <CustomESTForm
        formData={testData}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    // data is valid, so submit should be enabled
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();
  });

  it("disables submission when form is invalid", async () => {
    const testData = createTestFormData();
    // make name invalid by setting it to an empty string
    testData.name = "";
    render(
      <CustomESTForm
        formData={testData}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );
    // name is required, so submit should be disabled
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("disables submit when isSubmitting is set to true", () => {
    render(
      <CustomESTForm
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
      <CustomESTForm
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
      <CustomESTForm
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
