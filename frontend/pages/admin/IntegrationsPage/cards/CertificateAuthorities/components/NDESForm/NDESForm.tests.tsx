import React, { useState } from "react";
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

// NDESForm is controlled by its parent, so exercising input changes requires
// a harness that applies them to the form data like the add/edit modals do.
const ControlledNDESForm = () => {
  const [formData, setFormData] = useState(createTestFormData());
  return (
    <NDESForm
      formData={formData}
      isSubmitting={false}
      submitBtnText="Submit"
      onChange={({ name, value }) =>
        setFormData((prev) => ({ ...prev, [name]: value }))
      }
      onSubmit={noop}
      onCancel={noop}
    />
  );
};

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
    const { user } = renderWithSetup(<ControlledNDESForm />);

    // data is valid, so submit should be enabled
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();

    // scepURL input is invalidated, submit should be disabled
    await user.clear(screen.getByLabelText("SCEP URL"));
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("re-validates when the parent changes the form data outside of an input change", () => {
    const formProps = {
      isSubmitting: false,
      submitBtnText: "Submit",
      onChange: noop,
      onSubmit: noop,
      onCancel: noop,
    };
    const { rerender } = render(
      <NDESForm formData={createTestFormData()} {...formProps} />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();

    // the edit modal clears the password itself when another credential field
    // changes; the form has to pick that up even though no input changed
    rerender(
      <NDESForm
        formData={createTestFormData({ password: "" })}
        {...formProps}
      />
    );
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
