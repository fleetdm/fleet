import React from "react";
import { renderWithSetup } from "test/test-utils";
import { screen, render } from "@testing-library/react";
import { noop } from "lodash";

// @ts-ignore
import InputField from "components/forms/fields/InputField";

import LabelForm from "./LabelForm";

describe("LabelForm", () => {
  it("should validate the name to be required", async () => {
    const { user } = renderWithSetup(
      <LabelForm onSave={noop} onCancel={noop} />
    );

    const nameInput = screen.getByLabelText("Name");

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(screen.getByText("Label name must be present")).toBeInTheDocument();

    await user.type(nameInput, "Label name");
    expect(
      screen.queryByText("Label name must be present")
    ).not.toBeInTheDocument();
  });

  it("should render any additional field the user provides", () => {
    render(
      <LabelForm
        onSave={noop}
        onCancel={noop}
        additionalFields={<InputField name="test field" label="test field" />}
      />
    );

    expect(screen.getByLabelText("test field")).toBeInTheDocument();
  });

  it("should pass up the form data when the form is submitted and valid", async () => {
    const onSave = jest.fn();
    const { user } = renderWithSetup(
      <LabelForm onSave={onSave} onCancel={jest.fn()} />
    );

    const nameValue = "Test Name";
    const descriptionValue = "Test Description";
    await user.type(screen.getByLabelText("Name"), nameValue);
    await user.type(screen.getByLabelText("Description"), descriptionValue);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(
      { name: nameValue, description: descriptionValue },
      true
    );
  });
});
