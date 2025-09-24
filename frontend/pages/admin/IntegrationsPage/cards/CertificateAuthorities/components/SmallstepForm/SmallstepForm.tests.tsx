import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

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
        onChange={() => undefined}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeVisible();
  });

  it("enables and disabled form submittion depending on the form validation", async () => {
    const { user } = renderWithSetup(
      <SmallstepForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={() => undefined}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    // data is valid, so submit should be enabled
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();

    // name input is invalidated, submit should be disabled
    await user.clear(screen.getByLabelText("Name"));
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("disables submit when isSubmitting is set to true", () => {
    render(
      <SmallstepForm
        formData={createTestFormData()}
        isSubmitting
        submitBtnText="Submit"
        onChange={() => undefined}
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
        onChange={() => undefined}
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
        onChange={() => undefined}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();
  });
});
