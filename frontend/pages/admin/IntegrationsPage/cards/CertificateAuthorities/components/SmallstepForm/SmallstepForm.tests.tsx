import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import SmallstepForm, { ISmallstepFormData } from "./SmallstepForm";

const createTestFormData = (overrides?: Partial<ISmallstepFormData>) => ({
  name: "TEST_NAME",
  scepURL: "https://test.com",
  challengeURL: "https://test.com/challenge",
  username: "testuser",
  password: "testpassword",
  ...overrides,
});

describe("SmallstepForm", () => {
  it("render the custom button text", () => {
    render(
      <SmallstepForm
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
    const testData = createTestFormData();
    render(
      <SmallstepForm
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

    // make name invalid by setting it to an empty string
    testData.name = "";
    render(
      <SmallstepForm
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
      <SmallstepForm
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
      <SmallstepForm
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
      <SmallstepForm
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
