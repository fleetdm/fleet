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

    // data is valid, submit should be enabled
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
});
